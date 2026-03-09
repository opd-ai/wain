package primitives

import (
	"testing"
)

func TestNewBuffer(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		wantErr   bool
		wantPixel int
	}{
		{
			name:      "valid small buffer",
			width:     10,
			height:    10,
			wantErr:   false,
			wantPixel: 10 * 10 * 4,
		},
		{
			name:      "valid large buffer",
			width:     1920,
			height:    1080,
			wantErr:   false,
			wantPixel: 1920 * 1080 * 4,
		},
		{
			name:    "zero width",
			width:   0,
			height:  10,
			wantErr: true,
		},
		{
			name:    "zero height",
			width:   10,
			height:  0,
			wantErr: true,
		},
		{
			name:    "negative width",
			width:   -10,
			height:  10,
			wantErr: true,
		},
		{
			name:    "negative height",
			width:   10,
			height:  -10,
			wantErr: true,
		},
		{
			name:    "exceeds maximum width",
			width:   20000,
			height:  10,
			wantErr: true,
		},
		{
			name:    "exceeds maximum height",
			width:   10,
			height:  20000,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := NewBuffer(tt.width, tt.height)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewBuffer() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("NewBuffer() unexpected error: %v", err)
				return
			}
			if buf == nil {
				t.Error("NewBuffer() returned nil buffer")
				return
			}
			if buf.Width != tt.width {
				t.Errorf("Width = %d, want %d", buf.Width, tt.width)
			}
			if buf.Height != tt.height {
				t.Errorf("Height = %d, want %d", buf.Height, tt.height)
			}
			if buf.Stride != tt.width*4 {
				t.Errorf("Stride = %d, want %d", buf.Stride, tt.width*4)
			}
			if len(buf.Pixels) != tt.wantPixel {
				t.Errorf("Pixels length = %d, want %d", len(buf.Pixels), tt.wantPixel)
			}
		})
	}
}

func TestBufferClear(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		color  Color
	}{
		{
			name:   "clear with opaque red",
			width:  10,
			height: 10,
			color:  Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:   "clear with transparent color",
			width:  10,
			height: 10,
			color:  Color{R: 100, G: 150, B: 200, A: 128},
		},
		{
			name:   "clear with fully transparent",
			width:  10,
			height: 10,
			color:  Color{R: 0, G: 0, B: 0, A: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := NewBuffer(tt.width, tt.height)
			if err != nil {
				t.Fatalf("NewBuffer() failed: %v", err)
			}

			buf.Clear(tt.color)

			for y := 0; y < tt.height; y++ {
				for x := 0; x < tt.width; x++ {
					got := buf.At(x, y)
					if got != tt.color {
						t.Errorf("At(%d, %d) = %+v, want %+v", x, y, got, tt.color)
					}
				}
			}
		})
	}
}

func TestBufferAtSet(t *testing.T) {
	buf, err := NewBuffer(10, 10)
	if err != nil {
		t.Fatalf("NewBuffer() failed: %v", err)
	}

	buf.Clear(Color{R: 0, G: 0, B: 0, A: 0})

	tests := []struct {
		name  string
		x     int
		y     int
		color Color
		valid bool
	}{
		{
			name:  "set center pixel",
			x:     5,
			y:     5,
			color: Color{R: 255, G: 0, B: 0, A: 255},
			valid: true,
		},
		{
			name:  "set top-left corner",
			x:     0,
			y:     0,
			color: Color{R: 0, G: 255, B: 0, A: 255},
			valid: true,
		},
		{
			name:  "set bottom-right corner",
			x:     9,
			y:     9,
			color: Color{R: 0, G: 0, B: 255, A: 255},
			valid: true,
		},
		{
			name:  "set out of bounds negative",
			x:     -1,
			y:     5,
			color: Color{R: 100, G: 100, B: 100, A: 255},
			valid: false,
		},
		{
			name:  "set out of bounds positive",
			x:     10,
			y:     5,
			color: Color{R: 100, G: 100, B: 100, A: 255},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Set(tt.x, tt.y, tt.color)

			if tt.valid {
				got := buf.At(tt.x, tt.y)
				if got != tt.color {
					t.Errorf("At(%d, %d) = %+v, want %+v", tt.x, tt.y, got, tt.color)
				}
			}
		})
	}
}

func TestBufferBlending(t *testing.T) {
	buf, err := NewBuffer(10, 10)
	if err != nil {
		t.Fatalf("NewBuffer() failed: %v", err)
	}

	buf.Clear(Color{R: 255, G: 0, B: 0, A: 255})

	buf.Set(5, 5, Color{R: 0, G: 255, B: 0, A: 128})

	got := buf.At(5, 5)

	if got.A == 0 {
		t.Error("Blending resulted in fully transparent pixel")
	}
	if got.R == 255 && got.G == 0 {
		t.Error("Blending did not modify pixel")
	}
}

func TestColorRGBA(t *testing.T) {
	tests := []struct {
		name  string
		color Color
	}{
		{
			name:  "opaque red",
			color: Color{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:  "opaque green",
			color: Color{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:  "transparent blue",
			color: Color{R: 0, G: 0, B: 255, A: 128},
		},
		{
			name:  "fully transparent",
			color: Color{R: 100, G: 100, B: 100, A: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, g, b, a := tt.color.RGBA()

			if a == 0 && tt.color.A != 0 {
				t.Errorf("RGBA() alpha = 0, expected non-zero for alpha %d", tt.color.A)
			}
			if a == 0xffff && tt.color.A != 255 {
				t.Errorf("RGBA() alpha = 0xffff, expected less for alpha %d", tt.color.A)
			}

			if tt.color.A == 255 {
				if r>>8 != uint32(tt.color.R) {
					t.Errorf("RGBA() red = %d, want %d", r>>8, tt.color.R)
				}
				if g>>8 != uint32(tt.color.G) {
					t.Errorf("RGBA() green = %d, want %d", g>>8, tt.color.G)
				}
				if b>>8 != uint32(tt.color.B) {
					t.Errorf("RGBA() blue = %d, want %d", b>>8, tt.color.B)
				}
			}
		})
	}
}
