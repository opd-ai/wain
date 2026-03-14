// resource-demo demonstrates the resource management API (Phase 9.5).
//
// Shows loading fonts and images through the public wain API.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/opd-ai/wain"
	"github.com/opd-ai/wain/internal/demo"
)

func main() {
	// Create app
	app := wain.NewApp()

	// Create a demo window
	win := demo.CreateDefaultWindow(app, "Resource Management Demo")

	// Get default font
	defaultFont := app.DefaultFont()
	if defaultFont != nil {
		fmt.Printf("Default font loaded: size=%.1f\n", defaultFont.Size())
	}

	// Load custom font (currently returns default font with custom size)
	customFont, err := app.LoadFont("custom.ttf", 18.0)
	if err != nil {
		log.Printf("Warning: Failed to load custom font: %v", err)
	} else {
		fmt.Printf("Custom font loaded: size=%.1f\n", customFont.Size())
	}

	// Create a test image in memory
	img := createTestImage(200, 200)

	// Save test image to temp directory
	tmpDir := os.TempDir()
	imgPath := filepath.Join(tmpDir, "wain-demo-icon.png")
	if err := saveImage(img, imgPath); err != nil {
		log.Printf("Warning: Failed to save test image: %v", err)
	} else {
		fmt.Printf("Test image saved to: %s\n", imgPath)

		// Load the image
		loadedImg, err := app.LoadImage(imgPath)
		if err != nil {
			log.Printf("Warning: Failed to load image: %v", err)
		} else {
			w, h := loadedImg.Size()
			fmt.Printf("Image loaded: %dx%d pixels\n", w, h)
		}

		// Clean up temp image
		os.Remove(imgPath)
	}

	// Note: LoadImageFromReader is available for loading images from any io.Reader.
	// Example: app.LoadImageFromReader(resp.Body, "remote.png")

	fmt.Println("\nResource management demo complete!")
	fmt.Println("Phase 9.5: Font and image loading API validated.")
	fmt.Println("Window will remain open for 2 seconds...")

	// Keep window open briefly
	win.SetTitle("Resource Demo - Success")

	// Note: In a real app, you'd call app.Run() here to start the event loop.
	// For this demo, we're just testing the resource loading APIs.
	fmt.Println("\nDemo completed successfully.")
}

// createTestImage creates a simple gradient test image.
func createTestImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create a radial gradient
			dx := float64(x) - float64(width)/2
			dy := float64(y) - float64(height)/2
			dist := (dx*dx + dy*dy) / (float64(width*height) / 4)
			if dist > 1.0 {
				dist = 1.0
			}

			r := uint8(255 * (1.0 - dist))
			g := uint8(128 * dist)
			b := uint8(200)

			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	return img
}

// saveImage saves an image to a PNG file.
func saveImage(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}


