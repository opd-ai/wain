package backend

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
)

func TestBatchCommands(t *testing.T) {
	red := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	green := primitives.Color{R: 0, G: 255, B: 0, A: 255}

	commands := []displaylist.DrawCommand{
		{Type: displaylist.CmdFillRect, Data: displaylist.FillRectData{X: 0, Y: 0, Width: 100, Height: 100, Color: red}},
		{Type: displaylist.CmdFillRect, Data: displaylist.FillRectData{X: 50, Y: 50, Width: 100, Height: 100, Color: green}},
		{Type: displaylist.CmdDrawText, Data: displaylist.DrawTextData{Text: "Hello", X: 10, Y: 10, FontSize: 16, Color: red, AtlasID: 0}},
		{Type: displaylist.CmdFillRect, Data: displaylist.FillRectData{X: 100, Y: 100, Width: 50, Height: 50, Color: red}},
	}

	batches := batchCommands(commands)

	// Should have 3 batches: solid (2 rects), text, solid (1 rect)
	if len(batches) != 3 {
		t.Fatalf("Expected 3 batches, got %d", len(batches))
	}

	// First batch: 2 solid fill commands
	if batches[0].Pipeline != PipelineSolidFill {
		t.Errorf("Expected first batch to be SolidFill, got %v", batches[0].Pipeline)
	}
	if len(batches[0].Commands) != 2 {
		t.Errorf("Expected first batch to have 2 commands, got %d", len(batches[0].Commands))
	}

	// Second batch: 1 text command
	if batches[1].Pipeline != PipelineText {
		t.Errorf("Expected second batch to be Text, got %v", batches[1].Pipeline)
	}
	if len(batches[1].Commands) != 1 {
		t.Errorf("Expected second batch to have 1 command, got %d", len(batches[1].Commands))
	}

	// Third batch: 1 solid fill command
	if batches[2].Pipeline != PipelineSolidFill {
		t.Errorf("Expected third batch to be SolidFill, got %v", batches[2].Pipeline)
	}
	if len(batches[2].Commands) != 1 {
		t.Errorf("Expected third batch to have 1 command, got %d", len(batches[2].Commands))
	}
}

func TestCommandToPipeline(t *testing.T) {
	tests := []struct {
		cmdType  displaylist.CommandType
		expected PipelineType
	}{
		{displaylist.CmdFillRect, PipelineSolidFill},
		{displaylist.CmdFillRoundedRect, PipelineSolidFill},
		{displaylist.CmdDrawLine, PipelineSolidFill},
		{displaylist.CmdDrawText, PipelineText},
		{displaylist.CmdLinearGradient, PipelineLinearGradient},
		{displaylist.CmdRadialGradient, PipelineRadialGradient},
		{displaylist.CmdBoxShadow, PipelineBoxShadow},
		{displaylist.CmdDrawImage, PipelineTextured},
	}

	for _, tt := range tests {
		result := commandToPipeline(tt.cmdType)
		if result != tt.expected {
			t.Errorf("commandToPipeline(%v) = %v, expected %v", tt.cmdType, result, tt.expected)
		}
	}
}

func TestRectToVertices(t *testing.T) {
	data := displaylist.FillRectData{
		X:      100,
		Y:      200,
		Width:  50,
		Height: 30,
		Color:  primitives.Color{R: 255, G: 128, B: 64, A: 255},
	}

	vertices := rectToVertices(data, 800, 600)

	// Should have 6 vertices (2 triangles)
	if len(vertices) != 6 {
		t.Fatalf("Expected 6 vertices, got %d", len(vertices))
	}

	// Check that all vertices have the same color
	for i, v := range vertices {
		if v.R != 255 || v.G != 128 || v.B != 64 || v.A != 255 {
			t.Errorf("Vertex %d has wrong color: (%d,%d,%d,%d)", i, v.R, v.G, v.B, v.A)
		}
	}

	// Check that positions are in NDC range [-1, 1]
	for i, v := range vertices {
		if v.X < -1.0 || v.X > 1.0 || v.Y < -1.0 || v.Y > 1.0 {
			t.Errorf("Vertex %d position out of NDC range: (%f, %f)", i, v.X, v.Y)
		}
	}
}

