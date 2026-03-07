package xdg

import (
	"testing"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// mockConn is a mock implementation of the Conn interface for testing.
type mockConn struct {
	nextID       uint32
	objects      map[uint32]interface{}
	sentRequests []mockRequest
}

type mockRequest struct {
	objectID uint32
	opcode   uint16
	args     []wire.Argument
}

func newMockConn() *mockConn {
	return &mockConn{
		nextID:       2,
		objects:      make(map[uint32]interface{}),
		sentRequests: make([]mockRequest, 0),
	}
}

func (m *mockConn) AllocID() uint32 {
	id := m.nextID
	m.nextID++
	return id
}

func (m *mockConn) RegisterObject(obj interface{}) {
	if o, ok := obj.(interface{ ID() uint32 }); ok {
		m.objects[o.ID()] = obj
	}
}

func (m *mockConn) SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error {
	m.sentRequests = append(m.sentRequests, mockRequest{
		objectID: objectID,
		opcode:   opcode,
		args:     args,
	})
	return nil
}

func (m *mockConn) lastRequest() *mockRequest {
	if len(m.sentRequests) == 0 {
		return nil
	}
	return &m.sentRequests[len(m.sentRequests)-1]
}

// TestNewWmBase verifies WmBase creation.
func TestNewWmBase(t *testing.T) {
	conn := newMockConn()
	wmBase := NewWmBase(conn, 5, 3)

	if wmBase.ID() != 5 {
		t.Errorf("expected ID 5, got %d", wmBase.ID())
	}
	if wmBase.Interface() != "xdg_wm_base" {
		t.Errorf("expected interface xdg_wm_base, got %s", wmBase.Interface())
	}
	if wmBase.version != 3 {
		t.Errorf("expected version 3, got %d", wmBase.version)
	}
}

// TestWmBaseGetXdgSurface verifies xdg_surface creation.
func TestWmBaseGetXdgSurface(t *testing.T) {
	conn := newMockConn()
	wmBase := NewWmBase(conn, 5, 3)

	surfaceID := uint32(100)
	xdgSurface, err := wmBase.GetXdgSurface(surfaceID)
	if err != nil {
		t.Fatalf("GetXdgSurface failed: %v", err)
	}

	if xdgSurface.ID() != 2 {
		t.Errorf("expected xdg_surface ID 2, got %d", xdgSurface.ID())
	}

	if xdgSurface.Interface() != "xdg_surface" {
		t.Errorf("expected interface xdg_surface, got %s", xdgSurface.Interface())
	}

	req := conn.lastRequest()
	if req == nil {
		t.Fatal("no request sent")
	}

	if req.objectID != 5 {
		t.Errorf("expected request on object 5, got %d", req.objectID)
	}

	if req.opcode != wmBaseOpcodeGetXdgSurface {
		t.Errorf("expected opcode %d, got %d", wmBaseOpcodeGetXdgSurface, req.opcode)
	}

	if len(req.args) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(req.args))
	}

	if req.args[0].Type != wire.ArgTypeNewID {
		t.Errorf("expected first arg type NewID, got %d", req.args[0].Type)
	}

	if req.args[1].Type != wire.ArgTypeObject {
		t.Errorf("expected second arg type Object, got %d", req.args[1].Type)
	}

	if req.args[1].Value.(uint32) != surfaceID {
		t.Errorf("expected surface ID %d, got %d", surfaceID, req.args[1].Value)
	}
}

// TestWmBasePong verifies pong request.
func TestWmBasePong(t *testing.T) {
	conn := newMockConn()
	wmBase := NewWmBase(conn, 5, 3)

	serial := uint32(12345)
	err := wmBase.Pong(serial)
	if err != nil {
		t.Fatalf("Pong failed: %v", err)
	}

	req := conn.lastRequest()
	if req == nil {
		t.Fatal("no request sent")
	}

	if req.objectID != 5 {
		t.Errorf("expected request on object 5, got %d", req.objectID)
	}

	if req.opcode != wmBaseOpcodePong {
		t.Errorf("expected opcode %d, got %d", wmBaseOpcodePong, req.opcode)
	}

	if len(req.args) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(req.args))
	}

	if req.args[0].Value.(uint32) != serial {
		t.Errorf("expected serial %d, got %d", serial, req.args[0].Value)
	}
}

// TestWmBaseHandlePingEvent verifies automatic pong response to ping events.
func TestWmBaseHandlePingEvent(t *testing.T) {
	conn := newMockConn()
	wmBase := NewWmBase(conn, 5, 3)

	serial := uint32(98765)
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: serial},
	}

	err := wmBase.HandleEvent(wmBaseEventPing, args)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	req := conn.lastRequest()
	if req == nil {
		t.Fatal("no pong request sent")
	}

	if req.opcode != wmBaseOpcodePong {
		t.Errorf("expected pong opcode, got %d", req.opcode)
	}

	if req.args[0].Value.(uint32) != serial {
		t.Errorf("expected pong serial %d, got %d", serial, req.args[0].Value)
	}
}

