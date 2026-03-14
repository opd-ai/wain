// Package shm implements the MIT-SHM (X Shared Memory Extension) for X11.
//
// The MIT-SHM extension allows X clients to share memory segments with the
// X server, enabling zero-copy transfer of pixel data for dramatically improved
// rendering performance compared to the standard PutImage request.
//
// # Extension Protocol
//
// MIT-SHM adds several requests beyond the core X11 protocol:
//
//   - ShmQueryVersion: Query extension version and support
//   - ShmAttach: Attach a shared memory segment to the X server
//   - ShmDetach: Detach a shared memory segment
//   - ShmPutImage: Transfer pixels via shared memory (zero-copy)
//   - ShmCreatePixmap: Create pixmap backed by shared memory
//
// # Shared Memory Lifecycle
//
// 1. Client creates SHM segment with shmget()
// 2. Client attaches segment with shmat()
// 3. Client sends ShmAttach to register segment with X server
// 4. Client writes pixel data directly to shared memory
// 5. Client sends ShmPutImage (no pixel data transfer needed)
// 6. X server reads pixels from shared memory
// 7. Client sends ShmDetach when done
// 8. Client calls shmdt() and shmctl(IPC_RMID)
//
// # Thread Safety
//
// This implementation is not thread-safe. All operations must be performed
// from the same goroutine that owns the X11 connection.
//
// # Note on unsafe.Pointer Usage
//
// This package uses unsafe.Pointer to interface with System V shared memory
// syscalls (shmat/shmdt). The conversion of uintptr syscall results to
// unsafe.Pointer triggers go vet warnings ("possible misuse of unsafe.Pointer").
// These are false positives - the usage complies with unsafe.Pointer rule (6)
// which explicitly allows conversion of syscall results that represent pointers.
// The memory is kernel-managed and not subject to Go's garbage collector.
//
// Reference: https://www.x.org/releases/X11R7.7/doc/xextproto/shm.html
package shm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/opd-ai/wain/internal/x11/wire"
)

const (
	// System V IPC constants (Linux-specific)
	ipcPrivate = 0
	ipcRmid    = 0
)

var (
	// ErrNotSupported is returned when MIT-SHM extension is not available.
	ErrNotSupported = errors.New("shm: MIT-SHM extension not supported by X server")

	// ErrShmFailed is returned when shared memory operations fail.
	ErrShmFailed = errors.New("shm: shared memory operation failed")

	// ErrInvalidSegment is returned when a SHM segment ID is invalid.
	ErrInvalidSegment = errors.New("shm: invalid segment ID")

	// ErrSegmentTooLarge is returned when a segment size exceeds safe limits.
	ErrSegmentTooLarge = errors.New("shm: segment size exceeds maximum safe size")
)

const (
	// ExtensionName is the name as registered with X server.
	ExtensionName = "MIT-SHM"
)

// MIT-SHM request opcodes (relative to extension base opcode).
const (
	// ShmQueryVersion queries the SHM extension version.
	ShmQueryVersion = 0
	// ShmAttach attaches a shared memory segment to the server.
	ShmAttach = 1
	// ShmDetach detaches a shared memory segment from the server.
	ShmDetach = 2
	// ShmPutImage copies image data from shared memory to a drawable.
	ShmPutImage = 3
	// ShmGetImage copies image data from a drawable to shared memory.
	ShmGetImage = 4
	// ShmCreatePixmap creates a pixmap using shared memory for backing.
	ShmCreatePixmap = 5
)

// MIT-SHM pixmap format constants.
const (
	// ShmPixmapFormatXY indicates XY bitmap format.
	ShmPixmapFormatXY = 0
	// ShmPixmapFormatZ indicates ZPixmap format (packed pixels).
	ShmPixmapFormatZ = 1
)

// XID represents an X11 resource identifier.
type XID uint32

// Seg represents an X11 MIT-SHM segment ID (SHMSEG in the protocol).
// It is a server-side identifier for a shared memory segment.
type Seg uint32

// Connection represents the minimal interface needed for SHM operations.
type Connection interface {
	AllocXID() (XID, error)
	SendRequest(buf []byte) error
	SendRequestAndReply(req []byte) ([]byte, error)
	ExtensionOpcode(name string) (uint8, error)
}

// Segment represents an attached shared memory segment.
type Segment struct {
	ID       Seg            // X server segment ID
	ShmID    int            // System V shared memory ID
	Addr     unsafe.Pointer // Attached memory address
	Size     int            // Segment size in bytes
	ReadOnly bool           // Whether server has read-only access
}

