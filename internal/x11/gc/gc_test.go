package gc_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/opd-ai/wain/internal/x11/gc"
)

// mockConnection implements gc.Connection for testing.
type mockConnection struct {
	nextXID  uint32
	requests [][]byte
	allocErr error
	sendErr  error
}

func (m *mockConnection) AllocXID() (gc.XID, error) {
	if m.allocErr != nil {
		return 0, m.allocErr
	}
	xid := m.nextXID
	m.nextXID++
	return gc.XID(xid), nil
}

func (m *mockConnection) SendRequest(buf []byte) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	// Copy to avoid mutation
	reqCopy := make([]byte, len(buf))
	copy(reqCopy, buf)
	m.requests = append(m.requests, reqCopy)
	return nil
}

func TestCreateGC(t *testing.T) {
	tests := []struct {
		name       string
		drawable   gc.XID
		mask       uint32
		attrs      []uint32
		wantOpcode uint8
		wantXID    uint32
	}{
		{
			name:       "basic GC with foreground",
			drawable:   0x12345678,
			mask:       gc.GCForeground,
			attrs:      []uint32{0xFF0000},
			wantOpcode: 55,
			wantXID:    1000,
		},
		{
			name:       "GC with multiple attributes",
			drawable:   0x87654321,
			mask:       gc.GCForeground | gc.GCBackground | gc.GCFunction,
			attrs:      []uint32{0xFF0000, 0x00FF00, gc.GXCopy},
			wantOpcode: 55,
			wantXID:    1000,
		},
		{
			name:       "GC with no attributes",
			drawable:   0x11111111,
			mask:       0,
			attrs:      []uint32{},
			wantOpcode: 55,
			wantXID:    1000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn := &mockConnection{nextXID: tc.wantXID}

			gcID, err := gc.CreateGC(conn, tc.drawable, tc.mask, tc.attrs)
			if err != nil {
				t.Fatalf("CreateGC failed: %v", err)
			}

			if uint32(gcID) != tc.wantXID {
				t.Errorf("got GC ID %d, want %d", gcID, tc.wantXID)
			}

			if len(conn.requests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(conn.requests))
			}

			req := conn.requests[0]

			// Verify opcode
			if req[0] != tc.wantOpcode {
				t.Errorf("got opcode %d, want %d", req[0], tc.wantOpcode)
			}

			// Verify length (in 4-byte units)
			expectedLen := 4 + len(tc.attrs)
			gotLen := binary.LittleEndian.Uint16(req[2:4])
			if int(gotLen) != expectedLen {
				t.Errorf("got length %d, want %d", gotLen, expectedLen)
			}

			// Verify GC ID
			gotGC := binary.LittleEndian.Uint32(req[4:8])
			if gotGC != tc.wantXID {
				t.Errorf("got GC in request %d, want %d", gotGC, tc.wantXID)
			}

			// Verify drawable
			gotDrawable := binary.LittleEndian.Uint32(req[8:12])
			if gotDrawable != uint32(tc.drawable) {
				t.Errorf("got drawable %d, want %d", gotDrawable, tc.drawable)
			}

			// Verify mask
			gotMask := binary.LittleEndian.Uint32(req[12:16])
			if gotMask != tc.mask {
				t.Errorf("got mask %#x, want %#x", gotMask, tc.mask)
			}

			// Verify attributes
			for i, expectedAttr := range tc.attrs {
				offset := 16 + (i * 4)
				gotAttr := binary.LittleEndian.Uint32(req[offset : offset+4])
				if gotAttr != expectedAttr {
					t.Errorf("attr[%d]: got %#x, want %#x", i, gotAttr, expectedAttr)
				}
			}
		})
	}
}

