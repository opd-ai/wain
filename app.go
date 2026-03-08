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
	"time"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/render/backend"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/shm"
	"github.com/opd-ai/wain/internal/wayland/xdg"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/wire"
)

var (
	// ErrNotRunning is returned when calling methods that require Run() to be called first.
	ErrNotRunning = errors.New("wain: app not running")

	// ErrAlreadyRunning is returned when Run() is called multiple times.
	ErrAlreadyRunning = errors.New("wain: app already running")

	// ErrNoDisplay is returned when no display server is available.
	ErrNoDisplay = errors.New("wain: no display server available")

	// ErrInvalidWindowConfig is returned when window configuration is invalid.
	ErrInvalidWindowConfig = errors.New("wain: invalid window configuration")
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
	waylandRegistry   *client.Registry
	waylandCompositor *client.Compositor
	waylandShm        *shm.SHM
	waylandWmBase     *xdg.WmBase
	waylandSurface    *client.Surface
	waylandXdgSurface *xdg.Surface
	waylandToplevel   *xdg.Toplevel

	// X11-specific objects
	x11Window x11client.XID
	x11GC     x11client.XID

	// Rendering backend
	renderer    backend.Renderer
	backendType backend.BackendType
	displayList *displaylist.DisplayList

	// Windows
	windows []*Window

	// State
	running     bool
	shouldQuit  bool
	initialized bool
	width       int
	height      int
	verbose     bool
	drmPath     string
	forceSW     bool
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
		drmPath:     cfg.DRMPath,
		forceSW:     cfg.ForceSoftware,
		displayList: displaylist.New(),
	}
}

// WindowConfig contains configuration options for creating a Window.
type WindowConfig struct {
	// Title is the window title displayed in the title bar (default: "").
	Title string

	// Width is the initial window width in pixels (default: 800).
	Width int

	// Height is the initial window height in pixels (default: 600).
	Height int

	// MinWidth is the minimum window width in pixels (default: 0, no minimum).
	MinWidth int

	// MinHeight is the minimum window height in pixels (default: 0, no minimum).
	MinHeight int

	// MaxWidth is the maximum window width in pixels (default: 0, no maximum).
	MaxWidth int

	// MaxHeight is the maximum window height in pixels (default: 0, no maximum).
	MaxHeight int

	// Fullscreen indicates whether the window should start in fullscreen mode (default: false).
	Fullscreen bool

	// Decorations indicates whether the window should have decorations (title bar, borders) (default: true).
	Decorations bool
}

// Window represents a UI window with platform-agnostic operations.
type Window struct {
	app *App
	mu  sync.Mutex

	// Window properties
	title       string
	width       int
	height      int
	minWidth    int
	minHeight   int
	maxWidth    int
	maxHeight   int
	fullscreen  bool
	decorations bool
	closed      bool
	focused     bool
	scale       float64

	// Platform-specific objects (Wayland)
	waylandSurface    *client.Surface
	waylandXdgSurface *xdg.Surface
	waylandToplevel   *xdg.Toplevel

	// Platform-specific objects (X11)
	x11Window x11client.XID
	x11GC     x11client.XID

	// Event handlers
	onResize      func(width, height int)
	onClose       func()
	onFocus       func(focused bool)
	onScaleChange func(scale float64)
	onPointer     func(*PointerEvent)
	onKeyPress    func(*KeyEvent)
	onKeyRelease  func(*KeyEvent)
	onTouch       func(*TouchEvent)

	// Event dispatcher
	dispatcher *EventDispatcher

	// Root widget for hit-testing
	rootWidget Widget
}

