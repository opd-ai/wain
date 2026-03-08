package xdg

import (
	"testing"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

func TestNewDecorationManager(t *testing.T) {
	conn := newMockConn()
	mgr := NewDecorationManager(conn, 50, 1)

	if mgr.ID() != 50 {
		t.Errorf("expected ID 50, got %d", mgr.ID())
	}
	if mgr.Interface() != "zxdg_decoration_manager_v1" {
		t.Errorf("expected interface zxdg_decoration_manager_v1, got %s", mgr.Interface())
	}
	if mgr.version != 1 {
		t.Errorf("expected version 1, got %d", mgr.version)
	}
}

func TestDecorationManager_GetToplevelDecoration(t *testing.T) {
	conn := newMockConn()
	mgr := NewDecorationManager(conn, 50, 1)

	toplevel := &Toplevel{}
	toplevel.id = 200

	deco, err := mgr.GetToplevelDecoration(toplevel)
	if err != nil {
		t.Fatalf("GetToplevelDecoration failed: %v", err)
	}

	if deco == nil {
		t.Fatal("GetToplevelDecoration returned nil")
	}

	// Verify request was sent
	req := conn.lastRequest()
	if req == nil {
		t.Fatal("no request sent")
	}
	if req.objectID != 50 {
		t.Errorf("expected request to object 50, got %d", req.objectID)
	}
	if req.opcode != 1 {
		t.Errorf("expected opcode 1, got %d", req.opcode)
	}
	if len(req.args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(req.args))
	}
	if req.args[0].Type != wire.ArgTypeNewID {
		t.Errorf("expected first arg type NewID, got %d", req.args[0].Type)
	}
	if req.args[1].Type != wire.ArgTypeObject {
		t.Errorf("expected second arg type Object, got %d", req.args[1].Type)
	}
	if req.args[1].Value.(uint32) != 200 {
		t.Errorf("expected toplevel ID 200, got %d", req.args[1].Value.(uint32))
	}
}

func TestNewToplevelDecoration(t *testing.T) {
	conn := newMockConn()
	deco := NewToplevelDecoration(conn, 100, 1)

	if deco.ID() != 100 {
		t.Errorf("expected ID 100, got %d", deco.ID())
	}
	if deco.Interface() != "zxdg_toplevel_decoration_v1" {
		t.Errorf("expected interface zxdg_toplevel_decoration_v1, got %s", deco.Interface())
	}
	if deco.version != 1 {
		t.Errorf("expected version 1, got %d", deco.version)
	}
}

func TestToplevelDecoration_SetMode(t *testing.T) {
	conn := newMockConn()
	deco := NewToplevelDecoration(conn, 100, 1)

	err := deco.SetMode(DecorationModeClientSide)
	if err != nil {
		t.Fatalf("SetMode failed: %v", err)
	}

	req := conn.lastRequest()
	if req == nil {
		t.Fatal("no request sent")
	}
	if req.objectID != 100 {
		t.Errorf("expected request to object 100, got %d", req.objectID)
	}
	if req.opcode != 1 {
		t.Errorf("expected opcode 1, got %d", req.opcode)
	}
	if len(req.args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(req.args))
	}
	if req.args[0].Type != wire.ArgTypeUint32 {
		t.Errorf("expected arg type Uint32, got %d", req.args[0].Type)
	}
	if req.args[0].Value.(uint32) != uint32(DecorationModeClientSide) {
		t.Errorf("expected mode %d, got %d", DecorationModeClientSide, req.args[0].Value.(uint32))
	}
}

func TestToplevelDecoration_UnsetMode(t *testing.T) {
	conn := newMockConn()
	deco := NewToplevelDecoration(conn, 100, 1)

	err := deco.UnsetMode()
	if err != nil {
		t.Fatalf("UnsetMode failed: %v", err)
	}

	req := conn.lastRequest()
	if req == nil {
		t.Fatal("no request sent")
	}
	if req.objectID != 100 {
		t.Errorf("expected request to object 100, got %d", req.objectID)
	}
	if req.opcode != 2 {
		t.Errorf("expected opcode 2, got %d", req.opcode)
	}
}

func TestToplevelDecoration_HandleEvent_Configure(t *testing.T) {
	conn := newMockConn()
	deco := NewToplevelDecoration(conn, 100, 1)

	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: uint32(DecorationModeServerSide)},
	}

	err := deco.HandleEvent(0, args)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	if deco.Mode() != DecorationModeServerSide {
		t.Errorf("expected mode %d, got %d", DecorationModeServerSide, deco.Mode())
	}
}

func TestToplevelDecoration_HandleEvent_UnknownOpcode(t *testing.T) {
	conn := newMockConn()
	deco := NewToplevelDecoration(conn, 100, 1)

	err := deco.HandleEvent(999, nil)
	if err == nil {
		t.Fatal("expected error for unknown opcode, got nil")
	}
}

func TestDecorationManager_Destroy(t *testing.T) {
	conn := newMockConn()
	mgr := NewDecorationManager(conn, 50, 1)

	err := mgr.Destroy()
	if err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	req := conn.lastRequest()
	if req == nil {
		t.Fatal("no request sent")
	}
	if req.objectID != 50 {
		t.Errorf("expected request to object 50, got %d", req.objectID)
	}
	if req.opcode != 0 {
		t.Errorf("expected opcode 0, got %d", req.opcode)
	}
}

func TestToplevelDecoration_Destroy(t *testing.T) {
	conn := newMockConn()
	deco := NewToplevelDecoration(conn, 100, 1)

	err := deco.Destroy()
	if err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	req := conn.lastRequest()
	if req == nil {
		t.Fatal("no request sent")
	}
	if req.objectID != 100 {
		t.Errorf("expected request to object 100, got %d", req.objectID)
	}
	if req.opcode != 0 {
		t.Errorf("expected opcode 0, got %d", req.opcode)
	}
}
