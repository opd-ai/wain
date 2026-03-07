// Package effects implements visual effects for software rasterization.
//
// This package provides CPU-based visual effects for ARGB8888 buffers:
//
//   - Box shadow: Gaussian blur approximation with configurable radius
//   - Linear gradients: color interpolation along a line
//   - Radial gradients: color interpolation from a center point
//   - Scissor clipping: rectangular clipping region for rendering
//
// # Coordinate System
//
// The coordinate system follows standard 2D raster conventions: origin (0,0) at
// top-left, X increases right, Y increases down. Coordinates are in pixels.
//
// # Performance
//
// Effects are optimized for the hot path and avoid allocations during rendering.
// Box shadow uses a separable Gaussian blur approximation (horizontal + vertical
// passes) for O(n) complexity instead of O(n²).
package effects

import (
	"math"

	"github.com/opd-ai/wain/internal/raster/core"
)

// BoxShadow renders a box shadow with Gaussian blur.
// The shadow is drawn at (x, y) with dimensions (width, height).
// The blur radius controls the blur amount (in pixels).
// The color specifies the shadow color and opacity.
// The shadow is drawn underneath the specified rectangle (not on top).
func BoxShadow(buf *core.Buffer, x, y, width, height, blurRadius int, color core.Color) {
	if buf == nil || width <= 0 || height <= 0 || blurRadius <= 0 {
		return
	}

	// Clamp blur radius to reasonable values
	if blurRadius > 50 {
		blurRadius = 50
	}

	// Create shadow area with blur radius padding
	shadowX := x - blurRadius
	shadowY := y - blurRadius
	shadowWidth := width + 2*blurRadius
	shadowHeight := height + 2*blurRadius

	// Clip to buffer bounds
	if shadowX >= buf.Width || shadowY >= buf.Height {
		return
	}
	if shadowX+shadowWidth < 0 || shadowY+shadowHeight < 0 {
		return
	}

	x1 := max(0, shadowX)
	y1 := max(0, shadowY)
	x2 := min(buf.Width, shadowX+shadowWidth)
	y2 := min(buf.Height, shadowY+shadowHeight)

	// Create alpha mask for the shadow
	maskWidth := x2 - x1
	maskHeight := y2 - y1
	if maskWidth <= 0 || maskHeight <= 0 {
		return
	}

	mask := make([]uint8, maskWidth*maskHeight)

	// Fill the core rectangle area in the mask
	coreX1 := max(0, x-shadowX)
	coreY1 := max(0, y-shadowY)
	coreX2 := min(shadowWidth, x+width-shadowX)
	coreY2 := min(shadowHeight, y+height-shadowY)

	for row := coreY1; row < coreY2; row++ {
		for col := coreX1; col < coreX2; col++ {
			if row >= 0 && row < maskHeight && col >= 0 && col < maskWidth {
				mask[row*maskWidth+col] = 255
			}
		}
	}

	// Apply Gaussian blur using box blur approximation (3 passes)
	for pass := 0; pass < 3; pass++ {
		blurHorizontal(mask, maskWidth, maskHeight, blurRadius/3)
		blurVertical(mask, maskWidth, maskHeight, blurRadius/3)
	}

	// Composite the shadow onto the buffer
	for row := 0; row < maskHeight; row++ {
		bufY := y1 + row
		if bufY < 0 || bufY >= buf.Height {
			continue
		}

		for col := 0; col < maskWidth; col++ {
			bufX := x1 + col
			if bufX < 0 || bufX >= buf.Width {
				continue
			}

			maskAlpha := mask[row*maskWidth+col]
			if maskAlpha == 0 {
				continue
			}

			// Modulate shadow color alpha by mask
			shadowAlpha := (uint32(color.A) * uint32(maskAlpha)) / 255

			idx := bufY*buf.Stride + bufX*4
			dstR := uint32(buf.Pixels[idx+2])
			dstG := uint32(buf.Pixels[idx+1])
			dstB := uint32(buf.Pixels[idx])
			dstA := uint32(buf.Pixels[idx+3])

			srcR := uint32(color.R)
			srcG := uint32(color.G)
			srcB := uint32(color.B)

			invA := 255 - shadowAlpha

			outR := (srcR*shadowAlpha + dstR*invA) / 255
			outG := (srcG*shadowAlpha + dstG*invA) / 255
			outB := (srcB*shadowAlpha + dstB*invA) / 255
			outA := shadowAlpha + (dstA*invA)/255

			buf.Pixels[idx] = uint8(outB)
			buf.Pixels[idx+1] = uint8(outG)
			buf.Pixels[idx+2] = uint8(outR)
			buf.Pixels[idx+3] = uint8(outA)
		}
	}
}

