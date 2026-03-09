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

const (
	// AtlasTextureSize is the size of the embedded SDF font atlas texture in pixels.
	// The atlas is square (256x256) and contains ASCII printable characters (0x20-0x7E).
	AtlasTextureSize = 256
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

	header, offset, err := parseAtlasHeader()
	if err != nil {
		return nil, err
	}

	sdf, offset, err := extractSDFData(offset, header.width*header.height)
	if err != nil {
		return nil, err
	}

	glyphs, err := parseGlyphMetadata(offset, header.glyphCount)
	if err != nil {
		return nil, err
	}

	return &Atlas{
		Width:      header.width,
		Height:     header.height,
		SDF:        sdf,
		Glyphs:     glyphs,
		LineHeight: header.lineHeight,
		Baseline:   header.lineHeight * 0.75,
	}, nil
}

// atlasHeader holds the binary format header for a font atlas file.
type atlasHeader struct {
	width, height int
	glyphCount    int
	lineHeight    float64
}

// parseAtlasHeader parses the atlas binary header.
func parseAtlasHeader() (*atlasHeader, int, error) {
	width := int(binary.LittleEndian.Uint32(atlasData[0:4]))
	height := int(binary.LittleEndian.Uint32(atlasData[4:8]))
	glyphCount := int(binary.LittleEndian.Uint32(atlasData[8:12]))
	lineHeight := float64(binary.LittleEndian.Uint32(atlasData[12:16])) / 64.0
	return &atlasHeader{width, height, glyphCount, lineHeight}, 16, nil
}

// extractSDFData extracts SDF bitmap from atlas data.
func extractSDFData(offset, size int) ([]uint8, int, error) {
	if len(atlasData) < offset+size {
		return nil, 0, ErrInvalidAtlas
	}
	sdf := make([]uint8, size)
	copy(sdf, atlasData[offset:offset+size])
	return sdf, offset + size, nil
}

// parseGlyphMetadata parses glyph metadata (each glyph: 36 bytes).
func parseGlyphMetadata(offset, count int) (map[rune]*Glyph, error) {
	glyphs := make(map[rune]*Glyph, count)
	for i := 0; i < count; i++ {
		if len(atlasData) < offset+36 {
			return nil, ErrInvalidAtlas
		}
		g := parseGlyph(atlasData[offset : offset+36])
		glyphs[g.Rune] = g
		offset += 36
	}
	return glyphs, nil
}

// parseGlyph parses a single glyph from 36 bytes of data.
func parseGlyph(data []byte) *Glyph {
	return &Glyph{
		Rune:    rune(binary.LittleEndian.Uint32(data[0:4])),
		X:       int(binary.LittleEndian.Uint32(data[4:8])),
		Y:       int(binary.LittleEndian.Uint32(data[8:12])),
		Width:   int(binary.LittleEndian.Uint32(data[12:16])),
		Height:  int(binary.LittleEndian.Uint32(data[16:20])),
		OffsetX: float64(int32(binary.LittleEndian.Uint32(data[20:24]))) / 64.0,
		OffsetY: float64(int32(binary.LittleEndian.Uint32(data[24:28]))) / 64.0,
		Advance: float64(binary.LittleEndian.Uint32(data[28:32])) / 64.0,
	}
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
