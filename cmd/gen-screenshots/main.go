// Command gen-screenshots renders demo scenes using the software rasteriser
// and writes them to PNG files. It is used in CI to generate showcase images.
//
// Usage:
//
//	go run ./cmd/gen-screenshots [-o screenshots]
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/opd-ai/wain/internal/raster/consumer"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
)

// scene describes a single demo image to generate.
type scene struct {
	Name   string
	Width  int
	Height int
	Build  func(*displaylist.DisplayList)
}

func main() {
	outDir := flag.String("o", "screenshots", "output directory for PNG files")
	flag.Parse()

	atlas, err := text.NewAtlas()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-screenshots: %v\n", err)
		os.Exit(1)
	}

	sw := consumer.NewSoftwareConsumer(atlas)

	scenes := defineScenes()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "gen-screenshots: mkdir %s: %v\n", *outDir, err)
		os.Exit(1)
	}

	for _, sc := range scenes {
		dl := displaylist.New()
		sc.Build(dl)

		buf, err := primitives.NewBuffer(sc.Width, sc.Height)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gen-screenshots: buffer for %s: %v\n", sc.Name, err)
			os.Exit(1)
		}

		if err := sw.Render(dl, buf); err != nil {
			fmt.Fprintf(os.Stderr, "gen-screenshots: render %s: %v\n", sc.Name, err)
			os.Exit(1)
		}

		path := filepath.Join(*outDir, sc.Name+".png")
		if err := savePNG(buf, path); err != nil {
			fmt.Fprintf(os.Stderr, "gen-screenshots: save %s: %v\n", path, err)
			os.Exit(1)
		}

		fmt.Printf("✓ %s (%dx%d) → %s\n", sc.Name, sc.Width, sc.Height, path)
	}

	fmt.Printf("\nGenerated %d screenshots in %s/\n", len(scenes), *outDir)
}

