// Package client implements a Wayland protocol client with core object support.
//
// This package provides a higher-level interface over the wire protocol and socket
// layers, implementing the core Wayland objects needed for basic compositor interaction:
//
//   - wl_display: Connection management and synchronization
//   - wl_registry: Discovery of global compositor interfaces
//   - wl_compositor: Factory for creating surfaces
//   - wl_surface: Drawable surface for rendering content
//
// The implementation follows the Wayland protocol specification:
// https://wayland.freedesktop.org/docs/html/apa.html
//
// # Object Lifecycle
//
// Objects are created with sequential IDs starting from 2 (ID 1 is reserved for
// wl_display). Each object maintains a reference to the parent connection and
// can send requests via the wire protocol.
//
// # Thread Safety
//
// This implementation is not thread-safe. All operations on a connection and
// its objects must be performed from a single goroutine.
package client

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/opd-ai/wain/internal/wayland/socket"
	"github.com/opd-ai/wain/internal/wayland/wire"
)

var (
	// ErrClosed is returned when attempting operations on a closed connection.
	ErrClosed = errors.New("client: connection closed")

	// ErrInvalidObjectID is returned when an object ID is invalid.
	ErrInvalidObjectID = errors.New("client: invalid object ID")

	// ErrProtocolError is returned when the compositor sends a protocol error.
	ErrProtocolError = errors.New("client: protocol error")
)

const (
	// DisplayObjectID is the fixed object ID for wl_display (always 1).
	DisplayObjectID uint32 = 1

	// FirstClientObjectID is the first object ID available for client-created objects.
	FirstClientObjectID uint32 = 2
)

// Connection represents a connection to a Wayland compositor.
type Connection struct {
	socket      *socket.Conn
	nextID      atomic.Uint32
	objects     map[uint32]Object
	display     *Display
	closed      bool
	eventBuffer []byte
}

// Object represents a Wayland protocol object.
type Object interface {
	// ID returns the object's unique identifier.
	ID() uint32

	// Interface returns the Wayland interface name.
	Interface() string
}

// baseObject provides common implementation for all Wayland objects.
type baseObject struct {
	id    uint32
	iface string
	conn  *Connection
}

// ID returns the object's unique identifier.
func (o *baseObject) ID() uint32 {
	return o.id
}

// Interface returns the Wayland interface name.
func (o *baseObject) Interface() string {
	return o.iface
}

// Connect establishes a connection to the Wayland compositor.
// The socket path is typically obtained from the WAYLAND_DISPLAY environment
// variable combined with XDG_RUNTIME_DIR.
func Connect(path string) (*Connection, error) {
	sock, err := socket.Dial(path)
	if err != nil {
		return nil, fmt.Errorf("client: failed to connect: %w", err)
	}

	conn := &Connection{
		socket:      sock,
		objects:     make(map[uint32]Object),
		eventBuffer: make([]byte, 4096),
	}

	// Set the initial object ID counter.
	conn.nextID.Store(FirstClientObjectID)

	// Create the wl_display object (ID 1 is pre-assigned).
	conn.display = &Display{
		baseObject: baseObject{
			id:    DisplayObjectID,
			iface: "wl_display",
			conn:  conn,
		},
	}
	conn.objects[DisplayObjectID] = conn.display

	return conn, nil
}

// Display returns the wl_display object for this connection.
func (c *Connection) Display() *Display {
	return c.display
}

// Close terminates the connection to the compositor.
func (c *Connection) Close() error {
	if c.closed {
		return nil
	}

	c.closed = true
	c.objects = nil

	if err := c.socket.Close(); err != nil {
		return fmt.Errorf("client: close failed: %w", err)
	}

	return nil
}

// allocID allocates a new unique object ID.
func (c *Connection) allocID() uint32 {
	return c.nextID.Add(1) - 1
}

// registerObject adds an object to the connection's object registry.
func (c *Connection) registerObject(obj Object) {
	c.objects[obj.ID()] = obj
}

