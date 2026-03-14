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

	// Protocol field offsets within message buffers.
	offsetType     = 0  // Message type byte (reply/event/error)
	offsetData     = 1  // Data/detail/code byte
	offsetSequence = 2  // Sequence number (uint16)
	offsetLength   = 4  // Length field (uint32)
	offsetBadValue = 4  // Error bad value (uint32)
	offsetMinor    = 8  // Error minor opcode (uint16)
	offsetMajor    = 10 // Error major opcode (uint8)

	// Data section offsets for reply and event messages.
	replyInlineDataOffset = 8  // Reply inline data starts after 8-byte header
	eventDataOffset       = 4  // Event data starts after 4-byte header
	replyInlineDataSize   = 24 // Reply inline data is 24 bytes
	eventDataSize         = 28 // Event data is 28 bytes

	// Protocol constants.
	sendEventMask  = 0x7F // Mask to clear SendEvent flag from event type
	alignmentBytes = 4    // X11 protocol uses 4-byte alignment
)

// Request opcodes for core X11 protocol operations.
const (
	// OpcodeCreateWindow creates a new window.
	OpcodeCreateWindow = 1
	// OpcodeChangeWindowAttrs changes window attributes.
	OpcodeChangeWindowAttrs = 2
	// OpcodeGetWindowAttrs gets window attributes.
	OpcodeGetWindowAttrs = 3
	// OpcodeDestroyWindow destroys a window.
	OpcodeDestroyWindow = 4
	// OpcodeChangeSaveSet changes the save set.
	OpcodeChangeSaveSet = 6
	// OpcodeReparentWindow reparents a window.
	OpcodeReparentWindow = 7
	// OpcodeMapWindow maps a window to the screen.
	OpcodeMapWindow = 8
	// OpcodeUnmapWindow unmaps a window from the screen.
	OpcodeUnmapWindow = 9
	// OpcodeConfigureWindow configures window geometry.
	OpcodeConfigureWindow = 10
	// OpcodeInternAtom interns an atom name.
	OpcodeInternAtom = 16
	// OpcodeChangeProperty changes a window property.
	OpcodeChangeProperty = 18
	// OpcodeDeleteProperty deletes a window property.
	OpcodeDeleteProperty = 19
	// OpcodeGetProperty gets a window property.
	OpcodeGetProperty = 20
	// OpcodeSendEvent sends an event to a window.
	OpcodeSendEvent = 25
	// OpcodeCreateGC creates a graphics context.
	OpcodeCreateGC = 55
	// OpcodePutImage puts an image to a drawable.
	OpcodePutImage = 72
	// OpcodeQueryExtension queries for an extension.
	OpcodeQueryExtension = 98
)

// Predefined X11 atom IDs (XA_ constants from X11/Xatom.h).
const (
	// AtomString is the XA_STRING atom.
	AtomString = 31
	// AtomWMName is the XA_WM_NAME atom.
	AtomWMName = 39
	// AtomWMNormalHints is the XA_WM_NORMAL_HINTS atom.
	AtomWMNormalHints = 40
	// AtomWMSizeHints is the XA_WM_SIZE_HINTS atom.
	AtomWMSizeHints = 41
)

// WM_SIZE_HINTS flag bits used in WM_NORMAL_HINTS.
const (
	// WMSizeHintsPMinSize indicates min size fields are set.
	WMSizeHintsPMinSize = 1 << 4
	// WMSizeHintsPMaxSize indicates max size fields are set.
	WMSizeHintsPMaxSize = 1 << 5
)

// Window attribute mask bits for CreateWindow and ChangeWindowAttributes.
const (
	// CWBackPixmap sets the background pixmap.
	CWBackPixmap = 1 << 0
	// CWBackPixel sets the background pixel value.
	CWBackPixel = 1 << 1
	// CWBorderPixmap sets the border pixmap.
	CWBorderPixmap = 1 << 2
	// CWBorderPixel sets the border pixel value.
	CWBorderPixel = 1 << 3
	// CWBitGravity sets the bit gravity.
	CWBitGravity = 1 << 4
	// CWWinGravity sets the window gravity.
	CWWinGravity = 1 << 5
	// CWBackingStore sets the backing store mode.
	CWBackingStore = 1 << 6
	// CWBackingPlanes sets the backing planes.
	CWBackingPlanes = 1 << 7
	// CWBackingPixel sets the backing pixel value.
	CWBackingPixel = 1 << 8
	// CWOverrideRedirect sets the override redirect flag.
	CWOverrideRedirect = 1 << 9
	// CWSaveUnder sets the save under flag.
	CWSaveUnder = 1 << 10
	// CWEventMask sets the event mask.
	CWEventMask = 1 << 11
	// CWDontPropagate sets the do not propagate mask.
	CWDontPropagate = 1 << 12
	// CWColormap sets the colormap.
	CWColormap = 1 << 13
	// CWCursor sets the cursor.
	CWCursor = 1 << 14
)

