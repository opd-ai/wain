// Package gc implements X11 graphics context operations.
//
// This package provides graphics context management, image blitting,
// and pixmap operations for X11 rendering:
//
//   - CreateGC: Create graphics contexts with configurable attributes
//   - PutImage: Transfer pixel data to windows/pixmaps
//   - CreatePixmap: Create offscreen drawing surfaces
//   - FreeGC/FreePixmap: Resource cleanup
//
// # Graphics Context
//
// A Graphics Context (GC) is a server-side rendering state object that holds
// drawing attributes like foreground color, background color, line width, etc.
// GCs are created with CreateGC and must be freed with FreeGC when no longer needed.
//
// # Image Format
//
// PutImage supports ZPixmap format (packed pixel data) for transferring
// ARGB8888 image data from client to server. The image data must be in
// little-endian format with 4-byte alignment.
//
// # Pixmaps
//
// Pixmaps are offscreen rendering surfaces that can be used as backing stores,
// texture atlases, or intermediate render targets. They share the same drawable
// interface as windows for rendering operations.
//
// Reference: https://www.x.org/releases/current/doc/xproto/x11protocol.html
package gc

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/opd-ai/wain/internal/x11/wire"
)

var (
	// ErrInvalidGC is returned when a GC operation fails.
	ErrInvalidGC = errors.New("gc: invalid graphics context")

	// ErrInvalidPixmap is returned when a pixmap operation fails.
	ErrInvalidPixmap = errors.New("gc: invalid pixmap")

	// ErrInvalidImage is returned when image data is malformed.
	ErrInvalidImage = errors.New("gc: invalid image data")
)

// Image format constants for PutImage operations.
const (
	// FormatBitmap indicates 1-bit bitmap format.
	FormatBitmap = 0
	// FormatXYPixmap indicates XY format (plane-separated pixel data).
	FormatXYPixmap = 1
	// FormatZPixmap indicates Z format (packed pixels, most common for ARGB8888).
	FormatZPixmap = 2
)

// GC attribute mask bits for CreateGC. Each bit corresponds to a value
// that can be set in the graphics context. Values are provided in the
// attrs slice in the order of these bits (lowest bit first).
const (
	// GCFunction selects the logical raster operation (e.g., copy, XOR).
	GCFunction = 1 << 0
	// GCPlaneMask selects which bit planes are affected by drawing operations.
	GCPlaneMask = 1 << 1
	// GCForeground sets the foreground pixel value for drawing operations.
	GCForeground = 1 << 2
	// GCBackground sets the background pixel value for drawing operations.
	GCBackground = 1 << 3
	// GCLineWidth sets the line width in pixels (0 means 1-pixel thin line).
	GCLineWidth = 1 << 4
	// GCLineStyle sets the line style (solid, dashed, etc.).
	GCLineStyle = 1 << 5
	// GCCapStyle sets the line cap style (butt, round, projecting).
	GCCapStyle = 1 << 6
	// GCJoinStyle sets the line join style (miter, round, bevel).
	GCJoinStyle = 1 << 7
	// GCFillStyle sets the fill style (solid, tiled, stippled, etc.).
	GCFillStyle = 1 << 8
	// GCFillRule sets the fill rule (even-odd or winding).
	GCFillRule = 1 << 9
	// GCTile sets the pixmap to use for tiled fill operations.
	GCTile = 1 << 10
	// GCStipple sets the pixmap to use for stippled fill operations.
	GCStipple = 1 << 11
	// GCTileStippleXOrigin sets the X origin for tile/stipple patterns.
	GCTileStippleXOrigin = 1 << 12
	// GCTileStippleYOrigin sets the Y origin for tile/stipple patterns.
	GCTileStippleYOrigin = 1 << 13
	// GCFont sets the font for text drawing operations.
	GCFont = 1 << 14
	// GCSubwindowMode controls whether subwindows clip drawing operations.
	GCSubwindowMode = 1 << 15
	// GCGraphicsExposures controls whether GraphicsExpose events are generated.
	GCGraphicsExposures = 1 << 16
	// GCClipXOrigin sets the X origin of the clipping region.
	GCClipXOrigin = 1 << 17
	// GCClipYOrigin sets the Y origin of the clipping region.
	GCClipYOrigin = 1 << 18
	// GCClipMask sets the pixmap to use as the clipping mask.
	GCClipMask = 1 << 19
	// GCDashOffset sets the starting offset for dashed lines.
	GCDashOffset = 1 << 20
	// GCDashes sets the dash pattern for dashed lines.
	GCDashes = 1 << 21
	// GCArcMode sets whether arcs are filled as pie slices or chords.
	GCArcMode = 1 << 22
)

// GC function values for the GCFunction attribute. These define the
// logical raster operation applied when drawing.
const (
	// GXCopy copies source pixels to destination (most common).
	GXCopy = 3
	// GXXor performs XOR between source and destination pixels.
	GXXor = 6
)

// Opcodes for X11 GC and pixmap protocol requests.
const (
	// OpcodeCreateGC is the X11 protocol opcode for CreateGC request.
	OpcodeCreateGC = 55
	// OpcodeFreeGC is the X11 protocol opcode for FreeGC request.
	OpcodeFreeGC = 60
	// OpcodeCreatePixmap is the X11 protocol opcode for CreatePixmap request.
	OpcodeCreatePixmap = 53
	// OpcodeFreePixmap is the X11 protocol opcode for FreePixmap request.
	OpcodeFreePixmap = 54
	// OpcodePutImage is the X11 protocol opcode for PutImage request.
	OpcodePutImage = 72
)