// NewWindow creates a new window with the specified configuration.
// The app must be running (Run() must be called first) before creating windows.
func (a *App) NewWindow(cfg WindowConfig) (*Window, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil, ErrNotRunning
	}

	if cfg.Width <= 0 {
		cfg.Width = 800
	}
	if cfg.Height <= 0 {
		cfg.Height = 600
	}

	if cfg.MinWidth < 0 || cfg.MinHeight < 0 || cfg.MaxWidth < 0 || cfg.MaxHeight < 0 {
		return nil, ErrInvalidWindowConfig
	}

	if cfg.MaxWidth > 0 && cfg.Width > cfg.MaxWidth {
		cfg.Width = cfg.MaxWidth
	}
	if cfg.MaxHeight > 0 && cfg.Height > cfg.MaxHeight {
		cfg.Height = cfg.MaxHeight
	}
	if cfg.MinWidth > 0 && cfg.Width < cfg.MinWidth {
		cfg.Width = cfg.MinWidth
	}
	if cfg.MinHeight > 0 && cfg.Height < cfg.MinHeight {
		cfg.Height = cfg.MinHeight
	}

	win := &Window{
		app:         a,
		title:       cfg.Title,
		width:       cfg.Width,
		height:      cfg.Height,
		minWidth:    cfg.MinWidth,
		minHeight:   cfg.MinHeight,
		maxWidth:    cfg.MaxWidth,
		maxHeight:   cfg.MaxHeight,
		fullscreen:  cfg.Fullscreen,
		decorations: cfg.Decorations,
		scale:       1.0,
		dispatcher:  NewEventDispatcher(),
	}

	if err := win.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize window: %w", err)
	}

	a.windows = append(a.windows, win)
	return win, nil
}

// initialize creates the platform-specific window.
func (w *Window) initialize() error {
	switch w.app.displayServer {
	case DisplayServerWayland:
		return w.initWaylandWindow()
	case DisplayServerX11:
		return w.initX11Window()
	default:
		return ErrNoDisplay
	}
}

// initWaylandWindow creates a Wayland surface and toplevel.
func (w *Window) initWaylandWindow() error {
	surface, err := w.app.waylandCompositor.CreateSurface()
	if err != nil {
		return fmt.Errorf("failed to create surface: %w", err)
	}
	w.waylandSurface = surface

	xdgSurface, err := w.app.waylandWmBase.GetXdgSurface(surface.ID())
	if err != nil {
		return fmt.Errorf("failed to create xdg_surface: %w", err)
	}
	w.waylandXdgSurface = xdgSurface

	toplevel, err := xdgSurface.GetToplevel()
	if err != nil {
		return fmt.Errorf("failed to create xdg_toplevel: %w", err)
	}
	w.waylandToplevel = toplevel

	if w.title != "" {
		if err := toplevel.SetTitle(w.title); err != nil {
			return fmt.Errorf("failed to set title: %w", err)
		}
	}

	if w.minWidth > 0 || w.minHeight > 0 {
		if err := toplevel.SetMinSize(int32(w.minWidth), int32(w.minHeight)); err != nil {
			return fmt.Errorf("failed to set min size: %w", err)
		}
	}

	if w.maxWidth > 0 || w.maxHeight > 0 {
		if err := toplevel.SetMaxSize(int32(w.maxWidth), int32(w.maxHeight)); err != nil {
			return fmt.Errorf("failed to set max size: %w", err)
		}
	}

	if w.fullscreen {
		if err := toplevel.SetFullscreen(0); err != nil {
			return fmt.Errorf("failed to set fullscreen: %w", err)
		}
	}

	if err := surface.Commit(); err != nil {
		return fmt.Errorf("failed to commit surface: %w", err)
	}

	return nil
}

// initX11Window creates an X11 window.
func (w *Window) initX11Window() error {
	wid, err := w.app.x11Conn.AllocXID()
	if err != nil {
		return fmt.Errorf("failed to allocate window XID: %w", err)
	}
	w.x11Window = wid

	root := w.app.x11Conn.RootWindow()

	_, err = w.app.x11Conn.CreateWindow(
		root,
		0, 0,
		uint16(w.width), uint16(w.height),
		0,
		wire.WindowClassInputOutput,
		0,
		wire.CWBackPixel|wire.CWEventMask,
		[]uint32{0x000000, wire.EventMaskExposure | wire.EventMaskStructureNotify | wire.EventMaskKeyPress | wire.EventMaskKeyRelease | wire.EventMaskButtonPress | wire.EventMaskButtonRelease | wire.EventMaskPointerMotion},
	)
	if err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}

	if err := w.app.x11Conn.MapWindow(w.x11Window); err != nil {
		return fmt.Errorf("failed to map window: %w", err)
	}

	return nil
}

