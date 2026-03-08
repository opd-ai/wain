// Package dri3 implements the DRI3 (Direct Rendering Infrastructure version 3) extension for X11.
//
// DRI3 enables direct GPU buffer sharing between X11 clients and the X server,
// replacing the older DRI2 protocol with a cleaner file descriptor-based approach.
// Combined with the Present extension, DRI3 allows zero-copy GPU rendering with
// proper frame synchronization.
//
// # Extension Protocol
//
// DRI3 adds several key operations:
//
//   - DRI3QueryVersion: Query extension version and capabilities
//   - DRI3Open: Open DRI device and get render node fd
//   - DRI3PixmapFromBuffer: Create X11 pixmap from DMA-BUF file descriptor
//   - DRI3BufferFromPixmap: Export pixmap as DMA-BUF (inverse operation)
//   - DRI3FenceFromFD: Create X11 fence from sync fd
//   - DRI3FDFromFence: Export X11 fence as sync fd
//
// # GPU Buffer Sharing Workflow
//
// 1. Client calls DRI3Open to get render node file descriptor
// 2. Client allocates GPU buffer via DRM/GBM (or Rust allocator)
// 3. Client exports buffer as DMA-BUF file descriptor
// 4. Client calls DRI3PixmapFromBuffer to create X11 pixmap
// 5. Client uses Present extension to display the pixmap with vsync
//
// # Version Support
//
// This implementation targets DRI3 version 1.2, which adds:
//   - DRI3PixmapFromBuffers (multi-planar buffers)
//   - DRI3BuffersFromPixmap (multi-planar export)
//   - Modifier support for tiled/compressed formats
//
// DRI3 1.0 is sufficient for basic single-plane ARGB buffers.
//
// # Thread Safety
//
// This implementation is not thread-safe. All operations must be performed
// from the same goroutine that owns the X11 connection.
//
// Reference: https://gitlab.freedesktop.org/xorg/proto/xorgproto/-/blob/master/dri3proto.txt
package dri3

import (
	"bytes"
	"errors"
	"fmt"
	"syscall"

	"github.com/opd-ai/wain/internal/x11/wire"
)

var (
	// ErrNotSupported is returned when DRI3 extension is not available.
	ErrNotSupported = errors.New("dri3: extension not supported by X server")

	// ErrVersionTooOld is returned when server DRI3 version is too old.
	ErrVersionTooOld = errors.New("dri3: server version too old (need 1.0+)")

	// ErrOpenFailed is returned when DRI3Open fails.
	ErrOpenFailed = errors.New("dri3: failed to open render node")

	// ErrInvalidFD is returned when a file descriptor is invalid.
	ErrInvalidFD = errors.New("dri3: invalid file descriptor")

	// ErrPixmapCreationFailed is returned when pixmap creation fails.
	ErrPixmapCreationFailed = errors.New("dri3: pixmap creation failed")
)

const (
	// ExtensionName is the name as registered with X server.
	ExtensionName = "DRI3"
)

// DRI3 request opcodes (relative to extension base opcode).
const (
	DRI3QueryVersion          = 0
	DRI3Open                  = 1
	DRI3PixmapFromBuffer      = 2
	DRI3BufferFromPixmap      = 3
	DRI3FenceFromFD           = 4
	DRI3FDFromFence           = 5
	DRI3GetSupportedModifiers = 6
	DRI3PixmapFromBuffers     = 7
	DRI3BuffersFromPixmap     = 8
)

// XID represents an X11 resource identifier.
type XID uint32

// Connection represents the minimal interface needed for DRI3 operations.
type Connection interface {
	AllocXID() (XID, error)
	SendRequest(buf []byte) error
	SendRequestAndReply(req []byte) ([]byte, error)
	SendRequestWithFDs(req []byte, fds []int) error
	SendRequestAndReplyWithFDs(req []byte, fds []int) ([]byte, []int, error)
	ExtensionOpcode(name string) (uint8, error)
}

// Extension represents the DRI3 extension state.
type Extension struct {
	baseOpcode   uint8
	supported    bool
	majorVersion uint32
	minorVersion uint32
}

