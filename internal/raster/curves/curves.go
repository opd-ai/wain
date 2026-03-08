// Package curves implements Bezier curve and arc rasterization for 2D software rendering.
//
// This package extends the core rasterizer with support for:
//
//   - Quadratic Bezier curves: three control points defining a smooth curve
//   - Cubic Bezier curves: four control points defining a more complex smooth curve
//   - Elliptical arcs: portions of ellipses with configurable start/end angles
//
// All curve rendering uses adaptive subdivision to maintain quality while minimizing
// computational cost. The subdivision depth is determined by the curve's flatness.
//
// # Coordinate System
//
// Uses the same coordinate system as the core package: origin (0,0) at top-left,
// X increases right, Y increases down. All coordinates are in pixels.
//
// # Rendering Quality
//
// Curves are subdivided until each segment is approximately flat (within 0.5 pixels
// of the true curve). This provides anti-aliased output without excessive computation.
package curves

import (
	"math"

	"github.com/opd-ai/wain/internal/raster/core"
)

const (
	// flatnessTolerance is the maximum distance (in pixels) a subdivided curve
	// segment can deviate from the true curve before further subdivision.
	flatnessTolerance = 0.5

	// maxSubdivisionDepth prevents infinite recursion for degenerate curves.
	maxSubdivisionDepth = 16
)

// Point represents a 2D point with floating-point coordinates.
type Point struct {
	X, Y float64
}

// DrawQuadraticBezier renders a quadratic Bezier curve from p0 to p2 with control point p1.
// The curve is drawn with the specified width and color using anti-aliased line segments.
//
// A quadratic Bezier is defined by the parametric equation:
//
//	B(t) = (1-t)²·p0 + 2(1-t)t·p1 + t²·p2, where t ∈ [0, 1]
//
// The curve is adaptively subdivided until each segment is approximately flat.
func DrawQuadraticBezier(b *core.Buffer, p0, p1, p2 Point, width float64, c core.Color) {
	if width <= 0 {
		return
	}
	subdivideQuadratic(b, p0, p1, p2, width, c, 0)
}

// DrawCubicBezier renders a cubic Bezier curve from p0 to p3 with control points p1 and p2.
// The curve is drawn with the specified width and color using anti-aliased line segments.
//
// A cubic Bezier is defined by the parametric equation:
//
//	B(t) = (1-t)³·p0 + 3(1-t)²t·p1 + 3(1-t)t²·p2 + t³·p3, where t ∈ [0, 1]
//
// The curve is adaptively subdivided until each segment is approximately flat.
func DrawCubicBezier(b *core.Buffer, p0, p1, p2, p3 Point, width float64, c core.Color) {
	if width <= 0 {
		return
	}
	subdivideCubic(b, p0, p1, p2, p3, width, c, 0)
}

// DrawArc renders an elliptical arc centered at (cx, cy) with radii rx and ry.
// The arc spans from startAngle to endAngle (in radians, clockwise from positive X axis).
// The arc is drawn with the specified width and color.
//
// Angles are measured clockwise from the positive X axis (right direction).
// For example: 0 = right, π/2 = down, π = left, 3π/2 = up.
func DrawArc(b *core.Buffer, cx, cy, rx, ry, startAngle, endAngle, width float64, c core.Color) {
	if width <= 0 || rx <= 0 || ry <= 0 {
		return
	}

	// Normalize angles to [0, 2π)
	startAngle = normalizeAngle(startAngle)
	endAngle = normalizeAngle(endAngle)

	// Handle wraparound
	if endAngle <= startAngle {
		endAngle += 2 * math.Pi
	}

	// Calculate the number of segments based on arc length and radius
	arcLength := (endAngle - startAngle) * math.Max(rx, ry)
	segments := int(math.Ceil(arcLength / 5.0))
	if segments < 4 {
		segments = 4
	}
	if segments > 360 {
		segments = 360
	}

	// Generate points along the arc
	dt := (endAngle - startAngle) / float64(segments)
	prevX := cx + rx*math.Cos(startAngle)
	prevY := cy + ry*math.Sin(startAngle)

	for i := 1; i <= segments; i++ {
		t := startAngle + dt*float64(i)
		currX := cx + rx*math.Cos(t)
		currY := cy + ry*math.Sin(t)

		b.DrawLine(
			int(math.Round(prevX)),
			int(math.Round(prevY)),
			int(math.Round(currX)),
			int(math.Round(currY)),
			width,
			c,
		)

		prevX = currX
		prevY = currY
	}
}

