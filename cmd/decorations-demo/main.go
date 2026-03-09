// decorations-demo demonstrates client-side window decorations.
// Showcases title bar, window control buttons, and resize handles.
package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/ui/decorations"
)

func main() {
	demo.CheckHelpFlag("decorations-demo", "Client-side window decorations demonstration", []string{
		demo.FormatExample("decorations-demo", "Run window decorations demo"),
		demo.FormatExample("decorations-demo --help", "Show this help message"),
	})

	fmt.Println("======================================")
	fmt.Println("Window Decorations Demo - Phase 8.3")
	fmt.Println("======================================")
	fmt.Println()

	// Create window with full decorations
	width := 640
	height := 480

	// Create window frame with full decorations
	windowFrame := decorations.NewWindowFrame("Window Decorations Demo", width, height)

	// Create text atlas for title rendering
	atlas, err := text.NewAtlas()
	if err != nil {
		log.Fatalf("Failed to create text atlas: %v", err)
	}
	windowFrame.SetAtlas(atlas)

	// Create buffer for rendering
	buf, err := primitives.NewBuffer(width, height)
	if err != nil {
		log.Fatalf("Failed to create buffer: %v", err)
	}

	// Render full window frame
	renderFrame(buf, windowFrame, width, height)

	// Demonstrate display list rendering
	dl := renderFrameWithDisplayList(windowFrame, width, height)

	// Display results
	frameW, frameH := windowFrame.Bounds()
	contentW, contentH := windowFrame.ContentBounds()
	offsetX, offsetY := windowFrame.ContentOffset()

	fmt.Println("Window Frame Dimensions:")
	fmt.Printf("  Total size: %dx%d pixels (including decorations)\n", frameW, frameH)
	fmt.Printf("  Content area: %dx%d pixels (usable area)\n", contentW, contentH)
	fmt.Printf("  Content offset: (%d, %d) (top-left of content area)\n", offsetX, offsetY)
	fmt.Println()

	fmt.Println("Decoration Components:")
	fmt.Println("  ✓ Title bar with window title")
	fmt.Println("  ✓ Minimize button (hover/press states)")
	fmt.Println("  ✓ Maximize button (hover/press states)")
	fmt.Println("  ✓ Close button (hover/press states)")
	fmt.Println("  ✓ 8 resize handles (4 edges + 4 corners)")
	fmt.Println("  ✓ Window drag area in title bar")
	fmt.Println()

	demo.PrintFeatureList("Protocol Support:", []string{
		"XDG decoration protocol (zxdg_decoration_manager_v1)",
		"Server-side decoration negotiation",
		"Client-side fallback (automatic)",
	})
	fmt.Println()

	// Demonstrate hit testing
	fmt.Println("Hit Testing Examples:")
	demonstrateHitTesting(windowFrame)
	fmt.Println()

	fmt.Printf("Display list: %d commands generated\n", len(dl.Commands()))
	fmt.Println("\n✓ Demo complete!")
}

func renderFrame(buf *primitives.Buffer, windowFrame *decorations.WindowFrame, width, height int) {
	// Clear background
	bgColor := primitives.Color{R: 255, G: 255, B: 255, A: 255}
	buf.FillRect(0, 0, width, height, bgColor)

	// Render window frame (title bar + resize handles)
	if err := windowFrame.Draw(buf, 0, 0); err != nil {
		log.Printf("Warning: Failed to draw window frame: %v", err)
	}

	// Render content area
	offsetX, offsetY := windowFrame.ContentOffset()
	contentW, contentH := windowFrame.ContentBounds()

	contentColor := primitives.Color{R: 250, G: 250, B: 250, A: 255}
	buf.FillRect(offsetX, offsetY, contentW, contentH, contentColor)

	// Draw some example content
	exampleColor := primitives.Color{R: 100, G: 150, B: 200, A: 255}
	buf.FillRect(offsetX+50, offsetY+50, 200, 100, exampleColor)

	// Draw resize handle demonstration (simulate hover on corner)
	windowFrame.HandlePointerMotion(width-4, height-4)
}

func renderFrameWithDisplayList(windowFrame *decorations.WindowFrame, width, height int) *displaylist.DisplayList {
	dl := displaylist.New()

	// Background
	bgColor := primitives.Color{R: 255, G: 255, B: 255, A: 255}
	dl.AddFillRect(0, 0, width, height, bgColor)

	// Window frame (title bar + resize handles)
	windowFrame.RenderToDisplayList(dl, 0, 0)

	// Content area
	offsetX, offsetY := windowFrame.ContentOffset()
	contentW, contentH := windowFrame.ContentBounds()

	contentColor := primitives.Color{R: 250, G: 250, B: 250, A: 255}
	dl.AddFillRect(offsetX, offsetY, contentW, contentH, contentColor)

	// Example content
	exampleColor := primitives.Color{R: 100, G: 150, B: 200, A: 255}
	dl.AddFillRect(offsetX+50, offsetY+50, 200, 100, exampleColor)

	return dl
}

func demonstrateHitTesting(windowFrame *decorations.WindowFrame) {
	// Test resize handle detection
	edge := windowFrame.HitTestResize(5, 5)
	fmt.Printf("  Pointer at (5, 5): %s resize handle\n", edge)

	edge = windowFrame.HitTestResize(635, 475)
	fmt.Printf("  Pointer at (635, 475): %s resize handle\n", edge)

	// Test title bar button detection
	button := windowFrame.HitTestTitleBarButton(600, 20)
	if button != nil {
		fmt.Printf("  Pointer at (600, 20): over window button\n")
	} else {
		fmt.Printf("  Pointer at (600, 20): not over button\n")
	}

	// Test drag area
	isDrag := windowFrame.IsTitleBarDragArea(320, 20)
	fmt.Printf("  Pointer at (320, 20): drag area = %v\n", isDrag)
}
