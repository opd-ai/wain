// Package demo provides shared utilities for demonstration binaries.
package demo

import (
	"log"

	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/ui/widgets"
)

// RenderDemoContent renders Phase 1 features to the buffer.
//
// This function showcases the software rasterizer capabilities:
//   - Clear with solid color
//   - Filled rectangles (title bar)
//   - UI widgets (button, text input)
//   - Rounded rectangles with anti-aliasing
//   - Alpha gradients
//   - Anti-aliased lines
//   - Color grids
func RenderDemoContent(buf *primitives.Buffer, btn *widgets.Button, input *widgets.TextInput) {
	// Feature 1: Clear with solid color
	buf.Clear(primitives.Color{R: 240, G: 240, B: 245, A: 255})

	// Feature 2: Filled rectangle (title bar)
	titleColor := primitives.Color{R: 60, G: 60, B: 80, A: 255}
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
		primitives.Color{R: 100, G: 200, B: 150, A: 255})

	// 5b. Alpha gradient (manual blending demonstration)
	for i := 0; i < 60; i++ {
		alpha := uint8(255 - (i * 4))
		buf.FillRect(80+i, showcaseY, 1, 40,
			primitives.Color{R: 200, G: 100, B: 150, A: alpha})
	}

	// 5c. Anti-aliased line (3px width)
	buf.DrawLine(160, showcaseY, 220, showcaseY+40, 3,
		primitives.Color{R: 150, G: 150, B: 200, A: 255})

	// 5d. Grid of colored rectangles
	for i := 0; i < 4; i++ {
		for j := 0; j < 2; j++ {
			x := 240 + i*20
			y := showcaseY + j*20
			c := primitives.Color{
				R: uint8(50 + i*40),
				G: uint8(50 + j*80),
				B: uint8(200 - i*30),
				A: 255,
			}
			buf.FillRect(x, y, 15, 15, c)
		}
	}
}
