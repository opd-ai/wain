// Command auto-render-demo demonstrates automatic backend selection with fallback.
//
// This demo implements Phase 7.1 of the roadmap: AUTO-DETECTION & FALLBACK.
// It automatically detects available GPU hardware and selects the appropriate
// rendering backend:
//   - Intel GPU (Gen9-Xe) → Intel backend
//   - AMD GPU (RDNA1-3) → AMD backend
//   - No GPU or init failure → Software renderer fallback
//
// All three backends produce identical visual output.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/render/backend"
)

func main() {
	demo.CheckHelpFlag("auto-render-demo", "Automatic backend selection with GPU fallback", []string{
		demo.FormatExample("auto-render-demo", "Auto-detect GPU and render"),
		demo.FormatExample("FORCE_SOFTWARE=1 auto-render-demo", "Force software renderer"),
		demo.FormatExample("auto-render-demo --help", "Show this help message"),
	})

	log.SetFlags(0)

	cfg := backend.DefaultAutoConfig()
	cfg.Width = 800
	cfg.Height = 600
	cfg.Verbose = true

	if os.Getenv("FORCE_SOFTWARE") == "1" {
		cfg.ForceSoftware = true
		log.Println("Environment: forcing software renderer")
	}

	renderer, backendType, err := backend.NewRenderer(cfg)
	if err != nil {
		log.Fatalf("Failed to create renderer: %v", err)
	}
	defer func() { _ = renderer.Destroy() }()

	log.Printf("Using backend: %s", backendType)

	dl := displaylist.New()
	buildDemoScene(dl)

	if err := renderer.Render(dl); err != nil {
		log.Fatalf("Render failed: %v", err)
	}

	log.Printf("Rendered %d commands successfully", dl.Len())

	w, h := renderer.Dimensions()
	log.Printf("Render target: %dx%d", w, h)

	reportRenderResults(renderer, backendType, dl, w, h)
}

// buildDemoScene populates dl with a representative set of UI draw commands.
func buildDemoScene(dl *displaylist.DisplayList) {
	dl.AddFillRect(0, 0, 800, 600, primitives.Color{R: 240, G: 240, B: 240, A: 255})
	dl.AddFillRect(50, 50, 200, 150, primitives.Color{R: 255, G: 0, B: 0, A: 255})
	dl.AddFillRoundedRect(300, 50, 200, 150, 20, primitives.Color{R: 0, G: 255, B: 0, A: 255})
	dl.AddFillRect(550, 50, 200, 150, primitives.Color{R: 0, G: 0, B: 255, A: 255})
	dl.AddLinearGradient(
		50, 250, 700, 100,
		50, 250, 750, 250,
		primitives.Color{R: 255, G: 0, B: 255, A: 255},
		primitives.Color{R: 255, G: 255, B: 0, A: 255},
	)
	dl.AddRadialGradient(
		50, 400, 700, 150,
		400, 475, 100,
		primitives.Color{R: 0, G: 255, B: 255, A: 255},
		primitives.Color{R: 255, G: 128, B: 0, A: 255},
	)
}

// reportRenderResults logs backend-specific post-render diagnostics and prints
// the summary to stdout.
func reportRenderResults(renderer backend.Renderer, backendType backend.BackendType, dl *displaylist.DisplayList, w, h int) {
	if backendType == backend.BackendSoftware {
		logSoftwareBuffer(renderer)
	}

	if backendType == backend.BackendIntelGPU || backendType == backend.BackendAMDGPU {
		logGPUPresent(renderer)
	}

	fmt.Println("\n✓ Auto-detection demo complete")
	fmt.Printf("  Backend: %s\n", backendType)
	fmt.Printf("  Commands: %d\n", dl.Len())
	fmt.Printf("  Resolution: %dx%d\n", w, h)
}

// logSoftwareBuffer logs the pixel buffer size for a software backend.
func logSoftwareBuffer(renderer backend.Renderer) {
	softBackend := renderer.(*backend.SoftwareBackend)
	buf := softBackend.Buffer()
	if buf != nil {
		log.Printf("Software buffer: %d bytes (%dx%d)", len(buf.Pixels), buf.Width, buf.Height)
	}
}

// logGPUPresent exports the GPU render target via Present and logs the result.
func logGPUPresent(renderer backend.Renderer) {
	fd, err := renderer.Present()
	if err != nil {
		log.Printf("Warning: Present() failed: %v", err)
		return
	}
	log.Printf("DMA-BUF fd: %d (not saved, would be sent to compositor)", fd)
	if fd >= 0 {
		log.Printf("DMA-BUF export successful")
	}
}