// FillArc renders a filled elliptical arc (pie slice) centered at (cx, cy).
// The arc is filled from the center point to the arc perimeter, creating a wedge shape.
func FillArc(b *core.Buffer, cx, cy, rx, ry, startAngle, endAngle float64, c core.Color) {
	if rx <= 0 || ry <= 0 {
		return
	}

	// Normalize angles
	startAngle = normalizeAngle(startAngle)
	endAngle = normalizeAngle(endAngle)

	if endAngle <= startAngle {
		endAngle += 2 * math.Pi
	}

	// Calculate bounding box with some margin for anti-aliasing
	minX := int(math.Floor(cx - rx - 1))
	maxX := int(math.Ceil(cx + rx + 1))
	minY := int(math.Floor(cy - ry - 1))
	maxY := int(math.Ceil(cy + ry + 1))

	// Clip to buffer bounds
	minX = max(0, minX)
	maxX = min(b.Width, maxX)
	minY = max(0, minY)
	maxY = min(b.Height, maxY)

	// Rasterize each pixel in the bounding box
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			coverage := arcCoverage(float64(x), float64(y), cx, cy, rx, ry, startAngle, endAngle)
			if coverage <= 0 {
				continue
			}

			alpha := uint8(float64(c.A) * coverage)
			if alpha == 0 {
				continue
			}

			pixelColor := core.Color{R: c.R, G: c.G, B: c.B, A: alpha}
			idx := y*b.Stride + x*4
			core.BlendPixel(b.Pixels[idx:idx+4], pixelColor)
		}
	}
}

// subdivideQuadratic recursively subdivides a quadratic Bezier curve until it's flat enough.
func subdivideQuadratic(buf *core.Buffer, p0, p1, p2 Point, width float64, c core.Color, depth int) {
	if depth >= maxSubdivisionDepth || isQuadraticFlat(p0, p1, p2) {
		buf.DrawLine(
			int(math.Round(p0.X)),
			int(math.Round(p0.Y)),
			int(math.Round(p2.X)),
			int(math.Round(p2.Y)),
			width,
			c,
		)
		return
	}

	// De Casteljau's algorithm: subdivide at t=0.5
	q0 := Point{(p0.X + p1.X) / 2, (p0.Y + p1.Y) / 2}
	q1 := Point{(p1.X + p2.X) / 2, (p1.Y + p2.Y) / 2}
	r := Point{(q0.X + q1.X) / 2, (q0.Y + q1.Y) / 2}

	subdivideQuadratic(buf, p0, q0, r, width, c, depth+1)
	subdivideQuadratic(buf, r, q1, p2, width, c, depth+1)
}

// subdivideCubic recursively subdivides a cubic Bezier curve until it's flat enough.
func subdivideCubic(buf *core.Buffer, p0, p1, p2, p3 Point, width float64, c core.Color, depth int) {
	if depth >= maxSubdivisionDepth || isCubicFlat(p0, p1, p2, p3) {
		buf.DrawLine(
			int(math.Round(p0.X)),
			int(math.Round(p0.Y)),
			int(math.Round(p3.X)),
			int(math.Round(p3.Y)),
			width,
			c,
		)
		return
	}

	// De Casteljau's algorithm: subdivide at t=0.5
	q0 := Point{(p0.X + p1.X) / 2, (p0.Y + p1.Y) / 2}
	q1 := Point{(p1.X + p2.X) / 2, (p1.Y + p2.Y) / 2}
	q2 := Point{(p2.X + p3.X) / 2, (p2.Y + p3.Y) / 2}
	r0 := Point{(q0.X + q1.X) / 2, (q0.Y + q1.Y) / 2}
	r1 := Point{(q1.X + q2.X) / 2, (q1.Y + q2.Y) / 2}
	s := Point{(r0.X + r1.X) / 2, (r0.Y + r1.Y) / 2}

	subdivideCubic(buf, p0, q0, r0, s, width, c, depth+1)
	subdivideCubic(buf, s, r1, q2, p3, width, c, depth+1)
}

