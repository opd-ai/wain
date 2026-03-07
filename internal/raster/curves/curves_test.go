package curves

import (
	"math"
	"testing"

	"github.com/opd-ai/wain/internal/raster/core"
)

func TestDrawQuadraticBezier(t *testing.T) {
	tests := []struct {
		name  string
		p0    Point
		p1    Point
		p2    Point
		width float64
		color core.Color
	}{
		{
			name:  "simple curve",
			p0:    Point{10, 10},
			p1:    Point{50, 5},
			p2:    Point{90, 10},
			width: 2.0,
			color: core.Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:  "vertical curve",
			p0:    Point{50, 10},
			p1:    Point{30, 50},
			p2:    Point{50, 90},
			width: 3.0,
			color: core.Color{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:  "degenerate curve (straight line)",
			p0:    Point{10, 10},
			p1:    Point{50, 50},
			p2:    Point{90, 90},
			width: 1.0,
			color: core.Color{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:  "degenerate curve (single point)",
			p0:    Point{50, 50},
			p1:    Point{50, 50},
			p2:    Point{50, 50},
			width: 1.0,
			color: core.Color{R: 255, G: 255, B: 0, A: 255},
		},
		{
			name:  "zero width (should not draw)",
			p0:    Point{10, 10},
			p1:    Point{50, 5},
			p2:    Point{90, 10},
			width: 0,
			color: core.Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:  "negative width (should not draw)",
			p0:    Point{10, 10},
			p1:    Point{50, 5},
			p2:    Point{90, 10},
			width: -1.0,
			color: core.Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:  "tight curve",
			p0:    Point{50, 50},
			p1:    Point{60, 10},
			p2:    Point{70, 50},
			width: 2.0,
			color: core.Color{R: 255, G: 128, B: 0, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := core.NewBuffer(100, 100)
			if err != nil {
				t.Fatalf("NewBuffer failed: %v", err)
			}

			DrawQuadraticBezier(buf, tt.p0, tt.p1, tt.p2, tt.width, tt.color)

			if tt.width <= 0 {
				if !isBufferClear(buf) {
					t.Error("expected buffer to remain clear for non-positive width")
				}
			}
		})
	}
}

func TestDrawCubicBezier(t *testing.T) {
	tests := []struct {
		name  string
		p0    Point
		p1    Point
		p2    Point
		p3    Point
		width float64
		color core.Color
	}{
		{
			name:  "simple S-curve",
			p0:    Point{10, 50},
			p1:    Point{30, 10},
			p2:    Point{70, 90},
			p3:    Point{90, 50},
			width: 2.0,
			color: core.Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:  "degenerate curve (straight line)",
			p0:    Point{10, 10},
			p1:    Point{40, 40},
			p2:    Point{60, 60},
			p3:    Point{90, 90},
			width: 1.0,
			color: core.Color{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:  "degenerate curve (single point)",
			p0:    Point{50, 50},
			p1:    Point{50, 50},
			p2:    Point{50, 50},
			p3:    Point{50, 50},
			width: 1.0,
			color: core.Color{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:  "zero width (should not draw)",
			p0:    Point{10, 50},
			p1:    Point{30, 10},
			p2:    Point{70, 90},
			p3:    Point{90, 50},
			width: 0,
			color: core.Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:  "complex curve",
			p0:    Point{10, 10},
			p1:    Point{90, 30},
			p2:    Point{10, 70},
			p3:    Point{90, 90},
			width: 3.0,
			color: core.Color{R: 128, G: 0, B: 128, A: 255},
		},
		{
			name:  "tight loop",
			p0:    Point{50, 50},
			p1:    Point{80, 20},
			p2:    Point{20, 20},
			p3:    Point{50, 50},
			width: 2.0,
			color: core.Color{R: 0, G: 128, B: 255, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := core.NewBuffer(100, 100)
			if err != nil {
				t.Fatalf("NewBuffer failed: %v", err)
			}

			DrawCubicBezier(buf, tt.p0, tt.p1, tt.p2, tt.p3, tt.width, tt.color)

			if tt.width <= 0 {
				if !isBufferClear(buf) {
					t.Error("expected buffer to remain clear for non-positive width")
				}
			}
		})
	}
}

func TestDrawArc(t *testing.T) {
	tests := []struct {
		name       string
		cx, cy     float64
		rx, ry     float64
		startAngle float64
		endAngle   float64
		width      float64
		color      core.Color
	}{
		{
			name:       "quarter circle (top-right)",
			cx:         50,
			cy:         50,
			rx:         30,
			ry:         30,
			startAngle: 0,
			endAngle:   math.Pi / 2,
			width:      2.0,
			color:      core.Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:       "semicircle",
			cx:         50,
			cy:         50,
			rx:         25,
			ry:         25,
			startAngle: 0,
			endAngle:   math.Pi,
			width:      3.0,
			color:      core.Color{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:       "full circle",
			cx:         50,
			cy:         50,
			rx:         20,
			ry:         20,
			startAngle: 0,
			endAngle:   2 * math.Pi,
			width:      1.0,
			color:      core.Color{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:       "elliptical arc",
			cx:         50,
			cy:         50,
			rx:         40,
			ry:         20,
			startAngle: math.Pi / 4,
			endAngle:   3 * math.Pi / 4,
			width:      2.0,
			color:      core.Color{R: 255, G: 255, B: 0, A: 255},
		},
		{
			name:       "wraparound arc",
			cx:         50,
			cy:         50,
			rx:         30,
			ry:         30,
			startAngle: 3 * math.Pi / 2,
			endAngle:   math.Pi / 2,
			width:      2.0,
			color:      core.Color{R: 255, G: 0, B: 255, A: 255},
		},
		{
			name:       "zero width (should not draw)",
			cx:         50,
			cy:         50,
			rx:         30,
			ry:         30,
			startAngle: 0,
			endAngle:   math.Pi,
			width:      0,
			color:      core.Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:       "zero radius (should not draw)",
			cx:         50,
			cy:         50,
			rx:         0,
			ry:         30,
			startAngle: 0,
			endAngle:   math.Pi,
			width:      2.0,
			color:      core.Color{R: 255, G: 0, B: 0, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := core.NewBuffer(100, 100)
			if err != nil {
				t.Fatalf("NewBuffer failed: %v", err)
			}

			DrawArc(buf, tt.cx, tt.cy, tt.rx, tt.ry, tt.startAngle, tt.endAngle, tt.width, tt.color)

			if tt.width <= 0 || tt.rx <= 0 || tt.ry <= 0 {
				if !isBufferClear(buf) {
					t.Error("expected buffer to remain clear for invalid parameters")
				}
			}
		})
	}
}

func TestFillArc(t *testing.T) {
	tests := []struct {
		name       string
		cx, cy     float64
		rx, ry     float64
		startAngle float64
		endAngle   float64
		color      core.Color
	}{
		{
			name:       "quarter pie slice",
			cx:         50,
			cy:         50,
			rx:         30,
			ry:         30,
			startAngle: 0,
			endAngle:   math.Pi / 2,
			color:      core.Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:       "half ellipse",
			cx:         50,
			cy:         50,
			rx:         40,
			ry:         20,
			startAngle: 0,
			endAngle:   math.Pi,
			color:      core.Color{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:       "full circle",
			cx:         50,
			cy:         50,
			rx:         25,
			ry:         25,
			startAngle: 0,
			endAngle:   2 * math.Pi,
			color:      core.Color{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:       "small wedge",
			cx:         50,
			cy:         50,
			rx:         30,
			ry:         30,
			startAngle: 0,
			endAngle:   math.Pi / 6,
			color:      core.Color{R: 255, G: 255, B: 0, A: 255},
		},
		{
			name:       "wraparound wedge",
			cx:         50,
			cy:         50,
			rx:         30,
			ry:         30,
			startAngle: 5 * math.Pi / 4,
			endAngle:   math.Pi / 4,
			color:      core.Color{R: 255, G: 0, B: 255, A: 255},
		},
		{
			name:       "zero radius (should not draw)",
			cx:         50,
			cy:         50,
			rx:         0,
			ry:         0,
			startAngle: 0,
			endAngle:   math.Pi,
			color:      core.Color{R: 255, G: 0, B: 0, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := core.NewBuffer(100, 100)
			if err != nil {
				t.Fatalf("NewBuffer failed: %v", err)
			}

			FillArc(buf, tt.cx, tt.cy, tt.rx, tt.ry, tt.startAngle, tt.endAngle, tt.color)

			if tt.rx <= 0 || tt.ry <= 0 {
				if !isBufferClear(buf) {
					t.Error("expected buffer to remain clear for zero radius")
				}
			}
		})
	}
}

func TestIsQuadraticFlat(t *testing.T) {
	tests := []struct {
		name     string
		p0       Point
		p1       Point
		p2       Point
		expected bool
	}{
		{
			name:     "perfectly flat (collinear)",
			p0:       Point{0, 0},
			p1:       Point{50, 50},
			p2:       Point{100, 100},
			expected: true,
		},
		{
			name:     "very curved",
			p0:       Point{0, 0},
			p1:       Point{50, 100},
			p2:       Point{100, 0},
			expected: false,
		},
		{
			name:     "degenerate (single point)",
			p0:       Point{50, 50},
			p1:       Point{50, 50},
			p2:       Point{50, 50},
			expected: true,
		},
		{
			name:     "slightly curved (within tolerance)",
			p0:       Point{0, 0},
			p1:       Point{50, 0.3},
			p2:       Point{100, 0},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isQuadraticFlat(tt.p0, tt.p1, tt.p2)
			if result != tt.expected {
				t.Errorf("isQuadraticFlat() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsCubicFlat(t *testing.T) {
	tests := []struct {
		name     string
		p0       Point
		p1       Point
		p2       Point
		p3       Point
		expected bool
	}{
		{
			name:     "perfectly flat (collinear)",
			p0:       Point{0, 0},
			p1:       Point{33, 33},
			p2:       Point{67, 67},
			p3:       Point{100, 100},
			expected: true,
		},
		{
			name:     "very curved",
			p0:       Point{0, 0},
			p1:       Point{0, 100},
			p2:       Point{100, 100},
			p3:       Point{100, 0},
			expected: false,
		},
		{
			name:     "degenerate (single point)",
			p0:       Point{50, 50},
			p1:       Point{50, 50},
			p2:       Point{50, 50},
			p3:       Point{50, 50},
			expected: true,
		},
		{
			name:     "slightly curved (within tolerance)",
			p0:       Point{0, 0},
			p1:       Point{33, 0.2},
			p2:       Point{67, 0.2},
			p3:       Point{100, 0},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCubicFlat(tt.p0, tt.p1, tt.p2, tt.p3)
			if result != tt.expected {
				t.Errorf("isCubicFlat() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNormalizeAngle(t *testing.T) {
	tests := []struct {
		name     string
		angle    float64
		expected float64
	}{
		{
			name:     "zero",
			angle:    0,
			expected: 0,
		},
		{
			name:     "positive in range",
			angle:    math.Pi / 2,
			expected: math.Pi / 2,
		},
		{
			name:     "negative angle",
			angle:    -math.Pi / 2,
			expected: 3 * math.Pi / 2,
		},
		{
			name:     "angle > 2π",
			angle:    3 * math.Pi,
			expected: math.Pi,
		},
		{
			name:     "large negative angle",
			angle:    -3 * math.Pi,
			expected: math.Pi,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeAngle(tt.angle)
			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("normalizeAngle(%f) = %f, want %f", tt.angle, result, tt.expected)
			}
		})
	}
}

func TestArcCoverage(t *testing.T) {
	tests := []struct {
		name       string
		px, py     float64
		cx, cy     float64
		rx, ry     float64
		startAngle float64
		endAngle   float64
		minExpect  float64
	}{
		{
			name:       "point inside arc",
			px:         60,
			py:         50,
			cx:         50,
			cy:         50,
			rx:         30,
			ry:         30,
			startAngle: 0,
			endAngle:   math.Pi / 2,
			minExpect:  0.5,
		},
		{
			name:       "point outside arc angle",
			px:         50,
			py:         80,
			cx:         50,
			cy:         50,
			rx:         30,
			ry:         30,
			startAngle: 0,
			endAngle:   math.Pi / 2,
			minExpect:  0.0,
		},
		{
			name:       "point outside ellipse",
			px:         100,
			py:         50,
			cx:         50,
			cy:         50,
			rx:         30,
			ry:         30,
			startAngle: 0,
			endAngle:   2 * math.Pi,
			minExpect:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := arcCoverage(tt.px, tt.py, tt.cx, tt.cy, tt.rx, tt.ry, tt.startAngle, tt.endAngle)
			if result < tt.minExpect {
				t.Errorf("arcCoverage() = %f, want >= %f", result, tt.minExpect)
			}
		})
	}
}

// isBufferClear checks if all pixels in the buffer are transparent black.
func isBufferClear(buf *core.Buffer) bool {
	for i := 0; i < len(buf.Pixels); i++ {
		if buf.Pixels[i] != 0 {
			return false
		}
	}
	return true
}

// Benchmark tests
func BenchmarkDrawQuadraticBezier(b *testing.B) {
	buf, _ := core.NewBuffer(500, 500)
	p0 := Point{50, 50}
	p1 := Point{250, 10}
	p2 := Point{450, 50}
	color := core.Color{R: 255, G: 0, B: 0, A: 255}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DrawQuadraticBezier(buf, p0, p1, p2, 2.0, color)
	}
}

func BenchmarkDrawCubicBezier(b *testing.B) {
	buf, _ := core.NewBuffer(500, 500)
	p0 := Point{50, 250}
	p1 := Point{150, 50}
	p2 := Point{350, 450}
	p3 := Point{450, 250}
	color := core.Color{R: 255, G: 0, B: 0, A: 255}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DrawCubicBezier(buf, p0, p1, p2, p3, 2.0, color)
	}
}

func BenchmarkDrawArc(b *testing.B) {
	buf, _ := core.NewBuffer(500, 500)
	color := core.Color{R: 255, G: 0, B: 0, A: 255}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DrawArc(buf, 250, 250, 100, 100, 0, math.Pi/2, 2.0, color)
	}
}

func BenchmarkFillArc(b *testing.B) {
	buf, _ := core.NewBuffer(500, 500)
	color := core.Color{R: 255, G: 0, B: 0, A: 255}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FillArc(buf, 250, 250, 100, 100, 0, math.Pi/2, color)
	}
}
