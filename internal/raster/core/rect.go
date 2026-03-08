package core

import "math"

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
		for row := y1; row < y2; row++ {
			offset := row*b.Stride + x1*4
			for col := x1; col < x2; col++ {
				idx := offset + (col-x1)*4
				b.Pixels[idx] = pixel[0]
				b.Pixels[idx+1] = pixel[1]
				b.Pixels[idx+2] = pixel[2]
				b.Pixels[idx+3] = pixel[3]
			}
		}
	} else {
		for row := y1; row < y2; row++ {
			offset := row * b.Stride
			for col := x1; col < x2; col++ {
				idx := offset + col*4
				BlendPixel(b.Pixels[idx:idx+4], c)
			}
		}
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

	r := int(radius)
	if r*2 > width {
		r = width / 2
	}
	if r*2 > height {
		r = height / 2
	}
	radius = float64(r)

	x1 := max(0, x)
	y1 := max(0, y)
	x2 := min(b.Width, x+width)
	y2 := min(b.Height, y+height)

	if x1 >= x2 || y1 >= y2 {
		return
	}

	for row := y1; row < y2; row++ {
		for col := x1; col < x2; col++ {
			localX := col - x
			localY := row - y

			coverage := roundedRectCoverage(localX, localY, width, height, radius)
			if coverage <= 0 {
				continue
			}

			alpha := uint8(float64(c.A) * coverage)
			if alpha == 0 {
				continue
			}

			pixelColor := Color{c.R, c.G, c.B, alpha}
			idx := row*b.Stride + col*4
			BlendPixel(b.Pixels[idx:idx+4], pixelColor)
		}
	}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
