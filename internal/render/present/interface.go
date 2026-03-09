package present

import (
	"context"
	"errors"

	"github.com/opd-ai/wain/internal/raster/displaylist"
)

// ErrPresenterClosed is returned when operations are attempted on a closed presenter.
var ErrPresenterClosed = errors.New("present: presenter is closed")

// FramebufferHandle is an opaque handle to a platform-specific framebuffer.
// The concrete type depends on the backend implementation (e.g., *Ring.Buffer
// for the ring buffer pool). This type alias improves code clarity by making
// the framebuffer abstraction explicit.
type FramebufferHandle interface{}

// PlatformPresenter handles platform-specific rendering, buffer creation, and presentation.
// Both WaylandPipeline and X11Pipeline implement this interface.
type PlatformPresenter interface {
	// RenderToFramebuffer renders the display list to the framebuffer's GPU target.
	RenderToFramebuffer(dl *displaylist.DisplayList, fb FramebufferHandle) error

	// EnsurePlatformBuffer creates or retrieves the platform-specific buffer handle.
	EnsurePlatformBuffer(fb FramebufferHandle) error

	// PresentBuffer presents the framebuffer to the display server.
	PresentBuffer(fb FramebufferHandle) error

	// ReleaseFramebuffer releases the framebuffer back to the pool.
	ReleaseFramebuffer(fb FramebufferHandle)

	// IsClosed returns true if the presenter has been closed.
	IsClosed() bool
}

// FramebufferPool manages the lifecycle of framebuffers.
type FramebufferPool interface {
	Acquire(ctx context.Context) (FramebufferHandle, error)
	MarkDisplaying(fb FramebufferHandle) error
}

// RenderAndPresent implements the common render-and-present pattern shared by
// Wayland and X11 display pipelines.
//
// This function:
//  1. Acquires an available framebuffer from the pool
//  2. Renders the display list to the GPU render target
//  3. Creates or retrieves the platform-specific buffer handle
//  4. Presents the buffer to the display server
//  5. Marks the framebuffer as displaying
//
// The function blocks until a framebuffer is available. Use context for timeout control.
func RenderAndPresent(
	ctx context.Context,
	dl *displaylist.DisplayList,
	pool FramebufferPool,
	presenter PlatformPresenter,
) error {
	if presenter.IsClosed() {
		return ErrPresenterClosed
	}

	fb, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}

	if err := presenter.RenderToFramebuffer(dl, fb); err != nil {
		presenter.ReleaseFramebuffer(fb)
		return err
	}

	if err := presenter.EnsurePlatformBuffer(fb); err != nil {
		presenter.ReleaseFramebuffer(fb)
		return err
	}

	if err := presenter.PresentBuffer(fb); err != nil {
		presenter.ReleaseFramebuffer(fb)
		return err
	}

	if err := pool.MarkDisplaying(fb); err != nil {
		presenter.ReleaseFramebuffer(fb)
		return err
	}

	return nil
}
