// Package client implements an X11 protocol client with core operations.
//
// This package provides a higher-level interface over the wire protocol,
// implementing the core X11 operations needed for basic window management:
//
//   - Connection setup and authentication
//   - Window creation and mapping
//   - Event handling
//   - Request-reply matching via sequence numbers
//
// # Connection Lifecycle
//
// The client connects to an X server via a Unix socket (typically /tmp/.X11-unix/X0),
// performs authentication using .Xauthority, and establishes a bidirectional channel
// for sending requests and receiving replies/events/errors.
//
// # Sequence Number Tracking
//
// Unlike Wayland's synchronous callback model, X11 uses sequence numbers to match
// asynchronous replies back to requests. The client maintains a u16 counter that
// increments after each request. The server echoes this number in replies and errors.
//
// # Thread Safety
//
// This implementation is not thread-safe. All operations on a connection must be
// performed from a single goroutine.
//
// Reference: https://www.x.org/releases/current/doc/xproto/x11protocol.html
package client

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/opd-ai/wain/internal/x11/wire"
)

var (
	// ErrClosed is returned when attempting operations on a closed connection.
	ErrClosed = errors.New("client: connection closed")

	// ErrInvalidXID is returned when an XID is invalid.
	ErrInvalidXID = errors.New("client: invalid XID")
)

// XID represents an X11 resource identifier (window, pixmap, GC, etc.).
type XID uint32

// Connection represents a connection to an X server.
type Connection struct {
	conn           net.Conn
	sequence       atomic.Uint32
	resourceIDBase uint32
	resourceIDMask uint32
	nextResourceID uint32
	rootWindow     XID
	rootVisual     uint32
	rootDepth      uint8
	screens        []wire.Screen
	closed         bool
}

// Connect establishes a connection to the X server on the specified display.
// The display string is typically "0" for the local display :0.
func Connect(display string) (*Connection, error) {
	// Connect to X server Unix socket
	socketPath := "/tmp/.X11-unix/X" + display
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("client: failed to connect to X server: %w", err)
	}

	// Read authentication data
	authName, authData, err := wire.ReadAuthority(display)
	if err != nil {
		// Allow connection without auth (for some X servers)
		authName = ""
		authData = nil
	}

	// Send setup request
	setupReq := wire.SetupRequest{
		ByteOrder:            wire.ByteOrderLSB,
		ProtocolMajorVersion: wire.ProtocolMajorVersion,
		ProtocolMinorVersion: wire.ProtocolMinorVersion,
		AuthName:             authName,
		AuthData:             authData,
	}

	if err := wire.EncodeSetupRequest(conn, setupReq); err != nil {
		conn.Close()
		return nil, fmt.Errorf("client: failed to send setup request: %w", err)
	}

	// Receive setup reply
	setupReply, err := wire.DecodeSetupReply(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("client: setup failed: %w", err)
	}

	if setupReply.Status != wire.SetupStatusSuccess {
		conn.Close()
		return nil, fmt.Errorf("client: setup failed with status %d", setupReply.Status)
	}

	// Extract root window information
	if len(setupReply.Screens) == 0 {
		conn.Close()
		return nil, errors.New("client: no screens in setup reply")
	}

	screen := setupReply.Screens[0]

	c := &Connection{
		conn:           conn,
		resourceIDBase: setupReply.ResourceIDBase,
		resourceIDMask: setupReply.ResourceIDMask,
		nextResourceID: setupReply.ResourceIDBase,
		rootWindow:     XID(screen.Root),
		rootVisual:     screen.RootVisual,
		rootDepth:      screen.RootDepth,
		screens:        setupReply.Screens,
	}

	return c, nil
}

// Close closes the connection to the X server.
func (c *Connection) Close() error {
	if c.closed {
		return ErrClosed
	}
	c.closed = true
	return c.conn.Close()
}

// RootWindow returns the root window XID.
func (c *Connection) RootWindow() XID {
	return c.rootWindow
}

// AllocXID allocates a new X resource ID.
func (c *Connection) AllocXID() (XID, error) {
	if c.closed {
		return 0, ErrClosed
	}

	// Allocate from the resource ID pool
	id := c.nextResourceID
	c.nextResourceID++

	// Check if we've exhausted the pool
	if (id & ^c.resourceIDMask) != c.resourceIDBase {
		return 0, ErrInvalidXID
	}

	return XID(id), nil
}

