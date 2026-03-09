package effects

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/primitives"
)

func TestBoxShadow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		bufW, bufH int
		x, y       int
		w, h       int
		blurRadius int
		color      primitives.Color
		shouldDraw bool
	}{
		{
			name:       "basic shadow",
			bufW:       100,
			bufH:       100,
			x:          30,
			y:          30,
			w:          20,
			h:          20,
			blurRadius: 5,
			color:      primitives.Color{R: 0, G: 0, B: 0, A: 128},
			shouldDraw: true,
		},
		{
			name:       "shadow with large blur",
			bufW:       100,
			bufH:       100,
			x:          50,
			y:          50,
			w:          20,
			h:          20,
			blurRadius: 15,
			color:      primitives.Color{R: 64, G: 64, B: 64, A: 200},
			shouldDraw: true,
		},
		{
			name:       "shadow clamped to max blur",
			bufW:       200,
			bufH:       200,
			x:          100,
			y:          100,
			w:          40,
			h:          40,
			blurRadius: 100, // Should be clamped to 50
			color:      primitives.Color{R: 0, G: 0, B: 0, A: 255},
			shouldDraw: true,
		},
		{
			name:       "shadow at edge",
			bufW:       50,
			bufH:       50,
			x:          5,
			y:          5,
			w:          10,
			h:          10,
			blurRadius: 3,
			color:      primitives.Color{R: 0, G: 0, B: 0, A: 100},
			shouldDraw: true,
		},
		{
			name:       "nil buffer",
			bufW:       0,
			bufH:       0,
			x:          10,
			y:          10,
			w:          20,
			h:          20,
			blurRadius: 5,
			color:      primitives.Color{R: 0, G: 0, B: 0, A: 128},
			shouldDraw: false,
		},
		{
			name:       "zero width",
			bufW:       100,
			bufH:       100,
			x:          10,
			y:          10,
			w:          0,
			h:          20,
			blurRadius: 5,
			color:      primitives.Color{R: 0, G: 0, B: 0, A: 128},
			shouldDraw: false,
		},
		{
			name:       "zero height",
			bufW:       100,
			bufH:       100,
			x:          10,
			y:          10,
			w:          20,
			h:          0,
			blurRadius: 5,
			color:      primitives.Color{R: 0, G: 0, B: 0, A: 128},
			shouldDraw: false,
		},
		{
			name:       "zero blur radius",
			bufW:       100,
			bufH:       100,
			x:          10,
			y:          10,
			w:          20,
			h:          20,
			blurRadius: 0,
			color:      primitives.Color{R: 0, G: 0, B: 0, A: 128},
			shouldDraw: false,
		},
		{
			name:       "shadow completely outside buffer (right)",
			bufW:       50,
			bufH:       50,
			x:          100,
			y:          20,
			w:          20,
			h:          20,
			blurRadius: 5,
			color:      primitives.Color{R: 0, G: 0, B: 0, A: 128},
			shouldDraw: false,
		},
		{
			name:       "shadow completely outside buffer (bottom)",
			bufW:       50,
			bufH:       50,
			x:          20,
			y:          100,
			w:          20,
			h:          20,
			blurRadius: 5,
			color:      primitives.Color{R: 0, G: 0, B: 0, A: 128},
			shouldDraw: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf *primitives.Buffer
			var err error

			if tt.bufW > 0 && tt.bufH > 0 {
				buf, err = primitives.NewBuffer(tt.bufW, tt.bufH)
				if err != nil {
					t.Fatalf("NewBuffer failed: %v", err)
				}
				// Clear to white background
				buf.Clear(primitives.Color{R: 255, G: 255, B: 255, A: 255})
			}

			// Should not panic
			BoxShadow(buf, tt.x, tt.y, tt.w, tt.h, tt.blurRadius, tt.color)

			if tt.shouldDraw && buf != nil {
				// Verify that some pixels were modified from the white background
				modified := false
				for i := 3; i < len(buf.Pixels); i += 4 {
					// Check if any pixel is not white
					if buf.Pixels[i-3] != 255 || buf.Pixels[i-2] != 255 ||
						buf.Pixels[i-1] != 255 || buf.Pixels[i] != 255 {
						modified = true
						break
					}
				}

				if !modified {
					t.Errorf("BoxShadow should have modified pixels but didn't")
				}
			}
		})
	}
}