// blurHorizontal applies a horizontal box blur to the mask.
func blurHorizontal(mask []uint8, width, height, radius int) {
	if radius <= 0 {
		return
	}

	temp := make([]uint8, width*height)
	copy(temp, mask)

	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			sum := uint32(0)
			count := 0

			for dx := -radius; dx <= radius; dx++ {
				x := col + dx
				if x >= 0 && x < width {
					sum += uint32(temp[row*width+x])
					count++
				}
			}

			if count > 0 {
				mask[row*width+col] = uint8(sum / uint32(count))
			}
		}
	}
}

// blurVertical applies a vertical box blur to the mask.
func blurVertical(mask []uint8, width, height, radius int) {
	if radius <= 0 {
		return
	}

	temp := make([]uint8, width*height)
	copy(temp, mask)

	for col := 0; col < width; col++ {
		for row := 0; row < height; row++ {
			sum := uint32(0)
			count := 0

			for dy := -radius; dy <= radius; dy++ {
				y := row + dy
				if y >= 0 && y < height {
					sum += uint32(temp[y*width+col])
					count++
				}
			}

			if count > 0 {
				mask[row*width+col] = uint8(sum / uint32(count))
			}
		}
	}
}

// LinearGradient fills a rectangular region with a linear gradient.
// The gradient interpolates from startColor to endColor along the line
// from (startX, startY) to (endX, endY).
// The rectangle is defined by (x, y, width, height).
func LinearGradient(buf *core.Buffer, x, y, width, height int,
	startX, startY int, startColor core.Color,
	endX, endY int, endColor core.Color,
) {
	if buf == nil || width <= 0 || height <= 0 {
		return
	}

	// Clip to buffer bounds
	x1 := max(0, x)
	y1 := max(0, y)
	x2 := min(buf.Width, x+width)
	y2 := min(buf.Height, y+height)

	if x1 >= x2 || y1 >= y2 {
		return
	}

	// Compute gradient vector
	dx := float64(endX - startX)
	dy := float64(endY - startY)
	length := math.Sqrt(dx*dx + dy*dy)

	if length < 0.001 {
		// Degenerate gradient, fill with start color
		for row := y1; row < y2; row++ {
			for col := x1; col < x2; col++ {
				setPixel(buf, col, row, startColor)
			}
		}
		return
	}

	// Normalize gradient vector
	dx /= length
	dy /= length

	for row := y1; row < y2; row++ {
		for col := x1; col < x2; col++ {
			// Project point onto gradient line
			px := float64(col - startX)
			py := float64(row - startY)
			t := (px*dx + py*dy) / length

			// Clamp t to [0, 1]
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}

			// Interpolate color
			color := interpolateColor(startColor, endColor, t)
			setPixel(buf, col, row, color)
		}
	}
}

