// Command demo demonstrates Phase 1 features of the wain UI toolkit.
//
// This binary showcases:
//   - X11 protocol client
//   - Software rasterizer (rectangles, rounded rects, lines)
//   - UI widgets (button, text input)
//   - Complete rendering pipeline
//
// Usage:
//
//	./bin/demo
package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain/internal/demo"
	x11client "github.com/opd-ai/wain/internal/x11/client"
	"github.com/opd-ai/wain/internal/x11/wire"
)

const (
	windowWidth  = 400
	windowHeight = 300
)

func main() {
	fmt.Println("=================================")
	fmt.Println("wain Phase 1 Demo - X11 Backend")
	fmt.Println("=================================")
	fmt.Println()

	if err := runDemo(); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}

	fmt.Println("\n✓ Demo completed successfully!")
}

// runDemo demonstrates the Phase 1 feature stack.
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
	btn, input := demo.StandardWidgets()
	fmt.Println("      ✓ Created Button widget (120x40)")
	fmt.Println("      ✓ Created TextInput widget (200x30)")

	// Step 5: Render content with software rasterizer
	fmt.Println("\n[5/6] Rendering content to framebuffer...")
	renderBuffer, err := demo.CreateDemoBuffer(windowWidth, windowHeight)
	if err != nil {
		return err
	}
	demo.RenderDemoContent(renderBuffer, btn, input)
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
	demo.PrintRenderingFeatures()
	fmt.Println()
	demo.PrintUIFeatures()
	fmt.Println()

	demo.PrintBufferStats(windowWidth, windowHeight, renderBuffer)

	return nil
}