func TestLinearGradient(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		bufW, bufH     int
		x, y, w, h     int
		startX, startY int
		startColor     primitives.Color
		endX, endY     int
		endColor       primitives.Color
		shouldFill     bool
		checkColor     bool
		checkX, checkY int
		expectedApprox primitives.Color
		tolerance      uint8
	}{
		{
			name:           "horizontal gradient",
			bufW:           100,
			bufH:           100,
			x:              10,
			y:              10,
			w:              80,
			h:              20,
			startX:         10,
			startY:         20,
			startColor:     primitives.Color{R: 255, G: 0, B: 0, A: 255},
			endX:           90,
			endY:           20,
			endColor:       primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill:     true,
			checkColor:     true,
			checkX:         10,
			checkY:         20,
			expectedApprox: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			tolerance:      5,
		},
		{
			name:           "vertical gradient",
			bufW:           100,
			bufH:           100,
			x:              20,
			y:              10,
			w:              20,
			h:              80,
			startX:         30,
			startY:         10,
			startColor:     primitives.Color{R: 0, G: 255, B: 0, A: 255},
			endX:           30,
			endY:           90,
			endColor:       primitives.Color{R: 255, G: 255, B: 0, A: 255},
			shouldFill:     true,
			checkColor:     true,
			checkX:         30,
			checkY:         10,
			expectedApprox: primitives.Color{R: 0, G: 255, B: 0, A: 255},
			tolerance:      5,
		},
		{
			name:       "diagonal gradient",
			bufW:       100,
			bufH:       100,
			x:          0,
			y:          0,
			w:          100,
			h:          100,
			startX:     0,
			startY:     0,
			startColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			endX:       100,
			endY:       100,
			endColor:   primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill: true,
		},
		{
			name:           "degenerate gradient (same start and end)",
			bufW:           50,
			bufH:           50,
			x:              10,
			y:              10,
			w:              30,
			h:              30,
			startX:         25,
			startY:         25,
			startColor:     primitives.Color{R: 128, G: 128, B: 128, A: 255},
			endX:           25,
			endY:           25,
			endColor:       primitives.Color{R: 0, G: 0, B: 0, A: 255},
			shouldFill:     true,
			checkColor:     true,
			checkX:         25,
			checkY:         25,
			expectedApprox: primitives.Color{R: 128, G: 128, B: 128, A: 255},
			tolerance:      1,
		},
		{
			name:       "nil buffer",
			bufW:       0,
			bufH:       0,
			x:          10,
			y:          10,
			w:          20,
			h:          20,
			startX:     10,
			startY:     10,
			startColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			endX:       30,
			endY:       10,
			endColor:   primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill: false,
		},
		{
			name:       "zero width",
			bufW:       100,
			bufH:       100,
			x:          10,
			y:          10,
			w:          0,
			h:          20,
			startX:     10,
			startY:     10,
			startColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			endX:       30,
			endY:       10,
			endColor:   primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill: false,
		},
		{
			name:       "zero height",
			bufW:       100,
			bufH:       100,
			x:          10,
			y:          10,
			w:          20,
			h:          0,
			startX:     10,
			startY:     10,
			startColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			endX:       30,
			endY:       10,
			endColor:   primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill: false,
		},
		{
			name:       "gradient completely outside buffer",
			bufW:       50,
			bufH:       50,
			x:          100,
			y:          100,
			w:          20,
			h:          20,
			startX:     100,
			startY:     100,
			startColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			endX:       120,
			endY:       100,
			endColor:   primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf *primitives.Buffer
			var err error

			if tt.bufW > 0 && tt.bufH > 0 {
				buf, err = primitives.NewBuffer(tt.bufW, tt.bufH)
				if err != nil {
					t.Fatalf("NewBuffer failed: %v", err)
				}
				buf.Clear(primitives.Color{R: 0, G: 0, B: 0, A: 0})
			}

			// Should not panic
			LinearGradient(buf, tt.x, tt.y, tt.w, tt.h,
				tt.startX, tt.startY, tt.startColor,
				tt.endX, tt.endY, tt.endColor)

			if tt.checkColor && buf != nil {
				idx := tt.checkY*buf.Stride + tt.checkX*4
				if idx+3 < len(buf.Pixels) {
					gotB := buf.Pixels[idx]
					gotG := buf.Pixels[idx+1]
					gotR := buf.Pixels[idx+2]
					gotA := buf.Pixels[idx+3]

					if !colorApproxEqual(gotR, tt.expectedApprox.R, tt.tolerance) ||
						!colorApproxEqual(gotG, tt.expectedApprox.G, tt.tolerance) ||
						!colorApproxEqual(gotB, tt.expectedApprox.B, tt.tolerance) ||
						!colorApproxEqual(gotA, tt.expectedApprox.A, tt.tolerance) {
						t.Errorf("Color at (%d, %d) = RGBA(%d, %d, %d, %d), want approximately RGBA(%d, %d, %d, %d) ±%d",
							tt.checkX, tt.checkY,
							gotR, gotG, gotB, gotA,
							tt.expectedApprox.R, tt.expectedApprox.G, tt.expectedApprox.B, tt.expectedApprox.A,
							tt.tolerance)
					}
				}
			}
		})
	}
}

