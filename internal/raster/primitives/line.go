package primitives

import "math"

// DrawLine draws an anti-aliased line segment from (x0, y0) to (x1, y1).
// The line has the specified width in pixels and is rendered with the given color.
func (b *Buffer) DrawLine(x0, y0, x1, y1 int, width float64, c Color) {
	if width <= 0 {
		return
	}

	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	length := math.Sqrt(dx*dx + dy*dy)

	if length < 0.5 {
		b.Set(x0, y0, c)
		return
	}

	halfWidth := width / 2.0

	dirX := dx / length
	dirY := dy / length

	perpX := -dirY
	perpY := dirX

	minX := int(math.Floor(min2f(float64(x0), float64(x1)) - halfWidth - 1))
	maxX := int(math.Ceil(max2f(float64(x0), float64(x1)) + halfWidth + 1))
	minY := int(math.Floor(min2f(float64(y0), float64(y1)) - halfWidth - 1))
	maxY := int(math.Ceil(max2f(float64(y0), float64(y1)) + halfWidth + 1))

	minX = max(0, minX)
	maxX = min(b.Width, maxX)
	minY = max(0, minY)
	maxY = min(b.Height, maxY)

	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			coverage := lineCoverage(float64(x), float64(y), float64(x0), float64(y0), dirX, dirY, perpX, perpY, length, halfWidth)
			if coverage <= 0 {
				continue
			}

			alpha := uint8(float64(c.A) * coverage)
			if alpha == 0 {
				continue
			}

			pixelColor := Color{c.R, c.G, c.B, alpha}
			idx := y*b.Stride + x*4
			BlendPixel(b.Pixels[idx:idx+4], pixelColor)
		}
	}
}

// lineCoverage calculates the anti-aliased coverage for a pixel at (px, py)
// for a line segment from (x0, y0) with direction (dirX, dirY) and perpendicular (perpX, perpY).
func lineCoverage(px, py, x0, y0, dirX, dirY, perpX, perpY, length, halfWidth float64) float64 {
	dx := px - x0
	dy := py - y0

	parallel := dx*dirX + dy*dirY
	if parallel < -1 || parallel > length+1 {
		return 0
	}

	perp := math.Abs(dx*perpX + dy*perpY)
	if perp > halfWidth+1 {
		return 0
	}

	coverage := perpendicularCoverage(perp, halfWidth)
	coverage = startCapCoverage(parallel, dx, dy, halfWidth, coverage)
	coverage = endCapCoverage(parallel, px, py, x0, y0, dirX, dirY, length, halfWidth, coverage)

	return clamp(coverage, 0, 1)
}

// perpendicularCoverage computes anti-aliased coverage based on perpendicular distance.
func perpendicularCoverage(perp, halfWidth float64) float64 {
	if perp > halfWidth-1 {
		return halfWidth + 1 - perp
	}
	return 1.0
}

// startCapCoverage computes anti-aliased coverage for the line's start cap.
func startCapCoverage(parallel, dx, dy, halfWidth, coverage float64) float64 {
	if parallel >= 1 {
		return coverage
	}
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist > halfWidth+1 {
		return 0
	}
	if dist > halfWidth-1 {
		return min2f(coverage, halfWidth+1-dist)
	}
	return coverage
}

// endCapCoverage computes anti-aliased coverage for the line's end cap.
func endCapCoverage(parallel, px, py, x0, y0, dirX, dirY, length, halfWidth, coverage float64) float64 {
	if parallel <= length-1 {
		return coverage
	}
	dx2 := px - (x0 + dirX*length)
	dy2 := py - (y0 + dirY*length)
	dist := math.Sqrt(dx2*dx2 + dy2*dy2)
	if dist > halfWidth+1 {
		return 0
	}
	if dist > halfWidth-1 {
		return min2f(coverage, halfWidth+1-dist)
	}
	return coverage
}

func min2f(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max2f(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
