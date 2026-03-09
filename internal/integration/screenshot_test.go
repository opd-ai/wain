// Package integration provides cross-backend integration tests.
//
// Phase 7.3: Screenshot comparison tests that render the same scene on all
// available backends (Intel GPU, AMD GPU, software) and verify output parity.
package integration

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

const (
	// screenshotWidth is the width of the test render target
	screenshotWidth = 256

	// screenshotHeight is the height of the test render target
	screenshotHeight = 256

	// pixelTolerance is the maximum per-channel difference allowed
	// This accounts for minor precision differences between backends
	pixelTolerance = 2
)

// Backend represents a rendering backend for testing
type Backend string

const (
	BackendSoftware Backend = "software"
	BackendIntelGPU Backend = "intel_gpu"
	BackendAMDGPU   Backend = "amd_gpu"
)

// TestScene represents a test scene with expected visual output
type TestScene struct {
	Name        string
	BuildFunc   func(*displaylist.DisplayList)
	Description string
}

// buildSimpleRect creates a simple solid rectangle scene
func buildSimpleRect(dl *displaylist.DisplayList) {
	dl.AddFillRect(50, 50, 100, 100, primitives.Color{R: 255, G: 0, B: 0, A: 255})
}

// buildRoundedRect creates a rounded rectangle scene
func buildRoundedRect(dl *displaylist.DisplayList) {
	dl.AddFillRoundedRect(40, 40, 120, 80, 15, primitives.Color{R: 0, G: 128, B: 255, A: 255})
}

// buildLinearGradient creates a linear gradient scene
func buildLinearGradient(dl *displaylist.DisplayList) {
	dl.AddLinearGradient(
		30, 30, 180, 120, // bounds
		30, 75, // start point
		210, 75, // end point
		primitives.Color{R: 255, G: 200, B: 0, A: 255}, // color0
		primitives.Color{R: 255, G: 0, B: 128, A: 255}, // color1
	)
}

// buildRadialGradient creates a radial gradient scene
func buildRadialGradient(dl *displaylist.DisplayList) {
	dl.AddRadialGradient(
		30, 30, 180, 120,
		120, 90, 60,
		primitives.Color{R: 255, G: 255, B: 0, A: 255},
		primitives.Color{R: 128, G: 0, B: 255, A: 255},
	)
}

// buildMultiRect creates multiple overlapping rectangles
func buildMultiRect(dl *displaylist.DisplayList) {
	dl.AddFillRect(20, 20, 80, 80, primitives.Color{R: 255, G: 0, B: 0, A: 200})
	dl.AddFillRect(60, 60, 80, 80, primitives.Color{R: 0, G: 255, B: 0, A: 200})
	dl.AddFillRect(100, 100, 80, 80, primitives.Color{R: 0, G: 0, B: 255, A: 200})
}

// buildComplexScene creates a complex scene with multiple primitives
func buildComplexScene(dl *displaylist.DisplayList) {
	// Background
	dl.AddFillRect(0, 0, screenshotWidth, screenshotHeight, primitives.Color{R: 240, G: 240, B: 240, A: 255})

	// Rounded rects with gradients
	dl.AddFillRoundedRect(10, 10, 100, 60, 10, primitives.Color{R: 100, G: 150, B: 250, A: 255})
	dl.AddFillRoundedRect(120, 10, 100, 60, 10, primitives.Color{R: 250, G: 150, B: 100, A: 255})

	// Linear gradient
	dl.AddLinearGradient(
		10, 80, 210, 40,
		10, 100, 220, 100,
		primitives.Color{R: 255, G: 100, B: 100, A: 255},
		primitives.Color{R: 100, G: 100, B: 255, A: 255},
	)

	// Radial gradient
	dl.AddRadialGradient(
		10, 130, 210, 100,
		115, 180, 50,
		primitives.Color{R: 255, G: 255, B: 0, A: 255},
		primitives.Color{R: 255, G: 0, B: 255, A: 255},
	)
}

