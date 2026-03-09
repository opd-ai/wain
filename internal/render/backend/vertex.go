package backend

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
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
		batchData, err := packBatchVertices(batch, b.width, b.height)
		if err != nil {
			return nil, err
		}
		data = append(data, batchData...)
	}

	return data, nil
}

// packBatchVertices packs vertices for a single batch.
func packBatchVertices(batch Batch, fbWidth, fbHeight int) ([]byte, error) {
	data := make([]byte, 0, len(batch.Commands)*6*24)

	for _, cmd := range batch.Commands {
		vertices, err := commandToVertices(cmd, fbWidth, fbHeight)
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
func commandToVertices(cmd displaylist.DrawCommand, fbWidth, fbHeight int) ([]Vertex, error) {
	switch cmd.Type {
	case displaylist.CmdFillRect:
		return rectToVertices(cmd.Data.(displaylist.FillRectData), fbWidth, fbHeight), nil

	case displaylist.CmdFillRoundedRect:
		return roundedRectToVertices(cmd.Data.(displaylist.FillRoundedRectData), fbWidth, fbHeight), nil

	case displaylist.CmdDrawLine:
		return lineToVertices(cmd.Data.(displaylist.DrawLineData), fbWidth, fbHeight), nil

	case displaylist.CmdDrawText:
		// Text rendering requires glyph atlas lookup (deferred to Phase 5.2)
		return textToVertices(cmd.Data.(displaylist.DrawTextData), fbWidth, fbHeight), nil

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

// rectToVertices converts a filled rectangle to 6 vertices (2 triangles).
func rectToVertices(data displaylist.FillRectData, fbWidth, fbHeight int) []Vertex {
	// Convert pixel coordinates to normalized device coordinates [-1, 1]
	x0 := float32(data.X*2)/float32(fbWidth) - 1.0
	y0 := 1.0 - float32(data.Y*2)/float32(fbHeight)
	x1 := float32((data.X+data.Width)*2)/float32(fbWidth) - 1.0
	y1 := 1.0 - float32((data.Y+data.Height)*2)/float32(fbHeight)

	r, g, b, a := data.Color.R, data.Color.G, data.Color.B, data.Color.A

	// Two triangles forming a quad
	return []Vertex{
		{x0, y0, 0, 0, r, g, b, a},
		{x1, y0, 1, 0, r, g, b, a},
		{x0, y1, 0, 1, r, g, b, a},
		{x1, y0, 1, 0, r, g, b, a},
		{x1, y1, 1, 1, r, g, b, a},
		{x0, y1, 0, 1, r, g, b, a},
	}
}

// roundedRectToVertices converts a rounded rectangle to vertices.
func roundedRectToVertices(data displaylist.FillRoundedRectData, fbWidth, fbHeight int) []Vertex {
	// Simplified: treat as regular rect for now (SDF rounding handled in fragment shader)
	x0 := float32(data.X*2)/float32(fbWidth) - 1.0
	y0 := 1.0 - float32(data.Y*2)/float32(fbHeight)
	x1 := float32((data.X+data.Width)*2)/float32(fbWidth) - 1.0
	y1 := 1.0 - float32((data.Y+data.Height)*2)/float32(fbHeight)

	r, g, b, a := data.Color.R, data.Color.G, data.Color.B, data.Color.A

	return []Vertex{
		{x0, y0, 0, 0, r, g, b, a},
		{x1, y0, 1, 0, r, g, b, a},
		{x0, y1, 0, 1, r, g, b, a},
		{x1, y0, 1, 0, r, g, b, a},
		{x1, y1, 1, 1, r, g, b, a},
		{x0, y1, 0, 1, r, g, b, a},
	}
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

// textToVertices converts text to vertices (placeholder for Phase 5.2).
func textToVertices(data displaylist.DrawTextData, fbWidth, fbHeight int) []Vertex {
	// Phase 5.2 will implement glyph atlas lookup
	// For now, return empty (text won't render)
	return nil
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

	return []Vertex{
		{x0, y0, 0, 0, r, g, b, a},
		{x1, y0, 1, 0, r, g, b, a},
		{x0, y1, 0, 1, r, g, b, a},
		{x1, y0, 1, 0, r, g, b, a},
		{x1, y1, 1, 1, r, g, b, a},
		{x0, y1, 0, 1, r, g, b, a},
	}
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
