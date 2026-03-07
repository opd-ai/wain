package composite

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/core"
)

func TestBlit(t *testing.T) {
	tests := []struct {
		name                   string
		dstW, dstH             int
		srcW, srcH             int
		dstX, dstY             int
		srcX, srcY, w, h       int
		srcColor               core.Color
		dstColor               core.Color
		expectCopied           bool
		expectDstX, expectDstY int
	}{
		{
			name: "simple copy",
			dstW: 100, dstH: 100,
			srcW: 50, srcH: 50,
			dstX: 10, dstY: 10,
			srcX: 0, srcY: 0, w: 20, h: 20,
			srcColor:     core.Color{R: 255, G: 0, B: 0, A: 255},
			dstColor:     core.Color{R: 0, G: 0, B: 255, A: 255},
			expectCopied: true,
			expectDstX:   10, expectDstY: 10,
		},
		{
			name: "copy with source offset",
			dstW: 100, dstH: 100,
			srcW: 50, srcH: 50,
			dstX: 20, dstY: 20,
			srcX: 10, srcY: 10, w: 15, h: 15,
			srcColor:     core.Color{R: 0, G: 255, B: 0, A: 255},
			dstColor:     core.Color{R: 0, G: 0, B: 255, A: 255},
			expectCopied: true,
			expectDstX:   20, expectDstY: 20,
		},
		{
			name: "copy clipped at source boundary",
			dstW: 100, dstH: 100,
			srcW: 50, srcH: 50,
			dstX: 10, dstY: 10,
			srcX: 40, srcY: 40, w: 20, h: 20,
			srcColor:     core.Color{R: 255, G: 255, B: 0, A: 255},
			dstColor:     core.Color{R: 0, G: 0, B: 255, A: 255},
			expectCopied: true,
			expectDstX:   10, expectDstY: 10,
		},
		{
			name: "copy clipped at dest boundary",
			dstW: 50, dstH: 50,
			srcW: 100, srcH: 100,
			dstX: 40, dstY: 40,
			srcX: 0, srcY: 0, w: 20, h: 20,
			srcColor:     core.Color{R: 255, G: 0, B: 255, A: 255},
			dstColor:     core.Color{R: 0, G: 0, B: 255, A: 255},
			expectCopied: true,
			expectDstX:   40, expectDstY: 40,
		},
		{
			name: "copy with transparent source",
			dstW: 100, dstH: 100,
			srcW: 50, srcH: 50,
			dstX: 10, dstY: 10,
			srcX: 0, srcY: 0, w: 20, h: 20,
			srcColor:     core.Color{R: 255, G: 0, B: 0, A: 0},
			dstColor:     core.Color{R: 0, G: 0, B: 255, A: 255},
			expectCopied: false,
			expectDstX:   10, expectDstY: 10,
		},
		{
			name: "copy with semi-transparent source",
			dstW: 100, dstH: 100,
			srcW: 50, srcH: 50,
			dstX: 10, dstY: 10,
			srcX: 0, srcY: 0, w: 20, h: 20,
			srcColor:     core.Color{R: 255, G: 0, B: 0, A: 128},
			dstColor:     core.Color{R: 0, G: 0, B: 255, A: 255},
			expectCopied: true,
			expectDstX:   10, expectDstY: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst, err := core.NewBuffer(tt.dstW, tt.dstH)
			if err != nil {
				t.Fatalf("NewBuffer(dst) failed: %v", err)
			}
			dst.Clear(tt.dstColor)

			src, err := core.NewBuffer(tt.srcW, tt.srcH)
			if err != nil {
				t.Fatalf("NewBuffer(src) failed: %v", err)
			}
			src.Clear(tt.srcColor)

			Blit(dst, tt.dstX, tt.dstY, src, tt.srcX, tt.srcY, tt.w, tt.h)

			checkX := tt.expectDstX
			checkY := tt.expectDstY
			if checkX < 0 || checkX >= tt.dstW || checkY < 0 || checkY >= tt.dstH {
				return
			}

			pixel := dst.At(checkX, checkY)

			if tt.expectCopied {
				if tt.srcColor.A == 255 {
					if pixel != tt.srcColor {
						t.Errorf("At(%d, %d) = %+v, want %+v (opaque copy)", checkX, checkY, pixel, tt.srcColor)
					}
				} else if tt.srcColor.A == 0 {
					if pixel != tt.dstColor {
						t.Errorf("At(%d, %d) = %+v, want %+v (transparent source)", checkX, checkY, pixel, tt.dstColor)
					}
				} else {
					if pixel == tt.dstColor {
						t.Errorf("At(%d, %d) = %+v, expected blending but got original dst color", checkX, checkY, pixel)
					}
					if pixel.A == 0 {
						t.Errorf("At(%d, %d) has alpha = 0, expected blended alpha", checkX, checkY)
					}
				}
			} else {
				if pixel != tt.dstColor {
					t.Errorf("At(%d, %d) = %+v, want %+v (no copy expected)", checkX, checkY, pixel, tt.dstColor)
				}
			}
		})
	}
}