// Test scenes for cross-backend comparison
var testScenes = []TestScene{
	{
		Name:        "simple_rect",
		BuildFunc:   buildSimpleRect,
		Description: "Single solid red rectangle",
	},
	{
		Name:        "rounded_rect",
		BuildFunc:   buildRoundedRect,
		Description: "Blue rounded rectangle",
	},
	{
		Name:        "linear_gradient",
		BuildFunc:   buildLinearGradient,
		Description: "Linear gradient from yellow to pink",
	},
	{
		Name:        "radial_gradient",
		BuildFunc:   buildRadialGradient,
		Description: "Radial gradient from yellow to purple",
	},
	{
		Name:        "multi_rect",
		BuildFunc:   buildMultiRect,
		Description: "Three overlapping semi-transparent rectangles",
	},
	{
		Name:        "complex_scene",
		BuildFunc:   buildComplexScene,
		Description: "Complex scene with multiple primitives and gradients",
	},
}

// renderSoftware renders a display list using the software backend
func renderSoftware(dl *displaylist.DisplayList) (*primitives.Buffer, error) {
	buf, err := primitives.NewBuffer(screenshotWidth, screenshotHeight)
	if err != nil {
		return nil, fmt.Errorf("buffer creation failed: %w", err)
	}

	atlas, err := text.NewAtlas()
	if err != nil {
		return nil, fmt.Errorf("atlas creation failed: %w", err)
	}

	consumer := consumer.NewSoftwareConsumer(atlas)
	if err := consumer.Render(dl, buf); err != nil {
		return nil, fmt.Errorf("software render failed: %w", err)
	}

	return buf, nil
}

// compareBuffers compares two buffers pixel-by-pixel with tolerance
func compareBuffers(buf1, buf2 *primitives.Buffer, tolerance uint8) error {
	if buf1.Width != buf2.Width || buf1.Height != buf2.Height {
		return fmt.Errorf("buffer dimension mismatch: %dx%d vs %dx%d",
			buf1.Width, buf1.Height, buf2.Width, buf2.Height)
	}

	var mismatches []struct {
		x, y     int
		c1, c2   primitives.Color
		maxDelta int
	}

	for y := 0; y < buf1.Height; y++ {
		for x := 0; x < buf1.Width; x++ {
			c1 := buf1.At(x, y)
			c2 := buf2.At(x, y)

			deltaR := abs(int(c1.R) - int(c2.R))
			deltaG := abs(int(c1.G) - int(c2.G))
			deltaB := abs(int(c1.B) - int(c2.B))
			deltaA := abs(int(c1.A) - int(c2.A))

			maxDelta := max(max(deltaR, deltaG), max(deltaB, deltaA))

			if maxDelta > int(tolerance) {
				mismatches = append(mismatches, struct {
					x, y     int
					c1, c2   primitives.Color
					maxDelta int
				}{x, y, c1, c2, maxDelta})

				if len(mismatches) >= 20 {
					goto done
				}
			}
		}
	}

done:
	if len(mismatches) == 0 {
		return nil
	}

	// Calculate total mismatched pixels
	totalMismatches := 0
	for y := 0; y < buf1.Height; y++ {
		for x := 0; x < buf1.Width; x++ {
			c1 := buf1.At(x, y)
			c2 := buf2.At(x, y)

			deltaR := abs(int(c1.R) - int(c2.R))
			deltaG := abs(int(c1.G) - int(c2.G))
			deltaB := abs(int(c1.B) - int(c2.B))
			deltaA := abs(int(c1.A) - int(c2.A))

			if max(max(deltaR, deltaG), max(deltaB, deltaA)) > int(tolerance) {
				totalMismatches++
			}
		}
	}

	msg := fmt.Sprintf("Buffers differ: %d/%d pixels mismatched (%.2f%%, tolerance=%d)\n",
		totalMismatches, buf1.Width*buf1.Height,
		100.0*float64(totalMismatches)/float64(buf1.Width*buf1.Height),
		tolerance)

	msg += "First 10 mismatches:\n"
	for i, m := range mismatches {
		if i >= 10 {
			break
		}
		msg += fmt.Sprintf("  (%d,%d): RGBA(%d,%d,%d,%d) vs RGBA(%d,%d,%d,%d) delta=%d\n",
			m.x, m.y,
			m.c1.R, m.c1.G, m.c1.B, m.c1.A,
			m.c2.R, m.c2.G, m.c2.B, m.c2.A,
			m.maxDelta)
	}

	return fmt.Errorf("%s", msg)
}

