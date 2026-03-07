package core

import (
	"math"
	"testing"
)

func TestDrawLine(t *testing.T) {
	tests := []struct {
		name  string
		x0    int
		y0    int
		x1    int
		y1    int
		width float64
		color Color
	}{
		{
			name:  "horizontal line",
			x0:    5,
			y0:    10,
			x1:    15,
			y1:    10,
			width: 2,
			color: Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:  "vertical line",
			x0:    10,
			y0:    5,
			x1:    10,
			y1:    15,
			width: 2,
			color: Color{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:  "diagonal line",
			x0:    5,
			y0:    5,
			x1:    15,
			y1:    15,
			width: 2,
			color: Color{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:  "single point line",
			x0:    10,
			y0:    10,
			x1:    10,
			y1:    10,
			width: 1,
			color: Color{R: 255, G: 255, B: 0, A: 255},
		},
		{
			name:  "line with zero width",
			x0:    5,
			y0:    5,
			x1:    15,
			y1:    15,
			width: 0,
			color: Color{R: 255, G: 0, B: 255, A: 255},
		},
		{
			name:  "line with fractional width",
			x0:    5,
			y0:    10,
			x1:    15,
			y1:    10,
			width: 1.5,
			color: Color{R: 128, G: 128, B: 128, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := NewBuffer(20, 20)
			if err != nil {
				t.Fatalf("NewBuffer() failed: %v", err)
			}

			buf.Clear(Color{R: 0, G: 0, B: 0, A: 0})
			buf.DrawLine(tt.x0, tt.y0, tt.x1, tt.y1, tt.width, tt.color)

			if tt.width <= 0 {
				for y := 0; y < 20; y++ {
					for x := 0; x < 20; x++ {
						got := buf.At(x, y)
						expected := Color{R: 0, G: 0, B: 0, A: 0}
						if got != expected {
							t.Errorf("At(%d, %d) = %+v, expected buffer unchanged", x, y, got)
							return
						}
					}
				}
				return
			}

			if tt.x0 == tt.x1 && tt.y0 == tt.y1 {
				got := buf.At(tt.x0, tt.y0)
				if got.A == 0 {
					t.Errorf("Single point line at (%d, %d) not drawn", tt.x0, tt.y0)
				}
				return
			}

			dx := float64(tt.x1 - tt.x0)
			dy := float64(tt.y1 - tt.y0)
			length := math.Sqrt(dx*dx + dy*dy)

			if length >= 1 {
				midX := (tt.x0 + tt.x1) / 2
				midY := (tt.y0 + tt.y1) / 2
				got := buf.At(midX, midY)
				if got.A == 0 {
					t.Errorf("Line midpoint at (%d, %d) has zero alpha", midX, midY)
				}
			}
		})
	}
}

func TestLineCoverage(t *testing.T) {
	tests := []struct {
		name       string
		px         float64
		py         float64
		x0         float64
		y0         float64
		x1         float64
		y1         float64
		width      float64
		expectZero bool
	}{
		{
			name:       "point on horizontal line center",
			px:         10,
			py:         10,
			x0:         5,
			y0:         10,
			x1:         15,
			y1:         10,
			width:      2,
			expectZero: false,
		},
		{
			name:       "point far from line",
			px:         10,
			py:         20,
			x0:         5,
			y0:         10,
			x1:         15,
			y1:         10,
			width:      2,
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dx := tt.x1 - tt.x0
			dy := tt.y1 - tt.y0
			length := math.Sqrt(dx*dx + dy*dy)

			dirX := dx / length
			dirY := dy / length

			perpX := -dirY
			perpY := dirX

			halfWidth := tt.width / 2

			coverage := lineCoverage(tt.px, tt.py, tt.x0, tt.y0, dirX, dirY, perpX, perpY, length, halfWidth)

			if tt.expectZero {
				if coverage != 0 {
					t.Errorf("lineCoverage() = %f, expected 0 for point far from line", coverage)
				}
			} else {
				if coverage <= 0 {
					t.Errorf("lineCoverage() = %f, expected > 0 for point on line", coverage)
				}
			}
		})
	}
}

func TestMin2fMax2f(t *testing.T) {
	if min2f(5.5, 10.5) != 5.5 {
		t.Errorf("min2f(5.5, 10.5) = %f, want 5.5", min2f(5.5, 10.5))
	}
	if min2f(10.5, 5.5) != 5.5 {
		t.Errorf("min2f(10.5, 5.5) = %f, want 5.5", min2f(10.5, 5.5))
	}
	if max2f(5.5, 10.5) != 10.5 {
		t.Errorf("max2f(5.5, 10.5) = %f, want 10.5", max2f(5.5, 10.5))
	}
	if max2f(10.5, 5.5) != 10.5 {
		t.Errorf("max2f(10.5, 5.5) = %f, want 10.5", max2f(10.5, 5.5))
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name     string
		v        float64
		lo       float64
		hi       float64
		expected float64
	}{
		{
			name:     "value in range",
			v:        5,
			lo:       0,
			hi:       10,
			expected: 5,
		},
		{
			name:     "value below range",
			v:        -5,
			lo:       0,
			hi:       10,
			expected: 0,
		},
		{
			name:     "value above range",
			v:        15,
			lo:       0,
			hi:       10,
			expected: 10,
		},
		{
			name:     "value at lower bound",
			v:        0,
			lo:       0,
			hi:       10,
			expected: 0,
		},
		{
			name:     "value at upper bound",
			v:        10,
			lo:       0,
			hi:       10,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clamp(tt.v, tt.lo, tt.hi)
			if got != tt.expected {
				t.Errorf("clamp(%f, %f, %f) = %f, want %f", tt.v, tt.lo, tt.hi, got, tt.expected)
			}
		})
	}
}
