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
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/render/backend"
	"github.com/opd-ai/wain/internal/render/display"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/datadevice"
	"github.com/opd-ai/wain/internal/wayland/dmabuf"
	"github.com/opd-ai/wain/internal/wayland/input"
	"github.com/opd-ai/wain/internal/wayland/shm"
	wlwire "github.com/opd-ai/wain/internal/wayland/wire"
	"github.com/opd-ai/wain/internal/wayland/xdg"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	x11events "github.com/opd-ai/wain/internal/x11/events"
	"github.com/opd-ai/wain/internal/x11/selection"
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

// Presenter is implemented by platform-specific types that commit rendered
// pixels to the compositor or display server each frame.
type Presenter interface {
	// Present submits the current frame to the compositor.
	Present(ctx context.Context) error
	// Close releases platform resources held by the presenter.
	Close() error
}

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

// String returns a human-readable name for the display server.
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
	waylandSeat       *input.Seat
	waylandKeyboard   *input.Keyboard
	waylandPointer    *input.Pointer
	waylandDmabuf     *dmabuf.Dmabuf // optional — nil when compositor lacks zwp_linux_dmabuf_v1

	// X11-specific objects
	x11Window       x11client.XID
	x11GC           x11client.XID
	x11SelectionMgr *selection.Manager

	// Wayland clipboard objects
	waylandDataDeviceMgr *datadevice.Manager
	waylandDataDevice    *datadevice.Device

	// Rendering backend
	renderer    backend.Renderer
	backendType backend.BackendType
	displayList *displaylist.DisplayList

	// Windows
	windows         []*Window
	surfaceToWindow map[uint32]*Window // Wayland surface ID to Window mapping

	// Resource management
	resources *ResourceManager

	// Theming
	theme Theme

	// State
	running     bool
	shouldQuit  bool
	initialized bool
	width       int
	height      int
	verbose     bool
	drmPath     string
	forceSW     bool

	// Cross-goroutine notification
	notifyChan chan func()
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
		width:           cfg.Width,
		height:          cfg.Height,
		verbose:         cfg.Verbose,
		drmPath:         cfg.DRMPath,
		forceSW:         cfg.ForceSoftware,
		displayList:     displaylist.New(),
		theme:           DefaultDark(),
		notifyChan:      make(chan func(), 100),
		surfaceToWindow: make(map[uint32]*Window),
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

	// Render bridge
	renderBridge *RenderBridge

	// presenter submits rendered frames to the compositor / display server.
	presenter Presenter
}

// NewWindow creates a new window with the specified configuration.
// The app must be running (Run() must be called first) before creating windows.
func (a *App) NewWindow(cfg WindowConfig) (*Window, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil, ErrNotRunning
	}

	if err := validateAndNormalizeConfig(&cfg); err != nil {
		return nil, err
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

	win.renderBridge = NewRenderBridge(a.renderer)

	a.windows = append(a.windows, win)
	return win, nil
}

// clampDimension clamps value within [minVal, maxVal].
// A constraint of 0 means unconstrained in that direction.
func clampDimension(value, minVal, maxVal int) int {
	if maxVal > 0 && value > maxVal {
		value = maxVal
	}
	if minVal > 0 && value < minVal {
		value = minVal
	}
	return value
}

// validateAndNormalizeConfig validates and normalizes window configuration.
func validateAndNormalizeConfig(cfg *WindowConfig) error {
	if cfg.Width <= 0 {
		cfg.Width = 800
	}
	if cfg.Height <= 0 {
		cfg.Height = 600
	}

	if cfg.MinWidth < 0 || cfg.MinHeight < 0 || cfg.MaxWidth < 0 || cfg.MaxHeight < 0 {
		return ErrInvalidWindowConfig
	}

	cfg.Width = clampDimension(cfg.Width, cfg.MinWidth, cfg.MaxWidth)
	cfg.Height = clampDimension(cfg.Height, cfg.MinHeight, cfg.MaxHeight)

	return nil
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
	if err := w.createWaylandSurface(); err != nil {
		return fmt.Errorf("app: init wayland window: create surface: %w", err)
	}

	if err := w.configureWaylandToplevel(); err != nil {
		return fmt.Errorf("app: init wayland window: configure toplevel: %w", err)
	}

	if err := w.waylandSurface.Commit(); err != nil {
		return fmt.Errorf("failed to commit surface: %w", err)
	}

	w.initWaylandPresenter()
	return nil
}

// initWaylandPresenter selects and initialises the presenter for a Wayland window.
// Software backend uses SHM; GPU backend uses DMA-BUF when available.
func (w *Window) initWaylandPresenter() {
	if sw, ok := w.app.renderer.(*backend.SoftwareBackend); ok {
		w.presenter = display.NewSoftwareWaylandPresenter(w.app.waylandShm, w.waylandSurface, sw)
		return
	}
	if _, ok := w.app.renderer.(*backend.GPUBackend); !ok || w.app.waylandDmabuf == nil {
		return
	}
	pipeline, err := display.NewWaylandPipeline(w.waylandSurface, w.app.waylandDmabuf, w.app.renderer)
	if err != nil {
		log.Printf("Warning: GPU Wayland pipeline unavailable: %v", err)
		return
	}
	w.presenter = display.NewGPUWaylandPresenter(pipeline)
}

