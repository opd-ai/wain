package main

import (
"encoding/binary"
"fmt"
"os"
)

func main() {
const atlasWidth = 256
const atlasHeight = 256
const glyphSize = 16
const firstChar = 0x20
const lastChar = 0x7E
glyphCount := lastChar - firstChar + 1 + 1

sdf := make([]uint8, atlasWidth*atlasHeight)
for i := range sdf {
sdf[i] = 0
}

type GlyphMeta struct {
Rune    rune
X, Y    int
W, H    int
OffsetX float64
OffsetY float64
Advance float64
}

glyphs := make([]GlyphMeta, 0, glyphCount)
col, row := 0, 0

for r := firstChar; r <= lastChar; r++ {
x, y := col*glyphSize, row*glyphSize
drawSimpleGlyph(sdf, atlasWidth, atlasHeight, x, y, glyphSize, r)
glyphs = append(glyphs, GlyphMeta{
Rune: rune(r), X: x, Y: y, W: glyphSize, H: glyphSize,
OffsetX: 0, OffsetY: -float64(glyphSize) * 0.75, Advance: float64(glyphSize) * 0.6,
})
col++
if col >= 16 {
col, row = 0, row+1
}
}

x, y := col*glyphSize, row*glyphSize
drawReplacementGlyph(sdf, atlasWidth, atlasHeight, x, y, glyphSize)
glyphs = append(glyphs, GlyphMeta{
Rune: '□', X: x, Y: y, W: glyphSize, H: glyphSize,
OffsetX: 0, OffsetY: -float64(glyphSize) * 0.75, Advance: float64(glyphSize) * 0.6,
})

f, err := os.Create("internal/raster/text/data/atlas.bin")
if err != nil {
panic(err)
}
defer f.Close()

lineHeight := uint32(glyphSize * 64)
binary.Write(f, binary.LittleEndian, uint32(atlasWidth))
binary.Write(f, binary.LittleEndian, uint32(atlasHeight))
binary.Write(f, binary.LittleEndian, uint32(len(glyphs)))
binary.Write(f, binary.LittleEndian, lineHeight)
f.Write(sdf)

for _, g := range glyphs {
binary.Write(f, binary.LittleEndian, uint32(g.Rune))
binary.Write(f, binary.LittleEndian, uint32(g.X))
binary.Write(f, binary.LittleEndian, uint32(g.Y))
binary.Write(f, binary.LittleEndian, uint32(g.W))
binary.Write(f, binary.LittleEndian, uint32(g.H))
binary.Write(f, binary.LittleEndian, int32(g.OffsetX*64))
binary.Write(f, binary.LittleEndian, int32(g.OffsetY*64))
binary.Write(f, binary.LittleEndian, uint32(g.Advance*64))
binary.Write(f, binary.LittleEndian, uint32(0))
}

fmt.Printf("Generated atlas: %dx%d, %d glyphs\n", atlasWidth, atlasHeight, len(glyphs))
}

func drawSimpleGlyph(sdf []uint8, w, h, x, y, size int, r int) {
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
setPixel(sdf, w, h, x+px, y+py, sdfVal)
}
}
}

func drawReplacementGlyph(sdf []uint8, w, h, x, y, size int) {
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
setPixel(sdf, w, h, x+px, y+py, sdfVal)
}
}
}

func setPixel(sdf []uint8, w, h, x, y int, val uint8) {
if x >= 0 && x < w && y >= 0 && y < h {
sdf[y*w+x] = val
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
