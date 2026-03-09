package text

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/primitives"
)

func TestNewAtlas(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	if atlas.Width != 256 {
		t.Errorf("atlas.Width = %d, want 256", atlas.Width)
	}
	if atlas.Height != 256 {
		t.Errorf("atlas.Height = %d, want 256", atlas.Height)
	}
	if len(atlas.SDF) != 256*256 {
		t.Errorf("len(atlas.SDF) = %d, want %d", len(atlas.SDF), 256*256)
	}
	if len(atlas.Glyphs) == 0 {
		t.Fatal("atlas.Glyphs is empty")
	}
	if atlas.LineHeight <= 0 {
		t.Errorf("atlas.LineHeight = %f, want > 0", atlas.LineHeight)
	}
}

func TestGetGlyph(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	tests := []struct {
		name    string
		rune    rune
		wantErr bool
	}{
		{"ASCII letter", 'A', false},
		{"ASCII space", ' ', false},
		{"ASCII tilde", '~', false},
		{"Unsupported char", '§', false}, // Should get replacement
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := atlas.GetGlyph(tt.rune)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGlyph() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && g == nil {
				t.Error("GetGlyph() returned nil glyph")
			}
			if g != nil && g.Width == 0 {
				t.Error("glyph has zero width")
			}
			if g != nil && g.Advance <= 0 {
				t.Error("glyph has non-positive advance")
			}
		})
	}
}

func TestSampleSDF(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	tests := []struct {
		name string
		x, y int
	}{
		{"top-left corner", 0, 0},
		{"center", 128, 128},
		{"bottom-right", 255, 255},
		{"out of bounds negative", -10, -10},
		{"out of bounds positive", 300, 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			val := atlas.SampleSDF(tt.x, tt.y)
			_ = val // Value is clamped, so all values are valid
		})
	}
}

func TestDrawText(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	buf, err := primitives.NewBuffer(200, 100)
	if err != nil {
		t.Fatalf("NewBuffer() error = %v", err)
	}

	color := primitives.Color{R: 255, G: 255, B: 255, A: 255}

	tests := []struct {
		name string
		text string
		x, y float64
		size float64
	}{
		{"simple text", "Hello", 10, 20, 16},
		{"empty string", "", 10, 20, 16},
		{"single char", "A", 10, 20, 16},
		{"large text", "BIG", 10, 30, 32},
		{"small text", "tiny", 10, 40, 8},
		{"at origin", "Origin", 0, 16, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			DrawText(buf, tt.text, tt.x, tt.y, tt.size, color, atlas)
		})
	}
}

func TestDrawTextNilInputs(t *testing.T) {
	atlas, _ := NewAtlas()
	buf, _ := primitives.NewBuffer(100, 100)
	color := primitives.Color{R: 255, G: 255, B: 255, A: 255}

	// Should not panic with nil inputs
	DrawText(nil, "test", 10, 10, 16, color, atlas)
	DrawText(buf, "test", 10, 10, 16, color, nil)
}

func TestMeasureText(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	tests := []struct {
		name       string
		text       string
		size       float64
		wantWidth  bool // true if width should be > 0
		wantHeight bool // true if height should be > 0
	}{
		{"simple text", "Hello", 16, true, true},
		{"empty string", "", 16, false, true},
		{"single char", "A", 16, true, true},
		{"spaces", "   ", 16, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := MeasureText(tt.text, tt.size, atlas)

			if tt.wantWidth && w <= 0 {
				t.Errorf("MeasureText() width = %f, want > 0", w)
			}
			if !tt.wantWidth && w != 0 {
				t.Errorf("MeasureText() width = %f, want 0", w)
			}
			if tt.wantHeight && h <= 0 {
				t.Errorf("MeasureText() height = %f, want > 0", h)
			}
		})
	}
}

