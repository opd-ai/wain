package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGenAtlasSmoke verifies gen-atlas compiles and runs without panicking.
// This is a smoke test to ensure the binary remains functional after code changes.
func TestGenAtlasSmoke(t *testing.T) {
	// Create temporary directory for output
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Create internal/raster/text/data directory structure in temp dir
	dataDir := filepath.Join(tmpDir, "internal", "raster", "text", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("failed to create data directory: %v", err)
	}

	// Change to temp directory so gen-atlas writes there
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Run main() - should not panic
	main()

	// Verify atlas.bin was created
	atlasPath := filepath.Join(dataDir, "atlas.bin")
	info, err := os.Stat(atlasPath)
	if err != nil {
		t.Fatalf("atlas.bin not created: %v", err)
	}

	// Verify atlas.bin has reasonable size (header + SDF data + glyph metadata)
	// Minimum: 16 bytes header + 65536 bytes SDF + ~95 glyphs * 36 bytes = ~69KB
	if info.Size() < 60000 {
		t.Errorf("atlas.bin too small: got %d bytes, expected at least 60000", info.Size())
	}
}

// TestGetCharPattern verifies character pattern lookup returns valid data.
func TestGetCharPattern(t *testing.T) {
	tests := []struct {
		char rune
		want int // expected pattern length
	}{
		{' ', 0}, // space has empty pattern
		{'!', 6}, // exclamation has pattern
		{'A', 6}, // A has pattern
		{'H', 5}, // H has pattern
		{'e', 5}, // e has pattern
		{'Z', 5}, // unknown char gets default pattern
	}

	for _, tt := range tests {
		pattern := getCharPattern(int(tt.char))
		if len(pattern) != tt.want {
			t.Errorf("getCharPattern(%q) = %d bytes, want %d", tt.char, len(pattern), tt.want)
		}
	}
}
