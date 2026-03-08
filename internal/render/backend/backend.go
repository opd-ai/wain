// Package backend provides a GPU rendering backend that consumes display lists.
//
// This package implements Phase 5.1 of the roadmap: Display List Consumer.
// It takes a display list of draw commands and renders them using GPU command
// submission, leveraging the shader pipeline and batch infrastructure from Phase 4.
package backend

import (
	"errors"
	"fmt"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/render"
)

var (
	// ErrNilDisplayList is returned when a nil display list is provided.
	ErrNilDisplayList = errors.New("backend: nil display list")

	// ErrNoGPU is returned when GPU initialization fails.
	ErrNoGPU = errors.New("backend: GPU not available")
)

// GPUBackend manages GPU resources for rendering display lists.
type GPUBackend struct {
	drmPath   string
	allocator *render.Allocator
	ctx       *render.GpuContext

	// Vertex buffer for dynamic geometry
	vertexBuffer *render.BufferHandle
	vertexHandle uint32

	// Render target
	renderTarget *render.BufferHandle
	targetHandle uint32
	width        int
	height       int
}

// Config contains configuration for the GPU backend.
type Config struct {
	// DRMPath is the path to the DRM device (e.g., "/dev/dri/renderD128")
	DRMPath string

	// Width is the render target width in pixels
	Width int

	// Height is the render target height in pixels
	Height int

	// VertexBufferSize is the size of the dynamic vertex buffer in bytes
	VertexBufferSize int
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		DRMPath:          "/dev/dri/renderD128",
		Width:            800,
		Height:           600,
		VertexBufferSize: 1024 * 1024, // 1MB vertex buffer
	}
}

// New creates a new GPU backend.
func New(cfg Config) (*GPUBackend, error) {
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, fmt.Errorf("backend: invalid dimensions %dx%d", cfg.Width, cfg.Height)
	}

	if cfg.VertexBufferSize <= 0 {
		return nil, fmt.Errorf("backend: invalid vertex buffer size %d", cfg.VertexBufferSize)
	}

	// Detect GPU
	gen := render.DetectGPU(cfg.DRMPath)
	if gen < 0 {
		return nil, ErrNoGPU
	}

	// Create allocator
	allocator, err := render.NewAllocator(cfg.DRMPath)
	if err != nil {
		return nil, fmt.Errorf("backend: allocator creation failed: %w", err)
	}

	// Create GPU context
	ctx, err := render.CreateContext(cfg.DRMPath)
	if err != nil {
		allocator.Close()
		return nil, fmt.Errorf("backend: context creation failed: %w", err)
	}

	// Allocate vertex buffer (1D buffer, so height=1)
	// Width represents buffer size in pixels (each "pixel" is 4 bytes for alignment)
	vertexWidthPx := uint32(cfg.VertexBufferSize / 4)
	vertexBuffer, err := allocator.Allocate(vertexWidthPx, 1, 32, render.TilingNone)
	if err != nil {
		allocator.Close()
		return nil, fmt.Errorf("backend: vertex buffer allocation failed: %w", err)
	}

	// Allocate render target with Y-tiling for better cache performance
	renderTarget, err := allocator.Allocate(uint32(cfg.Width), uint32(cfg.Height), 32, render.TilingY)
	if err != nil {
		vertexBuffer.Destroy()
		allocator.Close()
		return nil, fmt.Errorf("backend: render target allocation failed: %w", err)
	}

	backend := &GPUBackend{
		drmPath:      cfg.DRMPath,
		allocator:    allocator,
		ctx:          ctx,
		vertexBuffer: vertexBuffer,
		vertexHandle: vertexBuffer.GemHandle(),
		renderTarget: renderTarget,
		targetHandle: renderTarget.GemHandle(),
		width:        cfg.Width,
		height:       cfg.Height,
	}

	return backend, nil
}

// Destroy frees all GPU resources.
func (b *GPUBackend) Destroy() error {
	if b.vertexBuffer != nil {
		b.vertexBuffer.Destroy()
		b.vertexBuffer = nil
	}
	if b.renderTarget != nil {
		b.renderTarget.Destroy()
		b.renderTarget = nil
	}
	if b.allocator != nil {
		b.allocator.Close()
		b.allocator = nil
	}
	return nil
}

// Render renders a display list to the GPU render target.
func (b *GPUBackend) Render(dl *displaylist.DisplayList) error {
	if dl == nil {
		return ErrNilDisplayList
	}

	if dl.Len() == 0 {
		return nil // Nothing to render
	}

	// Phase 5.1 implementation plan:
	// 1. Sort/batch commands by pipeline state
	// 2. Pack vertices into vertex buffer
	// 3. Build batch buffer with GPU commands
	// 4. Submit and wait

	// Sort and batch commands by type to minimize state changes
	batches := batchCommands(dl.Commands())

	// Pack vertices for all batches
	vertexData, err := b.packVertices(batches)
	if err != nil {
		return fmt.Errorf("backend: vertex packing failed: %w", err)
	}

	// Build and submit GPU batch buffer
	if err := b.submitBatches(batches, vertexData); err != nil {
		return fmt.Errorf("backend: batch submission failed: %w", err)
	}

	return nil
}

// RenderTarget returns the render target buffer for display.
func (b *GPUBackend) RenderTarget() *render.BufferHandle {
	return b.renderTarget
}

// Dimensions returns the render target dimensions.
func (b *GPUBackend) Dimensions() (width, height int) {
	return b.width, b.height
}

// Present exports the render target as a DMA-BUF file descriptor for display.
// The caller is responsible for closing the returned file descriptor.
func (b *GPUBackend) Present() (int, error) {
	// Export the render target as a DMA-BUF file descriptor
	fd, err := b.allocator.ExportDmabuf(b.renderTarget)
	if err != nil {
		return -1, fmt.Errorf("backend: export dmabuf failed: %w", err)
	}
	return fd, nil
}
