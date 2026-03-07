package wire_test

import (
	"bytes"
	"testing"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// FuzzDecodeHeader tests header decoding with arbitrary input.
func FuzzDecodeHeader(f *testing.F) {
	// Seed corpus with valid and edge-case inputs
	f.Add([]byte{0x01, 0x00, 0x00, 0x00, 0x0A, 0x00, 0x10, 0x00})
	f.Add([]byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x0F})
	f.Add([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		header, err := wire.DecodeHeader(r)
		if err != nil {
			return
		}

		if header.Size < 8 {
			t.Errorf("decoded invalid header with size %d < 8", header.Size)
		}
		if header.Size > 4096 {
			t.Errorf("decoded invalid header with size %d > 4096", header.Size)
		}
	})
}

// FuzzDecodeInt32 tests int32 decoding with arbitrary input.
func FuzzDecodeInt32(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0x2A, 0x00, 0x00, 0x00})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0x7F})
	f.Add([]byte{0x00, 0x00, 0x00, 0x80})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		value, err := wire.DecodeInt32(r)
		if err != nil {
			return
		}

		var buf bytes.Buffer
		if err := wire.EncodeInt32(&buf, value); err != nil {
			t.Errorf("failed to encode decoded value %d: %v", value, err)
			return
		}

		decoded, err := wire.DecodeInt32(&buf)
		if err != nil {
			t.Errorf("failed to decode encoded value %d: %v", value, err)
			return
		}

		if decoded != value {
			t.Errorf("roundtrip mismatch: original %d, roundtrip %d", value, decoded)
		}
	})
}

// FuzzDecodeUint32 tests uint32 decoding with arbitrary input.
func FuzzDecodeUint32(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0x2A, 0x00, 0x00, 0x00})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})

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

// FuzzDecodeFixed tests fixed-point decoding with arbitrary input.
func FuzzDecodeFixed(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0x00, 0x01, 0x00, 0x00})
	f.Add([]byte{0x80, 0x00, 0x00, 0x00})
	f.Add([]byte{0x00, 0xFF, 0xFF, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		value, err := wire.DecodeFixed(r)
		if err != nil {
			return
		}

		var buf bytes.Buffer
		if err := wire.EncodeFixed(&buf, value); err != nil {
			t.Errorf("failed to encode decoded value %f: %v", value, err)
			return
		}

		decoded, err := wire.DecodeFixed(&buf)
		if err != nil {
			t.Errorf("failed to decode encoded value %f: %v", value, err)
			return
		}

		const epsilon = 0.01
		if decoded < value-epsilon || decoded > value+epsilon {
			t.Errorf("roundtrip mismatch: original %f, roundtrip %f", value, decoded)
		}
	})
}

// FuzzDecodeString tests string decoding with arbitrary input.
func FuzzDecodeString(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0x06, 0x00, 0x00, 0x00, 'h', 'e', 'l', 'l', 'o', 0x00, 0x00, 0x00})
	f.Add([]byte{0x02, 0x00, 0x00, 0x00, 'a', 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		str, err := wire.DecodeString(r)
		if err != nil {
			return
		}

		var buf bytes.Buffer
		if err := wire.EncodeString(&buf, str); err != nil {
			t.Errorf("failed to encode decoded string %q: %v", str, err)
			return
		}

		decoded, err := wire.DecodeString(&buf)
		if err != nil {
			t.Errorf("failed to decode encoded string %q: %v", str, err)
			return
		}

		if decoded != str {
			t.Errorf("roundtrip mismatch: original %q, roundtrip %q", str, decoded)
		}
	})
}

// FuzzDecodeArray tests array decoding with arbitrary input.
func FuzzDecodeArray(f *testing.F) {
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0x01, 0x00, 0x00, 0x00, 0x42, 0x00, 0x00, 0x00})
	f.Add([]byte{0x04, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04})
	f.Add([]byte{0x05, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		arr, err := wire.DecodeArray(r)
		if err != nil {
			return
		}

		var buf bytes.Buffer
		if err := wire.EncodeArray(&buf, arr); err != nil {
			t.Errorf("failed to encode decoded array %v: %v", arr, err)
			return
		}

		decoded, err := wire.DecodeArray(&buf)
		if err != nil {
			t.Errorf("failed to decode encoded array %v: %v", arr, err)
			return
		}

		if !bytes.Equal(decoded, arr) {
			t.Errorf("roundtrip mismatch: original %v, roundtrip %v", arr, decoded)
		}
	})
}

// FuzzEncodeMessage tests message encoding with arbitrary arguments.
func FuzzEncodeMessage(f *testing.F) {
	// Seed with valid message structures
	f.Add(uint32(1), uint16(0), []byte{})
	f.Add(uint32(2), uint16(1), []byte{0x2A, 0x00, 0x00, 0x00})
	f.Add(uint32(3), uint16(2), []byte{0x05, 0x00, 0x00, 0x00, 't', 'e', 's', 't', 0x00, 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, objectID uint32, opcode uint16, argData []byte) {
		msg := &wire.Message{
			Header: wire.Header{
				ObjectID: objectID,
				Opcode:   opcode,
				Size:     8,
			},
			Args: []wire.Argument{},
		}

		data, fds, err := wire.EncodeMessage(msg)
		if err != nil {
			return
		}

		if len(data) < 8 {
			t.Errorf("encoded message too short: %d bytes", len(data))
		}

		if len(fds) > 0 {
			t.Errorf("unexpected file descriptors in empty message: %v", fds)
		}

		r := bytes.NewReader(data)
		header, err := wire.DecodeHeader(r)
		if err != nil {
			t.Errorf("failed to decode encoded message header: %v", err)
			return
		}

		if header.ObjectID != objectID {
			t.Errorf("objectID mismatch: encoded %d, decoded %d", objectID, header.ObjectID)
		}
		if header.Opcode != opcode {
			t.Errorf("opcode mismatch: encoded %d, decoded %d", opcode, header.Opcode)
		}
	})
}