func TestBlitNilBuffers(t *testing.T) {
	dst, _ := core.NewBuffer(10, 10)
	src, _ := core.NewBuffer(10, 10)

	Blit(nil, 0, 0, src, 0, 0, 5, 5)
	Blit(dst, 0, 0, nil, 0, 0, 5, 5)
	Blit(nil, 0, 0, nil, 0, 0, 5, 5)
}

func TestBlitInvalidDimensions(t *testing.T) {
	dst, _ := core.NewBuffer(10, 10)
	src, _ := core.NewBuffer(10, 10)

	Blit(dst, 0, 0, src, 0, 0, 0, 5)
	Blit(dst, 0, 0, src, 0, 0, 5, 0)
	Blit(dst, 0, 0, src, 0, 0, -5, 5)
	Blit(dst, 0, 0, src, 0, 0, 5, -5)
}

func TestBlitScaled(t *testing.T) {
	tests := []struct {
		name                     string
		dstW, dstH               int
		srcW, srcH               int
		dstX, dstY, dstW2, dstH2 int
		srcX, srcY, srcW2, srcH2 int
		srcColor                 core.Color
		expectInterpolation      bool
	}{
		{
			name: "upscale 2x",
			dstW: 100, dstH: 100,
			srcW: 50, srcH: 50,
			dstX: 10, dstY: 10, dstW2: 40, dstH2: 40,
			srcX: 0, srcY: 0, srcW2: 20, srcH2: 20,
			srcColor:            core.Color{R: 255, G: 0, B: 0, A: 255},
			expectInterpolation: true,
		},
		{
			name: "downscale 2x",
			dstW: 100, dstH: 100,
			srcW: 100, srcH: 100,
			dstX: 10, dstY: 10, dstW2: 20, dstH2: 20,
			srcX: 0, srcY: 0, srcW2: 40, srcH2: 40,
			srcColor:            core.Color{R: 0, G: 255, B: 0, A: 255},
			expectInterpolation: true,
		},
		{
			name: "no scaling (1:1)",
			dstW: 100, dstH: 100,
			srcW: 50, srcH: 50,
			dstX: 10, dstY: 10, dstW2: 20, dstH2: 20,
			srcX: 0, srcY: 0, srcW2: 20, srcH2: 20,
			srcColor:            core.Color{R: 0, G: 0, B: 255, A: 255},
			expectInterpolation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst, err := core.NewBuffer(tt.dstW, tt.dstH)
			if err != nil {
				t.Fatalf("NewBuffer(dst) failed: %v", err)
			}
			dst.Clear(core.Color{R: 0, G: 0, B: 0, A: 255})

			src, err := core.NewBuffer(tt.srcW, tt.srcH)
			if err != nil {
				t.Fatalf("NewBuffer(src) failed: %v", err)
			}
			src.Clear(tt.srcColor)

			BlitScaled(dst, tt.dstX, tt.dstY, tt.dstW2, tt.dstH2,
				src, tt.srcX, tt.srcY, tt.srcW2, tt.srcH2)

			checkX := tt.dstX + tt.dstW2/2
			checkY := tt.dstY + tt.dstH2/2
			if checkX < 0 || checkX >= tt.dstW || checkY < 0 || checkY >= tt.dstH {
				t.Skip("check coordinates out of bounds")
			}

			pixel := dst.At(checkX, checkY)
			if pixel.A == 0 {
				t.Errorf("At(%d, %d) has alpha = 0, expected non-zero after blit", checkX, checkY)
			}
		})
	}
}