// RadialGradient fills a rectangular region with a radial gradient.
// The gradient interpolates from centerColor to edgeColor, radiating from
// (centerX, centerY) with the specified radius.
// The rectangle is defined by (x, y, width, height).
func RadialGradient(buf *core.Buffer, x, y, width, height int,
	centerX, centerY, radius int,
	centerColor, edgeColor core.Color,
) {
	if buf == nil || width <= 0 || height <= 0 || radius <= 0 {
		return
	}

	// Clip to buffer bounds
	x1 := max(0, x)
	y1 := max(0, y)
	x2 := min(buf.Width, x+width)
	y2 := min(buf.Height, y+height)

	if x1 >= x2 || y1 >= y2 {
		return
	}

	radiusF := float64(radius)

	for row := y1; row < y2; row++ {
		for col := x1; col < x2; col++ {
			// Compute distance from center
			dx := float64(col - centerX)
			dy := float64(row - centerY)
			dist := math.Sqrt(dx*dx + dy*dy)

			// Normalize distance to [0, 1]
			t := dist / radiusF
			if t > 1 {
				t = 1
			}

			// Interpolate color
			color := interpolateColor(centerColor, edgeColor, t)
			setPixel(buf, col, row, color)
		}
	}
}

// Scissor represents a rectangular clipping region.
type Scissor struct {
	X, Y, Width, Height int
}

// NewScissor creates a scissor clipping region.
func NewScissor(x, y, width, height int) Scissor {
	return Scissor{X: x, Y: y, Width: width, Height: height}
}

// Clip returns the intersection of this scissor with the given rectangle.
// This returns the clipped bounds as (x1, y1, x2, y2) or (0, 0, 0, 0) if empty.
func (s Scissor) Clip(x, y, width, height int) (x1, y1, x2, y2 int) {
	x1 = max(s.X, x)
	y1 = max(s.Y, y)
	x2 = min(s.X+s.Width, x+width)
	y2 = min(s.Y+s.Height, y+height)

	if x1 >= x2 || y1 >= y2 {
		return 0, 0, 0, 0
	}

	return x1, y1, x2, y2
}

// Contains checks if the point (x, y) is inside the scissor region.
func (s Scissor) Contains(x, y int) bool {
	return x >= s.X && x < s.X+s.Width && y >= s.Y && y < s.Y+s.Height
}

// interpolateColor linearly interpolates between two colors.
// t should be in the range [0, 1].
func interpolateColor(c1, c2 core.Color, t float64) core.Color {
	inv := 1.0 - t

	return core.Color{
		R: uint8(float64(c1.R)*inv + float64(c2.R)*t + 0.5),
		G: uint8(float64(c1.G)*inv + float64(c2.G)*t + 0.5),
		B: uint8(float64(c1.B)*inv + float64(c2.B)*t + 0.5),
		A: uint8(float64(c1.A)*inv + float64(c2.A)*t + 0.5),
	}
}

// setPixel sets a pixel with SrcOver compositing.
func setPixel(buf *core.Buffer, x, y int, color core.Color) {
	if x < 0 || x >= buf.Width || y < 0 || y >= buf.Height {
		return
	}

	idx := y*buf.Stride + x*4

	srcA := uint32(color.A)
	if srcA == 255 {
		buf.Pixels[idx] = color.B
		buf.Pixels[idx+1] = color.G
		buf.Pixels[idx+2] = color.R
		buf.Pixels[idx+3] = color.A
		return
	}

	if srcA == 0 {
		return
	}

	dstR := uint32(buf.Pixels[idx+2])
	dstG := uint32(buf.Pixels[idx+1])
	dstB := uint32(buf.Pixels[idx])
	dstA := uint32(buf.Pixels[idx+3])

	invA := 255 - srcA

	outR := (uint32(color.R)*srcA + dstR*invA) / 255
	outG := (uint32(color.G)*srcA + dstG*invA) / 255
	outB := (uint32(color.B)*srcA + dstB*invA) / 255
	outA := srcA + (dstA*invA)/255

	buf.Pixels[idx] = uint8(outB)
	buf.Pixels[idx+1] = uint8(outG)
	buf.Pixels[idx+2] = uint8(outR)
	buf.Pixels[idx+3] = uint8(outA)
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
