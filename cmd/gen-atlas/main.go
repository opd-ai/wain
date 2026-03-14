package main

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/opd-ai/wain/internal/demo"
)

// writeLE writes v to f in little-endian byte order.
func writeLE(f *os.File, v any) error {
	return binary.Write(f, binary.LittleEndian, v)
}

type glyphMeta struct {
	Rune    rune
	X, Y    int
	W, H    int
	OffsetX float64
	OffsetY float64
	Advance float64
}

type atlasConfig struct {
	width      int
	height     int
	glyphSize  int
	firstChar  int
	lastChar   int
	lineHeight uint32
}

func main() {
	demo.CheckHelpFlag("gen-atlas", "Generate SDF font atlas for text rendering", []string{
		demo.FormatExample("gen-atlas", "Generate atlas.bin font atlas file"),
		demo.FormatExample("gen-atlas --help", "Show this help message"),
	})

	cfg := atlasConfig{
		width:      256,
		height:     256,
		glyphSize:  16,
		firstChar:  0x20,
		lastChar:   0x7E,
		lineHeight: uint32(16 * 64),
	}

	sdf, glyphs := generateAtlas(cfg)
	if err := writeAtlasFile("internal/raster/text/data/atlas.bin", cfg, sdf, glyphs); err != nil {
		fmt.Fprintf(os.Stderr, "gen-atlas: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated atlas: %dx%d, %d glyphs\n", cfg.width, cfg.height, len(glyphs))
}

func generateAtlas(cfg atlasConfig) ([]uint8, []glyphMeta) {
	sdf := make([]uint8, cfg.width*cfg.height)
	glyphCount := cfg.lastChar - cfg.firstChar + 1 + 1
	glyphs := make([]glyphMeta, 0, glyphCount)

	col, row := 0, 0
	for r := cfg.firstChar; r <= cfg.lastChar; r++ {
		x, y := col*cfg.glyphSize, row*cfg.glyphSize
		drawSimpleGlyph(sdf, cfg.width, cfg.height, x, y, cfg.glyphSize, r)
		glyphs = append(glyphs, glyphMeta{
			Rune: rune(r), X: x, Y: y, W: cfg.glyphSize, H: cfg.glyphSize,
			OffsetX: 0, OffsetY: -float64(cfg.glyphSize) * 0.75, Advance: float64(cfg.glyphSize) * 0.6,
		})
		col++
		if col >= 16 {
			col, row = 0, row+1
		}
	}

	x, y := col*cfg.glyphSize, row*cfg.glyphSize
	drawReplacementGlyph(sdf, cfg.width, cfg.height, x, y, cfg.glyphSize)
	glyphs = append(glyphs, glyphMeta{
		Rune: '□', X: x, Y: y, W: cfg.glyphSize, H: cfg.glyphSize,
		OffsetX: 0, OffsetY: -float64(cfg.glyphSize) * 0.75, Advance: float64(cfg.glyphSize) * 0.6,
	})

	return sdf, glyphs
}

func writeAtlasFile(path string, cfg atlasConfig, sdf []uint8, glyphs []glyphMeta) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	if err := writeAtlasHeader(f, cfg, len(glyphs)); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := f.Write(sdf); err != nil {
		return fmt.Errorf("write pixel data: %w", err)
	}
	if err := writeGlyphMetadata(f, glyphs); err != nil {
		return fmt.Errorf("write glyph metadata: %w", err)
	}
	return nil
}

func writeAtlasHeader(f *os.File, cfg atlasConfig, glyphCount int) error {
	if err := writeLE(f, uint32(cfg.width)); err != nil {
		return err
	}
	if err := writeLE(f, uint32(cfg.height)); err != nil {
		return err
	}
	if err := writeLE(f, uint32(glyphCount)); err != nil {
		return err
	}
	return writeLE(f, cfg.lineHeight)
}

// writeGlyphMetadata writes packed glyph layout records to the atlas file.
func writeGlyphMetadata(f *os.File, glyphs []glyphMeta) error {
	for _, g := range glyphs {
		if err := writeSingleGlyph(f, g); err != nil {
			return err
		}
	}
	return nil
}

// writeSingleGlyph encodes one glyph's metrics to the atlas file in
// little-endian fixed-point format (sub-pixel values are scaled by 64).
func writeSingleGlyph(f *os.File, g glyphMeta) error {
	for _, v := range []any{
		uint32(g.Rune),
		uint32(g.X), uint32(g.Y),
		uint32(g.W), uint32(g.H),
		int32(g.OffsetX * 64),
		int32(g.OffsetY * 64),
		uint32(g.Advance * 64),
		uint32(0),
	} {
		if err := writeLE(f, v); err != nil {
			return err
		}
	}
	return nil
}

func drawSimpleGlyph(sdf []uint8, width, height, xPos, yPos, size, r int) {
	pattern := getCharPattern(r)
	cellW, cellH := float64(size)/6.0, float64(size)/8.0

	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			gridX, gridY := int(float64(px)/cellW), int(float64(py)/cellH)
			if gridX >= 5 || gridY >= 7 {
				continue
			}
			bit := gridY*5 + gridX
			inside := bit < len(pattern)*8 && (pattern[bit/8]&(1<<(bit%8))) != 0
			var sdfVal uint8
			if inside {
				sdfVal = 200
			} else {
				sdfVal = 56
			}
			setPixel(sdf, width, height, xPos+px, yPos+py, sdfVal)
		}
	}
}

func drawReplacementGlyph(sdf []uint8, width, height, xPos, yPos, size int) {
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			inside := (px >= 2 && px < size-2 && py >= 2 && py < size-2) &&
				(px < 4 || px >= size-4 || py < 4 || py >= size-4)
			var sdfVal uint8
			if inside {
				sdfVal = 200
			} else {
				sdfVal = 56
			}
			setPixel(sdf, width, height, xPos+px, yPos+py, sdfVal)
		}
	}
}

func setPixel(sdf []uint8, width, height, xPos, yPos int, val uint8) {
	if xPos >= 0 && xPos < width && yPos >= 0 && yPos < height {
		sdf[yPos*width+xPos] = val
	}
}

func getCharPattern(r int) []byte {
	patterns := map[int][]byte{
		' ': {}, '!': {0x04, 0x04, 0x04, 0x04, 0x00, 0x04},
		'A': {0x04, 0x0A, 0x0A, 0x0E, 0x0A, 0x0A},
		'H': {0x0A, 0x0A, 0x0E, 0x0A, 0x0A},
		'W': {0x0A, 0x0A, 0x0A, 0x0E, 0x0A},
		'e': {0x00, 0x06, 0x0A, 0x0E, 0x06},
		'l': {0x04, 0x04, 0x04, 0x04, 0x06},
		'o': {0x00, 0x04, 0x0A, 0x0A, 0x04},
		'r': {0x00, 0x06, 0x08, 0x08, 0x08},
	}
	if p, ok := patterns[r]; ok {
		return p
	}
	return []byte{0x04, 0x04, 0x04, 0x04, 0x04}
}
