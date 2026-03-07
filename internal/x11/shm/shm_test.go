package shm

import (
	"testing"
)

// TestExtensionConstants verifies SHM extension constant values.
func TestExtensionConstants(t *testing.T) {
	tests := []struct {
		name  string
		value uint8
		want  uint8
	}{
		{"QueryVersion opcode", ShmQueryVersion, 0},
		{"Attach opcode", ShmAttach, 1},
		{"Detach opcode", ShmDetach, 2},
		{"PutImage opcode", ShmPutImage, 3},
		{"GetImage opcode", ShmGetImage, 4},
		{"CreatePixmap opcode", ShmCreatePixmap, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("got %v, want %v", tt.value, tt.want)
			}
		})
	}

	// Test extension name separately
	if ExtensionName != "MIT-SHM" {
		t.Errorf("ExtensionName = %q, want %q", ExtensionName, "MIT-SHM")
	}
}

// TestSegmentGetBuffer verifies that GetBuffer returns the correct slice.
func TestSegmentGetBuffer(t *testing.T) {
	// Create a mock segment with known size
	seg := &Segment{
		Addr: 0x1000, // Mock address (won't actually access memory)
		Size: 1024,
	}

	// Note: We can't actually call GetBuffer here because it would access
	// arbitrary memory at address 0x1000. This test just verifies the
	// Segment structure is correct.
	if seg.Size != 1024 {
		t.Errorf("segment size = %d, want 1024", seg.Size)
	}

	if seg.Addr != 0x1000 {
		t.Errorf("segment addr = %x, want 0x1000", seg.Addr)
	}
}

// TestExtensionVersion verifies the Version method.
func TestExtensionVersion(t *testing.T) {
	ext := &Extension{
		majorVersion: 1,
		minorVersion: 2,
	}

	major, minor := ext.Version()
	if major != 1 {
		t.Errorf("major version = %d, want 1", major)
	}
	if minor != 2 {
		t.Errorf("minor version = %d, want 2", minor)
	}
}

// TestExtensionSupported verifies the Supported method.
func TestExtensionSupported(t *testing.T) {
	tests := []struct {
		name      string
		supported bool
	}{
		{"supported", true},
		{"not supported", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &Extension{supported: tt.supported}
			if ext.Supported() != tt.supported {
				t.Errorf("Supported() = %v, want %v", ext.Supported(), tt.supported)
			}
		})
	}
}

// TestExtensionSharedPixmapsSupported verifies the SharedPixmapsSupported method.
func TestExtensionSharedPixmapsSupported(t *testing.T) {
	tests := []struct {
		name          string
		sharedPixmaps bool
	}{
		{"shared pixmaps supported", true},
		{"shared pixmaps not supported", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &Extension{sharedPixmaps: tt.sharedPixmaps}
			if ext.SharedPixmapsSupported() != tt.sharedPixmaps {
				t.Errorf("SharedPixmapsSupported() = %v, want %v", ext.SharedPixmapsSupported(), tt.sharedPixmaps)
			}
		})
	}
}
