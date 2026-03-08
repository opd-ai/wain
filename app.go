// Package wain provides a statically-compiled Go UI toolkit with GPU rendering.
//
// This package serves as the primary public API entry point, exposing a high-level
// interface for creating UI applications without requiring knowledge of platform
// internals (Wayland/X11) or GPU details (Intel/AMD/software rendering).
//
// # Quick Start
//
//	app := wain.NewApp()
//	app.Run()  // blocks until app.Quit() is called
//
// # Architecture
//
// The App type manages three core responsibilities:
//   - Display server auto-detection (Wayland preferred, X11 fallback)
//   - Renderer auto-detection (Intel GPU → AMD GPU → software fallback)
//   - Event loop management (single-goroutine event dispatch)
//
// All platform and GPU lifecycle management is handled internally.
package wain

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/render/backend"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/shm"
	"github.com/opd-ai/wain/internal/wayland/xdg"
	x11client "github.com/opd-ai/wain/internal/x11/client"
)

var (
	// ErrNotRunning is returned when calling methods that require Run() to be called first.
	ErrNotRunning = errors.New("wain: app not running")

	// ErrAlreadyRunning is returned when Run() is called multiple times.
	ErrAlreadyRunning = errors.New("wain: app already running")

	// ErrNoDisplay is returned when no display server is available.
	ErrNoDisplay = errors.New("wain: no display server available")
)

// DisplayServer represents the detected display server type.
type DisplayServer int

const (
	// DisplayServerUnknown indicates no display server was detected.
	DisplayServerUnknown DisplayServer = iota

	// DisplayServerWayland indicates a Wayland compositor.
	DisplayServerWayland

	// DisplayServerX11 indicates an X11 server.
	DisplayServerX11
)

func (d DisplayServer) String() string {
	switch d {
	case DisplayServerWayland:
		return "Wayland"
	case DisplayServerX11:
		return "X11"
	default:
		return "Unknown"
	}
}

// App represents a UI application with automatic platform and GPU detection.
type App struct {
	mu sync.Mutex

	// Display server connection
	displayServer DisplayServer
	waylandConn   *client.Connection
	x11Conn       *x11client.Connection

	// Wayland-specific objects
	waylandRegistry  *client.Registry
	waylandCompositor *client.Compositor
	waylandShm       *shm.SHM
	waylandWmBase    *xdg.WmBase
	waylandSurface   *client.Surface
	waylandXdgSurface *xdg.Surface
	waylandToplevel  *xdg.Toplevel

	// X11-specific objects
	x11Window x11client.XID
	x11GC     x11client.XID

	// Rendering backend
	renderer     backend.Renderer
	backendType  backend.BackendType
	displayList  *displaylist.DisplayList

	// State
	running      bool
	shouldQuit   bool
	initialized  bool
	width        int
	height       int
	verbose      bool
}

// AppConfig contains configuration options for creating an App.
type AppConfig struct {
	// Width is the initial window width in pixels (default: 800).
	Width int

	// Height is the initial window height in pixels (default: 600).
	Height int

	// ForceSoftware forces software rendering even if GPU is available (default: false).
	ForceSoftware bool

	// ForceX11 forces X11 even if Wayland is available (default: false).
	ForceX11 bool

	// DRMPath is the path to the DRM device for GPU detection (default: "/dev/dri/renderD128").
	DRMPath string

	// Verbose enables logging of backend selection decisions (default: false).
	Verbose bool
}

// DefaultConfig returns the default application configuration.
func DefaultConfig() AppConfig {
	return AppConfig{
		Width:         800,
		Height:        600,
		ForceSoftware: false,
		ForceX11:      false,
		DRMPath:       "/dev/dri/renderD128",
		Verbose:       false,
	}
}

// NewApp creates a new application with default configuration.
func NewApp() *App {
	return NewAppWithConfig(DefaultConfig())
}

// NewAppWithConfig creates a new application with the specified configuration.
func NewAppWithConfig(cfg AppConfig) *App {
	if cfg.Width <= 0 {
		cfg.Width = 800
	}
	if cfg.Height <= 0 {
		cfg.Height = 600
	}
	if cfg.DRMPath == "" {
		cfg.DRMPath = "/dev/dri/renderD128"
	}

	return &App{
		width:       cfg.Width,
		height:      cfg.Height,
		verbose:     cfg.Verbose,
		displayList: displaylist.New(),
	}
}