// sendRequest sends a request message to the compositor.
func (c *Connection) sendRequest(objectID uint32, opcode uint16, args []wire.Argument) error {
	if c.closed {
		return ErrClosed
	}

	msg := wire.Message{
		Header: wire.Header{
			ObjectID: objectID,
			Opcode:   opcode,
			Size:     wire.HeaderSize,
		},
		Args: args,
	}

	// Calculate total message size.
	for _, arg := range args {
		msg.Header.Size += arg.Size()
	}

	// Encode and send the message.
	data, fds, err := wire.EncodeMessage(&msg)
	if err != nil {
		return fmt.Errorf("client: encode failed: %w", err)
	}

	if len(fds) > 0 {
		if err := c.socket.SendWithFDs(data, fds); err != nil {
			return fmt.Errorf("client: send with fds failed: %w", err)
		}
	} else {
		if err := c.socket.Send(data); err != nil {
			return fmt.Errorf("client: send failed: %w", err)
		}
	}

	return nil
}

// Flush sends all pending requests to the compositor.
func (c *Connection) Flush() error {
	if c.closed {
		return ErrClosed
	}

	// Note: Our socket implementation doesn't buffer, so this is a no-op.
	// Included for API completeness and future buffering support.
	return nil
}

// AllocID allocates a new unique object ID.
// Exported for use by extension packages (SHM, XDG, etc.).
func (c *Connection) AllocID() uint32 {
	return c.allocID()
}

// RegisterObject adds an object to the connection's object registry.
// Exported for use by extension packages (SHM, XDG, etc.).
func (c *Connection) RegisterObject(obj interface{}) {
	if o, ok := obj.(Object); ok {
		c.registerObject(o)
	}
}

// SendRequest sends a request message to the compositor.
// Exported for use by extension packages (SHM, XDG, etc.).
func (c *Connection) SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error {
	return c.sendRequest(objectID, opcode, args)
}

// EventHandler is an interface for objects that can handle Wayland events.
type EventHandler interface {
	Object
	HandleEvent(opcode uint16, args []wire.Argument) error
}

// ReadMessage reads a single event message from the compositor.
// Returns nil, nil if no message is available (non-blocking read).
func (c *Connection) ReadMessage() (*wire.Message, error) {
	if c.closed {
		return nil, ErrClosed
	}

	// Try to read a message header
	n, _, err := c.socket.RecvMsg(c.eventBuffer[:wire.HeaderSize], 0)
	if err != nil {
		return nil, fmt.Errorf("client: read header failed: %w", err)
	}
	if n == 0 {
		return nil, nil // No message available
	}
	if n < wire.HeaderSize {
		return nil, fmt.Errorf("client: incomplete header (%d bytes)", n)
	}

	// Decode header
	header := wire.Header{
		ObjectID: binary.LittleEndian.Uint32(c.eventBuffer[0:4]),
		Opcode:   uint16(binary.LittleEndian.Uint32(c.eventBuffer[4:8]) & 0xFFFF),
		Size:     uint16(binary.LittleEndian.Uint32(c.eventBuffer[4:8]) >> 16),
	}

	// Read the rest of the message if needed
	payloadSize := int(header.Size) - wire.HeaderSize
	if payloadSize > 0 {
		n, _, err = c.socket.RecvMsg(c.eventBuffer[wire.HeaderSize:header.Size], 0)
		if err != nil {
			return nil, fmt.Errorf("client: read payload failed: %w", err)
		}
		if n < payloadSize {
			return nil, fmt.Errorf("client: incomplete payload (%d of %d bytes)", n, payloadSize)
		}
	}

	msg := &wire.Message{
		Header: header,
	}

	return msg, nil
}

// DispatchMessage routes an event message to the appropriate object handler.
func (c *Connection) DispatchMessage(msg *wire.Message) error {
	if c.closed {
		return ErrClosed
	}

	obj, exists := c.objects[msg.Header.ObjectID]
	if !exists {
		return fmt.Errorf("%w: %d", ErrInvalidObjectID, msg.Header.ObjectID)
	}

	handler, ok := obj.(EventHandler)
	if !ok {
		// Object doesn't handle events, silently ignore
		return nil
	}

	return handler.HandleEvent(msg.Header.Opcode, msg.Args)
}
