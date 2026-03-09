package shm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"unsafe"

	"github.com/opd-ai/wain/internal/x11/wire"
)

// mockConnection is a test double for the Connection interface.
type mockConnection struct {
	extensionOpcode uint8
	reply           []byte
	sentRequests    [][]byte
	nextXID         uint32
	allocXIDError   error
	sendReqError    error
	sendReplyError  error
	extOpcodeError  error
}

func (m *mockConnection) AllocXID() (XID, error) {
	if m.allocXIDError != nil {
		return 0, m.allocXIDError
	}
	if m.nextXID == 0 {
		m.nextXID = 1000
	}
	xid := m.nextXID
	m.nextXID++
	return XID(xid), nil
}

func (m *mockConnection) SendRequest(buf []byte) error {
	if m.sendReqError != nil {
		return m.sendReqError
	}
	m.sentRequests = append(m.sentRequests, buf)
	return nil
}

func (m *mockConnection) SendRequestAndReply(req []byte) ([]byte, error) {
	if m.sendReplyError != nil {
		return nil, m.sendReplyError
	}
	m.sentRequests = append(m.sentRequests, req)
	return m.reply, nil
}

func (m *mockConnection) ExtensionOpcode(name string) (uint8, error) {
	if m.extOpcodeError != nil {
		return 0, m.extOpcodeError
	}
	if name == ExtensionName {
		return m.extensionOpcode, nil
	}
	return 0, errors.New("extension not found")
}

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
	// Create a valid address for tests that need non-nil Addr.
	// Use the address of a local variable to get a valid pointer without uintptr conversion.
	var dummyByte byte
	validAddr := unsafe.Pointer(&dummyByte)

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
				Addr: validAddr,
				Size: -1,
			},
			wantErr: ErrSegmentTooLarge,
		},
		{
			name: "size exceeds maximum",
			seg: &Segment{
				Addr: validAddr,
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

// TestQueryExtension tests extension query with various server responses.
func TestQueryExtension(t *testing.T) {
	tests := []struct {
		name              string
		extensionOpcode   uint8
		reply             []byte
		extOpcodeError    error
		sendReplyError    error
		wantErr           bool
		wantMajor         uint16
		wantMinor         uint16
		wantSharedPixmaps bool
		wantPixmapFormat  uint8
	}{
		{
			name:            "successful query with shared pixmaps",
			extensionOpcode: 130,
			reply: func() []byte {
				buf := make([]byte, 32)
				buf[0] = 1                                   // reply type
				buf[1] = 1                                   // shared pixmaps = true
				binary.LittleEndian.PutUint16(buf[8:10], 1)  // major version = 1
				binary.LittleEndian.PutUint16(buf[10:12], 2) // minor version = 2
				buf[16] = ShmPixmapFormatZ                   // pixmap format
				return buf
			}(),
			wantErr:           false,
			wantMajor:         1,
			wantMinor:         2,
			wantSharedPixmaps: true,
			wantPixmapFormat:  ShmPixmapFormatZ,
		},
		{
			name:            "successful query without shared pixmaps",
			extensionOpcode: 130,
			reply: func() []byte {
				buf := make([]byte, 32)
				buf[0] = 1                                   // reply type
				buf[1] = 0                                   // shared pixmaps = false
				binary.LittleEndian.PutUint16(buf[8:10], 1)  // major version = 1
				binary.LittleEndian.PutUint16(buf[10:12], 0) // minor version = 0
				buf[16] = ShmPixmapFormatXY                  // pixmap format
				return buf
			}(),
			wantErr:           false,
			wantMajor:         1,
			wantMinor:         0,
			wantSharedPixmaps: false,
			wantPixmapFormat:  ShmPixmapFormatXY,
		},
		{
			name:           "extension not found",
			extOpcodeError: errors.New("extension not found"),
			wantErr:        true,
		},
		{
			name:            "query failed",
			extensionOpcode: 130,
			sendReplyError:  errors.New("connection error"),
			wantErr:         true,
		},
		{
			name:            "invalid reply too short",
			extensionOpcode: 130,
			reply:           make([]byte, 16),
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &mockConnection{
				extensionOpcode: tt.extensionOpcode,
				reply:           tt.reply,
				extOpcodeError:  tt.extOpcodeError,
				sendReplyError:  tt.sendReplyError,
			}

			ext, err := QueryExtension(conn)
			if tt.wantErr {
				if err == nil {
					t.Error("QueryExtension() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("QueryExtension() unexpected error: %v", err)
			}

			if !ext.Supported() {
				t.Error("extension should be marked as supported")
			}

			major, minor := ext.Version()
			if major != tt.wantMajor {
				t.Errorf("major version = %d, want %d", major, tt.wantMajor)
			}
			if minor != tt.wantMinor {
				t.Errorf("minor version = %d, want %d", minor, tt.wantMinor)
			}
			if ext.SharedPixmapsSupported() != tt.wantSharedPixmaps {
				t.Errorf("SharedPixmapsSupported() = %v, want %v", ext.SharedPixmapsSupported(), tt.wantSharedPixmaps)
			}
			if ext.pixmapFormat != tt.wantPixmapFormat {
				t.Errorf("pixmap format = %d, want %d", ext.pixmapFormat, tt.wantPixmapFormat)
			}
		})
	}
}

// TestCreateSegment tests segment creation with various parameters.
func TestCreateSegment(t *testing.T) {
	// Note: This test skips actual SHM syscalls since they're OS-dependent
	// and require kernel support. We test the error paths and structure.
	tests := []struct {
		name          string
		supported     bool
		size          int
		readOnly      bool
		allocXIDError error
		wantErr       error
	}{
		{
			name:      "extension not supported",
			supported: false,
			size:      4096,
			readOnly:  false,
			wantErr:   ErrNotSupported,
		},
		{
			name:          "XID allocation fails",
			supported:     true,
			size:          4096,
			readOnly:      false,
			allocXIDError: errors.New("no XIDs available"),
			wantErr:       nil, // wrapped error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &Extension{
				supported: tt.supported,
				segments:  make(map[Seg]*Segment),
			}

			conn := &mockConnection{
				allocXIDError: tt.allocXIDError,
			}

			seg, err := ext.CreateSegment(conn, tt.size, tt.readOnly)
			if tt.wantErr != nil {
				if err == nil {
					t.Error("CreateSegment() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("CreateSegment() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if tt.allocXIDError != nil {
				if err == nil {
					t.Error("CreateSegment() should fail when AllocXID fails")
				}
				return
			}

			// For successful cases that don't trigger syscalls,
			// the test would need actual kernel support
			if err != nil && !errors.Is(err, ErrShmFailed) {
				t.Errorf("CreateSegment() unexpected error type: %v", err)
			}

			// If we got a segment (unlikely without kernel support), validate it
			if seg != nil {
				if seg.Size != tt.size {
					t.Errorf("segment size = %d, want %d", seg.Size, tt.size)
				}
				if seg.ReadOnly != tt.readOnly {
					t.Errorf("segment readOnly = %v, want %v", seg.ReadOnly, tt.readOnly)
				}
			}
		})
	}
}

// TestAttachSegment tests segment attachment protocol encoding.
func TestAttachSegment(t *testing.T) {
	tests := []struct {
		name         string
		supported    bool
		segment      *Segment
		sendReqError error
		wantErr      error
	}{
		{
			name:      "extension not supported",
			supported: false,
			segment: &Segment{
				ID:       100,
				ShmID:    12345,
				ReadOnly: false,
			},
			wantErr: ErrNotSupported,
		},
		{
			name:      "successful attach read-write",
			supported: true,
			segment: &Segment{
				ID:       100,
				ShmID:    12345,
				ReadOnly: false,
			},
			wantErr: nil,
		},
		{
			name:      "successful attach read-only",
			supported: true,
			segment: &Segment{
				ID:       200,
				ShmID:    54321,
				ReadOnly: true,
			},
			wantErr: nil,
		},
		{
			name:      "send request fails",
			supported: true,
			segment: &Segment{
				ID:       100,
				ShmID:    12345,
				ReadOnly: false,
			},
			sendReqError: errors.New("connection error"),
			wantErr:      nil, // wrapped error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &Extension{
				supported:  tt.supported,
				baseOpcode: 130,
				segments:   make(map[Seg]*Segment),
			}

			conn := &mockConnection{
				sendReqError: tt.sendReqError,
			}

			err := ext.AttachSegment(conn, tt.segment)
			if tt.wantErr != nil {
				if err == nil {
					t.Error("AttachSegment() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("AttachSegment() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if tt.sendReqError != nil {
				if err == nil {
					t.Error("AttachSegment() should fail when SendRequest fails")
				}
				return
			}

			if err != nil {
				t.Errorf("AttachSegment() unexpected error: %v", err)
				return
			}

			// Verify the request was sent
			if len(conn.sentRequests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(conn.sentRequests))
			}

			req := conn.sentRequests[0]
			if len(req) != 16 {
				t.Errorf("request length = %d, want 16", len(req))
			}

			// Verify request header
			if req[0] != ext.baseOpcode+ShmAttach {
				t.Errorf("request opcode = %d, want %d", req[0], ext.baseOpcode+ShmAttach)
			}

			// Verify segment ID
			gotSegID := binary.LittleEndian.Uint32(req[4:8])
			if gotSegID != uint32(tt.segment.ID) {
				t.Errorf("segment ID = %d, want %d", gotSegID, tt.segment.ID)
			}

			// Verify SHM ID
			gotShmID := binary.LittleEndian.Uint32(req[8:12])
			if gotShmID != uint32(tt.segment.ShmID) {
				t.Errorf("SHM ID = %d, want %d", gotShmID, tt.segment.ShmID)
			}

			// Verify read-only flag
			readOnlyByte := req[12]
			wantReadOnly := uint8(0)
			if tt.segment.ReadOnly {
				wantReadOnly = 1
			}
			if readOnlyByte != wantReadOnly {
				t.Errorf("read-only flag = %d, want %d", readOnlyByte, wantReadOnly)
			}
		})
	}
}

// TestDetachSegment tests segment detachment protocol encoding.
func TestDetachSegment(t *testing.T) {
	tests := []struct {
		name         string
		supported    bool
		segment      *Segment
		sendReqError error
		wantErr      error
	}{
		{
			name:      "extension not supported",
			supported: false,
			segment:   &Segment{ID: 100},
			wantErr:   ErrNotSupported,
		},
		{
			name:      "successful detach",
			supported: true,
			segment:   &Segment{ID: 100},
			wantErr:   nil,
		},
		{
			name:         "send request fails",
			supported:    true,
			segment:      &Segment{ID: 100},
			sendReqError: errors.New("connection error"),
			wantErr:      nil, // wrapped error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &Extension{
				supported:  tt.supported,
				baseOpcode: 130,
				segments:   make(map[Seg]*Segment),
			}
			ext.segments[tt.segment.ID] = tt.segment

			conn := &mockConnection{
				sendReqError: tt.sendReqError,
			}

			err := ext.DetachSegment(conn, tt.segment)
			if tt.wantErr != nil {
				if err == nil {
					t.Error("DetachSegment() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("DetachSegment() error = %v, want %v", err, tt.wantErr)
				}
				// Segment should still be in map on error
				if _, exists := ext.segments[tt.segment.ID]; !exists {
					t.Error("segment should remain in map on error")
				}
				return
			}

			if tt.sendReqError != nil {
				if err == nil {
					t.Error("DetachSegment() should fail when SendRequest fails")
				}
				return
			}

			if err != nil {
				t.Errorf("DetachSegment() unexpected error: %v", err)
				return
			}

			// Verify the request was sent
			if len(conn.sentRequests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(conn.sentRequests))
			}

			req := conn.sentRequests[0]
			if len(req) != 8 {
				t.Errorf("request length = %d, want 8", len(req))
			}

			// Verify request opcode
			if req[0] != ext.baseOpcode+ShmDetach {
				t.Errorf("request opcode = %d, want %d", req[0], ext.baseOpcode+ShmDetach)
			}

			// Verify segment ID
			gotSegID := binary.LittleEndian.Uint32(req[4:8])
			if gotSegID != uint32(tt.segment.ID) {
				t.Errorf("segment ID = %d, want %d", gotSegID, tt.segment.ID)
			}

			// Verify segment was removed from map
			if _, exists := ext.segments[tt.segment.ID]; exists {
				t.Error("segment should be removed from map on success")
			}
		})
	}
}

// TestPutImage tests PutImage protocol encoding.
func TestPutImage(t *testing.T) {
	tests := []struct {
		name         string
		supported    bool
		drawable     XID
		gc           XID
		segment      *Segment
		width        uint16
		height       uint16
		srcX         int16
		srcY         int16
		dstX         int16
		dstY         int16
		depth        uint8
		format       uint8
		sendEvent    bool
		sendReqError error
		wantErr      error
	}{
		{
			name:      "extension not supported",
			supported: false,
			drawable:  1000,
			gc:        2000,
			segment:   &Segment{ID: 100},
			width:     800,
			height:    600,
			wantErr:   ErrNotSupported,
		},
		{
			name:      "successful put without event",
			supported: true,
			drawable:  1000,
			gc:        2000,
			segment:   &Segment{ID: 100},
			width:     800,
			height:    600,
			srcX:      0,
			srcY:      0,
			dstX:      10,
			dstY:      20,
			depth:     24,
			format:    ShmPixmapFormatZ,
			sendEvent: false,
			wantErr:   nil,
		},
		{
			name:      "successful put with event",
			supported: true,
			drawable:  1000,
			gc:        2000,
			segment:   &Segment{ID: 100},
			width:     800,
			height:    600,
			srcX:      0,
			srcY:      0,
			dstX:      0,
			dstY:      0,
			depth:     32,
			format:    ShmPixmapFormatZ,
			sendEvent: true,
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &Extension{
				supported:  tt.supported,
				baseOpcode: 130,
			}

			conn := &mockConnection{
				sendReqError: tt.sendReqError,
			}

			err := ext.PutImage(conn, tt.drawable, tt.gc, tt.segment, tt.width, tt.height,
				tt.srcX, tt.srcY, tt.dstX, tt.dstY, tt.depth, tt.format, tt.sendEvent)

			if tt.wantErr != nil {
				if err == nil {
					t.Error("PutImage() expected error, got nil")
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("PutImage() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("PutImage() unexpected error: %v", err)
				return
			}

			// Verify the request was sent
			if len(conn.sentRequests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(conn.sentRequests))
			}

			req := conn.sentRequests[0]
			if len(req) != 40 {
				t.Errorf("request length = %d, want 40", len(req))
			}

			// Verify request opcode
			if req[0] != ext.baseOpcode+ShmPutImage {
				t.Errorf("request opcode = %d, want %d", req[0], ext.baseOpcode+ShmPutImage)
			}

			// Verify drawable
			gotDrawable := binary.LittleEndian.Uint32(req[4:8])
			if gotDrawable != uint32(tt.drawable) {
				t.Errorf("drawable = %d, want %d", gotDrawable, tt.drawable)
			}

			// Verify GC
			gotGC := binary.LittleEndian.Uint32(req[8:12])
			if gotGC != uint32(tt.gc) {
				t.Errorf("GC = %d, want %d", gotGC, tt.gc)
			}

			// Verify dimensions
			gotWidth := binary.LittleEndian.Uint16(req[12:14])
			if gotWidth != tt.width {
				t.Errorf("width = %d, want %d", gotWidth, tt.width)
			}
			gotHeight := binary.LittleEndian.Uint16(req[14:16])
			if gotHeight != tt.height {
				t.Errorf("height = %d, want %d", gotHeight, tt.height)
			}

			// Verify depth and format
			if req[28] != tt.depth {
				t.Errorf("depth = %d, want %d", req[28], tt.depth)
			}
			if req[29] != tt.format {
				t.Errorf("format = %d, want %d", req[29], tt.format)
			}

			// Verify send-event flag
			wantEventByte := uint8(0)
			if tt.sendEvent {
				wantEventByte = 1
			}
			if req[30] != wantEventByte {
				t.Errorf("send-event = %d, want %d", req[30], wantEventByte)
			}

			// Verify segment ID
			gotSegID := binary.LittleEndian.Uint32(req[32:36])
			if gotSegID != uint32(tt.segment.ID) {
				t.Errorf("segment ID = %d, want %d", gotSegID, tt.segment.ID)
			}

			// Verify offset is 0
			gotOffset := binary.LittleEndian.Uint32(req[36:40])
			if gotOffset != 0 {
				t.Errorf("offset = %d, want 0", gotOffset)
			}
		})
	}
}

// TestDestroySegment tests the segment cleanup lifecycle.
func TestDestroySegment(t *testing.T) {
	// Create a segment with a valid but dummy address
	var dummyByte byte
	seg := &Segment{
		Addr: unsafe.Pointer(&dummyByte),
		Size: 1024,
	}

	// Note: Calling DestroySegment on non-SHM memory will fail,
	// which is expected. We're testing error handling.
	err := seg.DestroySegment()
	if err == nil {
		// Unexpected success - probably got lucky with syscall
		t.Log("DestroySegment unexpectedly succeeded on non-SHM memory")
	} else {
		// Expected: should fail because it's not real SHM
		if !errors.Is(err, ErrShmFailed) {
			t.Errorf("DestroySegment() error = %v, want error wrapping ErrShmFailed", err)
		}
	}
}

// TestGetBufferValidSize tests GetBuffer with valid segment.
func TestGetBufferValidSize(t *testing.T) {
	var dummyData [1024]byte
	seg := &Segment{
		Addr: unsafe.Pointer(&dummyData[0]),
		Size: 1024,
	}

	buf, err := seg.GetBuffer()
	if err != nil {
		t.Fatalf("GetBuffer() unexpected error: %v", err)
	}

	if len(buf) != 1024 {
		t.Errorf("buffer length = %d, want 1024", len(buf))
	}

	// Test that we can write to the buffer
	buf[0] = 0xAB
	buf[1023] = 0xCD

	if dummyData[0] != 0xAB {
		t.Errorf("write to buffer[0] failed")
	}
	if dummyData[1023] != 0xCD {
		t.Errorf("write to buffer[1023] failed")
	}
}

// TestProtocolEncodingHelpers verifies wire protocol encoding is correct.
func TestProtocolEncodingHelpers(t *testing.T) {
	var buf bytes.Buffer

	// Test request header encoding
	wire.EncodeRequestHeader(&buf, 130, 0, 10)
	if buf.Len() != 4 {
		t.Errorf("request header length = %d, want 4", buf.Len())
	}

	// Verify opcode
	data := buf.Bytes()
	if data[0] != 130 {
		t.Errorf("opcode = %d, want 130", data[0])
	}

	// Test uint32 encoding
	buf.Reset()
	wire.EncodeUint32(&buf, 0x12345678)
	if buf.Len() != 4 {
		t.Errorf("uint32 encoding length = %d, want 4", buf.Len())
	}
	encoded := binary.LittleEndian.Uint32(buf.Bytes())
	if encoded != 0x12345678 {
		t.Errorf("encoded uint32 = %#x, want 0x12345678", encoded)
	}
}
