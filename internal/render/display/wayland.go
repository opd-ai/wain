package display

import (
	"context"
	"errors"
	"fmt"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/render/backend"
	presentpkg "github.com/opd-ai/wain/internal/render/present"
	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/dmabuf"
)

var (
	// ErrWaylandInvalidSurface is returned when the Wayland surface is nil.
	ErrWaylandInvalidSurface = errors.New("display: invalid Wayland surface")

	// ErrWaylandNoDmabuf is returned when the compositor doesn't support dmabuf.
	ErrWaylandNoDmabuf = errors.New("display: compositor doesn't support zwp_linux_dmabuf_v1")

	// ErrWaylandFormatUnsupported is returned when the compositor doesn't support ARGB8888.
	ErrWaylandFormatUnsupported = errors.New("display: compositor doesn't support ARGB8888 format")
)

// WaylandPipeline integrates GPU rendering with Wayland compositor via DMA-BUF.
type WaylandPipeline struct {
	renderer backend.Renderer
	surface  *client.Surface
	dmabuf   *dmabuf.Dmabuf
	pool     *FramebufferPool
	callback *client.Callback
	closed   bool
}

// NewWaylandPipeline creates a new GPU→Wayland display pipeline.
//
// Parameters:
//   - surface: wl_surface to present to
//   - dmabuf: zwp_linux_dmabuf_v1 global (from registry)
//   - renderer: GPU backend that renders and exports DMA-BUF
//
// Returns an error if the compositor doesn't support required formats.
func NewWaylandPipeline(surface *client.Surface, dmabufGlobal *dmabuf.Dmabuf, renderer backend.Renderer) (*WaylandPipeline, error) {
	if surface == nil {
		return nil, ErrWaylandInvalidSurface
	}
	if dmabufGlobal == nil {
		return nil, ErrWaylandNoDmabuf
	}

	// Verify compositor supports ARGB8888 (most common format)
	if !dmabufGlobal.HasFormat(dmabuf.FormatARGB8888) {
		return nil, ErrWaylandFormatUnsupported
	}

	// Create triple-buffered framebuffer pool
	pool, err := NewFramebufferPool(3)
	if err != nil {
		return nil, fmt.Errorf("display: failed to create framebuffer pool: %w", err)
	}

	return &WaylandPipeline{
		renderer: renderer,
		surface:  surface,
		dmabuf:   dmabufGlobal,
		pool:     pool,
	}, nil
}

// RenderAndPresent renders a display list and presents the result to the compositor.
//
// This method:
//  1. Acquires an available framebuffer from the pool
//  2. Renders the display list to the GPU render target
//  3. Exports the render target as DMA-BUF
//  4. Creates a wl_buffer from the DMA-BUF (if first time)
//  5. Attaches the buffer to the surface
//  6. Commits the surface to make it visible
//
// The method blocks until a framebuffer is available. Use context for timeout control.
func (p *WaylandPipeline) RenderAndPresent(ctx context.Context, dl *displaylist.DisplayList) error {
	return presentpkg.RenderAndPresent(ctx, dl, &poolAdapter{p.pool}, p)
}

// IsClosed implements presentpkg.PlatformPresenter.
func (p *WaylandPipeline) IsClosed() bool {
	return p.closed
}

// poolAdapter adapts *FramebufferPool to presentpkg.FramebufferPool interface.
type poolAdapter struct {
	pool *FramebufferPool
}

// Acquire obtains an available framebuffer from the pool, blocking if necessary.
// It delegates to the underlying FramebufferPool's Acquire method and returns
// the framebuffer as a FramebufferHandle to satisfy the presentpkg.FramebufferPool contract.
func (a *poolAdapter) Acquire(ctx context.Context) (presentpkg.FramebufferHandle, error) {
	return a.pool.Acquire(ctx)
}

// MarkDisplaying marks the framebuffer as actively being displayed by the compositor.
// It type-asserts the FramebufferHandle back to *Framebuffer and delegates to the underlying
// pool's MarkDisplaying method. This prevents the framebuffer from being recycled while
// still visible on screen.
func (a *poolAdapter) MarkDisplaying(fb presentpkg.FramebufferHandle) error {
	return a.pool.MarkDisplaying(fb.(*Framebuffer))
}

// RenderToFramebuffer implements presentpkg.PlatformPresenter.
func (p *WaylandPipeline) RenderToFramebuffer(dl *displaylist.DisplayList, fb presentpkg.FramebufferHandle) error {
	return p.renderToFramebuffer(dl, fb.(*Framebuffer))
}

