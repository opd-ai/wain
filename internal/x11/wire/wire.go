// Package wire implements the X11 wire protocol format.
//
// The X11 wire protocol uses a binary format for communication between client
// and server. Unlike Wayland, X11 uses a request-reply model with sequence
// number tracking for matching asynchronous responses.
//
// # Message Types
//
// The protocol defines four message types:
//   - REQUEST: Client sends operations to the server (CreateWindow, MapWindow, etc.)
//   - REPLY: Server sends data back for specific requests (GetProperty, QueryPointer)
//   - EVENT: Server sends unsolicited notifications (KeyPress, Expose, etc.)
//   - ERROR: Server reports request failures, tagged with sequence number
//
// # Binary Format
//
// REQUEST: [opcode:u8][data:u8][length:u16][args padded to 4 bytes]
// REPLY:   [0x01][format:u8][sequence:u16][length:u32][32 bytes inline][extra data]
// EVENT:   [type:u8][detail:u8][sequence:u16][28 bytes event data] (always 32 bytes)
// ERROR:   [0x00][code:u8][sequence:u16][bad_value:u32][...] (always 32 bytes)
//
// All integers are little-endian. Messages are padded to 4-byte alignment.
// Length field is in 4-byte units, not bytes.
//
// # Sequence Numbers
//
// The client maintains a u16 sequence counter that increments after each request.
// The server echoes this sequence number in replies and errors, allowing the
// client to match asynchronous responses back to their originating requests.
//
// Reference: https://www.x.org/releases/current/doc/xproto/x11protocol.html
package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidMessage is returned when a message cannot be decoded.
	ErrInvalidMessage = errors.New("wire: invalid message")

	// ErrMessageTooShort is returned when insufficient data is available.
	ErrMessageTooShort = errors.New("wire: message too short")

	// ErrInvalidLength is returned when message length is malformed.
	ErrInvalidLength = errors.New("wire: invalid length")
)

const (
	// RequestHeaderSize is the size of an X11 request header in bytes.
	RequestHeaderSize = 4

	// ReplyHeaderSize is the size of a reply/event/error header in bytes.
	ReplyHeaderSize = 32

	// MaxMessageSize is the maximum message size (64KB for safety).
	MaxMessageSize = 65536
)

// Request opcodes for core X11 protocol operations.
const (
	OpcodeCreateWindow      = 1
	OpcodeChangeWindowAttrs = 2
	OpcodeGetWindowAttrs    = 3
	OpcodeDestroyWindow     = 4
	OpcodeChangeSaveSet     = 6
	OpcodeReparentWindow    = 7
	OpcodeMapWindow         = 8
	OpcodeUnmapWindow       = 9
	OpcodeConfigureWindow   = 10
	OpcodeGetProperty       = 20
	OpcodeChangeProperty    = 18
	OpcodeDeleteProperty    = 19
	OpcodeCreateGC          = 55
	OpcodePutImage          = 72
)

// Window attribute mask bits for CreateWindow and ChangeWindowAttributes.
const (
	CWBackPixmap       = 1 << 0
	CWBackPixel        = 1 << 1
	CWBorderPixmap     = 1 << 2
	CWBorderPixel      = 1 << 3
	CWBitGravity       = 1 << 4
	CWWinGravity       = 1 << 5
	CWBackingStore     = 1 << 6
	CWBackingPlanes    = 1 << 7
	CWBackingPixel     = 1 << 8
	CWOverrideRedirect = 1 << 9
	CWSaveUnder        = 1 << 10
	CWEventMask        = 1 << 11
	CWDontPropagate    = 1 << 12
	CWColormap         = 1 << 13
	CWCursor           = 1 << 14
)

// Event mask bits for receiving events.
const (
	EventMaskKeyPress           = 1 << 0
	EventMaskKeyRelease         = 1 << 1
	EventMaskButtonPress        = 1 << 2
	EventMaskButtonRelease      = 1 << 3
	EventMaskEnterWindow        = 1 << 4
	EventMaskLeaveWindow        = 1 << 5
	EventMaskPointerMotion      = 1 << 6
	EventMaskExposure           = 1 << 15
	EventMaskStructureNotify    = 1 << 17
	EventMaskSubstructureNotify = 1 << 18
)

// Window classes for CreateWindow.
const (
	WindowClassCopyFromParent = 0
	WindowClassInputOutput    = 1
	WindowClassInputOnly      = 2
)

// RequestHeader represents the common header for all X11 requests.
type RequestHeader struct {
	Opcode uint8  // Request operation code
	Data   uint8  // Optional request-specific data
	Length uint16 // Request length in 4-byte units
}

// EncodeRequestHeader writes a request header to w.
func EncodeRequestHeader(w io.Writer, opcode, data uint8, length uint16) error {
	var buf [RequestHeaderSize]byte
	buf[0] = opcode
	buf[1] = data
	binary.LittleEndian.PutUint16(buf[2:4], length)

	if _, err := w.Write(buf[:]); err != nil {
		return fmt.Errorf("wire: failed to write request header: %w", err)
	}
	return nil
}

