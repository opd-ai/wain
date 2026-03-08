package backend

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/displaylist"
)

func TestEncodeScissorState(t *testing.T) {
	rect := ScissorRect{X: 10, Y: 20, Width: 100, Height: 50}
	data := encodeScissorState(rect)

	if len(data) != 2 {
		t.Fatalf("Expected 2 dwords, got %d", len(data))
	}

	// DW0: MinX | MinY << 16
	// Expected: 10 | (20 << 16) = 10 | 1310720 = 1310730
	expectedDW0 := uint32(10 | (20 << 16))
	if data[0] != expectedDW0 {
		t.Errorf("Expected DW0=0x%08x, got 0x%08x", expectedDW0, data[0])
	}

	// DW1: MaxX | MaxY << 16
	// Expected: 110 | (70 << 16) = 110 | 4587520 = 4587630
	expectedDW1 := uint32(110 | (70 << 16))
	if data[1] != expectedDW1 {
		t.Errorf("Expected DW1=0x%08x, got 0x%08x", expectedDW1, data[1])
	}
}

func TestBuildScissorStateBuffer(t *testing.T) {
	rects := []ScissorRect{
		{X: 0, Y: 0, Width: 100, Height: 100},
		{X: 200, Y: 200, Width: 150, Height: 150},
	}

	data := buildScissorStateBuffer(rects)

	// Each rect is 2 dwords (8 bytes), so 2 rects = 16 bytes
	expectedLen := 16
	if len(data) != expectedLen {
		t.Fatalf("Expected %d bytes, got %d", expectedLen, len(data))
	}
}

func TestBuildScissorStateBufferEmpty(t *testing.T) {
	data := buildScissorStateBuffer(nil)

	// Should create a default full-screen scissor
	// 1 rect = 2 dwords = 8 bytes
	if len(data) != 8 {
		t.Fatalf("Expected 8 bytes for default scissor, got %d", len(data))
	}
}

func TestScissorRectFromDamage(t *testing.T) {
	damage := displaylist.Rect{X: 10, Y: 20, Width: 100, Height: 50}
	scissor := ScissorRectFromDamage(damage)

	if scissor.X != 10 || scissor.Y != 20 {
		t.Errorf("Expected origin (10,20), got (%d,%d)", scissor.X, scissor.Y)
	}

	if scissor.Width != 100 || scissor.Height != 50 {
		t.Errorf("Expected size (100,50), got (%d,%d)", scissor.Width, scissor.Height)
	}
}

func TestClampScissorRect(t *testing.T) {
	tests := []struct {
		name       string
		rect       ScissorRect
		maxW, maxH int
		expected   ScissorRect
	}{
		{
			name:     "within bounds",
			rect:     ScissorRect{X: 10, Y: 20, Width: 100, Height: 50},
			maxW:     800,
			maxH:     600,
			expected: ScissorRect{X: 10, Y: 20, Width: 100, Height: 50},
		},
		{
			name:     "exceeds right boundary",
			rect:     ScissorRect{X: 700, Y: 20, Width: 200, Height: 50},
			maxW:     800,
			maxH:     600,
			expected: ScissorRect{X: 700, Y: 20, Width: 100, Height: 50},
		},
		{
			name:     "exceeds bottom boundary",
			rect:     ScissorRect{X: 10, Y: 500, Width: 100, Height: 200},
			maxW:     800,
			maxH:     600,
			expected: ScissorRect{X: 10, Y: 500, Width: 100, Height: 100},
		},
		{
			name:     "negative origin X",
			rect:     ScissorRect{X: -10, Y: 20, Width: 100, Height: 50},
			maxW:     800,
			maxH:     600,
			expected: ScissorRect{X: 0, Y: 20, Width: 90, Height: 50},
		},
		{
			name:     "negative origin Y",
			rect:     ScissorRect{X: 10, Y: -20, Width: 100, Height: 50},
			maxW:     800,
			maxH:     600,
			expected: ScissorRect{X: 10, Y: 0, Width: 100, Height: 30},
		},
		{
			name:     "completely out of bounds",
			rect:     ScissorRect{X: 900, Y: 700, Width: 100, Height: 100},
			maxW:     800,
			maxH:     600,
			expected: ScissorRect{X: 900, Y: 700, Width: 0, Height: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClampScissorRect(tt.rect, tt.maxW, tt.maxH)

			if result.X != tt.expected.X || result.Y != tt.expected.Y {
				t.Errorf("Expected origin (%d,%d), got (%d,%d)",
					tt.expected.X, tt.expected.Y, result.X, result.Y)
			}

			if result.Width != tt.expected.Width || result.Height != tt.expected.Height {
				t.Errorf("Expected size (%d,%d), got (%d,%d)",
					tt.expected.Width, tt.expected.Height, result.Width, result.Height)
			}
		})
	}
}
