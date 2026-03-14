package dmabuf

import (
	"testing"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// mockConn implements the Conn interface for testing.
type mockConn struct {
	nextID        uint32
	registeredObj interface{}
	lastRequest   struct {
		objectID uint32
		opcode   uint16
		args     []wire.Argument
	}
}

func (m *mockConn) AllocID() uint32 {
	m.nextID++
	return m.nextID
}

func (m *mockConn) RegisterObject(obj interface{}) {
	m.registeredObj = obj
}

func (m *mockConn) SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error {
	m.lastRequest.objectID = objectID
	m.lastRequest.opcode = opcode
	m.lastRequest.args = args
	return nil
}

func TestNewDmabuf(t *testing.T) {
	conn := &mockConn{nextID: 100}
	dmabuf := NewDmabuf(conn, 42)

	if dmabuf.ID() != 42 {
		t.Errorf("expected ID 42, got %d", dmabuf.ID())
	}

	if dmabuf.Interface() != "zwp_linux_dmabuf_v1" {
		t.Errorf("expected interface zwp_linux_dmabuf_v1, got %s", dmabuf.Interface())
	}

	if dmabuf.formats == nil {
		t.Error("formats map should be initialized")
	}
}

func TestDmabufHandleFormatEvent(t *testing.T) {
	conn := &mockConn{}
	dmabuf := NewDmabuf(conn, 1)

	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: uint32(FormatARGB8888)},
	}

	if err := dmabuf.HandleEvent(0, args); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	if !dmabuf.HasFormat(FormatARGB8888) {
		t.Error("format should be registered")
	}

	if !dmabuf.HasFormatModifier(FormatARGB8888, ModifierLinear) {
		t.Error("linear modifier should be added for format events")
	}
}

func TestDmabufHandleModifierEvent(t *testing.T) {
	conn := &mockConn{}
	dmabuf := NewDmabuf(conn, 1)

	// Modifier event with format ARGB8888 and modifier 0x0100000000000001
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: uint32(FormatARGB8888)},
		{Type: wire.ArgTypeUint32, Value: uint32(0x01000000)}, // high 32 bits
		{Type: wire.ArgTypeUint32, Value: uint32(0x00000001)}, // low 32 bits
	}

	if err := dmabuf.HandleEvent(1, args); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	expectedModifier := uint64(0x0100000000000001)
	if !dmabuf.HasFormatModifier(FormatARGB8888, expectedModifier) {
		t.Errorf("modifier %#x should be registered for format %#x", expectedModifier, FormatARGB8888)
	}
}

func TestDmabufCreateParams(t *testing.T) {
	conn := &mockConn{nextID: 100}
	dmabuf := NewDmabuf(conn, 42)

	params, err := dmabuf.CreateParams()
	if err != nil {
		t.Fatalf("CreateParams failed: %v", err)
	}

	if params == nil {
		t.Fatal("params should not be nil")
	}

	if params.ID() != 101 {
		t.Errorf("expected params ID 101, got %d", params.ID())
	}

	if params.Interface() != "zwp_linux_buffer_params_v1" {
		t.Errorf("expected interface zwp_linux_buffer_params_v1, got %s", params.Interface())
	}

	// Verify request was sent
	if conn.lastRequest.objectID != 42 {
		t.Errorf("expected request to object 42, got %d", conn.lastRequest.objectID)
	}

	if conn.lastRequest.opcode != 2 {
		t.Errorf("expected opcode 2, got %d", conn.lastRequest.opcode)
	}

	// Verify params was registered
	if conn.registeredObj != params {
		t.Error("params should be registered with connection")
	}
}