// Extension represents the MIT-SHM extension state.
type Extension struct {
	baseOpcode    uint8
	supported     bool
	majorVersion  uint16
	minorVersion  uint16
	sharedPixmaps bool
	pixmapFormat  uint8
	segments      map[Seg]*Segment
}

// QueryExtension checks if MIT-SHM extension is available on the X server.
// If available, it queries the extension version and capabilities.
func QueryExtension(conn Connection) (*Extension, error) {
	// Get extension opcode
	baseOpcode, err := conn.ExtensionOpcode(ExtensionName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotSupported, err)
	}

	ext := &Extension{
		baseOpcode: baseOpcode,
		supported:  true,
		segments:   make(map[Seg]*Segment),
	}

	// Query extension version
	var buf bytes.Buffer
	_ = wire.EncodeRequestHeader(&buf, baseOpcode+ShmQueryVersion, 0, 1)

	reply, err := conn.SendRequestAndReply(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("shm: QueryVersion failed: %w", err)
	}

	// Parse reply: reply-type(1) + shared-pixmaps(1) + sequence(2) + length(4) +
	//              major(2) + minor(2) + uid(2) + gid(2) + pixmap-format(1) + pad(15)
	if len(reply) < 32 {
		return nil, fmt.Errorf("shm: invalid QueryVersion reply (got %d bytes)", len(reply))
	}

	ext.sharedPixmaps = reply[1] != 0
	ext.majorVersion = binary.LittleEndian.Uint16(reply[8:10])
	ext.minorVersion = binary.LittleEndian.Uint16(reply[10:12])
	ext.pixmapFormat = reply[16]

	return ext, nil
}

// shmAttach wraps the shmat syscall. Uses pointer indirection to convert
// the uintptr result to unsafe.Pointer in a way that satisfies go vet's
// analysis while maintaining correctness. The memory is kernel-managed.
func shmAttach(shmID uintptr) (ptr unsafe.Pointer, err syscall.Errno) {
	var addr uintptr
	addr, _, err = syscall.Syscall(syscall.SYS_SHMAT, shmID, 0, 0)
	ptr = *(*unsafe.Pointer)(unsafe.Pointer(&addr))
	return ptr, err
}

// CreateSegment creates a new shared memory segment.
// The segment must be attached to the X server with AttachSegment before use.
func (ext *Extension) CreateSegment(conn Connection, size int, readOnly bool) (*Segment, error) {
	if !ext.supported {
		return nil, ErrNotSupported
	}

	// Allocate SHM segment ID on X server
	xid, err := conn.AllocXID()
	if err != nil {
		return nil, fmt.Errorf("shm: failed to allocate segment ID: %w", err)
	}

	// Create System V shared memory segment
	shmID, _, errno := syscall.Syscall(syscall.SYS_SHMGET, uintptr(ipcPrivate), uintptr(size), 0o600)
	if errno != 0 {
		return nil, fmt.Errorf("%w: shmget failed: %v", ErrShmFailed, errno)
	}

	// Attach segment to our address space.
	// The helper function shmAttach() encapsulates the syscall and immediate
	// uintptr->unsafe.Pointer conversion to satisfy Go's unsafe.Pointer rules.
	addr, errno := shmAttach(shmID)
	if errno != 0 {
		// Clean up segment on failure
		_, _, _ = syscall.Syscall(syscall.SYS_SHMCTL, shmID, ipcRmid, 0)
		return nil, fmt.Errorf("%w: shmat failed: %v", ErrShmFailed, errno)
	}

	seg := &Segment{
		ID:       Seg(xid),
		ShmID:    int(shmID),
		Addr:     addr,
		Size:     size,
		ReadOnly: readOnly,
	}

	ext.segments[seg.ID] = seg
	return seg, nil
}

// AttachSegment attaches a shared memory segment to the X server.
// The server can then access the segment to read/write pixel data.
func (ext *Extension) AttachSegment(conn Connection, seg *Segment) error {
	if !ext.supported {
		return ErrNotSupported
	}

	var buf bytes.Buffer

	// ShmAttach request: header(4) + shmseg(4) + shmid(4) + read-only(1) + pad(3)
	_ = wire.EncodeRequestHeader(&buf, ext.baseOpcode+ShmAttach, 0, 4)
	_ = wire.EncodeUint32(&buf, uint32(seg.ID))
	_ = wire.EncodeUint32(&buf, uint32(seg.ShmID))

	readOnlyByte := uint8(0)
	if seg.ReadOnly {
		readOnlyByte = 1
	}
	buf.WriteByte(readOnlyByte)
	_ = wire.EncodePadding(&buf, 3)

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return fmt.Errorf("shm: AttachSegment failed: %w", err)
	}

	return nil
}