func TestRadialGradient(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		bufW, bufH     int
		x, y, w, h     int
		centerX        int
		centerY        int
		radius         int
		centerColor    primitives.Color
		edgeColor      primitives.Color
		shouldFill     bool
		checkColor     bool
		checkX         int
		checkY         int
		expectedApprox primitives.Color
		tolerance      uint8
	}{
		{
			name:           "basic radial gradient",
			bufW:           100,
			bufH:           100,
			x:              0,
			y:              0,
			w:              100,
			h:              100,
			centerX:        50,
			centerY:        50,
			radius:         30,
			centerColor:    primitives.Color{R: 255, G: 255, B: 255, A: 255},
			edgeColor:      primitives.Color{R: 0, G: 0, B: 0, A: 255},
			shouldFill:     true,
			checkColor:     true,
			checkX:         50,
			checkY:         50,
			expectedApprox: primitives.Color{R: 255, G: 255, B: 255, A: 255},
			tolerance:      5,
		},
		{
			name:        "small radius",
			bufW:        80,
			bufH:        80,
			x:           20,
			y:           20,
			w:           40,
			h:           40,
			centerX:     40,
			centerY:     40,
			radius:      10,
			centerColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			edgeColor:   primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill:  true,
		},
		{
			name:        "large radius",
			bufW:        200,
			bufH:        200,
			x:           0,
			y:           0,
			w:           200,
			h:           200,
			centerX:     100,
			centerY:     100,
			radius:      100,
			centerColor: primitives.Color{R: 0, G: 255, B: 0, A: 255},
			edgeColor:   primitives.Color{R: 255, G: 255, B: 0, A: 255},
			shouldFill:  true,
		},
		{
			name:        "off-center gradient",
			bufW:        100,
			bufH:        100,
			x:           0,
			y:           0,
			w:           100,
			h:           100,
			centerX:     25,
			centerY:     25,
			radius:      50,
			centerColor: primitives.Color{R: 128, G: 0, B: 128, A: 255},
			edgeColor:   primitives.Color{R: 0, G: 128, B: 128, A: 255},
			shouldFill:  true,
		},
		{
			name:        "nil buffer",
			bufW:        0,
			bufH:        0,
			x:           10,
			y:           10,
			w:           20,
			h:           20,
			centerX:     20,
			centerY:     20,
			radius:      10,
			centerColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			edgeColor:   primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill:  false,
		},
		{
			name:        "zero width",
			bufW:        100,
			bufH:        100,
			x:           10,
			y:           10,
			w:           0,
			h:           20,
			centerX:     20,
			centerY:     20,
			radius:      10,
			centerColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			edgeColor:   primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill:  false,
		},
		{
			name:        "zero height",
			bufW:        100,
			bufH:        100,
			x:           10,
			y:           10,
			w:           20,
			h:           0,
			centerX:     20,
			centerY:     20,
			radius:      10,
			centerColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			edgeColor:   primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill:  false,
		},
		{
			name:        "zero radius",
			bufW:        100,
			bufH:        100,
			x:           10,
			y:           10,
			w:           20,
			h:           20,
			centerX:     20,
			centerY:     20,
			radius:      0,
			centerColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
			edgeColor:   primitives.Color{R: 0, G: 0, B: 255, A: 255},
			shouldFill:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf *primitives.Buffer
			var err error

			if tt.bufW > 0 && tt.bufH > 0 {
				buf, err = primitives.NewBuffer(tt.bufW, tt.bufH)
				if err != nil {
					t.Fatalf("NewBuffer failed: %v", err)
				}
				buf.Clear(primitives.Color{R: 0, G: 0, B: 0, A: 0})
			}

			// Should not panic
			RadialGradient(buf, tt.x, tt.y, tt.w, tt.h,
				tt.centerX, tt.centerY, tt.radius,
				tt.centerColor, tt.edgeColor)

			if tt.checkColor && buf != nil {
				idx := tt.checkY*buf.Stride + tt.checkX*4
				if idx+3 < len(buf.Pixels) {
					gotB := buf.Pixels[idx]
					gotG := buf.Pixels[idx+1]
					gotR := buf.Pixels[idx+2]
					gotA := buf.Pixels[idx+3]

					if !colorApproxEqual(gotR, tt.expectedApprox.R, tt.tolerance) ||
						!colorApproxEqual(gotG, tt.expectedApprox.G, tt.tolerance) ||
						!colorApproxEqual(gotB, tt.expectedApprox.B, tt.tolerance) ||
						!colorApproxEqual(gotA, tt.expectedApprox.A, tt.tolerance) {
						t.Errorf("Color at (%d, %d) = RGBA(%d, %d, %d, %d), want approximately RGBA(%d, %d, %d, %d) ±%d",
							tt.checkX, tt.checkY,
							gotR, gotG, gotB, gotA,
							tt.expectedApprox.R, tt.expectedApprox.G, tt.expectedApprox.B, tt.expectedApprox.A,
							tt.tolerance)
					}
				}
			}
		})
	}
}

