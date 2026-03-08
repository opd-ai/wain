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

// Argument type constants for Wayland wire protocol encoding/decoding.
const (
	// ArgTypeInt32 represents a signed 32-bit integer argument.
	ArgTypeInt32 ArgumentType = 0
	// ArgTypeUint32 represents an unsigned 32-bit integer argument.
	ArgTypeUint32 ArgumentType = 1
	// ArgTypeFixed represents a fixed-point decimal (Q24.8 format) argument.
	ArgTypeFixed ArgumentType = 2
	// ArgTypeString represents a null-terminated UTF-8 string argument.
	ArgTypeString ArgumentType = 3
	// ArgTypeObject represents a reference to an existing Wayland object.
	ArgTypeObject ArgumentType = 4
	// ArgTypeNewID represents a newly created Wayland object identifier.
	ArgTypeNewID ArgumentType = 5
	// ArgTypeArray represents a byte array with length prefix.
	ArgTypeArray ArgumentType = 6
	// ArgTypeFD represents a file descriptor passed via SCM_RIGHTS.
	ArgTypeFD ArgumentType = 7
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

// readPadding reads padding bytes to align to 4-byte boundary.
func readPadding(r io.Reader, length uint32) error {
	padding := (4 - (length % 4)) % 4
	if padding > 0 {
		var pad [3]byte
		if _, err := io.ReadFull(r, pad[:padding]); err != nil {
			return err
		}
	}
	return nil
}

// writePadding writes padding bytes to align to 4-byte boundary.
func writePadding(w io.Writer, length uint32) error {
	padding := (4 - (length % 4)) % 4
	if padding > 0 {
		var pad [3]byte
		if _, err := w.Write(pad[:padding]); err != nil {
			return err
		}
	}
	return nil
}

// DecodeString reads a null-terminated UTF-8 string with length prefix.
// The length includes the null terminator. Empty strings are encoded as length 0.
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

	if buf[length-1] != 0 {
		return "", fmt.Errorf("%w: string not null-terminated", ErrInvalidArgument)
	}

	if err := readPadding(r, length); err != nil {
		return "", fmt.Errorf("%w: string padding: %v", ErrInvalidArgument, err)
	}

	return string(buf[:length-1]), nil
}

// EncodeString writes a null-terminated UTF-8 string with length prefix.
// Strings are padded to 4-byte alignment.
func EncodeString(w io.Writer, s string) error {
	if len(s) == 0 {
		return EncodeUint32(w, 0)
	}

	length := uint32(len(s) + 1)
	if err := EncodeUint32(w, length); err != nil {
		return err
	}

	if _, err := w.Write([]byte(s)); err != nil {
		return fmt.Errorf("wire: encode string data: %w", err)
	}

	if _, err := w.Write([]byte{0}); err != nil {
		return fmt.Errorf("wire: encode string null: %w", err)
	}

	if err := writePadding(w, length); err != nil {
		return fmt.Errorf("wire: encode string padding: %w", err)
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

	if err := readPadding(r, length); err != nil {
		return nil, fmt.Errorf("%w: array padding: %v", ErrInvalidArgument, err)
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

		if err := writePadding(w, length); err != nil {
			return fmt.Errorf("wire: encode array padding: %w", err)
		}
	}

	return nil
}

// Size returns the wire format size of an argument in bytes.
func (a *Argument) Size() uint16 {
	switch a.Type {
	case ArgTypeInt32, ArgTypeUint32, ArgTypeFixed, ArgTypeObject, ArgTypeNewID:
		return 4
	case ArgTypeString:
		s, ok := a.Value.(string)
		if !ok {
			return 0
		}
		if len(s) == 0 {
			return 4 // Just the length field (0)
		}
		length := uint32(len(s) + 1) // Include null terminator
		padding := (4 - (length % 4)) % 4
		return uint16(4 + length + padding) // length prefix + data + padding
	case ArgTypeArray:
		data, ok := a.Value.([]byte)
		if !ok {
			return 0
		}
		length := uint32(len(data))
		padding := (4 - (length % 4)) % 4
		return uint16(4 + length + padding) // length prefix + data + padding
	case ArgTypeFD:
		return 0 // FDs are passed out-of-band
	default:
		return 0
	}
}

// EncodeMessage encodes a complete message to wire format.
// Returns the encoded data and any file descriptors to be passed via SCM_RIGHTS.
func EncodeMessage(msg *Message) ([]byte, []int, error) {
	buf := make([]byte, 0, msg.Header.Size)
	w := &byteWriter{buf: buf}

	if err := EncodeHeader(w, msg.Header); err != nil {
		return nil, nil, err
	}

	fds, err := encodeArguments(w, msg.Args)
	if err != nil {
		return nil, nil, err
	}

	return w.buf, fds, nil
}

// encodeArguments encodes message arguments and collects file descriptors.
func encodeArguments(w io.Writer, args []Argument) ([]int, error) {
	var fds []int
	for i := range args {
		fd, err := encodeArgument(w, &args[i])
		if err != nil {
			return nil, err
		}
		if fd != -1 {
			fds = append(fds, fd)
		}
	}
	return fds, nil
}

// encodeArgument encodes a single argument, returning any file descriptor or -1.
// encodeArgument encodes a single argument to the writer.
// Returns the file descriptor index if arg is a fd type, otherwise -1.
func encodeArgument(w io.Writer, arg *Argument) (int, error) {
	switch arg.Type {
	case ArgTypeInt32:
		v, ok := arg.Value.(int32)
		if !ok {
			return -1, fmt.Errorf("%w: int32 value has wrong type", ErrInvalidArgument)
		}
		return -1, EncodeInt32(w, v)

	case ArgTypeUint32:
		v, ok := arg.Value.(uint32)
		if !ok {
			return -1, fmt.Errorf("%w: uint32 value has wrong type", ErrInvalidArgument)
		}
		return -1, EncodeUint32(w, v)

	case ArgTypeFixed:
		v, ok := arg.Value.(float64)
		if !ok {
			return -1, fmt.Errorf("%w: fixed value has wrong type", ErrInvalidArgument)
		}
		return -1, EncodeFixed(w, v)

	case ArgTypeString:
		v, ok := arg.Value.(string)
		if !ok {
			return -1, fmt.Errorf("%w: string value has wrong type", ErrInvalidArgument)
		}
		return -1, EncodeString(w, v)

	case ArgTypeObject, ArgTypeNewID:
		v, ok := arg.Value.(uint32)
		if !ok {
			return -1, fmt.Errorf("%w: object/new_id value has wrong type", ErrInvalidArgument)
		}
		return -1, EncodeUint32(w, v)

	case ArgTypeArray:
		v, ok := arg.Value.([]byte)
		if !ok {
			return -1, fmt.Errorf("%w: array value has wrong type", ErrInvalidArgument)
		}
		return -1, EncodeArray(w, v)

	case ArgTypeFD:
		v, ok := arg.Value.(int)
		if !ok {
			return -1, fmt.Errorf("%w: fd value has wrong type", ErrInvalidArgument)
		}
		return v, nil

	default:
		return -1, fmt.Errorf("%w: unknown argument type %d", ErrInvalidArgument, arg.Type)
	}
}

// byteWriter is an io.Writer that appends to a byte slice.
type byteWriter struct {
	buf []byte
}

// Write appends data to the internal buffer. Implements io.Writer.
func (w *byteWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}
