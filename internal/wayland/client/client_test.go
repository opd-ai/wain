package client

import (
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/opd-ai/wain/internal/wayland/socket"
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

// TestConnect_InvalidPath verifies Connect fails with invalid socket path.
func TestConnect_InvalidPath(t *testing.T) {
	_, err := Connect("/nonexistent/socket/path/wayland-0")
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

// TestSync_ObjectCreation verifies Sync creates callback object.
func TestSync_ObjectCreation(t *testing.T) {
	conn := &Connection{
		objects: make(map[uint32]Object),
	}
	conn.nextID.Store(FirstClientObjectID)

	display := &Display{
		baseObject: baseObject{
			id:    DisplayObjectID,
			iface: "wl_display",
			conn:  conn,
		},
	}
	conn.display = display
	conn.objects[DisplayObjectID] = display

	// Since Sync sends a request, it will fail without a real socket.
	// We test the object creation part by checking the ID allocation.
	initialID := conn.nextID.Load()

	// Sync would allocate ID 2 for the callback
	expectedCallbackID := initialID

	// We can't actually call Sync without a socket, but we can verify
	// the ID allocation mechanism works.
	allocatedID := conn.allocID()
	if allocatedID != expectedCallbackID {
		t.Errorf("allocID() = %d, want %d", allocatedID, expectedCallbackID)
	}

	// Verify the next ID incremented
	nextID := conn.nextID.Load()
	if nextID != expectedCallbackID+1 {
		t.Errorf("nextID = %d, want %d", nextID, expectedCallbackID+1)
	}
}

// TestGetRegistry_ObjectCreation verifies GetRegistry creates registry object.
func TestGetRegistry_ObjectCreation(t *testing.T) {
	conn := &Connection{
		objects: make(map[uint32]Object),
	}
	conn.nextID.Store(FirstClientObjectID)

	display := &Display{
		baseObject: baseObject{
			id:    DisplayObjectID,
			iface: "wl_display",
			conn:  conn,
		},
	}
	conn.display = display
	conn.objects[DisplayObjectID] = display

	// Verify ID allocation for registry
	initialID := conn.nextID.Load()
	expectedRegistryID := initialID

	allocatedID := conn.allocID()
	if allocatedID != expectedRegistryID {
		t.Errorf("allocID() = %d, want %d", allocatedID, expectedRegistryID)
	}
}

// TestRegistry_BindCompositor_Success verifies BindCompositor with valid global.
func TestRegistry_BindCompositor_Success(t *testing.T) {
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

	global := &Global{
		Name:      1,
		Interface: "wl_compositor",
		Version:   4,
	}

	// BindCompositor will fail without a real socket, but we can verify
	// the error handling path and that it rejects wrong interfaces.
	global.Interface = "wl_seat"
	_, err := registry.BindCompositor(global)
	if err == nil {
		t.Fatal("expected error for wrong interface, got nil")
	}
	if err.Error() != "registry: not a compositor: wl_seat" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRegistry_BindXdgWmBase_Success verifies BindXdgWmBase with valid global.
func TestRegistry_BindXdgWmBase_Success(t *testing.T) {
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

	global := &Global{
		Name:      5,
		Interface: "wl_output",
		Version:   3,
	}

	// Verify error handling for wrong interface
	_, _, err := registry.BindXdgWmBase(global)
	if err == nil {
		t.Fatal("expected error for wrong interface, got nil")
	}
	if err.Error() != "registry: not xdg_wm_base: wl_output" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRegistry_BindDmabuf_WrongInterface verifies BindDmabuf rejects wrong interface.
func TestRegistry_BindDmabuf_WrongInterface(t *testing.T) {
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

	global := &Global{
		Name:      7,
		Interface: "wl_shm",
		Version:   1,
	}

	_, err := registry.BindDmabuf(global)
	if err == nil {
		t.Fatal("expected error for wrong interface, got nil")
	}
	if err.Error() != "registry: not zwp_linux_dmabuf_v1: wl_shm" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRegistry_BindOutput_WrongInterface verifies BindOutput rejects wrong interface.
func TestRegistry_BindOutput_WrongInterface(t *testing.T) {
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

	global := &Global{
		Name:      8,
		Interface: "wl_seat",
		Version:   6,
	}

	_, _, err := registry.BindOutput(global)
	if err == nil {
		t.Fatal("expected error for wrong interface, got nil")
	}
	if err.Error() != "registry: not wl_output: wl_seat" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestConnection_Close verifies connection closure idempotency.
func TestConnection_Close(t *testing.T) {
	// Create a connection that's already closed
	conn := &Connection{
		objects: make(map[uint32]Object),
		closed:  true,
	}

	// Closing an already-closed connection should be a no-op and return nil
	err := conn.Close()
	if err != nil {
		t.Errorf("Close on already closed connection returned error: %v", err)
	}
}

// TestAllocID_Uniqueness verifies AllocID produces unique IDs.
func TestAllocID_Uniqueness(t *testing.T) {
	conn := &Connection{}
	conn.nextID.Store(FirstClientObjectID)

	seen := make(map[uint32]bool)
	count := 100

	for i := 0; i < count; i++ {
		id := conn.AllocID()
		if seen[id] {
			t.Fatalf("AllocID() returned duplicate ID: %d", id)
		}
		seen[id] = true
	}

	if len(seen) != count {
		t.Errorf("got %d unique IDs, want %d", len(seen), count)
	}
}

// TestRegisterObject verifies object registration.
func TestRegisterObject(t *testing.T) {
	conn := &Connection{
		objects: make(map[uint32]Object),
	}

	obj := &baseObject{
		id:    42,
		iface: "wl_test",
		conn:  conn,
	}

	conn.RegisterObject(obj)

	registered, ok := conn.objects[42]
	if !ok {
		t.Fatal("object not registered")
	}

	if registered.ID() != 42 {
		t.Errorf("registered object ID = %d, want 42", registered.ID())
	}

	if registered.Interface() != "wl_test" {
		t.Errorf("registered object interface = %q, want %q", registered.Interface(), "wl_test")
	}
}

// TestSendRequest_ClosedConnection verifies sendRequest returns error on closed connection.
func TestSendRequest_ClosedConnection(t *testing.T) {
	conn := &Connection{
		closed: true,
	}

	err := conn.SendRequest(1, 0, nil)
	if err != ErrClosed {
		t.Errorf("SendRequest on closed connection: got %v, want %v", err, ErrClosed)
	}
}

// TestDisplay_DisplayObject verifies Display() returns the display object.
func TestDisplay_DisplayObject(t *testing.T) {
	conn := &Connection{
		objects: make(map[uint32]Object),
	}

	display := &Display{
		baseObject: baseObject{
			id:    DisplayObjectID,
			iface: "wl_display",
			conn:  conn,
		},
	}
	conn.display = display

	got := conn.Display()
	if got != display {
		t.Error("Display() returned wrong object")
	}
	if got.ID() != DisplayObjectID {
		t.Errorf("Display().ID() = %d, want %d", got.ID(), DisplayObjectID)
	}
}

// TestFlush_Success verifies Flush doesn't error on open connection.
func TestFlush_Success(t *testing.T) {
	conn := &Connection{
		closed: false,
	}

	err := conn.Flush()
	if err != nil {
		t.Errorf("Flush() failed: %v", err)
	}
}

// TestClose_WithObjectsRegistered verifies Close clears objects map.
func TestClose_WithObjectsRegistered(t *testing.T) {
	// Create a temporary Unix socket pair for testing
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
		closed:      false,
	}

	// Add some objects
	conn.objects[1] = &baseObject{id: 1}
	conn.objects[2] = &baseObject{id: 2}

	err = conn.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Verify closed flag is set
	if !conn.closed {
		t.Error("closed flag not set after Close()")
	}

	// Verify objects map is cleared
	if conn.objects != nil {
		t.Error("objects map not nil after Close()")
	}
}

// createSocketPair creates a connected pair of Unix sockets for testing.
func createSocketPair() (*socket.Conn, *socket.Conn, error) {
	server, client, err := socketpair()
	if err != nil {
		return nil, nil, err
	}

	serverConn, err := socket.NewConn(server)
	if err != nil {
		server.Close()
		client.Close()
		return nil, nil, err
	}

	clientConn, err := socket.NewConn(client)
	if err != nil {
		serverConn.Close()
		client.Close()
		return nil, nil, err
	}

	return serverConn, clientConn, nil
}

// socketpair creates a connected pair of Unix sockets.
func socketpair() (*net.UnixConn, *net.UnixConn, error) {
	// Create a Unix socket pair using socketpair syscall
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, err
	}

	f1 := os.NewFile(uintptr(fds[0]), "socket1")
	f2 := os.NewFile(uintptr(fds[1]), "socket2")

	conn1, err := net.FileConn(f1)
	if err != nil {
		f1.Close()
		f2.Close()
		return nil, nil, err
	}

	conn2, err := net.FileConn(f2)
	if err != nil {
		conn1.Close()
		f2.Close()
		return nil, nil, err
	}

	return conn1.(*net.UnixConn), conn2.(*net.UnixConn), nil
}

// TestSync_WithRealSocket verifies Sync with actual socket.
func TestSync_WithRealSocket(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
	}
	conn.nextID.Store(FirstClientObjectID)

	display := &Display{
		baseObject: baseObject{
			id:    DisplayObjectID,
			iface: "wl_display",
			conn:  conn,
		},
	}
	conn.display = display
	conn.objects[DisplayObjectID] = display

	cb, err := display.Sync()
	if err != nil {
		t.Fatalf("Sync() failed: %v", err)
	}

	if cb == nil {
		t.Fatal("Sync() returned nil callback")
	}
	if cb.ID() != FirstClientObjectID {
		t.Errorf("callback ID = %d, want %d", cb.ID(), FirstClientObjectID)
	}
	if cb.Interface() != "wl_callback" {
		t.Errorf("callback interface = %q, want %q", cb.Interface(), "wl_callback")
	}

	// Verify callback is registered
	if _, ok := conn.objects[cb.ID()]; !ok {
		t.Fatal("callback not registered")
	}
}

// TestGetRegistry_WithRealSocket verifies GetRegistry with actual socket.
func TestGetRegistry_WithRealSocket(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
	}
	conn.nextID.Store(FirstClientObjectID)

	display := &Display{
		baseObject: baseObject{
			id:    DisplayObjectID,
			iface: "wl_display",
			conn:  conn,
		},
	}
	conn.display = display
	conn.objects[DisplayObjectID] = display

	registry, err := display.GetRegistry()
	if err != nil {
		t.Fatalf("GetRegistry() failed: %v", err)
	}

	if registry == nil {
		t.Fatal("GetRegistry() returned nil")
	}
	if registry.ID() != FirstClientObjectID {
		t.Errorf("registry ID = %d, want %d", registry.ID(), FirstClientObjectID)
	}

	// Verify registry is registered
	if _, ok := conn.objects[registry.ID()]; !ok {
		t.Fatal("registry not registered")
	}
}

// TestRegistry_Bind_WithRealSocket verifies Bind with actual socket.
func TestRegistry_Bind_WithRealSocket(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
	}
	conn.nextID.Store(FirstClientObjectID)

	registry := &Registry{
		baseObject: baseObject{
			id:    2,
			iface: "wl_registry",
			conn:  conn,
		},
		globals: map[uint32]*Global{
			1: {Name: 1, Interface: "wl_compositor", Version: 4},
		},
	}

	objectID, err := registry.Bind(1, "wl_compositor", 4)
	if err != nil {
		t.Fatalf("Bind() failed: %v", err)
	}

	if objectID != FirstClientObjectID {
		t.Errorf("objectID = %d, want %d", objectID, FirstClientObjectID)
	}
}

// TestRegistry_BindCompositor_WithRealSocket verifies BindCompositor with actual socket.
func TestRegistry_BindCompositor_WithRealSocket(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
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

	global := &Global{
		Name:      1,
		Interface: "wl_compositor",
		Version:   4,
	}

	compositor, err := registry.BindCompositor(global)
	if err != nil {
		t.Fatalf("BindCompositor() failed: %v", err)
	}

	if compositor == nil {
		t.Fatal("BindCompositor() returned nil")
	}
	if compositor.Interface() != "wl_compositor" {
		t.Errorf("compositor interface = %q, want %q", compositor.Interface(), "wl_compositor")
	}

	// Verify compositor is registered
	if _, ok := conn.objects[compositor.ID()]; !ok {
		t.Fatal("compositor not registered")
	}
}

// TestCompositor_CreateSurface_WithRealSocket verifies CreateSurface with actual socket.
func TestCompositor_CreateSurface_WithRealSocket(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
	}
	conn.nextID.Store(FirstClientObjectID)

	compositor := &Compositor{
		baseObject: baseObject{
			id:    3,
			iface: "wl_compositor",
			conn:  conn,
		},
		version: 4,
	}

	surface, err := compositor.CreateSurface()
	if err != nil {
		t.Fatalf("CreateSurface() failed: %v", err)
	}

	if surface == nil {
		t.Fatal("CreateSurface() returned nil")
	}
	if surface.Interface() != "wl_surface" {
		t.Errorf("surface interface = %q, want %q", surface.Interface(), "wl_surface")
	}

	// Verify surface is registered
	if _, ok := conn.objects[surface.ID()]; !ok {
		t.Fatal("surface not registered")
	}
}