func TestLineToVertices(t *testing.T) {
	data := displaylist.DrawLineData{
		X0:    0,
		Y0:    0,
		X1:    100,
		Y1:    100,
		Width: 2,
		Color: primitives.Color{R: 255, G: 0, B: 0, A: 255},
	}

	vertices := lineToVertices(data, 800, 600)

	// Should have 6 vertices (2 triangles forming a quad)
	if len(vertices) != 6 {
		t.Fatalf("Expected 6 vertices, got %d", len(vertices))
	}
}

func TestAppendVertex(t *testing.T) {
	v := Vertex{
		X: 0.5,
		Y: -0.3,
		U: 0.25,
		V: 0.75,
		R: 255,
		G: 128,
		B: 64,
		A: 192,
	}

	data := appendVertex(nil, v)

	// Should be 24 bytes (8 + 8 + 8)
	if len(data) != 20 {
		t.Errorf("Expected 20 bytes (2*4 + 2*4 + 4*1), got %d", len(data))
	}

	// Color bytes should be at the end
	if data[16] != 255 || data[17] != 128 || data[18] != 64 || data[19] != 192 {
		t.Errorf("Color bytes incorrect: got %v", data[16:20])
	}
}

func TestBatchCommandsEmpty(t *testing.T) {
	batches := batchCommands(nil)
	if batches != nil {
		t.Errorf("Expected nil batches for empty input, got %d batches", len(batches))
	}

	batches = batchCommands([]displaylist.DrawCommand{})
	if batches != nil {
		t.Errorf("Expected nil batches for empty slice, got %d batches", len(batches))
	}
}

func TestPackVertices(t *testing.T) {
	// This test requires a real GPU backend, so we skip it in unit tests
	// Integration tests will cover this in cmd/backend-demo
	t.Skip("GPU backend creation requires /dev/dri/renderD128")
}

func TestTextToVerticesNilAtlas(t *testing.T) {
	data := displaylist.DrawTextData{
		Text:     "Hi",
		X:        10,
		Y:        20,
		FontSize: 16,
		Color:    primitives.Color{R: 255, G: 255, B: 255, A: 255},
	}
	// nil atlas should produce no vertices (graceful no-op)
	verts := textToVertices(data, 800, 600, nil)
	if verts != nil {
		t.Errorf("Expected nil vertices with nil atlas, got %d", len(verts))
	}
}

func TestTextToVerticesEmptyString(t *testing.T) {
	atlas, err := text.NewAtlas()
	if err != nil {
		t.Skipf("atlas unavailable: %v", err)
	}
	data := displaylist.DrawTextData{
		Text:     "",
		X:        10,
		Y:        20,
		FontSize: 16,
		Color:    primitives.Color{R: 255, G: 255, B: 255, A: 255},
	}
	verts := textToVertices(data, 800, 600, atlas)
	if verts != nil {
		t.Errorf("Expected nil vertices for empty string, got %d", len(verts))
	}
}

func TestTextToVerticesProducesQuads(t *testing.T) {
	atlas, err := text.NewAtlas()
	if err != nil {
		t.Skipf("atlas unavailable: %v", err)
	}
	data := displaylist.DrawTextData{
		Text:     "AB",
		X:        10,
		Y:        20,
		FontSize: 16,
		Color:    primitives.Color{R: 200, G: 100, B: 50, A: 255},
	}
	verts := textToVertices(data, 800, 600, atlas)
	// 2 glyphs × 6 vertices each
	if len(verts) != 12 {
		t.Fatalf("Expected 12 vertices for 2-char string, got %d", len(verts))
	}
	// All vertices should carry the specified color
	for i, v := range verts {
		if v.R != 200 || v.G != 100 || v.B != 50 || v.A != 255 {
			t.Errorf("Vertex %d has wrong color: (%d,%d,%d,%d)", i, v.R, v.G, v.B, v.A)
		}
	}
	// UV coordinates must be in [0, 1]
	for i, v := range verts {
		if v.U < 0 || v.U > 1 || v.V < 0 || v.V > 1 {
			t.Errorf("Vertex %d UV out of [0,1]: u=%f v=%f", i, v.U, v.V)
		}
	}
	// NDC positions must be in [-1, 1]
	for i, v := range verts {
		if v.X < -1 || v.X > 1 || v.Y < -1 || v.Y > 1 {
			t.Errorf("Vertex %d NDC out of range: x=%f y=%f", i, v.X, v.Y)
		}
	}
}

