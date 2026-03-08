package output

import (
	"testing"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// mockConnection implements the Connection interface for testing.
type mockConnection struct {
	requests []mockRequest
}

type mockRequest struct {
	objectID uint32
	opcode   uint16
	args     []wire.Argument
}

func (m *mockConnection) sendRequest(objectID uint32, opcode uint16, args []wire.Argument) error {
	m.requests = append(m.requests, mockRequest{objectID, opcode, args})
	return nil
}

func TestNew(t *testing.T) {
	conn := &mockConnection{}
	output := New(10, conn, 3)

	if output.ID() != 10 {
		t.Errorf("expected ID 10, got %d", output.ID())
	}
	if output.Interface() != "wl_output" {
		t.Errorf("expected interface wl_output, got %s", output.Interface())
	}
	if output.Scale() != 1 {
		t.Errorf("expected default scale 1, got %d", output.Scale())
	}
}

func TestHandleScaleEvent(t *testing.T) {
	conn := &mockConnection{}
	output := New(10, conn, 3)

	args := []wire.Argument{
		{Type: wire.ArgTypeInt32, Value: int32(2)},
	}

	err := output.HandleEvent(outputEventScale, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Scale() != 2 {
		t.Errorf("expected scale 2, got %d", output.Scale())
	}
}

func TestHandleGeometryEvent(t *testing.T) {
	conn := &mockConnection{}
	output := New(10, conn, 3)

	args := []wire.Argument{
		{Type: wire.ArgTypeInt32, Value: int32(0)},      // x
		{Type: wire.ArgTypeInt32, Value: int32(0)},      // y
		{Type: wire.ArgTypeInt32, Value: int32(508)},    // physical_width
		{Type: wire.ArgTypeInt32, Value: int32(285)},    // physical_height
		{Type: wire.ArgTypeInt32, Value: SubpixelHorizontalRGB}, // subpixel
		{Type: wire.ArgTypeString, Value: "Dell Inc."},  // make
		{Type: wire.ArgTypeString, Value: "P2415Q"},     // model
		{Type: wire.ArgTypeInt32, Value: TransformNormal}, // transform
	}

	err := output.HandleEvent(outputEventGeometry, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	geom := output.Geometry()
	if geom.X != 0 || geom.Y != 0 {
		t.Errorf("expected position (0,0), got (%d,%d)", geom.X, geom.Y)
	}
	if geom.PhysicalW != 508 || geom.PhysicalH != 285 {
		t.Errorf("expected physical size (508,285), got (%d,%d)", geom.PhysicalW, geom.PhysicalH)
	}
	if geom.Make != "Dell Inc." {
		t.Errorf("expected make 'Dell Inc.', got '%s'", geom.Make)
	}
	if geom.Model != "P2415Q" {
		t.Errorf("expected model 'P2415Q', got '%s'", geom.Model)
	}
}

func TestHandleModeEvent(t *testing.T) {
	conn := &mockConnection{}
	output := New(10, conn, 3)

	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: ModeFlagCurrent | ModeFlagPreferred},
		{Type: wire.ArgTypeInt32, Value: int32(3840)}, // width
		{Type: wire.ArgTypeInt32, Value: int32(2160)}, // height
		{Type: wire.ArgTypeInt32, Value: int32(60000)}, // refresh (60 Hz in mHz)
	}

	err := output.HandleEvent(outputEventMode, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mode := output.Mode()
	if mode.Flags != (ModeFlagCurrent | ModeFlagPreferred) {
		t.Errorf("expected flags %d, got %d", ModeFlagCurrent|ModeFlagPreferred, mode.Flags)
	}
	if mode.Width != 3840 || mode.Height != 2160 {
		t.Errorf("expected resolution (3840,2160), got (%d,%d)", mode.Width, mode.Height)
	}
	if mode.Refresh != 60000 {
		t.Errorf("expected refresh 60000, got %d", mode.Refresh)
	}
}

func TestHandleDoneEvent(t *testing.T) {
	conn := &mockConnection{}
	output := New(10, conn, 3)

	err := output.HandleEvent(outputEventDone, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !output.done {
		t.Error("expected done to be true")
	}
}

func TestRelease(t *testing.T) {
	conn := &mockConnection{}
	output := New(10, conn, 3)

	err := output.Release()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(conn.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(conn.requests))
	}

	req := conn.requests[0]
	if req.objectID != 10 {
		t.Errorf("expected object ID 10, got %d", req.objectID)
	}
	if req.opcode != outputOpcodeRelease {
		t.Errorf("expected opcode %d, got %d", outputOpcodeRelease, req.opcode)
	}
}

func TestInvalidScaleEvent(t *testing.T) {
	conn := &mockConnection{}
	output := New(10, conn, 3)

	// Wrong number of arguments
	args := []wire.Argument{}
	err := output.HandleEvent(outputEventScale, args)
	if err == nil {
		t.Error("expected error for invalid scale event")
	}

	// Wrong type
	args = []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: uint32(2)},
	}
	err = output.HandleEvent(outputEventScale, args)
	if err == nil {
		t.Error("expected error for wrong type in scale event")
	}
}

func TestUnknownEvent(t *testing.T) {
	conn := &mockConnection{}
	output := New(10, conn, 3)

	err := output.HandleEvent(999, nil)
	if err == nil {
		t.Error("expected error for unknown event opcode")
	}
}