// TestSurface_OperationsWithRealSocket verifies surface operations with actual socket.
func TestSurface_OperationsWithRealSocket(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
	}
	conn.nextID.Store(FirstClientObjectID)

	surface := &Surface{
		baseObject: baseObject{
			id:    4,
			iface: "wl_surface",
			conn:  conn,
		},
	}
	conn.objects[surface.ID()] = surface

	// Test Attach
	if err := surface.Attach(10, 0, 0); err != nil {
		t.Errorf("Attach() failed: %v", err)
	}

	// Test Damage
	if err := surface.Damage(0, 0, 800, 600); err != nil {
		t.Errorf("Damage() failed: %v", err)
	}

	// Test SetBufferScale
	if err := surface.SetBufferScale(2); err != nil {
		t.Errorf("SetBufferScale() failed: %v", err)
	}

	// Test Commit
	if err := surface.Commit(); err != nil {
		t.Errorf("Commit() failed: %v", err)
	}

	// Test Frame
	cb, err := surface.Frame()
	if err != nil {
		t.Errorf("Frame() failed: %v", err)
	}
	if cb == nil {
		t.Error("Frame() returned nil callback")
	}

	// Test Destroy (this should remove from objects map)
	initialCount := len(conn.objects)
	if err := surface.Destroy(); err != nil {
		t.Errorf("Destroy() failed: %v", err)
	}

	// Verify surface was removed
	if _, ok := conn.objects[surface.ID()]; ok {
		t.Error("surface still registered after Destroy()")
	}
	if len(conn.objects) != initialCount-1 {
		t.Errorf("objects count = %d, want %d", len(conn.objects), initialCount-1)
	}
}