// Event mask bits for receiving events.
const (
	// EventMaskKeyPress selects key press events.
	EventMaskKeyPress = 1 << 0
	// EventMaskKeyRelease selects key release events.
	EventMaskKeyRelease = 1 << 1
	// EventMaskButtonPress selects button press events.
	EventMaskButtonPress = 1 << 2
	// EventMaskButtonRelease selects button release events.
	EventMaskButtonRelease = 1 << 3
	// EventMaskEnterWindow selects pointer enter events.
	EventMaskEnterWindow = 1 << 4
	// EventMaskLeaveWindow selects pointer leave events.
	EventMaskLeaveWindow = 1 << 5
	// EventMaskPointerMotion selects pointer motion events.
	EventMaskPointerMotion = 1 << 6
	// EventMaskExposure selects exposure events.
	EventMaskExposure = 1 << 15
	// EventMaskStructureNotify selects structure notify events.
	EventMaskStructureNotify = 1 << 17
	// EventMaskSubstructureNotify selects substructure notify events.
	EventMaskSubstructureNotify = 1 << 18
)

// Window classes for CreateWindow.
const (
	// WindowClassCopyFromParent copies the class from the parent.
	WindowClassCopyFromParent = 0
	// WindowClassInputOutput is for windows that can display graphics.
	WindowClassInputOutput = 1
	// WindowClassInputOnly is for windows that only receive input.
	WindowClassInputOnly = 2
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
	buf[offsetType] = opcode
	buf[offsetData] = data
	binary.LittleEndian.PutUint16(buf[offsetSequence:offsetLength], length)

	if _, err := w.Write(buf[:]); err != nil {
		return fmt.Errorf("wire: failed to write request header: %w", err)
	}
	return nil
}

// EncodeUint64 writes a uint64 in little-endian format.
func EncodeUint64(w io.Writer, v uint64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	if _, err := w.Write(buf[:]); err != nil {
		return fmt.Errorf("wire: failed to write uint64: %w", err)
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

// EncodeUint8 writes a uint8.
func EncodeUint8(w io.Writer, v uint8) error {
	if _, err := w.Write([]byte{v}); err != nil {
		return fmt.Errorf("wire: failed to write uint8: %w", err)
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

// EncodeDrawableGeometry encodes a common X11 pattern: drawable, gc, width, height, x, y.
// This pattern appears in multiple image/graphics requests (PutImage, ShmPutImage, etc.).
func EncodeDrawableGeometry(w io.Writer, drawable, gc uint32, width, height uint16, xOffset, yOffset int16) error {
	if err := EncodeUint32(w, drawable); err != nil {
		return err
	}
	if err := EncodeUint32(w, gc); err != nil {
		return err
	}
	if err := EncodeUint16(w, width); err != nil {
		return err
	}
	if err := EncodeUint16(w, height); err != nil {
		return err
	}
	if err := EncodeInt16(w, xOffset); err != nil {
		return err
	}
	return EncodeInt16(w, yOffset)
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
	return buf[offsetType], nil
}

// MessageType represents the type of a server message.
type MessageType uint8

// X11 server message type constants.
const (
	// MessageTypeError indicates an error response from the server.
	MessageTypeError MessageType = 0
	// MessageTypeReply indicates a successful reply to a request.
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
	var header ReplyHeader
	var buf [ReplyHeaderSize]byte

	if _, err := io.ReadFull(r, buf[:]); err != nil {
		if err == io.EOF {
			return header, nil, io.EOF
		}
		return header, nil, fmt.Errorf("%w: %v", ErrMessageTooShort, err)
	}

	header.Type = MessageType(buf[offsetType])
	header.Data = buf[offsetData]
	header.Sequence = binary.LittleEndian.Uint16(buf[offsetSequence:offsetLength])
	header.Length = binary.LittleEndian.Uint32(buf[offsetLength:replyInlineDataOffset])

	// Return header and the 24-byte inline data portion
	return header, buf[replyInlineDataOffset:], nil
}

// EventHeader represents the header of an event message.
type EventHeader struct {
	Type     uint8  // Event type code
	Detail   uint8  // Event-specific detail
	Sequence uint16 // Sequence number
}

// DecodeEventHeader reads an event from r (32 bytes total).
func DecodeEventHeader(r io.Reader) (EventHeader, []byte, error) {
	var header EventHeader
	var buf [ReplyHeaderSize]byte

	if _, err := io.ReadFull(r, buf[:]); err != nil {
		if err == io.EOF {
			return header, nil, io.EOF
		}
		return header, nil, fmt.Errorf("%w: %v", ErrMessageTooShort, err)
	}

	header.Type = buf[offsetType] & sendEventMask // Clear highest bit (SendEvent flag)
	header.Detail = buf[offsetData]
	header.Sequence = binary.LittleEndian.Uint16(buf[offsetSequence:offsetLength])

	// Return header and the 28-byte event data
	return header, buf[eventDataOffset:], nil
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
	var header ErrorHeader
	var buf [ReplyHeaderSize]byte

	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return header, fmt.Errorf("%w: %v", ErrMessageTooShort, err)
	}

	header.Type = MessageType(buf[offsetType])
	header.Code = buf[offsetData]
	header.Sequence = binary.LittleEndian.Uint16(buf[offsetSequence:offsetLength])
	header.BadValue = binary.LittleEndian.Uint32(buf[offsetBadValue:offsetMinor])
	header.MinorOpcode = binary.LittleEndian.Uint16(buf[offsetMinor:offsetMajor])
	header.MajorOpcode = buf[offsetMajor]

	return header, nil
}

// Pad calculates the number of padding bytes needed for 4-byte alignment.
func Pad(length int) int {
	return (alignmentBytes - (length % alignmentBytes)) % alignmentBytes
}