// createWaylandSurface creates the Wayland surface and toplevel objects.
func (w *Window) createWaylandSurface() error {
	surface, err := w.app.waylandCompositor.CreateSurface()
	if err != nil {
		return fmt.Errorf("failed to create surface: %w", err)
	}
	w.waylandSurface = surface

	// Register surface-to-window mapping for event routing
	w.app.mu.Lock()
	w.app.surfaceToWindow[surface.ID()] = w
	w.app.mu.Unlock()

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

	return nil
}

// configureWaylandToplevel configures window properties on the Wayland toplevel.
func (w *Window) configureWaylandToplevel() error {
	toplevel := w.waylandToplevel

	if w.title != "" {
		if err := toplevel.SetTitle(w.title); err != nil {
			return fmt.Errorf("failed to set title: %w", err)
		}
	}

	if err := w.setWaylandSizeLimits(toplevel); err != nil {
		return fmt.Errorf("app: configure wayland toplevel: set size limits: %w", err)
	}

	if w.fullscreen {
		if err := toplevel.SetFullscreen(0); err != nil {
			return fmt.Errorf("failed to set fullscreen: %w", err)
		}
	}

	return nil
}

// setWaylandSizeLimits sets min/max size constraints on a Wayland toplevel.
func (w *Window) setWaylandSizeLimits(toplevel *xdg.Toplevel) error {
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

	w.initX11Presenter()
	return nil
}

// initX11Presenter selects and initialises the presenter for an X11 window.
func (w *Window) initX11Presenter() {
	if sw, ok := w.app.renderer.(*backend.SoftwareBackend); ok {
		w.tryInitSoftwareX11Presenter(sw)
		return
	}
	if _, ok := w.app.renderer.(*backend.GPUBackend); ok {
		w.tryInitGPUX11Presenter()
	}
}

// tryInitSoftwareX11Presenter initialises a software-rendered X11 presenter.
func (w *Window) tryInitSoftwareX11Presenter(sw *backend.SoftwareBackend) {
	p, err := display.NewSoftwareX11Presenter(w.app.x11Conn, w.x11Window, sw)
	if err != nil {
		if w.app.verbose {
			log.Printf("Warning: failed to create X11 presenter: %v", err)
		}
		return
	}
	w.presenter = p
}

// tryInitGPUX11Presenter initialises a GPU-accelerated X11 presenter.
func (w *Window) tryInitGPUX11Presenter() {
	p, err := display.NewGPUX11PresenterFromConn(w.app.x11Conn, w.x11Window, w.app.renderer)
	if err != nil {
		if w.app.verbose {
			log.Printf("Warning: GPU X11 pipeline unavailable: %v", err)
		}
		return
	}
	w.presenter = p
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
		if w.x11Window != 0 {
			if err := w.x11SetTitle(title); err != nil {
				return fmt.Errorf("app: set title: x11: %w", err)
			}
		}
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

	width = clampDimension(width, w.minWidth, w.maxWidth)
	height = clampDimension(height, w.minHeight, w.maxHeight)

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
		if w.x11Window != 0 {
			if err := w.x11SetWMNormalHints(); err != nil {
				return fmt.Errorf("app: set min size: x11 hints: %w", err)
			}
		}
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
		if w.x11Window != 0 {
			if err := w.x11SetWMNormalHints(); err != nil {
				return fmt.Errorf("app: set max size: x11 hints: %w", err)
			}
		}
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
		return w.applyWaylandFullscreen(fullscreen)
	case DisplayServerX11:
		if w.x11Window != 0 {
			if err := w.x11SetFullscreen(fullscreen); err != nil {
				return fmt.Errorf("app: set fullscreen: x11: %w", err)
			}
		}
	}

	return nil
}

// applyWaylandFullscreen sends the appropriate fullscreen request to the Wayland
// toplevel surface, or does nothing if the toplevel is not yet initialised.
func (w *Window) applyWaylandFullscreen(fullscreen bool) error {
	if w.waylandToplevel == nil {
		return nil
	}
	if fullscreen {
		return w.waylandToplevel.SetFullscreen(0)
	}
	return w.waylandToplevel.UnsetFullscreen()
}

// x11SetTitle sets WM_NAME (STRING) and _NET_WM_NAME (UTF8_STRING) on the X11
// window so that both legacy and modern window managers show the correct title.
func (w *Window) x11SetTitle(title string) error {
	if err := w.app.x11Conn.ChangeProperty(
		uint32(w.x11Window),
		wire.AtomWMName,
		wire.AtomString,
		8, 0,
		[]byte(title),
	); err != nil {
		return fmt.Errorf("failed to set WM_NAME: %w", err)
	}

	netWMName, err := w.app.x11Conn.InternAtom("_NET_WM_NAME", false)
	if err != nil {
		return fmt.Errorf("failed to intern _NET_WM_NAME: %w", err)
	}

	utf8String, err := w.app.x11Conn.InternAtom("UTF8_STRING", false)
	if err != nil {
		return fmt.Errorf("failed to intern UTF8_STRING: %w", err)
	}

	return w.app.x11Conn.ChangeProperty(
		uint32(w.x11Window),
		netWMName,
		utf8String,
		8, 0,
		[]byte(title),
	)
}

