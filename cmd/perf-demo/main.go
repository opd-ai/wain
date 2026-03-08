// Package main implements a performance profiling demo.
//
// This demo renders typical UI workloads and measures frame times to validate
// the <2ms target for GPU rendering.
package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/render"
	"github.com/opd-ai/wain/internal/render/backend"
)

func main() {
	fmt.Println("=== GPU Performance Profiling Demo ===")
	fmt.Println()

	// Create GPU backend
	cfg := backend.DefaultConfig()
	cfg.Width = 1920
	cfg.Height = 1080
	cfg.DRMPath = "/dev/dri/renderD128"

	b, err := backend.New(cfg)
	if err != nil {
		log.Printf("GPU backend unavailable: %v", err)
		log.Println("This demo requires an Intel or AMD GPU")
		return
	}
	defer b.Destroy()

	fmt.Printf("GPU initialized: %dx%d\n", cfg.Width, cfg.Height)
	fmt.Println()

	// Create typical UI workload scenarios
	scenarios := []struct {
		name     string
		commands int
		rects    int
		texts    int
		shadows  int
	}{
		{"Light UI", 50, 40, 8, 2},
		{"Medium UI", 200, 150, 30, 10},
		{"Heavy UI", 500, 400, 50, 20},
	}

	for _, scenario := range scenarios {
		fmt.Printf("Scenario: %s\n", scenario.name)
		fmt.Printf("  Commands: %d (%d rects, %d texts, %d shadows)\n",
			scenario.commands, scenario.rects, scenario.texts, scenario.shadows)

		dl := createWorkload(scenario.rects, scenario.texts, scenario.shadows)

		// Warm up (first frame may allocate)
		b.Render(dl)

		// Reset stats and render test frames
		b.ResetFrameStats()
		for i := 0; i < 60; i++ {
			if err := b.Render(dl); err != nil {
				log.Fatalf("Render failed: %v", err)
			}
		}

		// Print results
		stats := b.GetFrameStats()
		fmt.Printf("  Frames rendered: %d\n", stats.TotalFrames)
		fmt.Printf("  Avg frame time: %.3f ms (CPU: %.3f ms, GPU: %.3f ms)\n",
			stats.AvgFrameTimeMs, stats.AvgCPUTimeMs, stats.AvgGPUTimeMs)
		fmt.Printf("  Recent avg: %.3f ms\n", stats.RecentAvgFrameTimeMs)
		fmt.Printf("  Min/Max: %.3f / %.3f ms\n", stats.MinFrameTimeMs, stats.MaxFrameTimeMs)

		// Check target
		if stats.RecentAvgFrameTimeMs < 2.0 {
			fmt.Printf("  ✓ Meets <2ms target\n")
		} else {
			fmt.Printf("  ✗ Exceeds <2ms target by %.3f ms\n", stats.RecentAvgFrameTimeMs-2.0)
		}
		fmt.Println()
	}

	// Memory statistics
	memStats := render.GetMemoryStats()
	fmt.Println("=== GPU Memory Statistics ===")
	fmt.Printf("Allocated buffers: %d\n", memStats.AllocatedBuffers)
	fmt.Printf("Allocated memory: %.2f MB\n", float64(memStats.AllocatedBytes)/1024/1024)
	fmt.Printf("Peak memory: %.2f MB\n", float64(memStats.PeakAllocatedBytes)/1024/1024)
	fmt.Printf("Total allocations: %d\n", memStats.TotalAllocations)
	fmt.Printf("Total deallocations: %d\n", memStats.TotalDeallocations)
	fmt.Println()

	fmt.Println("=== Profiling Complete ===")
}

// createWorkload creates a synthetic UI workload for profiling.
func createWorkload(rects, texts, shadows int) *displaylist.DisplayList {
	dl := displaylist.New()

	// Add filled rectangles
	for i := 0; i < rects; i++ {
		x := float32(i*10) % 1920
		y := float32((i*7)%1080)
		dl.FillRect(x, y, 50, 30, [4]uint8{255, 128, 64, 255})
	}

	// Add text runs (simulated with small rects for now)
	for i := 0; i < texts; i++ {
		x := float32((i * 100) % 1920)
		y := float32((i*50)%1080)
		dl.FillRect(x, y, 80, 12, [4]uint8{0, 0, 0, 255})
	}

	// Add box shadows (rounded rects with blur)
	for i := 0; i < shadows; i++ {
		x := float32((i * 150) % 1920)
		y := float32((i*100)%1080)
		dl.FillRoundedRect(x, y, 100, 80, 8, [4]uint8{200, 200, 200, 128})
	}

	return dl
}
