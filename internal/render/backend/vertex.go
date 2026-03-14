package backend

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
)

// ErrVertexBufferFull is returned when the vertex buffer is too small.
var ErrVertexBufferFull = errors.New("backend: vertex buffer full")

// Vertex represents a single vertex in GPU format.
// Layout: position (x, y), UV (u, v), color (r, g, b, a)
type Vertex struct {
	X, Y       float32
	U, V       float32
	R, G, B, A uint8
}

// packVertices packs all vertices from batches into a flat byte array.
func (b *GPUBackend) packVertices(batches []Batch) ([]byte, error) {
	// Estimate total vertex count (conservative upper bound: 6 vertices per command)
	estimatedVertices := 0
	for _, batch := range batches {
		estimatedVertices += len(batch.Commands) * 6
	}

	// Each vertex is 20 bytes (2*float32 + 2*float32 + 4*uint8)
	const vertexSize = 20
	estimatedSize := estimatedVertices * vertexSize

	// Calculate buffer size from width (each pixel is 4 bytes)
	bufferSize := int(b.vertexBuffer.Width * b.vertexBuffer.Stride)

	if estimatedSize > bufferSize {
		return nil, fmt.Errorf("%w: need %d bytes, have %d",
			ErrVertexBufferFull, estimatedSize, bufferSize)
	}

	// Allocate vertex data buffer
	data := make([]byte, 0, estimatedSize)

	// Pack vertices for each batch
	for _, batch := range batches {
		batchData, err := packBatchVertices(batch, b.width, b.height, b.fontAtlas)
		if err != nil {
			return nil, err
		}
		data = append(data, batchData...)
	}

	return data, nil
}

// packBatchVertices packs vertices for a single batch.
func packBatchVertices(batch Batch, fbWidth, fbHeight int, atlas *text.Atlas) ([]byte, error) {
	data := make([]byte, 0, len(batch.Commands)*6*24)

	for _, cmd := range batch.Commands {
		vertices, err := commandToVertices(cmd, fbWidth, fbHeight, atlas)
		if err != nil {
			return nil, err
		}

		for _, v := range vertices {
			data = appendVertex(data, v)
		}
	}

	return data, nil
}

// commandToVertices converts a draw command to a list of vertices.
func commandToVertices(cmd displaylist.DrawCommand, fbWidth, fbHeight int, atlas *text.Atlas) ([]Vertex, error) {
	switch cmd.Type {
	case displaylist.CmdFillRect:
		return rectToVertices(cmd.Data.(displaylist.FillRectData), fbWidth, fbHeight), nil

	case displaylist.CmdFillRoundedRect:
		return roundedRectToVertices(cmd.Data.(displaylist.FillRoundedRectData), fbWidth, fbHeight), nil

	case displaylist.CmdDrawLine:
		return lineToVertices(cmd.Data.(displaylist.DrawLineData), fbWidth, fbHeight), nil

	case displaylist.CmdDrawText:
		return textToVertices(cmd.Data.(displaylist.DrawTextData), fbWidth, fbHeight, atlas), nil

	case displaylist.CmdLinearGradient:
		return linearGradientToVertices(cmd.Data.(displaylist.LinearGradientData), fbWidth, fbHeight), nil

	case displaylist.CmdRadialGradient:
		return radialGradientToVertices(cmd.Data.(displaylist.RadialGradientData), fbWidth, fbHeight), nil

	case displaylist.CmdBoxShadow:
		return boxShadowToVertices(cmd.Data.(displaylist.BoxShadowData), fbWidth, fbHeight), nil

	case displaylist.CmdDrawImage:
		return imageToVertices(cmd.Data.(displaylist.DrawImageData), fbWidth, fbHeight), nil

	default:
		return nil, fmt.Errorf("unknown command type %d", cmd.Type)
	}
}

