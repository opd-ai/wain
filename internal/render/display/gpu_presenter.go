package display

import (
	"context"

	"github.com/opd-ai/wain/internal/render/backend"
	"github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/dri3"
	"github.com/opd-ai/wain/internal/x11/present"
)

// GPUWaylandPresenter wraps a WaylandPipeline and implements the Presenter interface
// for the GPU→Wayland rendering path. It assumes the renderer has already been asked
// to render before Present is called (via RenderBridge.Render).
type GPUWaylandPresenter struct {
	pipeline *WaylandPipeline
}

// NewGPUWaylandPresenter creates a Presenter for the GPU→Wayland path.
func NewGPUWaylandPresenter(pipeline *WaylandPipeline) *GPUWaylandPresenter {
	return &GPUWaylandPresenter{pipeline: pipeline}
}

// Present exports the already-rendered GPU framebuffer as a DMA-BUF and commits
// it to the Wayland compositor.
func (p *GPUWaylandPresenter) Present(ctx context.Context) error {
	return p.pipeline.PresentRendered(ctx)
}

// Close shuts down the underlying Wayland pipeline and releases its resources.
func (p *GPUWaylandPresenter) Close() error {
	return p.pipeline.Close()
}

// GPUX11Presenter wraps an X11Pipeline and implements the Presenter interface
// for the GPU→X11 rendering path.
type GPUX11Presenter struct {
	pipeline *X11Pipeline
}

// NewGPUX11Presenter creates a Presenter for the GPU→X11 path.
func NewGPUX11Presenter(pipeline *X11Pipeline) *GPUX11Presenter {
	return &GPUX11Presenter{pipeline: pipeline}
}

// Present exports the already-rendered GPU framebuffer and presents it via the
// X11 Present extension.
func (p *GPUX11Presenter) Present(ctx context.Context) error {
	return p.pipeline.PresentRendered(ctx)
}

// Close shuts down the underlying X11 pipeline and releases its resources.
func (p *GPUX11Presenter) Close() error {
	return p.pipeline.Close()
}

// NewGPUX11PresenterFromConn queries the X11 server for DRI3 and Present
// extensions, creates an X11Pipeline, and returns a GPU presenter.
// Returns an error if either extension is unavailable or pipeline creation fails.
func NewGPUX11PresenterFromConn(conn *client.Connection, window client.XID, renderer backend.Renderer) (*GPUX11Presenter, error) {
	dri3Adapter := &x11ConnectionAdapter{conn: conn}
	presentAdapter := &x11PresentAdapter{x11ConnectionAdapter: dri3Adapter}

	dri3Ext, err := dri3.QueryExtension(dri3Adapter)
	if err != nil {
		return nil, err
	}

	presentExt, err := present.QueryExtension(presentAdapter)
	if err != nil {
		return nil, err
	}

	pipeline, err := NewX11Pipeline(conn, window, dri3Ext, presentExt, renderer)
	if err != nil {
		return nil, err
	}

	return NewGPUX11Presenter(pipeline), nil
}