// Run initializes the application and starts the event loop.
// This method blocks until Quit() is called.
func (a *App) Run() error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return ErrAlreadyRunning
	}
	a.running = true
	a.mu.Unlock()

	// Initialize display server and renderer
	if err := a.initialize(); err != nil {
		a.cleanup()
		return fmt.Errorf("wain: initialization failed: %w", err)
	}

	// Run event loop
	if err := a.eventLoop(); err != nil {
		a.cleanup()
		return fmt.Errorf("wain: event loop error: %w", err)
	}

	// Clean up resources
	a.cleanup()
	return nil
}

// Quit signals the application to exit the event loop.
func (a *App) Quit() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.shouldQuit = true
}

// DisplayServer returns the detected display server type.
func (a *App) DisplayServer() DisplayServer {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.displayServer
}

// BackendType returns the detected rendering backend type.
func (a *App) BackendType() backend.BackendType {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.backendType
}

// Dimensions returns the current window dimensions.
func (a *App) Dimensions() (width, height int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.width, a.height
}

// initialize performs display server and renderer initialization.
func (a *App) initialize() error {
	// Phase 1: Detect and connect to display server
	if err := a.initDisplayServer(); err != nil {
		return err
	}

	// Phase 2: Create window/surface
	if err := a.initWindow(); err != nil {
		return err
	}

	// Phase 3: Initialize renderer
	if err := a.initRenderer(); err != nil {
		return err
	}

	// Phase 4: Render initial frame
	if err := a.renderInitialFrame(); err != nil {
		return err
	}

	a.initialized = true
	return nil
}

// initDisplayServer detects and connects to a display server.
// Wayland is preferred, with X11 as fallback.
func (a *App) initDisplayServer() error {
	// Try Wayland first
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	if waylandDisplay == "" {
		waylandDisplay = "wayland-0"
	}

	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntimeDir != "" {
		waylandPath := fmt.Sprintf("%s/%s", xdgRuntimeDir, waylandDisplay)
		if _, err := os.Stat(waylandPath); err == nil {
			if err := a.connectWayland(waylandPath); err == nil {
				a.displayServer = DisplayServerWayland
				if a.verbose {
					log.Printf("wain: connected to Wayland display: %s", waylandPath)
				}
				return nil
			} else if a.verbose {
				log.Printf("wain: Wayland connection failed: %v", err)
			}
		}
	}

	// Fall back to X11
	x11Display := os.Getenv("DISPLAY")
	if x11Display == "" {
		x11Display = ":0"
	}

	// Extract display number
	displayNum := "0"
	if len(x11Display) > 1 && x11Display[0] == ':' {
		displayNum = x11Display[1:]
	}

	if err := a.connectX11(displayNum); err != nil {
		return fmt.Errorf("failed to connect to any display server: %w", err)
	}

	a.displayServer = DisplayServerX11
	if a.verbose {
		log.Printf("wain: connected to X11 display: %s", displayNum)
	}
	return nil
}

// connectWayland establishes a Wayland connection.
func (a *App) connectWayland(path string) error {
	conn, err := client.Connect(path)
	if err != nil {
		return err
	}
	a.waylandConn = conn

	// Get registry via Display
	registry, err := conn.Display().GetRegistry()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}
	a.waylandRegistry = registry

	// Bind compositor
	compositorGlobal := registry.FindGlobal("wl_compositor")
	if compositorGlobal == nil {
		return fmt.Errorf("wl_compositor not found")
	}
	compositor, err := registry.BindCompositor(compositorGlobal)
	if err != nil {
		return fmt.Errorf("failed to bind compositor: %w", err)
	}
	a.waylandCompositor = compositor

	// Bind shm
	shmGlobal := registry.FindGlobal("wl_shm")
	if shmGlobal == nil {
		return fmt.Errorf("wl_shm not found")
	}
	shmID, err := registry.Bind(shmGlobal.Name, "wl_shm", shmGlobal.Version)
	if err != nil {
		return fmt.Errorf("failed to bind shm: %w", err)
	}
	shmObj := shm.NewSHM(conn, shmID)
	conn.RegisterObject(shmObj)
	a.waylandShm = shmObj

	// Bind xdg_wm_base
	xdgGlobal := registry.FindGlobal("xdg_wm_base")
	if xdgGlobal == nil {
		return fmt.Errorf("xdg_wm_base not found")
	}
	wmBaseID, _, err := registry.BindXdgWmBase(xdgGlobal)
	if err != nil {
		return fmt.Errorf("failed to bind xdg_wm_base: %w", err)
	}
	wmBase := xdg.NewWmBase(conn, wmBaseID, xdgGlobal.Version)
	conn.RegisterObject(wmBase)
	a.waylandWmBase = wmBase

	return nil
}