// x11SetWMNormalHints writes the WM_NORMAL_HINTS property reflecting the
// window's current min/max size constraints.
func (w *Window) x11SetWMNormalHints() error {
	hints := w.buildWMSizeHints()
	return w.app.x11Conn.ChangeProperty(
		uint32(w.x11Window),
		wire.AtomWMNormalHints,
		wire.AtomWMSizeHints,
		32, 0,
		hints,
	)
}

// buildWMSizeHints serialises the window's min/max constraints into the 72-byte
// WM_SIZE_HINTS structure defined by the ICCCM.
func (w *Window) buildWMSizeHints() []byte {
	hints := make([]byte, 72) // 18 INT32 fields × 4 bytes

	var flags uint32

	if w.minWidth > 0 || w.minHeight > 0 {
		flags |= wire.WMSizeHintsPMinSize
		binary.LittleEndian.PutUint32(hints[20:24], uint32(w.minWidth))
		binary.LittleEndian.PutUint32(hints[24:28], uint32(w.minHeight))
	}

	if w.maxWidth > 0 || w.maxHeight > 0 {
		flags |= wire.WMSizeHintsPMaxSize
		binary.LittleEndian.PutUint32(hints[28:32], uint32(w.maxWidth))
		binary.LittleEndian.PutUint32(hints[32:36], uint32(w.maxHeight))
	}

	binary.LittleEndian.PutUint32(hints[0:4], flags)

	return hints
}

// x11SetFullscreen sends a _NET_WM_STATE ClientMessage to the root window to
// add or remove the fullscreen state per the Extended Window Manager Hints spec.
func (w *Window) x11SetFullscreen(fullscreen bool) error {
	netWMState, err := w.app.x11Conn.InternAtom("_NET_WM_STATE", false)
	if err != nil {
		return fmt.Errorf("failed to intern _NET_WM_STATE: %w", err)
	}

	netWMStateFullscreen, err := w.app.x11Conn.InternAtom("_NET_WM_STATE_FULLSCREEN", false)
	if err != nil {
		return fmt.Errorf("failed to intern _NET_WM_STATE_FULLSCREEN: %w", err)
	}

	const (
		netWMStateRemove = 0
		netWMStateAdd    = 1
	)

	action := uint32(netWMStateRemove)
	if fullscreen {
		action = netWMStateAdd
	}

	// ClientMessage event (32 bytes) per EWMH §2.
	event := make([]byte, 32)
	event[0] = 33 // ClientMessage type
	event[1] = 32 // format: 32-bit data elements
	binary.LittleEndian.PutUint32(event[4:8], uint32(w.x11Window))
	binary.LittleEndian.PutUint32(event[8:12], netWMState)
	binary.LittleEndian.PutUint32(event[12:16], action)
	binary.LittleEndian.PutUint32(event[16:20], netWMStateFullscreen)

	root := uint32(w.app.x11Conn.RootWindow())

	const (
		substructureRedirectMask = 1 << 20
		substructureNotifyMask   = 1 << 19
	)

	return w.app.x11Conn.SendEvent(root, false, substructureRedirectMask|substructureNotifyMask, event)
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
				return fmt.Errorf("app: close window: destroy x11 window: %w", err)
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
	if w.renderBridge != nil {
		w.renderBridge.MarkDirty()
	}
}

// SetLayout attaches a PublicWidget tree as the root layout for the window.
//
// SetLayout is the preferred way for application code to attach widgets created
// with the public API (wain.Column, wain.Row, wain.Button, etc.) to a window.
// Internally it wraps the widget with an adapter so it satisfies the renderer's
// Widget interface.
//
// Example:
//
//	col := wain.NewColumn()
//	col.Add(wain.NewButton("OK", wain.Size{Width: 50, Height: 10}))
//	win.SetLayout(col)
func (w *Window) SetLayout(root PublicWidget) {
	w.SetRootWidget(newLayoutAdapter(root))
}

// Redraw marks the entire window as needing a full redraw.
func (w *Window) Redraw() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.renderBridge != nil {
		w.renderBridge.MarkDirty()
	}
}

// RedrawRegion marks a specific region as needing a redraw.
func (w *Window) RedrawRegion(x, y, width, height int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.renderBridge != nil {
		w.renderBridge.MarkRegionDirty(x, y, width, height)
	}
}

// RenderFrame renders the current widget tree to the display.
// This is called automatically by the event loop, but can be called
// manually to force an immediate render.
func (w *Window) RenderFrame() error {
	w.mu.Lock()
	rootWidget := w.rootWidget
	renderBridge := w.renderBridge
	presenter := w.presenter
	w.mu.Unlock()

	if renderBridge == nil {
		return ErrNoRenderer
	}

	if err := renderBridge.Render(rootWidget); err != nil {
		return fmt.Errorf("app: render: %w", err)
	}

	if presenter != nil {
		if err := presenter.Present(context.Background()); err != nil {
			return fmt.Errorf("present frame: %w", err)
		}
	}

	return nil
}