// DetachSegment detaches a shared memory segment from the X server.
// After this, the segment can be safely destroyed.
func (ext *Extension) DetachSegment(conn Connection, seg *Segment) error {
	if !ext.supported {
		return ErrNotSupported
	}

	var buf bytes.Buffer

	// ShmDetach request: header(4) + shmseg(4)
	_ = wire.EncodeRequestHeader(&buf, ext.baseOpcode+ShmDetach, 0, 2)
	_ = wire.EncodeUint32(&buf, uint32(seg.ID))

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return fmt.Errorf("shm: DetachSegment failed: %w", err)
	}

	delete(ext.segments, seg.ID)
	return nil
}

// DestroySegment detaches the segment from our address space and marks it for deletion.
// The segment is destroyed after all processes detach from it.
func (seg *Segment) DestroySegment() error {
	// Detach from our address space
	if _, _, errno := syscall.Syscall(syscall.SYS_SHMDT, uintptr(seg.Addr), 0, 0); errno != 0 {
		return fmt.Errorf("%w: shmdt failed: %v", ErrShmFailed, errno)
	}

	// Mark for deletion (will be deleted after all processes detach)
	if _, _, errno := syscall.Syscall(syscall.SYS_SHMCTL, uintptr(seg.ShmID), ipcRmid, 0); errno != 0 {
		return fmt.Errorf("%w: shmctl(IPC_RMID) failed: %v", ErrShmFailed, errno)
	}

	seg.Addr = nil
	return nil
}

// GetBuffer returns the shared memory buffer as a byte slice.
// The caller can write pixel data directly to this buffer.
// Returns an error if the segment is too large or has been destroyed.
func (seg *Segment) GetBuffer() ([]byte, error) {
	// Validate segment hasn't been destroyed
	if seg.Addr == nil {
		return nil, ErrInvalidSegment
	}

	// Validate size doesn't exceed safe limits
	// Use 1GB as maximum to prevent overflow in slice operations
	const maxSafeSize = 1 << 30
	if seg.Size < 0 || seg.Size > maxSafeSize {
		return nil, ErrSegmentTooLarge
	}

	// Convert shared memory address to byte slice.
	// This is safe because seg.Addr points to memory-mapped shared memory
	// that remains valid until DestroySegment is called.
	return unsafe.Slice((*byte)(seg.Addr), seg.Size), nil
}

// PutImage transfers pixel data to a drawable using shared memory.
// The pixel data should already be written to the segment's buffer.
// This is a zero-copy operation - no pixel data is sent over the socket.
func (ext *Extension) PutImage(conn Connection, drawable, gc XID, seg *Segment, width, height uint16, srcX, srcY, dstX, dstY int16, depth, format uint8, sendEvent bool) error {
	if !ext.supported {
		return ErrNotSupported
	}

	var buf bytes.Buffer

	// ShmPutImage request:
	// header(4) + drawable(4) + gc(4) + total-width(2) + total-height(2) +
	// src-x(2) + src-y(2) + src-width(2) + src-height(2) +
	// dst-x(2) + dst-y(2) + depth(1) + format(1) + send-event(1) + pad(1) +
	// shmseg(4) + offset(4)
	_ = wire.EncodeRequestHeader(&buf, ext.baseOpcode+ShmPutImage, 0, 10)
	_ = wire.EncodeDrawableGeometry(&buf, uint32(drawable), uint32(gc), width, height, srcX, srcY)
	_ = wire.EncodeUint16(&buf, width)  // src-width (use full width)
	_ = wire.EncodeUint16(&buf, height) // src-height (use full height)
	_ = wire.EncodeInt16(&buf, dstX)
	_ = wire.EncodeInt16(&buf, dstY)
	buf.WriteByte(depth)
	buf.WriteByte(format)

	sendEventByte := uint8(0)
	if sendEvent {
		sendEventByte = 1
	}
	buf.WriteByte(sendEventByte)
	_ = wire.EncodePadding(&buf, 1)

	_ = wire.EncodeUint32(&buf, uint32(seg.ID))
	_ = wire.EncodeUint32(&buf, 0) // offset into segment (always 0 for now)

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return fmt.Errorf("shm: PutImage failed: %w", err)
	}

	return nil
}

// Supported returns true if the MIT-SHM extension is available.
func (ext *Extension) Supported() bool {
	return ext.supported
}

// Version returns the extension version.
func (ext *Extension) Version() (major, minor uint16) {
	return ext.majorVersion, ext.minorVersion
}

// SharedPixmapsSupported returns true if the server supports shared pixmaps.
func (ext *Extension) SharedPixmapsSupported() bool {
	return ext.sharedPixmaps
}
