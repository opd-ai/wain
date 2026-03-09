package demo

import (
	"fmt"

	"github.com/opd-ai/wain/internal/raster/primitives"
)

// PrintBufferStats prints statistics about a render buffer.
func PrintBufferStats(width, height int, buf *primitives.Buffer) {
	fmt.Println("Buffer Stats:")
	fmt.Printf("  Pixels rendered: %d\n", width*height)
	fmt.Printf("  Buffer size: %d bytes\n", len(buf.Pixels))
	fmt.Printf("  Stride: %d bytes/row\n", buf.Stride)
}

// PrintRenderingFeatures prints the standard rendering layer feature list.
func PrintRenderingFeatures() {
	fmt.Println("      RENDERING LAYER (Software Rasterizer)")
	fmt.Println("      -------------------------------------")
	fmt.Println("      • Filled rectangles (title bar)")
	fmt.Println("      • Rounded rectangles (radius=8px, anti-aliased)")
	fmt.Println("      • Alpha gradient (manual alpha blending)")
	fmt.Println("      • Anti-aliased lines (3px width)")
	fmt.Println("      • Color grid (8 unique colors)")
}

// PrintUIFeatures prints the standard UI layer feature list.
func PrintUIFeatures() {
	fmt.Println("      UI LAYER (Widgets)")
	fmt.Println("      ------------------")
	fmt.Println("      • Button widget with state management")
	fmt.Println("      • TextInput widget with placeholder")
}
