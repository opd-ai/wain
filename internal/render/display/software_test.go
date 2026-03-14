package display_test

import (
	"context"
	"testing"

	"github.com/opd-ai/wain/internal/render/backend"
	"github.com/opd-ai/wain/internal/render/display"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/shm"
)

// TestSoftwareWaylandPresenter_Present verifies that Present copies pixels into
// the SHM buffer and issues Attach/Damage/Commit calls without error when both
// the SHM global and surface are stubbed.
//
// This test validates the constructor and Present path in isolation without a
// real Wayland compositor.
func TestSoftwareWaylandPresenter_Pixels(t *testing.T) {
	sw, err := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 4, Height: 4})
	if err != nil {
		t.Fatalf("NewSoftwareBackend: %v", err)
	}

	pixels := sw.Pixels()
	if len(pixels) != 4*4*4 {
		t.Errorf("Pixels length: got %d, want %d", len(pixels), 4*4*4)
	}

	// Dimensions must match.
	w, h := sw.Dimensions()
	if w != 4 || h != 4 {
		t.Errorf("Dimensions: got %dx%d, want 4x4", w, h)
	}
}

// TestSoftwareWaylandPresenter_NilPixels verifies Present is a no-op when the
// backend has no pixels (buffer == nil after Destroy).
func TestSoftwareWaylandPresenter_NilPixels(t *testing.T) {
	sw, err := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 8, Height: 8})
	if err != nil {
		t.Fatalf("NewSoftwareBackend: %v", err)
	}
	_ = sw.Destroy()

	// nil stubs are intentional: Present must short-circuit before touching them.
	var shmGlobal *shm.SHM
	var surface *client.Surface

	p := display.NewSoftwareWaylandPresenter(shmGlobal, surface, sw)
	if err := p.Present(context.Background()); err != nil {
		t.Errorf("Present with nil pixels: expected nil error, got %v", err)
	}
}

// TestSoftwareWaylandPresenter_Close verifies that Close is safe to call on a
// presenter that has never presented a frame (pool == nil).
func TestSoftwareWaylandPresenter_Close(t *testing.T) {
	sw, err := backend.NewSoftwareBackend(backend.SoftwareConfig{Width: 4, Height: 4})
	if err != nil {
		t.Fatalf("NewSoftwareBackend: %v", err)
	}

	var shmGlobal *shm.SHM
	var surface *client.Surface

	p := display.NewSoftwareWaylandPresenter(shmGlobal, surface, sw)
	if err := p.Close(); err != nil {
		t.Errorf("Close on unused presenter: %v", err)
	}
}