// QueryExtension checks if DRI3 extension is available on the X server.
// If available, it queries the extension version and capabilities.
func QueryExtension(conn Connection) (*Extension, error) {
	// Query extension using common helper
	info, err := wire.QueryExtensionVersion(conn, ExtensionName, DRI3QueryVersion, 1, 2)
	if err != nil {
		// Wrap with DRI3-specific error if it's a basic not-supported error
		if errors.Is(err, syscall.Errno(0)) || errors.Is(err, fmt.Errorf("")) {
			return nil, fmt.Errorf("%w: %v", ErrNotSupported, err)
		}
		return nil, err
	}

	// Verify version is at least 1.0
	if info.MajorVersion < 1 {
		return nil, ErrVersionTooOld
	}

	return &Extension{
		baseOpcode:   info.BaseOpcode,
		supported:    true,
		majorVersion: info.MajorVersion,
		minorVersion: info.MinorVersion,
	}, nil
}

// MajorVersion returns the negotiated DRI3 major version.
func (e *Extension) MajorVersion() uint32 {
	return e.majorVersion
}

// MinorVersion returns the negotiated DRI3 minor version.
func (e *Extension) MinorVersion() uint32 {
	return e.minorVersion
}

// SupportsModifiers returns true if the server supports DRM format modifiers (version 1.2+).
func (e *Extension) SupportsModifiers() bool {
	return e.majorVersion > 1 || (e.majorVersion == 1 && e.minorVersion >= 2)
}

// Open opens the DRI3 render node for the specified drawable and provider.
// Returns a file descriptor to the render node (typically /dev/dri/renderD128).
//
// The drawable is typically the root window. The provider can be set to 0
// to use the default GPU, or to a specific provider XID for multi-GPU systems.
//
// The caller is responsible for closing the returned file descriptor.
func (e *Extension) Open(conn Connection, drawable XID, provider uint32) (int, error) {
	var buf bytes.Buffer

	// DRI3Open request: header(4) + drawable(4) + provider(4)
	wire.EncodeRequestHeader(&buf, e.baseOpcode+DRI3Open, 0, 3)
	wire.EncodeUint32(&buf, uint32(drawable))
	wire.EncodeUint32(&buf, provider)

	reply, fds, err := conn.SendRequestAndReplyWithFDs(buf.Bytes(), nil)
	if err != nil {
		return -1, fmt.Errorf("%w: %v", ErrOpenFailed, err)
	}

	if len(fds) != 1 {
		// Close any fds we got
		for _, fd := range fds {
			syscall.Close(fd)
		}
		return -1, fmt.Errorf("%w: expected 1 fd, got %d", ErrOpenFailed, len(fds))
	}

	if len(reply) < 32 {
		syscall.Close(fds[0])
		return -1, fmt.Errorf("dri3: invalid Open reply (got %d bytes)", len(reply))
	}

	return fds[0], nil
}

// PixmapFromBuffer creates an X11 pixmap from a DMA-BUF file descriptor.
// This is the core DRI3 operation for sharing GPU buffers with the X server.
//
// Parameters:
//   - pixmap: the XID to assign to the new pixmap (from AllocXID)
//   - drawable: the drawable (typically root window) for visual/depth inheritance
//   - size: buffer size in bytes
//   - width, height, stride: buffer dimensions and stride (in bytes)
//   - depth: color depth (typically 24 for RGB, 32 for RGBA)
//   - bpp: bits per pixel (typically 32 for ARGB8888)
//   - fd: DMA-BUF file descriptor (will be duplicated by X server, caller retains ownership)
//
// The fd should point to a GPU-allocated buffer. The X server will import it
// and the pixmap can then be used with Present or standard X11 operations.
func (e *Extension) PixmapFromBuffer(conn Connection, pixmap, drawable XID,
	size uint32, width, height, stride uint16, depth, bpp uint8, fd int,
) error {
	if fd < 0 {
		return ErrInvalidFD
	}

	var buf bytes.Buffer

	// DRI3PixmapFromBuffer request (version 1.0):
	// header(4) + pixmap(4) + drawable(4) + size(4) + width(2) + height(2) +
	// stride(2) + depth(1) + bpp(1)
	wire.EncodeRequestHeader(&buf, e.baseOpcode+DRI3PixmapFromBuffer, 0, 6)
	wire.EncodeUint32(&buf, uint32(pixmap))
	wire.EncodeUint32(&buf, uint32(drawable))
	wire.EncodeUint32(&buf, size)
	wire.EncodeUint16(&buf, width)
	wire.EncodeUint16(&buf, height)
	wire.EncodeUint16(&buf, stride)
	wire.EncodeUint8(&buf, depth)
	wire.EncodeUint8(&buf, bpp)

	// Send request with fd attachment
	if err := conn.SendRequestWithFDs(buf.Bytes(), []int{fd}); err != nil {
		return fmt.Errorf("%w: %v", ErrPixmapCreationFailed, err)
	}

	return nil
}

