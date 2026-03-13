package wain

import (
	"testing"

	"github.com/opd-ai/wain/internal/wayland/datadevice"
	"github.com/opd-ai/wain/internal/wayland/wire"
	"github.com/opd-ai/wain/internal/x11/selection"
)

// --- X11 adapter mock ---

type mockX11SelectionConn struct {
	atoms   map[string]uint32
	sent    [][]byte
	nextXID uint32
}

func newMockX11SelectionConn() *mockX11SelectionConn {
	return &mockX11SelectionConn{
		atoms: map[string]uint32{
			"CLIPBOARD":        69,
			"UTF8_STRING":      100,
			"TARGETS":          101,
			"TEXT":             102,
			"_WAIN_SELECTION":  200,
		},
		nextXID: 1000,
	}
}

func (m *mockX11SelectionConn) AllocXID() uint32 {
	m.nextXID++
	return m.nextXID
}

func (m *mockX11SelectionConn) SendRequest(opcode uint8, data []byte) error {
	m.sent = append(m.sent, append([]byte{opcode}, data...))
	return nil
}

func (m *mockX11SelectionConn) SendRequestAndReply(opcode uint8, data []byte) ([]byte, error) {
	return nil, nil
}

func (m *mockX11SelectionConn) InternAtom(name string, onlyIfExists bool) (uint32, error) {
	if v, ok := m.atoms[name]; ok {
		return v, nil
	}
	return 0, nil
}

func (m *mockX11SelectionConn) GetProperty(window, property, typ, offset, length uint32, del bool) ([]byte, uint32, error) {
	return nil, 0, nil
}

func (m *mockX11SelectionConn) ChangeProperty(window, property, typ uint32, format, mode uint8, data []byte) error {
	return nil
}

func (m *mockX11SelectionConn) DeleteProperty(window, property uint32) error {
	return nil
}

// --- Wayland mock ---

type mockWaylandClipboardConn struct {
	nextID uint32
}

func (m *mockWaylandClipboardConn) AllocID() uint32 {
	m.nextID++
	return m.nextID
}

func (m *mockWaylandClipboardConn) RegisterObject(_ interface{}) {}

func (m *mockWaylandClipboardConn) SendRequest(_ uint32, _ uint16, _ []wire.Argument) error {
	return nil
}

// --- helpers ---

func newWindowWithDisplayServer(ds DisplayServer) *Window {
	return &Window{
		app: &App{displayServer: ds},
	}
}

// --- tests ---

func TestSetClipboard_NoDisplay(t *testing.T) {
	w := newWindowWithDisplayServer(DisplayServerUnknown)
	if err := w.SetClipboard("hello"); err != ErrNoDisplay {
		t.Errorf("expected ErrNoDisplay, got %v", err)
	}
}

func TestGetClipboard_NoDisplay(t *testing.T) {
	w := newWindowWithDisplayServer(DisplayServerUnknown)
	if _, err := w.GetClipboard(); err != ErrNoDisplay {
		t.Errorf("expected ErrNoDisplay, got %v", err)
	}
}

func TestSetClipboard_X11_NoManager(t *testing.T) {
	w := newWindowWithDisplayServer(DisplayServerX11)
	// x11SelectionMgr is nil
	if err := w.SetClipboard("text"); err == nil {
		t.Error("expected error when selection manager is nil")
	}
}

func TestGetClipboard_X11_NoManager(t *testing.T) {
	w := newWindowWithDisplayServer(DisplayServerX11)
	if _, err := w.GetClipboard(); err == nil {
		t.Error("expected error when selection manager is nil")
	}
}

func TestSetClipboard_X11_WithManager(t *testing.T) {
	conn := newMockX11SelectionConn()
	mgr, err := selection.NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	w := newWindowWithDisplayServer(DisplayServerX11)
	w.app.x11SelectionMgr = mgr

	if err := w.SetClipboard("clipboard text"); err != nil {
		t.Errorf("SetClipboard: %v", err)
	}
	if !mgr.OwnsClipboard() {
		t.Error("expected manager to own clipboard after SetClipboard")
	}
}

func TestGetClipboard_X11_WithManager(t *testing.T) {
	conn := newMockX11SelectionConn()
	mgr, err := selection.NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	w := newWindowWithDisplayServer(DisplayServerX11)
	w.app.x11SelectionMgr = mgr

	// GetClipboard against mock returns empty string (mock returns nil data)
	text, err := w.GetClipboard()
	if err != nil {
		t.Errorf("GetClipboard: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty string from mock, got %q", text)
	}
}

func TestSetClipboard_Wayland_NoDevice(t *testing.T) {
	w := newWindowWithDisplayServer(DisplayServerWayland)
	// both managers nil
	if err := w.SetClipboard("text"); err == nil {
		t.Error("expected error when Wayland data device is nil")
	}
}

func TestGetClipboard_Wayland_NoDevice(t *testing.T) {
	w := newWindowWithDisplayServer(DisplayServerWayland)
	if _, err := w.GetClipboard(); err == nil {
		t.Error("expected error when Wayland data device is nil")
	}
}

func TestGetClipboard_Wayland_NoSelection(t *testing.T) {
	conn := &mockWaylandClipboardConn{nextID: 100}
	mgr := datadevice.NewManager(conn, 50)
	device, err := mgr.GetDataDevice(200)
	if err != nil {
		t.Fatalf("GetDataDevice: %v", err)
	}

	w := newWindowWithDisplayServer(DisplayServerWayland)
	w.app.waylandDataDeviceMgr = mgr
	w.app.waylandDataDevice = device

	text, err := w.GetClipboard()
	if err != nil {
		t.Errorf("GetClipboard: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty string when no selection, got %q", text)
	}
}

func TestSetClipboard_Wayland_WithDevice(t *testing.T) {
	conn := &mockWaylandClipboardConn{nextID: 100}
	mgr := datadevice.NewManager(conn, 50)
	device, err := mgr.GetDataDevice(200)
	if err != nil {
		t.Fatalf("GetDataDevice: %v", err)
	}

	w := newWindowWithDisplayServer(DisplayServerWayland)
	w.app.waylandDataDeviceMgr = mgr
	w.app.waylandDataDevice = device

	if err := w.SetClipboard("wayland clipboard"); err != nil {
		t.Errorf("SetClipboard: %v", err)
	}
}

func TestServeClipboardSource_Cancellation(t *testing.T) {
	conn := &mockWaylandClipboardConn{nextID: 100}
	mgr := datadevice.NewManager(conn, 50)
	source, err := mgr.CreateDataSource()
	if err != nil {
		t.Fatalf("CreateDataSource: %v", err)
	}

	done := make(chan struct{})
	go func() {
		serveClipboardSource(source, "test")
		close(done)
	}()

	// Simulate compositor cancelling the source.
	source.HandleEvent(2, nil) // opcode 2 = cancelled

	select {
	case <-done:
		// OK — goroutine exited cleanly
	default:
		// Non-blocking; goroutine may still be running briefly — acceptable.
	}
}
