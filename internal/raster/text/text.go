package text

import (
	"math"

	"github.com/opd-ai/wain/internal/raster/core"
)

// DrawText renders a text string to the buffer using SDF-based rendering.
//
// Parameters:
//   - buf: The target buffer to render into
//   - text: The string to render (supports ASCII printable characters)
//   - x, y: The baseline starting position in pixels
//   - size: The font size in pixels (height from baseline to ascender)
//   - color: The text color
//   - atlas: The font atlas containing glyph data
//
// The text is rendered left-to-right starting at (x, y). Characters not in
// the atlas are replaced with a replacement glyph.
//
// SDF rendering provides smooth antialiasing at any scale. The size parameter
// controls the final rendered height.
func DrawText(buf *core.Buffer, text string, x, y, size float64, color core.Color, atlas *Atlas) {
	if atlas == nil || buf == nil {
		return
	}

	// Calculate scale from atlas baseline to requested size
	scale := size / atlas.Baseline

	cursorX := x
	cursorY := y

	for _, r := range text {
		glyph, err := atlas.GetGlyph(r)
		if err != nil {
			continue // Skip unknown glyphs
		}

		drawGlyph(buf, glyph, cursorX, cursorY, scale, color, atlas)
		cursorX += glyph.Advance * scale
	}
}

// drawGlyph renders a single glyph to the buffer.
func drawGlyph(buf *core.Buffer, g *Glyph, x, y, scale float64, color core.Color, atlas *Atlas) {
	// Calculate scaled glyph dimensions
	glyphX := x + g.OffsetX*scale
	glyphY := y + g.OffsetY*scale
	glyphW := float64(g.Width) * scale
	glyphH := float64(g.Height) * scale

	// Integer bounds for pixel iteration
	x0 := int(math.Floor(glyphX))
	y0 := int(math.Floor(glyphY))
	x1 := int(math.Ceil(glyphX + glyphW))
	y1 := int(math.Ceil(glyphY + glyphH))

	// Clip to buffer bounds
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > buf.Width {
		x1 = buf.Width
	}
	if y1 > buf.Height {
		y1 = buf.Height
	}

	// Render each pixel
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			// Map buffer pixel to atlas coordinates
			normX := (float64(px) - glyphX) / glyphW
			normY := (float64(py) - glyphY) / glyphH

			if normX < 0 || normX > 1 || normY < 0 || normY > 1 {
				continue
			}

			// Sample SDF at atlas position
			atlasX := int(normX*float64(g.Width)) + g.X
			atlasY := int(normY*float64(g.Height)) + g.Y

			sdfValue := atlas.SampleSDF(atlasX, atlasY)

			// Convert SDF to coverage (alpha)
			// SDF: 128 = edge, >128 = inside, <128 = outside
			// Apply smoothing for antialiasing
			alpha := sdfToCoverage(sdfValue, scale)

			if alpha > 0 {
				blendPixel(buf, px, py, color, alpha)
			}
		}
	}
}

// sdfToCoverage converts an SDF value to an alpha coverage value.
//
// The SDF value is in range [0, 255] with 128 representing the edge.
// Values >128 are inside the glyph, <128 are outside.
//
// Scale affects the smoothing range: larger scales need wider smoothing.
func sdfToCoverage(sdfValue uint8, scale float64) uint8 {
	// Center around 0 (edge = 0, inside = positive, outside = negative)
	dist := float64(sdfValue) - 128.0

	// Smoothing range based on scale (wider for smaller text)
	smoothRange := 8.0 / math.Max(scale, 0.5)

	// Apply smooth step for antialiasing
	coverage := smoothstep(-smoothRange, smoothRange, dist)

	return uint8(coverage * 255.0)
}

// smoothstep implements the smoothstep function for smooth interpolation.
//
// Returns a value smoothly interpolated from 0 to 1 as x moves from edge0 to edge1.
// Returns 0 if x <= edge0, 1 if x >= edge1.
func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

// clamp restricts a value to the range [min, max].
func clamp(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

// blendPixel blends a color onto a buffer pixel using SrcOver compositing.
func blendPixel(buf *core.Buffer, xPos, yPos int, color core.Color, alpha uint8) {
	if xPos < 0 || xPos >= buf.Width || yPos < 0 || yPos >= buf.Height {
		return
	}

	// Get destination pixel
	offset := yPos*buf.Stride + xPos*4
	dstB := buf.Pixels[offset]
	dstG := buf.Pixels[offset+1]
	dstR := buf.Pixels[offset+2]
	dstA := buf.Pixels[offset+3]

	// Source alpha is modulated by the SDF coverage
	srcA := uint32(color.A) * uint32(alpha) / 255

	if srcA == 0 {
		return
	}

	// Porter-Duff SrcOver blending
	invSrcA := 255 - srcA

	outR := (uint32(color.R)*srcA + uint32(dstR)*invSrcA) / 255
	outG := (uint32(color.G)*srcA + uint32(dstG)*invSrcA) / 255
	outB := (uint32(color.B)*srcA + uint32(dstB)*invSrcA) / 255
	outA := srcA + (uint32(dstA)*invSrcA)/255

	buf.Pixels[offset] = uint8(outB)
	buf.Pixels[offset+1] = uint8(outG)
	buf.Pixels[offset+2] = uint8(outR)
	buf.Pixels[offset+3] = uint8(outA)
}

// MeasureText calculates the width and height of a text string.
//
// Returns the bounding box dimensions in pixels for the given text and size.
// The height is always the line height of the font.
func MeasureText(text string, size float64, atlas *Atlas) (width, height float64) {
	if atlas == nil {
		return 0, 0
	}

	scale := size / atlas.Baseline
	totalAdvance := 0.0

	for _, r := range text {
		glyph, err := atlas.GetGlyph(r)
		if err != nil {
			continue
		}
		totalAdvance += glyph.Advance * scale
	}

	return totalAdvance, atlas.LineHeight * scale
}
