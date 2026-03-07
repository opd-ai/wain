package dri3

import (
	"testing"
)

// TestQueryExtension validates DRI3 extension query structure.
func TestQueryExtension(t *testing.T) {
	tests := []struct {
		name          string
		mockReply     []byte
		wantMajor     uint32
		wantMinor     uint32
		wantErr       bool
		wantModifiers bool
	}{
		{
			name: "DRI3 1.2 supported",
			mockReply: []byte{
				1, 0, 0, 0, // type=1 (reply), pad, sequence
				0, 0, 0, 0, // length=0 (no extra data)
				1, 0, 0, 0, // major version = 1
				2, 0, 0, 0, // minor version = 2
				0, 0, 0, 0, 0, 0, 0, 0, // padding
				0, 0, 0, 0, 0, 0, 0, 0, // padding
			},
			wantMajor:     1,
			wantMinor:     2,
			wantErr:       false,
			wantModifiers: true,
		},
		{
			name: "DRI3 1.0 supported (no modifiers)",
			mockReply: []byte{
				1, 0, 0, 0, // type=1 (reply)
				0, 0, 0, 0, // length=0
				1, 0, 0, 0, // major version = 1
				0, 0, 0, 0, // minor version = 0
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
			},
			wantMajor:     1,
			wantMinor:     0,
			wantErr:       false,
			wantModifiers: false,
		},
		{
			name: "DRI3 0.x too old",
			mockReply: []byte{
				1, 0, 0, 0,
				0, 0, 0, 0,
				0, 0, 0, 0, // major version = 0
				9, 0, 0, 0, // minor version = 9
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &mockConnection{
				extensionOpcode: 150, // arbitrary DRI3 opcode
				reply:           tt.mockReply,
			}

			ext, err := QueryExtension(conn)
			if tt.wantErr {
				if err == nil {
					t.Errorf("QueryExtension() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("QueryExtension() unexpected error: %v", err)
			}

			if ext.MajorVersion() != tt.wantMajor {
				t.Errorf("MajorVersion() = %d, want %d", ext.MajorVersion(), tt.wantMajor)
			}

			if ext.MinorVersion() != tt.wantMinor {
				t.Errorf("MinorVersion() = %d, want %d", ext.MinorVersion(), tt.wantMinor)
			}

			if ext.SupportsModifiers() != tt.wantModifiers {
				t.Errorf("SupportsModifiers() = %v, want %v", ext.SupportsModifiers(), tt.wantModifiers)
			}
		})
	}
}

// TestPixmapFromBufferValidation tests parameter validation.
func TestPixmapFromBufferValidation(t *testing.T) {
	ext := &Extension{
		baseOpcode:   150,
		majorVersion: 1,
		minorVersion: 2,
	}

	conn := &mockConnection{}

	// Invalid file descriptor
	err := ext.PixmapFromBuffer(conn, 100, 50, 1024, 800, 600, 3200, 32, 32, -1)
	if err != ErrInvalidFD {
		t.Errorf("PixmapFromBuffer(-1) = %v, want ErrInvalidFD", err)
	}
}

// TestPixmapFromBuffersValidation tests multi-plane validation.
func TestPixmapFromBuffersValidation(t *testing.T) {
	ext12 := &Extension{
		baseOpcode:   150,
		majorVersion: 1,
		minorVersion: 2,
	}

	ext10 := &Extension{
		baseOpcode:   150,
		majorVersion: 1,
		minorVersion: 0,
	}

	conn := &mockConnection{}

	tests := []struct {
		name    string
		ext     *Extension
		fds     []int
		strides []uint32
		offsets []uint32
		wantErr error
	}{
		{
			name:    "version too old",
			ext:     ext10,
			fds:     []int{3},
			strides: []uint32{3200},
			offsets: []uint32{0},
			wantErr: nil, // will fail with version check
		},
		{
			name:    "empty fds",
			ext:     ext12,
			fds:     []int{},
			strides: []uint32{},
			offsets: []uint32{},
			wantErr: nil, // will fail with validation
		},
		{
			name:    "mismatched counts",
			ext:     ext12,
			fds:     []int{3, 4},
			strides: []uint32{3200},
			offsets: []uint32{0},
			wantErr: nil, // will fail with validation
		},
		{
			name:    "invalid fd",
			ext:     ext12,
			fds:     []int{-1},
			strides: []uint32{3200},
			offsets: []uint32{0},
			wantErr: ErrInvalidFD,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ext.PixmapFromBuffers(conn, 100, 50, 800, 600, 0x34325241, 0, 32, 32,
				tt.strides, tt.offsets, tt.fds)
			if err == nil {
				t.Errorf("PixmapFromBuffers() expected error, got nil")
			}
		})
	}
}

// mockConnection is a test double for the Connection interface.
type mockConnection struct {
	extensionOpcode uint8
	reply           []byte
	replyFDs        []int
	sentRequests    [][]byte
	sentFDs         [][]int
}

func (m *mockConnection) AllocXID() (XID, error) {
	return 12345, nil
}

func (m *mockConnection) SendRequest(buf []byte) error {
	m.sentRequests = append(m.sentRequests, buf)
	return nil
}

func (m *mockConnection) SendRequestAndReply(req []byte) ([]byte, error) {
	m.sentRequests = append(m.sentRequests, req)
	return m.reply, nil
}

func (m *mockConnection) SendRequestWithFDs(req []byte, fds []int) error {
	m.sentRequests = append(m.sentRequests, req)
	m.sentFDs = append(m.sentFDs, fds)
	return nil
}

func (m *mockConnection) SendRequestAndReplyWithFDs(req []byte, fds []int) ([]byte, []int, error) {
	m.sentRequests = append(m.sentRequests, req)
	m.sentFDs = append(m.sentFDs, fds)
	return m.reply, m.replyFDs, nil
}

func (m *mockConnection) ExtensionOpcode(name string) (uint8, error) {
	if name == ExtensionName {
		return m.extensionOpcode, nil
	}
	return 0, ErrNotSupported
}