// sendRequest sends a request and increments the sequence number.
func (c *Connection) sendRequest(buf []byte) error {
	if c.closed {
		return ErrClosed
	}

	_, err := c.conn.Write(buf)
	if err != nil {
		return fmt.Errorf("client: failed to send request: %w", err)
	}

	// Increment sequence number (wraps at u16 boundary)
	c.sequence.Add(1)

	return nil
}

// SendRequest sends a raw request buffer (exposed for gc package).
func (c *Connection) SendRequest(buf []byte) error {
	return c.sendRequest(buf)
}

// CreateWindow creates a new window.
func (c *Connection) CreateWindow(parent XID, x, y int16, width, height, borderWidth, class uint16, visual, mask uint32, attrs []uint32) (XID, error) {
	wid, err := c.AllocXID()
	if err != nil {
		return 0, err
	}

	var buf bytes.Buffer

	// Calculate message length: header(4) + fixed args(28) + attrs(4*count)
	msgLen := uint16(8 + len(attrs))

	// Encode request header
	wire.EncodeRequestHeader(&buf, wire.OpcodeCreateWindow, c.rootDepth, msgLen)

	// Encode arguments
	wire.EncodeUint32(&buf, uint32(wid))
	wire.EncodeUint32(&buf, uint32(parent))
	wire.EncodeInt16(&buf, x)
	wire.EncodeInt16(&buf, y)
	wire.EncodeUint16(&buf, width)
	wire.EncodeUint16(&buf, height)
	wire.EncodeUint16(&buf, borderWidth)
	wire.EncodeUint16(&buf, class)
	wire.EncodeUint32(&buf, visual)
	wire.EncodeUint32(&buf, mask)

	// Encode attribute values
	for _, attr := range attrs {
		wire.EncodeUint32(&buf, attr)
	}

	if err := c.sendRequest(buf.Bytes()); err != nil {
		return 0, err
	}

	return wid, nil
}

// MapWindow makes a window visible on the screen.
func (c *Connection) MapWindow(window XID) error {
	var buf bytes.Buffer

	// MapWindow request is 8 bytes total (header + window ID)
	wire.EncodeRequestHeader(&buf, wire.OpcodeMapWindow, 0, 2)
	wire.EncodeUint32(&buf, uint32(window))

	return c.sendRequest(buf.Bytes())
}

// UnmapWindow makes a window invisible.
func (c *Connection) UnmapWindow(window XID) error {
	var buf bytes.Buffer

	// UnmapWindow request is 8 bytes total
	wire.EncodeRequestHeader(&buf, wire.OpcodeUnmapWindow, 0, 2)
	wire.EncodeUint32(&buf, uint32(window))

	return c.sendRequest(buf.Bytes())
}

// DestroyWindow destroys a window and frees its resources.
func (c *Connection) DestroyWindow(window XID) error {
	var buf bytes.Buffer

	// DestroyWindow request is 8 bytes total
	wire.EncodeRequestHeader(&buf, wire.OpcodeDestroyWindow, 0, 2)
	wire.EncodeUint32(&buf, uint32(window))

	return c.sendRequest(buf.Bytes())
}

// ConfigureWindowMask represents configuration value mask bits.
type ConfigureWindowMask uint16

const (
	ConfigMaskX           ConfigureWindowMask = 1 << 0
	ConfigMaskY           ConfigureWindowMask = 1 << 1
	ConfigMaskWidth       ConfigureWindowMask = 1 << 2
	ConfigMaskHeight      ConfigureWindowMask = 1 << 3
	ConfigMaskBorderWidth ConfigureWindowMask = 1 << 4
	ConfigMaskSibling     ConfigureWindowMask = 1 << 5
	ConfigMaskStackMode   ConfigureWindowMask = 1 << 6
)

// ConfigureWindow changes window attributes like position and size.
func (c *Connection) ConfigureWindow(window XID, mask ConfigureWindowMask, values []uint32) error {
	var buf bytes.Buffer

	// Calculate message length: header(4) + window(4) + mask(2) + pad(2) + values(4*count)
	msgLen := uint16(3 + len(values))

	wire.EncodeRequestHeader(&buf, wire.OpcodeConfigureWindow, 0, msgLen)
	wire.EncodeUint32(&buf, uint32(window))
	wire.EncodeUint16(&buf, uint16(mask))
	wire.EncodePadding(&buf, 2)

	// Encode values
	for _, v := range values {
		wire.EncodeUint32(&buf, v)
	}

	return c.sendRequest(buf.Bytes())
}

// RootVisual returns the root visual ID.
func (c *Connection) RootVisual() uint32 {
	return c.rootVisual
}

// RootDepth returns the root depth.
func (c *Connection) RootDepth() uint8 {
	return c.rootDepth
}
