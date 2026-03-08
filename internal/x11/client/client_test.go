package client_test

import (
	"testing"

	"github.com/opd-ai/wain/internal/x11/client"
)

func TestXIDAllocation(t *testing.T) {
	// This test doesn't require a real X server connection
	// We test the XID type and basic validation
	var xid client.XID = 0x12345678

	if uint32(xid) != 0x12345678 {
		t.Errorf("XID conversion failed: got %#x", uint32(xid))
	}
}

func TestConfigureWindowMask(t *testing.T) {
	tests := []struct {
		name string
		mask client.ConfigureWindowMask
		want uint16
	}{
		{
			name: "X position",
			mask: client.ConfigMaskX,
			want: 1 << 0,
		},
		{
			name: "Y position",
			mask: client.ConfigMaskY,
			want: 1 << 1,
		},
		{
			name: "Width",
			mask: client.ConfigMaskWidth,
			want: 1 << 2,
		},
		{
			name: "Height",
			mask: client.ConfigMaskHeight,
			want: 1 << 3,
		},
		{
			name: "combined X and Y",
			mask: client.ConfigMaskX | client.ConfigMaskY,
			want: (1 << 0) | (1 << 1),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := uint16(tc.mask)
			if got != tc.want {
				t.Errorf("got %#x, want %#x", got, tc.want)
			}
		})
	}
}

// Note: Full integration tests for Connect, CreateWindow, and MapWindow
// require a running X server and are tested separately as integration tests.
// These tests verify the API surface and basic type conversions.

func TestClosedConnection(t *testing.T) {
	// We can't test a full connection without an X server,
	// but we can verify the error handling patterns exist
	// by checking that ErrClosed is defined
	if client.ErrClosed == nil {
		t.Error("ErrClosed should be defined")
	}

	if client.ErrInvalidXID == nil {
		t.Error("ErrInvalidXID should be defined")
	}
}

func TestConnectInvalidDisplay(t *testing.T) {
	tests := []struct {
		name    string
		display string
	}{
		{
			name:    "nonexistent display",
			display: "999",
		},
		{
			name:    "invalid display number",
			display: "invalid",
		},
		{
			name:    "empty display",
			display: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.Connect(tc.display)
			if err == nil {
				t.Error("Connect should fail with invalid display path")
			}
		})
	}
}

func TestAllocXIDUniqueness(t *testing.T) {
	// Create a mock connection structure with controlled resource ID parameters
	// Since we can't easily create a real connection without an X server,
	// we test the actual AllocXID behavior by connecting to a real server if available,
	// or skip if not available
	conn, err := client.Connect("0")
	if err != nil {
		// X server not available, test with synthetic parameters
		t.Skip("X server not available for XID allocation test")
		return
	}
	defer conn.Close()

	// Allocate 100 XIDs and check uniqueness
	xids := make(map[client.XID]bool)
	for i := 0; i < 100; i++ {
		xid, err := conn.AllocXID()
		if err != nil {
			t.Fatalf("AllocXID failed at iteration %d: %v", i, err)
		}

		if xids[xid] {
			t.Errorf("AllocXID produced duplicate XID %#x at iteration %d", xid, i)
		}
		xids[xid] = true
	}

	if len(xids) != 100 {
		t.Errorf("Expected 100 unique XIDs, got %d", len(xids))
	}
}

func TestExtensionOpcodeKnownExtension(t *testing.T) {
	conn, err := client.Connect("0")
	if err != nil {
		t.Skip("X server not available for extension query test")
		return
	}
	defer conn.Close()

	// Test known X11 extension that should be universally supported
	opcode, err := conn.ExtensionOpcode("BIG-REQUESTS")
	if err != nil {
		t.Fatalf("ExtensionOpcode failed for BIG-REQUESTS: %v", err)
	}

	if opcode == 0 {
		t.Error("ExtensionOpcode should return non-zero opcode for BIG-REQUESTS")
	}
}

func TestExtensionOpcodeUnknownExtension(t *testing.T) {
	conn, err := client.Connect("0")
	if err != nil {
		t.Skip("X server not available for extension query test")
		return
	}
	defer conn.Close()

	// Test with a made-up extension name that won't exist
	_, err = conn.ExtensionOpcode("DEFINITELY-NOT-A-REAL-EXTENSION-9999")
	if err == nil {
		t.Error("ExtensionOpcode should fail for nonexistent extension")
	}
}

func TestConnectionProperties(t *testing.T) {
	conn, err := client.Connect("0")
	if err != nil {
		t.Skip("X server not available for connection properties test")
		return
	}
	defer conn.Close()

	// Test RootWindow returns non-zero
	rootWin := conn.RootWindow()
	if rootWin == 0 {
		t.Error("RootWindow should return non-zero XID")
	}

	// Test RootVisual returns non-zero
	rootVis := conn.RootVisual()
	if rootVis == 0 {
		t.Error("RootVisual should return non-zero visual ID")
	}

	// Test RootDepth returns reasonable value (typically 24 or 32)
	rootDepth := conn.RootDepth()
	if rootDepth == 0 || rootDepth > 32 {
		t.Errorf("RootDepth returned unexpected value: %d", rootDepth)
	}
}

func TestDoubleClose(t *testing.T) {
	conn, err := client.Connect("0")
	if err != nil {
		t.Skip("X server not available for double close test")
		return
	}

	// First close should succeed
	err = conn.Close()
	if err != nil {
		t.Errorf("First Close failed: %v", err)
	}

	// Second close should return ErrClosed
	err = conn.Close()
	if err != client.ErrClosed {
		t.Errorf("Second Close should return ErrClosed, got: %v", err)
	}
}

func TestAllocXIDAfterClose(t *testing.T) {
	conn, err := client.Connect("0")
	if err != nil {
		t.Skip("X server not available for AllocXID after close test")
		return
	}

	// Close connection
	conn.Close()

	// AllocXID should return ErrClosed
	_, err = conn.AllocXID()
	if err != client.ErrClosed {
		t.Errorf("AllocXID after Close should return ErrClosed, got: %v", err)
	}
}
