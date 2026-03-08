// Package backend provides rendering backends with automatic GPU detection and fallback.
//
// This package implements Phase 7.1: AUTO-DETECTION & FALLBACK.
// It provides a unified interface for GPU (Intel/AMD) and software rendering backends,
// with automatic selection based on available hardware.
package backend

import (
	"github.com/opd-ai/wain/internal/raster/displaylist"
)

// Renderer is the common interface for all rendering backends.
//
// Implementations include:
//   - GPUBackend (Intel or AMD GPU rendering)
//   - SoftwareBackend (CPU rasterization)
type Renderer interface {
	// Render renders a display list to the target surface.
	Render(dl *displaylist.DisplayList) error

	// RenderWithDamage renders only the damaged regions for incremental updates.
	// If damage is nil or empty, renders the full frame.
	RenderWithDamage(dl *displaylist.DisplayList, damage []displaylist.Rect) error

	// Present exports the render target for display.
	// Returns a file descriptor (DMA-BUF for GPU, or -1 for software).
	Present() (int, error)

	// Dimensions returns the render target width and height in pixels.
	Dimensions() (width, height int)

	// Destroy frees all resources.
	Destroy() error
}

// BackendType identifies the active rendering backend.
type BackendType int

const (
	// BackendUnknown indicates no backend is active.
	BackendUnknown BackendType = iota

	// BackendIntelGPU indicates Intel GPU rendering (Gen9-Gen12/Xe).
	BackendIntelGPU

	// BackendAMDGPU indicates AMD GPU rendering (RDNA1/RDNA2/RDNA3).
	BackendAMDGPU

	// BackendSoftware indicates CPU-based software rendering.
	BackendSoftware
)

// String returns a human-readable name for the backend type.
func (bt BackendType) String() string {
	switch bt {
	case BackendIntelGPU:
		return "Intel GPU"
	case BackendAMDGPU:
		return "AMD GPU"
	case BackendSoftware:
		return "Software"
	default:
		return "Unknown"
	}
}