// TestSurfaceGetToplevel verifies toplevel creation.
func TestSurfaceGetToplevel(t *testing.T) {
	conn := newMockConn()
	wmBase := NewWmBase(conn, 5, 3)
	xdgSurface := &Surface{
		objectBase: objectBase{
			id:    10,
			iface: "xdg_surface",
			conn:  conn,
		},
		wmBase:        wmBase,
		configureChan: make(chan uint32, 8),
	}

	toplevel, err := xdgSurface.GetToplevel()
	if err != nil {
		t.Fatalf("GetToplevel failed: %v", err)
	}

	if toplevel.ID() != 2 {
		t.Errorf("expected toplevel ID 2, got %d", toplevel.ID())
	}

	if toplevel.Interface() != "xdg_toplevel" {
		t.Errorf("expected interface xdg_toplevel, got %s", toplevel.Interface())
	}

	req := conn.lastRequest()
	if req.objectID != 10 {
		t.Errorf("expected request on object 10, got %d", req.objectID)
	}

	if req.opcode != surfaceOpcodeGetToplevel {
		t.Errorf("expected opcode %d, got %d", surfaceOpcodeGetToplevel, req.opcode)
	}
}

// TestSurfaceSetWindowGeometry verifies window geometry setting.
func TestSurfaceSetWindowGeometry(t *testing.T) {
	conn := newMockConn()
	surface := &Surface{
		objectBase: objectBase{
			id:    10,
			iface: "xdg_surface",
			conn:  conn,
		},
	}

	err := surface.SetWindowGeometry(0, 0, 800, 600)
	if err != nil {
		t.Fatalf("SetWindowGeometry failed: %v", err)
	}

	req := conn.lastRequest()
	if len(req.args) != 4 {
		t.Fatalf("expected 4 arguments, got %d", len(req.args))
	}

	if req.args[2].Value.(int32) != 800 {
		t.Errorf("expected width 800, got %d", req.args[2].Value)
	}

	if req.args[3].Value.(int32) != 600 {
		t.Errorf("expected height 600, got %d", req.args[3].Value)
	}
}

// TestSurfaceAckConfigure verifies configure acknowledgment.
func TestSurfaceAckConfigure(t *testing.T) {
	conn := newMockConn()
	surface := &Surface{
		objectBase: objectBase{
			id:    10,
			iface: "xdg_surface",
			conn:  conn,
		},
	}

	serial := uint32(42)
	err := surface.AckConfigure(serial)
	if err != nil {
		t.Fatalf("AckConfigure failed: %v", err)
	}

	req := conn.lastRequest()
	if req.opcode != surfaceOpcodeAckConfigure {
		t.Errorf("expected opcode %d, got %d", surfaceOpcodeAckConfigure, req.opcode)
	}

	if req.args[0].Value.(uint32) != serial {
		t.Errorf("expected serial %d, got %d", serial, req.args[0].Value)
	}
}

// TestSurfaceHandleConfigureEvent verifies configure event handling.
func TestSurfaceHandleConfigureEvent(t *testing.T) {
	conn := newMockConn()
	surface := &Surface{
		objectBase: objectBase{
			id:    10,
			iface: "xdg_surface",
			conn:  conn,
		},
		configureChan: make(chan uint32, 1),
	}

	serial := uint32(99)
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: serial},
	}

	err := surface.HandleEvent(surfaceEventConfigure, args)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	select {
	case receivedSerial := <-surface.configureChan:
		if receivedSerial != serial {
			t.Errorf("expected serial %d, got %d", serial, receivedSerial)
		}
	default:
		t.Error("no serial received on configure channel")
	}
}

// TestToplevelSetTitle verifies title setting.
func TestToplevelSetTitle(t *testing.T) {
	conn := newMockConn()
	toplevel := &Toplevel{
		objectBase: objectBase{
			id:    15,
			iface: "xdg_toplevel",
			conn:  conn,
		},
	}

	title := "Test Window"
	err := toplevel.SetTitle(title)
	if err != nil {
		t.Fatalf("SetTitle failed: %v", err)
	}

	req := conn.lastRequest()
	if req.opcode != toplevelOpcodeSetTitle {
		t.Errorf("expected opcode %d, got %d", toplevelOpcodeSetTitle, req.opcode)
	}

	if req.args[0].Value.(string) != title {
		t.Errorf("expected title %s, got %s", title, req.args[0].Value)
	}
}

