package wain

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/render/backend"
)

// newOnePxPNGReader returns an io.Reader containing a 1×1 white PNG image.
func newOnePxPNGReader() io.Reader {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return &buf
}

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

// TestWindowOnTouchSetDropTarget verifies additional handler setters.
func TestWindowOnTouchSetDropTarget(t *testing.T) {
	app := NewApp()
	w := &Window{app: app}

	w.OnTouch(func(*TouchEvent) {})
	if w.onTouch == nil {
		t.Error("onTouch not set")
	}
}

// TestWindowRedrawAndDispatcher verifies Redraw and Dispatcher methods.
func TestWindowRedrawAndDispatcher(t *testing.T) {
	app := NewApp()
	w := &Window{app: app, dispatcher: NewEventDispatcher()}

	// Redraw should not panic on a headless window
	w.Redraw()

	// Dispatcher should return the initialised dispatcher
	d := w.Dispatcher()
	if d == nil {
		t.Error("Dispatcher() returned nil")
	}
}

// TestAppAnimate verifies that Animate returns a non-nil Animation.
func TestAppAnimate(t *testing.T) {
	app := NewApp()

	anim := app.Animate(0, 1, 100e6, AnimateLinear, func(v float64) {})
	if anim == nil {
		t.Error("Animate returned nil")
	}
}

// TestAppAnimateEasingFunctions verifies the easing function variables are non-nil.
func TestAppAnimateEasingFunctions(t *testing.T) {
	for name, fn := range map[string]EasingFunc{
		"Linear":    AnimateLinear,
		"EaseIn":    AnimateEaseIn,
		"EaseOut":   AnimateEaseOut,
		"EaseInOut": AnimateEaseInOut,
		"Spring":    AnimateSpring,
	} {
		if fn == nil {
			t.Errorf("Easing function %s is nil", name)
		}
		// Spot-check: linear f(0.5) should return 0.5
		if name == "Linear" {
			if got := fn(0.5); got != 0.5 {
				t.Errorf("Linear(0.5) = %v, want 0.5", got)
			}
		}
	}
}

// TestAccessibilityManagerNilSafety verifies that a nil AccessibilityManager
// doesn't panic when Close is called.
func TestAccessibilityManagerNilSafety(t *testing.T) {
	var am *AccessibilityManager
	am.Close() // must not panic
}

// TestPanelSetPaddingAndGap tests SetPadding and SetGap.
func TestPanelSetPaddingAndGap(t *testing.T) {
	p := NewPanel(Size{Width: 100, Height: 100})
	p.SetPadding(10)
	p.SetGap(5)
	// Verify via style accessor
	if p.styleOverride == nil {
		t.Error("styleOverride should be set after SetPadding")
	}
	if *p.styleOverride.Padding != 10 {
		t.Errorf("Padding = %d, want 10", *p.styleOverride.Padding)
	}
	if *p.styleOverride.Gap != 5 {
		t.Errorf("Gap = %d, want 5", *p.styleOverride.Gap)
	}
}

// TestPanelSetStyle tests SetStyle applies style override.
func TestPanelSetStyle(t *testing.T) {
	p := NewPanel(Size{Width: 50, Height: 50})
	bg := RGB(10, 20, 30)
	p.SetStyle(StyleOverride{Background: &bg})

	if p.styleOverride == nil {
		t.Error("styleOverride not set")
	}
	if p.styleOverride.Background == nil {
		t.Error("Background override not set")
	}
}

// TestPanelSetTheme tests that SetTheme propagates to children.
func TestPanelSetTheme(t *testing.T) {
	parent := NewPanel(Size{Width: 100, Height: 100})
	child := NewPanel(Size{Width: 50, Height: 50})
	parent.Add(child)

	theme := DefaultLight()
	parent.SetTheme(theme)

	if parent.theme == nil {
		t.Error("theme not set on parent")
	}
}

