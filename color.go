package wain

import "github.com/opd-ai/wain/internal/raster/core"

// Color represents an RGBA color with 8-bit channels.
//
// Colors are specified in sRGB color space with separate red, green, blue,
// and alpha components. Alpha of 255 is fully opaque, 0 is fully transparent.
//
// Example:
//
//	red := wain.RGB(255, 0, 0)           // Opaque red
//	transparentBlue := wain.RGBA(0, 0, 255, 128)  // 50% transparent blue
type Color struct {
	R, G, B, A uint8
}

// RGB creates an opaque color from red, green, and blue components.
func RGB(r, g, b uint8) Color {
	return Color{R: r, G: g, B: b, A: 255}
}

// RGBA creates a color from red, green, blue, and alpha components.
func RGBA(r, g, b, a uint8) Color {
	return Color{R: r, G: g, B: b, A: a}
}

// toInternal converts a public Color to the internal core.Color representation.
func (c Color) toInternal() core.Color {
	return core.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

// toU32 converts the color to a 32-bit ARGB value for display list commands.
func (c Color) toU32() uint32 {
	return uint32(c.A)<<24 | uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
}

// WithAlpha returns a new color with the specified alpha value.
func (c Color) WithAlpha(a uint8) Color {
	return Color{R: c.R, G: c.G, B: c.B, A: a}
}

// Common color constants for convenience.
var (
	// Transparent is fully transparent black.
	Transparent = RGBA(0, 0, 0, 0)

	// Black is opaque black.
	Black = RGB(0, 0, 0)

	// White is opaque white.
	White = RGB(255, 255, 255)

	// Red is opaque red.
	Red = RGB(255, 0, 0)

	// Green is opaque green.
	Green = RGB(0, 255, 0)

	// Blue is opaque blue.
	Blue = RGB(0, 0, 255)

	// Gray is opaque medium gray.
	Gray = RGB(128, 128, 128)

	// LightGray is opaque light gray.
	LightGray = RGB(192, 192, 192)

	// DarkGray is opaque dark gray.
	DarkGray = RGB(64, 64, 64)
)
