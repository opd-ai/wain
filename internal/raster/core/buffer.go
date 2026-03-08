// Package core implements a tile-based 2D software rasterizer for ARGB8888 buffers.
//
// This package provides CPU-based rendering primitives for UI elements:
//
//   - Filled rectangles: solid color rectangle fills with clipping
//   - Rounded rectangles: rectangles with circular corner radius
//   - Line segments: anti-aliased line drawing
//
// # Buffer Format
//
// All rendering operates on ARGB8888 buffers with pixels stored in little-endian
// format: [B, G, R, A] in memory. Each pixel is 4 bytes aligned. The buffer has
// a configurable stride (bytes per row) to support subrectangle rendering.
//
// # Coordinate System
//
// The coordinate system is standard 2D raster: origin (0,0) at top-left,
// X increases right, Y increases down. Coordinates are in pixels.
//
// # Alpha Compositing
//
// All drawing operations use Porter-Duff SrcOver compositing:
//
//	result.rgb = src.rgb * src.a + dst.rgb * (1 - src.a)
//	result.a = src.a + dst.a * (1 - src.a)
//
// # Clipping
//
// Drawing operations are automatically clipped to buffer bounds. Out-of-bounds
// coordinates are safely ignored without panic.
package core

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidBuffer is returned when a buffer is malformed or has invalid dimensions.
	ErrInvalidBuffer = errors.New("core: invalid buffer")

	// ErrInvalidColor is returned when a color value is out of range.
	ErrInvalidColor = errors.New("core: invalid color")
)

// Color represents an ARGB color with 8-bit channels.
type Color struct {
	R, G, B, A uint8
}

// RGBA returns the color as premultiplied alpha components.
func (c Color) RGBA() (r, g, b, a uint32) {
	a = uint32(c.A)
	a |= a << 8
	r = uint32(c.R)
	r |= r << 8
	r = (r * a) / 0xffff
	g = uint32(c.G)
	g |= g << 8
	g = (g * a) / 0xffff
	b = uint32(c.B)
	b |= b << 8
	b = (b * a) / 0xffff
	return r, g, b, a
}

// Buffer represents an ARGB8888 pixel buffer.
type Buffer struct {
	// Pixels is the raw pixel data in ARGB8888 format (little-endian: B, G, R, A).
	Pixels []byte

	// Width is the buffer width in pixels.
	Width int

	// Height is the buffer height in pixels.
	Height int

	// Stride is the number of bytes per row (usually Width * 4).
	Stride int
}

// NewBuffer creates a new ARGB8888 buffer with the specified dimensions.
func NewBuffer(width, height int) (*Buffer, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("%w: dimensions must be positive", ErrInvalidBuffer)
	}
	if width > 16384 || height > 16384 {
		return nil, fmt.Errorf("%w: dimensions exceed maximum (16384x16384)", ErrInvalidBuffer)
	}

	stride := width * 4
	size := stride * height
	pixels := make([]byte, size)

	return &Buffer{
		Pixels: pixels,
		Width:  width,
		Height: height,
		Stride: stride,
	}, nil
}

// Clear fills the entire buffer with the specified color.
func (b *Buffer) Clear(c Color) {
	if len(b.Pixels) == 0 {
		return
	}

	pixel := packColor(c)
	for y := 0; y < b.Height; y++ {
		offset := y * b.Stride
		for x := 0; x < b.Width; x++ {
			idx := offset + x*4
			b.Pixels[idx] = pixel[0]
			b.Pixels[idx+1] = pixel[1]
			b.Pixels[idx+2] = pixel[2]
			b.Pixels[idx+3] = pixel[3]
		}
	}
}

// At returns the color at the specified pixel coordinates.
func (b *Buffer) At(x, y int) Color {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return Color{0, 0, 0, 0}
	}

	idx := y*b.Stride + x*4
	return Color{
		R: b.Pixels[idx+2],
		G: b.Pixels[idx+1],
		B: b.Pixels[idx],
		A: b.Pixels[idx+3],
	}
}

// Set sets the color at the specified pixel coordinates using SrcOver compositing.
func (b *Buffer) Set(x, y int, c Color) {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return
	}

	idx := y*b.Stride + x*4
	BlendPixel(b.Pixels[idx:idx+4], c)
}

// packColor converts a Color to ARGB8888 bytes (little-endian: B, G, R, A).
func packColor(c Color) [4]byte {
	return [4]byte{c.B, c.G, c.R, c.A}
}

// BlendPixel applies SrcOver compositing of src onto dst (4-byte ARGB8888 pixel).
// This is exported for use by other raster packages (e.g., curves, effects).
func BlendPixel(dst []byte, src Color) {
	if src.A == 255 {
		dst[0] = src.B
		dst[1] = src.G
		dst[2] = src.R
		dst[3] = src.A
		return
	}

	if src.A == 0 {
		return
	}

	srcA := uint32(src.A)
	invA := 255 - srcA

	dstR := uint32(dst[2])
	dstG := uint32(dst[1])
	dstB := uint32(dst[0])
	dstA := uint32(dst[3])

	outR := (uint32(src.R)*srcA + dstR*invA) / 255
	outG := (uint32(src.G)*srcA + dstG*invA) / 255
	outB := (uint32(src.B)*srcA + dstB*invA) / 255
	outA := srcA + (dstA*invA)/255

	dst[0] = uint8(outB)
	dst[1] = uint8(outG)
	dst[2] = uint8(outR)
	dst[3] = uint8(outA)
}
