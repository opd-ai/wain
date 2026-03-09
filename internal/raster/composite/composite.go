// Package composite implements image blitting and alpha compositing operations.
//
// This package provides CPU-based compositing operations for ARGB8888 buffers:
//
//   - Image blitting: copy rectangular regions between buffers
//   - Bilinear filtering: smooth image scaling and interpolation
//   - Alpha compositing: Porter-Duff SrcOver blending
//
// # Coordinate System
//
// The coordinate system follows standard 2D raster conventions: origin (0,0) at
// top-left, X increases right, Y increases down. Coordinates are in pixels.
//
// # Alpha Compositing
//
// All compositing operations use Porter-Duff SrcOver:
//
//	result.rgb = src.rgb * src.a + dst.rgb * (1 - src.a)
//	result.a = src.a + dst.a * (1 - src.a)
//
// # Performance
//
// All functions are optimized for the hot path and avoid allocations during
// rendering. Clipping is handled automatically for out-of-bounds regions.
package composite

import (
	"github.com/opd-ai/wain/internal/raster/core"
)

// Blit copies a rectangular region from src to dst with no filtering.
// The source rectangle is defined by (srcX, srcY, width, height).
// The destination position is (dstX, dstY).
// Coordinates are automatically clipped to buffer bounds.
func Blit(dst *core.Buffer, dstX, dstY int, src *core.Buffer, srcX, srcY, width, height int) {
	if dst == nil || src == nil || width <= 0 || height <= 0 {
		return
	}

	srcX1, srcY1, copyWidth, copyHeight := calculateClippedRegion(
		dst, dstX, dstY, src, srcX, srcY, width, height,
	)
	if copyWidth <= 0 || copyHeight <= 0 {
		return
	}

	blitRows(dst, src, srcX1, srcY1, dstX, dstY, copyWidth, copyHeight)
}

// calculateClippedRegion computes clipped source coordinates and copy dimensions.
func calculateClippedRegion(dst *core.Buffer, dstX, dstY int, src *core.Buffer, srcX, srcY, width, height int) (srcX1, srcY1, copyWidth, copyHeight int) {
	srcX1 = max(0, srcX)
	srcY1 = max(0, srcY)
	srcX2 := min(src.Width, srcX+width)
	srcY2 := min(src.Height, srcY+height)

	if srcX1 >= srcX2 || srcY1 >= srcY2 {
		return 0, 0, 0, 0
	}

	dstX1 := max(0, dstX+(srcX1-srcX))
	dstY1 := max(0, dstY+(srcY1-srcY))
	dstX2 := min(dst.Width, dstX+(srcX2-srcX))
	dstY2 := min(dst.Height, dstY+(srcY2-srcY))

	if dstX1 >= dstX2 || dstY1 >= dstY2 {
		return 0, 0, 0, 0
	}

	srcX1 += dstX1 - (dstX + (srcX1 - srcX))
	srcY1 += dstY1 - (dstY + (srcY1 - srcY))
	return srcX1, srcY1, dstX2 - dstX1, dstY2 - dstY1
}

// blitRows copies pixel rows from source to destination with alpha blending.
func blitRows(dst, src *core.Buffer, srcX, srcY, dstX, dstY, width, height int) {
	for row := 0; row < height; row++ {
		srcOffset := (srcY+row)*src.Stride + srcX*4
		dstOffset := (dstY+row)*dst.Stride + dstX*4
		blitRow(dst.Pixels[dstOffset:], src.Pixels[srcOffset:], width)
	}
}

// blitRow copies a single row of pixels with alpha blending.
func blitRow(dstRow, srcRow []byte, width int) {
	for col := 0; col < width; col++ {
		srcIdx := col * 4
		dstIdx := col * 4
		srcA := srcRow[srcIdx+3]

		if srcA == 0 {
			continue
		}
		if srcA == 255 {
			copy(dstRow[dstIdx:dstIdx+4], srcRow[srcIdx:srcIdx+4])
			continue
		}
		blendPixelDirect(dstRow[dstIdx:dstIdx+4], srcRow[srcIdx:srcIdx+4])
	}
}

