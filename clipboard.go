// Package wain provides clipboard access via SetClipboard and GetClipboard on Window.
//
// Clipboard operations are dispatched to the active display server backend:
//   - Wayland: wl_data_device protocol (data source/device/offer)
//   - X11: ICCCM selection protocol (CLIPBOARD atom via SetSelectionOwner/ConvertSelection)
package wain

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/opd-ai/wain/internal/wayland/datadevice"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/selection"
	x11wire "github.com/opd-ai/wain/internal/x11/wire"
)

// SetClipboard sets the clipboard to the provided text.
// It is dispatched to the appropriate backend based on the active display server.
func (w *Window) SetClipboard(text string) error {
	switch w.app.displayServer {
	case DisplayServerWayland:
		return w.setWaylandClipboard(text)
	case DisplayServerX11:
		return w.setX11Clipboard(text)
	default:
		return ErrNoDisplay
	}
}

// GetClipboard returns the current clipboard text content.
// It is dispatched to the appropriate backend based on the active display server.
func (w *Window) GetClipboard() (string, error) {
	switch w.app.displayServer {
	case DisplayServerWayland:
		return w.getWaylandClipboard()
	case DisplayServerX11:
		return w.getX11Clipboard()
	default:
		return "", ErrNoDisplay
	}
}

// setX11Clipboard writes text into the X11 CLIPBOARD selection.
func (w *Window) setX11Clipboard(text string) error {
	if w.app.x11SelectionMgr == nil {
		return fmt.Errorf("wain: clipboard unavailable (X11 selection manager not initialised)")
	}
	return w.app.x11SelectionMgr.SetClipboard(text)
}

// getX11Clipboard reads the current X11 CLIPBOARD selection as plain text.
func (w *Window) getX11Clipboard() (string, error) {
	if w.app.x11SelectionMgr == nil {
		return "", fmt.Errorf("wain: clipboard unavailable (X11 selection manager not initialised)")
	}
	return w.app.x11SelectionMgr.GetClipboard()
}

// setWaylandClipboard offers text to the Wayland compositor as a data source and
// sets it as the active clipboard selection.  A background goroutine serves
// any Receive requests until the compositor cancels the source.
func (w *Window) setWaylandClipboard(text string) error {
	if w.app.waylandDataDeviceMgr == nil || w.app.waylandDataDevice == nil {
		return fmt.Errorf("wain: clipboard unavailable (Wayland data device not initialised)")
	}

	source, err := w.app.waylandDataDeviceMgr.CreateDataSource()
	if err != nil {
		return fmt.Errorf("wain: failed to create data source: %w", err)
	}
	for _, mime := range []string{"text/plain;charset=utf-8", "text/plain"} {
		if err := source.Offer(mime); err != nil {
			return fmt.Errorf("wain: failed to offer MIME type %q: %w", mime, err)
		}
	}

	go serveClipboardSource(source, text)

	if err := w.app.waylandDataDevice.SetSelection(source, 0); err != nil {
		return fmt.Errorf("wain: failed to set clipboard selection: %w", err)
	}
	return nil
}

// getWaylandClipboard reads plain text from the current Wayland clipboard offer.
func (w *Window) getWaylandClipboard() (string, error) {
	if w.app.waylandDataDevice == nil {
		return "", fmt.Errorf("wain: clipboard unavailable (Wayland data device not initialised)")
	}

	offer := w.app.waylandDataDevice.Selection()
	if offer == nil {
		return "", nil
	}

	mime := selectClipboardMime(offer.MimeTypes())
	if mime == "" {
		return "", nil
	}

	data, err := offer.ReadData(mime)
	if err != nil {
		return "", fmt.Errorf("wain: failed to read clipboard data: %w", err)
	}
	return string(data), nil
}

// selectClipboardMime returns the best plain-text MIME type from the offered set,
// preferring UTF-8 encoded variants.
func selectClipboardMime(offered []string) string {
	const utf8Mime = "text/plain;charset=utf-8"
	const plainMime = "text/plain"

	offeredSet := make(map[string]bool, len(offered))
	for _, m := range offered {
		offeredSet[m] = true
	}
	if offeredSet[utf8Mime] {
		return utf8Mime
	}
	if offeredSet[plainMime] {
		return plainMime
	}
	return ""
}

// serveClipboardSource writes text to any file-descriptor send requests from
// the compositor until the source is cancelled.
func serveClipboardSource(source *datadevice.Source, text string) {
	for {
		select {
		case req := <-source.SendRequests():
			f := os.NewFile(uintptr(req.FD), "wain-clipboard")
			_, _ = f.WriteString(text)
			_ = f.Close()
		case <-source.Cancelled():
			return
		}
	}
}

// x11SelectionAdapter adapts *x11client.Connection to the selection.Conn interface,
// which uses a higher-level (opcode, data) request model rather than raw buffers.
type x11SelectionAdapter struct {
	conn *x11client.Connection
}