func TestScissor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		scissorX, scissorY int
		scissorW, scissorH int
		rectX, rectY       int
		rectW, rectH       int
		expectX1, expectY1 int
		expectX2, expectY2 int
		pointX, pointY     int
		expectContains     bool
	}{
		{
			name:           "full overlap",
			scissorX:       10,
			scissorY:       10,
			scissorW:       50,
			scissorH:       50,
			rectX:          20,
			rectY:          20,
			rectW:          30,
			rectH:          30,
			expectX1:       20,
			expectY1:       20,
			expectX2:       50,
			expectY2:       50,
			pointX:         25,
			pointY:         25,
			expectContains: true,
		},
		{
			name:           "partial overlap (left)",
			scissorX:       20,
			scissorY:       20,
			scissorW:       40,
			scissorH:       40,
			rectX:          10,
			rectY:          30,
			rectW:          30,
			rectH:          20,
			expectX1:       20,
			expectY1:       30,
			expectX2:       40,
			expectY2:       50,
			pointX:         25,
			pointY:         35,
			expectContains: true,
		},
		{
			name:           "partial overlap (right)",
			scissorX:       10,
			scissorY:       10,
			scissorW:       40,
			scissorH:       40,
			rectX:          30,
			rectY:          20,
			rectW:          30,
			rectH:          20,
			expectX1:       30,
			expectY1:       20,
			expectX2:       50,
			expectY2:       40,
			pointX:         35,
			pointY:         25,
			expectContains: true,
		},
		{
			name:           "partial overlap (top)",
			scissorX:       20,
			scissorY:       20,
			scissorW:       40,
			scissorH:       40,
			rectX:          30,
			rectY:          10,
			rectW:          20,
			rectH:          30,
			expectX1:       30,
			expectY1:       20,
			expectX2:       50,
			expectY2:       40,
			pointX:         35,
			pointY:         25,
			expectContains: true,
		},
		{
			name:           "partial overlap (bottom)",
			scissorX:       20,
			scissorY:       20,
			scissorW:       40,
			scissorH:       40,
			rectX:          30,
			rectY:          50,
			rectW:          20,
			rectH:          30,
			expectX1:       30,
			expectY1:       50,
			expectX2:       50,
			expectY2:       60,
			pointX:         35,
			pointY:         55,
			expectContains: true,
		},
		{
			name:           "no overlap (completely outside)",
			scissorX:       10,
			scissorY:       10,
			scissorW:       20,
			scissorH:       20,
			rectX:          50,
			rectY:          50,
			rectW:          20,
			rectH:          20,
			expectX1:       0,
			expectY1:       0,
			expectX2:       0,
			expectY2:       0,
			pointX:         60,
			pointY:         60,
			expectContains: false,
		},
		{
			name:           "rect completely contains scissor",
			scissorX:       30,
			scissorY:       30,
			scissorW:       20,
			scissorH:       20,
			rectX:          10,
			rectY:          10,
			rectW:          100,
			rectH:          100,
			expectX1:       30,
			expectY1:       30,
			expectX2:       50,
			expectY2:       50,
			pointX:         40,
			pointY:         40,
			expectContains: true,
		},
		{
			name:           "scissor completely contains rect",
			scissorX:       10,
			scissorY:       10,
			scissorW:       100,
			scissorH:       100,
			rectX:          30,
			rectY:          30,
			rectW:          20,
			rectH:          20,
			expectX1:       30,
			expectY1:       30,
			expectX2:       50,
			expectY2:       50,
			pointX:         40,
			pointY:         40,
			expectContains: true,
		},
		{
			name:           "point outside scissor",
			scissorX:       20,
			scissorY:       20,
			scissorW:       30,
			scissorH:       30,
			rectX:          25,
			rectY:          25,
			rectW:          10,
			rectH:          10,
			expectX1:       25,
			expectY1:       25,
			expectX2:       35,
			expectY2:       35,
			pointX:         60,
			pointY:         60,
			expectContains: false,
		},
		{
			name:           "point at scissor edge (inside)",
			scissorX:       10,
			scissorY:       10,
			scissorW:       40,
			scissorH:       40,
			rectX:          15,
			rectY:          15,
			rectW:          20,
			rectH:          20,
			expectX1:       15,
			expectY1:       15,
			expectX2:       35,
			expectY2:       35,
			pointX:         10,
			pointY:         10,
			expectContains: true,
		},
		{
			name:           "point at scissor edge (outside)",
			scissorX:       10,
			scissorY:       10,
			scissorW:       40,
			scissorH:       40,
			rectX:          15,
			rectY:          15,
			rectW:          20,
			rectH:          20,
			expectX1:       15,
			expectY1:       15,
			expectX2:       35,
			expectY2:       35,
			pointX:         50,
			pointY:         50,
			expectContains: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScissor(tt.scissorX, tt.scissorY, tt.scissorW, tt.scissorH)

			// Test Clip
			x1, y1, x2, y2 := s.Clip(tt.rectX, tt.rectY, tt.rectW, tt.rectH)
			if x1 != tt.expectX1 || y1 != tt.expectY1 || x2 != tt.expectX2 || y2 != tt.expectY2 {
				t.Errorf("Clip(%d, %d, %d, %d) = (%d, %d, %d, %d), want (%d, %d, %d, %d)",
					tt.rectX, tt.rectY, tt.rectW, tt.rectH,
					x1, y1, x2, y2,
					tt.expectX1, tt.expectY1, tt.expectX2, tt.expectY2)
			}

			// Test Contains
			contains := s.Contains(tt.pointX, tt.pointY)
			if contains != tt.expectContains {
				t.Errorf("Contains(%d, %d) = %v, want %v",
					tt.pointX, tt.pointY, contains, tt.expectContains)
			}
		})
	}
}

func TestInterpolateColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		c1, c2   primitives.Color
		t        float64
		expected primitives.Color
	}{
		{
			name:     "t=0 returns c1",
			c1:       primitives.Color{R: 255, G: 0, B: 0, A: 255},
			c2:       primitives.Color{R: 0, G: 0, B: 255, A: 255},
			t:        0.0,
			expected: primitives.Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:     "t=1 returns c2",
			c1:       primitives.Color{R: 255, G: 0, B: 0, A: 255},
			c2:       primitives.Color{R: 0, G: 0, B: 255, A: 255},
			t:        1.0,
			expected: primitives.Color{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:     "t=0.5 returns midpoint",
			c1:       primitives.Color{R: 0, G: 0, B: 0, A: 0},
			c2:       primitives.Color{R: 200, G: 100, B: 50, A: 200},
			t:        0.5,
			expected: primitives.Color{R: 100, G: 50, B: 25, A: 100},
		},
		{
			name:     "t=0.25 interpolates correctly",
			c1:       primitives.Color{R: 0, G: 0, B: 0, A: 255},
			c2:       primitives.Color{R: 100, G: 200, B: 100, A: 255},
			t:        0.25,
			expected: primitives.Color{R: 25, G: 50, B: 25, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interpolateColor(tt.c1, tt.c2, tt.t)

			// Allow ±1 tolerance due to rounding
			if !colorApproxEqual(result.R, tt.expected.R, 1) ||
				!colorApproxEqual(result.G, tt.expected.G, 1) ||
				!colorApproxEqual(result.B, tt.expected.B, 1) ||
				!colorApproxEqual(result.A, tt.expected.A, 1) {
				t.Errorf("interpolateColor(%+v, %+v, %.2f) = %+v, want %+v",
					tt.c1, tt.c2, tt.t, result, tt.expected)
			}
		})
	}
}

