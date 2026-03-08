package client

import (
	"testing"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// TestObjectID verifies object ID allocation.
func TestObjectID(t *testing.T) {
	tests := []struct {
		name     string
		allocate int
		wantIDs  []uint32
	}{
		{
			name:     "first allocation",
			allocate: 1,
			wantIDs:  []uint32{2},
		},
		{
			name:     "sequential allocations",
			allocate: 5,
			wantIDs:  []uint32{2, 3, 4, 5, 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &Connection{}
			conn.nextID.Store(FirstClientObjectID)

			var gotIDs []uint32
			for i := 0; i < tt.allocate; i++ {
				gotIDs = append(gotIDs, conn.allocID())
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("got %d IDs, want %d", len(gotIDs), len(tt.wantIDs))
			}

			for i := range gotIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Errorf("ID[%d] = %d, want %d", i, gotIDs[i], tt.wantIDs[i])
				}
			}
		})
	}
}

// TestBaseObject verifies the base object implementation.
func TestBaseObject(t *testing.T) {
	obj := &baseObject{
		id:    42,
		iface: "wl_test",
	}

	if got := obj.ID(); got != 42 {
		t.Errorf("ID() = %d, want 42", got)
	}

	if got := obj.Interface(); got != "wl_test" {
		t.Errorf("Interface() = %q, want %q", got, "wl_test")
	}
}

// TestDisplayConstants verifies display object constants.
func TestDisplayConstants(t *testing.T) {
	if DisplayObjectID != 1 {
		t.Errorf("DisplayObjectID = %d, want 1", DisplayObjectID)
	}

	if FirstClientObjectID != 2 {
		t.Errorf("FirstClientObjectID = %d, want 2", FirstClientObjectID)
	}
}

// TestRegistryGlobalManagement verifies global addition and removal.
func TestRegistryGlobalManagement(t *testing.T) {
	conn := &Connection{
		objects: make(map[uint32]Object),
	}
	conn.nextID.Store(FirstClientObjectID)

	registry := &Registry{
		baseObject: baseObject{
			id:    2,
			iface: "wl_registry",
			conn:  conn,
		},
		globals: make(map[uint32]*Global),
	}

	// Add a global.
	registry.addGlobal(1, "wl_compositor", 4)

	globals := registry.Globals()
	if len(globals) != 1 {
		t.Fatalf("got %d globals, want 1", len(globals))
	}

	global := globals[1]
	if global.Name != 1 {
		t.Errorf("global.Name = %d, want 1", global.Name)
	}
	if global.Interface != "wl_compositor" {
		t.Errorf("global.Interface = %q, want %q", global.Interface, "wl_compositor")
	}
	if global.Version != 4 {
		t.Errorf("global.Version = %d, want 4", global.Version)
	}

	// Find the global by interface.
	found := registry.FindGlobal("wl_compositor")
	if found == nil {
		t.Fatal("FindGlobal returned nil")
	}
	if found.Name != 1 {
		t.Errorf("found.Name = %d, want 1", found.Name)
	}

	// Remove the global.
	registry.removeGlobal(1)

	globals = registry.Globals()
	if len(globals) != 0 {
		t.Errorf("got %d globals after removal, want 0", len(globals))
	}

	// Verify FindGlobal returns nil after removal.
	found = registry.FindGlobal("wl_compositor")
	if found != nil {
		t.Errorf("FindGlobal returned %v after removal, want nil", found)
	}
}

// TestRegistryFindGlobal verifies global search.
func TestRegistryFindGlobal(t *testing.T) {
	registry := &Registry{
		globals: map[uint32]*Global{
			1: {Name: 1, Interface: "wl_compositor", Version: 4},
			2: {Name: 2, Interface: "wl_shm", Version: 1},
			3: {Name: 3, Interface: "wl_seat", Version: 5},
		},
	}

	tests := []struct {
		name      string
		iface     string
		wantFound bool
		wantName  uint32
	}{
		{
			name:      "find compositor",
			iface:     "wl_compositor",
			wantFound: true,
			wantName:  1,
		},
		{
			name:      "find shm",
			iface:     "wl_shm",
			wantFound: true,
			wantName:  2,
		},
		{
			name:      "not found",
			iface:     "wl_output",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			global := registry.FindGlobal(tt.iface)

			if tt.wantFound {
				if global == nil {
					t.Fatal("FindGlobal returned nil, want global")
				}
				if global.Name != tt.wantName {
					t.Errorf("global.Name = %d, want %d", global.Name, tt.wantName)
				}
			} else {
				if global != nil {
					t.Errorf("FindGlobal returned %v, want nil", global)
				}
			}
		})
	}
}

