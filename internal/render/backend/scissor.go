package backend

import (
	"encoding/binary"

	"github.com/opd-ai/wain/internal/raster/displaylist"
)

// ScissorRect represents a scissor rectangle for GPU clipping.
type ScissorRect struct {
	X, Y          int
	Width, Height int
}

// encodeScissorState generates GPU commands for scissor rectangle configuration.
// Intel Gen9+ uses 3DSTATE_SCISSOR_STATE_POINTERS to set scissor rectangles.
func encodeScissorState(rect ScissorRect) []uint32 {
	// Scissor rect structure for Intel Gen9+:
	// DW0: MinX | MinY << 16
	// DW1: MaxX | MaxY << 16
	// Coordinates are in pixels, with (0,0) at top-left

	minX := uint32(rect.X)
	minY := uint32(rect.Y)
	maxX := uint32(rect.X + rect.Width)
	maxY := uint32(rect.Y + rect.Height)

	// Scissor state (2 dwords per rectangle)
	scissorData := []uint32{
		(minY << 16) | minX, // DW0: MinX, MinY
		(maxY << 16) | maxX, // DW1: MaxX, MaxY
	}

	return scissorData
}

// buildScissorStateBuffer creates a state buffer containing scissor rectangles.
// Returns the buffer data as bytes.
func buildScissorStateBuffer(rects []ScissorRect) []byte {
	if len(rects) == 0 {
		// Default to full screen scissor (disabled clipping)
		rects = []ScissorRect{{X: 0, Y: 0, Width: 8192, Height: 8192}}
	}

	var commands []uint32
	for _, rect := range rects {
		commands = append(commands, encodeScissorState(rect)...)
	}

	// Convert to bytes
	data := make([]byte, len(commands)*4)
	for i, cmd := range commands {
		binary.LittleEndian.PutUint32(data[i*4:], cmd)
	}

	return data
}

// ScissorRectFromDamage converts a damage rect to a scissor rect.
func ScissorRectFromDamage(damage displaylist.Rect) ScissorRect {
	return ScissorRect{
		X:      damage.X,
		Y:      damage.Y,
		Width:  damage.Width,
		Height: damage.Height,
	}
}

// ClampScissorRect clamps a scissor rect to fit within target dimensions.
func ClampScissorRect(rect ScissorRect, maxWidth, maxHeight int) ScissorRect {
	rect.X, rect.Width = clampAxis(rect.X, rect.Width, maxWidth)
	rect.Y, rect.Height = clampAxis(rect.Y, rect.Height, maxHeight)
	return rect
}

// clampAxis adjusts an axis origin and size so the range [origin, origin+size) fits within [0, max).
func clampAxis(origin, size, max int) (int, int) {
	if origin < 0 {
		size += origin
		origin = 0
	}
	if origin+size > max {
		size = max - origin
	}
	if size < 0 {
		size = 0
	}
	return origin, size
}
