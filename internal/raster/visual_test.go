package raster

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/opd-ai/wain/internal/raster/consumer"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
)

// TestVisual runs visual regression tests for all rendering primitives.
// These tests generate images and compare them against reference images.
func TestVisual(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping visual regression tests in short mode")
	}

	tests := []struct {
		name     string
		width    int
		height   int
		commands func() *displaylist.DisplayList
	}{
		{
			name:   "fill_rect",
			width:  100,
			height: 100,
			commands: func() *displaylist.DisplayList {
				dl := displaylist.New()
				dl.AddFillRect(20, 20, 60, 60, primitives.Color{R: 255, G: 0, B: 0, A: 255})
				return dl
			},
		},
		{
			name:   "fill_rounded_rect",
			width:  100,
			height: 100,
			commands: func() *displaylist.DisplayList {
				dl := displaylist.New()
				dl.AddFillRoundedRect(20, 20, 60, 60, 10, primitives.Color{R: 0, G: 255, B: 0, A: 255})
				return dl
			},
		},
		{
			name:   "draw_line",
			width:  100,
			height: 100,
			commands: func() *displaylist.DisplayList {
				dl := displaylist.New()
				dl.AddDrawLine(10, 10, 90, 90, 3, primitives.Color{R: 0, G: 0, B: 255, A: 255})
				return dl
			},
		},
		{
			name:   "draw_text",
			width:  200,
			height: 50,
			commands: func() *displaylist.DisplayList {
				dl := displaylist.New()
				dl.AddDrawText("Hello", 10, 30, 16, primitives.Color{R: 255, G: 255, B: 255, A: 255}, 0)
				return dl
			},
		},
		{
			name:   "linear_gradient",
			width:  100,
			height: 100,
			commands: func() *displaylist.DisplayList {
				dl := displaylist.New()
				dl.AddLinearGradient(
					10, 10, 80, 80,
					10, 10, 90, 90,
					primitives.Color{R: 255, G: 0, B: 0, A: 255},
					primitives.Color{R: 0, G: 0, B: 255, A: 255},
				)
				return dl
			},
		},
		{
			name:   "radial_gradient",
			width:  100,
			height: 100,
			commands: func() *displaylist.DisplayList {
				dl := displaylist.New()
				dl.AddRadialGradient(
					10, 10, 80, 80,
					50, 50, 40,
					primitives.Color{R: 255, G: 255, B: 0, A: 255},
					primitives.Color{R: 255, G: 0, B: 255, A: 255},
				)
				return dl
			},
		},
		{
			name:   "box_shadow",
			width:  150,
			height: 150,
			commands: func() *displaylist.DisplayList {
				dl := displaylist.New()
				dl.AddBoxShadow(50, 50, 50, 50, 10, 5, primitives.Color{R: 0, G: 0, B: 0, A: 128})
				return dl
			},
		},
		{
			name:   "multiple_primitives",
			width:  200,
			height: 200,
			commands: func() *displaylist.DisplayList {
				dl := displaylist.New()
				dl.AddFillRect(10, 10, 60, 60, primitives.Color{R: 255, G: 0, B: 0, A: 255})
				dl.AddFillRoundedRect(80, 10, 60, 60, 8, primitives.Color{R: 0, G: 255, B: 0, A: 255})
				dl.AddDrawLine(10, 100, 190, 100, 2, primitives.Color{R: 0, G: 0, B: 255, A: 255})
				dl.AddFillRect(10, 120, 180, 70, primitives.Color{R: 128, G: 128, B: 128, A: 255})
				return dl
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate the image
			buf, err := primitives.NewBuffer(tt.width, tt.height)
			if err != nil {
				t.Fatalf("Failed to create buffer: %v", err)
			}

			buf.Clear(primitives.Color{R: 0, G: 0, B: 0, A: 0})

			// Create display list and render
			dl := tt.commands()

			// Create a minimal atlas for text rendering (can be nil for non-text tests)
			atlas, err := text.NewAtlas()
			if err != nil {
				// If atlas fails, use nil - most tests don't need text
				atlas = nil
			}
			sw := consumer.NewSoftwareConsumer(atlas)
			if err := sw.Render(dl, buf); err != nil {
				t.Fatalf("Failed to render display list: %v", err)
			}

			// Convert buffer to image.Image
			img := bufferToImage(buf)

			// Compare with reference or generate if missing
			refPath := filepath.Join("testdata", tt.name+".png")
			if err := compareOrGenerate(t, img, refPath); err != nil {
				t.Errorf("Visual regression test failed: %v", err)
			}
		})
	}
}

