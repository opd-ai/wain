package wire_test

import (
	"bytes"
	"testing"

	"github.com/opd-ai/wain/internal/x11/wire"
)

// FuzzDecodeUint32 tests uint32 decoding with arbitrary input.
func FuzzDecodeUint32(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0x2A, 0x00, 0x00, 0x00})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0x7F})
	f.Add([]byte{0x00, 0x00, 0x00, 0x80})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		value, err := wire.DecodeUint32(r)
		if err != nil {
			return
		}

		var buf bytes.Buffer
		if err := wire.EncodeUint32(&buf, value); err != nil {
			t.Errorf("failed to encode decoded value %d: %v", value, err)
			return
		}

		decoded, err := wire.DecodeUint32(&buf)
		if err != nil {
			t.Errorf("failed to decode encoded value %d: %v", value, err)
			return
		}

		if decoded != value {
			t.Errorf("roundtrip mismatch: original %d, roundtrip %d", value, decoded)
		}
	})
}

// FuzzDecodeUint16 tests uint16 decoding with arbitrary input.
func FuzzDecodeUint16(f *testing.F) {
	f.Add([]byte{0x00, 0x00})
	f.Add([]byte{0x2A, 0x00})
	f.Add([]byte{0xFF, 0xFF})
	f.Add([]byte{0xFF, 0x7F})
	f.Add([]byte{0x00, 0x80})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		value, err := wire.DecodeUint16(r)
		if err != nil {
			return
		}

		var buf bytes.Buffer
		if err := wire.EncodeUint16(&buf, value); err != nil {
			t.Errorf("failed to encode decoded value %d: %v", value, err)
			return
		}

		decoded, err := wire.DecodeUint16(&buf)
		if err != nil {
			t.Errorf("failed to decode encoded value %d: %v", value, err)
			return
		}

		if decoded != value {
			t.Errorf("roundtrip mismatch: original %d, roundtrip %d", value, decoded)
		}
	})
}

// FuzzDecodeUint8 tests uint8 decoding with arbitrary input.
func FuzzDecodeUint8(f *testing.F) {
	f.Add([]byte{0x00})
	f.Add([]byte{0x2A})
	f.Add([]byte{0xFF})
	f.Add([]byte{0x7F})
	f.Add([]byte{0x80})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		value, err := wire.DecodeUint8(r)
		if err != nil {
			return
		}

		// Verify we got a valid uint8
		if value > 255 {
			t.Errorf("decoded value %d exceeds uint8 max", value)
		}
	})
}

// FuzzEncodeInt16 tests int16 encoding with arbitrary input.
func FuzzEncodeInt16(f *testing.F) {
	f.Add(int16(0))
	f.Add(int16(42))
	f.Add(int16(-1))
	f.Add(int16(32767))
	f.Add(int16(-32768))

	f.Fuzz(func(t *testing.T, value int16) {
		var buf bytes.Buffer
		if err := wire.EncodeInt16(&buf, value); err != nil {
			t.Errorf("failed to encode value %d: %v", value, err)
			return
		}

		// Verify the encoded data is 2 bytes
		if buf.Len() != 2 {
			t.Errorf("encoded int16 should be 2 bytes, got %d", buf.Len())
		}
	})
}

// FuzzDecodeReplyHeader tests reply header decoding with arbitrary input.
func FuzzDecodeReplyHeader(f *testing.F) {
	// Seed with valid reply headers
	f.Add([]byte{
		0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, // Header: Type=1, Data=0, Seq=1, Len=0
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Inline data (24 bytes)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	})
	f.Add([]byte{
		0x01, 0xFF, 0xFF, 0xFF, 0x01, 0x00, 0x00, 0x00, // Header with length=1
		0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		header, inlineData, err := wire.DecodeReplyHeader(r)
		if err != nil {
			return
		}

		// Validate header constraints
		if header.Type != 0 && header.Type != 1 {
			t.Errorf("decoded invalid message type %d (expected 0 or 1)", header.Type)
		}

		if len(inlineData) != 24 {
			t.Errorf("inline data should be 24 bytes, got %d", len(inlineData))
		}

		// Verify length is reasonable (X11 length is in 4-byte units)
		if header.Length > 16384 {
			t.Errorf("decoded unreasonably large length %d (max 16384)", header.Length)
		}
	})
}

// FuzzDecodeEventHeader tests event header decoding with arbitrary input.
func FuzzDecodeEventHeader(f *testing.F) {
	// Seed with valid event headers (32 bytes total)
	f.Add([]byte{
		0x02, 0x00, 0x01, 0x00, // Type=2 (KeyPress), Detail=0, Seq=1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	})
	f.Add([]byte{
		0x04, 0x01, 0xFF, 0xFF, // Type=4 (ButtonPress), Detail=1, Seq=65535
		0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		header, eventData, err := wire.DecodeEventHeader(r)
		if err != nil {
			return
		}

		// Validate event header constraints
		if header.Type > 127 {
			t.Errorf("decoded invalid event type %d (max 127)", header.Type)
		}

		if len(eventData) != 28 {
			t.Errorf("event data should be 28 bytes, got %d", len(eventData))
		}
	})
}

// FuzzEncodeRequestHeader tests request header encoding with arbitrary data.
func FuzzEncodeRequestHeader(f *testing.F) {
	f.Add(uint8(1), uint8(0), uint16(1))
	f.Add(uint8(8), uint8(0), uint16(2))
	f.Add(uint8(72), uint8(2), uint16(100))

	f.Fuzz(func(t *testing.T, opcode uint8, data uint8, length uint16) {
		var buf bytes.Buffer
		if err := wire.EncodeRequestHeader(&buf, opcode, data, length); err != nil {
			t.Errorf("failed to encode request header: %v", err)
			return
		}

		// Verify the encoded header is 4 bytes
		if buf.Len() != 4 {
			t.Errorf("encoded request header should be 4 bytes, got %d", buf.Len())
		}
	})
}