// EnsurePlatformBuffer implements presentpkg.PlatformPresenter.
func (p *WaylandPipeline) EnsurePlatformBuffer(fb presentpkg.FramebufferHandle) error {
	return p.ensureWaylandBuffer(fb.(*Framebuffer))
}

// PresentBuffer implements presentpkg.PlatformPresenter.
func (p *WaylandPipeline) PresentBuffer(fb presentpkg.FramebufferHandle) error {
	return p.commitToSurface(fb.(*Framebuffer))
}

// ReleaseFramebuffer implements presentpkg.PlatformPresenter.
func (p *WaylandPipeline) ReleaseFramebuffer(fb presentpkg.FramebufferHandle) {
	p.releaseFramebuffer(fb.(*Framebuffer))
}

// releaseFramebuffer marks a framebuffer as available after compositor release.
func (p *WaylandPipeline) releaseFramebuffer(fb *Framebuffer) {
	fb.setState(FramebufferAvailable)
	fb.signalRelease()
}

// renderToFramebuffer renders a display list to a framebuffer using the configured renderer.
func (p *WaylandPipeline) renderToFramebuffer(dl *displaylist.DisplayList, fb *Framebuffer) error {
	if err := p.renderer.Render(dl); err != nil {
		return fmt.Errorf("display: render failed: %w", err)
	}

	fd, err := p.renderer.Present()
	if err != nil {
		return fmt.Errorf("display: present failed: %w", err)
	}

	if fb.Fd < 0 {
		fb.Fd = fd
		width, height := p.renderer.Dimensions()
		fb.Width = uint32(width)
		fb.Height = uint32(height)
		fb.Stride = uint32(width) * 4
	}

	return nil
}

// ensureWaylandBuffer creates a wl_buffer for the framebuffer if one does not exist.
func (p *WaylandPipeline) ensureWaylandBuffer(fb *Framebuffer) error {
	if fb.BufferID != 0 {
		return nil
	}

	bufferID, err := p.createWaylandBuffer(fb)
	if err != nil {
		return fmt.Errorf("display: failed to create wl_buffer: %w", err)
	}
	p.pool.Register(fb, bufferID)
	return nil
}

// commitToSurface attaches and commits a framebuffer to the Wayland surface with damage tracking.
func (p *WaylandPipeline) commitToSurface(fb *Framebuffer) error {
	if err := p.surface.Attach(fb.BufferID, 0, 0); err != nil {
		return fmt.Errorf("display: attach failed: %w", err)
	}

	if err := p.surface.Damage(0, 0, int32(fb.Width), int32(fb.Height)); err != nil {
		return fmt.Errorf("display: damage failed: %w", err)
	}

	callback, err := p.surface.Frame()
	if err != nil {
		return fmt.Errorf("display: frame callback failed: %w", err)
	}
	p.callback = callback

	if err := p.surface.Commit(); err != nil {
		return fmt.Errorf("display: commit failed: %w", err)
	}

	return nil
}

// createWaylandBuffer creates a wl_buffer from a framebuffer's DMA-BUF.
func (p *WaylandPipeline) createWaylandBuffer(fb *Framebuffer) (uint32, error) {
	params, err := p.dmabuf.CreateParams()
	if err != nil {
		return 0, fmt.Errorf("create_params failed: %w", err)
	}

	// Add the DMA-BUF plane (single plane for ARGB8888)
	if err := params.Add(
		int32(fb.Fd), // DMA-BUF file descriptor
		0,            // plane index (0 for single-plane)
		0,            // offset
		fb.Stride,    // stride in bytes
		0,            // modifier high (linear)
		0,            // modifier low (linear)
	); err != nil {
		return 0, fmt.Errorf("add plane failed: %w", err)
	}

	// Create the wl_buffer
	bufferID, err := params.Create(
		int32(fb.Width),
		int32(fb.Height),
		dmabuf.FormatARGB8888,
		0, // flags
	)
	if err != nil {
		return 0, fmt.Errorf("create buffer failed: %w", err)
	}

	return bufferID, nil
}

// OnBufferRelease should be called when the compositor sends a wl_buffer.release event.
// This marks the buffer as available for reuse.
func (p *WaylandPipeline) OnBufferRelease(bufferID uint32) error {
	return p.pool.OnRelease(bufferID)
}

// WaitFrameCallback blocks until the compositor signals it's ready for the next frame.
// Returns the frame timestamp in milliseconds.
func (p *WaylandPipeline) WaitFrameCallback(ctx context.Context) (uint32, error) {
	if p.callback == nil {
		return 0, errors.New("display: no active frame callback")
	}

	select {
	case timestamp := <-p.callback.Done():
		return timestamp, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// Close closes the pipeline and releases all resources.
func (p *WaylandPipeline) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	return p.pool.Close()
}