// bufferToImage converts a primitives.Buffer to an image.Image.
func bufferToImage(buf *primitives.Buffer) image.Image {
	bounds := image.Rect(0, 0, buf.Width, buf.Height)
	img := image.NewRGBA(bounds)

	for y := 0; y < buf.Height; y++ {
		for x := 0; x < buf.Width; x++ {
			c := buf.At(x, y)
			img.Set(x, y, color.RGBA{
				R: c.R,
				G: c.G,
				B: c.B,
				A: c.A,
			})
		}
	}

	return img
}

// compareOrGenerate compares the image against a reference, or generates it if missing.
func compareOrGenerate(t *testing.T, img image.Image, refPath string) error {
	t.Helper()

	// Check if reference exists
	if _, err := os.Stat(refPath); os.IsNotExist(err) {
		// Generate reference image
		t.Logf("Generating reference image: %s", refPath)
		return saveImage(img, refPath)
	}

	// Load reference image
	refImg, err := loadImage(refPath)
	if err != nil {
		return fmt.Errorf("failed to load reference image: %w", err)
	}

	// Compare images
	diff, percent := compareImages(img, refImg)
	// Visual regression threshold: 99.9% pixel match required.
	// Allows 0.1% tolerance for minor differences from:
	// - Antialiasing variations across different CPU SIMD implementations
	// - Subpixel rounding differences in gradient/curve calculations
	// - Font hinting differences when freetype library versions differ
	// For a 1920×1080 image, 0.1% = ~2073 pixels, which catches rendering bugs
	// while allowing for numerical precision variations in software rendering.
	const threshold = 99.9
	if percent < threshold {
		// Save diff image for inspection
		diffPath := filepath.Join("testdata", "diff_"+filepath.Base(refPath))
		if err := saveImage(diff, diffPath); err != nil {
			t.Logf("Failed to save diff image: %v", err)
		}
		return fmt.Errorf("image match: %.2f%% (threshold: %.1f%%), diff saved to %s", percent, threshold, diffPath)
	}

	t.Logf("Image match: %.2f%%", percent)
	return nil
}

// compareImages compares two images and returns a diff image and match percentage.
func compareImages(img1, img2 image.Image) (image.Image, float64) {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	if bounds1 != bounds2 {
		return nil, 0.0
	}

	diff := image.NewRGBA(bounds1)

	matching := 0
	total := 0

	for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
		for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
			c1 := img1.At(x, y)
			c2 := img2.At(x, y)

			r1, g1, b1, a1 := c1.RGBA()
			r2, g2, b2, a2 := c2.RGBA()

			// Compare with tolerance (255 units out of 65535)
			tolerance := uint32(255)
			if absDiff(r1, r2) <= tolerance &&
				absDiff(g1, g2) <= tolerance &&
				absDiff(b1, b2) <= tolerance &&
				absDiff(a1, a2) <= tolerance {
				matching++
				// Matching pixels are green
				diff.Set(x, y, color.RGBA{R: 0, G: 100, B: 0, A: 255})
			} else {
				// Different pixels are red
				diff.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			}
			total++
		}
	}

	percent := (float64(matching) / float64(total)) * 100.0
	return diff, percent
}

// absDiff returns the absolute difference between two uint32 values.
func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// saveImage saves an image to a file in PNG format.
func saveImage(img image.Image, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// loadImage loads an image from a file.
func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// BenchmarkVisualRender benchmarks the rendering of all primitives.
func BenchmarkVisualRender(b *testing.B) {
	buf, _ := primitives.NewBuffer(200, 200)
	dl := displaylist.New()
	dl.AddFillRect(10, 10, 60, 60, primitives.Color{R: 255, G: 0, B: 0, A: 255})
	dl.AddFillRoundedRect(80, 10, 60, 60, 8, primitives.Color{R: 0, G: 255, B: 0, A: 255})
	dl.AddDrawLine(10, 100, 190, 100, 2, primitives.Color{R: 0, G: 0, B: 255, A: 255})

	atlas, _ := text.NewAtlas()
	sw := consumer.NewSoftwareConsumer(atlas)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Clear(primitives.Color{R: 0, G: 0, B: 0, A: 0})
		sw.Render(dl, buf)
	}
}
