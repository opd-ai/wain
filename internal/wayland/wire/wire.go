// Package wire implements the Wayland wire protocol format.
//
// The Wayland wire protocol uses a simple binary format for messages:
// - Message header: object_id (uint32), opcode (uint16), message_size (uint16)
// - Arguments: variable-length sequence of typed arguments
//
// Argument types supported:
// - int32: signed 32-bit integer
// - uint32: unsigned 32-bit integer
// - fixed: fixed-point decimal (Q24.8)
// - string: UTF-8 null-terminated string with length prefix
// - object: reference to another object (uint32 object_id)
// - new_id: newly created object (uint32 object_id)
// - array: byte array with length prefix
// - fd: file descriptor (passed via SCM_RIGHTS, placeholder in message)
//
// All integers are little-endian. Strings and arrays are padded to 4-byte alignment.
//
// Reference: https://wayland.freedesktop.org/docs/html/ch04.html#sect-Protocol-Wire-Format
package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidHeader is returned when a message header is malformed.
	ErrInvalidHeader = errors.New("wire: invalid message header")

	// ErrInvalidArgument is returned when an argument cannot be decoded.
	ErrInvalidArgument = errors.New("wire: invalid argument")

	// ErrMessageTooShort is returned when a message is shorter than the declared size.
	ErrMessageTooShort = errors.New("wire: message too short")

	// ErrMessageTooLong is returned when a message is longer than the declared size.
	ErrMessageTooLong = errors.New("wire: message too long")
)

const (
	// HeaderSize is the size of a Wayland message header in bytes.
	HeaderSize = 8

	// MinMessageSize is the minimum valid message size (header only).
	MinMessageSize = HeaderSize

	// MaxMessageSize is the maximum message size (4KB for safety).
	MaxMessageSize = 4096
)

// Header represents a Wayland message header.
type Header struct {
	ObjectID uint32 // Object ID this message is for
	Opcode   uint16 // Opcode identifying the message type
	Size     uint16 // Total message size including header
}

// Message represents a complete Wayland wire protocol message.
type Message struct {
	Header Header
	Args   []Argument
}

// ArgumentType identifies the type of a Wayland argument.
type ArgumentType uint8

const (
	ArgTypeInt32  ArgumentType = 0
	ArgTypeUint32 ArgumentType = 1
	ArgTypeFixed  ArgumentType = 2
	ArgTypeString ArgumentType = 3
	ArgTypeObject ArgumentType = 4
	ArgTypeNewID  ArgumentType = 5
	ArgTypeArray  ArgumentType = 6
	ArgTypeFD     ArgumentType = 7
)

// Argument represents a single argument in a Wayland message.
type Argument struct {
	Type  ArgumentType
	Value interface{}
}

// DecodeHeader reads a message header from r.
func DecodeHeader(r io.Reader) (Header, error) {
	var h Header
	var buf [HeaderSize]byte

	if _, err := io.ReadFull(r, buf[:]); err != nil {
		if err == io.EOF {
			return h, io.EOF
		}
		return h, fmt.Errorf("%w: %v", ErrInvalidHeader, err)
	}

	h.ObjectID = binary.LittleEndian.Uint32(buf[0:4])
	combined := binary.LittleEndian.Uint32(buf[4:8])
	h.Size = uint16(combined >> 16)
	h.Opcode = uint16(combined & 0xFFFF)

	if h.Size < MinMessageSize || h.Size > MaxMessageSize {
		return h, fmt.Errorf("%w: size %d out of range", ErrInvalidHeader, h.Size)
	}

	return h, nil
}

// EncodeHeader writes a message header to w.
func EncodeHeader(w io.Writer, h Header) error {
	if h.Size < MinMessageSize || h.Size > MaxMessageSize {
		return fmt.Errorf("%w: size %d out of range", ErrInvalidHeader, h.Size)
	}

	var buf [HeaderSize]byte
	binary.LittleEndian.PutUint32(buf[0:4], h.ObjectID)
	combined := (uint32(h.Size) << 16) | uint32(h.Opcode)
	binary.LittleEndian.PutUint32(buf[4:8], combined)

	if _, err := w.Write(buf[:]); err != nil {
		return fmt.Errorf("wire: encode header: %w", err)
	}
	return nil
}

// DecodeInt32 reads a signed 32-bit integer argument.
func DecodeInt32(r io.Reader) (int32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("%w: int32: %v", ErrInvalidArgument, err)
	}
	return int32(binary.LittleEndian.Uint32(buf[:])), nil
}