// savePNG saves a buffer as PNG to the specified path (for debugging)
func savePNG(buf *primitives.Buffer, path string) error {
	img := image.NewRGBA(image.Rect(0, 0, buf.Width, buf.Height))

	for y := 0; y < buf.Height; y++ {
		for x := 0; x < buf.Width; x++ {
			c := buf.At(x, y)
			img.SetRGBA(x, y, color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A})
		}
	}

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

// bufferToBytes converts a buffer to raw RGBA bytes for comparison
func bufferToBytes(buf *primitives.Buffer) []byte {
	data := make([]byte, buf.Width*buf.Height*4)
	for y := 0; y < buf.Height; y++ {
		for x := 0; x < buf.Width; x++ {
			pixel := buf.At(x, y)
			idx := (y*buf.Width + x) * 4
			data[idx+0] = pixel.R
			data[idx+1] = pixel.G
			data[idx+2] = pixel.B
			data[idx+3] = pixel.A
		}
	}
	return data
}

// TestScreenshotComparison tests that software backend produces consistent output
func TestScreenshotComparison(t *testing.T) {
	for _, scene := range testScenes {
		t.Run(scene.Name, func(t *testing.T) {
			// Build display list
			dl := displaylist.New()
			scene.BuildFunc(dl)

			// Render with software backend
			buf1, err := renderSoftware(dl)
			if err != nil {
				t.Fatalf("First render failed: %v", err)
			}

			// Render again with software backend (should be identical)
			buf2, err := renderSoftware(dl)
			if err != nil {
				t.Fatalf("Second render failed: %v", err)
			}

			// Compare - software backend should be deterministic
			if err := compareBuffers(buf1, buf2, 0); err != nil {
				t.Errorf("Software backend is non-deterministic: %v", err)
			}
		})
	}
}

// TestScreenshotGoldenImages tests software backend against golden reference images
func TestScreenshotGoldenImages(t *testing.T) {
	// This test verifies that the software backend produces expected output
	// Golden images would be stored in testdata/ directory

	for _, scene := range testScenes {
		t.Run(scene.Name, func(t *testing.T) {
			dl := displaylist.New()
			scene.BuildFunc(dl)

			buf, err := renderSoftware(dl)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Path for golden image
			goldenPath := filepath.Join("testdata", "golden", scene.Name+".rgba")

			// Check if GENERATE_GOLDEN env var is set
			if os.Getenv("GENERATE_GOLDEN") == "1" {
				// Generate golden image
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
					t.Fatalf("Failed to create golden directory: %v", err)
				}
				if err := os.WriteFile(goldenPath, bufferToBytes(buf), 0o644); err != nil {
					t.Fatalf("Failed to write golden image: %v", err)
				}
				t.Logf("Generated golden image: %s", goldenPath)
				return
			}

			// Load golden image
			goldenData, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Skipf("Golden image not found: %s (run with GENERATE_GOLDEN=1 to create)", goldenPath)
				return
			}
			if err != nil {
				t.Fatalf("Failed to read golden image: %v", err)
			}

			// Convert golden data to buffer
			if len(goldenData) != screenshotWidth*screenshotHeight*4 {
				t.Fatalf("Invalid golden image size: expected %d bytes, got %d",
					screenshotWidth*screenshotHeight*4, len(goldenData))
			}

			goldenBuf, err := primitives.NewBuffer(screenshotWidth, screenshotHeight)
			if err != nil {
				t.Fatalf("Failed to create golden buffer: %v", err)
			}
			for y := 0; y < screenshotHeight; y++ {
				for x := 0; x < screenshotWidth; x++ {
					idx := (y*screenshotWidth + x) * 4
					color := primitives.Color{
						R: goldenData[idx+0],
						G: goldenData[idx+1],
						B: goldenData[idx+2],
						A: goldenData[idx+3],
					}
					goldenBuf.Set(x, y, color)
				}
			}

			// Compare against golden
			if err := compareBuffers(buf, goldenBuf, pixelTolerance); err != nil {
				// Save actual output for debugging
				debugPath := filepath.Join("testdata", "debug", scene.Name+".png")
				if saveErr := savePNG(buf, debugPath); saveErr == nil {
					t.Logf("Saved actual output to: %s", debugPath)
				}

				goldenPNGPath := filepath.Join("testdata", "debug", scene.Name+"_golden.png")
				if saveErr := savePNG(goldenBuf, goldenPNGPath); saveErr == nil {
					t.Logf("Saved golden output to: %s", goldenPNGPath)
				}

				t.Errorf("Output differs from golden: %v", err)
			}
		})
	}
}