// Dispatcher returns the window's event dispatcher.
func (w *Window) Dispatcher() *EventDispatcher {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.dispatcher
}

// SendCustomEvent injects a custom event into the event loop.
func (w *Window) SendCustomEvent(data CustomEventPayload) {
	evt := &CustomEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		data:      data,
	}
	if w.dispatcher != nil {
		w.dispatcher.Dispatch(evt)
	}
}

// handleX11Event processes an X11 event for this window.
func (w *Window) handleX11Event(eventType x11events.EventType, eventBuf []byte) error {
	reader := bytes.NewReader(eventBuf)
	header, data, err := wire.DecodeEventHeader(reader)
	if err != nil {
		return fmt.Errorf("decode event header: %w", err)
	}

	switch eventType {
	case x11events.EventTypeKeyPress:
		return w.handleX11KeyPress(header, data)
	case x11events.EventTypeKeyRelease:
		return w.handleX11KeyRelease(header, data)
	case x11events.EventTypeButtonPress:
		return w.handleX11ButtonPress(header, data)
	case x11events.EventTypeButtonRelease:
		return w.handleX11ButtonRelease(header, data)
	case x11events.EventTypeMotionNotify:
		return w.handleX11Motion(header, data)
	case x11events.EventTypeConfigureNotify:
		return w.handleX11Configure(header, data)
	}

	return nil
}

// handleX11KeyPress processes an X11 KeyPress event.
func (w *Window) handleX11KeyPress(header wire.EventHeader, data []byte) error {
	e, err := x11events.ParseKeyPressEvent(header, data)
	if err != nil {
		return fmt.Errorf("app: handle x11 key press: parse event: %w", err)
	}
	evt := translateX11KeyPressEvent(e)
	w.dispatchEvent(evt)
	return nil
}

// handleX11KeyRelease processes an X11 KeyRelease event.
func (w *Window) handleX11KeyRelease(header wire.EventHeader, data []byte) error {
	e, err := x11events.ParseKeyReleaseEvent(header, data)
	if err != nil {
		return fmt.Errorf("app: handle x11 key release: parse event: %w", err)
	}
	evt := translateX11KeyReleaseEvent(e)
	w.dispatchEvent(evt)
	return nil
}

// handleX11ButtonPress processes an X11 ButtonPress event.
func (w *Window) handleX11ButtonPress(header wire.EventHeader, data []byte) error {
	e, err := x11events.ParseButtonPressEvent(header, data)
	if err != nil {
		return fmt.Errorf("app: handle x11 button press: parse event: %w", err)
	}
	evt := translateX11ButtonPressEvent(e)
	w.dispatchEvent(evt)
	return nil
}

// handleX11ButtonRelease processes an X11 ButtonRelease event.
func (w *Window) handleX11ButtonRelease(header wire.EventHeader, data []byte) error {
	e, err := x11events.ParseButtonReleaseEvent(header, data)
	if err != nil {
		return fmt.Errorf("app: handle x11 button release: parse event: %w", err)
	}
	evt := translateX11ButtonReleaseEvent(e)
	if evt != nil {
		w.dispatchEvent(evt)
	}
	return nil
}

// handleX11Motion processes an X11 MotionNotify event.
func (w *Window) handleX11Motion(header wire.EventHeader, data []byte) error {
	e, err := x11events.ParseMotionNotifyEvent(header, data)
	if err != nil {
		return fmt.Errorf("app: handle x11 motion: parse event: %w", err)
	}
	evt := translateX11MotionNotifyEvent(e)
	w.dispatchEvent(evt)
	return nil
}

// handleX11Configure processes an X11 ConfigureNotify event.
func (w *Window) handleX11Configure(header wire.EventHeader, data []byte) error {
	e, err := x11events.ParseConfigureNotifyEvent(header, data)
	if err != nil {
		return fmt.Errorf("app: handle x11 configure: parse event: %w", err)
	}
	evt := translateX11ConfigureNotifyEvent(e)
	w.handleWindowResize(evt)
	w.dispatchEvent(evt)
	return nil
}

// dispatchEvent dispatches an event to the window's dispatcher and callbacks.
func (w *Window) dispatchEvent(evt Event) {
	if w.dispatcher != nil {
		w.dispatcher.Dispatch(evt)
	}

	// Also call legacy callbacks
	switch e := evt.(type) {
	case *PointerEvent:
		if w.onPointer != nil {
			w.onPointer(e)
		}
	case *KeyEvent:
		if e.EventType() == KeyPress && w.onKeyPress != nil {
			w.onKeyPress(e)
		} else if e.EventType() == KeyRelease && w.onKeyRelease != nil {
			w.onKeyRelease(e)
		}
	}
}

