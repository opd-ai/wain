package display

import (
	"context"
	"fmt"

	"github.com/opd-ai/wain/internal/render/backend"
)

// presentRenderedFramebuffer handles the common acquire→export→populate→ensure→commit
// flow shared by WaylandPipeline and X11Pipeline. The ensureFunc and commitFunc are
// platform-specific steps; each must return a fully-wrapped error on failure.
func presentRenderedFramebuffer(
	ctx context.Context,
	pool *FramebufferPool,
	renderer backend.Renderer,
	releaseFunc func(*Framebuffer),
	ensureFunc func(*Framebuffer) error,
	commitFunc func(*Framebuffer) error,
) error {
	fb, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("display: acquire framebuffer: %w", err)
	}

	fd, err := renderer.Present()
	if err != nil {
		releaseFunc(fb)
		return fmt.Errorf("display: export dmabuf: %w", err)
	}

	if fb.Fd < 0 {
		fb.Fd = fd
		w, h := renderer.Dimensions()
		fb.Width = uint32(w)
		fb.Height = uint32(h)
		fb.Stride = uint32(w) * 4
	}

	if err := ensureFunc(fb); err != nil {
		releaseFunc(fb)
		return err
	}

	if err := commitFunc(fb); err != nil {
		releaseFunc(fb)
		return err
	}

	return pool.MarkDisplaying(fb)
}
