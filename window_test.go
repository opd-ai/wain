package wain

import (
	"testing"
)

func TestWindowConfig(t *testing.T) {
	tests := []struct {
		name   string
		config WindowConfig
		want   WindowConfig
	}{
		{
			name: "default values",
			config: WindowConfig{
				Title: "Test Window",
			},
			want: WindowConfig{
				Title:       "Test Window",
				Width:       0,
				Height:      0,
				Decorations: false,
			},
		},
		{
			name: "custom dimensions",
			config: WindowConfig{
				Title:  "Custom",
				Width:  1024,
				Height: 768,
			},
			want: WindowConfig{
				Title:       "Custom",
				Width:       1024,
				Height:      768,
				Decorations: false,
			},
		},
		{
			name: "size constraints",
			config: WindowConfig{
				Title:     "Constrained",
				Width:     800,
				Height:    600,
				MinWidth:  400,
				MinHeight: 300,
				MaxWidth:  1920,
				MaxHeight: 1080,
			},
			want: WindowConfig{
				Title:       "Constrained",
				Width:       800,
				Height:      600,
				MinWidth:    400,
				MinHeight:   300,
				MaxWidth:    1920,
				MaxHeight:   1080,
				Decorations: false,
			},
		},
		{
			name: "fullscreen enabled",
			config: WindowConfig{
				Title:      "Fullscreen",
				Width:      1920,
				Height:     1080,
				Fullscreen: true,
			},
			want: WindowConfig{
				Title:       "Fullscreen",
				Width:       1920,
				Height:      1080,
				Fullscreen:  true,
				Decorations: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config
			if cfg.Title != tt.want.Title {
				t.Errorf("Title = %v, want %v", cfg.Title, tt.want.Title)
			}
			if cfg.Width != tt.want.Width {
				t.Errorf("Width = %v, want %v", cfg.Width, tt.want.Width)
			}
			if cfg.Height != tt.want.Height {
				t.Errorf("Height = %v, want %v", cfg.Height, tt.want.Height)
			}
			if cfg.Fullscreen != tt.want.Fullscreen {
				t.Errorf("Fullscreen = %v, want %v", cfg.Fullscreen, tt.want.Fullscreen)
			}
		})
	}
}

func TestNewWindow_NotRunning(t *testing.T) {
	app := NewApp()
	_, err := app.NewWindow(WindowConfig{
		Title:  "Test",
		Width:  800,
		Height: 600,
	})

	if err != ErrNotRunning {
		t.Errorf("NewWindow() error = %v, want %v", err, ErrNotRunning)
	}
}

func TestNewWindow_InvalidConfig(t *testing.T) {
	app := NewApp()
	app.running = true

	tests := []struct {
		name   string
		config WindowConfig
	}{
		{
			name: "negative min width",
			config: WindowConfig{
				Title:    "Test",
				Width:    800,
				Height:   600,
				MinWidth: -100,
			},
		},
		{
			name: "negative min height",
			config: WindowConfig{
				Title:     "Test",
				Width:     800,
				Height:    600,
				MinHeight: -100,
			},
		},
		{
			name: "negative max width",
			config: WindowConfig{
				Title:    "Test",
				Width:    800,
				Height:   600,
				MaxWidth: -100,
			},
		},
		{
			name: "negative max height",
			config: WindowConfig{
				Title:     "Test",
				Width:     800,
				Height:    600,
				MaxHeight: -100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := app.NewWindow(tt.config)
			if err != ErrInvalidWindowConfig {
				t.Errorf("NewWindow() error = %v, want %v", err, ErrInvalidWindowConfig)
			}
		})
	}
}

func TestWindow_GettersBeforeInit(t *testing.T) {
	win := &Window{
		title:  "Test Window",
		width:  800,
		height: 600,
		scale:  2.0,
	}

	if got := win.Title(); got != "Test Window" {
		t.Errorf("Title() = %v, want %v", got, "Test Window")
	}

	w, h := win.Size()
	if w != 800 || h != 600 {
		t.Errorf("Size() = %v, %v, want %v, %v", w, h, 800, 600)
	}

	if got := win.Scale(); got != 2.0 {
		t.Errorf("Scale() = %v, want %v", got, 2.0)
	}

	if win.IsClosed() {
		t.Error("IsClosed() = true, want false")
	}

	if win.IsFocused() {
		t.Error("IsFocused() = true, want false")
	}

	if win.IsFullscreen() {
		t.Error("IsFullscreen() = true, want false")
	}
}