// TestToplevelSetAppID verifies app ID setting.
func TestToplevelSetAppID(t *testing.T) {
	conn := newMockConn()
	toplevel := &Toplevel{
		objectBase: objectBase{
			id:    15,
			iface: "xdg_toplevel",
			conn:  conn,
		},
	}

	appID := "org.example.testapp"
	err := toplevel.SetAppID(appID)
	if err != nil {
		t.Fatalf("SetAppID failed: %v", err)
	}

	req := conn.lastRequest()
	if req.opcode != toplevelOpcodeSetAppID {
		t.Errorf("expected opcode %d, got %d", toplevelOpcodeSetAppID, req.opcode)
	}

	if req.args[0].Value.(string) != appID {
		t.Errorf("expected app ID %s, got %s", appID, req.args[0].Value)
	}
}

// TestToplevelSetMinMaxSize verifies min/max size setting.
func TestToplevelSetMinMaxSize(t *testing.T) {
	conn := newMockConn()
	toplevel := &Toplevel{
		objectBase: objectBase{
			id:    15,
			iface: "xdg_toplevel",
			conn:  conn,
		},
	}

	err := toplevel.SetMinSize(320, 240)
	if err != nil {
		t.Fatalf("SetMinSize failed: %v", err)
	}

	req := conn.lastRequest()
	if req.args[0].Value.(int32) != 320 || req.args[1].Value.(int32) != 240 {
		t.Errorf("unexpected min size values")
	}

	err = toplevel.SetMaxSize(1920, 1080)
	if err != nil {
		t.Fatalf("SetMaxSize failed: %v", err)
	}

	req = conn.lastRequest()
	if req.args[0].Value.(int32) != 1920 || req.args[1].Value.(int32) != 1080 {
		t.Errorf("unexpected max size values")
	}
}

// TestToplevelStateRequests verifies state change requests.
func TestToplevelStateRequests(t *testing.T) {
	conn := newMockConn()
	toplevel := &Toplevel{
		objectBase: objectBase{
			id:    15,
			iface: "xdg_toplevel",
			conn:  conn,
		},
	}

	tests := []struct {
		name   string
		fn     func() error
		opcode uint16
	}{
		{"SetMaximized", toplevel.SetMaximized, toplevelOpcodeSetMaximized},
		{"UnsetMaximized", toplevel.UnsetMaximized, toplevelOpcodeUnsetMaximized},
		{"SetFullscreen", func() error { return toplevel.SetFullscreen(0) }, toplevelOpcodeSetFullscreen},
		{"UnsetFullscreen", toplevel.UnsetFullscreen, toplevelOpcodeUnsetFullscreen},
		{"SetMinimized", toplevel.SetMinimized, toplevelOpcodeSetMinimized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err != nil {
				t.Fatalf("%s failed: %v", tt.name, err)
			}

			req := conn.lastRequest()
			if req.opcode != tt.opcode {
				t.Errorf("expected opcode %d, got %d", tt.opcode, req.opcode)
			}
		})
	}
}

// TestToplevelHandleConfigureEvent verifies toplevel configure event handling.
func TestToplevelHandleConfigureEvent(t *testing.T) {
	conn := newMockConn()
	toplevel := &Toplevel{
		objectBase: objectBase{
			id:    15,
			iface: "xdg_toplevel",
			conn:  conn,
		},
		configureChan: make(chan *ConfigureEvent, 1),
	}

	// Prepare states array (e.g., maximized).
	statesData := make([]byte, 4)
	statesData[0] = byte(StateMaximized)
	statesData[1] = byte(StateMaximized >> 8)
	statesData[2] = byte(StateMaximized >> 16)
	statesData[3] = byte(StateMaximized >> 24)

	args := []wire.Argument{
		{Type: wire.ArgTypeInt32, Value: int32(1024)},
		{Type: wire.ArgTypeInt32, Value: int32(768)},
		{Type: wire.ArgTypeArray, Value: statesData},
	}

	err := toplevel.HandleEvent(toplevelEventConfigure, args)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	select {
	case event := <-toplevel.configureChan:
		if event.Width != 1024 {
			t.Errorf("expected width 1024, got %d", event.Width)
		}
		if event.Height != 768 {
			t.Errorf("expected height 768, got %d", event.Height)
		}
		if len(event.States) != 1 {
			t.Errorf("expected 1 state, got %d", len(event.States))
		}
		if len(event.States) > 0 && event.States[0] != StateMaximized {
			t.Errorf("expected StateMaximized, got %d", event.States[0])
		}
	default:
		t.Error("no event received on configure channel")
	}
}

// TestToplevelHandleCloseEvent verifies close event handling.
func TestToplevelHandleCloseEvent(t *testing.T) {
	conn := newMockConn()
	toplevel := &Toplevel{
		objectBase: objectBase{
			id:    15,
			iface: "xdg_toplevel",
			conn:  conn,
		},
	}

	err := toplevel.HandleEvent(toplevelEventClose, nil)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}
}
