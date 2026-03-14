// Package socket provides Unix domain socket operations with file descriptor
// passing support for Wayland protocol communication.
//
// The Wayland protocol requires the ability to send and receive file descriptors
// over a Unix domain socket using the SCM_RIGHTS control message mechanism.
// This package wraps the low-level socket operations needed for Wayland clients.
//
// File descriptors are passed out-of-band alongside regular message data:
// - Messages are sent/received through normal socket I/O
// - File descriptors are sent/received via sendmsg/recvmsg with SCM_RIGHTS
//
// Reference: https://wayland.freedesktop.org/docs/html/ch04.html#sect-Protocol-Wire-Format
package socket

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
)

var (
	// ErrNoFileDescriptors is returned when attempting to receive FDs but none are available.
	ErrNoFileDescriptors = errors.New("socket: no file descriptors received")

	// ErrTooManyFileDescriptors is returned when too many FDs are passed in a single message.
	ErrTooManyFileDescriptors = errors.New("socket: too many file descriptors")

	// ErrInvalidSocket is returned when the socket is not a Unix domain socket.
	ErrInvalidSocket = errors.New("socket: not a unix domain socket")
)

const (
	// MaxFDsPerMessage is the maximum number of file descriptors that can be
	// passed in a single message. This matches typical kernel limits.
	MaxFDsPerMessage = 28

	// MaxControlMessageSize is the buffer size for control messages (ancillary data).
	// Large enough to hold MaxFDsPerMessage file descriptors.
	// This is computed as syscall.CmsgSpace(MaxFDsPerMessage * 4) = 512 bytes on Linux.
	MaxControlMessageSize = 512
)

// Conn wraps a Unix domain socket connection with file descriptor passing support.
type Conn struct {
	conn *net.UnixConn
	file *os.File
	fd   int
}

// Dial connects to a Wayland compositor socket at the given path.
// The path is typically obtained from the WAYLAND_DISPLAY environment variable,
// resolved against XDG_RUNTIME_DIR.
func Dial(path string) (*Conn, error) {
	addr := &net.UnixAddr{
		Name: path,
		Net:  "unix",
	}

	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("socket: dial failed: %w", err)
	}

	c, err := NewConn(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return c, nil
}

// NewConn creates a Conn from an existing net.UnixConn.
// Note: conn.File() creates a duplicate fd, which we use for low-level syscalls.
// The *os.File is stored to prevent the GC from closing the duplicate fd.
func NewConn(conn *net.UnixConn) (*Conn, error) {
	file, err := conn.File()
	if err != nil {
		return nil, fmt.Errorf("socket: failed to get fd: %w", err)
	}

	return &Conn{
		conn: conn,
		file: file,
		fd:   int(file.Fd()),
	}, nil
}