// handleWindowResize updates window dimensions from a resize event.
func (w *Window) handleWindowResize(evt *WindowEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.width = evt.Width()
	w.height = evt.Height()

	if w.renderBridge != nil {
		w.renderBridge.MarkDirty()
	}

	if w.onResize != nil {
		w.onResize(evt.Width(), evt.Height())
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

// Notify schedules a callback to be executed on the UI goroutine.
//
// This function provides safe cross-goroutine communication by ensuring
// that UI updates from background goroutines are executed on the main
// UI event loop goroutine. Callbacks are queued and executed during the
// next event loop iteration.
//
// Example usage:
//
//	go func() {
//	    result := fetchDataFromAPI()
//	    app.Notify(func() {
//	        label.SetText(result)
//	    })
//	}()
//
// The callback will be executed on the UI goroutine, making it safe to
// call any widget methods or perform UI updates.
//
// If the notification channel is full (100 pending callbacks), Notify
// will block until space is available. This prevents unbounded memory
// growth while allowing reasonable buffering.
func (a *App) Notify(callback func()) {
	if callback == nil {
		return
	}
	a.notifyChan <- callback
}

// SetTheme sets the application-wide theme.
//
// The theme controls the visual appearance of all widgets that do not have
// a StyleOverride applied. Changing the theme triggers a redraw of all
// windows on the next frame.
//
// Example:
//
//	app.SetTheme(wain.DefaultLight())
//	app.SetTheme(wain.HighContrast())
func (a *App) SetTheme(theme Theme) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.theme = theme

	// Trigger redraw of all windows
	for _, w := range a.windows {
		if w.renderBridge != nil {
			w.renderBridge.MarkDirty()
		}
	}
}

// GetTheme returns the current application-wide theme.
//
// The returned theme is a copy and can be safely modified without affecting
// the application's theme. To apply changes, use SetTheme.
func (a *App) GetTheme() Theme {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.theme
}

// Theme returns the current application-wide theme.
// This is an alias for GetTheme for convenience.
func (a *App) Theme() Theme {
	return a.GetTheme()
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
		return fmt.Errorf("app: initialize: display server: %w", err)
	}

	// Phase 2: Create window/surface
	if err := a.initWindow(); err != nil {
		return fmt.Errorf("app: initialize: window: %w", err)
	}

	// Phase 3: Initialize renderer
	if err := a.initRenderer(); err != nil {
		return fmt.Errorf("app: initialize: renderer: %w", err)
	}

	// Phase 4: Render initial frame
	if err := a.renderInitialFrame(); err != nil {
		return fmt.Errorf("app: initialize: initial frame: %w", err)
	}

	a.initialized = true
	return nil
}

// initDisplayServer detects and connects to a display server.
// Wayland is preferred, with X11 as fallback.
func (a *App) initDisplayServer() error {
	var errs []error

	// Try Wayland first
	waylandErr := a.tryWaylandConnection()
	if waylandErr == nil {
		return nil
	}
	errs = append(errs, waylandErr)

	// Fall back to X11
	x11Err := a.tryX11Connection()
	if x11Err == nil {
		return nil
	}
	errs = append(errs, x11Err)

	// Both failed - log all errors with diagnostic hints
	fmt.Fprintf(os.Stderr, "wain: failed to connect to display server\n")
	fmt.Fprintf(os.Stderr, "  Wayland failed: %v (check $WAYLAND_DISPLAY and $XDG_RUNTIME_DIR)\n", errs[0])
	fmt.Fprintf(os.Stderr, "  X11 failed: %v (check $DISPLAY)\n", errs[1])

	return fmt.Errorf("failed to connect to any display server: Wayland: %v, X11: %v", errs[0], errs[1])
}

// tryWaylandConnection attempts to connect to Wayland.
// Returns nil on success, error on failure.
func (a *App) tryWaylandConnection() error {
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	if waylandDisplay == "" {
		waylandDisplay = "wayland-0"
	}

	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntimeDir == "" {
		return fmt.Errorf("XDG_RUNTIME_DIR not set")
	}

	waylandPath := fmt.Sprintf("%s/%s", xdgRuntimeDir, waylandDisplay)
	if _, err := os.Stat(waylandPath); err != nil {
		return fmt.Errorf("Wayland socket not found at %s: %w", waylandPath, err)
	}

	if err := a.connectWayland(waylandPath); err != nil {
		return fmt.Errorf("Wayland connection to %s failed: %w", waylandPath, err)
	}

	a.displayServer = DisplayServerWayland
	if a.verbose {
		log.Printf("wain: connected to Wayland display: %s", waylandPath)
	}
	return nil
}

// tryX11Connection attempts to connect to X11.
// Returns nil on success, error on failure.
func (a *App) tryX11Connection() error {
	x11Display := os.Getenv("DISPLAY")
	if x11Display == "" {
		x11Display = ":0"
	}

	displayNum := a.extractX11DisplayNumber(x11Display)

	if err := a.connectX11(displayNum); err != nil {
		return fmt.Errorf("X11 connection to %s failed: %w", x11Display, err)
	}

	a.displayServer = DisplayServerX11
	if a.verbose {
		log.Printf("wain: connected to X11 display: %s", displayNum)
	}
	return nil
}

// extractX11DisplayNumber extracts the display number from DISPLAY env var.
// DISPLAY format is [host]:displaynumber[.screennumber] — the screen number
// must be stripped because the Unix socket path uses only the display number.
func (a *App) extractX11DisplayNumber(display string) string {
	s := display
	// Strip optional host prefix (everything before the last colon)
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		s = s[idx+1:]
	}
	// Strip optional screen number (.N suffix)
	if dotIdx := strings.Index(s, "."); dotIdx >= 0 {
		s = s[:dotIdx]
	}
	if s == "" {
		return "0"
	}
	return s
}