// EncodeUint32 writes a uint32 in little-endian format.
func EncodeUint32(w io.Writer, v uint32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	if _, err := w.Write(buf[:]); err != nil {
		return fmt.Errorf("wire: failed to write uint32: %w", err)
	}
	return nil
}

// EncodeUint16 writes a uint16 in little-endian format.
func EncodeUint16(w io.Writer, v uint16) error {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], v)
	if _, err := w.Write(buf[:]); err != nil {
		return fmt.Errorf("wire: failed to write uint16: %w", err)
	}
	return nil
}

// EncodeInt16 writes an int16 in little-endian format.
func EncodeInt16(w io.Writer, v int16) error {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], uint16(v))
	if _, err := w.Write(buf[:]); err != nil {
		return fmt.Errorf("wire: failed to write int16: %w", err)
	}
	return nil
}

// EncodePadding writes n zero bytes for alignment.
func EncodePadding(w io.Writer, n int) error {
	if n <= 0 {
		return nil
	}
	padding := make([]byte, n)
	if _, err := w.Write(padding); err != nil {
		return fmt.Errorf("wire: failed to write padding: %w", err)
	}
	return nil
}

// DecodeUint32 reads a uint32 in little-endian format.
func DecodeUint32(r io.Reader) (uint32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrMessageTooShort, err)
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}

// DecodeUint16 reads a uint16 in little-endian format.
func DecodeUint16(r io.Reader) (uint16, error) {
	var buf [2]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrMessageTooShort, err)
	}
	return binary.LittleEndian.Uint16(buf[:]), nil
}

// DecodeUint8 reads a single byte.
func DecodeUint8(r io.Reader) (uint8, error) {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrMessageTooShort, err)
	}
	return buf[0], nil
}

// MessageType represents the type of a server message.
type MessageType uint8

const (
	MessageTypeError MessageType = 0
	MessageTypeReply MessageType = 1
)

// ReplyHeader represents the header of a reply message.
type ReplyHeader struct {
	Type     MessageType // Should be MessageTypeReply (1)
	Data     uint8       // Reply-specific data
	Sequence uint16      // Sequence number matching the request
	Length   uint32      // Additional data length in 4-byte units
}

// DecodeReplyHeader reads a reply header from r.
func DecodeReplyHeader(r io.Reader) (ReplyHeader, []byte, error) {
	var h ReplyHeader
	var buf [ReplyHeaderSize]byte

	if _, err := io.ReadFull(r, buf[:]); err != nil {
		if err == io.EOF {
			return h, nil, io.EOF
		}
		return h, nil, fmt.Errorf("%w: %v", ErrMessageTooShort, err)
	}

	h.Type = MessageType(buf[0])
	h.Data = buf[1]
	h.Sequence = binary.LittleEndian.Uint16(buf[2:4])
	h.Length = binary.LittleEndian.Uint32(buf[4:8])

	// Return header and the 24-byte inline data portion
	return h, buf[8:], nil
}

// EventHeader represents the header of an event message.
type EventHeader struct {
	Type     uint8  // Event type code
	Detail   uint8  // Event-specific detail
	Sequence uint16 // Sequence number
}

// DecodeEventHeader reads an event from r (32 bytes total).
func DecodeEventHeader(r io.Reader) (EventHeader, []byte, error) {
	var h EventHeader
	var buf [ReplyHeaderSize]byte

	if _, err := io.ReadFull(r, buf[:]); err != nil {
		if err == io.EOF {
			return h, nil, io.EOF
		}
		return h, nil, fmt.Errorf("%w: %v", ErrMessageTooShort, err)
	}

	h.Type = buf[0] & 0x7F // Clear highest bit (SendEvent flag)
	h.Detail = buf[1]
	h.Sequence = binary.LittleEndian.Uint16(buf[2:4])

	// Return header and the 28-byte event data
	return h, buf[4:], nil
}

// ErrorHeader represents an error message from the server.
type ErrorHeader struct {
	Type        MessageType // Should be MessageTypeError (0)
	Code        uint8       // Error code
	Sequence    uint16      // Sequence number of failed request
	BadValue    uint32      // Value that caused the error
	MinorOpcode uint16      // Minor opcode (for extensions)
	MajorOpcode uint8       // Major opcode of failed request
}

// DecodeErrorHeader reads an error message from r.
func DecodeErrorHeader(r io.Reader) (ErrorHeader, error) {
	var h ErrorHeader
	var buf [ReplyHeaderSize]byte

	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return h, fmt.Errorf("%w: %v", ErrMessageTooShort, err)
	}

	h.Type = MessageType(buf[0])
	h.Code = buf[1]
	h.Sequence = binary.LittleEndian.Uint16(buf[2:4])
	h.BadValue = binary.LittleEndian.Uint32(buf[4:8])
	h.MinorOpcode = binary.LittleEndian.Uint16(buf[8:10])
	h.MajorOpcode = buf[10]

	return h, nil
}

// Pad calculates the number of padding bytes needed for 4-byte alignment.
func Pad(length int) int {
	return (4 - (length % 4)) % 4
}