func TestMeasureTextNilAtlas(t *testing.T) {
	w, h := MeasureText("test", 16, nil)
	if w != 0 || h != 0 {
		t.Errorf("MeasureText(nil atlas) = (%f, %f), want (0, 0)", w, h)
	}
}

func TestSDFToCoverage(t *testing.T) {
	tests := []struct {
		name     string
		sdfValue uint8
		scale    float64
		wantLow  uint8 // Coverage should be >= this
		wantHigh uint8 // Coverage should be <= this
	}{
		{"inside edge", 128, 1.0, 100, 200},
		{"well inside", 200, 1.0, 200, 255},
		{"well outside", 56, 1.0, 0, 100},
		{"edge at high scale", 128, 2.0, 100, 200},
		{"edge at low scale", 128, 0.5, 50, 250},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coverage := sdfToCoverage(tt.sdfValue, tt.scale)
			if coverage < tt.wantLow || coverage > tt.wantHigh {
				t.Errorf("sdfToCoverage(%d, %f) = %d, want in range [%d, %d]",
					tt.sdfValue, tt.scale, coverage, tt.wantLow, tt.wantHigh)
			}
		})
	}
}

func TestSmoothstep(t *testing.T) {
	tests := []struct {
		name  string
		edge0 float64
		edge1 float64
		x     float64
		want  float64
	}{
		{"below range", 0, 1, -0.5, 0},
		{"at start", 0, 1, 0, 0},
		{"midpoint", 0, 1, 0.5, 0.5},
		{"at end", 0, 1, 1, 1},
		{"above range", 0, 1, 1.5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := smoothstep(tt.edge0, tt.edge1, tt.x)
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.01 {
				t.Errorf("smoothstep(%f, %f, %f) = %f, want %f",
					tt.edge0, tt.edge1, tt.x, got, tt.want)
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name      string
		x, mn, mx float64
		want      float64
	}{
		{"below range", -5, 0, 10, 0},
		{"in range", 5, 0, 10, 5},
		{"above range", 15, 0, 10, 10},
		{"at min", 0, 0, 10, 0},
		{"at max", 10, 0, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clamp(tt.x, tt.mn, tt.mx)
			if got != tt.want {
				t.Errorf("clamp(%f, %f, %f) = %f, want %f",
					tt.x, tt.mn, tt.mx, got, tt.want)
			}
		})
	}
}

func TestBlendPixel(t *testing.T) {
	buf, err := primitives.NewBuffer(10, 10)
	if err != nil {
		t.Fatalf("NewBuffer() error = %v", err)
	}

	// Set a known background
	for i := range buf.Pixels {
		buf.Pixels[i] = 128
	}

	color := primitives.Color{R: 255, G: 0, B: 0, A: 255}

	// Blend at various alpha levels
	blendPixel(buf, 5, 5, color, 255) // Full coverage
	blendPixel(buf, 6, 5, color, 128) // Half coverage
	blendPixel(buf, 7, 5, color, 0)   // No coverage

	// Out of bounds should not panic
	blendPixel(buf, -1, -1, color, 255)
	blendPixel(buf, 100, 100, color, 255)
}

func BenchmarkDrawText(b *testing.B) {
	atlas, err := NewAtlas()
	if err != nil {
		b.Fatalf("NewAtlas() error = %v", err)
	}

	buf, err := primitives.NewBuffer(800, 600)
	if err != nil {
		b.Fatalf("NewBuffer() error = %v", err)
	}

	color := primitives.Color{R: 255, G: 255, B: 255, A: 255}
	text := "The quick brown fox jumps over the lazy dog"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DrawText(buf, text, 10, 50, 16, color, atlas)
	}
}

func BenchmarkMeasureText(b *testing.B) {
	atlas, err := NewAtlas()
	if err != nil {
		b.Fatalf("NewAtlas() error = %v", err)
	}

	text := "The quick brown fox jumps over the lazy dog"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MeasureText(text, 16, atlas)
	}
}