// EncodeInt32 writes a signed 32-bit integer argument.
func EncodeInt32(w io.Writer, v int32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(v))
	if _, err := w.Write(buf[:]); err != nil {
		return fmt.Errorf("wire: encode int32: %w", err)
	}
	return nil
}

// DecodeUint32 reads an unsigned 32-bit integer argument.
func DecodeUint32(r io.Reader) (uint32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("%w: uint32: %v", ErrInvalidArgument, err)
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}

// EncodeUint32 writes an unsigned 32-bit integer argument.
func EncodeUint32(w io.Writer, v uint32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	if _, err := w.Write(buf[:]); err != nil {
		return fmt.Errorf("wire: encode uint32: %w", err)
	}
	return nil
}

// DecodeFixed reads a fixed-point decimal (Q24.8 format).
func DecodeFixed(r io.Reader) (float64, error) {
	v, err := DecodeInt32(r)
	if err != nil {
		return 0, err
	}
	return float64(v) / 256.0, nil
}

// EncodeFixed writes a fixed-point decimal (Q24.8 format).
func EncodeFixed(w io.Writer, v float64) error {
	return EncodeInt32(w, int32(v*256.0))
}

// DecodeString reads a null-terminated UTF-8 string with length prefix.
// Strings are padded to 4-byte alignment.
func DecodeString(r io.Reader) (string, error) {
	length, err := DecodeUint32(r)
	if err != nil {
		return "", err
	}

	if length == 0 {
		return "", nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("%w: string data: %v", ErrInvalidArgument, err)
	}

	// Strings include null terminator in length
	if buf[length-1] != 0 {
		return "", fmt.Errorf("%w: string not null-terminated", ErrInvalidArgument)
	}

	// Read padding to 4-byte alignment
	padding := (4 - (length % 4)) % 4
	if padding > 0 {
		var pad [3]byte
		if _, err := io.ReadFull(r, pad[:padding]); err != nil {
			return "", fmt.Errorf("%w: string padding: %v", ErrInvalidArgument, err)
		}
	}

	return string(buf[:length-1]), nil
}

// EncodeString writes a null-terminated UTF-8 string with length prefix.
// Strings are padded to 4-byte alignment.
func EncodeString(w io.Writer, s string) error {
	if len(s) == 0 {
		// Empty string is encoded as length 0
		return EncodeUint32(w, 0)
	}

	length := uint32(len(s) + 1)
	if err := EncodeUint32(w, length); err != nil {
		return err
	}

	if _, err := w.Write([]byte(s)); err != nil {
		return fmt.Errorf("wire: encode string data: %w", err)
	}

	nullTerm := []byte{0}
	if _, err := w.Write(nullTerm); err != nil {
		return fmt.Errorf("wire: encode string null: %w", err)
	}

	padding := (4 - (length % 4)) % 4
	if padding > 0 {
		var pad [3]byte
		if _, err := w.Write(pad[:padding]); err != nil {
			return fmt.Errorf("wire: encode string padding: %w", err)
		}
	}

	return nil
}

// DecodeArray reads a byte array with length prefix.
// Arrays are padded to 4-byte alignment.
func DecodeArray(r io.Reader) ([]byte, error) {
	length, err := DecodeUint32(r)
	if err != nil {
		return nil, err
	}

	if length == 0 {
		return nil, nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("%w: array data: %v", ErrInvalidArgument, err)
	}

	padding := (4 - (length % 4)) % 4
	if padding > 0 {
		var pad [3]byte
		if _, err := io.ReadFull(r, pad[:padding]); err != nil {
			return nil, fmt.Errorf("%w: array padding: %v", ErrInvalidArgument, err)
		}
	}

	return buf, nil
}

// EncodeArray writes a byte array with length prefix.
// Arrays are padded to 4-byte alignment.
func EncodeArray(w io.Writer, data []byte) error {
	length := uint32(len(data))
	if err := EncodeUint32(w, length); err != nil {
		return err
	}

	if length > 0 {
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("wire: encode array data: %w", err)
		}

		padding := (4 - (length % 4)) % 4
		if padding > 0 {
			var pad [3]byte
			if _, err := w.Write(pad[:padding]); err != nil {
				return fmt.Errorf("wire: encode array padding: %w", err)
			}
		}
	}

	return nil
}
