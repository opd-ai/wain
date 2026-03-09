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
	AtomPRIMARY   = 1
	AtomSECONDARY = 2
	AtomCLIPBOARD = 69 // Standard clipboard atom number
)

// Well-known target MIME types.
const (
	TargetUTF8String = "UTF8_STRING"
	TargetString     = "STRING"
	TargetText       = "TEXT"
	TargetTextPlain  = "text/plain"
	TargetTargets    = "TARGETS"
	TargetTimestamp  = "TIMESTAMP"
)

// Conn represents the subset of X11 connection methods needed by selection handling.
type Conn interface {
	AllocXID() uint32
	SendRequest(opcode uint8, data []byte) error
	SendRequestAndReply(opcode uint8, data []byte) ([]byte, error)
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

	// ConvertSelection request
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data[0:4], m.window)
	binary.LittleEndian.PutUint32(data[4:8], selection)
	binary.LittleEndian.PutUint32(data[8:12], target)
	binary.LittleEndian.PutUint32(data[12:16], propertyAtom)
	binary.LittleEndian.PutUint32(data[16:20], uint32(time.Now().Unix()))

	if err := m.conn.SendRequest(24, data); err != nil { // Opcode 24 = ConvertSelection
		return "", fmt.Errorf("ConvertSelection failed: %w", err)
	}

	// Wait for SelectionNotify event (handled externally via event loop)
	// For now, we read the property directly (simplified approach)
	time.Sleep(50 * time.Millisecond)

	// GetProperty to read the data
	propData, _, err := m.conn.GetProperty(m.window, propertyAtom, 0, 0, 65536, true)
	if err != nil {
		return "", fmt.Errorf("GetProperty failed: %w", err)
	}

	return string(propData), nil
}

// HandleSelectionRequest processes a SelectionRequest event.
func (m *Manager) HandleSelectionRequest(requestor, selection, target, property, timestamp uint32) error {
	var data []byte
	var actualType uint32

	// Determine which selection is being requested
	if selection == m.clipboardAtom && m.ownsClipboard {
		data = m.clipboardData
		actualType = m.utf8Atom
	} else if selection == AtomPRIMARY && m.ownsPrimary {
		data = m.primaryData
		actualType = m.utf8Atom
	} else if target == m.targetsAtom {
		// Reply with supported targets
		targets := []uint32{m.utf8Atom, m.textAtom, m.targetsAtom}
		data = make([]byte, len(targets)*4)
		for i, t := range targets {
			binary.LittleEndian.PutUint32(data[i*4:], t)
		}
		actualType = 4 // ATOM type
	} else {
		// Unsupported target, send property = None
		return m.sendSelectionNotify(requestor, selection, target, 0, timestamp)
	}

	// Set the property on the requestor window
	if err := m.conn.ChangeProperty(requestor, property, actualType, 8, 0, data); err != nil {
		return fmt.Errorf("ChangeProperty failed: %w", err)
	}

	// Send SelectionNotify event
	return m.sendSelectionNotify(requestor, selection, target, property, timestamp)
}

// sendSelectionNotify sends a SelectionNotify event.
func (m *Manager) sendSelectionNotify(requestor, selection, target, property, timestamp uint32) error {
	// SelectionNotify event (type 31)
	event := make([]byte, 32)
	event[0] = 31 // SelectionNotify event code

	binary.LittleEndian.PutUint32(event[4:8], timestamp)
	binary.LittleEndian.PutUint32(event[8:12], requestor)
	binary.LittleEndian.PutUint32(event[12:16], selection)
	binary.LittleEndian.PutUint32(event[16:20], target)
	binary.LittleEndian.PutUint32(event[20:24], property)

	// SendEvent request (opcode 25)
	data := make([]byte, 44)
	data[0] = 0 // propagate = false
	binary.LittleEndian.PutUint32(data[1:5], requestor)
	binary.LittleEndian.PutUint32(data[5:9], 0) // event_mask = 0
	copy(data[9:41], event)

	return m.conn.SendRequest(25, data)
}

// HandleSelectionClear processes a SelectionClear event.
func (m *Manager) HandleSelectionClear(selection uint32) {
	if selection == m.clipboardAtom {
		m.ownsClipboard = false
		m.clipboardData = nil
	} else if selection == AtomPRIMARY {
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