// connectWayland establishes a Wayland connection.
func (a *App) connectWayland(path string) error {
	conn, err := client.Connect(path)
	if err != nil {
		return fmt.Errorf("app: connect wayland: %w", err)
	}
	a.waylandConn = conn

	// Get registry via Display
	registry, err := conn.Display().GetRegistry()
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}
	a.waylandRegistry = registry

	if err := a.bindWaylandGlobals(registry); err != nil {
		return fmt.Errorf("app: connect wayland: bind globals: %w", err)
	}

	return nil
}

// bindWaylandGlobals binds required Wayland global objects.
func (a *App) bindWaylandGlobals(registry *client.Registry) error {
	if err := a.bindCompositor(registry); err != nil {
		return fmt.Errorf("app: bind wayland globals: compositor: %w", err)
	}
	if err := a.bindShellProtocols(registry); err != nil {
		return fmt.Errorf("app: bind wayland globals: shell protocols: %w", err)
	}
	// Clipboard support — optional, compositor may not expose it.
	if err := a.bindDataDeviceManager(registry); err != nil && a.verbose {
		log.Printf("Warning: clipboard unavailable: %v", err)
	}
	// DMA-BUF support — optional, required only for GPU rendering path.
	if err := a.bindDmabuf(registry); err != nil && a.verbose {
		log.Printf("Warning: DMA-BUF unavailable (GPU path disabled): %v", err)
	}
	if err := a.bindInputDevices(registry); err != nil {
		return fmt.Errorf("app: bind wayland globals: input devices: %w", err)
	}
	return nil
}

// findAndBindGlobal looks up a Wayland global by name and binds it,
// returning the assigned object ID. notFoundMsg and bindFailMsg provide
// context for error messages.
func findAndBindGlobal(registry *client.Registry, name, notFoundMsg, bindFailMsg string) (uint32, error) {
	global := registry.FindGlobal(name)
	if global == nil {
		return 0, fmt.Errorf("%s", notFoundMsg)
	}
	id, err := registry.Bind(global.Name, name, global.Version)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", bindFailMsg, err)
	}
	return id, nil
}

// bindDataDeviceManager binds wl_data_device_manager and creates a per-seat
// data device for clipboard access.  Errors are non-fatal.
func (a *App) bindDataDeviceManager(registry *client.Registry) error {
	ddmID, err := findAndBindGlobal(registry, "wl_data_device_manager",
		"wl_data_device_manager not advertised by compositor",
		"failed to bind wl_data_device_manager")
	if err != nil {
		return fmt.Errorf("app: bind data device manager: %w", err)
	}
	mgr := datadevice.NewManager(a.waylandConn, ddmID)
	a.waylandConn.RegisterObject(mgr)
	a.waylandDataDeviceMgr = mgr
	return nil
}

// bindDmabuf optionally binds zwp_linux_dmabuf_v1.
// Failure is non-fatal — the GPU presentation path is simply disabled.
func (a *App) bindDmabuf(registry *client.Registry) error {
	global := registry.FindGlobal("zwp_linux_dmabuf_v1")
	if global == nil {
		return fmt.Errorf("zwp_linux_dmabuf_v1 not advertised by compositor")
	}
	id, err := registry.Bind(global.Name, "zwp_linux_dmabuf_v1", global.Version)
	if err != nil {
		return fmt.Errorf("failed to bind zwp_linux_dmabuf_v1: %w", err)
	}
	db := dmabuf.NewDmabuf(a.waylandConn, id)
	a.waylandConn.RegisterObject(db)
	a.waylandDmabuf = db
	return nil
}

// bindCompositor binds the compositor and SHM globals.
func (a *App) bindCompositor(registry *client.Registry) error {
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
	shmID, err := findAndBindGlobal(registry, "wl_shm",
		"wl_shm not found",
		"failed to bind shm")
	if err != nil {
		return fmt.Errorf("app: bind compositor: shm: %w", err)
	}
	shmObj := shm.NewSHM(a.waylandConn, shmID)
	a.waylandConn.RegisterObject(shmObj)
	a.waylandShm = shmObj

	return nil
}

// bindShellProtocols binds xdg-shell and related window management protocols.
func (a *App) bindShellProtocols(registry *client.Registry) error {
	// Bind xdg_wm_base
	xdgGlobal := registry.FindGlobal("xdg_wm_base")
	if xdgGlobal == nil {
		return fmt.Errorf("xdg_wm_base not found")
	}
	wmBaseID, _, err := registry.BindXdgWmBase(xdgGlobal)
	if err != nil {
		return fmt.Errorf("failed to bind xdg_wm_base: %w", err)
	}
	wmBase := xdg.NewWmBase(a.waylandConn, wmBaseID, xdgGlobal.Version)
	a.waylandConn.RegisterObject(wmBase)
	a.waylandWmBase = wmBase

	return nil
}

