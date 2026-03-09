package wire_test

import (
	"bytes"
	"testing"

	"github.com/opd-ai/wain/internal/x11/wire"
)

func TestEncodeRequestHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		opcode uint8
		data   uint8
		length uint16
		want   []byte
	}{
		{
			name:   "CreateWindow",
			opcode: wire.OpcodeCreateWindow,
			data:   24,
			length: 8,
			want:   []byte{1, 24, 8, 0},
		},
		{
			name:   "MapWindow",
			opcode: wire.OpcodeMapWindow,
			data:   0,
			length: 2,
			want:   []byte{8, 0, 2, 0},
		},
		{
			name:   "zero length",
			opcode: 42,
			data:   0,
			length: 0,
			want:   []byte{42, 0, 0, 0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := wire.EncodeRequestHeader(&buf, tc.opcode, tc.data, tc.length)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEncodeUint32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value uint32
		want  []byte
	}{
		{
			name:  "zero",
			value: 0,
			want:  []byte{0, 0, 0, 0},
		},
		{
			name:  "small value",
			value: 42,
			want:  []byte{42, 0, 0, 0},
		},
		{
			name:  "large value",
			value: 0x12345678,
			want:  []byte{0x78, 0x56, 0x34, 0x12},
		},
		{
			name:  "max value",
			value: 0xFFFFFFFF,
			want:  []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := wire.EncodeUint32(&buf, tc.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEncodeUint16(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value uint16
		want  []byte
	}{
		{
			name:  "zero",
			value: 0,
			want:  []byte{0, 0},
		},
		{
			name:  "small value",
			value: 256,
			want:  []byte{0, 1},
		},
		{
			name:  "max value",
			value: 0xFFFF,
			want:  []byte{0xFF, 0xFF},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := wire.EncodeUint16(&buf, tc.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEncodeInt16(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value int16
		want  []byte
	}{
		{
			name:  "zero",
			value: 0,
			want:  []byte{0, 0},
		},
		{
			name:  "positive",
			value: 100,
			want:  []byte{100, 0},
		},
		{
			name:  "negative",
			value: -100,
			want:  []byte{0x9C, 0xFF},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := wire.EncodeInt16(&buf, tc.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDecodeUint32(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    uint32
		wantErr bool
	}{
		{
			name:    "zero",
			input:   []byte{0, 0, 0, 0},
			want:    0,
			wantErr: false,
		},
		{
			name:    "small value",
			input:   []byte{42, 0, 0, 0},
			want:    42,
			wantErr: false,
		},
		{
			name:    "large value",
			input:   []byte{0x78, 0x56, 0x34, 0x12},
			want:    0x12345678,
			wantErr: false,
		},
		{
			name:    "too short",
			input:   []byte{1, 2},
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty",
			input:   []byte{},
			want:    0,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			got, err := wire.DecodeUint32(r)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDecodeReplyHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    wire.ReplyHeader
		wantErr bool
	}{
		{
			name: "valid reply",
			input: []byte{
				0x01, 0x00, 0x05, 0x00, // type=1, data=0, sequence=5
				0x10, 0x00, 0x00, 0x00, // length=16 (64 extra bytes)
				0, 0, 0, 0, 0, 0, 0, 0, // inline data (24 bytes)
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
			},
			want: wire.ReplyHeader{
				Type:     wire.MessageTypeReply,
				Data:     0,
				Sequence: 5,
				Length:   16,
			},
			wantErr: false,
		},
		{
			name:    "too short",
			input:   []byte{0x01, 0x00},
			want:    wire.ReplyHeader{},
			wantErr: true,
		},
		{
			name:    "empty",
			input:   []byte{},
			want:    wire.ReplyHeader{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			got, data, err := wire.DecodeReplyHeader(r)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}

			if len(data) != 24 {
				t.Errorf("got data length %d, want 24", len(data))
			}
		})
	}
}

func TestDecodeErrorHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    wire.ErrorHeader
		wantErr bool
	}{
		{
			name: "valid error",
			input: []byte{
				0x00, 0x03, 0x05, 0x00, // type=0, code=3 (BadWindow), sequence=5
				0x01, 0x00, 0x00, 0x20, // bad_value=0x20000001
				0x00, 0x00, // minor_opcode=0
				0x01, // major_opcode=1 (CreateWindow)
				0,    // padding
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
			want: wire.ErrorHeader{
				Type:        wire.MessageTypeError,
				Code:        3,
				Sequence:    5,
				BadValue:    0x20000001,
				MinorOpcode: 0,
				MajorOpcode: 1,
			},
			wantErr: false,
		},
		{
			name:    "too short",
			input:   []byte{0x00, 0x03},
			want:    wire.ErrorHeader{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			got, err := wire.DecodeErrorHeader(r)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestPad(t *testing.T) {
	t.Parallel()
	tests := []struct {
		length int
		want   int
	}{
		{0, 0},
		{1, 3},
		{2, 2},
		{3, 1},
		{4, 0},
		{5, 3},
		{8, 0},
		{9, 3},
	}

	for _, tc := range tests {
		got := wire.Pad(tc.length)
		if got != tc.want {
			t.Errorf("Pad(%d) = %d, want %d", tc.length, got, tc.want)
		}
	}
}

func TestEncodeSetupRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  wire.SetupRequest
	}{
		{
			name: "minimal setup",
			req: wire.SetupRequest{
				ByteOrder:            wire.ByteOrderLSB,
				ProtocolMajorVersion: wire.ProtocolMajorVersion,
				ProtocolMinorVersion: wire.ProtocolMinorVersion,
				AuthName:             "",
				AuthData:             nil,
			},
		},
		{
			name: "with auth",
			req: wire.SetupRequest{
				ByteOrder:            wire.ByteOrderLSB,
				ProtocolMajorVersion: wire.ProtocolMajorVersion,
				ProtocolMinorVersion: wire.ProtocolMinorVersion,
				AuthName:             "MIT-MAGIC-COOKIE-1",
				AuthData:             []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := wire.EncodeSetupRequest(&buf, tc.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify message is properly padded to 4-byte alignment
			if buf.Len()%4 != 0 {
				t.Errorf("message length %d is not 4-byte aligned", buf.Len())
			}

			// Verify byte order
			if buf.Bytes()[0] != tc.req.ByteOrder {
				t.Errorf("got byte order %x, want %x", buf.Bytes()[0], tc.req.ByteOrder)
			}
		})
	}
}

func TestDecodeSetupReply(t *testing.T) {
	t.Parallel()
	// Test failure response
	t.Run("failed setup", func(t *testing.T) {
		// Minimal failure response
		input := []byte{
			0x00,       // status=0 (failed)
			0x06,       // reason length=6
			0x0B, 0x00, // protocol major=11
			0x00, 0x00, // protocol minor=0
			0x00, 0x00, // data length=0
			'f', 'a', 'i', 'l', 'e', 'd', // reason string
		}

		r := bytes.NewReader(input)
		reply, err := wire.DecodeSetupReply(r)

		if err == nil {
			t.Fatal("expected error for failed setup")
		}

		if reply.Status != wire.SetupStatusFailed {
			t.Errorf("got status %d, want %d", reply.Status, wire.SetupStatusFailed)
		}
	})

	// Test success response would require a full 228+ byte message
	// which is complex to construct. In practice, this is tested
	// via integration tests with a real X server.
}

func TestDecodeEventHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    wire.EventHeader
		wantErr bool
	}{
		{
			name: "KeyPress event",
			input: append(
				[]byte{
					0x02,       // type=2 (KeyPress)
					0x41,       // detail='A'
					0x10, 0x00, // sequence=16
				},
				make([]byte, 28)..., // 28 bytes event data
			),
			want: wire.EventHeader{
				Type:     2,
				Detail:   0x41,
				Sequence: 16,
			},
			wantErr: false,
		},
		{
			name: "SendEvent flag cleared",
			input: append(
				[]byte{
					0x82,       // type with SendEvent flag (0x80)
					0x00,       // detail
					0x00, 0x00, // sequence=0
				},
				make([]byte, 28)...,
			),
			want: wire.EventHeader{
				Type:     2, // SendEvent flag should be cleared
				Detail:   0,
				Sequence: 0,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			got, data, err := wire.DecodeEventHeader(r)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}

			if len(data) != 28 {
				t.Errorf("got data length %d, want 28", len(data))
			}
		})
	}
}

func TestEncodePadding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		n    int
		want int
	}{
		{"zero", 0, 0},
		{"one", 1, 1},
		{"four", 4, 4},
		{"negative", -1, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := wire.EncodePadding(&buf, tc.n)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if buf.Len() != tc.want {
				t.Errorf("got length %d, want %d", buf.Len(), tc.want)
			}

			// Verify all bytes are zero
			for _, b := range buf.Bytes() {
				if b != 0 {
					t.Errorf("expected zero byte, got %d", b)
				}
			}
		})
	}
}

func TestEncodeDrawableGeometry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		drawable uint32
		gc       uint32
		width    uint16
		height   uint16
		x        int16
		y        int16
		want     []byte
	}{
		{
			name:     "typical PutImage parameters",
			drawable: 0x12345678,
			gc:       0xABCDEF00,
			width:    640,
			height:   480,
			x:        10,
			y:        20,
			want: []byte{
				0x78, 0x56, 0x34, 0x12, // drawable (little-endian)
				0x00, 0xEF, 0xCD, 0xAB, // gc (little-endian)
				0x80, 0x02, // width=640 (little-endian)
				0xE0, 0x01, // height=480 (little-endian)
				0x0A, 0x00, // x=10 (little-endian)
				0x14, 0x00, // y=20 (little-endian)
			},
		},
		{
			name:     "negative coordinates",
			drawable: 1000,
			gc:       2000,
			width:    100,
			height:   200,
			x:        -50,
			y:        -75,
			want: []byte{
				0xE8, 0x03, 0x00, 0x00, // drawable=1000
				0xD0, 0x07, 0x00, 0x00, // gc=2000
				0x64, 0x00, // width=100
				0xC8, 0x00, // height=200
				0xCE, 0xFF, // x=-50 (two's complement)
				0xB5, 0xFF, // y=-75 (two's complement)
			},
		},
		{
			name:     "zero values",
			drawable: 0,
			gc:       0,
			width:    0,
			height:   0,
			x:        0,
			y:        0,
			want: []byte{
				0x00, 0x00, 0x00, 0x00, // drawable=0
				0x00, 0x00, 0x00, 0x00, // gc=0
				0x00, 0x00, // width=0
				0x00, 0x00, // height=0
				0x00, 0x00, // x=0
				0x00, 0x00, // y=0
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := wire.EncodeDrawableGeometry(&buf, tc.drawable, tc.gc, tc.width, tc.height, tc.x, tc.y)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := buf.Bytes()
			if !bytes.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}

			if len(got) != 16 {
				t.Errorf("got length %d, want 16", len(got))
			}
		})
	}
}