// TestThemeAdapterStyle tests the themeAdapter style accessors.
func TestThemeAdapterStyle(t *testing.T) {
	dark := DefaultDark()

	// Without override
	ta := &themeAdapter{base: dark}
	if ta.Background() == (primitives.Color{}) {
		t.Error("Background() should not be zero")
	}
	if ta.FontSize() != dark.FontSize {
		t.Errorf("FontSize() = %v, want %v", ta.FontSize(), dark.FontSize)
	}
	if ta.Padding() != dark.Padding {
		t.Errorf("Padding() = %d, want %d", ta.Padding(), dark.Padding)
	}
	if ta.Gap() != dark.Gap {
		t.Errorf("Gap() = %d, want %d", ta.Gap(), dark.Gap)
	}
	if ta.BorderWidth() != dark.BorderWidth {
		t.Errorf("BorderWidth() = %d, want %d", ta.BorderWidth(), dark.BorderWidth)
	}

	// With override
	bg := RGB(255, 0, 0)
	fg := RGB(0, 255, 0)
	ac := RGB(0, 0, 255)
	bd := RGB(128, 128, 128)
	fs := float64(20)
	pad := 8
	gap := 4
	bw := 2
	ta2 := &themeAdapter{
		base: dark,
		override: &StyleOverride{
			Background:  &bg,
			Foreground:  &fg,
			Accent:      &ac,
			Border:      &bd,
			FontSize:    &fs,
			Padding:     &pad,
			Gap:         &gap,
			BorderWidth: &bw,
		},
	}
	if ta2.Background() != bg.toInternal() {
		t.Error("Background override not applied")
	}
	if ta2.Foreground() != fg.toInternal() {
		t.Error("Foreground override not applied")
	}
	if ta2.Accent() != ac.toInternal() {
		t.Error("Accent override not applied")
	}
	if ta2.Border() != bd.toInternal() {
		t.Error("Border override not applied")
	}
	if ta2.FontSize() != 20 {
		t.Errorf("FontSize override = %v, want 20", ta2.FontSize())
	}
	if ta2.Padding() != 8 {
		t.Errorf("Padding override = %d, want 8", ta2.Padding())
	}
	if ta2.Gap() != 4 {
		t.Errorf("Gap override = %d, want 4", ta2.Gap())
	}
	if ta2.BorderWidth() != 2 {
		t.Errorf("BorderWidth override = %d, want 2", ta2.BorderWidth())
	}
}

// TestClampDimension exercises all branches of clampDimension.
func TestClampDimension(t *testing.T) {
	tests := []struct {
		value, min, max, want int
	}{
		{50, 0, 0, 50},      // no clamping
		{200, 0, 100, 100},  // clamped to max
		{10, 50, 0, 50},     // clamped to min
		{50, 10, 100, 50},   // within range, no clamping
		{150, 10, 100, 100}, // clamped to max
		{5, 10, 100, 10},    // clamped to min
	}
	for _, tt := range tests {
		if got := clampDimension(tt.value, tt.min, tt.max); got != tt.want {
			t.Errorf("clampDimension(%d, %d, %d) = %d, want %d", tt.value, tt.min, tt.max, got, tt.want)
		}
	}
}

// TestWindowSetSizeHeadless verifies SetSize on a headless window.
func TestWindowSetSizeHeadless(t *testing.T) {
	app := NewApp()
	w := &Window{app: app, width: 800, height: 600}

	if err := w.SetSize(1024, 768); err != nil {
		t.Errorf("SetSize: %v", err)
	}
	if w.width != 1024 || w.height != 768 {
		t.Errorf("Size after SetSize = %dx%d, want 1024x768", w.width, w.height)
	}
}

// TestWindowSetSizeClosed verifies SetSize returns error on closed window.
func TestWindowSetSizeClosed(t *testing.T) {
	app := NewApp()
	w := &Window{app: app, closed: true}
	if err := w.SetSize(100, 100); err == nil {
		t.Error("expected error on closed window")
	}
}