// bindInputDevices binds seat, keyboard, and pointer input devices.
func (a *App) bindInputDevices(registry *client.Registry) error {
	// Bind seat for input
	seatGlobal := registry.FindGlobal("wl_seat")
	if seatGlobal != nil {
		seatID, err := registry.Bind(seatGlobal.Name, "wl_seat", seatGlobal.Version)
		if err != nil {
			return fmt.Errorf("failed to bind seat: %w", err)
		}
		seat := input.NewSeat(a.waylandConn, seatID, seatGlobal.Version)
		a.waylandConn.RegisterObject(seat)
		a.waylandSeat = seat

		// Get keyboard and pointer capabilities
		if err := a.setupWaylandInput(seat); err != nil {
			// Non-fatal: input is optional
			if a.verbose {
				log.Printf("Warning: failed to setup input: %v", err)
			}
		}

		a.setupWaylandDataDevice(seat)
	}

	return nil
}

// setupWaylandDataDevice creates a per-seat data device for clipboard access.
func (a *App) setupWaylandDataDevice(seat *input.Seat) {
	if a.waylandDataDeviceMgr == nil {
		return
	}
	device, err := a.waylandDataDeviceMgr.GetDataDevice(seat.ID())
	if err != nil {
		if a.verbose {
			log.Printf("Warning: failed to get data device: %v", err)
		}
		return
	}
	a.waylandConn.RegisterObject(device)
	a.waylandDataDevice = device
}

// setupWaylandInput sets up keyboard and pointer input devices and wires event callbacks.
func (a *App) setupWaylandInput(seat *input.Seat) error {
	// Get keyboard
	keyboard, err := seat.GetKeyboard()
	if err != nil {
		return fmt.Errorf("failed to get keyboard: %w", err)
	}
	a.waylandKeyboard = keyboard

	// Wire keyboard callbacks
	keyboard.SetKeyCallback(func(surfaceID, key, state uint32) {
		a.handleWaylandKeyEvent(surfaceID, key, state)
	})
	keyboard.SetEnterCallback(func(surfaceID uint32) {
		a.handleWaylandKeyboardEnter(surfaceID)
	})
	keyboard.SetLeaveCallback(func(surfaceID uint32) {
		a.handleWaylandKeyboardLeave(surfaceID)
	})

	// Get pointer
	pointer, err := seat.GetPointer()
	if err != nil {
		return fmt.Errorf("failed to get pointer: %w", err)
	}
	a.waylandPointer = pointer

	// Wire pointer callbacks
	pointer.SetButtonCallback(func(surfaceID, button, state uint32, x, y float64) {
		a.handleWaylandPointerButton(surfaceID, button, state, x, y)
	})
	pointer.SetMotionCallback(func(surfaceID uint32, x, y float64) {
		a.handleWaylandPointerMotion(surfaceID, x, y)
	})
	pointer.SetAxisCallback(func(surfaceID, axis uint32, value, x, y float64) {
		a.handleWaylandPointerAxis(surfaceID, axis, value, x, y)
	})
	pointer.SetEnterCallback(func(surfaceID uint32, x, y float64) {
		a.handleWaylandPointerEnter(surfaceID, x, y)
	})
	pointer.SetLeaveCallback(func(surfaceID uint32) {
		a.handleWaylandPointerLeave(surfaceID)
	})

	return nil
}

// lookupWindow returns the window associated with the given Wayland surface ID,
// or nil if no window is registered for that surface.
func (a *App) lookupWindow(surfaceID uint32) *Window {
	a.mu.Lock()
	win := a.surfaceToWindow[surfaceID]
	a.mu.Unlock()
	return win
}

// handleWaylandKeyEvent processes a Wayland key event.
func (a *App) handleWaylandKeyEvent(surfaceID, key, state uint32) {
	win := a.lookupWindow(surfaceID)
	if win == nil {
		return
	}
	evt := translateWaylandKeyEvent(key, state)
	win.dispatchEvent(evt)
}

// handleWaylandKeyboardEnter processes keyboard focus enter.
func (a *App) handleWaylandKeyboardEnter(surfaceID uint32) {
	win := a.lookupWindow(surfaceID)
	if win == nil {
		return
	}
	evt := &WindowEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: WindowFocus,
	}
	win.focused = true
	win.dispatchEvent(evt)
}

// handleWaylandKeyboardLeave processes keyboard focus leave.
func (a *App) handleWaylandKeyboardLeave(surfaceID uint32) {
	win := a.lookupWindow(surfaceID)
	if win == nil {
		return
	}
	evt := &WindowEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: WindowUnfocus,
	}
	win.focused = false
	win.dispatchEvent(evt)
}

// handleWaylandPointerButton processes a Wayland pointer button event.
func (a *App) handleWaylandPointerButton(surfaceID, button, state uint32, x, y float64) {
	win := a.lookupWindow(surfaceID)
	if win == nil {
		return
	}
	evt := translateWaylandPointerButtonEvent(button, state, x, y)
	win.dispatchEvent(evt)
}

// handleWaylandPointerMotion processes a Wayland pointer motion event.
func (a *App) handleWaylandPointerMotion(surfaceID uint32, x, y float64) {
	win := a.lookupWindow(surfaceID)
	if win == nil {
		return
	}
	evt := translateWaylandPointerMotionEvent(x, y)
	win.dispatchEvent(evt)
}

