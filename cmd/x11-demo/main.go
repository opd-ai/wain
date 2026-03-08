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

	"github.com/opd-ai/wain/internal/demo"
)

const (
	windowWidth  = 400
	windowHeight = 300
)

func main() {
	demo.CheckHelpFlag("x11-demo", "X11 protocol client with software rasterizer", []string{
		demo.FormatExample("x11-demo", "Run demo on X11 server"),
		demo.FormatExample("x11-demo --help", "Show this help message"),
	})

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
	return demo.RunX11Demo(windowWidth, windowHeight)
}