// XID represents an X11 resource identifier.
type XID uint32

// Connection represents the minimal interface needed for GC operations.
type Connection interface {
	AllocXID() (XID, error)
	SendRequest(buf []byte) error
}

// CreateGC creates a new graphics context.
// The mask parameter specifies which attributes are set, and attrs provides
// the corresponding values in the order defined by the mask bits.
func CreateGC(conn Connection, drawable XID, mask uint32, attrs []uint32) (XID, error) {
	gc, err := conn.AllocXID()
	if err != nil {
		return 0, fmt.Errorf("gc: failed to allocate GC ID: %w", err)
	}

	var buf bytes.Buffer

	// Calculate message length: header(4) + gc(4) + drawable(4) + mask(4) + attrs(4*count)
	msgLen := uint16(4 + len(attrs))

	// Encode request
	_ = wire.EncodeRequestHeader(&buf, OpcodeCreateGC, 0, msgLen)
	_ = wire.EncodeUint32(&buf, uint32(gc))
	_ = wire.EncodeUint32(&buf, uint32(drawable))
	_ = wire.EncodeUint32(&buf, mask)

	// Encode attribute values
	for _, attr := range attrs {
		wire.EncodeUint32(&buf, attr)
	}

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return 0, fmt.Errorf("gc: CreateGC failed: %w", err)
	}

	return gc, nil
}

// FreeGC destroys a graphics context and frees server resources.
func FreeGC(conn Connection, gc XID) error {
	var buf bytes.Buffer

	// FreeGC request is 8 bytes total (header + GC ID)
	_ = wire.EncodeRequestHeader(&buf, OpcodeFreeGC, 0, 2)
	wire.EncodeUint32(&buf, uint32(gc))

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return fmt.Errorf("gc: FreeGC failed: %w", err)
	}

	return nil
}

// CreatePixmap creates an offscreen drawable surface.
func CreatePixmap(conn Connection, drawable XID, width, height uint16, depth uint8) (XID, error) {
	pixmap, err := conn.AllocXID()
	if err != nil {
		return 0, fmt.Errorf("gc: failed to allocate pixmap ID: %w", err)
	}

	var buf bytes.Buffer

	// CreatePixmap request: header(4) + depth(1) + pixmap(4) + drawable(4) + width(2) + height(2)
	_ = wire.EncodeRequestHeader(&buf, OpcodeCreatePixmap, depth, 4)
	wire.EncodeUint32(&buf, uint32(pixmap))
	wire.EncodeUint32(&buf, uint32(drawable))
	_ = wire.EncodeUint16(&buf, width)
	_ = wire.EncodeUint16(&buf, height)

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return 0, fmt.Errorf("gc: CreatePixmap failed: %w", err)
	}

	return pixmap, nil
}

// FreePixmap destroys a pixmap and frees server resources.
func FreePixmap(conn Connection, pixmap XID) error {
	var buf bytes.Buffer

	// FreePixmap request is 8 bytes total
	wire.EncodeRequestHeader(&buf, OpcodeFreePixmap, 0, 2)
	wire.EncodeUint32(&buf, uint32(pixmap))

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return fmt.Errorf("gc: FreePixmap failed: %w", err)
	}

	return nil
}

// PutImage transfers pixel data to a drawable (window or pixmap).
// The format should be FormatZPixmap for packed pixel data.
// The depth is the drawable's depth (typically 24 or 32).
// The data must be in ARGB8888 format (32 bits per pixel) for depth 24/32.
func PutImage(conn Connection, drawable, gc XID, width, height uint16, xOffset, yOffset int16, depth, format uint8, data []byte) error {
	if len(data) == 0 {
		return ErrInvalidImage
	}

	var buf bytes.Buffer

	// Calculate message length
	// header(4) + drawable(4) + gc(4) + width(2) + height(2) + x(2) + y(2) +
	// leftPad(1) + depth(1) + pad(2) + data(variable)
	dataLen := len(data)

	// Ensure data is padded to 4-byte boundary
	padLen := (4 - (dataLen % 4)) % 4
	totalDataLen := dataLen + padLen

	msgLen := uint16(6 + (totalDataLen / 4))

	// Encode request header (format goes in data byte)
	wire.EncodeRequestHeader(&buf, OpcodePutImage, format, msgLen)

	// Encode parameters
	_ = wire.EncodeDrawableGeometry(&buf, uint32(drawable), uint32(gc), width, height, xOffset, yOffset)

	// leftPad (0 for ZPixmap) and depth
	buf.WriteByte(0)
	buf.WriteByte(depth)

	// Padding to align image data
	_ = wire.EncodePadding(&buf, 2)

	// Write image data
	buf.Write(data)

	// Add padding to 4-byte boundary
	_ = wire.EncodePadding(&buf, padLen)

	if err := conn.SendRequest(buf.Bytes()); err != nil {
		return fmt.Errorf("gc: PutImage failed: %w", err)
	}

	return nil
}