// AllocXID allocates a new X11 resource ID (errors are silently discarded).
func (a *x11SelectionAdapter) AllocXID() uint32 {
	id, _ := a.conn.AllocXID()
	return uint32(id)
}

// SendRequest encodes an X11 request header and dispatches the raw buffer.
func (a *x11SelectionAdapter) SendRequest(opcode uint8, data []byte) error {
	buf := a.buildRequest(opcode, 0, data)
	return a.conn.SendRequest(buf)
}

// SendRequestAndReply encodes an X11 request header and waits for a reply.
func (a *x11SelectionAdapter) SendRequestAndReply(opcode uint8, data []byte) ([]byte, error) {
	buf := a.buildRequest(opcode, 0, data)
	return a.conn.SendRequestAndReply(buf)
}

// InternAtom delegates to the underlying connection.
func (a *x11SelectionAdapter) InternAtom(name string, onlyIfExists bool) (uint32, error) {
	return a.conn.InternAtom(name, onlyIfExists)
}

// ChangeProperty delegates to the underlying connection.
func (a *x11SelectionAdapter) ChangeProperty(window, property, typ uint32, format, mode uint8, data []byte) error {
	return a.conn.ChangeProperty(window, property, typ, format, mode, data)
}

// GetProperty reads a window property from the X server.
func (a *x11SelectionAdapter) GetProperty(window, property, typ, offset, length uint32, delete bool) ([]byte, uint32, error) {
	var deleteByte uint8
	if delete {
		deleteByte = 1
	}

	// GetProperty request: window(4) + property(4) + type(4) + longOffset(4) + longLength(4) = 20 bytes
	buf := new(bytes.Buffer)
	x11wire.EncodeRequestHeader(buf, x11wire.OpcodeGetProperty, deleteByte, 6) // 6 × 4 = 24 bytes
	binary.Write(buf, binary.LittleEndian, window)                             //nolint:errcheck
	binary.Write(buf, binary.LittleEndian, property)                           //nolint:errcheck
	binary.Write(buf, binary.LittleEndian, typ)                                //nolint:errcheck
	binary.Write(buf, binary.LittleEndian, offset)                             //nolint:errcheck
	binary.Write(buf, binary.LittleEndian, length)                             //nolint:errcheck

	reply, err := a.conn.SendRequestAndReply(buf.Bytes())
	if err != nil {
		return nil, 0, fmt.Errorf("GetProperty: %w", err)
	}
	if len(reply) < 32 {
		return nil, 0, fmt.Errorf("GetProperty: reply too short (%d bytes)", len(reply))
	}

	// reply[0]=1 (reply type), reply[1]=format, reply[4:8]=dataLength (in 4-byte units)
	// reply[8:12]=type, reply[12:16]=bytesAfter, reply[16:20]=nItems
	bytesAfter := binary.LittleEndian.Uint32(reply[12:16])
	nItems := binary.LittleEndian.Uint32(reply[16:20])
	format := reply[1]

	bytesPerItem := uint32(format) / 8
	if bytesPerItem == 0 {
		bytesPerItem = 1
	}
	dataLen := nItems * bytesPerItem
	if uint32(len(reply)) < 32+dataLen {
		return nil, bytesAfter, fmt.Errorf("GetProperty: reply data truncated")
	}
	return reply[32 : 32+dataLen], bytesAfter, nil
}

// DeleteProperty removes a property atom from a window.
func (a *x11SelectionAdapter) DeleteProperty(window, property uint32) error {
	// DeleteProperty request: window(4) + property(4) = 8 bytes → 3 × 4 units
	buf := new(bytes.Buffer)
	x11wire.EncodeRequestHeader(buf, x11wire.OpcodeDeleteProperty, 0, 3)
	binary.Write(buf, binary.LittleEndian, window)   //nolint:errcheck
	binary.Write(buf, binary.LittleEndian, property) //nolint:errcheck
	return a.conn.SendRequest(buf.Bytes())
}

// buildRequest constructs a full X11 request buffer with header.
func (a *x11SelectionAdapter) buildRequest(opcode, data1 uint8, payload []byte) []byte {
	totalBytes := 4 + len(payload) // header(4) + payload
	units := uint16((totalBytes + 3) / 4)
	buf := new(bytes.Buffer)
	x11wire.EncodeRequestHeader(buf, opcode, data1, units)
	buf.Write(payload)
	// Pad to 4-byte boundary
	if pad := totalBytes % 4; pad != 0 {
		buf.Write(make([]byte, 4-pad))
	}
	return buf.Bytes()
}

// newX11SelectionManager creates a selection.Manager backed by the real X11 connection.
func newX11SelectionManager(conn *x11client.Connection, window uint32) (*selection.Manager, error) {
	return selection.NewManager(&x11SelectionAdapter{conn: conn}, window)
}