// PixmapFromBuffers creates an X11 pixmap from multiple DMA-BUF file descriptors.
// This is the DRI3 1.2+ version that supports multi-planar formats and modifiers.
//
// Parameters:
//   - pixmap: the XID to assign to the new pixmap
//   - drawable: the drawable for visual/depth inheritance
//   - width, height: buffer dimensions in pixels
//   - fourcc: DRM fourcc format code (e.g., DRM_FORMAT_ARGB8888)
//   - modifier: DRM format modifier (0 for linear, otherwise tiling/compression)
//   - depth: color depth
//   - bpp: bits per pixel
//   - strides: array of stride values for each plane (in bytes)
//   - offsets: array of offset values for each plane (in bytes)
//   - fds: array of DMA-BUF file descriptors (one per plane)
//
// For simple single-plane formats, use PixmapFromBuffer instead.
// This function is needed for YUV formats or tiled/compressed GPU buffers.
func (e *Extension) PixmapFromBuffers(conn Connection, pixmap, drawable XID,
	width, height uint16, fourcc, modifier uint32, depth, bpp uint8,
	strides, offsets []uint32, fds []int,
) error {
	if !e.SupportsModifiers() {
		return fmt.Errorf("dri3: PixmapFromBuffers requires version 1.2+ (have %d.%d)",
			e.majorVersion, e.minorVersion)
	}

	if len(fds) == 0 || len(strides) != len(fds) || len(offsets) != len(fds) {
		return fmt.Errorf("dri3: invalid plane count (fds=%d, strides=%d, offsets=%d)",
			len(fds), len(strides), len(offsets))
	}

	for _, fd := range fds {
		if fd < 0 {
			return ErrInvalidFD
		}
	}

	var buf bytes.Buffer

	// Calculate message length
	numBuffers := uint8(len(fds))
	// header(4) + pixmap(4) + drawable(4) + num_buffers(1) + pad(3) +
	// width(2) + height(2) + stride0(4) + offset0(4) + ... + strideN + offsetN +
	// depth(1) + bpp(1) + pad(2) + modifier(8)
	msgLen := uint16(5 + 2*len(fds))

	wire.EncodeRequestHeader(&buf, e.baseOpcode+DRI3PixmapFromBuffers, 0, msgLen)
	wire.EncodeUint32(&buf, uint32(pixmap))
	// DRI3 pixmap geometry fields
	wire.EncodeUint32(&buf, uint32(drawable))
	wire.EncodeUint8(&buf, numBuffers)
	wire.EncodePadding(&buf, 3)
	wire.EncodeUint16(&buf, width)
	wire.EncodeUint16(&buf, height)
	wire.EncodeUint32(&buf, fourcc)

	// Encode strides and offsets
	for i := 0; i < len(strides); i++ {
		wire.EncodeUint32(&buf, strides[i])
		wire.EncodeUint32(&buf, offsets[i])
	}

	wire.EncodeUint8(&buf, depth)
	wire.EncodeUint8(&buf, bpp)
	wire.EncodePadding(&buf, 2)
	// Modifier is 64-bit (little-endian)
	wire.EncodeUint64(&buf, uint64(modifier))

	// Send request with fd attachments
	if err := conn.SendRequestWithFDs(buf.Bytes(), fds); err != nil {
		return fmt.Errorf("%w: %v", ErrPixmapCreationFailed, err)
	}

	return nil
}