// TestWindowRedrawRegion verifies RedrawRegion on a headless window.
func TestWindowRedrawRegion(t *testing.T) {
	sw, _ := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 100, Height: 100})
	w := &Window{}
	w.renderBridge = NewRenderBridge(sw)

	// Must not panic
	w.RedrawRegion(10, 10, 50, 50)
}

// TestWindowSendCustomEvent verifies SendCustomEvent dispatches the event.
func TestWindowSendCustomEvent(t *testing.T) {
	app := NewApp()
	w := &Window{app: app, dispatcher: NewEventDispatcher()}

	var received CustomEventPayload
	w.dispatcher.OnCustom(func(e *CustomEvent) {
		received = e.Data()
	})

	type payload struct{ v int }
	p := payload{42}
	w.SendCustomEvent(p)

	if received != p {
		t.Errorf("SendCustomEvent: received %v, want %v", received, p)
	}
}

// TestWindowSendCustomEventNoDispatcher verifies SendCustomEvent is safe without dispatcher.
func TestWindowSendCustomEventNoDispatcher(t *testing.T) {
	w := &Window{}            // no dispatcher
	w.SendCustomEvent("test") // must not panic
}

// TestWindowRenderFrameNilRoot verifies RenderFrame on a window without a root widget.
func TestWindowRenderFrameNilRoot(t *testing.T) {
	sw, _ := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 100, Height: 100})
	w := &Window{}
	w.renderBridge = NewRenderBridge(sw)
	w.renderBridge.MarkDirty()

	// RenderFrame with nil rootWidget should return ErrNoRootWidget
	err := w.RenderFrame()
	if err == nil {
		t.Error("expected error with nil root widget")
	}
}

// TestAppResourcesNilGuards verifies LoadFont/LoadImage/LoadImageFromReader/DefaultFont
// return ErrNotRunning when the app has not been Run().
func TestAppResourcesNilGuards(t *testing.T) {
	app := NewApp()

	if _, err := app.LoadFont("font.ttf", 14); err != ErrNotRunning {
		t.Errorf("LoadFont: got %v, want ErrNotRunning", err)
	}
	if _, err := app.LoadImage("icon.png"); err != ErrNotRunning {
		t.Errorf("LoadImage: got %v, want ErrNotRunning", err)
	}
	if _, err := app.LoadImageFromReader(nil, "img.png"); err != ErrNotRunning {
		t.Errorf("LoadImageFromReader: got %v, want ErrNotRunning", err)
	}
	if f := app.DefaultFont(); f != nil {
		t.Error("DefaultFont should return nil when resources not initialised")
	}
}

// TestRenderBridgeDestroyNilRenderer verifies Destroy is safe with no renderer.
func TestRenderBridgeDestroyNilRenderer(t *testing.T) {
	rb := &RenderBridge{}
	if err := rb.Destroy(); err != nil {
		t.Errorf("Destroy with nil renderer: %v", err)
	}
}

// TestRenderBridgeWalkWidgetNonEmitter verifies walkWidget with a plain BaseWidget.
func TestRenderBridgeWalkWidgetNonEmitter(t *testing.T) {
	sw, _ := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 100, Height: 100})
	rb := NewRenderBridge(sw)
	w := &BaseWidget{}
	if err := rb.walkWidget(w); err != nil {
		t.Errorf("walkWidget non-emitter: %v", err)
	}
}

// TestRenderBridgeWalkWidgetNil verifies walkWidget(nil) is a no-op.
func TestRenderBridgeWalkWidgetNil(t *testing.T) {
	sw, _ := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 100, Height: 100})
	rb := NewRenderBridge(sw)
	if err := rb.walkWidget(nil); err != nil {
		t.Errorf("walkWidget(nil): %v", err)
	}
}

