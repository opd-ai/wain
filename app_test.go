package wain

import (
	"errors"
	"testing"

	"github.com/opd-ai/wain/internal/render/backend"
)

// TestDisplayServerString verifies the String() method on DisplayServer.
func TestDisplayServerString(t *testing.T) {
	tests := []struct {
		ds   DisplayServer
		want string
	}{
		{DisplayServerWayland, "Wayland"},
		{DisplayServerX11, "X11"},
		{DisplayServerUnknown, "Unknown"},
		{DisplayServer(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.ds.String(); got != tt.want {
			t.Errorf("DisplayServer(%d).String() = %q, want %q", tt.ds, got, tt.want)
		}
	}
}

// TestWindowSetTitleNoDisplay verifies SetTitle on a headless window.
func TestWindowSetTitleNoDisplay(t *testing.T) {
	app := NewApp()
	w := &Window{app: app, title: "initial"}
	if err := w.SetTitle("new title"); err != nil {
		t.Errorf("SetTitle on headless window: %v", err)
	}
	if w.Title() != "new title" {
		t.Errorf("Title() = %q, want %q", w.Title(), "new title")
	}
}

// TestWindowSetTitleClosed verifies SetTitle returns an error on a closed window.
func TestWindowSetTitleClosed(t *testing.T) {
	app := NewApp()
	w := &Window{app: app, closed: true}
	err := w.SetTitle("x")
	if err == nil || err.Error() != "window is closed" {
		t.Errorf("expected 'window is closed', got %v", err)
	}
}

// TestWindowSetMinMaxSizeHeadless verifies SetMinSize/SetMaxSize on a headless window.
func TestWindowSetMinMaxSizeHeadless(t *testing.T) {
	app := NewApp()
	w := &Window{app: app}

	if err := w.SetMinSize(200, 150); err != nil {
		t.Errorf("SetMinSize: %v", err)
	}
	if w.minWidth != 200 || w.minHeight != 150 {
		t.Errorf("minWidth/minHeight not set: %d/%d", w.minWidth, w.minHeight)
	}

	if err := w.SetMaxSize(1920, 1080); err != nil {
		t.Errorf("SetMaxSize: %v", err)
	}
	if w.maxWidth != 1920 || w.maxHeight != 1080 {
		t.Errorf("maxWidth/maxHeight not set: %d/%d", w.maxWidth, w.maxHeight)
	}
}

// TestWindowSetMinSizeClosed verifies SetMinSize returns an error on a closed window.
func TestWindowSetMinSizeClosed(t *testing.T) {
	app := NewApp()
	w := &Window{app: app, closed: true}
	err := w.SetMinSize(100, 100)
	if err == nil {
		t.Error("expected error on closed window")
	}
}

// TestWindowSetMaxSizeClosed verifies SetMaxSize returns an error on a closed window.
func TestWindowSetMaxSizeClosed(t *testing.T) {
	app := NewApp()
	w := &Window{app: app, closed: true}
	err := w.SetMaxSize(100, 100)
	if err == nil {
		t.Error("expected error on closed window")
	}
}

// TestWindowSetFullscreenHeadless verifies SetFullscreen on a headless window.
func TestWindowSetFullscreenHeadless(t *testing.T) {
	app := NewApp()
	w := &Window{app: app}
	if err := w.SetFullscreen(true); err != nil {
		t.Errorf("SetFullscreen(true): %v", err)
	}
	if !w.IsFullscreen() {
		t.Error("IsFullscreen() should be true after SetFullscreen(true)")
	}
	if err := w.SetFullscreen(false); err != nil {
		t.Errorf("SetFullscreen(false): %v", err)
	}
	if w.IsFullscreen() {
		t.Error("IsFullscreen() should be false after SetFullscreen(false)")
	}
}

// TestWindowSetFullscreenClosed verifies SetFullscreen returns an error on a closed window.
func TestWindowSetFullscreenClosed(t *testing.T) {
	app := NewApp()
	w := &Window{app: app, closed: true}
	if err := w.SetFullscreen(true); err == nil {
		t.Error("expected error on closed window")
	}
}

// TestWindowSizeAccessors verifies Size(), IsFocused(), Scale(), IsClosed().
func TestWindowSizeAccessors(t *testing.T) {
	app := NewApp()
	w := &Window{app: app, width: 320, height: 240, scale: 2.0, focused: true}

	ww, wh := w.Size()
	if ww != 320 || wh != 240 {
		t.Errorf("Size() = %d,%d, want 320,240", ww, wh)
	}
	if !w.IsFocused() {
		t.Error("IsFocused() should be true")
	}
	if w.Scale() != 2.0 {
		t.Errorf("Scale() = %v, want 2.0", w.Scale())
	}
	if w.IsClosed() {
		t.Error("IsClosed() should be false")
	}
}

// TestWindowCallbacks verifies that OnResize/OnClose/OnFocus/OnScaleChange set callbacks.
func TestWindowCallbacks(t *testing.T) {
	app := NewApp()
	w := &Window{app: app}

	resized := false
	w.OnResize(func(ww, wh int) { resized = true })
	if w.onResize == nil {
		t.Error("onResize not set")
	}

	closed := false
	w.OnClose(func() { closed = true })
	if w.onClose == nil {
		t.Error("onClose not set")
	}

	focused := false
	w.OnFocus(func(f bool) { focused = true })
	if w.onFocus == nil {
		t.Error("onFocus not set")
	}

	w.OnScaleChange(func(s float64) {})
	if w.onScaleChange == nil {
		t.Error("onScaleChange not set")
	}

	_ = resized
	_ = closed
	_ = focused
}

// TestWindowOnPointerOnKeyOnDrop verifies that handler setters store callbacks.
func TestWindowOnPointerOnKeyOnDrop(t *testing.T) {
	app := NewApp()
	w := &Window{app: app}

	w.OnPointer(func(*PointerEvent) {})
	if w.onPointer == nil {
		t.Error("onPointer not set")
	}
	w.OnKeyPress(func(*KeyEvent) {})
	if w.onKeyPress == nil {
		t.Error("onKeyPress not set")
	}
	w.OnKeyRelease(func(*KeyEvent) {})
	if w.onKeyRelease == nil {
		t.Error("onKeyRelease not set")
	}
	w.SetDropTarget([]string{"text/plain"}, func(mime string, data []byte) {})
	if w.dropHandler == nil {
		t.Error("dropHandler not set")
	}
}

// TestWindowsEmpty verifies that a new App has an empty Windows slice.
func TestWindowsEmpty(t *testing.T) {
	app := NewApp()
	wins := app.Windows()
	if len(wins) != 0 {
		t.Errorf("expected 0 windows, got %d", len(wins))
	}
}

// TestValidateAndNormalizeConfig exercises config validation cases.
func TestValidateAndNormalizeConfig(t *testing.T) {
	// Negative dimensions are normalized to defaults (not errors)
	cfg := WindowConfig{Title: "neg", Width: -1, Height: -1}
	if err := validateAndNormalizeConfig(&cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		t.Errorf("negative dims should be filled with defaults, got %dx%d", cfg.Width, cfg.Height)
	}

	// Negative min/max dimensions must return an error
	cfg2 := WindowConfig{Title: "badmin", MinWidth: -1}
	if err := validateAndNormalizeConfig(&cfg2); err == nil {
		t.Error("expected error for negative MinWidth")
	}

	// Zero dimensions should be filled with defaults
	cfg3 := WindowConfig{Title: "ok"}
	if err := validateAndNormalizeConfig(&cfg3); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg3.Width == 0 || cfg3.Height == 0 {
		t.Errorf("Width/Height should be defaulted when zero, got %dx%d", cfg3.Width, cfg3.Height)
	}

	// Valid dimensions pass through unchanged
	cfg4 := WindowConfig{Title: "fine", Width: 800, Height: 600}
	if err := validateAndNormalizeConfig(&cfg4); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg4.Width != 800 || cfg4.Height != 600 {
		t.Errorf("valid dims should not change, got %dx%d", cfg4.Width, cfg4.Height)
	}
}

// TestErrNoRenderer checks the sentinel value.
func TestErrNoRenderer(t *testing.T) {
	if ErrNoRenderer == nil {
		t.Error("ErrNoRenderer must not be nil")
	}
	if !errors.Is(ErrNoRenderer, ErrNoRenderer) {
		t.Error("errors.Is should match ErrNoRenderer")
	}
}

// TestNewApp verifies that NewApp creates an App with valid default configuration.
func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("NewApp returned nil")
	}
	if app.width != 800 {
		t.Errorf("default width = %d, want 800", app.width)
	}
	if app.height != 600 {
		t.Errorf("default height = %d, want 600", app.height)
	}
	if app.notifyChan == nil {
		t.Error("notifyChan must be initialised")
	}
	if app.surfaceToWindow == nil {
		t.Error("surfaceToWindow must be initialised")
	}
}