// TestCallbackChannel verifies callback done channel.
func TestCallbackChannel(t *testing.T) {
	cb := &Callback{
		doneChan: make(chan uint32, 1),
	}

	// Verify the channel is initially empty.
	select {
	case <-cb.Done():
		t.Fatal("channel should be empty initially")
	default:
	}

	// Send a value and verify it can be received.
	cb.doneChan <- 12345

	select {
	case val := <-cb.Done():
		if val != 12345 {
			t.Errorf("got %d, want 12345", val)
		}
	default:
		t.Fatal("channel should have value")
	}
}

// TestArgumentSizeCalculation verifies message size calculations.
func TestArgumentSizeCalculation(t *testing.T) {
	tests := []struct {
		name string
		arg  wire.Argument
		want uint16
	}{
		{
			name: "uint32",
			arg:  wire.Argument{Type: wire.ArgTypeUint32, Value: uint32(42)},
			want: 4,
		},
		{
			name: "int32",
			arg:  wire.Argument{Type: wire.ArgTypeInt32, Value: int32(-42)},
			want: 4,
		},
		{
			name: "object",
			arg:  wire.Argument{Type: wire.ArgTypeObject, Value: uint32(10)},
			want: 4,
		},
		{
			name: "new_id",
			arg:  wire.Argument{Type: wire.ArgTypeNewID, Value: uint32(20)},
			want: 4,
		},
		{
			name: "empty string",
			arg:  wire.Argument{Type: wire.ArgTypeString, Value: ""},
			want: 4, // 4 bytes length (0)
		},
		{
			name: "short string",
			arg:  wire.Argument{Type: wire.ArgTypeString, Value: "hi"},
			want: 8, // 4 bytes length + 4 bytes data (including null + padding)
		},
		{
			name: "aligned string",
			arg:  wire.Argument{Type: wire.ArgTypeString, Value: "abc"},
			want: 8, // 4 bytes length + 4 bytes data (including null)
		},
		{
			name: "string needing padding",
			arg:  wire.Argument{Type: wire.ArgTypeString, Value: "abcd"},
			want: 12, // 4 bytes length + 8 bytes data (including null + padding)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.arg.Size()
			if got != tt.want {
				t.Errorf("Size() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestConnectionClosedError verifies operations on closed connections fail.
func TestConnectionClosedError(t *testing.T) {
	conn := &Connection{
		closed: true,
	}

	err := conn.sendRequest(1, 0, nil)
	if err != ErrClosed {
		t.Errorf("sendRequest on closed connection: got %v, want %v", err, ErrClosed)
	}

	err = conn.Flush()
	if err != ErrClosed {
		t.Errorf("Flush on closed connection: got %v, want %v", err, ErrClosed)
	}
}

// TestRegistry_BindXdgDecorationManager_WrongInterface verifies error on wrong interface.
func TestRegistry_BindXdgDecorationManager_WrongInterface(t *testing.T) {
	conn := &Connection{}
	conn.nextID.Store(FirstClientObjectID)

	registry := &Registry{
		baseObject: baseObject{
			id:    2,
			iface: "wl_registry",
			conn:  conn,
		},
		globals: make(map[uint32]*Global),
	}

	global := &Global{
		Name:      10,
		Interface: "wl_compositor",
		Version:   4,
	}

	_, _, err := registry.BindXdgDecorationManager(global)
	if err == nil {
		t.Fatal("expected error for wrong interface, got nil")
	}

	expectedErr := "registry: not zxdg_decoration_manager_v1: wl_compositor"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}