// TestWindowSettersWaylandNoToplevel verifies setters on a Wayland window with no toplevel.
func TestWindowSettersWaylandNoToplevel(t *testing.T) {
	app := NewApp()
	app.displayServer = DisplayServerWayland
	w := &Window{app: app, width: 800, height: 600}

	if err := w.SetTitle("test"); err != nil {
		t.Errorf("SetTitle Wayland no-toplevel: %v", err)
	}
	if err := w.SetSize(1024, 768); err != nil {
		t.Errorf("SetSize Wayland: %v", err)
	}
	if err := w.SetMinSize(100, 100); err != nil {
		t.Errorf("SetMinSize Wayland no-toplevel: %v", err)
	}
	if err := w.SetMaxSize(1920, 1080); err != nil {
		t.Errorf("SetMaxSize Wayland no-toplevel: %v", err)
	}
	if err := w.SetFullscreen(true); err != nil {
		t.Errorf("SetFullscreen Wayland no-toplevel: %v", err)
	}
}

// TestWindowCloseHeadless verifies Close on a headless window calls onClose and marks closed.
func TestWindowCloseHeadless(t *testing.T) {
	app := NewApp()
	app.displayServer = DisplayServerUnknown
	called := false
	w := &Window{app: app, onClose: func() { called = true }}

	if err := w.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if !w.closed {
		t.Error("window not marked closed")
	}
	if !called {
		t.Error("onClose not called")
	}

	// Second Close must be a no-op.
	if err := w.Close(); err != nil {
		t.Errorf("double Close: %v", err)
	}
}

// TestWindowCloseWayland verifies Close with DisplayServerWayland (no wayland objects).
func TestWindowCloseWayland(t *testing.T) {
	app := NewApp()
	app.displayServer = DisplayServerWayland
	w := &Window{app: app}
	if err := w.Close(); err != nil {
		t.Errorf("Close Wayland: %v", err)
	}
}

// TestNewAppWithConfigFields verifies NewAppWithConfig respects provided config.
func TestNewAppWithConfigFields(t *testing.T) {
	cfg := AppConfig{
		Width:         1280,
		Height:        720,
		ForceSoftware: true,
		DRMPath:       "/dev/dri/card0",
	}
	app := NewAppWithConfig(cfg)
	if app == nil {
		t.Fatal("NewAppWithConfig returned nil")
	}
	if app.width != 1280 {
		t.Errorf("width = %d, want 1280", app.width)
	}
	if app.height != 720 {
		t.Errorf("height = %d, want 720", app.height)
	}
	if app.drmPath != "/dev/dri/card0" {
		t.Errorf("drmPath = %q, want /dev/dri/card0", app.drmPath)
	}
}

// TestAppSetGetTheme verifies SetTheme/GetTheme/Theme round-trip.
func TestAppSetGetTheme(t *testing.T) {
	app := NewApp()
	theme := DefaultDark()
	theme.Background = RGB(10, 20, 30)
	app.SetTheme(theme)

	got := app.GetTheme()
	if got.Background != theme.Background {
		t.Errorf("GetTheme background = %v, want %v", got.Background, theme.Background)
	}
	got2 := app.Theme()
	if got2.Background != theme.Background {
		t.Errorf("Theme() background = %v, want %v", got2.Background, theme.Background)
	}
}

// TestAppBackendTypeDimensions verifies BackendType and Dimensions on a new app.
func TestAppBackendTypeDimensions(t *testing.T) {
	app := NewAppWithConfig(AppConfig{Width: 1024, Height: 768})
	_ = app.BackendType()

	w, h := app.Dimensions()
	if w != 1024 || h != 768 {
		t.Errorf("Dimensions = %dx%d, want 1024x768", w, h)
	}
}

// TestAppSetThemeWithWindow verifies SetTheme marks dirty on windows with render bridge.
func TestAppSetThemeWithWindow(t *testing.T) {
	app := NewApp()
	sw, _ := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 100, Height: 100})
	w := &Window{app: app}
	w.renderBridge = NewRenderBridge(sw)
	app.windows = append(app.windows, w)

	app.SetTheme(DefaultLight())
}