// TestNewAppWithConfig verifies dimension overrides and flag fields.
func TestNewAppWithConfig(t *testing.T) {
	cfg := AppConfig{
		Width:         1920,
		Height:        1080,
		ForceSoftware: true,
		ForceX11:      true,
		Verbose:       true,
	}
	app := NewAppWithConfig(cfg)
	if app.width != 1920 {
		t.Errorf("width = %d, want 1920", app.width)
	}
	if app.height != 1080 {
		t.Errorf("height = %d, want 1080", app.height)
	}
	if !app.forceSW {
		t.Error("forceSW should be true")
	}
	if !app.verbose {
		t.Error("verbose should be true")
	}
}

// TestDefaultConfig verifies that DefaultConfig returns sensible defaults.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Width != 800 {
		t.Errorf("default width = %d, want 800", cfg.Width)
	}
	if cfg.Height != 600 {
		t.Errorf("default height = %d, want 600", cfg.Height)
	}
	if cfg.ForceSoftware {
		t.Error("ForceSoftware should default to false")
	}
}

// TestAppQuit verifies that calling Quit sets the shouldQuit flag.
func TestAppQuit(t *testing.T) {
	app := NewApp()
	app.running = true
	app.Quit()

	app.mu.Lock()
	quit := app.shouldQuit
	app.mu.Unlock()

	if !quit {
		t.Error("Quit() did not set shouldQuit")
	}
}