// makeUniformQuad builds 6 vertices (two triangles) forming a rectangle with
// standard unit UV coordinates and a single uniform color.
func makeUniformQuad(x0, y0, x1, y1 float32, r, g, b, a uint8) []Vertex {
	return []Vertex{
		{x0, y0, 0, 0, r, g, b, a},
		{x1, y0, 1, 0, r, g, b, a},
		{x0, y1, 0, 1, r, g, b, a},
		{x1, y0, 1, 0, r, g, b, a},
		{x1, y1, 1, 1, r, g, b, a},
		{x0, y1, 0, 1, r, g, b, a},
	}
}

// rectToNDC converts pixel-space rectangle coordinates to OpenGL normalized
// device coordinates (NDC) in the range [-1, 1].
func rectToNDC(x, y, w, h, fbWidth, fbHeight int) (x0, y0, x1, y1 float32) {
	x0 = float32(x*2)/float32(fbWidth) - 1.0
	y0 = 1.0 - float32(y*2)/float32(fbHeight)
	x1 = float32((x+w)*2)/float32(fbWidth) - 1.0
	y1 = 1.0 - float32((y+h)*2)/float32(fbHeight)
	return x0, y0, x1, y1
}

// rectToVertices converts a filled rectangle to 6 vertices (2 triangles).
func rectToVertices(data displaylist.FillRectData, fbWidth, fbHeight int) []Vertex {
	x0, y0, x1, y1 := rectToNDC(data.X, data.Y, data.Width, data.Height, fbWidth, fbHeight)
	r, g, b, a := data.Color.R, data.Color.G, data.Color.B, data.Color.A
	return makeUniformQuad(x0, y0, x1, y1, r, g, b, a)
}

// roundedRectToVertices converts a rounded rectangle to vertices.
func roundedRectToVertices(data displaylist.FillRoundedRectData, fbWidth, fbHeight int) []Vertex {
	// Simplified: treat as regular rect for now (SDF rounding handled in fragment shader)
	x0, y0, x1, y1 := rectToNDC(data.X, data.Y, data.Width, data.Height, fbWidth, fbHeight)
	r, g, b, a := data.Color.R, data.Color.G, data.Color.B, data.Color.A
	return makeUniformQuad(x0, y0, x1, y1, r, g, b, a)
}

// lineToVertices converts a line to vertices (as a thin rectangle).
func lineToVertices(data displaylist.DrawLineData, fbWidth, fbHeight int) []Vertex {
	// Calculate line direction and perpendicular
	dx := float32(data.X1 - data.X0)
	dy := float32(data.Y1 - data.Y0)
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))

	if length < 0.001 {
		return nil // Degenerate line
	}

	// Normalized perpendicular vector
	halfWidth := float32(data.Width) / 2.0
	px := -dy / length * halfWidth
	py := dx / length * halfWidth

	// Convert to NDC
	x0 := float32(data.X0*2)/float32(fbWidth) - 1.0
	y0 := 1.0 - float32(data.Y0*2)/float32(fbHeight)
	x1 := float32(data.X1*2)/float32(fbWidth) - 1.0
	y1 := 1.0 - float32(data.Y1*2)/float32(fbHeight)

	pxNDC := px * 2.0 / float32(fbWidth)
	pyNDC := -py * 2.0 / float32(fbHeight)

	r, g, b, a := data.Color.R, data.Color.G, data.Color.B, data.Color.A

	return []Vertex{
		{x0 + pxNDC, y0 + pyNDC, 0, 0, r, g, b, a},
		{x1 + pxNDC, y1 + pyNDC, 1, 0, r, g, b, a},
		{x0 - pxNDC, y0 - pyNDC, 0, 1, r, g, b, a},
		{x1 + pxNDC, y1 + pyNDC, 1, 0, r, g, b, a},
		{x1 - pxNDC, y1 - pyNDC, 1, 1, r, g, b, a},
		{x0 - pxNDC, y0 - pyNDC, 0, 1, r, g, b, a},
	}
}

