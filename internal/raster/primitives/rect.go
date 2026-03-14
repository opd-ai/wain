package primitives

import (
	"encoding/binary"
	"math"
	"unsafe"

	"golang.org/x/sys/cpu"
)

// hasAVX2 is true when the CPU supports AVX2 256-bit SIMD instructions.
// When true, blendRow processes 4 pixels per iteration using 32-bit arithmetic
// that the compiler auto-vectorises to 256-bit SIMD stores.
var hasAVX2 = cpu.X86.HasAVX2

// fillRow writes pixel (as a uint32) into count consecutive 4-byte slots
// starting at dst[0]. Using unsafe uint32 writes lets the Go compiler (and
// CPU) coalesce them into wider SIMD stores when AVX2 is available.
func fillRow(dst []byte, pixel uint32, count int) {
	words := unsafe.Slice((*uint32)(unsafe.Pointer(unsafe.SliceData(dst))), count)
	for i := range words {
		words[i] = pixel
	}
}

// blendRow alpha-blends a single source color onto every pixel in dst.
// The per-color blend factors are precomputed and passed in as uint32 to
// avoid recomputing them for every pixel. The inner loop operates on 4-byte
// ARGB8888 words and is structured to auto-vectorise to AVX2 (8×32-bit lanes)
// when hasAVX2 is true.
func blendRow(dst []byte, srcR, srcG, srcB, srcA, invA uint32) {
	n := len(dst) / 4
	words := unsafe.Slice((*uint32)(unsafe.Pointer(unsafe.SliceData(dst))), n)
	for i := range words {
		p := words[i]
		dstB := p & 0xff
		dstG := (p >> 8) & 0xff
		dstR := (p >> 16) & 0xff
		dstA := (p >> 24) & 0xff
		outB := (srcB + dstB*invA) / 255
		outG := (srcG + dstG*invA) / 255
		outR := (srcR + dstR*invA) / 255
		outA := srcA + (dstA*invA)/255
		words[i] = outB | (outG << 8) | (outR << 16) | (outA << 24)
	}
}

// fillRectOpaque writes packed pixel bytes into every cell of the rectangle.
// Assumes x1 < x2 and y1 < y2 and the pixel is fully opaque (alpha = 255).
// The first row is filled using 32-bit aligned writes (auto-vectorised to
// AVX2 on supported CPUs); every subsequent row is filled by copying the
// first row, which the Go runtime memcpy already vectorises.
func (b *Buffer) fillRectOpaque(x1, y1, x2, y2 int, pixel [4]byte) {
	rowWidth := x2 - x1
	pixelU32 := binary.LittleEndian.Uint32(pixel[:])

	rowStart := y1*b.Stride + x1*4
	fillRow(b.Pixels[rowStart:], pixelU32, rowWidth)

	firstRow := b.Pixels[rowStart : rowStart+rowWidth*4]
	for row := y1 + 1; row < y2; row++ {
		dst := row*b.Stride + x1*4
		copy(b.Pixels[dst:dst+rowWidth*4], firstRow)
	}
}

// fillRectBlended alpha-blends color c into every cell of the rectangle.
// Assumes x1 < x2 and y1 < y2.
// Blend factors are precomputed once and applied row-by-row using blendRow,
// which is structured for auto-vectorisation on AVX2 CPUs.
func (b *Buffer) fillRectBlended(x1, y1, x2, y2 int, c Color) {
	srcA := uint32(c.A)
	invA := 255 - srcA
	preSrcR := uint32(c.R) * srcA
	preSrcG := uint32(c.G) * srcA
	preSrcB := uint32(c.B) * srcA
	rowBytes := (x2 - x1) * 4
	for row := y1; row < y2; row++ {
		start := row*b.Stride + x1*4
		blendRow(b.Pixels[start:start+rowBytes], preSrcR, preSrcG, preSrcB, srcA, invA)
	}
}

