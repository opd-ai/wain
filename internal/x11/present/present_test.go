package present

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

// mockConnection simulates an X11 connection for testing.
type mockConnection struct {
	extensionOpcode uint8
	reply           []byte
	replyErr        error
	lastRequest     []byte
	requestErr      error
}

func (m *mockConnection) AllocXID() (XID, error) {
	return 0x12345678, nil
}

func (m *mockConnection) SendRequest(buf []byte) error {
	m.lastRequest = append([]byte(nil), buf...)
	return m.requestErr
}

func (m *mockConnection) SendRequestAndReply(req []byte) ([]byte, error) {
	m.lastRequest = append([]byte(nil), req...)
	if m.replyErr != nil {
		return nil, m.replyErr
	}
	return m.reply, nil
}

func (m *mockConnection) ExtensionOpcode(name string) (uint8, error) {
	if name == ExtensionName {
		return m.extensionOpcode, nil
	}
	return 0, errors.New("extension not found")
}

// TestQueryExtension validates Present extension query structure.
func TestQueryExtension(t *testing.T) {
	tests := []struct {
		name      string
		mockReply []byte
		wantMajor uint32
		wantMinor uint32
		wantErr   bool
		wantAsync bool
	}{
		{
			name: "Present 1.2 supported",
			mockReply: []byte{
				1, 0, 0, 0, // type=1 (reply), pad, sequence
				0, 0, 0, 0, // length=0 (no extra data)
				1, 0, 0, 0, // major version = 1
				2, 0, 0, 0, // minor version = 2
				0, 0, 0, 0, 0, 0, 0, 0, // padding
				0, 0, 0, 0, 0, 0, 0, 0, // padding
			},
			wantMajor: 1,
			wantMinor: 2,
			wantErr:   false,
			wantAsync: true,
		},
		{
			name: "Present 1.0 supported (no async)",
			mockReply: []byte{
				1, 0, 0, 0, // type=1 (reply)
				0, 0, 0, 0, // length=0
				1, 0, 0, 0, // major version = 1
				0, 0, 0, 0, // minor version = 0
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
			},
			wantMajor: 1,
			wantMinor: 0,
			wantErr:   false,
			wantAsync: false,
		},
		{
			name: "Present 0.x too old",
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
		{
			name: "Short reply",
			mockReply: []byte{
				1, 0, 0, 0,
				0, 0, 0, 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &mockConnection{
				extensionOpcode: 140, // arbitrary Present opcode
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

			if ext.SupportsAsync() != tt.wantAsync {
				t.Errorf("SupportsAsync() = %v, want %v", ext.SupportsAsync(), tt.wantAsync)
			}
		})
	}
}

// TestPresentPixmap validates PresentPixmap request encoding.
func TestPresentPixmap(t *testing.T) {
	ext := &Extension{
		baseOpcode:   140,
		majorVersion: 1,
		minorVersion: 2,
	}

	conn := &mockConnection{}

	window := XID(0x400001)
	pixmap := XID(0x500001)
	serial := uint32(42)
	targetMSC := uint64(1000)

	err := ext.PresentPixmap(conn, window, pixmap, serial,
		0, 0, // validRegion, updateRegion
		0, 0, // xOff, yOff
		targetMSC, 0, 0, // targetMSC, divisor, remainder
		PresentOptionNone,
	)
	if err != nil {
		t.Fatalf("PresentPixmap() error = %v", err)
	}

	// Verify request structure
	req := conn.lastRequest
	if len(req) != 72 { // 18 * 4 bytes
		t.Errorf("PresentPixmap request length = %d, want 72", len(req))
	}

	// Check opcode
	if req[0] != ext.baseOpcode+PresentPixmap {
		t.Errorf("Request opcode = %d, want %d", req[0], ext.baseOpcode+PresentPixmap)
	}

	// Check window XID
	gotWindow := binary.LittleEndian.Uint32(req[4:8])
	if gotWindow != uint32(window) {
		t.Errorf("Window XID = 0x%x, want 0x%x", gotWindow, window)
	}

	// Check pixmap XID
	gotPixmap := binary.LittleEndian.Uint32(req[8:12])
	if gotPixmap != uint32(pixmap) {
		t.Errorf("Pixmap XID = 0x%x, want 0x%x", gotPixmap, pixmap)
	}

	// Check serial
	gotSerial := binary.LittleEndian.Uint32(req[12:16])
	if gotSerial != serial {
		t.Errorf("Serial = %d, want %d", gotSerial, serial)
	}

	// Check target MSC (at offset 44 bytes: header + 8*uint32 + 2*int16 + 3*uint32)
	gotMSC := binary.LittleEndian.Uint64(req[44:52])
	if gotMSC != targetMSC {
		t.Errorf("Target MSC = %d, want %d", gotMSC, targetMSC)
	}
}

// TestPresentPixmapError validates error handling.
func TestPresentPixmapError(t *testing.T) {
	ext := &Extension{
		baseOpcode:   140,
		majorVersion: 1,
		minorVersion: 0,
	}

	conn := &mockConnection{
		requestErr: errors.New("connection closed"),
	}

	err := ext.PresentPixmap(conn, 1, 2, 0, 0, 0, 0, 0, 0, 0, 0, PresentOptionNone)

	if err == nil {
		t.Error("PresentPixmap() expected error, got nil")
	}

	if !errors.Is(err, ErrPresentFailed) {
		t.Errorf("Error type = %v, want ErrPresentFailed", err)
	}
}

// TestSelectInput validates SelectInput request encoding.
func TestSelectInput(t *testing.T) {
	ext := &Extension{
		baseOpcode:   140,
		majorVersion: 1,
		minorVersion: 0,
	}

	conn := &mockConnection{}

	eid := XID(0x600001)
	window := XID(0x400001)
	eventMask := uint32(PresentEventMaskCompleteNotify | PresentEventMaskIdleNotify)

	err := ext.SelectInput(conn, eid, window, eventMask)
	if err != nil {
		t.Fatalf("SelectInput() error = %v", err)
	}

	// Verify request structure
	req := conn.lastRequest
	if len(req) != 16 { // 4 * 4 bytes
		t.Errorf("SelectInput request length = %d, want 16", len(req))
	}

	// Check opcode
	if req[0] != ext.baseOpcode+PresentSelectInput {
		t.Errorf("Request opcode = %d, want %d", req[0], ext.baseOpcode+PresentSelectInput)
	}

	// Check event ID
	gotEID := binary.LittleEndian.Uint32(req[4:8])
	if gotEID != uint32(eid) {
		t.Errorf("Event ID = 0x%x, want 0x%x", gotEID, eid)
	}

	// Check window
	gotWindow := binary.LittleEndian.Uint32(req[8:12])
	if gotWindow != uint32(window) {
		t.Errorf("Window = 0x%x, want 0x%x", gotWindow, window)
	}

	// Check event mask
	gotMask := binary.LittleEndian.Uint32(req[12:16])
	if gotMask != eventMask {
		t.Errorf("Event mask = 0x%x, want 0x%x", gotMask, eventMask)
	}
}

// TestNotifyMSC validates NotifyMSC request encoding.
func TestNotifyMSC(t *testing.T) {
	ext := &Extension{
		baseOpcode:   140,
		majorVersion: 1,
		minorVersion: 0,
	}

	conn := &mockConnection{}

	window := XID(0x400001)
	serial := uint32(99)
	targetMSC := uint64(5000)

	err := ext.NotifyMSC(conn, window, serial, targetMSC, 0, 0)
	if err != nil {
		t.Fatalf("NotifyMSC() error = %v", err)
	}

	// Verify request structure
	req := conn.lastRequest
	if len(req) != 40 { // 10 * 4 bytes
		t.Errorf("NotifyMSC request length = %d, want 40", len(req))
	}

	// Check opcode
	if req[0] != ext.baseOpcode+PresentNotifyMSC {
		t.Errorf("Request opcode = %d, want %d", req[0], ext.baseOpcode+PresentNotifyMSC)
	}

	// Check window
	gotWindow := binary.LittleEndian.Uint32(req[4:8])
	if gotWindow != uint32(window) {
		t.Errorf("Window = 0x%x, want 0x%x", gotWindow, window)
	}

	// Check serial
	gotSerial := binary.LittleEndian.Uint32(req[8:12])
	if gotSerial != serial {
		t.Errorf("Serial = %d, want %d", gotSerial, serial)
	}

	// Check target MSC (at offset 16)
	gotMSC := binary.LittleEndian.Uint64(req[16:24])
	if gotMSC != targetMSC {
		t.Errorf("Target MSC = %d, want %d", gotMSC, targetMSC)
	}
}

// TestParseCompleteNotify validates CompleteNotify event parsing.
func TestParseCompleteNotify(t *testing.T) {
	tests := []struct {
		name       string
		eventData  []byte
		wantSerial uint32
		wantKind   CompleteKind
		wantMode   CompleteMode
		wantUST    uint64
		wantMSC    uint64
		wantErr    bool
	}{
		{
			name: "Valid flip event",
			eventData: makeCompleteNotifyEvent(
				CompleteKindPixmap,
				CompleteModeFlip,
				42,         // serial
				1234567890, // UST
				100,        // MSC
			),
			wantSerial: 42,
			wantKind:   CompleteKindPixmap,
			wantMode:   CompleteModeFlip,
			wantUST:    1234567890,
			wantMSC:    100,
			wantErr:    false,
		},
		{
			name: "Copy mode",
			eventData: makeCompleteNotifyEvent(
				CompleteKindPixmap,
				CompleteModeCopy,
				99,
				9999999999,
				200,
			),
			wantSerial: 99,
			wantKind:   CompleteKindPixmap,
			wantMode:   CompleteModeCopy,
			wantUST:    9999999999,
			wantMSC:    200,
			wantErr:    false,
		},
		{
			name: "NotifyMSC event",
			eventData: makeCompleteNotifyEvent(
				CompleteKindNotifyMSC,
				CompleteModeSkip,
				77,
				555555,
				0,
			),
			wantSerial: 77,
			wantKind:   CompleteKindNotifyMSC,
			wantMode:   CompleteModeSkip,
			wantUST:    555555,
			wantMSC:    0,
			wantErr:    false,
		},
		{
			name:      "Short event data",
			eventData: []byte{1, 2, 3, 4, 5},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt, err := ParseCompleteNotify(tt.eventData)
			if tt.wantErr {
				if err == nil {
					t.Error("ParseCompleteNotify() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseCompleteNotify() unexpected error: %v", err)
			}

			if evt.Serial != tt.wantSerial {
				t.Errorf("Serial = %d, want %d", evt.Serial, tt.wantSerial)
			}

			if evt.Kind != tt.wantKind {
				t.Errorf("Kind = %v, want %v", evt.Kind, tt.wantKind)
			}

			if evt.Mode != tt.wantMode {
				t.Errorf("Mode = %v, want %v", evt.Mode, tt.wantMode)
			}

			if evt.UST != tt.wantUST {
				t.Errorf("UST = %d, want %d", evt.UST, tt.wantUST)
			}

			if evt.MSC != tt.wantMSC {
				t.Errorf("MSC = %d, want %d", evt.MSC, tt.wantMSC)
			}
		})
	}
}

// TestParseIdleNotify validates IdleNotify event parsing.
func TestParseIdleNotify(t *testing.T) {
	tests := []struct {
		name       string
		eventData  []byte
		wantPixmap XID
		wantErr    bool
	}{
		{
			name:       "Valid idle event",
			eventData:  makeIdleNotifyEvent(0x500001),
			wantPixmap: 0x500001,
			wantErr:    false,
		},
		{
			name:       "Different pixmap",
			eventData:  makeIdleNotifyEvent(0xABCDEF),
			wantPixmap: 0xABCDEF,
			wantErr:    false,
		},
		{
			name:      "Short event data",
			eventData: []byte{1, 2, 3},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt, err := ParseIdleNotify(tt.eventData)
			if tt.wantErr {
				if err == nil {
					t.Error("ParseIdleNotify() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseIdleNotify() unexpected error: %v", err)
			}

			if evt.Pixmap != tt.wantPixmap {
				t.Errorf("Pixmap = 0x%x, want 0x%x", evt.Pixmap, tt.wantPixmap)
			}
		})
	}
}

// makeCompleteNotifyEvent constructs a test CompleteNotify event.
func makeCompleteNotifyEvent(kind CompleteKind, mode CompleteMode, serial uint32, ust, msc uint64) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 40))

	// Event header
	buf.WriteByte(35)             // type (generic event)
	buf.WriteByte(0)              // extension
	buf.WriteByte(0)              // sequence low
	buf.WriteByte(0)              // sequence high
	buf.Write([]byte{0, 0, 0, 0}) // length

	// Event body
	buf.WriteByte(byte(kind))
	buf.WriteByte(byte(mode))
	buf.WriteByte(0) // pad
	buf.WriteByte(0) // pad
	binary.Write(buf, binary.LittleEndian, serial)
	binary.Write(buf, binary.LittleEndian, uint32(0x400001)) // window
	binary.Write(buf, binary.LittleEndian, uint32(0x500001)) // pixmap
	binary.Write(buf, binary.LittleEndian, ust)
	binary.Write(buf, binary.LittleEndian, msc)

	return buf.Bytes()
}

// makeIdleNotifyEvent constructs a test IdleNotify event.
func makeIdleNotifyEvent(pixmap XID) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, 32))

	// Event header
	buf.WriteByte(35)             // type (generic event)
	buf.WriteByte(0)              // extension
	buf.WriteByte(0)              // sequence low
	buf.WriteByte(0)              // sequence high
	buf.Write([]byte{0, 0, 0, 0}) // length

	// Event body
	binary.Write(buf, binary.LittleEndian, uint32(0x600001)) // event ID
	binary.Write(buf, binary.LittleEndian, uint32(0x400001)) // window
	binary.Write(buf, binary.LittleEndian, uint32(0))        // serial
	binary.Write(buf, binary.LittleEndian, uint32(pixmap))   // pixmap
	binary.Write(buf, binary.LittleEndian, uint32(0))        // idle_fence
	buf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})                // pad

	return buf.Bytes()
}

// TestExtensionNotFound validates error handling when extension is not available.
func TestExtensionNotFound(t *testing.T) {
	conn := &mockConnection{
		replyErr: errors.New("extension not found"),
	}

	_, err := QueryExtension(conn)

	if err == nil {
		t.Error("QueryExtension() expected error, got nil")
	}
}