func TestCreateGCErrors(t *testing.T) {
	t.Run("alloc error", func(t *testing.T) {
		conn := &mockConnection{allocErr: errors.New("alloc failed")}
		_, err := gc.CreateGC(conn, 100, gc.GCForeground, []uint32{0xFF0000})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("send error", func(t *testing.T) {
		conn := &mockConnection{nextXID: 1000, sendErr: errors.New("send failed")}
		_, err := gc.CreateGC(conn, 100, gc.GCForeground, []uint32{0xFF0000})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFreeGC(t *testing.T) {
	conn := &mockConnection{}
	gcID := gc.XID(0x12345678)

	err := gc.FreeGC(conn, gcID)
	if err != nil {
		t.Fatalf("FreeGC failed: %v", err)
	}

	if len(conn.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(conn.requests))
	}

	req := conn.requests[0]

	// Verify opcode (FreeGC = 60)
	if req[0] != 60 {
		t.Errorf("got opcode %d, want 60", req[0])
	}

	// Verify length (2 = 8 bytes total)
	gotLen := binary.LittleEndian.Uint16(req[2:4])
	if gotLen != 2 {
		t.Errorf("got length %d, want 2", gotLen)
	}

	// Verify GC ID
	gotGC := binary.LittleEndian.Uint32(req[4:8])
	if gotGC != uint32(gcID) {
		t.Errorf("got GC %#x, want %#x", gotGC, gcID)
	}
}

func TestCreatePixmap(t *testing.T) {
	tests := []struct {
		name     string
		drawable gc.XID
		width    uint16
		height   uint16
		depth    uint8
		wantXID  uint32
	}{
		{
			name:     "small pixmap",
			drawable: 0x100,
			width:    64,
			height:   64,
			depth:    24,
			wantXID:  2000,
		},
		{
			name:     "large pixmap",
			drawable: 0x200,
			width:    1920,
			height:   1080,
			depth:    32,
			wantXID:  2000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn := &mockConnection{nextXID: tc.wantXID}

			pixmap, err := gc.CreatePixmap(conn, tc.drawable, tc.width, tc.height, tc.depth)
			if err != nil {
				t.Fatalf("CreatePixmap failed: %v", err)
			}

			if uint32(pixmap) != tc.wantXID {
				t.Errorf("got pixmap ID %d, want %d", pixmap, tc.wantXID)
			}

			if len(conn.requests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(conn.requests))
			}

			req := conn.requests[0]

			// Verify opcode (CreatePixmap = 53)
			if req[0] != 53 {
				t.Errorf("got opcode %d, want 53", req[0])
			}

			// Verify depth in data field
			if req[1] != tc.depth {
				t.Errorf("got depth %d, want %d", req[1], tc.depth)
			}

			// Verify length (4 = 16 bytes total)
			gotLen := binary.LittleEndian.Uint16(req[2:4])
			if gotLen != 4 {
				t.Errorf("got length %d, want 4", gotLen)
			}

			// Verify pixmap ID
			gotPixmap := binary.LittleEndian.Uint32(req[4:8])
			if gotPixmap != tc.wantXID {
				t.Errorf("got pixmap %#x, want %#x", gotPixmap, tc.wantXID)
			}

			// Verify drawable
			gotDrawable := binary.LittleEndian.Uint32(req[8:12])
			if gotDrawable != uint32(tc.drawable) {
				t.Errorf("got drawable %#x, want %#x", gotDrawable, tc.drawable)
			}

			// Verify width
			gotWidth := binary.LittleEndian.Uint16(req[12:14])
			if gotWidth != tc.width {
				t.Errorf("got width %d, want %d", gotWidth, tc.width)
			}

			// Verify height
			gotHeight := binary.LittleEndian.Uint16(req[14:16])
			if gotHeight != tc.height {
				t.Errorf("got height %d, want %d", gotHeight, tc.height)
			}
		})
	}
}

func TestFreePixmap(t *testing.T) {
	conn := &mockConnection{}
	pixmapID := gc.XID(0x87654321)

	err := gc.FreePixmap(conn, pixmapID)
	if err != nil {
		t.Fatalf("FreePixmap failed: %v", err)
	}

	if len(conn.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(conn.requests))
	}

	req := conn.requests[0]

	// Verify opcode (FreePixmap = 54)
	if req[0] != 54 {
		t.Errorf("got opcode %d, want 54", req[0])
	}

	// Verify pixmap ID
	gotPixmap := binary.LittleEndian.Uint32(req[4:8])
	if gotPixmap != uint32(pixmapID) {
		t.Errorf("got pixmap %#x, want %#x", gotPixmap, pixmapID)
	}
}

func TestPutImage(t *testing.T) {
	tests := []struct {
		name     string
		drawable gc.XID
		gcID     gc.XID
		width    uint16
		height   uint16
		x        int16
		y        int16
		depth    uint8
		format   uint8
		data     []byte
		wantErr  bool
	}{
		{
			name:     "small image 2x2",
			drawable: 0x100,
			gcID:     0x200,
			width:    2,
			height:   2,
			x:        10,
			y:        20,
			depth:    24,
			format:   gc.FormatZPixmap,
			data:     bytes.Repeat([]byte{0xFF, 0x00, 0x00, 0xFF}, 4), // 4 pixels ARGB
			wantErr:  false,
		},
		{
			name:     "larger image 4x4",
			drawable: 0x100,
			gcID:     0x200,
			width:    4,
			height:   4,
			x:        0,
			y:        0,
			depth:    32,
			format:   gc.FormatZPixmap,
			data:     bytes.Repeat([]byte{0x00, 0xFF, 0x00, 0xFF}, 16), // 16 pixels
			wantErr:  false,
		},
		{
			name:     "negative offset",
			drawable: 0x100,
			gcID:     0x200,
			width:    2,
			height:   2,
			x:        -5,
			y:        -10,
			depth:    24,
			format:   gc.FormatZPixmap,
			data:     bytes.Repeat([]byte{0x00, 0x00, 0xFF, 0xFF}, 4),
			wantErr:  false,
		},
		{
			name:     "empty data",
			drawable: 0x100,
			gcID:     0x200,
			width:    2,
			height:   2,
			x:        0,
			y:        0,
			depth:    24,
			format:   gc.FormatZPixmap,
			data:     []byte{},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn := &mockConnection{}

			err := gc.PutImage(conn, tc.drawable, tc.gcID, tc.width, tc.height, tc.x, tc.y, tc.depth, tc.format, tc.data)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("PutImage failed: %v", err)
			}

			if len(conn.requests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(conn.requests))
			}

			req := conn.requests[0]

			// Verify opcode (PutImage = 72)
			if req[0] != 72 {
				t.Errorf("got opcode %d, want 72", req[0])
			}

			// Verify format in data field
			if req[1] != tc.format {
				t.Errorf("got format %d, want %d", req[1], tc.format)
			}

			// Verify drawable
			gotDrawable := binary.LittleEndian.Uint32(req[4:8])
			if gotDrawable != uint32(tc.drawable) {
				t.Errorf("got drawable %#x, want %#x", gotDrawable, tc.drawable)
			}

			// Verify GC
			gotGC := binary.LittleEndian.Uint32(req[8:12])
			if gotGC != uint32(tc.gcID) {
				t.Errorf("got GC %#x, want %#x", gotGC, tc.gcID)
			}

			// Verify width
			gotWidth := binary.LittleEndian.Uint16(req[12:14])
			if gotWidth != tc.width {
				t.Errorf("got width %d, want %d", gotWidth, tc.width)
			}

			// Verify height
			gotHeight := binary.LittleEndian.Uint16(req[14:16])
			if gotHeight != tc.height {
				t.Errorf("got height %d, want %d", gotHeight, tc.height)
			}

			// Verify X position
			gotX := int16(binary.LittleEndian.Uint16(req[16:18]))
			if gotX != tc.x {
				t.Errorf("got X %d, want %d", gotX, tc.x)
			}

			// Verify Y position
			gotY := int16(binary.LittleEndian.Uint16(req[18:20]))
			if gotY != tc.y {
				t.Errorf("got Y %d, want %d", gotY, tc.y)
			}

			// Verify depth
			if req[21] != tc.depth {
				t.Errorf("got depth %d, want %d", req[21], tc.depth)
			}

			// Verify image data is present
			// Data starts at offset 24 (after header + parameters + padding)
			dataStart := 24
			if len(req) < dataStart+len(tc.data) {
				t.Errorf("request too short: got %d bytes, need at least %d", len(req), dataStart+len(tc.data))
			}
		})
	}
}