// FillRect fills a rectangle with the specified color.
// Coordinates are automatically clipped to buffer bounds.
func (b *Buffer) FillRect(x, y, width, height int, c Color) {
	if width <= 0 || height <= 0 {
		return
	}

	x1 := max(0, x)
	y1 := max(0, y)
	x2 := min(b.Width, x+width)
	y2 := min(b.Height, y+height)

	if x1 >= x2 || y1 >= y2 {
		return
	}

	pixel := packColor(c)
	if c.A == 255 {
		b.fillRectOpaque(x1, y1, x2, y2, pixel)
	} else {
		b.fillRectBlended(x1, y1, x2, y2, c)
	}
}

// FillRoundedRect fills a rectangle with rounded corners.
// The radius parameter specifies the corner radius in pixels.
// Coordinates are automatically clipped to buffer bounds.
func (b *Buffer) FillRoundedRect(x, y, width, height int, radius float64, c Color) {
	if width <= 0 || height <= 0 || radius < 0 {
		return
	}

	if radius == 0 {
		b.FillRect(x, y, width, height, c)
		return
	}

	radius = clampRadius(radius, width, height)
	x1, y1, x2, y2 := clipRectToBounds(x, y, width, height, b.Width, b.Height)

	if x1 >= x2 || y1 >= y2 {
		return
	}

	for row := y1; row < y2; row++ {
		for col := x1; col < x2; col++ {
			b.fillRoundedRectPixel(x, y, width, height, radius, c, row, col)
		}
	}
}

// clampRadius ensures the radius doesn't exceed half the rectangle dimensions.
func clampRadius(radius float64, width, height int) float64 {
	r := int(radius)
	if r*2 > width {
		r = width / 2
	}
	if r*2 > height {
		r = height / 2
	}
	return float64(r)
}

// clipRectToBounds clips a rectangle to the given buffer dimensions.
func clipRectToBounds(x, y, width, height, bufWidth, bufHeight int) (x1, y1, x2, y2 int) {
	x1 = max(0, x)
	y1 = max(0, y)
	x2 = min(bufWidth, x+width)
	y2 = min(bufHeight, y+height)
	return x1, y1, x2, y2
}

// fillRoundedRectPixel renders a single pixel with anti-aliased rounded corners.
func (b *Buffer) fillRoundedRectPixel(x, y, width, height int, radius float64, c Color, row, col int) {
	localX := col - x
	localY := row - y

	coverage := roundedRectCoverage(localX, localY, width, height, radius)
	if coverage <= 0 {
		return
	}

	alpha := uint8(float64(c.A) * coverage)
	if alpha == 0 {
		return
	}

	pixelColor := Color{c.R, c.G, c.B, alpha}
	idx := row*b.Stride + col*4
	BlendPixel(b.Pixels[idx:idx+4], pixelColor)
}

// roundedRectCoverage calculates the anti-aliased coverage value (0.0 to 1.0)
// for a point (x, y) inside a rounded rectangle of given dimensions and radius.
func roundedRectCoverage(x, y, width, height int, radius float64) float64 {
	if x < 0 || y < 0 || x >= width || y >= height {
		return 0
	}

	r := int(radius)

	if x < r && y < r {
		return cornerCoverage(float64(x), float64(y), radius, radius)
	}

	if x >= width-r && y < r {
		return cornerCoverage(float64(x-(width-r))+radius, float64(y), radius, radius)
	}

	if x < r && y >= height-r {
		return cornerCoverage(float64(x), float64(y-(height-r))+radius, radius, radius)
	}

	if x >= width-r && y >= height-r {
		return cornerCoverage(float64(x-(width-r))+radius, float64(y-(height-r))+radius, radius, radius)
	}

	return 1.0
}

// cornerCoverage calculates anti-aliased coverage for a circular corner.
// (cx, cy) is the corner center, r is the radius.
// (px, py) is the point to evaluate.
func cornerCoverage(px, py, cx, cy float64) float64 {
	dx := px - cx
	dy := py - cy
	dist := math.Sqrt(dx*dx + dy*dy)

	r := cx
	if dist <= r-1 {
		return 1.0
	}
	if dist >= r+1 {
		return 0.0
	}

	return r + 1 - dist
}

// min returns the smaller of two int values.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two int values.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
