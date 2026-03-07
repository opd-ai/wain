// Package text implements SDF-based text rendering for 2D software rasterization.
//
// This package provides high-quality scalable text rendering using Signed Distance
// Fields (SDF). The font atlas is pre-baked and embedded in the binary.
//
// # SDF Text Rendering
//
// SDF encoding stores the distance to the nearest glyph edge in each texel.
// This allows smooth scaling and antialiasing at any size without loss of quality.
//
// # Supported Characters
//
// The embedded font atlas supports printable ASCII characters (0x20-0x7E).
// Unsupported characters are rendered as a replacement glyph (□).
//
// # Usage
//
//	atlas := text.NewAtlas()
//	text.DrawText(buf, "Hello, World!", 10, 10, 16, color, atlas)
package text

import (
	_ "embed"
	"encoding/binary"
	"errors"
)

var (
	// ErrInvalidAtlas is returned when the atlas data is malformed.
	ErrInvalidAtlas = errors.New("text: invalid atlas data")

	// ErrGlyphNotFound is returned when a glyph is not in the atlas.
	ErrGlyphNotFound = errors.New("text: glyph not found")
)

// Atlas represents a pre-baked SDF font atlas with glyph metadata.
//
// The atlas contains signed distance field data for each glyph, along with
// metrics for text layout. Glyphs are stored in a packed texture atlas.
type Atlas struct {
	// Width and Height of the atlas texture in pixels
	Width, Height int

	// SDF holds the signed distance field data (8-bit per pixel)
	// Value 128 represents the glyph edge, >128 is inside, <128 is outside
	SDF []uint8

	// Glyphs maps Unicode runes to their glyph metadata
	Glyphs map[rune]*Glyph

	// LineHeight is the recommended vertical spacing between lines
	LineHeight float64

	// Baseline is the distance from the top to the baseline
	Baseline float64
}

// Glyph represents a single character glyph in the atlas.
type Glyph struct {
	// Rune is the Unicode character this glyph represents
	Rune rune

	// X, Y are the atlas texture coordinates (top-left corner) in pixels
	X, Y int

	// Width, Height are the glyph dimensions in the atlas in pixels
	Width, Height int

	// OffsetX, OffsetY are the bearing offsets from the cursor position
	OffsetX, OffsetY float64

	// Advance is the horizontal distance to advance the cursor after rendering
	Advance float64
}

//go:embed data/atlas.bin
var atlasData []byte

// NewAtlas creates a new font atlas from the embedded data.
//
// The atlas is pre-baked at build time and embedded in the binary.
// This function parses the binary atlas format and returns a ready-to-use Atlas.
func NewAtlas() (*Atlas, error) {
	if len(atlasData) < 16 {
		return nil, ErrInvalidAtlas
	}

	// Parse atlas header
	width := int(binary.LittleEndian.Uint32(atlasData[0:4]))
	height := int(binary.LittleEndian.Uint32(atlasData[4:8]))
	glyphCount := int(binary.LittleEndian.Uint32(atlasData[8:12]))
	lineHeight := float64(binary.LittleEndian.Uint32(atlasData[12:16])) / 64.0

	offset := 16
	expectedSDFSize := width * height
	if len(atlasData) < offset+expectedSDFSize {
		return nil, ErrInvalidAtlas
	}

	// Extract SDF data
	sdf := make([]uint8, expectedSDFSize)
	copy(sdf, atlasData[offset:offset+expectedSDFSize])
	offset += expectedSDFSize

	// Parse glyph metadata (each glyph: 36 bytes)
	glyphs := make(map[rune]*Glyph, glyphCount)
	for i := 0; i < glyphCount; i++ {
		if len(atlasData) < offset+36 {
			return nil, ErrInvalidAtlas
		}

		r := rune(binary.LittleEndian.Uint32(atlasData[offset : offset+4]))
		x := int(binary.LittleEndian.Uint32(atlasData[offset+4 : offset+8]))
		y := int(binary.LittleEndian.Uint32(atlasData[offset+8 : offset+12]))
		w := int(binary.LittleEndian.Uint32(atlasData[offset+12 : offset+16]))
		h := int(binary.LittleEndian.Uint32(atlasData[offset+16 : offset+20]))

		// Fixed-point values (divide by 64 to get float)
		ox := float64(int32(binary.LittleEndian.Uint32(atlasData[offset+20:offset+24]))) / 64.0
		oy := float64(int32(binary.LittleEndian.Uint32(atlasData[offset+24:offset+28]))) / 64.0
		adv := float64(binary.LittleEndian.Uint32(atlasData[offset+28:offset+32])) / 64.0

		glyphs[r] = &Glyph{
			Rune:    r,
			X:       x,
			Y:       y,
			Width:   w,
			Height:  h,
			OffsetX: ox,
			OffsetY: oy,
			Advance: adv,
		}
		offset += 36
	}

	return &Atlas{
		Width:      width,
		Height:     height,
		SDF:        sdf,
		Glyphs:     glyphs,
		LineHeight: lineHeight,
		Baseline:   lineHeight * 0.75, // Typical baseline is ~75% of line height
	}, nil
}

// GetGlyph returns the glyph metadata for a rune.
//
// If the rune is not in the atlas, returns a replacement glyph (□) or
// ErrGlyphNotFound if even the replacement is missing.
func (a *Atlas) GetGlyph(r rune) (*Glyph, error) {
	if g, ok := a.Glyphs[r]; ok {
		return g, nil
	}

	// Try replacement character
	if g, ok := a.Glyphs['□']; ok {
		return g, nil
	}

	return nil, ErrGlyphNotFound
}

// SampleSDF samples the SDF value at the given atlas coordinates.
//
// Coordinates are clamped to atlas bounds. Returns a value in [0, 255]
// where 128 represents the glyph edge.
func (a *Atlas) SampleSDF(x, y int) uint8 {
	if x < 0 {
		x = 0
	}
	if x >= a.Width {
		x = a.Width - 1
	}
	if y < 0 {
		y = 0
	}
	if y >= a.Height {
		y = a.Height - 1
	}

	return a.SDF[y*a.Width+x]
}