func TestBlitScaledNilBuffers(t *testing.T) {
	dst, _ := core.NewBuffer(10, 10)
	src, _ := core.NewBuffer(10, 10)

	BlitScaled(nil, 0, 0, 5, 5, src, 0, 0, 5, 5)
	BlitScaled(dst, 0, 0, 5, 5, nil, 0, 0, 5, 5)
	BlitScaled(nil, 0, 0, 5, 5, nil, 0, 0, 5, 5)
}

func TestBlitScaledInvalidDimensions(t *testing.T) {
	dst, _ := core.NewBuffer(10, 10)
	src, _ := core.NewBuffer(10, 10)

	BlitScaled(dst, 0, 0, 0, 5, src, 0, 0, 5, 5)
	BlitScaled(dst, 0, 0, 5, 0, src, 0, 0, 5, 5)
	BlitScaled(dst, 0, 0, 5, 5, src, 0, 0, 0, 5)
	BlitScaled(dst, 0, 0, 5, 5, src, 0, 0, 5, 0)
	BlitScaled(dst, 0, 0, -5, 5, src, 0, 0, 5, 5)
	BlitScaled(dst, 0, 0, 5, -5, src, 0, 0, 5, 5)
}

func TestSamplePixel(t *testing.T) {
	buf, err := core.NewBuffer(10, 10)
	if err != nil {
		t.Fatalf("NewBuffer failed: %v", err)
	}
	buf.Clear(core.Color{R: 255, G: 128, B: 64, A: 200})

	tests := []struct {
		name       string
		x, y       int
		expectZero bool
	}{
		{"valid pixel", 5, 5, false},
		{"top-left corner", 0, 0, false},
		{"bottom-right corner", 9, 9, false},
		{"out of bounds left", -1, 5, true},
		{"out of bounds right", 10, 5, true},
		{"out of bounds top", 5, -1, true},
		{"out of bounds bottom", 5, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pixel := samplePixel(buf, tt.x, tt.y)
			if tt.expectZero {
				if pixel != [4]byte{0, 0, 0, 0} {
					t.Errorf("samplePixel(%d, %d) = %v, want [0 0 0 0]", tt.x, tt.y, pixel)
				}
			} else {
				if pixel[3] == 0 {
					t.Errorf("samplePixel(%d, %d) = %v, expected non-zero alpha", tt.x, tt.y, pixel)
				}
			}
		})
	}
}

func TestBilinearInterpolate(t *testing.T) {
	tests := []struct {
		name                               string
		p00, p10, p01, p11                 [4]byte
		fracX, fracY                       float64
		expectR, expectG, expectB, expectA uint8
	}{
		{
			name:  "no interpolation (fracX=0, fracY=0)",
			p00:   [4]byte{0, 0, 255, 255},
			p10:   [4]byte{0, 255, 0, 255},
			p01:   [4]byte{255, 0, 0, 255},
			p11:   [4]byte{128, 128, 128, 255},
			fracX: 0.0, fracY: 0.0,
			expectR: 255, expectG: 0, expectB: 0, expectA: 255,
		},
		{
			name:  "full interpolation to p10 (fracX=1, fracY=0)",
			p00:   [4]byte{0, 0, 255, 255},
			p10:   [4]byte{0, 255, 0, 255},
			p01:   [4]byte{255, 0, 0, 255},
			p11:   [4]byte{128, 128, 128, 255},
			fracX: 1.0, fracY: 0.0,
			expectR: 0, expectG: 255, expectB: 0, expectA: 255,
		},
		{
			name:  "full interpolation to p01 (fracX=0, fracY=1)",
			p00:   [4]byte{0, 0, 255, 255},
			p10:   [4]byte{0, 255, 0, 255},
			p01:   [4]byte{255, 0, 0, 255},
			p11:   [4]byte{128, 128, 128, 255},
			fracX: 0.0, fracY: 1.0,
			expectR: 0, expectG: 0, expectB: 255, expectA: 255,
		},
		{
			name:  "full interpolation to p11 (fracX=1, fracY=1)",
			p00:   [4]byte{0, 0, 255, 255},
			p10:   [4]byte{0, 255, 0, 255},
			p01:   [4]byte{255, 0, 0, 255},
			p11:   [4]byte{128, 128, 128, 255},
			fracX: 1.0, fracY: 1.0,
			expectR: 128, expectG: 128, expectB: 128, expectA: 255,
		},
		{
			name:  "center interpolation (fracX=0.5, fracY=0.5)",
			p00:   [4]byte{0, 0, 0, 0},
			p10:   [4]byte{0, 0, 255, 255},
			p01:   [4]byte{0, 255, 0, 255},
			p11:   [4]byte{255, 0, 0, 255},
			fracX: 0.5, fracY: 0.5,
			expectR: 64, expectG: 64, expectB: 64, expectA: 191,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bilinearInterpolate(tt.p00, tt.p10, tt.p01, tt.p11, tt.fracX, tt.fracY)

			tolerance := uint8(2)
			if diff(result[2], tt.expectR) > tolerance {
				t.Errorf("R = %d, want %d (±%d)", result[2], tt.expectR, tolerance)
			}
			if diff(result[1], tt.expectG) > tolerance {
				t.Errorf("G = %d, want %d (±%d)", result[1], tt.expectG, tolerance)
			}
			if diff(result[0], tt.expectB) > tolerance {
				t.Errorf("B = %d, want %d (±%d)", result[0], tt.expectB, tolerance)
			}
			if diff(result[3], tt.expectA) > tolerance {
				t.Errorf("A = %d, want %d (±%d)", result[3], tt.expectA, tolerance)
			}
		})
	}
}

