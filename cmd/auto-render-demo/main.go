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

	// Configure automatic backend selection
	cfg := backend.DefaultAutoConfig()
	cfg.Width = 800
	cfg.Height = 600
	cfg.Verbose = true // Log backend selection

	// Allow forcing software renderer via environment variable
	if os.Getenv("FORCE_SOFTWARE") == "1" {
		cfg.ForceSoftware = true
		log.Println("Environment: forcing software renderer")
	}

	// Create renderer with auto-detection
	renderer, backendType, err := backend.NewRenderer(cfg)
	if err != nil {
		log.Fatalf("Failed to create renderer: %v", err)
	}
	defer renderer.Destroy()

	log.Printf("Using backend: %s", backendType)

	// Create a simple test scene
	dl := displaylist.New()

	// Background
	dl.AddFillRect(0, 0, 800, 600, primitives.Color{R: 240, G: 240, B: 240, A: 255})

	// Red rectangle
	dl.AddFillRect(50, 50, 200, 150, primitives.Color{R: 255, G: 0, B: 0, A: 255})

	// Green rounded rectangle
	dl.AddFillRoundedRect(300, 50, 200, 150, 20, primitives.Color{R: 0, G: 255, B: 0, A: 255})

	// Blue rectangle
	dl.AddFillRect(550, 50, 200, 150, primitives.Color{R: 0, G: 0, B: 255, A: 255})

	// Linear gradient
	dl.AddLinearGradient(
		50, 250, 700, 100,
		50, 250, 750, 250,
		primitives.Color{R: 255, G: 0, B: 255, A: 255},
		primitives.Color{R: 255, G: 255, B: 0, A: 255},
	)

	// Radial gradient
	dl.AddRadialGradient(
		50, 400, 700, 150,
		400, 475, 100,
		primitives.Color{R: 0, G: 255, B: 255, A: 255},
		primitives.Color{R: 255, G: 128, B: 0, A: 255},
	)

	// Render the scene
	if err := renderer.Render(dl); err != nil {
		log.Fatalf("Render failed: %v", err)
	}

	log.Printf("Rendered %d commands successfully", dl.Len())

	// Get dimensions
	w, h := renderer.Dimensions()
	log.Printf("Render target: %dx%d", w, h)

	// For software backend, we can access the buffer directly
	if backendType == backend.BackendSoftware {
		softBackend := renderer.(*backend.SoftwareBackend)
		buf := softBackend.Buffer()
		if buf != nil {
			log.Printf("Software buffer: %d bytes (%dx%d)", len(buf.Pixels), buf.Width, buf.Height)
		}
	}

	// For GPU backends, we would export via Present()
	if backendType == backend.BackendIntelGPU || backendType == backend.BackendAMDGPU {
		fd, err := renderer.Present()
		if err != nil {
			log.Printf("Warning: Present() failed: %v", err)
		} else {
			log.Printf("DMA-BUF fd: %d (not saved, would be sent to compositor)", fd)
			// In a real application, this fd would be sent to Wayland/X11
			// For this demo, we just close it
			if fd >= 0 {
				// syscall.Close(fd) - but we don't want to import syscall for demo
				log.Printf("DMA-BUF export successful")
			}
		}
	}

	fmt.Println("\n✓ Auto-detection demo complete")
	fmt.Printf("  Backend: %s\n", backendType)
	fmt.Printf("  Commands: %d\n", dl.Len())
	fmt.Printf("  Resolution: %dx%d\n", w, h)
}