// BlitScaled copies a rectangular region from src to dst with bilinear filtering.
// The source rectangle is defined by (srcX, srcY, srcWidth, srcHeight).
// The destination rectangle is defined by (dstX, dstY, dstWidth, dstHeight).
// Bilinear interpolation is applied for smooth scaling.
// Coordinates are automatically clipped to buffer bounds.
func BlitScaled(dst *core.Buffer, dstX, dstY, dstWidth, dstHeight int,
	src *core.Buffer, srcX, srcY, srcWidth, srcHeight int,
) {
	if dst == nil || src == nil || dstWidth <= 0 || dstHeight <= 0 || srcWidth <= 0 || srcHeight <= 0 {
		return
	}

	dstX1 := max(0, dstX)
	dstY1 := max(0, dstY)
	dstX2 := min(dst.Width, dstX+dstWidth)
	dstY2 := min(dst.Height, dstY+dstHeight)

	if dstX1 >= dstX2 || dstY1 >= dstY2 {
		return
	}

	scaleX := float64(srcWidth) / float64(dstWidth)
	scaleY := float64(srcHeight) / float64(dstHeight)

	for dstRow := dstY1; dstRow < dstY2; dstRow++ {
		srcYf := (float64(dstRow-dstY) + 0.5) * scaleY
		srcYi := int(srcYf)
		srcYfrac := srcYf - float64(srcYi)

		srcY0 := srcY + srcYi
		srcY1 := srcY0 + 1

		if srcY0 < 0 || srcY1 >= src.Height {
			continue
		}

		for dstCol := dstX1; dstCol < dstX2; dstCol++ {
			srcXf := (float64(dstCol-dstX) + 0.5) * scaleX
			srcXi := int(srcXf)
			srcXfrac := srcXf - float64(srcXi)

			srcX0 := srcX + srcXi
			srcX1 := srcX0 + 1

			if srcX0 < 0 || srcX1 >= src.Width {
				continue
			}

			p00 := samplePixel(src, srcX0, srcY0)
			p10 := samplePixel(src, srcX1, srcY0)
			p01 := samplePixel(src, srcX0, srcY1)
			p11 := samplePixel(src, srcX1, srcY1)

			result := bilinearInterpolate(p00, p10, p01, p11, srcXfrac, srcYfrac)

			dstIdx := dstRow*dst.Stride + dstCol*4
			blendPixelDirect(dst.Pixels[dstIdx:dstIdx+4], result[:])
		}
	}
}

// samplePixel reads a pixel from the buffer at (x, y) and returns it as [4]byte.
func samplePixel(buf *core.Buffer, xPos, yPos int) [4]byte {
	if xPos < 0 || xPos >= buf.Width || yPos < 0 || yPos >= buf.Height {
		return [4]byte{0, 0, 0, 0}
	}
	idx := yPos*buf.Stride + xPos*4
	return [4]byte{
		buf.Pixels[idx],
		buf.Pixels[idx+1],
		buf.Pixels[idx+2],
		buf.Pixels[idx+3],
	}
}

// bilinearInterpolate performs bilinear interpolation of four pixels.
// p00, p10, p01, p11 are the four corner pixels in ARGB8888 format.
// fracX and fracY are the fractional coordinates (0.0 to 1.0).
func bilinearInterpolate(p00, p10, p01, p11 [4]byte, fracX, fracY float64) [4]byte {
	invFracX := 1.0 - fracX
	invFracY := 1.0 - fracY

	w00 := invFracX * invFracY
	w10 := fracX * invFracY
	w01 := invFracX * fracY
	w11 := fracX * fracY

	b := w00*float64(p00[0]) + w10*float64(p10[0]) + w01*float64(p01[0]) + w11*float64(p11[0])
	g := w00*float64(p00[1]) + w10*float64(p10[1]) + w01*float64(p01[1]) + w11*float64(p11[1])
	r := w00*float64(p00[2]) + w10*float64(p10[2]) + w01*float64(p01[2]) + w11*float64(p11[2])
	a := w00*float64(p00[3]) + w10*float64(p10[3]) + w01*float64(p01[3]) + w11*float64(p11[3])

	return [4]byte{
		uint8(b + 0.5),
		uint8(g + 0.5),
		uint8(r + 0.5),
		uint8(a + 0.5),
	}
}

// blendPixelDirect applies SrcOver compositing of src onto dst.
// Both src and dst are 4-byte ARGB8888 pixels (little-endian: B, G, R, A).
// This function modifies dst in place and avoids allocations.
func blendPixelDirect(dst, src []byte) {
	srcA := uint32(src[3])
	if srcA == 255 {
		dst[0] = src[0]
		dst[1] = src[1]
		dst[2] = src[2]
		dst[3] = src[3]
		return
	}

	if srcA == 0 {
		return
	}

	invA := 255 - srcA

	dstR := uint32(dst[2])
	dstG := uint32(dst[1])
	dstB := uint32(dst[0])
	dstA := uint32(dst[3])

	outR := (uint32(src[2])*srcA + dstR*invA) / 255
	outG := (uint32(src[1])*srcA + dstG*invA) / 255
	outB := (uint32(src[0])*srcA + dstB*invA) / 255
	outA := srcA + (dstA*invA)/255

	dst[0] = uint8(outB)
	dst[1] = uint8(outG)
	dst[2] = uint8(outR)
	dst[3] = uint8(outA)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
