// Package selection implements X11 clipboard and selection support.
//
// X11 uses an asynchronous selection transfer protocol with three main atoms:
//   - CLIPBOARD: Modern clipboard (Ctrl+C/Ctrl+V)
//   - PRIMARY: Traditional X11 selection (middle-click paste)
//   - SECONDARY: Rarely used secondary selection
//
// The protocol involves:
//   - SetSelectionOwner: Claim ownership of a selection
//   - ConvertSelection: Request data from the selection owner
//   - SelectionNotify: Reply with requested data
//   - SelectionClear: Notification that ownership was lost
//
// Reference: https://www.x.org/releases/current/doc/xproto/x11protocol.html
package selection

import (
	"encoding/binary"
	"fmt"
	"time"
)

// Atom constants for common selections and targets.
const (
	// AtomPRIMARY is the X11 PRIMARY selection atom.
	AtomPRIMARY = 1
	// AtomSECONDARY is the X11 SECONDARY selection atom.
	AtomSECONDARY = 2
	// AtomCLIPBOARD is the X11 CLIPBOARD selection atom.
	AtomCLIPBOARD = 69
)

// Well-known target MIME types.
const (
	// TargetUTF8String is the UTF-8 encoded string target.
	TargetUTF8String = "UTF8_STRING"
	// TargetString is the Latin-1 encoded string target.
	TargetString = "STRING"
	// TargetText is a generic text target.
	TargetText = "TEXT"
	// TargetTextPlain is the text/plain MIME type target.
	TargetTextPlain = "text/plain"
	// TargetTargets requests the list of available targets.
	TargetTargets = "TARGETS"
	// TargetTimestamp requests the selection timestamp.
	TargetTimestamp = "TIMESTAMP"
)

// Conn represents the subset of X11 connection methods needed by selection handling.
type Conn interface {
	AllocXID() uint32
	SendRequest(opcode uint8, data []byte) error
	SendRequestAndReply(opcode uint8, data []byte) ([]byte, error)
	SendEvent(destination uint32, propagate bool, eventMask uint32, event []byte) error
	InternAtom(name string, onlyIfExists bool) (uint32, error)
	GetProperty(window, property, typ, offset, length uint32, delete bool) ([]byte, uint32, error)
	ChangeProperty(window, property, typ uint32, format, mode uint8, data []byte) error
	DeleteProperty(window, property uint32) error
}

// Manager handles X11 selections (clipboard, primary, etc.).
type Manager struct {
	conn          Conn
	window        uint32
	clipboardAtom uint32
	utf8Atom      uint32
	targetsAtom   uint32
	textAtom      uint32
	clipboardData []byte
	primaryData   []byte
	ownsClipboard bool
	ownsPrimary   bool
	// selNotify is signalled by HandleSelectionNotify when a SelectionNotify
	// event arrives for our pending ConvertSelection request.
	selNotify chan uint32 // receives the property atom from the event
}

// NewManager creates a selection manager for the given window.
func NewManager(conn Conn, window uint32) (*Manager, error) {
	// Intern required atoms
	clipboardAtom, err := conn.InternAtom("CLIPBOARD", false)
	if err != nil {
		return nil, fmt.Errorf("failed to intern CLIPBOARD: %w", err)
	}

	utf8Atom, err := conn.InternAtom("UTF8_STRING", false)
	if err != nil {
		return nil, fmt.Errorf("failed to intern UTF8_STRING: %w", err)
	}

	targetsAtom, err := conn.InternAtom("TARGETS", false)
	if err != nil {
		return nil, fmt.Errorf("failed to intern TARGETS: %w", err)
	}

	textAtom, err := conn.InternAtom("TEXT", false)
	if err != nil {
		return nil, fmt.Errorf("failed to intern TEXT: %w", err)
	}

	return &Manager{
		conn:          conn,
		window:        window,
		clipboardAtom: clipboardAtom,
		utf8Atom:      utf8Atom,
		targetsAtom:   targetsAtom,
		textAtom:      textAtom,
		selNotify:     make(chan uint32, 1),
	}, nil
}

// SetClipboard sets the clipboard content to the given text.
func (m *Manager) SetClipboard(text string) error {
	m.clipboardData = []byte(text)
	m.ownsClipboard = true

	// SetSelectionOwner request
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:4], m.window)
	binary.LittleEndian.PutUint32(data[4:8], m.clipboardAtom)
	binary.LittleEndian.PutUint32(data[8:12], uint32(time.Now().Unix()))

	return m.conn.SendRequest(22, data) // Opcode 22 = SetSelectionOwner
}

// SetPrimary sets the PRIMARY selection content to the given text.
func (m *Manager) SetPrimary(text string) error {
	m.primaryData = []byte(text)
	m.ownsPrimary = true

	// SetSelectionOwner request
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:4], m.window)
	binary.LittleEndian.PutUint32(data[4:8], AtomPRIMARY)
	binary.LittleEndian.PutUint32(data[8:12], uint32(time.Now().Unix()))

	return m.conn.SendRequest(22, data)
}

// GetClipboard retrieves the current clipboard content.
func (m *Manager) GetClipboard() (string, error) {
	return m.getSelection(m.clipboardAtom, m.utf8Atom)
}

// GetPrimary retrieves the current PRIMARY selection content.
func (m *Manager) GetPrimary() (string, error) {
	return m.getSelection(AtomPRIMARY, m.utf8Atom)
}

