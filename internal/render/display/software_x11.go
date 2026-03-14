package display

import (
	"context"
	"fmt"

	"github.com/opd-ai/wain/internal/render/backend"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/gc"
)

// gcConnectionAdapter adapts *x11client.Connection to gc.Connection.
type gcConnectionAdapter struct {
	conn *x11client.Connection
}

// AllocXID allocates an X11 resource ID, adapting the type to gc.XID.
func (a *gcConnectionAdapter) AllocXID() (gc.XID, error) {
	xid, err := a.conn.AllocXID()
	return gc.XID(xid), err
}

// SendRequest sends a raw X11 protocol request via the underlying connection.
func (a *gcConnectionAdapter) SendRequest(buf []byte) error {
	return a.conn.SendRequest(buf)
}

// SoftwareX11Presenter presents software-rendered frames to an X11 window via PutImage.
type SoftwareX11Presenter struct {
	conn     *x11client.Connection
	window   x11client.XID
	gcID     gc.XID
	renderer *backend.SoftwareBackend
}

// NewSoftwareX11Presenter creates a presenter that transfers pixels using the X11
// PutImage request. A GC is created on first use and reused for all subsequent frames.
func NewSoftwareX11Presenter(conn *x11client.Connection, window x11client.XID, renderer *backend.SoftwareBackend) (*SoftwareX11Presenter, error) {
	adapter := &gcConnectionAdapter{conn: conn}
	gcID, err := gc.CreateGC(adapter, gc.XID(window), 0, nil)
	if err != nil {
		return nil, fmt.Errorf("display/x11: create GC: %w", err)
	}
	return &SoftwareX11Presenter{
		conn:     conn,
		window:   window,
		gcID:     gcID,
		renderer: renderer,
	}, nil
}

// Present transfers the software-rendered pixel buffer to the X11 window using PutImage.
func (p *SoftwareX11Presenter) Present(_ context.Context) error {
	pixels := p.renderer.Pixels()
	if pixels == nil {
		return nil
	}

	width, height := p.renderer.Dimensions()
	if width <= 0 || height <= 0 {
		return nil
	}

	adapter := &gcConnectionAdapter{conn: p.conn}
	if err := gc.PutImage(
		adapter,
		gc.XID(p.window),
		p.gcID,
		uint16(width), uint16(height),
		0, 0,
		24,
		gc.FormatZPixmap,
		pixels,
	); err != nil {
		return fmt.Errorf("display/x11: PutImage: %w", err)
	}

	return nil
}

// Close frees the X11 GC used for rendering.
func (p *SoftwareX11Presenter) Close() error {
	adapter := &gcConnectionAdapter{conn: p.conn}
	if err := gc.FreeGC(adapter, p.gcID); err != nil {
		return fmt.Errorf("display/x11: FreeGC: %w", err)
	}
	return nil
}
