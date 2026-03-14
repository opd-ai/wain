// Package dnd implements the XDND drag-and-drop protocol for X11.
//
// This package provides source-side and target-side implementations of the
// XDND (X Drag and Drop) protocol version 5 as described at:
// https://www.freedesktop.org/wiki/Specifications/XDND/
//
// # Protocol Overview
//
// XDND is built on top of X11 ClientMessage events. The sequence for a
// successful drag-and-drop is:
//
//  1. Source sets the XdndAware property on its window.
//  2. Source sends XdndEnter to the target when dragging begins.
//  3. Source sends XdndPosition messages as the pointer moves.
//  4. Target replies with XdndStatus to accept or reject the drop.
//  5. Source sends XdndDrop when the user releases the mouse.
//  6. Target transfers the data via ICCCM selection, then sends XdndFinished.
//
// Or if the drag leaves without a drop:
//
//  1. Source sends XdndLeave to the target.
//
// # Conn interface
//
// The Conn interface abstracts the X11 client connection so the package can be
// tested without a live X server.
package dnd

import (
	"encoding/binary"
	"fmt"
)

// Conn is the subset of x11/client.Connection required by this package.
type Conn interface {
	// InternAtom returns the atom ID for name.
	InternAtom(name string, onlyIfExists bool) (uint32, error)
	// SendEvent sends a 32-byte ClientMessage to destination.
	SendEvent(destination uint32, propagate bool, eventMask uint32, event []byte) error
	// ChangeProperty sets a window property.
	ChangeProperty(window, property, propType uint32, format uint8, data []byte) error
}

// XDNDVersion is the XDND protocol version supported by this implementation.
const XDNDVersion = 5

// Atoms holds the interned X11 atoms needed for XDND.
type Atoms struct {
	XdndAware    uint32
	XdndEnter    uint32
	XdndLeave    uint32
	XdndPosition uint32
	XdndStatus   uint32
	XdndDrop     uint32
	XdndFinished uint32
	XdndActionCopy uint32
}

// InternAtoms interns all XDND atoms against the given connection.
func InternAtoms(conn Conn) (*Atoms, error) {
	names := []string{
		"XdndAware", "XdndEnter", "XdndLeave", "XdndPosition",
		"XdndStatus", "XdndDrop", "XdndFinished", "XdndActionCopy",
	}
	ids := make([]uint32, len(names))
	for i, name := range names {
		id, err := conn.InternAtom(name, false)
		if err != nil {
			return nil, fmt.Errorf("intern %s: %w", name, err)
		}
		ids[i] = id
	}
	return &Atoms{
		XdndAware:      ids[0],
		XdndEnter:      ids[1],
		XdndLeave:      ids[2],
		XdndPosition:   ids[3],
		XdndStatus:     ids[4],
		XdndDrop:       ids[5],
		XdndFinished:   ids[6],
		XdndActionCopy: ids[7],
	}, nil
}

// Manager handles XDND interactions on behalf of a single X11 window.
type Manager struct {
	conn   Conn
	window uint32
	atoms  *Atoms
}

// New creates a Manager for window wid.
func New(conn Conn, window uint32, atoms *Atoms) *Manager {
	return &Manager{conn: conn, window: window, atoms: atoms}
}

// AdvertiseAware sets the XdndAware property on the window, announcing XDND
// version support to potential drag sources.
func (m *Manager) AdvertiseAware() error {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, XDNDVersion)
	return m.conn.ChangeProperty(m.window, m.atoms.XdndAware, 6 /* XA_CARDINAL */, 32, data)
}

// SendEnter sends XdndEnter to the target window, announcing the start of a
// drag originating from m.window.
//
// mimeTypes may contain up to 3 MIME-type atom IDs; additional types are
// negotiated via the XdndTypeList property (not yet implemented here).
func (m *Manager) SendEnter(target uint32, mimeTypes []uint32) error {
	msg := m.newClientMessage(target, m.atoms.XdndEnter)
	// data[1] bits: high 4 bits = version, bit 0 = more than 3 types
	flags := uint32(XDNDVersion) << 24
	if len(mimeTypes) > 3 {
		flags |= 1
	}
	binary.LittleEndian.PutUint32(msg[16:20], flags)
	for i := 0; i < 3 && i < len(mimeTypes); i++ {
		binary.LittleEndian.PutUint32(msg[20+i*4:], mimeTypes[i])
	}
	return m.conn.SendEvent(target, false, 0, msg)
}

