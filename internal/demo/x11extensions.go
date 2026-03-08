package demo

import (
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/dri3"
	"github.com/opd-ai/wain/internal/x11/present"
)

// DRI3ConnectionAdapter adapts x11client.Connection to dri3.Connection interface.
type DRI3ConnectionAdapter struct {
	*x11client.Connection
}

// NewDRI3ConnectionAdapter creates a DRI3ConnectionAdapter from an x11client.Connection.
func NewDRI3ConnectionAdapter(conn *x11client.Connection) *DRI3ConnectionAdapter {
	return &DRI3ConnectionAdapter{Connection: conn}
}

// AllocXID allocates an X ID by delegating to the underlying connection.
func (a *DRI3ConnectionAdapter) AllocXID() (dri3.XID, error) {
	xid, err := a.Connection.AllocXID()
	return dri3.XID(xid), err
}

// SendRequest sends an X11 request by delegating to the underlying connection.
func (a *DRI3ConnectionAdapter) SendRequest(buf []byte) error {
	return a.Connection.SendRequest(buf)
}

// SendRequestAndReply sends a request and receives a reply by delegating to the underlying connection.
func (a *DRI3ConnectionAdapter) SendRequestAndReply(req []byte) ([]byte, error) {
	return a.Connection.SendRequestAndReply(req)
}

// SendRequestWithFDs sends a request with file descriptors by delegating to the underlying connection.
func (a *DRI3ConnectionAdapter) SendRequestWithFDs(req []byte, fds []int) error {
	return a.Connection.SendRequestWithFDs(req, fds)
}

// SendRequestAndReplyWithFDs sends a request with file descriptors and receives a reply.
func (a *DRI3ConnectionAdapter) SendRequestAndReplyWithFDs(req []byte, fds []int) ([]byte, []int, error) {
	return a.Connection.SendRequestAndReplyWithFDs(req, fds)
}

// ExtensionOpcode returns the opcode for an X11 extension by delegating to the underlying connection.
func (a *DRI3ConnectionAdapter) ExtensionOpcode(name string) (uint8, error) {
	return a.Connection.ExtensionOpcode(name)
}

// PresentConnectionAdapter adapts x11client.Connection to present.Connection interface.
type PresentConnectionAdapter struct {
	*x11client.Connection
}

// NewPresentConnectionAdapter creates a PresentConnectionAdapter from an x11client.Connection.
func NewPresentConnectionAdapter(conn *x11client.Connection) *PresentConnectionAdapter {
	return &PresentConnectionAdapter{Connection: conn}
}

// AllocXID allocates an X ID by delegating to the underlying connection.
func (a *PresentConnectionAdapter) AllocXID() (present.XID, error) {
	xid, err := a.Connection.AllocXID()
	return present.XID(xid), err
}

// SendRequest sends an X11 request by delegating to the underlying connection.
func (a *PresentConnectionAdapter) SendRequest(buf []byte) error {
	return a.Connection.SendRequest(buf)
}

// SendRequestAndReply sends a request and receives a reply by delegating to the underlying connection.
func (a *PresentConnectionAdapter) SendRequestAndReply(req []byte) ([]byte, error) {
	return a.Connection.SendRequestAndReply(req)
}

// ExtensionOpcode returns the opcode for an X11 extension by delegating to the underlying connection.
func (a *PresentConnectionAdapter) ExtensionOpcode(name string) (uint8, error) {
	return a.Connection.ExtensionOpcode(name)
}
