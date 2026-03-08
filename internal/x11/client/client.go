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
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"syscall"
	"unsafe"

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

// Window configuration mask bits for ConfigureWindow operations.
const (
	// ConfigMaskX enables setting the window's X position.
	ConfigMaskX ConfigureWindowMask = 1 << 0
	// ConfigMaskY enables setting the window's Y position.
	ConfigMaskY ConfigureWindowMask = 1 << 1
	// ConfigMaskWidth enables setting the window's width.
	ConfigMaskWidth ConfigureWindowMask = 1 << 2
	// ConfigMaskHeight enables setting the window's height.
	ConfigMaskHeight ConfigureWindowMask = 1 << 3
	// ConfigMaskBorderWidth enables setting the window's border width.
	ConfigMaskBorderWidth ConfigureWindowMask = 1 << 4
	// ConfigMaskSibling enables setting the sibling window for stacking.
	ConfigMaskSibling ConfigureWindowMask = 1 << 5
	// ConfigMaskStackMode enables setting the stacking mode (above/below).
	ConfigMaskStackMode ConfigureWindowMask = 1 << 6
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

// SendRequestAndReply sends a request and waits for a reply.
// This is used for requests that expect a response from the server.
func (c *Connection) SendRequestAndReply(req []byte) ([]byte, error) {
	if err := c.sendRequest(req); err != nil {
		return nil, err
	}

	// Read reply header (32 bytes)
	reply := make([]byte, wire.ReplyHeaderSize)
	if _, err := c.conn.Read(reply); err != nil {
		return nil, fmt.Errorf("client: failed to read reply: %w", err)
	}

	// Check if it's a reply (type 1) or error (type 0)
	if reply[0] == 0 {
		// Error response
		errCode := reply[1]
		return nil, fmt.Errorf("client: X11 error code %d", errCode)
	}

	// Parse additional data length (in 4-byte units)
	dataLen := binary.LittleEndian.Uint32(reply[4:8])
	if dataLen > 0 {
		// Read additional data beyond the 32-byte header
		additionalData := make([]byte, dataLen*4)
		if _, err := c.conn.Read(additionalData); err != nil {
			return nil, fmt.Errorf("client: failed to read reply data: %w", err)
		}
		// Append additional data to reply
		reply = append(reply, additionalData...)
	}

	return reply, nil
}

// ExtensionOpcode queries an X11 extension by name and returns its base opcode.
// Returns an error if the extension is not supported by the server.
func (c *Connection) ExtensionOpcode(name string) (uint8, error) {
	var buf bytes.Buffer

	// Calculate message length: header(4) + nameLen(2) + pad(2) + name + padding
	nameLen := len(name)
	namePad := wire.Pad(nameLen)
	msgLen := uint16(2 + (nameLen+namePad)/4)

	// Encode QueryExtension request
	wire.EncodeRequestHeader(&buf, wire.OpcodeQueryExtension, 0, msgLen)
	wire.EncodeUint16(&buf, uint16(nameLen))
	wire.EncodePadding(&buf, 2)
	buf.WriteString(name)
	wire.EncodePadding(&buf, namePad)

	reply, err := c.SendRequestAndReply(buf.Bytes())
	if err != nil {
		return 0, err
	}

	// Parse reply: type(1) + pad(1) + sequence(2) + length(4) + present(1) + major-opcode(1) + ...
	if len(reply) < 32 {
		return 0, fmt.Errorf("client: invalid QueryExtension reply")
	}

	present := reply[8]
	if present == 0 {
		return 0, fmt.Errorf("client: extension %q not present", name)
	}

	majorOpcode := reply[9]
	return majorOpcode, nil
}

// SendRequestWithFDs sends a request with attached file descriptors.
// This is used by extensions like DRI3 that need to pass DMA-BUF fds to the X server.
//
// The fds are sent via SCM_RIGHTS control messages over the Unix socket.
// The X server will duplicate the file descriptors, so the caller retains ownership
// and is responsible for closing them.
func (c *Connection) SendRequestWithFDs(req []byte, fds []int) error {
	if c.closed {
		return ErrClosed
	}

	// Convert net.Conn to *net.UnixConn for SendMsg support
	unixConn, ok := c.conn.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("client: connection is not a Unix socket")
	}

	// Build control message with file descriptors
	var oob []byte
	if len(fds) > 0 {
		// SCM_RIGHTS control message size: 16 bytes header + 4*len(fds) bytes data
		rights := make([]byte, syscall.CmsgSpace(4*len(fds)))
		header := (*syscall.Cmsghdr)(unsafe.Pointer(&rights[0]))
		header.Level = syscall.SOL_SOCKET
		header.Type = syscall.SCM_RIGHTS
		header.SetLen(syscall.CmsgLen(4 * len(fds)))

		// Copy file descriptors into control message
		data := rights[syscall.CmsgSpace(0):]
		for i, fd := range fds {
			binary.LittleEndian.PutUint32(data[i*4:], uint32(fd))
		}
		oob = rights
	}

	// Send request with file descriptors
	if _, _, err := unixConn.WriteMsgUnix(req, oob, nil); err != nil {
		return fmt.Errorf("client: failed to send request with fds: %w", err)
	}

	c.sequence.Add(1)
	return nil
}

