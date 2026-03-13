package backend

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/render"
)

// AutoConfig contains configuration for automatic backend selection.
type AutoConfig struct {
	// DRMPath is the path to the DRM device for GPU detection.
	// Default: "/dev/dri/renderD128"
	DRMPath string

	// Width is the render target width in pixels
	Width int

	// Height is the render target height in pixels
	Height int

	// VertexBufferSize is the size of the dynamic vertex buffer for GPU backends
	VertexBufferSize int

	// Atlas is the font atlas for text rendering (optional, used by software fallback)
	Atlas *text.Atlas

	// ForceSoftware forces software rendering even if GPU is available.
	// Useful for testing and debugging.
	ForceSoftware bool

	// Verbose enables logging of backend selection decisions.
	Verbose bool
}

// DefaultAutoConfig returns a default configuration for automatic backend selection.
func DefaultAutoConfig() AutoConfig {
	return AutoConfig{
		DRMPath:          "/dev/dri/renderD128",
		Width:            800,
		Height:           600,
		VertexBufferSize: 1024 * 1024, // 1MB
		ForceSoftware:    false,
		Verbose:          false,
	}
}

// NewRenderer creates a rendering backend with automatic GPU detection and fallback.
//
// Selection logic (Phase 7.1):
//  1. If ForceSoftware is true, use software renderer
//  2. Attempt GPU detection at DRMPath
//  3. If Intel GPU detected → use Intel backend
//  4. If AMD GPU detected → use AMD backend
//  5. If detection fails or GPU unsupported → fall back to software renderer
//
// Returns the selected renderer and its type.
func NewRenderer(cfg AutoConfig) (Renderer, BackendType, error) {
	// Validate dimensions
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, BackendUnknown, fmt.Errorf("backend: invalid dimensions %dx%d", cfg.Width, cfg.Height)
	}

	// Force software rendering if requested
	if cfg.ForceSoftware {
		return newSoftwareFallback(cfg, "forced by configuration")
	}

	// Attempt GPU detection
	gen := render.DetectGPU(cfg.DRMPath)

	// Check if GPU was detected
	if gen == render.GpuUnknown {
		return newSoftwareFallback(cfg, "no GPU detected")
	}

	// Determine backend type based on GPU generation
	backendType := gpuGenerationToBackendType(gen)
	if backendType == BackendUnknown {
		return newSoftwareFallback(cfg, fmt.Sprintf("unsupported GPU: %s", gen))
	}

	// Attempt to create GPU backend
	gpuCfg := Config{
		DRMPath:          cfg.DRMPath,
		Width:            cfg.Width,
		Height:           cfg.Height,
		VertexBufferSize: cfg.VertexBufferSize,
		FontAtlas:        cfg.Atlas,
	}

	backend, err := New(gpuCfg)
	if err != nil {
		// GPU backend creation failed - fall back to software
		return newSoftwareFallback(cfg, fmt.Sprintf("GPU init failed: %v", err))
	}

	if cfg.Verbose {
		log.Printf("backend: using %s (%s)", backendType, gen)
	}

	return backend, backendType, nil
}

// newSoftwareFallback creates a software rendering backend and logs the reason.
func newSoftwareFallback(cfg AutoConfig, reason string) (Renderer, BackendType, error) {
	if cfg.Verbose {
		log.Printf("backend: falling back to software rendering (%s)", reason)
	}

	softCfg := SoftwareConfig{
		Width:  cfg.Width,
		Height: cfg.Height,
		Atlas:  cfg.Atlas,
	}

	backend, err := NewSoftwareBackend(softCfg)
	if err != nil {
		return nil, BackendUnknown, fmt.Errorf("backend: software fallback failed: %w", err)
	}

	return backend, BackendSoftware, nil
}

// gpuGenerationToBackendType maps a detected GPU generation to backend type.
func gpuGenerationToBackendType(gen render.GpuGeneration) BackendType {
	switch gen {
	case render.GpuGen9, render.GpuGen11, render.GpuGen12, render.GpuXe:
		return BackendIntelGPU
	case render.GpuAmdRdna1, render.GpuAmdRdna2, render.GpuAmdRdna3:
		return BackendAMDGPU
	default:
		return BackendUnknown
	}
}

// IsIntelGPU returns true if the GPU generation is Intel.
func IsIntelGPU(gen render.GpuGeneration) bool {
	return gpuGenerationToBackendType(gen) == BackendIntelGPU
}

// IsAMDGPU returns true if the GPU generation is AMD.
func IsAMDGPU(gen render.GpuGeneration) bool {
	return gpuGenerationToBackendType(gen) == BackendAMDGPU
}