// isQuadraticFlat checks if a quadratic Bezier curve is flat enough to approximate with a line.
// Uses the distance from the control point to the line segment p0-p2.
func isQuadraticFlat(p0, p1, p2 Point) bool {
	dx := p2.X - p0.X
	dy := p2.Y - p0.Y
	lengthSq := dx*dx + dy*dy

	if lengthSq < 1e-6 {
		return true
	}

	// Distance from p1 to line p0-p2
	t := ((p1.X-p0.X)*dx + (p1.Y-p0.Y)*dy) / lengthSq
	t = clamp(t, 0, 1)
	closestX := p0.X + t*dx
	closestY := p0.Y + t*dy
	distX := p1.X - closestX
	distY := p1.Y - closestY
	distSq := distX*distX + distY*distY

	return distSq <= flatnessTolerance*flatnessTolerance
}

// isCubicFlat checks if a cubic Bezier curve is flat enough to approximate with a line.
// Uses the maximum distance from control points to the line segment p0-p3.
func isCubicFlat(p0, p1, p2, p3 Point) bool {
	dx := p3.X - p0.X
	dy := p3.Y - p0.Y
	lengthSq := dx*dx + dy*dy

	if lengthSq < 1e-6 {
		return true
	}

	// Distance from p1 to line p0-p3
	t1 := ((p1.X-p0.X)*dx + (p1.Y-p0.Y)*dy) / lengthSq
	t1 = clamp(t1, 0, 1)
	closestX1 := p0.X + t1*dx
	closestY1 := p0.Y + t1*dy
	distX1 := p1.X - closestX1
	distY1 := p1.Y - closestY1
	distSq1 := distX1*distX1 + distY1*distY1

	// Distance from p2 to line p0-p3
	t2 := ((p2.X-p0.X)*dx + (p2.Y-p0.Y)*dy) / lengthSq
	t2 = clamp(t2, 0, 1)
	closestX2 := p0.X + t2*dx
	closestY2 := p0.Y + t2*dy
	distX2 := p2.X - closestX2
	distY2 := p2.Y - closestY2
	distSq2 := distX2*distX2 + distY2*distY2

	maxDistSq := distSq1
	if distSq2 > maxDistSq {
		maxDistSq = distSq2
	}

	return maxDistSq <= flatnessTolerance*flatnessTolerance
}

// arcCoverage calculates the anti-aliased coverage for a pixel at (px, py)
// for an elliptical arc wedge centered at (cx, cy).
func arcCoverage(px, py, cx, cy, rx, ry, startAngle, endAngle float64) float64 {
	// Transform to unit circle
	nx := (px - cx) / rx
	ny := (py - cy) / ry
	distSq := nx*nx + ny*ny

	// Check if outside the ellipse
	if distSq > 1.0+2.0/math.Min(rx, ry) {
		return 0
	}

	// Calculate angle of this pixel
	angle := math.Atan2(ny, nx)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	// Check if angle is within the arc range
	inArc := false
	if endAngle > 2*math.Pi {
		inArc = angle >= startAngle || angle <= (endAngle-2*math.Pi)
	} else {
		inArc = angle >= startAngle && angle <= endAngle
	}

	if !inArc {
		return 0
	}

	// Anti-aliasing based on distance to ellipse boundary
	dist := math.Sqrt(distSq)
	if dist <= 1.0-2.0/math.Min(rx, ry) {
		return 1.0
	}

	// Smooth transition at the edge
	edgeDist := (1.0 - dist) * math.Min(rx, ry)
	return clamp(edgeDist+1.0, 0, 1)
}

// normalizeAngle normalizes an angle to the range [0, 2π).
func normalizeAngle(angle float64) float64 {
	angle = math.Mod(angle, 2*math.Pi)
	if angle < 0 {
		angle += 2 * math.Pi
	}
	return angle
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