func TestBufferParamsAdd(t *testing.T) {
	conn := &mockConn{nextID: 100}
	params := &BufferParams{
		objectBase: objectBase{
			id:    50,
			iface: "zwp_linux_buffer_params_v1",
			conn:  conn,
		},
	}

	err := params.Add(123, 0, 0, 7680, 0, 0)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Verify request
	if conn.lastRequest.objectID != 50 {
		t.Errorf("expected request to object 50, got %d", conn.lastRequest.objectID)
	}

	if conn.lastRequest.opcode != 0 {
		t.Errorf("expected opcode 0, got %d", conn.lastRequest.opcode)
	}

	if len(conn.lastRequest.args) != 6 {
		t.Fatalf("expected 6 arguments, got %d", len(conn.lastRequest.args))
	}

	// Verify fd argument
	if conn.lastRequest.args[0].Type != wire.ArgTypeFD {
		t.Errorf("expected ArgTypeFD, got %d", conn.lastRequest.args[0].Type)
	}
	if conn.lastRequest.args[0].Value.(int32) != 123 {
		t.Errorf("expected fd 123, got %v", conn.lastRequest.args[0].Value)
	}

	// Verify stride argument
	if conn.lastRequest.args[3].Value.(uint32) != 7680 {
		t.Errorf("expected stride 7680, got %v", conn.lastRequest.args[3].Value)
	}
}

func TestBufferParamsCreate(t *testing.T) {
	conn := &mockConn{nextID: 200}
	params := &BufferParams{
		objectBase: objectBase{
			id:    50,
			iface: "zwp_linux_buffer_params_v1",
			conn:  conn,
		},
	}

	bufferID, err := params.Create(1920, 1080, FormatARGB8888, 0)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if bufferID != 201 {
		t.Errorf("expected buffer ID 201, got %d", bufferID)
	}

	// Verify request
	if conn.lastRequest.opcode != 1 {
		t.Errorf("expected opcode 1 (create), got %d", conn.lastRequest.opcode)
	}

	if len(conn.lastRequest.args) != 5 {
		t.Fatalf("expected 5 arguments, got %d", len(conn.lastRequest.args))
	}

	// Verify dimensions
	if conn.lastRequest.args[1].Value.(int32) != 1920 {
		t.Errorf("expected width 1920, got %v", conn.lastRequest.args[1].Value)
	}
	if conn.lastRequest.args[2].Value.(int32) != 1080 {
		t.Errorf("expected height 1080, got %v", conn.lastRequest.args[2].Value)
	}

	// Verify format
	if conn.lastRequest.args[3].Value.(uint32) != FormatARGB8888 {
		t.Errorf("expected format ARGB8888, got %v", conn.lastRequest.args[3].Value)
	}
}

func TestBufferParamsHandleEvent(t *testing.T) {
	conn := &mockConn{}
	params := &BufferParams{
		objectBase: objectBase{
			id:    50,
			iface: "zwp_linux_buffer_params_v1",
			conn:  conn,
		},
	}

	// Test created event
	if err := params.HandleEvent(0, nil); err != nil {
		t.Errorf("created event should succeed: %v", err)
	}

	// Test failed event
	if err := params.HandleEvent(1, nil); err == nil {
		t.Error("failed event should return error")
	}

	// Test unknown event
	if err := params.HandleEvent(99, nil); err == nil {
		t.Error("unknown event should return error")
	}
}

func TestHasFormat(t *testing.T) {
	conn := &mockConn{}
	dmabuf := NewDmabuf(conn, 1)

	// Initially no formats
	if dmabuf.HasFormat(FormatARGB8888) {
		t.Error("should not have format before event")
	}

	// Add format via event
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: uint32(FormatARGB8888)},
	}
	if err := dmabuf.HandleEvent(0, args); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	if !dmabuf.HasFormat(FormatARGB8888) {
		t.Error("should have format after event")
	}

	// Different format should not exist
	if dmabuf.HasFormat(FormatXRGB8888) {
		t.Error("should not have unreported format")
	}
}

func TestFormatConstants(t *testing.T) {
	// Verify DRM fourcc format codes are correct
	// These are defined as little-endian fourcc codes

	tests := []struct {
		format   uint32
		expected string
	}{
		{FormatARGB8888, "AR24"},
		{FormatXRGB8888, "XR24"},
		{FormatABGR8888, "AB24"},
		{FormatXBGR8888, "XB24"},
	}

	for _, tt := range tests {
		// Convert fourcc to string
		fourcc := string([]byte{
			byte(tt.format & 0xFF),
			byte((tt.format >> 8) & 0xFF),
			byte((tt.format >> 16) & 0xFF),
			byte((tt.format >> 24) & 0xFF),
		})

		if fourcc != tt.expected {
			t.Errorf("format %#x: expected fourcc %q, got %q", tt.format, tt.expected, fourcc)
		}
	}
}