// connectX11 establishes an X11 connection.
func (a *App) connectX11(display string) error {
	conn, err := x11client.Connect(display)
	if err != nil {
		return err
	}
	a.x11Conn = conn
	return nil
}

// initWindow creates a window or surface on the display server.
func (a *App) initWindow() error {
	switch a.displayServer {
	case DisplayServerWayland:
		return a.initWaylandSurface()
	case DisplayServerX11:
		return a.initX11Window()
	default:
		return ErrNoDisplay
	}
}

// initWaylandSurface creates a Wayland surface and XDG toplevel.
func (a *App) initWaylandSurface() error {
	// Create surface
	surface, err := a.waylandCompositor.CreateSurface()
	if err != nil {
		return fmt.Errorf("failed to create surface: %w", err)
	}
	a.waylandSurface = surface

	// Create XDG surface
	xdgSurface, err := a.waylandWmBase.GetXdgSurface(surface.ID())
	if err != nil {
		return fmt.Errorf("failed to create xdg_surface: %w", err)
	}
	a.waylandXdgSurface = xdgSurface

	// Create XDG toplevel
	toplevel, err := xdgSurface.GetToplevel()
	if err != nil {
		return fmt.Errorf("failed to create xdg_toplevel: %w", err)
	}
	a.waylandToplevel = toplevel

	// Set window title
	if err := toplevel.SetTitle("wain application"); err != nil {
		return fmt.Errorf("failed to set title: %w", err)
	}

	// Commit surface
	if err := surface.Commit(); err != nil {
		return fmt.Errorf("failed to commit surface: %w", err)
	}

	return nil
}

// initX11Window creates an X11 window.
func (a *App) initX11Window() error {
	// Allocate window XID
	wid, err := a.x11Conn.AllocXID()
	if err != nil {
		return fmt.Errorf("failed to allocate window XID: %w", err)
	}
	a.x11Window = wid

	// Get root window
	root := a.x11Conn.RootWindow()

	// Create window
	_, err = a.x11Conn.CreateWindow(
		root,          // parent
		0, 0,          // x, y
		uint16(a.width), uint16(a.height),
		0,             // border width
		1,             // class: InputOutput
		0,             // visual: CopyFromParent
		0,             // value mask
		nil,           // attributes
	)
	if err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}

	// Map window
	if err := a.x11Conn.MapWindow(wid); err != nil {
		return fmt.Errorf("failed to map window: %w", err)
	}

	return nil
}

// initRenderer initializes the rendering backend with auto-detection.
func (a *App) initRenderer() error {
	// Create a default atlas for text rendering
	atlas, err := text.NewAtlas()
	if err != nil {
		return fmt.Errorf("failed to create font atlas: %w", err)
	}

	cfg := backend.AutoConfig{
		DRMPath:          "/dev/dri/renderD128",
		Width:            a.width,
		Height:           a.height,
		VertexBufferSize: 1024 * 1024,
		Atlas:            atlas,
		ForceSoftware:    false,
		Verbose:          a.verbose,
	}

	renderer, backendType, err := backend.NewRenderer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}

	a.renderer = renderer
	a.backendType = backendType

	if a.verbose {
		log.Printf("wain: using %s rendering backend", backendType)
	}

	return nil
}

// renderInitialFrame renders a blank initial frame.
func (a *App) renderInitialFrame() error {
	// Clear display list
	a.displayList = displaylist.New()

	// Render blank frame
	if err := a.renderer.Render(a.displayList); err != nil {
		return fmt.Errorf("initial render failed: %w", err)
	}

	return nil
}

// eventLoop runs the main event loop.
func (a *App) eventLoop() error {
	for {
		a.mu.Lock()
		if a.shouldQuit {
			a.mu.Unlock()
			break
		}
		a.mu.Unlock()

		// Process events
		if err := a.processEvents(); err != nil {
			return err
		}
	}
	return nil
}

// processEvents processes pending events from the display server.
func (a *App) processEvents() error {
	switch a.displayServer {
	case DisplayServerWayland:
		// Simple polling for now - just keep event loop running
		return nil
	case DisplayServerX11:
		// Simple polling for now
		return nil
	default:
		return ErrNoDisplay
	}
}

// cleanup releases all resources.
func (a *App) cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.renderer != nil {
		a.renderer.Destroy()
		a.renderer = nil
	}

	if a.waylandConn != nil {
		a.waylandConn.Close()
		a.waylandConn = nil
	}

	if a.x11Conn != nil {
		a.x11Conn.Close()
		a.x11Conn = nil
	}

	a.running = false
	a.initialized = false
}