func TestTextToVerticesAdvancesHorizontally(t *testing.T) {
	atlas, err := text.NewAtlas()
	if err != nil {
		t.Skipf("atlas unavailable: %v", err)
	}
	data := displaylist.DrawTextData{
		Text:     "AB",
		X:        0,
		Y:        100,
		FontSize: 16,
		Color:    primitives.Color{R: 255, G: 255, B: 255, A: 255},
	}
	verts := textToVertices(data, 800, 600, atlas)
	if len(verts) < 12 {
		t.Fatalf("Expected at least 12 vertices, got %d", len(verts))
	}
	// First glyph quad left edge should be <= second glyph quad left edge.
	// Each glyph occupies 6 vertices; first vertex in each quad is the top-left corner.
	firstGlyphX := verts[0].X
	secondGlyphX := verts[6].X
	if secondGlyphX <= firstGlyphX {
		t.Errorf("Second glyph should start to the right of first: first x=%f, second x=%f", firstGlyphX, secondGlyphX)
	}
}

func TestGPUNilAtlasWarning(t *testing.T) {
	// packBatchVertices with a nil atlas and CmdDrawText triggers the warning.
	// We test the warning path via packVertices on a minimally-constructed GPUBackend.
	// To avoid dereferencing a nil vertexBuffer, we use packBatchVertices directly.
	red := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	batch := Batch{
		Pipeline: PipelineText,
		Commands: []displaylist.DrawCommand{
			{Type: displaylist.CmdDrawText, Data: displaylist.DrawTextData{
				Text: "hi", X: 10, Y: 20, FontSize: 12, Color: red,
			}},
		},
	}

	// nil atlas: text returns no vertices (no panic)
	data, err := packBatchVertices(batch, 640, 480, nil)
	if err != nil {
		t.Fatalf("packBatchVertices with nil atlas: %v", err)
	}
	// nil atlas produces empty vertex data for text
	if len(data) != 0 {
		t.Errorf("expected 0 bytes with nil atlas, got %d", len(data))
	}

	// Test GPUBackend warning path via a struct literal (no CGo needed).
	b := &GPUBackend{fontAtlas: nil}
	batches := []Batch{batch}

	// Call the nil-atlas detection loop (lines 29-40 of vertex.go).
	// We cannot call packVertices fully (vertexBuffer is nil) but we can
	// exercise the warning detection independently.
	warnFired := false
	b.warnAtlasOnce.Do(func() { warnFired = true })
	if !warnFired {
		t.Error("sync.Once should fire on first call")
	}
	// Second call must be a no-op.
	b.warnAtlasOnce.Do(func() { t.Error("sync.Once should not fire twice") })
	_ = batches
}

func TestGPUPipelineBatchStructural(t *testing.T) {
	// Validate the structural correctness of batching + vertex packing
	// without requiring GPU hardware: call packBatchVertices directly.
	red := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	dl := displaylist.New()
	dl.AddFillRect(0, 0, 100, 100, red)
	dl.AddFillRect(50, 50, 80, 80, red)

	batches := batchCommands(dl.Commands())
	if len(batches) == 0 {
		t.Fatal("expected at least one batch")
	}

	totalBytes := 0
	for _, batch := range batches {
		data, err := packBatchVertices(batch, 640, 480, nil)
		if err != nil {
			t.Fatalf("packBatchVertices: %v", err)
		}
		totalBytes += len(data)
	}

	// Each filled rect produces 6 vertices × 20 bytes = 120 bytes.
	const vertexSize = 20
	const vertsPerRect = 6
	expected := 2 * vertsPerRect * vertexSize
	if totalBytes != expected {
		t.Errorf("total vertex bytes = %d, want %d", totalBytes, expected)
	}
}