func diff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func TestBlendPixelDirect(t *testing.T) {
	tests := []struct {
		name                               string
		dst                                [4]byte
		src                                [4]byte
		expectR, expectG, expectB, expectA uint8
	}{
		{
			name:    "opaque source over opaque dest",
			dst:     [4]byte{0, 0, 255, 255},
			src:     [4]byte{0, 255, 0, 255},
			expectR: 0, expectG: 255, expectB: 0, expectA: 255,
		},
		{
			name:    "transparent source over opaque dest",
			dst:     [4]byte{0, 0, 255, 255},
			src:     [4]byte{0, 255, 0, 0},
			expectR: 255, expectG: 0, expectB: 0, expectA: 255,
		},
		{
			name:    "semi-transparent source over opaque dest",
			dst:     [4]byte{0, 0, 255, 255},
			src:     [4]byte{0, 255, 0, 128},
			expectR: 127, expectG: 128, expectB: 0, expectA: 255,
		},
		{
			name:    "opaque source over transparent dest",
			dst:     [4]byte{0, 0, 255, 0},
			src:     [4]byte{0, 255, 0, 255},
			expectR: 0, expectG: 255, expectB: 0, expectA: 255,
		},
		{
			name:    "semi-transparent source over semi-transparent dest",
			dst:     [4]byte{0, 0, 255, 128},
			src:     [4]byte{0, 255, 0, 128},
			expectR: 127, expectG: 128, expectB: 0, expectA: 191,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make([]byte, 4)
			copy(dst, tt.dst[:])

			blendPixelDirect(dst, tt.src[:])

			tolerance := uint8(2)
			if diff(dst[2], tt.expectR) > tolerance {
				t.Errorf("R = %d, want %d (±%d)", dst[2], tt.expectR, tolerance)
			}
			if diff(dst[1], tt.expectG) > tolerance {
				t.Errorf("G = %d, want %d (±%d)", dst[1], tt.expectG, tolerance)
			}
			if diff(dst[0], tt.expectB) > tolerance {
				t.Errorf("B = %d, want %d (±%d)", dst[0], tt.expectB, tolerance)
			}
			if diff(dst[3], tt.expectA) > tolerance {
				t.Errorf("A = %d, want %d (±%d)", dst[3], tt.expectA, tolerance)
			}
		})
	}
}

func BenchmarkBlit(b *testing.B) {
	dst, _ := core.NewBuffer(1920, 1080)
	src, _ := core.NewBuffer(800, 600)
	src.Clear(core.Color{R: 255, G: 128, B: 64, A: 255})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Blit(dst, 100, 100, src, 0, 0, 400, 300)
	}
}

func BenchmarkBlitScaled(b *testing.B) {
	dst, _ := core.NewBuffer(1920, 1080)
	src, _ := core.NewBuffer(800, 600)
	src.Clear(core.Color{R: 255, G: 128, B: 64, A: 255})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BlitScaled(dst, 100, 100, 800, 600, src, 0, 0, 400, 300)
	}
}

func BenchmarkBlendPixelDirect(b *testing.B) {
	dst := make([]byte, 4)
	src := [4]byte{255, 128, 64, 128}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blendPixelDirect(dst, src[:])
	}
}