// Read reads data from the socket into the provided buffer.
// This is a wrapper around the standard Read operation.
func (c *Conn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

// Write writes data to the socket from the provided buffer.
// This is a wrapper around the standard Write operation.
func (c *Conn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

// SendMsg sends a message with optional file descriptors.
// The data is sent through normal socket I/O, while file descriptors
// are passed via SCM_RIGHTS control messages.
//
// If fds is empty, this is equivalent to a normal Write.
// If fds is provided, it must contain at most MaxFDsPerMessage descriptors.
func (c *Conn) SendMsg(data []byte, fds []int) (int, error) {
	if len(fds) > MaxFDsPerMessage {
		return 0, ErrTooManyFileDescriptors
	}

	if len(fds) == 0 {
		return c.Write(data)
	}

	rights := syscall.UnixRights(fds...)
	return syscall.SendmsgN(c.fd, data, rights, nil, 0)
}

// RecvMsg receives a message and any associated file descriptors.
// It reads up to len(data) bytes of message data and up to maxFDs file descriptors.
//
// Returns:
// - n: number of bytes read into data
// - fds: slice of received file descriptors (may be empty)
// - err: error if any
//
// The caller is responsible for closing the returned file descriptors.
func (c *Conn) RecvMsg(data []byte, maxFDs int) (n int, fds []int, err error) {
	oob := make([]byte, MaxControlMessageSize)
	n, oobn, _, _, err := syscall.Recvmsg(c.fd, data, oob, 0)
	if err != nil {
		return 0, nil, fmt.Errorf("socket: recvmsg failed: %w", err)
	}

	if oobn > 0 {
		fds, err = parseFDs(oob[:oobn])
		if err != nil {
			return n, nil, err
		}
	}

	return n, fds, nil
}

// parseFDs extracts file descriptors from SCM_RIGHTS control messages.
func parseFDs(oob []byte) ([]int, error) {
	fds := make([]int, 0, 4)
	scms, err := syscall.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, fmt.Errorf("socket: failed to parse control message: %w", err)
	}

	for _, scm := range scms {
		if scm.Header.Level == syscall.SOL_SOCKET && scm.Header.Type == syscall.SCM_RIGHTS {
			rights, err := syscall.ParseUnixRights(&scm)
			if err != nil {
				return nil, fmt.Errorf("socket: failed to parse unix rights: %w", err)
			}
			fds = append(fds, rights...)
		}
	}

	return fds, nil
}

// Send sends data without any file descriptors.
// This is a convenience wrapper around SendMsg.
func (c *Conn) Send(data []byte) error {
	_, err := c.SendMsg(data, nil)
	if err != nil {
		return fmt.Errorf("wayland/socket: send: %w", err)
	}
	return nil
}

// SendWithFDs sends data with file descriptors.
// This is a convenience wrapper around SendMsg.
func (c *Conn) SendWithFDs(data []byte, fds []int) error {
	_, err := c.SendMsg(data, fds)
	if err != nil {
		return fmt.Errorf("wayland/socket: send with fds: %w", err)
	}
	return nil
}

// SendFD sends a single file descriptor alongside a data message.
// This is a convenience wrapper around SendMsg for the common case of sending one FD.
func (c *Conn) SendFD(data []byte, fd int) (int, error) {
	return c.SendMsg(data, []int{fd})
}

// RecvFD receives a message and expects exactly one file descriptor.
// Returns an error if no FD is received or if multiple FDs are received.
func (c *Conn) RecvFD(data []byte) (n, fd int, err error) {
	n, fds, err := c.RecvMsg(data, 1)
	if err != nil {
		return 0, -1, err
	}

	if len(fds) == 0 {
		return n, -1, ErrNoFileDescriptors
	}

	if len(fds) > 1 {
		for i := 1; i < len(fds); i++ {
			syscall.Close(fds[i])
		}
		return n, fds[0], ErrTooManyFileDescriptors
	}

	return n, fds[0], nil
}

// Close closes the underlying socket connection and the duplicated file.
func (c *Conn) Close() error {
	err := c.conn.Close()
	if c.file != nil {
		if ferr := c.file.Close(); ferr != nil && err == nil {
			err = ferr
		}
	}
	if err != nil {
		return fmt.Errorf("wayland/socket: close: %w", err)
	}
	return nil
}

// Fd returns the underlying file descriptor.
// This is useful for integration with select/poll/epoll.
func (c *Conn) Fd() int {
	return c.fd
}

// LocalAddr returns the local address of the socket.
func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote address of the socket.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// MakePair creates a pair of connected Unix sockets for testing.
// The caller is responsible for closing both connections.
func MakePair() (*Conn, *Conn, error) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("socket: socketpair failed: %w", err)
	}

	conn1, conn2, err := createSocketPair(fds)
	if err != nil {
		return nil, nil, err
	}

	c1, c2, err := wrapConnections(conn1, conn2)
	if err != nil {
		conn1.Close()
		conn2.Close()
		return nil, nil, err
	}

	return c1, c2, nil
}

// createSocketPair creates net.Conn from socket file descriptors.
func createSocketPair(fds [2]int) (net.Conn, net.Conn, error) {
	file1 := os.NewFile(uintptr(fds[0]), "socket1")
	file2 := os.NewFile(uintptr(fds[1]), "socket2")

	conn1, err := net.FileConn(file1)
	file1.Close()
	if err != nil {
		file2.Close()
		return nil, nil, fmt.Errorf("socket: failed to create conn1: %w", err)
	}

	conn2, err := net.FileConn(file2)
	file2.Close()
	if err != nil {
		conn1.Close()
		return nil, nil, fmt.Errorf("socket: failed to create conn2: %w", err)
	}

	return conn1, conn2, nil
}

// wrapConnections wraps net.Conn as Unix connections and creates Conn wrappers.
func wrapConnections(conn1, conn2 net.Conn) (*Conn, *Conn, error) {
	unix1, ok := conn1.(*net.UnixConn)
	if !ok {
		return nil, nil, ErrInvalidSocket
	}

	unix2, ok := conn2.(*net.UnixConn)
	if !ok {
		return nil, nil, ErrInvalidSocket
	}

	c1, err := NewConn(unix1)
	if err != nil {
		unix1.Close()
		unix2.Close()
		return nil, nil, err
	}

	c2, err := NewConn(unix2)
	if err != nil {
		c1.Close()
		unix2.Close()
		return nil, nil, err
	}

	return c1, c2, nil
}