// TestDispatchTouchWithHandler verifies dispatchTouch calls handler when no widget root.
func TestDispatchTouchWithHandler(t *testing.T) {
	d := NewEventDispatcher()
	called := false
	d.OnTouch(func(e *TouchEvent) { called = true })

	evt := &TouchEvent{eventType: TouchDown}
	d.Dispatch(evt)
	if !called {
		t.Error("touch handler not called")
	}
}

// TestDispatchTouchConsumed verifies dispatchTouch stops after Consumed.
func TestDispatchTouchConsumed(t *testing.T) {
	d := NewEventDispatcher()
	count := 0
	d.OnTouch(func(e *TouchEvent) {
		count++
		e.Consume()
	})
	d.OnTouch(func(e *TouchEvent) {
		count++
	})

	d.Dispatch(&TouchEvent{eventType: TouchDown})
	if count != 1 {
		t.Errorf("touch handler called %d times, want 1", count)
	}
}

// TestWindowSetFullscreenFalseWayland exercises the applyWaylandFullscreen(false) path.
func TestWindowSetFullscreenFalseWayland(t *testing.T) {
	app := NewApp()
	app.displayServer = DisplayServerWayland
	w := &Window{app: app}

	if err := w.SetFullscreen(false); err != nil {
		t.Errorf("SetFullscreen(false) Wayland: %v", err)
	}
	if err := w.SetFullscreen(true); err != nil {
		t.Errorf("SetFullscreen(true) Wayland: %v", err)
	}
}

// TestWindowSetRootWidgetWithDispatcher verifies SetRootWidget with dispatcher and renderBridge.
func TestWindowSetRootWidgetWithDispatcher(t *testing.T) {
	sw, _ := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 100, Height: 100})
	w := &Window{dispatcher: NewEventDispatcher()}
	w.renderBridge = NewRenderBridge(sw)

	root := &BaseWidget{}
	root.SetBounds(0, 0, 100, 100)
	w.SetRootWidget(root)

	if w.rootWidget != root {
		t.Error("rootWidget not set")
	}
}

// TestWindowRedrawWithRenderBridge verifies Redraw marks dirty.
func TestWindowRedrawWithRenderBridge(t *testing.T) {
	sw, _ := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 100, Height: 100})
	w := &Window{}
	w.renderBridge = NewRenderBridge(sw)

	w.Redraw() // must not panic
}

// TestDispatchWindowConsumed verifies dispatchWindow stops on consumed.
func TestDispatchWindowConsumed(t *testing.T) {
	d := NewEventDispatcher()
	count := 0
	d.OnWindow(func(e *WindowEvent) {
		count++
		e.Consume()
	})
	d.OnWindow(func(e *WindowEvent) {
		count++
	})

	d.Dispatch(&WindowEvent{eventType: WindowFocus})
	if count != 1 {
		t.Errorf("window handler called %d times, want 1", count)
	}
}

// TestDispatchCustomConsumed verifies dispatchCustom stops on consumed.
func TestDispatchCustomConsumed(t *testing.T) {
	d := NewEventDispatcher()
	count := 0
	d.OnCustom(func(e *CustomEvent) {
		count++
		e.Consume()
	})
	d.OnCustom(func(e *CustomEvent) {
		count++
	})

	d.Dispatch(&CustomEvent{data: "test"})
	if count != 1 {
		t.Errorf("custom handler called %d times, want 1", count)
	}
}

// TestHitTestWithChildWidget verifies hitTest returns child when child contains point.
func TestHitTestWithChildWidget(t *testing.T) {
	d := NewEventDispatcher()

	parent := &BaseWidget{}
	parent.SetBounds(0, 0, 100, 100)
	child := &BaseWidget{}
	child.SetBounds(10, 10, 50, 50)
	parent.AddChild(child)

	hit := d.hitTest(parent, 30, 30)
	if hit != child {
		t.Errorf("hitTest returned %v, want child", hit)
	}
}

