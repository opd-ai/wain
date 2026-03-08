package backend

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/core"
	"github.com/opd-ai/wain/internal/raster/displaylist"
)

func TestNewSoftwareBackend(t *testing.T) {
	cfg := SoftwareConfig{
		Width:  640,
		Height: 480,
	}

	backend, err := NewSoftwareBackend(cfg)
	if err != nil {
		t.Fatalf("NewSoftwareBackend failed: %v", err)
	}
	defer backend.Destroy()

	w, h := backend.Dimensions()
	if w != cfg.Width || h != cfg.Height {
		t.Errorf("backend.Dimensions() = (%d, %d), want (%d, %d)",
			w, h, cfg.Width, cfg.Height)
	}
}

func TestSoftwareBackendInvalidDimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"zero width", 0, 480},
		{"zero height", 640, 0},
		{"negative width", -1, 480},
		{"negative height", 640, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SoftwareConfig{
				Width:  tt.width,
				Height: tt.height,
			}

			_, err := NewSoftwareBackend(cfg)
			if err == nil {
				t.Errorf("NewSoftwareBackend with dimensions %dx%d succeeded, want error",
					tt.width, tt.height)
			}
		})
	}
}

func TestSoftwareBackendRender(t *testing.T) {
	cfg := SoftwareConfig{
		Width:  320,
		Height: 240,
	}

	backend, err := NewSoftwareBackend(cfg)
	if err != nil {
		t.Fatalf("NewSoftwareBackend failed: %v", err)
	}
	defer backend.Destroy()

	// Create a simple display list
	dl := displaylist.New()
	dl.AddFillRect(10, 10, 100, 100, core.Color{R: 255, G: 0, B: 0, A: 255}) // Red rectangle

	// Render it
	err = backend.Render(dl)
	if err != nil {
		t.Errorf("backend.Render failed: %v", err)
	}

	// Verify buffer was updated (simple sanity check)
	buf := backend.Buffer()
	if buf == nil {
		t.Error("backend.Buffer() returned nil after render")
	}
}

func TestSoftwareBackendRenderWithDamage(t *testing.T) {
	cfg := SoftwareConfig{
		Width:  320,
		Height: 240,
	}

	backend, err := NewSoftwareBackend(cfg)
	if err != nil {
		t.Fatalf("NewSoftwareBackend failed: %v", err)
	}
	defer backend.Destroy()

	// Create display list with multiple rects
	dl := displaylist.New()
	dl.AddFillRect(10, 10, 50, 50, core.Color{R: 255, G: 0, B: 0, A: 255})   // Red - top left
	dl.AddFillRect(200, 150, 50, 50, core.Color{R: 0, G: 255, B: 0, A: 255}) // Green - bottom right

	// Render with damage only covering top-left rect
	damage := []displaylist.Rect{
		{X: 0, Y: 0, Width: 100, Height: 100},
	}

	err = backend.RenderWithDamage(dl, damage)
	if err != nil {
		t.Errorf("backend.RenderWithDamage failed: %v", err)
	}
}

func TestSoftwareBackendRenderNilDisplayList(t *testing.T) {
	cfg := SoftwareConfig{
		Width:  320,
		Height: 240,
	}

	backend, err := NewSoftwareBackend(cfg)
	if err != nil {
		t.Fatalf("NewSoftwareBackend failed: %v", err)
	}
	defer backend.Destroy()

	err = backend.Render(nil)
	if err != ErrNilDisplayList {
		t.Errorf("backend.Render(nil) = %v, want ErrNilDisplayList", err)
	}
}

func TestSoftwareBackendPresent(t *testing.T) {
	cfg := SoftwareConfig{
		Width:  320,
		Height: 240,
	}

	backend, err := NewSoftwareBackend(cfg)
	if err != nil {
		t.Fatalf("NewSoftwareBackend failed: %v", err)
	}
	defer backend.Destroy()

	fd, err := backend.Present()
	if err != ErrSoftwareNoDmabuf {
		t.Errorf("backend.Present() error = %v, want ErrSoftwareNoDmabuf", err)
	}

	if fd != -1 {
		t.Errorf("backend.Present() fd = %d, want -1", fd)
	}
}

func TestSoftwareBackendDestroy(t *testing.T) {
	cfg := SoftwareConfig{
		Width:  320,
		Height: 240,
	}

	backend, err := NewSoftwareBackend(cfg)
	if err != nil {
		t.Fatalf("NewSoftwareBackend failed: %v", err)
	}

	err = backend.Destroy()
	if err != nil {
		t.Errorf("backend.Destroy() failed: %v", err)
	}

	// Verify resources are cleared
	if backend.buffer != nil {
		t.Error("backend.buffer not nil after Destroy()")
	}

	if backend.consumer != nil {
		t.Error("backend.consumer not nil after Destroy()")
	}
}