// textToVertices converts a text draw command to glyph quad vertices.
//
// Each character in the string becomes a textured quad (6 vertices) whose UV
// coordinates reference the glyph's region in the SDF font atlas. Returns nil
// when the atlas is unavailable or the string is empty.
func textToVertices(data displaylist.DrawTextData, fbWidth, fbHeight int, atlas *text.Atlas) []Vertex {
	if atlas == nil || data.Text == "" || atlas.Baseline <= 0 {
		return nil
	}

	scale := float64(data.FontSize) / atlas.Baseline
	if scale <= 0 {
		return nil
	}

	r, g, b, a := data.Color.R, data.Color.G, data.Color.B, data.Color.A
	var verts []Vertex
	penX := float64(data.X)

	for _, ch := range data.Text {
		glyph, err := atlas.GetGlyph(ch)
		if err != nil {
			continue
		}
		verts = append(verts, glyphToVertices(glyph, penX, float64(data.Y), scale, fbWidth, fbHeight, r, g, b, a, atlas.Width, atlas.Height)...)
		penX += glyph.Advance * scale
	}

	return verts
}

// glyphToVertices generates 6 vertices (two triangles) for a single glyph quad.
// UV coordinates address the glyph's texel region within the SDF atlas texture.
func glyphToVertices(glyph *text.Glyph, penX, penY, scale float64, fbWidth, fbHeight int, r, g, b, a uint8, atlasW, atlasH int) []Vertex {
	sx0 := penX + glyph.OffsetX*scale
	sy0 := penY + glyph.OffsetY*scale
	sx1 := sx0 + float64(glyph.Width)*scale
	sy1 := sy0 + float64(glyph.Height)*scale

	x0 := float32(sx0*2)/float32(fbWidth) - 1.0
	y0 := 1.0 - float32(sy0*2)/float32(fbHeight)
	x1 := float32(sx1*2)/float32(fbWidth) - 1.0
	y1 := 1.0 - float32(sy1*2)/float32(fbHeight)

	u0 := float32(glyph.X) / float32(atlasW)
	v0 := float32(glyph.Y) / float32(atlasH)
	u1 := float32(glyph.X+glyph.Width) / float32(atlasW)
	v1 := float32(glyph.Y+glyph.Height) / float32(atlasH)

	return []Vertex{
		{x0, y0, u0, v0, r, g, b, a},
		{x1, y0, u1, v0, r, g, b, a},
		{x0, y1, u0, v1, r, g, b, a},
		{x1, y0, u1, v0, r, g, b, a},
		{x1, y1, u1, v1, r, g, b, a},
		{x0, y1, u0, v1, r, g, b, a},
	}
}

// linearGradientToVertices converts a linear gradient to vertices.
func linearGradientToVertices(data displaylist.LinearGradientData, fbWidth, fbHeight int) []Vertex {
	x0 := float32(data.X*2)/float32(fbWidth) - 1.0
	y0 := 1.0 - float32(data.Y*2)/float32(fbHeight)
	x1 := float32((data.X+data.Width)*2)/float32(fbWidth) - 1.0
	y1 := 1.0 - float32((data.Y+data.Height)*2)/float32(fbHeight)

	// Encode gradient direction in UV coordinates
	u0 := float32(data.X0) / float32(data.Width)
	v0 := float32(data.Y0) / float32(data.Height)
	u1 := float32(data.X1) / float32(data.Width)
	v1 := float32(data.Y1) / float32(data.Height)

	r0, g0, b0, a0 := data.Color0.R, data.Color0.G, data.Color0.B, data.Color0.A
	r1, g1, b1, a1 := data.Color1.R, data.Color1.G, data.Color1.B, data.Color1.A

	return []Vertex{
		{x0, y0, u0, v0, r0, g0, b0, a0},
		{x1, y0, u1, v0, r1, g1, b1, a1},
		{x0, y1, u0, v1, r0, g0, b0, a0},
		{x1, y0, u1, v0, r1, g1, b1, a1},
		{x1, y1, u1, v1, r1, g1, b1, a1},
		{x0, y1, u0, v1, r0, g0, b0, a0},
	}
}