// TestBuildWMSizeHints exercises the WM_SIZE_HINTS serialisation helper.
func TestBuildWMSizeHints(t *testing.T) {
	tests := []struct {
		name       string
		minW, minH int
		maxW, maxH int
		wantFlags  uint32
	}{
		{
			name:      "no constraints",
			wantFlags: 0,
		},
		{
			name:      "min size only",
			minW:      200, minH: 100,
			wantFlags: 1 << 4, // PMinSize
		},
		{
			name:      "max size only",
			maxW:      1920, maxH: 1080,
			wantFlags: 1 << 5, // PMaxSize
		},
		{
			name:      "both constraints",
			minW:      200, minH: 100,
			maxW:      1920, maxH: 1080,
			wantFlags: (1 << 4) | (1 << 5), // PMinSize | PMaxSize
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			win := &Window{
				minWidth: tt.minW, minHeight: tt.minH,
				maxWidth: tt.maxW, maxHeight: tt.maxH,
			}

			hints := win.buildWMSizeHints()

			if len(hints) != 72 {
				t.Fatalf("hints length = %d, want 72", len(hints))
			}

			flags := uint32(hints[0]) | uint32(hints[1])<<8 | uint32(hints[2])<<16 | uint32(hints[3])<<24
			if flags != tt.wantFlags {
				t.Errorf("flags = %#x, want %#x", flags, tt.wantFlags)
			}

			if tt.minW > 0 || tt.minH > 0 {
				gotMinW := uint32(hints[20]) | uint32(hints[21])<<8 | uint32(hints[22])<<16 | uint32(hints[23])<<24
				gotMinH := uint32(hints[24]) | uint32(hints[25])<<8 | uint32(hints[26])<<16 | uint32(hints[27])<<24
				if int(gotMinW) != tt.minW {
					t.Errorf("min_width = %d, want %d", gotMinW, tt.minW)
				}
				if int(gotMinH) != tt.minH {
					t.Errorf("min_height = %d, want %d", gotMinH, tt.minH)
				}
			}

			if tt.maxW > 0 || tt.maxH > 0 {
				gotMaxW := uint32(hints[28]) | uint32(hints[29])<<8 | uint32(hints[30])<<16 | uint32(hints[31])<<24
				gotMaxH := uint32(hints[32]) | uint32(hints[33])<<8 | uint32(hints[34])<<16 | uint32(hints[35])<<24
				if int(gotMaxW) != tt.maxW {
					t.Errorf("max_width = %d, want %d", gotMaxW, tt.maxW)
				}
				if int(gotMaxH) != tt.maxH {
					t.Errorf("max_height = %d, want %d", gotMaxH, tt.maxH)
				}
			}
		})
	}
}

// TestWindow_X11SettersClosedWindow verifies that X11 window-management
// methods return an error when the window is already closed.
func TestWindow_X11SettersClosedWindow(t *testing.T) {
	win := &Window{closed: true}

	if err := win.SetTitle("hello"); err == nil {
		t.Error("SetTitle on closed window should return error")
	}
	if err := win.SetMinSize(100, 100); err == nil {
		t.Error("SetMinSize on closed window should return error")
	}
	if err := win.SetMaxSize(800, 600); err == nil {
		t.Error("SetMaxSize on closed window should return error")
	}
	if err := win.SetFullscreen(true); err == nil {
		t.Error("SetFullscreen on closed window should return error")
	}
}

func TestWindow_EventHandlers(t *testing.T) {
	win := &Window{}

	resizeCalled := false
	win.OnResize(func(w, h int) {
		resizeCalled = true
	})

	closeCalled := false
	win.OnClose(func() {
		closeCalled = true
	})

	focusCalled := false
	win.OnFocus(func(focused bool) {
		focusCalled = true
	})

	scaleChangeCalled := false
	win.OnScaleChange(func(scale float64) {
		scaleChangeCalled = true
	})

	if win.onResize == nil {
		t.Error("OnResize callback not set")
	}
	if win.onClose == nil {
		t.Error("OnClose callback not set")
	}
	if win.onFocus == nil {
		t.Error("OnFocus callback not set")
	}
	if win.onScaleChange == nil {
		t.Error("OnScaleChange callback not set")
	}

	// Verify callbacks can be invoked
	if win.onResize != nil {
		win.onResize(100, 100)
	}
	if !resizeCalled {
		t.Error("Resize callback not called")
	}

	if win.onClose != nil {
		win.onClose()
	}
	if !closeCalled {
		t.Error("Close callback not called")
	}

	if win.onFocus != nil {
		win.onFocus(true)
	}
	if !focusCalled {
		t.Error("Focus callback not called")
	}

	if win.onScaleChange != nil {
		win.onScaleChange(2.0)
	}
	if !scaleChangeCalled {
		t.Error("ScaleChange callback not called")
	}
}
