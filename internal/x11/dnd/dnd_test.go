package dnd

import (
	"encoding/binary"
	"testing"
)

// mockConn implements Conn for testing without a live X server.
type mockConn struct {
	atoms    map[string]uint32
	nextAtom uint32
	sent     []sentEvent
	props    []setProperty
}

type sentEvent struct {
	destination uint32
	event       []byte
}

type setProperty struct {
	window   uint32
	property uint32
	data     []byte
}

func newMockConn() *mockConn {
	return &mockConn{atoms: make(map[string]uint32), nextAtom: 100}
}

func (m *mockConn) InternAtom(name string, _ bool) (uint32, error) {
	if id, ok := m.atoms[name]; ok {
		return id, nil
	}
	m.atoms[name] = m.nextAtom
	m.nextAtom++
	return m.atoms[name], nil
}

func (m *mockConn) SendEvent(destination uint32, _ bool, _ uint32, event []byte) error {
	cp := make([]byte, len(event))
	copy(cp, event)
	m.sent = append(m.sent, sentEvent{destination: destination, event: cp})
	return nil
}

func (m *mockConn) ChangeProperty(window, property, _ uint32, _ uint8, data []byte) error {
	cp := make([]byte, len(data))
	copy(cp, data)
	m.props = append(m.props, setProperty{window: window, property: property, data: cp})
	return nil
}

func (m *mockConn) lastSent() sentEvent {
	if len(m.sent) == 0 {
		return sentEvent{}
	}
	return m.sent[len(m.sent)-1]
}

func setupManager(t *testing.T) (*Manager, *mockConn, *Atoms) {
	t.Helper()
	conn := newMockConn()
	atoms, err := InternAtoms(conn)
	if err != nil {
		t.Fatalf("InternAtoms: %v", err)
	}
	mgr := New(conn, 42, atoms)
	return mgr, conn, atoms
}

func TestInternAtoms(t *testing.T) {
	conn := newMockConn()
	atoms, err := InternAtoms(conn)
	if err != nil {
		t.Fatalf("InternAtoms error: %v", err)
	}
	if atoms.XdndAware == 0 {
		t.Error("XdndAware atom is 0")
	}
	if atoms.XdndEnter == atoms.XdndLeave {
		t.Error("XdndEnter and XdndLeave must be distinct atoms")
	}
	if atoms.XdndDrop == atoms.XdndFinished {
		t.Error("XdndDrop and XdndFinished must be distinct atoms")
	}
}

func TestAdvertiseAware(t *testing.T) {
	mgr, conn, atoms := setupManager(t)
	if err := mgr.AdvertiseAware(); err != nil {
		t.Fatalf("AdvertiseAware: %v", err)
	}
	if len(conn.props) != 1 {
		t.Fatalf("expected 1 property set, got %d", len(conn.props))
	}
	prop := conn.props[0]
	if prop.window != 42 {
		t.Errorf("property set on window %d, want 42", prop.window)
	}
	if prop.property != atoms.XdndAware {
		t.Errorf("property atom %d, want XdndAware=%d", prop.property, atoms.XdndAware)
	}
	ver := binary.LittleEndian.Uint32(prop.data)
	if ver != XDNDVersion {
		t.Errorf("XdndAware version %d, want %d", ver, XDNDVersion)
	}
}

func TestSendEnter(t *testing.T) {
	mgr, conn, atoms := setupManager(t)
	mimeTypes := []uint32{200, 201, 202}
	if err := mgr.SendEnter(99, mimeTypes); err != nil {
		t.Fatalf("SendEnter: %v", err)
	}
	if len(conn.sent) != 1 {
		t.Fatalf("expected 1 event sent, got %d", len(conn.sent))
	}
	ev := conn.lastSent()
	if ev.destination != 99 {
		t.Errorf("sent to window %d, want 99", ev.destination)
	}
	if ev.event[0] != 33 {
		t.Errorf("event type %d, want 33 (ClientMessage)", ev.event[0])
	}
	msgType := binary.LittleEndian.Uint32(ev.event[8:12])
	if msgType != atoms.XdndEnter {
		t.Errorf("message type atom %d, want XdndEnter=%d", msgType, atoms.XdndEnter)
	}
	// data[0] at bytes 12-15 must be the source window (42).
	src := binary.LittleEndian.Uint32(ev.event[12:16])
	if src != 42 {
		t.Errorf("source window %d, want 42", src)
	}
}