// SendPosition sends XdndPosition to the target window during a drag.
// x and y are root-window coordinates. action is the proposed drop action atom.
func (m *Manager) SendPosition(target uint32, x, y int, action uint32) error {
	msg := m.newClientMessage(target, m.atoms.XdndPosition)
	xy := (uint32(x) << 16) | (uint32(y) & 0xFFFF)
	binary.LittleEndian.PutUint32(msg[16:20], xy)
	binary.LittleEndian.PutUint32(msg[20:24], action)
	return m.conn.SendEvent(target, false, 0, msg)
}

// SendLeave sends XdndLeave to the target window, cancelling the drag.
func (m *Manager) SendLeave(target uint32) error {
	msg := m.newClientMessage(target, m.atoms.XdndLeave)
	return m.conn.SendEvent(target, false, 0, msg)
}

// SendDrop sends XdndDrop to the target window, completing the drag.
// timestamp is the X11 server timestamp at the time of the button release.
func (m *Manager) SendDrop(target, timestamp uint32) error {
	msg := m.newClientMessage(target, m.atoms.XdndDrop)
	binary.LittleEndian.PutUint32(msg[16:20], timestamp)
	return m.conn.SendEvent(target, false, 0, msg)
}

// SendStatus sends XdndStatus from the target back to the source, indicating
// whether the target accepts the drop.
//
// accepted indicates whether the MIME type is acceptable. action is the drop
// action atom the target will perform (use 0 if not accepted).
func (m *Manager) SendStatus(source uint32, accepted bool, action uint32) error {
	msg := m.newClientMessage(source, m.atoms.XdndStatus)
	// data[1] bit 1 = accept; bit 0 = want position events
	var flags uint32
	if accepted {
		flags = 2
	}
	binary.LittleEndian.PutUint32(msg[16:20], flags)
	binary.LittleEndian.PutUint32(msg[24:28], action)
	return m.conn.SendEvent(source, false, 0, msg)
}

// SendFinished sends XdndFinished from the target to the source after the data
// has been transferred.
func (m *Manager) SendFinished(source uint32, success bool, action uint32) error {
	msg := m.newClientMessage(source, m.atoms.XdndFinished)
	var flags uint32
	if success {
		flags = 1
	}
	binary.LittleEndian.PutUint32(msg[16:20], flags)
	binary.LittleEndian.PutUint32(msg[20:24], action)
	return m.conn.SendEvent(source, false, 0, msg)
}

// ClientMessageEvent carries the parsed fields of an XDND client message.
type ClientMessageEvent struct {
	// MessageType is the atom identifying the XDND message kind
	// (e.g. XdndEnter, XdndPosition, etc.)
	MessageType uint32
	// Source is the window that sent the message.
	Source uint32
	// Data is the raw 20-byte data payload of the ClientMessage.
	Data [20]byte
}

// ParseClientMessage parses the 32-byte X11 ClientMessage event into an
// ClientMessageEvent. Returns an error if the event is not a ClientMessage (type 33).
func ParseClientMessage(event []byte) (*ClientMessageEvent, error) {
	if len(event) < 32 {
		return nil, fmt.Errorf("dnd: event too short (%d bytes)", len(event))
	}
	if event[0] != 33 {
		return nil, fmt.Errorf("dnd: not a ClientMessage (type %d)", event[0])
	}
	msgType := binary.LittleEndian.Uint32(event[8:12])
	source := binary.LittleEndian.Uint32(event[12:16])
	e := &ClientMessageEvent{
		MessageType: msgType,
		Source:      source,
	}
	copy(e.Data[:], event[12:32])
	return e, nil
}

// newClientMessage builds a zeroed 32-byte ClientMessage buffer with the
// target window, message-type, and source window (data[0]) fields pre-filled.
func (m *Manager) newClientMessage(target, messageType uint32) []byte {
	msg := make([]byte, 32)
	msg[0] = 33 // ClientMessage
	msg[1] = 32 // format = 32-bit
	binary.LittleEndian.PutUint32(msg[4:8], target)
	binary.LittleEndian.PutUint32(msg[8:12], messageType)
	binary.LittleEndian.PutUint32(msg[12:16], m.window) // data[0]: source window
	return msg
}