// SetTitle sets the window title.
func (w *Window) SetTitle(title string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("window is closed")
	}

	w.title = title

	switch w.app.displayServer {
	case DisplayServerWayland:
		if w.waylandToplevel != nil {
			return w.waylandToplevel.SetTitle(title)
		}
	case DisplayServerX11:
		// X11 title setting would require additional protocol implementation
		// This is a simplified placeholder
		return nil
	}

	return nil
}

// SetSize sets the window size in pixels.
func (w *Window) SetSize(width, height int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("window is closed")
	}

	if w.maxWidth > 0 && width > w.maxWidth {
		width = w.maxWidth
	}
	if w.maxHeight > 0 && height > w.maxHeight {
		height = w.maxHeight
	}
	if w.minWidth > 0 && width < w.minWidth {
		width = w.minWidth
	}
	if w.minHeight > 0 && height < w.minHeight {
		height = w.minHeight
	}

	w.width = width
	w.height = height

	switch w.app.displayServer {
	case DisplayServerWayland:
		// Wayland doesn't allow client-side resize directly
		// Size changes come from configure events
		return nil
	case DisplayServerX11:
		if w.x11Window != 0 {
			return w.app.x11Conn.ConfigureWindow(
				w.x11Window,
				x11client.ConfigMaskWidth|x11client.ConfigMaskHeight,
				[]uint32{uint32(width), uint32(height)},
			)
		}
	}

	return nil
}

// SetMinSize sets the minimum window size in pixels.
func (w *Window) SetMinSize(width, height int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("window is closed")
	}

	w.minWidth = width
	w.minHeight = height

	switch w.app.displayServer {
	case DisplayServerWayland:
		if w.waylandToplevel != nil {
			return w.waylandToplevel.SetMinSize(int32(width), int32(height))
		}
	case DisplayServerX11:
		// X11 size hints would require additional protocol implementation
		return nil
	}

	return nil
}

// SetMaxSize sets the maximum window size in pixels.
func (w *Window) SetMaxSize(width, height int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("window is closed")
	}

	w.maxWidth = width
	w.maxHeight = height

	switch w.app.displayServer {
	case DisplayServerWayland:
		if w.waylandToplevel != nil {
			return w.waylandToplevel.SetMaxSize(int32(width), int32(height))
		}
	case DisplayServerX11:
		// X11 size hints would require additional protocol implementation
		return nil
	}

	return nil
}

// SetFullscreen sets the window fullscreen state.
func (w *Window) SetFullscreen(fullscreen bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("window is closed")
	}

	w.fullscreen = fullscreen

	switch w.app.displayServer {
	case DisplayServerWayland:
		if w.waylandToplevel != nil {
			if fullscreen {
				return w.waylandToplevel.SetFullscreen(0)
			}
			return w.waylandToplevel.UnsetFullscreen()
		}
	case DisplayServerX11:
		// X11 fullscreen would require WM hints (_NET_WM_STATE_FULLSCREEN)
		return nil
	}

	return nil
}

// Close closes the window and releases its resources.
func (w *Window) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true

	switch w.app.displayServer {
	case DisplayServerWayland:
		// Wayland cleanup would happen here
		// For now, we just mark it closed
	case DisplayServerX11:
		if w.x11Window != 0 {
			if err := w.app.x11Conn.DestroyWindow(w.x11Window); err != nil {
				return err
			}
		}
	}

	if w.onClose != nil {
		w.onClose()
	}

	return nil
}

// Size returns the current window size in pixels.
func (w *Window) Size() (width, height int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.width, w.height
}

// Title returns the current window title.
func (w *Window) Title() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.title
}

// IsFullscreen returns whether the window is in fullscreen mode.
func (w *Window) IsFullscreen() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.fullscreen
}