func TestSendLeave(t *testing.T) {
	mgr, conn, atoms := setupManager(t)
	if err := mgr.SendLeave(77); err != nil {
		t.Fatalf("SendLeave: %v", err)
	}
	ev := conn.lastSent()
	msgType := binary.LittleEndian.Uint32(ev.event[8:12])
	if msgType != atoms.XdndLeave {
		t.Errorf("SendLeave message type %d, want XdndLeave=%d", msgType, atoms.XdndLeave)
	}
}

func TestSendStatus(t *testing.T) {
	mgr, conn, atoms := setupManager(t)
	if err := mgr.SendStatus(55, true, atoms.XdndActionCopy); err != nil {
		t.Fatalf("SendStatus: %v", err)
	}
	ev := conn.lastSent()
	msgType := binary.LittleEndian.Uint32(ev.event[8:12])
	if msgType != atoms.XdndStatus {
		t.Errorf("SendStatus message type %d, want XdndStatus=%d", msgType, atoms.XdndStatus)
	}
	// flags at data[1] = bytes 16-19; accepted=true → flags=2
	flags := binary.LittleEndian.Uint32(ev.event[16:20])
	if flags != 2 {
		t.Errorf("SendStatus flags %d, want 2 (accepted)", flags)
	}
}

func TestSendDrop(t *testing.T) {
	mgr, conn, atoms := setupManager(t)
	if err := mgr.SendDrop(33, 12345); err != nil {
		t.Fatalf("SendDrop: %v", err)
	}
	ev := conn.lastSent()
	msgType := binary.LittleEndian.Uint32(ev.event[8:12])
	if msgType != atoms.XdndDrop {
		t.Errorf("SendDrop message type %d, want XdndDrop=%d", msgType, atoms.XdndDrop)
	}
	// timestamp at data[1] = bytes 16-19
	ts := binary.LittleEndian.Uint32(ev.event[16:20])
	if ts != 12345 {
		t.Errorf("SendDrop timestamp %d, want 12345", ts)
	}
}

func TestSendFinished(t *testing.T) {
	mgr, conn, atoms := setupManager(t)
	if err := mgr.SendFinished(33, true, atoms.XdndActionCopy); err != nil {
		t.Fatalf("SendFinished: %v", err)
	}
	ev := conn.lastSent()
	msgType := binary.LittleEndian.Uint32(ev.event[8:12])
	if msgType != atoms.XdndFinished {
		t.Errorf("SendFinished message type %d, want XdndFinished=%d", msgType, atoms.XdndFinished)
	}
	// flags at data[1] = bytes 16-19; success=true → flags=1
	flags := binary.LittleEndian.Uint32(ev.event[16:20])
	if flags != 1 {
		t.Errorf("SendFinished flags %d, want 1 (success)", flags)
	}
}

func TestParseClientMessage(t *testing.T) {
	mgr, _, atoms := setupManager(t)
	_ = mgr.SendEnter(99, []uint32{200})
	// Build a synthetic 32-byte ClientMessage manually.
	raw := make([]byte, 32)
	raw[0] = 33
	raw[1] = 32
	binary.LittleEndian.PutUint32(raw[4:8], 999) // window
	binary.LittleEndian.PutUint32(raw[8:12], atoms.XdndPosition)
	binary.LittleEndian.PutUint32(raw[12:16], 42) // source window

	parsed, err := ParseClientMessage(raw)
	if err != nil {
		t.Fatalf("ParseClientMessage: %v", err)
	}
	if parsed.MessageType != atoms.XdndPosition {
		t.Errorf("MessageType %d, want XdndPosition=%d", parsed.MessageType, atoms.XdndPosition)
	}
	if parsed.Source != 42 {
		t.Errorf("Source %d, want 42", parsed.Source)
	}
}

func TestParseClientMessageErrors(t *testing.T) {
	// Too short.
	_, err := ParseClientMessage(make([]byte, 10))
	if err == nil {
		t.Error("expected error for short event")
	}

	// Wrong type byte.
	ev := make([]byte, 32)
	ev[0] = 5 // EnterNotify, not ClientMessage
	_, err = ParseClientMessage(ev)
	if err == nil {
		t.Error("expected error for non-ClientMessage event type")
	}
}