// getSelection requests data from a selection.
func (m *Manager) getSelection(selection, target uint32) (string, error) {
	// Create a temporary property atom
	propertyAtom, err := m.conn.InternAtom("_WAIN_SELECTION", false)
	if err != nil {
		return "", fmt.Errorf("failed to intern property atom: %w", err)
	}

	// Drain any stale notifications from previous requests.
	select {
	case <-m.selNotify:
	default:
	}

	// ConvertSelection request (opcode 24)
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data[0:4], m.window)
	binary.LittleEndian.PutUint32(data[4:8], selection)
	binary.LittleEndian.PutUint32(data[8:12], target)
	binary.LittleEndian.PutUint32(data[12:16], propertyAtom)
	binary.LittleEndian.PutUint32(data[16:20], uint32(time.Now().Unix()))

	if err := m.conn.SendRequest(24, data); err != nil {
		return "", fmt.Errorf("ConvertSelection failed: %w", err)
	}

	// Block until the event loop delivers a SelectionNotify event for our
	// request, or fall back after a generous timeout to handle compositors
	// that are slow or non-responsive.
	const timeout = 2 * time.Second
	select {
	case prop := <-m.selNotify:
		if prop == 0 {
			// property == None means the selection owner refused the conversion.
			return "", nil
		}
		propertyAtom = prop
	case <-time.After(timeout):
		// Timeout: the selection owner did not respond; return empty.
		return "", nil
	}

	// GetProperty to read the data
	propData, _, err := m.conn.GetProperty(m.window, propertyAtom, 0, 0, 65536, true)
	if err != nil {
		return "", fmt.Errorf("GetProperty failed: %w", err)
	}

	return string(propData), nil
}

// HandleSelectionNotify is called by the event loop when a SelectionNotify
// event arrives. It unblocks any pending getSelection call.
func (m *Manager) HandleSelectionNotify(property uint32) {
	select {
	case m.selNotify <- property:
	default:
		// No pending request; discard.
	}
}

// HandleSelectionRequest processes a SelectionRequest event.
func (m *Manager) HandleSelectionRequest(requestor, selection, target, property, timestamp uint32) error {
	if property == 0 {
		return m.sendSelectionNotify(requestor, selection, target, 0, timestamp)
	}

	data, actualType, ok := m.resolveSelectionData(selection, target)
	if !ok {
		return m.sendSelectionNotify(requestor, selection, target, 0, timestamp)
	}

	format := uint8(8)
	if actualType == 4 { // XA_ATOM requires format=32
		format = 32
	}
	if err := m.conn.ChangeProperty(requestor, property, actualType, format, 0, data); err != nil {
		return fmt.Errorf("ChangeProperty failed: %w", err)
	}

	return m.sendSelectionNotify(requestor, selection, target, property, timestamp)
}

// resolveSelectionData determines the byte payload and X11 type atom for a
// selection request. It returns ok=false when the requested target is
// unsupported, signalling that a "property = None" notify should be sent.
func (m *Manager) resolveSelectionData(selection, target uint32) (data []byte, actualType uint32, ok bool) {
	// TARGETS is a meta-request: respond with supported conversion types when we
	// own the requested selection.
	if target == m.targetsAtom {
		ownsRequested := (selection == m.clipboardAtom && m.ownsClipboard) ||
			(selection == AtomPRIMARY && m.ownsPrimary)
		if !ownsRequested {
			return nil, 0, false
		}
		targets := []uint32{m.utf8Atom, m.textAtom, m.targetsAtom}
		buf := make([]byte, len(targets)*4)
		for i, t := range targets {
			binary.LittleEndian.PutUint32(buf[i*4:], t)
		}
		return buf, 4, true // 4 = XA_ATOM
	}
	if selection == m.clipboardAtom && m.ownsClipboard {
		return m.clipboardData, m.utf8Atom, true
	}
	if selection == AtomPRIMARY && m.ownsPrimary {
		return m.primaryData, m.utf8Atom, true
	}
	return nil, 0, false
}

// sendSelectionNotify sends a SelectionNotify event.
func (m *Manager) sendSelectionNotify(requestor, selection, target, property, timestamp uint32) error {
	// Build a 32-byte SelectionNotify event body (type 31).
	// Bytes 2-3 (sequence number) are left as zero; the X server ignores them on
	// synthetic events.
	event := make([]byte, 32)
	event[0] = 31 // SelectionNotify event code
	binary.LittleEndian.PutUint32(event[4:8], timestamp)
	binary.LittleEndian.PutUint32(event[8:12], requestor)
	binary.LittleEndian.PutUint32(event[12:16], selection)
	binary.LittleEndian.PutUint32(event[16:20], target)
	binary.LittleEndian.PutUint32(event[20:24], property)

	// propagate=false, event-mask=0 (deliver directly to requestor window)
	return m.conn.SendEvent(requestor, false, 0, event)
}

// HandleSelectionClear processes a SelectionClear event.
func (m *Manager) HandleSelectionClear(selection uint32) {
	switch selection {
	case m.clipboardAtom:
		m.ownsClipboard = false
		m.clipboardData = nil
	case AtomPRIMARY:
		m.ownsPrimary = false
		m.primaryData = nil
	}
}

// OwnsClipboard returns true if we currently own the clipboard.
func (m *Manager) OwnsClipboard() bool {
	return m.ownsClipboard
}

// OwnsPrimary returns true if we currently own the PRIMARY selection.
func (m *Manager) OwnsPrimary() bool {
	return m.ownsPrimary
}
