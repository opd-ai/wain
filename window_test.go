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