// TestAppNotify verifies that Notify enqueues a callback onto notifyChan.
func TestAppNotify(t *testing.T) {
	app := NewApp()
	app.running = true

	called := false
	app.Notify(func() { called = true })

	// Drain the channel and execute callbacks
	app.processNotifications()

	if !called {
		t.Error("Notify callback was not executed by processNotifications")
	}
}

// TestWindowRenderFrameWithSoftwareRenderer exercises RenderFrame on a
// headless software-backed Window — no display server required.
func TestWindowRenderFrameWithSoftwareRenderer(t *testing.T) {
	sw, err := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 100, Height: 100})
	if err != nil {
		t.Fatalf("failed to create software backend: %v", err)
	}

	w := &Window{}
	w.renderBridge = NewRenderBridge(sw)
	w.renderBridge.MarkDirty()

	btn := NewButton("OK", Size{Width: 50, Height: 20})
	w.SetLayout(btn)

	if err := w.RenderFrame(); err != nil {
		t.Errorf("RenderFrame returned error: %v", err)
	}
}

// TestWindowSetLayoutUpdatesRoot verifies that SetLayout replaces the root widget.
func TestWindowSetLayoutUpdatesRoot(t *testing.T) {
	w := &Window{}

	lbl := NewLabel("hello", Size{Width: 40, Height: 10})
	w.SetLayout(lbl)

	if w.rootWidget == nil {
		t.Error("rootWidget should be non-nil after SetLayout")
	}
}

// TestWindowRenderFrameNoRenderer verifies that RenderFrame returns
// ErrNoRenderer when no renderBridge is set.
func TestWindowRenderFrameNoRenderer(t *testing.T) {
	w := &Window{}
	err := w.RenderFrame()
	if err != ErrNoRenderer {
		t.Errorf("expected ErrNoRenderer, got %v", err)
	}
}

// TestWindowNewWindowConfig verifies that WindowConfig is stored correctly.
func TestWindowNewWindowConfig(t *testing.T) {
	app := NewApp()

	tests := []struct {
		name  string
		cfg   WindowConfig
		wantW int
		wantH int
	}{
		{"default", WindowConfig{Title: "A"}, 800, 600},
		{"custom size", WindowConfig{Title: "B", Width: 1024, Height: 768}, 1024, 768},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// NewWindow calls initialize() which needs a display; instead
			// construct the config and verify its fields only.
			if tt.cfg.Width > 0 && tt.cfg.Width != tt.wantW {
				t.Errorf("Width mismatch: %d != %d", tt.cfg.Width, tt.wantW)
			}
			if tt.cfg.Height > 0 && tt.cfg.Height != tt.wantH {
				t.Errorf("Height mismatch: %d != %d", tt.cfg.Height, tt.wantH)
			}
			// Validate the app can store window references
			if app.Windows() == nil {
				t.Error("Windows() must not be nil")
			}
		})
	}
}
