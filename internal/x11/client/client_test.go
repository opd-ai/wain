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
