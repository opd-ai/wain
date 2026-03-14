package display

import (
	"context"
	"errors"
	"fmt"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/render/backend"
	presentpkg "github.com/opd-ai/wain/internal/render/present"
	"github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/dri3"
	"github.com/opd-ai/wain/internal/x11/present"
)

var (
	// ErrX11InvalidWindow is returned when the X11 window is invalid.
	ErrX11InvalidWindow = errors.New("display: invalid X11 window")

	// ErrX11NoDRI3 is returned when DRI3 extension is not supported.
	ErrX11NoDRI3 = errors.New("display: X server doesn't support DRI3 extension")

	// ErrX11NoPresent is returned when Present extension is not supported.
	ErrX11NoPresent = errors.New("display: X server doesn't support Present extension")
)

// x11ConnectionAdapter adapts client.Connection to dri3/present.Connection interfaces.
type x11ConnectionAdapter struct {
	conn *client.Connection
}

// For DRI3 Connection interface
func (a *x11ConnectionAdapter) AllocXID() (dri3.XID, error) {
	xid, err := a.conn.AllocXID()
	return dri3.XID(xid), err
}

// SendRequest sends an X11 protocol request without expecting a reply.
func (a *x11ConnectionAdapter) SendRequest(buf []byte) error {
	return a.conn.SendRequest(buf)
}

// SendRequestAndReply sends an X11 request and blocks until the reply arrives, returning the reply bytes or an error.
func (a *x11ConnectionAdapter) SendRequestAndReply(req []byte) ([]byte, error) {
	return a.conn.SendRequestAndReply(req)
}

// SendRequestWithFDs sends an X11 protocol request with file descriptors without expecting a reply.
func (a *x11ConnectionAdapter) SendRequestWithFDs(req []byte, fds []int) error {
	return a.conn.SendRequestWithFDs(req, fds)
}

// SendRequestAndReplyWithFDs sends an X11 request with file descriptors and blocks until the reply arrives, returning the reply bytes, reply file descriptors, or an error.
func (a *x11ConnectionAdapter) SendRequestAndReplyWithFDs(req []byte, fds []int) ([]byte, []int, error) {
	return a.conn.SendRequestAndReplyWithFDs(req, fds)
}

// ExtensionOpcode returns the major opcode for a named X11 extension.
func (a *x11ConnectionAdapter) ExtensionOpcode(name string) (uint8, error) {
	return a.conn.ExtensionOpcode(name)
}

// x11PresentAdapter wraps x11ConnectionAdapter for Present interface
type x11PresentAdapter struct {
	*x11ConnectionAdapter
}

// AllocXID for present.Connection interface (overrides with present.XID return type)
func (a *x11PresentAdapter) AllocXID() (present.XID, error) {
	xid, err := a.conn.AllocXID()
	return present.XID(xid), err
}

// X11Pipeline integrates GPU rendering with X11 server via DRI3 and Present.
type X11Pipeline struct {
	renderer       backend.Renderer
	conn           *client.Connection
	dri3Adapter    *x11ConnectionAdapter
	presentAdapter *x11PresentAdapter
	window         client.XID
	dri3           *dri3.Extension
	present        *present.Extension
	pool           *FramebufferPool
	serial         uint32
	closed         bool
}

// NewX11Pipeline creates a new GPU→X11 display pipeline.
//
// Parameters:
//   - conn: X11 connection
//   - window: target window for presentation
//   - dri3: DRI3 extension (must be initialized)
//   - present: Present extension (must be initialized)
//   - renderer: GPU backend that renders and exports DMA-BUF
//
// Returns an error if extensions are not supported.
func NewX11Pipeline(
	conn *client.Connection,
	window client.XID,
	dri3Ext *dri3.Extension,
	presentExt *present.Extension,
	renderer backend.Renderer,
) (*X11Pipeline, error) {
	if window == 0 {
		return nil, ErrX11InvalidWindow
	}
	if dri3Ext == nil {
		return nil, ErrX11NoDRI3
	}
	if presentExt == nil {
		return nil, ErrX11NoPresent
	}

	// Create triple-buffered framebuffer pool
	pool, err := NewFramebufferPool(3)
	if err != nil {
		return nil, fmt.Errorf("display: failed to create framebuffer pool: %w", err)
	}

	dri3Adapter := &x11ConnectionAdapter{conn: conn}
	presentAdapter := &x11PresentAdapter{x11ConnectionAdapter: dri3Adapter}

	return &X11Pipeline{
		renderer:       renderer,
		conn:           conn,
		dri3Adapter:    dri3Adapter,
		presentAdapter: presentAdapter,
		window:         window,
		dri3:           dri3Ext,
		present:        presentExt,
		pool:           pool,
		serial:         1,
	}, nil
}

// RenderAndPresent renders a display list and presents the result to X server.
//
// This method:
//  1. Acquires an available framebuffer from the pool
//  2. Renders the display list to the GPU render target
//  3. Exports the render target as DMA-BUF
//  4. Creates an X11 pixmap from the DMA-BUF (if first time)
//  5. Presents the pixmap using the Present extension
//
// The method blocks until a framebuffer is available. Use context for timeout control.
func (p *X11Pipeline) RenderAndPresent(ctx context.Context, dl *displaylist.DisplayList) error {
	return presentpkg.RenderAndPresent(ctx, dl, &poolAdapter{p.pool}, p)
}

// IsClosed implements presentpkg.PlatformPresenter.
func (p *X11Pipeline) IsClosed() bool {
	return p.closed
}