func TestGCConstants(t *testing.T) {
	tests := []struct {
		name  string
		value uint32
		want  uint32
	}{
		{"GCForeground", gc.GCForeground, 1 << 2},
		{"GCBackground", gc.GCBackground, 1 << 3},
		{"GCFunction", gc.GCFunction, 1 << 0},
		{"GCLineWidth", gc.GCLineWidth, 1 << 4},
		{"GCGraphicsExposures", gc.GCGraphicsExposures, 1 << 16},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != tc.want {
				t.Errorf("got %#x, want %#x", tc.value, tc.want)
			}
		})
	}
}

func TestImageFormatConstants(t *testing.T) {
	if gc.FormatBitmap != 0 {
		t.Errorf("FormatBitmap: got %d, want 0", gc.FormatBitmap)
	}
	if gc.FormatXYPixmap != 1 {
		t.Errorf("FormatXYPixmap: got %d, want 1", gc.FormatXYPixmap)
	}
	if gc.FormatZPixmap != 2 {
		t.Errorf("FormatZPixmap: got %d, want 2", gc.FormatZPixmap)
	}
}

func TestPutImagePadding(t *testing.T) {
	// Test that data is properly padded to 4-byte boundary
	tests := []struct {
		name    string
		dataLen int
		wantPad int
	}{
		{"aligned", 16, 0},
		{"1 byte over", 17, 3},
		{"2 bytes over", 18, 2},
		{"3 bytes over", 19, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn := &mockConnection{}
			data := make([]byte, tc.dataLen)

			err := gc.PutImage(conn, 0x100, 0x200, 2, 2, 0, 0, 24, gc.FormatZPixmap, data)
			if err != nil {
				t.Fatalf("PutImage failed: %v", err)
			}

			req := conn.requests[0]

			// Total length should be 4-byte aligned
			if len(req)%4 != 0 {
				t.Errorf("request not 4-byte aligned: length %d", len(req))
			}
		})
	}
}