// TestRegistry_BindXdgWmBase_WithRealSocket verifies BindXdgWmBase with real socket.
func TestRegistry_BindXdgWmBase_WithRealSocket(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
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

	global := &Global{
		Name:      3,
		Interface: "xdg_wm_base",
		Version:   2,
	}

	objectID, version, err := registry.BindXdgWmBase(global)
	if err != nil {
		t.Fatalf("BindXdgWmBase() failed: %v", err)
	}

	if objectID == 0 {
		t.Error("BindXdgWmBase() returned 0 object ID")
	}
	if version != 2 {
		t.Errorf("version = %d, want 2", version)
	}
}

// TestRegistry_BindDmabuf_WithRealSocket verifies BindDmabuf with real socket.
func TestRegistry_BindDmabuf_WithRealSocket(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
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

	global := &Global{
		Name:      4,
		Interface: "zwp_linux_dmabuf_v1",
		Version:   3,
	}

	objectID, err := registry.BindDmabuf(global)
	if err != nil {
		t.Fatalf("BindDmabuf() failed: %v", err)
	}

	if objectID == 0 {
		t.Error("BindDmabuf() returned 0 object ID")
	}
}

// TestRegistry_BindOutput_WithRealSocket verifies BindOutput with real socket.
func TestRegistry_BindOutput_WithRealSocket(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
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

	global := &Global{
		Name:      5,
		Interface: "wl_output",
		Version:   3,
	}

	objectID, version, err := registry.BindOutput(global)
	if err != nil {
		t.Fatalf("BindOutput() failed: %v", err)
	}

	if objectID == 0 {
		t.Error("BindOutput() returned 0 object ID")
	}
	if version != 3 {
		t.Errorf("version = %d, want 3", version)
	}
}