// radialGradientToVertices converts a radial gradient to vertices.
func radialGradientToVertices(data displaylist.RadialGradientData, fbWidth, fbHeight int) []Vertex {
	x0 := float32(data.X*2)/float32(fbWidth) - 1.0
	y0 := 1.0 - float32(data.Y*2)/float32(fbHeight)
	x1 := float32((data.X+data.Width)*2)/float32(fbWidth) - 1.0
	y1 := 1.0 - float32((data.Y+data.Height)*2)/float32(fbHeight)

	r0, g0, b0, a0 := data.Color0.R, data.Color0.G, data.Color0.B, data.Color0.A
	r1, g1, b1, a1 := data.Color1.R, data.Color1.G, data.Color1.B, data.Color1.A

	return []Vertex{
		{x0, y0, 0, 0, r0, g0, b0, a0},
		{x1, y0, 1, 0, r1, g1, b1, a1},
		{x0, y1, 0, 1, r0, g0, b0, a0},
		{x1, y0, 1, 0, r1, g1, b1, a1},
		{x1, y1, 1, 1, r1, g1, b1, a1},
		{x0, y1, 0, 1, r0, g0, b0, a0},
	}
}

// boxShadowToVertices converts a box shadow to vertices (simplified).
func boxShadowToVertices(data displaylist.BoxShadowData, fbWidth, fbHeight int) []Vertex {
	// Expand rect by blur + spread radius
	blur := data.BlurRadius + data.SpreadRadius
	x0 := float32((data.X-blur)*2)/float32(fbWidth) - 1.0
	y0 := 1.0 - float32((data.Y-blur)*2)/float32(fbHeight)
	x1 := float32((data.X+data.Width+blur)*2)/float32(fbWidth) - 1.0
	y1 := 1.0 - float32((data.Y+data.Height+blur)*2)/float32(fbHeight)

	r, g, b, a := data.Color.R, data.Color.G, data.Color.B, data.Color.A
	return makeUniformQuad(x0, y0, x1, y1, r, g, b, a)
}

// imageToVertices converts an image draw to vertices.
func imageToVertices(data displaylist.DrawImageData, fbWidth, fbHeight int) []Vertex {
	x0 := float32(data.X*2)/float32(fbWidth) - 1.0
	y0 := 1.0 - float32(data.Y*2)/float32(fbHeight)
	x1 := float32((data.X+data.Width)*2)/float32(fbWidth) - 1.0
	y1 := 1.0 - float32((data.Y+data.Height)*2)/float32(fbHeight)

	// White color for textured quads (texture provides color)
	r, g, b, a := uint8(255), uint8(255), uint8(255), uint8(255)

	return []Vertex{
		{x0, y0, data.U0, data.V0, r, g, b, a},
		{x1, y0, data.U1, data.V0, r, g, b, a},
		{x0, y1, data.U0, data.V1, r, g, b, a},
		{x1, y0, data.U1, data.V0, r, g, b, a},
		{x1, y1, data.U1, data.V1, r, g, b, a},
		{x0, y1, data.U0, data.V1, r, g, b, a},
	}
}

// appendVertex appends a single vertex to the byte slice.
func appendVertex(data []byte, v Vertex) []byte {
	// Position (8 bytes)
	data = binary.LittleEndian.AppendUint32(data, math.Float32bits(v.X))
	data = binary.LittleEndian.AppendUint32(data, math.Float32bits(v.Y))

	// UV (8 bytes)
	data = binary.LittleEndian.AppendUint32(data, math.Float32bits(v.U))
	data = binary.LittleEndian.AppendUint32(data, math.Float32bits(v.V))

	// Color (4 bytes)
	data = append(data, v.R, v.G, v.B, v.A)

	return data
}

// unused placeholder to satisfy the compiler for primitives.Color usage
var _ = primitives.Color{}