// handleWaylandPointerAxis processes a Wayland pointer axis (scroll) event.
func (a *App) handleWaylandPointerAxis(surfaceID, axis uint32, value, x, y float64) {
	win := a.lookupWindow(surfaceID)
	if win == nil {
		return
	}
	evt := translateWaylandPointerAxisEvent(axis, value, x, y)
	win.dispatchEvent(evt)
}

// handleWaylandPointerEnter processes pointer enter.
func (a *App) handleWaylandPointerEnter(surfaceID uint32, x, y float64) {
	// Could be used for hover effects in the future
}

// handleWaylandPointerLeave processes pointer leave.
func (a *App) handleWaylandPointerLeave(surfaceID uint32) {
	// Could be used for hover effects in the future
}

// connectX11 establishes an X11 connection.
func (a *App) connectX11(display string) error {
	conn, err := x11client.Connect(display)
	if err != nil {
		return fmt.Errorf("app: connect x11: %w", err)
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

	a.initX11Clipboard(wid)
	return nil
}

// initX11Clipboard creates the selection manager for the app window.  Non-fatal.
func (a *App) initX11Clipboard(wid x11client.XID) {
	selMgr, err := newX11SelectionManager(a.x11Conn, uint32(wid))
	if err == nil {
		a.x11SelectionMgr = selMgr
	} else if a.verbose {
		log.Printf("Warning: clipboard unavailable: %v", err)
	}
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

	// Initialize resource manager (Phase 9.5)
	// For now, texture atlas is nil (GPU atlas integration deferred)
	a.resources = newResourceManager(nil)
	if err := a.resources.initDefaultFont(); err != nil {
		return fmt.Errorf("failed to initialize default font: %w", err)
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

		// Process pending notifications from other goroutines
		a.processNotifications()

		// Process events
		if err := a.processEvents(); err != nil {
			return fmt.Errorf("app: run loop: process events: %w", err)
		}

		// Render frames for all windows
		if err := a.renderFrames(); err != nil {
			return fmt.Errorf("app: run loop: render frames: %w", err)
		}
	}
	return nil
}

// processNotifications executes all pending notification callbacks.
func (a *App) processNotifications() {
	for {
		select {
		case callback := <-a.notifyChan:
			if callback != nil {
				callback()
			}
		default:
			// No more pending notifications
			return
		}
	}
}

// processEvents processes pending events from the display server.
func (a *App) processEvents() error {
	switch a.displayServer {
	case DisplayServerWayland:
		return a.processWaylandEvents()
	case DisplayServerX11:
		return a.processX11Events()
	default:
		return ErrNoDisplay
	}
}

// processX11Events reads and processes pending X11 events.
func (a *App) processX11Events() error {
	if a.x11Conn == nil {
		return ErrNoDisplay
	}

	eventBuf, err := a.x11Conn.ReadEvent()
	if err != nil {
		return fmt.Errorf("read X11 event: %w", err)
	}
	if eventBuf == nil {
		return nil
	}

	return a.dispatchX11Event(eventBuf)
}

// dispatchX11Event parses and dispatches a single X11 event.
func (a *App) dispatchX11Event(eventBuf []byte) error {
	eventType := x11events.EventType(eventBuf[0] & 0x7F)

	a.mu.Lock()
	windows := a.windows
	a.mu.Unlock()

	for _, win := range windows {
		if err := win.handleX11Event(eventType, eventBuf); err != nil {
			return fmt.Errorf("app: dispatch x11 event: %w", err)
		}
	}

	return nil
}

// processWaylandEvents reads and processes pending Wayland events.
func (a *App) processWaylandEvents() error {
	if a.waylandConn == nil {
		return ErrNoDisplay
	}

	// Flush any pending outbound requests
	if err := a.waylandConn.Flush(); err != nil {
		return fmt.Errorf("flush wayland requests: %w", err)
	}

	// Read a single event message from the compositor
	msg, err := a.waylandConn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read wayland event: %w", err)
	}
	if msg == nil {
		return nil // No event available
	}

	// Dispatch the event to the appropriate object handler
	return a.dispatchWaylandEvent(msg)
}

// dispatchWaylandEvent routes a Wayland event to the appropriate window handler.
func (a *App) dispatchWaylandEvent(msg *wlwire.Message) error {
	// Dispatch through the connection's object registry. Object-level handlers
	// (Keyboard, Pointer, etc.) fire surfaceID-keyed callbacks which are routed
	// to the owning Window via lookupWindow — multi-window input routing is
	// fully supported through the surfaceToWindow map.
	if err := a.waylandConn.DispatchMessage(msg); err != nil {
		return fmt.Errorf("dispatch wayland event: %w", err)
	}

	return nil
}

// renderFrames renders frames for all dirty windows.
func (a *App) renderFrames() error {
	a.mu.Lock()
	windows := a.windows
	a.mu.Unlock()

	for _, win := range windows {
		if err := win.RenderFrame(); err != nil {
			return fmt.Errorf("render frame: %w", err)
		}
	}

	return nil
}

// cleanup releases all resources.
func (a *App) cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Clean up resource manager (Phase 9.5)
	if a.resources != nil {
		a.resources.cleanup()
		a.resources = nil
	}

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