// IsClosed returns whether the window is closed.
func (w *Window) IsClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

// IsFocused returns whether the window has keyboard focus.
func (w *Window) IsFocused() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.focused
}

// Scale returns the current window scale factor for HiDPI displays.
func (w *Window) Scale() float64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.scale
}

// OnResize sets the callback for window resize events.
func (w *Window) OnResize(callback func(width, height int)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onResize = callback
}

// OnClose sets the callback for window close events.
func (w *Window) OnClose(callback func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onClose = callback
}

// OnFocus sets the callback for window focus events.
func (w *Window) OnFocus(callback func(focused bool)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onFocus = callback
}

// OnScaleChange sets the callback for window scale change events (HiDPI).
func (w *Window) OnScaleChange(callback func(scale float64)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onScaleChange = callback
}

// OnPointer sets the callback for pointer (mouse/touchpad) events.
func (w *Window) OnPointer(callback func(*PointerEvent)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onPointer = callback
}

// OnKeyPress sets the callback for key press events.
func (w *Window) OnKeyPress(callback func(*KeyEvent)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onKeyPress = callback
}

// OnKeyRelease sets the callback for key release events.
func (w *Window) OnKeyRelease(callback func(*KeyEvent)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onKeyRelease = callback
}

// OnTouch sets the callback for touch events.
func (w *Window) OnTouch(callback func(*TouchEvent)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onTouch = callback
}

// SetRootWidget sets the root widget for event hit-testing.
func (w *Window) SetRootWidget(widget Widget) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rootWidget = widget
	if w.dispatcher != nil {
		w.dispatcher.SetWidgetRoot(widget)
	}
}

// Dispatcher returns the window's event dispatcher.
func (w *Window) Dispatcher() *EventDispatcher {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.dispatcher
}

// SendCustomEvent injects a custom event into the event loop.
func (w *Window) SendCustomEvent(data interface{}) {
	evt := &CustomEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		data:      data,
	}
	if w.dispatcher != nil {
		w.dispatcher.Dispatch(evt)
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
	if a.tryWaylandConnection() {
		return nil
	}

	// Fall back to X11
	return a.tryX11Connection()
}

// tryWaylandConnection attempts to connect to Wayland.
// Returns true if successful.
func (a *App) tryWaylandConnection() bool {
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	if waylandDisplay == "" {
		waylandDisplay = "wayland-0"
	}

	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntimeDir == "" {
		return false
	}

	waylandPath := fmt.Sprintf("%s/%s", xdgRuntimeDir, waylandDisplay)
	if _, err := os.Stat(waylandPath); err != nil {
		return false
	}

	if err := a.connectWayland(waylandPath); err != nil {
		if a.verbose {
			log.Printf("wain: Wayland connection failed: %v", err)
		}
		return false
	}

	a.displayServer = DisplayServerWayland
	if a.verbose {
		log.Printf("wain: connected to Wayland display: %s", waylandPath)
	}
	return true
}

// tryX11Connection attempts to connect to X11.
// Returns an error if connection fails.
func (a *App) tryX11Connection() error {
	x11Display := os.Getenv("DISPLAY")
	if x11Display == "" {
		x11Display = ":0"
	}

	displayNum := a.extractX11DisplayNumber(x11Display)

	if err := a.connectX11(displayNum); err != nil {
		return fmt.Errorf("failed to connect to any display server: %w", err)
	}

	a.displayServer = DisplayServerX11
	if a.verbose {
		log.Printf("wain: connected to X11 display: %s", displayNum)
	}
	return nil
}

// extractX11DisplayNumber extracts the display number from DISPLAY env var.
func (a *App) extractX11DisplayNumber(display string) string {
	if len(display) > 1 && display[0] == ':' {
		return display[1:]
	}
	return "0"
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
		root, // parent
		0, 0, // x, y
		uint16(a.width), uint16(a.height),
		0,   // border width
		1,   // class: InputOutput
		0,   // visual: CopyFromParent
		0,   // value mask
		nil, // attributes
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