// SendRequestAndReplyWithFDs sends a request with optional file descriptors
// and waits for a reply that may also contain file descriptors.
//
// This is used by extensions like DRI3 where both requests and replies can
// carry file descriptors (e.g., DRI3Open returns a render node fd).
func (c *Connection) SendRequestAndReplyWithFDs(req []byte, fds []int) ([]byte, []int, error) {
	if err := c.SendRequestWithFDs(req, fds); err != nil {
		return nil, nil, err
	}

	unixConn, ok := c.conn.(*net.UnixConn)
	if !ok {
		return nil, nil, fmt.Errorf("client: connection is not a Unix socket")
	}

	reply, oob, oobn, err := c.readReplyHeaderWithFDs(unixConn)
	if err != nil {
		return nil, nil, err
	}

	reply, err = c.readAdditionalReplyData(reply)
	if err != nil {
		return nil, nil, err
	}

	receivedFDs, err := extractFileDescriptors(oob, oobn)
	if err != nil {
		return nil, nil, err
	}

	return reply, receivedFDs, nil
}

// readReplyHeaderWithFDs reads the X11 reply header and any file descriptors from the Unix socket.
func (c *Connection) readReplyHeaderWithFDs(unixConn *net.UnixConn) ([]byte, []byte, int, error) {
	reply := make([]byte, wire.ReplyHeaderSize)
	oob := make([]byte, syscall.CmsgSpace(4*16)) // Space for up to 16 fds

	n, oobn, _, _, err := unixConn.ReadMsgUnix(reply, oob)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("client: failed to read reply: %w", err)
	}
	if n < wire.ReplyHeaderSize {
		return nil, nil, 0, fmt.Errorf("client: incomplete reply (got %d bytes)", n)
	}

	if reply[0] == 0 {
		errCode := reply[1]
		return nil, nil, 0, fmt.Errorf("client: X11 error code %d", errCode)
	}

	return reply, oob, oobn, nil
}

// readAdditionalReplyData reads any additional data beyond the header based on the reply length field.
func (c *Connection) readAdditionalReplyData(reply []byte) ([]byte, error) {
	dataLen := binary.LittleEndian.Uint32(reply[4:8])
	if dataLen == 0 {
		return reply, nil
	}

	additionalData := make([]byte, dataLen*4)
	if _, err := c.conn.Read(additionalData); err != nil {
		return nil, fmt.Errorf("client: failed to read reply data: %w", err)
	}
	return append(reply, additionalData...), nil
}

func extractFileDescriptors(oob []byte, oobn int) ([]int, error) {
	if oobn == 0 {
		return nil, nil
	}

	messages, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return nil, fmt.Errorf("client: failed to parse control message: %w", err)
	}

	var receivedFDs []int
	for _, msg := range messages {
		if msg.Header.Level == syscall.SOL_SOCKET && msg.Header.Type == syscall.SCM_RIGHTS {
			fds, err := syscall.ParseUnixRights(&msg)
			if err != nil {
				return nil, fmt.Errorf("client: failed to parse SCM_RIGHTS: %w", err)
			}
			receivedFDs = append(receivedFDs, fds...)
		}
	}

	return receivedFDs, nil
}
