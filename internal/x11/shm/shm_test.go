package shm

import (
	"testing"
	"unsafe"
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

// TestSegmentGetBuffer verifies GetBuffer validation.
func TestSegmentGetBuffer(t *testing.T) {
	tests := []struct {
		name    string
		seg     *Segment
		wantErr error
	}{
		{
			name: "destroyed segment",
			seg: &Segment{
				Addr: nil,
				Size: 1024,
			},
			wantErr: ErrInvalidSegment,
		},
		{
			name: "negative size",
			seg: &Segment{
				// Test fixture with constant address. The uintptr->unsafe.Pointer
				// conversion triggers go vet warning but is safe for test constants.
				Addr: unsafe.Pointer(uintptr(0x1000)),
				Size: -1,
			},
			wantErr: ErrSegmentTooLarge,
		},
		{
			name: "size exceeds maximum",
			seg: &Segment{
				// Test fixture with constant address. The uintptr->unsafe.Pointer
				// conversion triggers go vet warning but is safe for test constants.
				Addr: unsafe.Pointer(uintptr(0x1000)),
				Size: (1 << 30) + 1,
			},
			wantErr: ErrSegmentTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.seg.GetBuffer()
			if err != tt.wantErr {
				t.Errorf("GetBuffer() error = %v, want %v", err, tt.wantErr)
			}
		})
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
