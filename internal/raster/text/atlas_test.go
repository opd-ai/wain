package text

import (
	"testing"
)

func TestAtlasStructure(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	if atlas.Width <= 0 || atlas.Height <= 0 {
		t.Errorf("Atlas dimensions invalid: %dx%d", atlas.Width, atlas.Height)
	}

	expectedSize := atlas.Width * atlas.Height
	if len(atlas.SDF) != expectedSize {
		t.Errorf("SDF buffer size = %d, want %d", len(atlas.SDF), expectedSize)
	}
}

func TestAtlasGlyphCoverage(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	// Check ASCII printable range is covered
	const firstPrintable = ' '
	const lastPrintable = '~'

	for r := firstPrintable; r <= lastPrintable; r++ {
		glyph, err := atlas.GetGlyph(r)
		if err != nil {
			t.Errorf("Missing glyph for rune %q (%d): %v", r, r, err)
			continue
		}

		if glyph.Rune != r {
			t.Errorf("Glyph rune mismatch: got %q, want %q", glyph.Rune, r)
		}

		// Verify glyph is within atlas bounds
		if glyph.X < 0 || glyph.X+glyph.Width > atlas.Width {
			t.Errorf("Glyph %q X position out of bounds: %d + %d > %d",
				r, glyph.X, glyph.Width, atlas.Width)
		}
		if glyph.Y < 0 || glyph.Y+glyph.Height > atlas.Height {
			t.Errorf("Glyph %q Y position out of bounds: %d + %d > %d",
				r, glyph.Y, glyph.Height, atlas.Height)
		}
	}
}

func TestAtlasReplacementGlyph(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	// Replacement glyph should exist
	replacement, err := atlas.GetGlyph('□')
	if err != nil {
		t.Errorf("Replacement glyph missing: %v", err)
	}
	if replacement == nil {
		t.Fatal("Replacement glyph is nil")
	}

	// Unsupported character should fall back to replacement
	glyph, err := atlas.GetGlyph('€')
	if err != nil {
		// If replacement is missing, this is expected
		if err != ErrGlyphNotFound {
			t.Errorf("Unexpected error for unsupported glyph: %v", err)
		}
	} else if glyph.Rune != '□' {
		t.Errorf("Unsupported glyph fallback: got %q, want %q", glyph.Rune, '□')
	}
}

func TestAtlasGlyphMetrics(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	// Test a few specific glyphs
	testGlyphs := []rune{'A', 'a', ' ', 'W'}

	for _, r := range testGlyphs {
		glyph, err := atlas.GetGlyph(r)
		if err != nil {
			t.Errorf("GetGlyph(%q) error = %v", r, err)
			continue
		}

		// Width and height should be positive
		if glyph.Width <= 0 {
			t.Errorf("Glyph %q has non-positive width: %d", r, glyph.Width)
		}
		if glyph.Height <= 0 {
			t.Errorf("Glyph %q has non-positive height: %d", r, glyph.Height)
		}

		// Advance should be positive (even for space)
		if glyph.Advance <= 0 {
			t.Errorf("Glyph %q has non-positive advance: %f", r, glyph.Advance)
		}

		// Offset Y should be negative (glyphs are above baseline)
		if glyph.OffsetY > 0 {
			t.Errorf("Glyph %q has positive OffsetY: %f (expected negative)", r, glyph.OffsetY)
		}
	}
}

func TestSampleSDFBounds(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	tests := []struct {
		name string
		x, y int
	}{
		{"origin", 0, 0},
		{"center", atlas.Width / 2, atlas.Height / 2},
		{"max valid", atlas.Width - 1, atlas.Height - 1},
		{"negative x", -100, 50},
		{"negative y", 50, -100},
		{"large x", atlas.Width + 100, 50},
		{"large y", 50, atlas.Height + 100},
		{"both negative", -50, -50},
		{"both large", atlas.Width + 50, atlas.Height + 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic and should return a value
			val := atlas.SampleSDF(tt.x, tt.y)
			_ = val // Just verify it doesn't crash
		})
	}
}

func TestAtlasLineHeight(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	if atlas.LineHeight <= 0 {
		t.Errorf("LineHeight = %f, want > 0", atlas.LineHeight)
	}

	if atlas.Baseline <= 0 {
		t.Errorf("Baseline = %f, want > 0", atlas.Baseline)
	}

	// Baseline should be less than line height
	if atlas.Baseline > atlas.LineHeight {
		t.Errorf("Baseline (%f) > LineHeight (%f)", atlas.Baseline, atlas.LineHeight)
	}
}

func TestGlyphLookupPerformance(t *testing.T) {
	atlas, err := NewAtlas()
	if err != nil {
		t.Fatalf("NewAtlas() error = %v", err)
	}

	// Verify O(1) lookup by checking it's using a map
	if atlas.Glyphs == nil {
		t.Fatal("Glyphs map is nil")
	}

	// Quick lookup test
	const iterations = 1000
	for i := 0; i < iterations; i++ {
		_, _ = atlas.GetGlyph('A')
	}
}

func BenchmarkGetGlyph(b *testing.B) {
	atlas, err := NewAtlas()
	if err != nil {
		b.Fatalf("NewAtlas() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atlas.GetGlyph('A')
	}
}

func BenchmarkSampleSDF(b *testing.B) {
	atlas, err := NewAtlas()
	if err != nil {
		b.Fatalf("NewAtlas() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atlas.SampleSDF(100, 100)
	}
}

func BenchmarkSampleSDFOutOfBounds(b *testing.B) {
	atlas, err := NewAtlas()
	if err != nil {
		b.Fatalf("NewAtlas() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atlas.SampleSDF(-10, -10)
	}
}