// TestDisplayListIntegrity tests that display lists are immutable after rendering
func TestDisplayListIntegrity(t *testing.T) {
	dl := displaylist.New()
	dl.AddFillRect(10, 10, 50, 50, primitives.Color{R: 255, G: 0, B: 0, A: 255})

	// Render multiple times
	for i := 0; i < 3; i++ {
		buf, err := renderSoftware(dl)
		if err != nil {
			t.Fatalf("Render %d failed: %v", i, err)
		}

		// Verify the red rectangle is present
		centerPixel := buf.At(35, 35)
		if centerPixel.R != 255 || centerPixel.G != 0 || centerPixel.B != 0 {
			t.Errorf("Render %d: expected red pixel at (35,35), got RGBA(%d,%d,%d,%d)",
				i, centerPixel.R, centerPixel.G, centerPixel.B, centerPixel.A)
		}
	}
}

// BenchmarkSoftwareRender benchmarks the software renderer
func BenchmarkSoftwareRender(b *testing.B) {
	for _, scene := range testScenes {
		b.Run(scene.Name, func(b *testing.B) {
			dl := displaylist.New()
			scene.BuildFunc(dl)

			atlas, err := text.NewAtlas()
			if err != nil {
				b.Fatalf("Failed to create atlas: %v", err)
			}
			consumer := consumer.NewSoftwareConsumer(atlas)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf, err := primitives.NewBuffer(screenshotWidth, screenshotHeight)
				if err != nil {
					b.Fatalf("Failed to create buffer: %v", err)
				}
				if err := consumer.Render(dl, buf); err != nil {
					b.Fatalf("Render failed: %v", err)
				}
			}
		})
	}
}

// Helper functions

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TestCrossBackendParity is a placeholder for GPU backend comparison tests
// This will be enabled when GPU backends are fully integrated
func TestCrossBackendParity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GPU backend tests in short mode")
	}

	// TODO(TD-14): Implement GPU backend rendering when Phase 5 is complete
	// For now, we test software backend consistency
	t.Skip("GPU backend integration pending - Phase 5.5 in progress")

	/*
		// Example structure for future GPU tests:
		for _, scene := range testScenes {
			t.Run(scene.Name, func(t *testing.T) {
				dl := displaylist.New()
				scene.BuildFunc(dl)

				// Render on software backend
				swBuf, err := renderSoftware(dl)
				if err != nil {
					t.Fatalf("Software render failed: %v", err)
				}

				// Render on Intel GPU backend (if available)
				if intelBuf, err := renderIntelGPU(dl); err == nil {
					if err := compareBuffers(swBuf, intelBuf, pixelTolerance); err != nil {
						t.Errorf("Intel GPU differs from software: %v", err)
					}
				}

				// Render on AMD GPU backend (if available)
				if amdBuf, err := renderAMDGPU(dl); err == nil {
					if err := compareBuffers(swBuf, amdBuf, pixelTolerance); err != nil {
						t.Errorf("AMD GPU differs from software: %v", err)
					}
				}
			})
		}
	*/
}