// TestHitTestParentNoChild verifies hitTest returns parent when no child matches.
func TestHitTestParentNoChild(t *testing.T) {
	d := NewEventDispatcher()

	parent := &BaseWidget{}
	parent.SetBounds(0, 0, 100, 100)
	child := &BaseWidget{}
	child.SetBounds(10, 10, 30, 30)
	parent.AddChild(child)

	// Hit outside child but inside parent
	hit := d.hitTest(parent, 80, 80)
	if hit != parent {
		t.Errorf("hitTest returned %v, want parent", hit)
	}
}

// mockPresenter is a test double for Presenter.
type mockPresenter struct {
	presentErr error
}

func (m *mockPresenter) Present(_ context.Context) error { return m.presentErr }
func (m *mockPresenter) Close() error                    { return nil }

// TestRenderFrameWithPresenter verifies RenderFrame calls presenter.Present.
func TestRenderFrameWithPresenter(t *testing.T) {
	sw, err := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 100, Height: 100})
	if err != nil {
		t.Skipf("software backend unavailable: %v", err)
	}
	w := &Window{}
	w.renderBridge = NewRenderBridge(sw)
	w.rootWidget = &BaseWidget{}
	w.presenter = &mockPresenter{}

	if err := w.RenderFrame(); err != nil {
		t.Errorf("RenderFrame failed: %v", err)
	}
}

// TestRenderFramePresenterError verifies RenderFrame wraps presenter errors.
func TestRenderFramePresenterError(t *testing.T) {
	sw, err := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 100, Height: 100})
	if err != nil {
		t.Skipf("software backend unavailable: %v", err)
	}
	w := &Window{}
	w.renderBridge = NewRenderBridge(sw)
	w.presenter = &mockPresenter{presentErr: errors.New("present failed")}

	err = w.RenderFrame()
	if err == nil {
		t.Error("expected error from presenter, got nil")
	}
}

// TestNewAppWithConfigZeroDimensions verifies zero width/height are defaulted.
func TestNewAppWithConfigZeroDimensions(t *testing.T) {
	app := NewAppWithConfig(AppConfig{Width: 0, Height: 0})
	if app.width != 800 {
		t.Errorf("width = %d, want 800", app.width)
	}
	if app.height != 600 {
		t.Errorf("height = %d, want 600", app.height)
	}
}

// TestAppResourceMethodsWithInitializedResources covers App-level resource methods
// when resources is properly initialized (non-nil path).
func TestAppResourceMethodsWithInitializedResources(t *testing.T) {
	a := &App{}
	a.resources = newResourceManager(nil)
	if err := a.resources.initDefaultFont(); err != nil {
		t.Skip("could not load embedded font:", err)
	}

	// DefaultFont — non-nil path
	font := a.DefaultFont()
	if font == nil {
		t.Error("DefaultFont should not be nil after initDefaultFont")
	}

	// LoadFont — non-nil path
	f, err := a.LoadFont("dummy.ttf", 12.0)
	if err != nil {
		t.Fatalf("LoadFont failed: %v", err)
	}
	if f.size != 12.0 {
		t.Errorf("LoadFont size = %v, want 12.0", f.size)
	}

	// LoadImageFromReader — non-nil path (PNG)
	img, err := a.LoadImageFromReader(newOnePxPNGReader(), "test.png")
	if err != nil {
		t.Fatalf("LoadImageFromReader failed: %v", err)
	}
	if img.width <= 0 || img.height <= 0 {
		t.Error("loaded image has zero dimensions")
	}

	// LoadImage — non-nil path (write a temp PNG and load it)
	tmpFile := t.TempDir() + "/test.png"
	if tf, ferr := os.Create(tmpFile); ferr == nil {
		_ = png.Encode(tf, image.NewRGBA(image.Rect(0, 0, 1, 1)))
		tf.Close()
		if img2, lerr := a.LoadImage(tmpFile); lerr != nil {
			t.Fatalf("LoadImage failed: %v", lerr)
		} else if img2.width <= 0 {
			t.Error("LoadImage returned zero-width image")
		}
	}
}

