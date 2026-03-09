package main

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/opd-ai/wain/internal/demo"
)

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
	writeAtlasFile("internal/raster/text/data/atlas.bin", cfg, sdf, glyphs)
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

func writeAtlasFile(path string, cfg atlasConfig, sdf []uint8, glyphs []glyphMeta) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	writeAtlasHeader(f, cfg, len(glyphs))
	if _, err := f.Write(sdf); err != nil {
		panic(err)
	}
	writeGlyphMetadata(f, glyphs)
}

func writeAtlasHeader(f *os.File, cfg atlasConfig, glyphCount int) {
	if err := binary.Write(f, binary.LittleEndian, uint32(cfg.width)); err != nil {
		panic(err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(cfg.height)); err != nil {
		panic(err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(glyphCount)); err != nil {
		panic(err)
	}
	if err := binary.Write(f, binary.LittleEndian, cfg.lineHeight); err != nil {
		panic(err)
	}
}

func writeGlyphMetadata(f *os.File, glyphs []glyphMeta) {
	for _, g := range glyphs {
		if err := binary.Write(f, binary.LittleEndian, uint32(g.Rune)); err != nil {
			panic(err)
		}
		if err := binary.Write(f, binary.LittleEndian, uint32(g.X)); err != nil {
			panic(err)
		}
		if err := binary.Write(f, binary.LittleEndian, uint32(g.Y)); err != nil {
			panic(err)
		}
		if err := binary.Write(f, binary.LittleEndian, uint32(g.W)); err != nil {
			panic(err)
		}
		if err := binary.Write(f, binary.LittleEndian, uint32(g.H)); err != nil {
			panic(err)
		}
		if err := binary.Write(f, binary.LittleEndian, int32(g.OffsetX*64)); err != nil {
			panic(err)
		}
		if err := binary.Write(f, binary.LittleEndian, int32(g.OffsetY*64)); err != nil {
			panic(err)
		}
		if err := binary.Write(f, binary.LittleEndian, uint32(g.Advance*64)); err != nil {
			panic(err)
		}
		if err := binary.Write(f, binary.LittleEndian, uint32(0)); err != nil {
			panic(err)
		}
	}
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
				!(px >= 4 && px < size-4 && py >= 4 && py < size-4)
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
