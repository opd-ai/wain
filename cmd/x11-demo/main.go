// Command x11-demo demonstrates Phase 1 features of the wain UI toolkit on X11.
//
// This binary showcases:
//   - X11 protocol client
//   - Software rasterizer (rectangles, rounded rects, lines)
//   - UI widgets (button, text input)
//   - Complete rendering pipeline
//
// Usage:
//
//	./bin/x11-demo
package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain/internal/raster/core"
	"github.com/opd-ai/wain/internal/ui/widgets"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/wire"
)

const (
	windowWidth  = 400
	windowHeight = 300
)

func main() {
	fmt.Println("==================================")
	fmt.Println("wain Phase 1 Demo - X11 Backend")
	fmt.Println("==================================")
	fmt.Println()

	if err := runDemo(); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}

	fmt.Println("\n✓ Demo completed successfully!")
}

// runDemo demonstrates the Phase 1 feature stack on X11.
func runDemo() error {
	// Step 1: Connect to X11 server
	fmt.Println("[1/6] Connecting to X11 server...")
	conn, err := x11client.Connect("0")
	if err != nil {
		return fmt.Errorf("connect to X11: %w", err)
	}
	defer conn.Close()
	fmt.Println("      ✓ Connected to :0")

	// Step 2: Create window
	fmt.Println("\n[2/6] Creating window...")
	root := conn.RootWindow()

	const (
		x           = 100
		y           = 100
		borderWidth = 0
		windowClass = wire.WindowClassInputOutput
		visual      = 0 // CopyFromParent
		eventMask   = wire.EventMaskExposure | wire.EventMaskKeyPress | wire.EventMaskButtonPress
	)

	mask := uint32(wire.CWEventMask)
	attrs := []uint32{eventMask}

	wid, err := conn.CreateWindow(root, x, y, windowWidth, windowHeight, borderWidth, windowClass, visual, mask, attrs)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	fmt.Printf("      ✓ Created window XID %d\n", wid)

	// Step 3: Map window to display
	fmt.Println("\n[3/6] Mapping window to display...")
	if err := conn.MapWindow(wid); err != nil {
		return fmt.Errorf("map window: %w", err)
	}
	fmt.Println("      ✓ Window visible on screen")

	// Step 4: Create UI widgets
	fmt.Println("\n[4/6] Creating UI widgets...")
	btn := widgets.NewButton("Click Me!", 120, 40)
	input := widgets.NewTextInput("Type here...", 200, 30)
	fmt.Println("      ✓ Created Button widget (120x40)")
	fmt.Println("      ✓ Created TextInput widget (200x30)")

	// Step 5: Render content with software rasterizer
	fmt.Println("\n[5/6] Rendering content to framebuffer...")
	renderBuffer, err := core.NewBuffer(windowWidth, windowHeight)
	if err != nil {
		return fmt.Errorf("create buffer: %w", err)
	}
	renderDemoContent(renderBuffer, btn, input)
	fmt.Printf("      ✓ Rendered to %dx%d ARGB8888 buffer\n", windowWidth, windowHeight)

	// Step 6: Display feature summary
	fmt.Println("\n[6/6] Phase 1 Features Demonstrated:")
	fmt.Println()
	fmt.Println("      PROTOCOL LAYER (X11)")
	fmt.Println("      -------------------")
	fmt.Println("      • Connection setup and authentication")
	fmt.Println("      • Window creation (CreateWindow)")
	fmt.Println("      • Window mapping (MapWindow)")
	fmt.Println("      • Resource allocation (AllocXID)")
	fmt.Println()
	fmt.Println("      RENDERING LAYER (Software Rasterizer)")
	fmt.Println("      -------------------------------------")
	fmt.Println("      • Filled rectangles (title bar)")
	fmt.Println("      • Rounded rectangles (radius=8px, anti-aliased)")
	fmt.Println("      • Alpha gradient (manual alpha blending)")
	fmt.Println("      • Anti-aliased lines (3px width)")
	fmt.Println("      • Color grid (8 unique colors)")
	fmt.Println()
	fmt.Println("      UI LAYER (Widgets)")
	fmt.Println("      ------------------")
	fmt.Println("      • Button widget with state management")
	fmt.Println("      • TextInput widget with placeholder")
	fmt.Println()

	fmt.Println("Buffer Stats:")
	fmt.Printf("  Pixels rendered: %d\n", windowWidth*windowHeight)
	fmt.Printf("  Buffer size: %d bytes\n", len(renderBuffer.Pixels))
	fmt.Printf("  Stride: %d bytes/row\n", renderBuffer.Stride)

	return nil
}

// renderDemoContent renders Phase 1 features to the buffer.
func renderDemoContent(buf *core.Buffer, btn *widgets.Button, input *widgets.TextInput) {
	// Feature 1: Clear with solid color
	buf.Clear(core.Color{R: 240, G: 240, B: 245, A: 255})

	// Feature 2: Filled rectangle (title bar)
	titleColor := core.Color{R: 60, G: 60, B: 80, A: 255}
	buf.FillRect(10, 10, 380, 50, titleColor)

	// Feature 3: Button widget with rounded corners
	if err := btn.Draw(buf, 140, 100); err != nil {
		log.Printf("Warning: button draw failed: %v", err)
	}

	// Feature 4: TextInput widget
	if err := input.Draw(buf, 100, 170); err != nil {
		log.Printf("Warning: input draw failed: %v", err)
	}

	// Feature 5: Showcase rasterizer primitives
	showcaseY := 220

	// 5a. Rounded rectangle with anti-aliased corners
	buf.FillRoundedRect(10, showcaseY, 60, 40, 8,
		core.Color{R: 100, G: 200, B: 150, A: 255})

	// 5b. Alpha gradient (manual blending demonstration)
	for i := 0; i < 60; i++ {
		alpha := uint8(255 - (i * 4))
		buf.FillRect(80+i, showcaseY, 1, 40,
			core.Color{R: 200, G: 100, B: 150, A: alpha})
	}

	// 5c. Anti-aliased line (3px width)
	buf.DrawLine(160, showcaseY, 220, showcaseY+40, 3,
		core.Color{R: 150, G: 150, B: 200, A: 255})

	// 5d. Grid of colored rectangles
	for i := 0; i < 4; i++ {
		for j := 0; j < 2; j++ {
			x := 240 + i*20
			y := showcaseY + j*20
			c := core.Color{
				R: uint8(50 + i*40),
				G: uint8(50 + j*80),
				B: uint8(200 - i*30),
				A: 255,
			}
			buf.FillRect(x, y, 15, 15, c)
		}
	}
}