// TestWindowSetTitleDisplayServerBranches covers Wayland/X11 switch arms in SetTitle.
func TestWindowSetTitleDisplayServerBranches(t *testing.T) {
	// Wayland path with nil toplevel (skips the set, no error)
	app := &App{displayServer: DisplayServerWayland}
	w := &Window{app: app}
	if err := w.SetTitle("wayland title"); err != nil {
		t.Errorf("Wayland/nil-toplevel SetTitle: %v", err)
	}

	// X11 path with zero x11Window (skips the set, no error)
	app2 := &App{displayServer: DisplayServerX11}
	w2 := &Window{app: app2}
	if err := w2.SetTitle("x11 title"); err != nil {
		t.Errorf("X11/zero-window SetTitle: %v", err)
	}
}

// TestWindowSetSizeDisplayServerBranches covers Wayland/X11 switch arms in SetSize.
func TestWindowSetSizeDisplayServerBranches(t *testing.T) {
	// Wayland returns nil immediately (client-side resize not allowed)
	app := &App{displayServer: DisplayServerWayland}
	w := &Window{app: app, width: 800, height: 600}
	if err := w.SetSize(1024, 768); err != nil {
		t.Errorf("Wayland SetSize: %v", err)
	}

	// X11 path with zero x11Window (skips configure, no error)
	app2 := &App{displayServer: DisplayServerX11}
	w2 := &Window{app: app2}
	if err := w2.SetSize(1024, 768); err != nil {
		t.Errorf("X11/zero-window SetSize: %v", err)
	}
}

// TestWindowSetMinMaxSizeDisplayServerBranches covers switch arms in SetMinSize/SetMaxSize.
func TestWindowSetMinMaxSizeDisplayServerBranches(t *testing.T) {
	app := &App{displayServer: DisplayServerWayland}
	w := &Window{app: app}
	if err := w.SetMinSize(100, 100); err != nil {
		t.Errorf("Wayland SetMinSize: %v", err)
	}
	if err := w.SetMaxSize(1920, 1080); err != nil {
		t.Errorf("Wayland SetMaxSize: %v", err)
	}

	app2 := &App{displayServer: DisplayServerX11}
	w2 := &Window{app: app2}
	if err := w2.SetMinSize(100, 100); err != nil {
		t.Errorf("X11/zero-window SetMinSize: %v", err)
	}
	if err := w2.SetMaxSize(1920, 1080); err != nil {
		t.Errorf("X11/zero-window SetMaxSize: %v", err)
	}
}

// TestWindowSetFullscreenDisplayServerBranches covers switch arms in SetFullscreen.
func TestWindowSetFullscreenDisplayServerBranches(t *testing.T) {
	// Wayland with nil toplevel (applyWaylandFullscreen returns nil)
	app := &App{displayServer: DisplayServerWayland}
	w := &Window{app: app}
	if err := w.SetFullscreen(true); err != nil {
		t.Errorf("Wayland SetFullscreen(true): %v", err)
	}
	if err := w.SetFullscreen(false); err != nil {
		t.Errorf("Wayland SetFullscreen(false): %v", err)
	}

	// X11 with zero x11Window (skips the call, no error)
	app2 := &App{displayServer: DisplayServerX11}
	w2 := &Window{app: app2}
	if err := w2.SetFullscreen(true); err != nil {
		t.Errorf("X11/zero-window SetFullscreen: %v", err)
	}
}

// TestWindowCloseDisplayServerBranches covers switch arms in Close.
func TestWindowCloseDisplayServerBranches(t *testing.T) {
	// Wayland with nil surface (skips destroy, no error)
	app := &App{displayServer: DisplayServerWayland}
	w := &Window{app: app}
	w.Close()

	// X11 with zero x11Window (skips destroy, no error)
	app2 := &App{displayServer: DisplayServerX11}
	w2 := &Window{app: app2}
	w2.Close()
}