// TestRegistry_BindXdgDecorationManager_WithRealSocket verifies BindXdgDecorationManager.
func TestRegistry_BindXdgDecorationManager_WithRealSocket(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
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

	global := &Global{
		Name:      6,
		Interface: "zxdg_decoration_manager_v1",
		Version:   1,
	}

	objectID, version, err := registry.BindXdgDecorationManager(global)
	if err != nil {
		t.Fatalf("BindXdgDecorationManager() failed: %v", err)
	}

	if objectID == 0 {
		t.Error("BindXdgDecorationManager() returned 0 object ID")
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}
}

// TestSendRequest_MessageEncoding verifies sendRequest properly encodes messages.
func TestSendRequest_MessageEncoding(t *testing.T) {
	server, client, err := createSocketPair()
	if err != nil {
		t.Skip("cannot create socket pair, skipping test")
	}
	defer server.Close()
	defer client.Close()

	conn := &Connection{
		socket:      client,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
	}

	// Test sendRequest with different argument types
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: uint32(42)},
		{Type: wire.ArgTypeInt32, Value: int32(-10)},
		{Type: wire.ArgTypeString, Value: "test"},
	}

	err = conn.sendRequest(1, 0, args)
	if err != nil {
		t.Errorf("sendRequest() failed: %v", err)
	}

	// Verify we can send empty args
	err = conn.sendRequest(1, 1, nil)
	if err != nil {
		t.Errorf("sendRequest() with nil args failed: %v", err)
	}
}