// savePNG writes a primitives.Buffer to a PNG file.
func savePNG(buf *primitives.Buffer, path string) error {
	img := image.NewRGBA(image.Rect(0, 0, buf.Width, buf.Height))
	for y := 0; y < buf.Height; y++ {
		for x := 0; x < buf.Width; x++ {
			c := buf.At(x, y)
			img.SetRGBA(x, y, color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A})
		}
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// ---------------------------------------------------------------------------
// Demo scenes
// ---------------------------------------------------------------------------

func defineScenes() []scene {
	return []scene{
		{"primitives", 400, 300, buildPrimitives},
		{"gradients", 400, 300, buildGradients},
		{"shadows", 400, 300, buildShadows},
		{"text_showcase", 480, 200, buildTextShowcase},
		{"widget_mockup", 640, 480, buildWidgetMockup},
		{"complex_scene", 640, 480, buildComplexScene},
	}
}

// c is a helper for inline color literals.
func c(r, g, b, a uint8) primitives.Color {
	return primitives.Color{R: r, G: g, B: b, A: a}
}

// buildPrimitives showcases basic shapes: rectangles, rounded rects, lines.
func buildPrimitives(dl *displaylist.DisplayList) {
	// Light-grey background
	dl.AddFillRect(0, 0, 400, 300, c(240, 240, 245, 255))

	// Title
	dl.AddDrawText("Primitives", 16, 30, 20, c(40, 40, 40, 255), 0)

	// Solid rectangles
	dl.AddFillRect(20, 60, 80, 80, c(220, 50, 50, 255))
	dl.AddFillRect(120, 60, 80, 80, c(50, 180, 50, 255))
	dl.AddFillRect(220, 60, 80, 80, c(50, 100, 220, 255))

	// Rounded rectangles
	dl.AddFillRoundedRect(20, 160, 100, 60, 12, c(255, 140, 0, 255))
	dl.AddFillRoundedRect(140, 160, 100, 60, 20, c(160, 32, 240, 255))
	dl.AddFillRoundedRect(260, 160, 100, 60, 30, c(0, 180, 200, 255))

	// Lines
	dl.AddDrawLine(20, 250, 380, 250, 2, c(100, 100, 100, 255))
	dl.AddDrawLine(20, 260, 380, 290, 3, c(220, 50, 50, 200))
	dl.AddDrawLine(20, 290, 380, 260, 3, c(50, 100, 220, 200))
}

// buildGradients showcases linear and radial gradients.
func buildGradients(dl *displaylist.DisplayList) {
	dl.AddFillRect(0, 0, 400, 300, c(30, 30, 40, 255))

	dl.AddDrawText("Gradients", 16, 28, 20, c(240, 240, 240, 255), 0)

	// Linear gradients
	dl.AddLinearGradient(20, 50, 170, 80,
		20, 90, 190, 90,
		c(255, 60, 60, 255), c(60, 60, 255, 255))

	dl.AddLinearGradient(210, 50, 170, 80,
		210, 50, 380, 130,
		c(255, 200, 0, 255), c(0, 200, 100, 255))

	// Radial gradients
	dl.AddRadialGradient(20, 150, 170, 130,
		105, 215, 60,
		c(255, 255, 100, 255), c(200, 0, 255, 255))

	dl.AddRadialGradient(210, 150, 170, 130,
		295, 215, 70,
		c(100, 220, 255, 255), c(255, 50, 80, 255))
}

// buildShadows showcases box shadow effects.
func buildShadows(dl *displaylist.DisplayList) {
	dl.AddFillRect(0, 0, 400, 300, c(245, 245, 250, 255))

	dl.AddDrawText("Box Shadows", 16, 28, 20, c(40, 40, 40, 255), 0)

	// Cards with shadows of increasing blur
	offsets := []struct{ blur, spread int }{
		{4, 2},
		{10, 4},
		{20, 6},
	}
	for i, o := range offsets {
		x := 20 + i*130
		// Shadow first (renders underneath)
		dl.AddBoxShadow(x+4, 64, 100, 80, o.blur, o.spread, c(0, 0, 0, 80))
		// Card body
		dl.AddFillRoundedRect(x, 60, 100, 80, 8, c(255, 255, 255, 255))
	}

	// Coloured shadows
	dl.AddBoxShadow(24, 184, 100, 80, 12, 4, c(220, 50, 50, 100))
	dl.AddFillRoundedRect(20, 180, 100, 80, 8, c(255, 240, 240, 255))

	dl.AddBoxShadow(154, 184, 100, 80, 12, 4, c(50, 50, 220, 100))
	dl.AddFillRoundedRect(150, 180, 100, 80, 8, c(240, 240, 255, 255))

	dl.AddBoxShadow(284, 184, 100, 80, 12, 4, c(50, 180, 50, 100))
	dl.AddFillRoundedRect(280, 180, 100, 80, 8, c(240, 255, 240, 255))
}

// buildTextShowcase shows text rendering at various sizes.
func buildTextShowcase(dl *displaylist.DisplayList) {
	dl.AddFillRect(0, 0, 480, 200, c(255, 255, 255, 255))

	dl.AddDrawText("SDF Text Rendering", 16, 30, 22, c(40, 40, 40, 255), 0)
	dl.AddDrawText("The quick brown fox jumps over the lazy dog.", 16, 65, 14, c(80, 80, 80, 255), 0)
	dl.AddDrawText("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 16, 95, 16, c(60, 60, 60, 255), 0)
	dl.AddDrawText("0123456789 !@#$%^&*()", 16, 125, 16, c(60, 60, 60, 255), 0)
	dl.AddDrawText("Wain — a pure-Go Wayland/X11 toolkit", 16, 165, 18, c(50, 100, 220, 255), 0)
}

// buildWidgetMockup renders a mock application window with common UI elements.
func buildWidgetMockup(dl *displaylist.DisplayList) {
	w, h := 640, 480

	// Window background
	dl.AddFillRect(0, 0, w, h, c(235, 237, 240, 255))

	// Title bar
	dl.AddFillRect(0, 0, w, 44, c(50, 55, 70, 255))
	dl.AddDrawText("Wain Demo Application", 16, 28, 16, c(230, 230, 240, 255), 0)

	// Window control dots
	dl.AddFillRoundedRect(w-80, 14, 16, 16, 8, c(255, 95, 86, 255))
	dl.AddFillRoundedRect(w-56, 14, 16, 16, 8, c(255, 189, 46, 255))
	dl.AddFillRoundedRect(w-32, 14, 16, 16, 8, c(39, 201, 63, 255))

	// Sidebar
	dl.AddFillRect(0, 44, 180, h-44, c(45, 50, 65, 255))
	sideItems := []string{"Dashboard", "Settings", "Profile", "Messages", "Help"}
	for i, item := range sideItems {
		y := 70 + i*36
		if i == 0 {
			dl.AddFillRoundedRect(8, y-8, 164, 30, 6, c(70, 80, 110, 255))
		}
		dl.AddDrawText(item, 20, y+12, 14, c(200, 205, 220, 255), 0)
	}

	// Content area – heading
	dl.AddDrawText("Dashboard", 200, 76, 22, c(40, 44, 52, 255), 0)

	// Stat cards
	cardData := []struct {
		label string
		value string
		col   primitives.Color
	}{
		{"Users", "1,234", c(50, 120, 220, 255)},
		{"Revenue", "$42.8k", c(40, 180, 100, 255)},
		{"Tasks", "89", c(200, 80, 40, 255)},
	}
	for i, cd := range cardData {
		cx := 200 + i*145
		cy := 100
		dl.AddBoxShadow(cx+2, cy+2, 130, 90, 8, 2, c(0, 0, 0, 30))
		dl.AddFillRoundedRect(cx, cy, 130, 90, 8, c(255, 255, 255, 255))
		dl.AddFillRoundedRect(cx, cy, 130, 4, 2, cd.col)
		dl.AddDrawText(cd.value, cx+16, cy+42, 22, cd.col, 0)
		dl.AddDrawText(cd.label, cx+16, cy+70, 12, c(130, 130, 140, 255), 0)
	}

	// Chart placeholder
	chartY := 210
	dl.AddBoxShadow(202, chartY+2, 415, 180, 8, 2, c(0, 0, 0, 25))
	dl.AddFillRoundedRect(200, chartY, 415, 180, 8, c(255, 255, 255, 255))
	dl.AddDrawText("Activity", 216, chartY+28, 14, c(60, 60, 70, 255), 0)

	// Simple bar chart
	bars := []int{60, 90, 45, 110, 80, 130, 70, 100, 55, 120}
	for i, bh := range bars {
		bx := 220 + i*38
		by := chartY + 160 - bh
		dl.AddFillRoundedRect(bx, by, 24, bh, 4, c(80, 130, 230, 200))
	}

	// Bottom status bar
	dl.AddFillRect(0, h-28, w, 28, c(50, 55, 70, 255))
	dl.AddDrawText("Ready", 12, h-10, 11, c(160, 170, 180, 255), 0)
}

// buildComplexScene combines all primitives into a showcase composition.
func buildComplexScene(dl *displaylist.DisplayList) {
	w, h := 640, 480

	// Dark background gradient
	dl.AddLinearGradient(0, 0, w, h,
		0, 0, w, h,
		c(20, 20, 40, 255), c(40, 20, 60, 255))

	// Title
	dl.AddDrawText("Wain Rendering Engine", 24, 40, 24, c(255, 255, 255, 255), 0)
	dl.AddDrawText("Software rasteriser — all primitives", 24, 68, 13, c(180, 180, 200, 255), 0)

	// Gradient cards
	dl.AddLinearGradient(24, 90, 180, 120,
		24, 90, 204, 210,
		c(255, 100, 80, 255), c(255, 180, 50, 255))
	dl.AddLinearGradient(224, 90, 180, 120,
		224, 90, 404, 210,
		c(80, 200, 255, 255), c(100, 80, 255, 255))
	dl.AddRadialGradient(424, 90, 190, 120,
		519, 150, 80,
		c(255, 255, 120, 255), c(200, 50, 200, 255))

	// Overlapping translucent shapes
	dl.AddFillRoundedRect(40, 230, 200, 100, 16, c(255, 80, 80, 160))
	dl.AddFillRoundedRect(120, 260, 200, 100, 16, c(80, 200, 80, 160))
	dl.AddFillRoundedRect(200, 290, 200, 100, 16, c(80, 80, 255, 160))

	// Shadow panel
	dl.AddBoxShadow(424, 234, 190, 200, 16, 6, c(0, 0, 0, 100))
	dl.AddFillRoundedRect(420, 230, 190, 200, 12, c(255, 255, 255, 240))
	dl.AddDrawText("Panel", 440, 260, 16, c(40, 40, 60, 255), 0)
	dl.AddDrawLine(440, 272, 590, 272, 1, c(200, 200, 210, 255))
	dl.AddDrawText("Line 1", 440, 300, 12, c(100, 100, 120, 255), 0)
	dl.AddDrawText("Line 2", 440, 324, 12, c(100, 100, 120, 255), 0)
	dl.AddDrawText("Line 3", 440, 348, 12, c(100, 100, 120, 255), 0)
	dl.AddFillRoundedRect(440, 370, 100, 32, 6, c(80, 130, 230, 255))
	dl.AddDrawText("Action", 462, 392, 13, c(255, 255, 255, 255), 0)

	// Decorative lines
	for i := 0; i < 5; i++ {
		y := 440 + i*8
		dl.AddDrawLine(24, y, 400, y, 1, c(255, 255, 255, uint8(30+i*15)))
	}

	// Footer
	dl.AddDrawText("github.com/opd-ai/wain", 24, 468, 11, c(140, 140, 160, 255), 0)
}