// RenderToFramebuffer implements presentpkg.PlatformPresenter.
func (p *X11Pipeline) RenderToFramebuffer(dl *displaylist.DisplayList, fb presentpkg.FramebufferHandle) error {
	return p.renderToFramebuffer(dl, fb.(*Framebuffer))
}

// EnsurePlatformBuffer implements presentpkg.PlatformPresenter.
func (p *X11Pipeline) EnsurePlatformBuffer(fb presentpkg.FramebufferHandle) error {
	return p.ensureX11Pixmap(fb.(*Framebuffer))
}

// PresentBuffer implements presentpkg.PlatformPresenter.
func (p *X11Pipeline) PresentBuffer(fb presentpkg.FramebufferHandle) error {
	return p.presentPixmap(fb.(*Framebuffer))
}

// ReleaseFramebuffer implements presentpkg.PlatformPresenter.
func (p *X11Pipeline) ReleaseFramebuffer(fb presentpkg.FramebufferHandle) {
	p.releaseFramebuffer(fb.(*Framebuffer))
}

// releaseFramebuffer marks a framebuffer as available after presentation completes.
func (p *X11Pipeline) releaseFramebuffer(fb *Framebuffer) {
	fb.setState(FramebufferAvailable)
	fb.signalRelease()
}

// renderToFramebuffer renders a display list to a framebuffer using the configured renderer.
func (p *X11Pipeline) renderToFramebuffer(dl *displaylist.DisplayList, fb *Framebuffer) error {
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

// ensureX11Pixmap creates an X11 pixmap for the framebuffer if one does not exist.
func (p *X11Pipeline) ensureX11Pixmap(fb *Framebuffer) error {
	if fb.BufferID != 0 {
		return nil
	}

	pixmapXID, err := p.createX11Pixmap(fb)
	if err != nil {
		return fmt.Errorf("display: failed to create pixmap: %w", err)
	}
	p.pool.Register(fb, pixmapXID)
	return nil
}

// presentPixmap submits a pixmap to the X11 Present extension for display.
func (p *X11Pipeline) presentPixmap(fb *Framebuffer) error {
	if err := p.present.PresentPixmap(p.presentAdapter, present.PixmapPresentOptions{
		Window:       present.XID(p.window),
		Pixmap:       present.XID(fb.BufferID),
		Serial:       p.serial,
		ValidRegion:  0,
		UpdateRegion: 0,
		XOff:         0,
		YOff:         0,
		TargetMSC:    0,
		Divisor:      0,
		Remainder:    0,
		Options:      present.PresentOptionNone,
	}); err != nil {
		return fmt.Errorf("display: present pixmap failed: %w", err)
	}

	p.serial++
	return nil
}

// createX11Pixmap creates an X11 pixmap from a framebuffer's DMA-BUF.
func (p *X11Pipeline) createX11Pixmap(fb *Framebuffer) (uint32, error) {
	pixmapXID, err := p.conn.AllocXID()
	if err != nil {
		return 0, fmt.Errorf("failed to allocate pixmap XID: %w", err)
	}

	size := fb.Stride * fb.Height

	// Create pixmap via DRI3
	if err := p.dri3.PixmapFromBuffer(
		p.dri3Adapter,
		dri3.XID(pixmapXID),
		dri3.XID(p.window), // use window as drawable for depth/visual
		size,
		uint16(fb.Width),
		uint16(fb.Height),
		uint16(fb.Stride),
		24, // depth (RGB without alpha channel consideration)
		32, // bpp (ARGB8888)
		fb.Fd,
	); err != nil {
		return 0, fmt.Errorf("dri3 pixmap from buffer failed: %w", err)
	}

	return uint32(pixmapXID), nil
}

// OnPixmapReady should be called when the X server sends a PresentIdleNotify event.
// This marks the pixmap as available for reuse.
func (p *X11Pipeline) OnPixmapReady(pixmapID uint32) error {
	return p.pool.OnRelease(pixmapID)
}

// OnPresentComplete should be called when the X server sends a PresentCompleteNotify event.
// This is informational and doesn't affect the pool state.
func (p *X11Pipeline) OnPresentComplete(serial uint32) {
	// Could track presentation timing here if needed
}

// Close closes the pipeline and releases all resources.
func (p *X11Pipeline) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	return p.pool.Close()
}

// PresentRendered presents the most recently rendered GPU frame to the X server
// without re-rendering. The caller must have already invoked the renderer's
// Render method before calling this.
func (p *X11Pipeline) PresentRendered(ctx context.Context) error {
	if p.closed {
		return presentpkg.ErrPresenterClosed
	}

	fb, err := p.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("display: acquire framebuffer: %w", err)
	}

	fd, err := p.renderer.Present()
	if err != nil {
		p.releaseFramebuffer(fb)
		return fmt.Errorf("display: export dmabuf: %w", err)
	}

	if fb.Fd < 0 {
		fb.Fd = fd
		w, h := p.renderer.Dimensions()
		fb.Width = uint32(w)
		fb.Height = uint32(h)
		fb.Stride = uint32(w) * 4
	}

	if err := p.ensureX11Pixmap(fb); err != nil {
		p.releaseFramebuffer(fb)
		return fmt.Errorf("display: ensure pixmap: %w", err)
	}

	if err := p.presentPixmap(fb); err != nil {
		p.releaseFramebuffer(fb)
		return err
	}

	return p.pool.MarkDisplaying(fb)
}
