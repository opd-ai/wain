package core

import "testing"

func TestFillRect(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		x      int
		y      int
		w      int
		h      int
		color  Color
	}{
		{
			name:   "fill entire buffer",
			width:  10,
			height: 10,
			x:      0,
			y:      0,
			w:      10,
			h:      10,
			color:  Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:   "fill center region",
			width:  20,
			height: 20,
			x:      5,
			y:      5,
			w:      10,
			h:      10,
			color:  Color{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:   "fill with clipping left",
			width:  10,
			height: 10,
			x:      -5,
			y:      0,
			w:      10,
			h:      10,
			color:  Color{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:   "fill with clipping right",
			width:  10,
			height: 10,
			x:      5,
			y:      0,
			w:      10,
			h:      10,
			color:  Color{R: 255, G: 255, B: 0, A: 255},
		},
		{
			name:   "fill completely outside bounds",
			width:  10,
			height: 10,
			x:      20,
			y:      20,
			w:      5,
			h:      5,
			color:  Color{R: 255, G: 0, B: 255, A: 255},
		},
		{
			name:   "fill with zero width",
			width:  10,
			height: 10,
			x:      5,
			y:      5,
			w:      0,
			h:      10,
			color:  Color{R: 128, G: 128, B: 128, A: 255},
		},
		{
			name:   "fill with zero height",
			width:  10,
			height: 10,
			x:      5,
			y:      5,
			w:      10,
			h:      0,
			color:  Color{R: 128, G: 128, B: 128, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := NewBuffer(tt.width, tt.height)
			if err != nil {
				t.Fatalf("NewBuffer() failed: %v", err)
			}

			buf.Clear(Color{R: 0, G: 0, B: 0, A: 0})
			buf.FillRect(tt.x, tt.y, tt.w, tt.h, tt.color)

			x1 := max(0, tt.x)
			y1 := max(0, tt.y)
			x2 := min(tt.width, tt.x+tt.w)
			y2 := min(tt.height, tt.y+tt.h)

			if tt.w <= 0 || tt.h <= 0 || x1 >= x2 || y1 >= y2 {
				for y := 0; y < tt.height; y++ {
					for x := 0; x < tt.width; x++ {
						got := buf.At(x, y)
						expected := Color{R: 0, G: 0, B: 0, A: 0}
						if got != expected {
							t.Errorf("At(%d, %d) = %+v, expected buffer unchanged %+v", x, y, got, expected)
							return
						}
					}
				}
				return
			}

			for y := 0; y < tt.height; y++ {
				for x := 0; x < tt.width; x++ {
					got := buf.At(x, y)
					if x >= x1 && x < x2 && y >= y1 && y < y2 {
						if got != tt.color {
							t.Errorf("At(%d, %d) = %+v, want %+v", x, y, got, tt.color)
						}
					} else {
						expected := Color{R: 0, G: 0, B: 0, A: 0}
						if got != expected {
							t.Errorf("At(%d, %d) = %+v, expected outside rect %+v", x, y, got, expected)
						}
					}
				}
			}
		})
	}
}

func TestFillRoundedRect(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		x      int
		y      int
		w      int
		h      int
		radius float64
		color  Color
	}{
		{
			name:   "rounded rect with radius",
			width:  20,
			height: 20,
			x:      2,
			y:      2,
			w:      16,
			h:      16,
			radius: 4,
			color:  Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:   "rounded rect with zero radius",
			width:  20,
			height: 20,
			x:      5,
			y:      5,
			w:      10,
			h:      10,
			radius: 0,
			color:  Color{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:   "rounded rect with radius exceeding dimensions",
			width:  20,
			height: 20,
			x:      5,
			y:      5,
			w:      10,
			h:      10,
			radius: 20,
			color:  Color{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:   "rounded rect with zero width",
			width:  20,
			height: 20,
			x:      5,
			y:      5,
			w:      0,
			h:      10,
			radius: 4,
			color:  Color{R: 128, G: 128, B: 128, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := NewBuffer(tt.width, tt.height)
			if err != nil {
				t.Fatalf("NewBuffer() failed: %v", err)
			}

			buf.Clear(Color{R: 0, G: 0, B: 0, A: 0})
			buf.FillRoundedRect(tt.x, tt.y, tt.w, tt.h, tt.radius, tt.color)

			if tt.w <= 0 || tt.h <= 0 {
				return
			}

			centerX := tt.x + tt.w/2
			centerY := tt.y + tt.h/2
			if centerX >= 0 && centerX < tt.width && centerY >= 0 && centerY < tt.height {
				got := buf.At(centerX, centerY)
				if got.A == 0 {
					t.Errorf("Center pixel At(%d, %d) is transparent, expected filled", centerX, centerY)
				}
			}
		})
	}
}

func TestRoundedRectCoverage(t *testing.T) {
	tests := []struct {
		name     string
		x        int
		y        int
		width    int
		height   int
		radius   float64
		expected float64
	}{
		{
			name:     "center point full coverage",
			x:        50,
			y:        50,
			width:    100,
			height:   100,
			radius:   10,
			expected: 1.0,
		},
		{
			name:     "outside bounds zero coverage",
			x:        -1,
			y:        50,
			width:    100,
			height:   100,
			radius:   10,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundedRectCoverage(tt.x, tt.y, tt.width, tt.height, tt.radius)
			if got != tt.expected {
				t.Errorf("roundedRectCoverage() = %f, want %f", got, tt.expected)
			}
		})
	}
}

func TestMinMax(t *testing.T) {
	if min(5, 10) != 5 {
		t.Errorf("min(5, 10) = %d, want 5", min(5, 10))
	}
	if min(10, 5) != 5 {
		t.Errorf("min(10, 5) = %d, want 5", min(10, 5))
	}
	if max(5, 10) != 10 {
		t.Errorf("max(5, 10) = %d, want 10", max(5, 10))
	}
	if max(10, 5) != 10 {
		t.Errorf("max(10, 5) = %d, want 10", max(10, 5))
	}
}