func TestBlurFunctions(t *testing.T) {
	t.Parallel()
	t.Run("blurHorizontal basic", func(t *testing.T) {
		mask := []uint8{
			0, 0, 0, 0, 0,
			0, 0, 255, 0, 0,
			0, 0, 0, 0, 0,
		}
		width, height := 5, 3
		blurHorizontal(mask, width, height, 1)

		// Center row should have blur spread horizontally
		centerRow := mask[width : 2*width]
		if centerRow[1] == 0 || centerRow[3] == 0 {
			t.Errorf("blurHorizontal should spread values horizontally")
		}
	})

	t.Run("blurVertical basic", func(t *testing.T) {
		mask := []uint8{
			0, 0, 0,
			0, 255, 0,
			0, 0, 0,
			0, 0, 0,
			0, 0, 0,
		}
		width, height := 3, 5
		blurVertical(mask, width, height, 1)

		// Center column should have blur spread vertically
		// Row 0, col 1 and row 2, col 1 should have non-zero values after blur
		if mask[0*width+1] == 0 || mask[2*width+1] == 0 {
			t.Errorf("blurVertical should spread values vertically, got row0=%d, row2=%d",
				mask[0*width+1], mask[2*width+1])
		}
	})

	t.Run("blur with zero radius", func(t *testing.T) {
		original := []uint8{0, 128, 255}
		mask := make([]uint8, len(original))
		copy(mask, original)

		blurHorizontal(mask, 3, 1, 0)

		for i := range mask {
			if mask[i] != original[i] {
				t.Errorf("blurHorizontal with radius 0 should not modify mask")
			}
		}

		blurVertical(mask, 1, 3, 0)

		for i := range mask {
			if mask[i] != original[i] {
				t.Errorf("blurVertical with radius 0 should not modify mask")
			}
		}
	})
}

// Helper function to check if two color values are approximately equal
func colorApproxEqual(a, b, tolerance uint8) bool {
	diff := int(a) - int(b)
	if diff < 0 {
		diff = -diff
	}
	return diff <= int(tolerance)
}

// Benchmarks

func BenchmarkBoxShadow(b *testing.B) {
	buf, _ := primitives.NewBuffer(800, 600)
	color := primitives.Color{R: 0, G: 0, B: 0, A: 128}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BoxShadow(buf, 100, 100, 200, 150, 10, color)
	}
}

func BenchmarkLinearGradient(b *testing.B) {
	buf, _ := primitives.NewBuffer(800, 600)
	startColor := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	endColor := primitives.Color{R: 0, G: 0, B: 255, A: 255}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LinearGradient(buf, 0, 0, 800, 600, 0, 300, startColor, 800, 300, endColor)
	}
}

func BenchmarkRadialGradient(b *testing.B) {
	buf, _ := primitives.NewBuffer(800, 600)
	centerColor := primitives.Color{R: 255, G: 255, B: 255, A: 255}
	edgeColor := primitives.Color{R: 0, G: 0, B: 0, A: 255}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RadialGradient(buf, 0, 0, 800, 600, 400, 300, 200, centerColor, edgeColor)
	}
}

func BenchmarkScissorClip(b *testing.B) {
	s := NewScissor(100, 100, 600, 400)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Clip(150, 150, 400, 300)
	}
}
