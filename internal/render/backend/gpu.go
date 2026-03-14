// Package backend provides a GPU rendering backend that consumes display lists.
//
// This package implements Phase 5.1 of the roadmap: Display List Consumer.
// It takes a display list of draw commands and renders them using GPU command
// submission, leveraging the shader pipeline and batch infrastructure from Phase 4.
package backend

import (
	_ "embed"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/render"
)

//go:embed solid_fill.wgsl
var solidFillWGSL []byte

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

	// Font atlas for text rendering
	fontAtlas *text.Atlas

	// warnAtlasOnce ensures the nil-atlas warning is emitted only once per backend instance.
	warnAtlasOnce sync.Once

	// Frame profiler for performance metrics
	profiler *FrameProfiler

	// Compiled solid-fill fragment shader binary (nil on no-GPU paths)
	solidFillShader []byte
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

	// FontAtlas is the font atlas for GPU text rendering (optional).
	// When nil, text commands produce no visible output.
	FontAtlas *text.Atlas
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
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	allocator, ctx, err := initGPUResources(cfg.DRMPath)
	if err != nil {
		return nil, err
	}

	vertexBuffer, renderTarget, err := allocateBuffers(allocator, cfg)
	if err != nil {
		render.DestroyContext(cfg.DRMPath, ctx)
		allocator.Close()
		return nil, err
	}

	// Detect GPU generation and compile the solid-fill fragment shader once at
	// startup.  On non-GPU paths (or when the GPU gen is unrecognised) the
	// shader is silently skipped; the backend falls back to the fixed-function
	// batch construction that was used before Phase 4.3 integration.
	solidFillShader := compileSolidFillShader(cfg.DRMPath)

	backend := &GPUBackend{
		drmPath:         cfg.DRMPath,
		allocator:       allocator,
		ctx:             ctx,
		vertexBuffer:    vertexBuffer,
		vertexHandle:    vertexBuffer.GemHandle(),
		renderTarget:    renderTarget,
		targetHandle:    renderTarget.GemHandle(),
		width:           cfg.Width,
		height:          cfg.Height,
		fontAtlas:       cfg.FontAtlas,
		profiler:        NewFrameProfiler(),
		solidFillShader: solidFillShader,
	}

	return backend, nil
}

// validateConfig checks that backend configuration values are valid.
func validateConfig(cfg Config) error {
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return fmt.Errorf("backend: invalid dimensions %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.VertexBufferSize <= 0 {
		return fmt.Errorf("backend: invalid vertex buffer size %d", cfg.VertexBufferSize)
	}
	return nil
}

// initGPUResources detects and initializes GPU allocator and context for the specified DRM device.
func initGPUResources(drmPath string) (*render.Allocator, *render.GpuContext, error) {
	gen := render.DetectGPU(drmPath)
	if gen < 0 {
		return nil, nil, ErrNoGPU
	}

	allocator, err := render.NewAllocator(drmPath)
	if err != nil {
		return nil, nil, fmt.Errorf("backend: allocator creation failed: %w", err)
	}

	ctx, err := render.CreateContext(drmPath)
	if err != nil {
		allocator.Close()
		return nil, nil, fmt.Errorf("backend: context creation failed: %w", err)
	}

	return allocator, ctx, nil
}

// allocateBuffers creates vertex buffer and render target buffers with the specified configuration.
func allocateBuffers(allocator *render.Allocator, cfg Config) (*render.BufferHandle, *render.BufferHandle, error) {
	vertexWidthPx := uint32(cfg.VertexBufferSize / 4)
	vertexBuffer, err := allocator.Allocate(vertexWidthPx, 1, 32, render.TilingNone)
	if err != nil {
		return nil, nil, fmt.Errorf("backend: vertex buffer allocation failed: %w", err)
	}

	renderTarget, err := allocator.Allocate(uint32(cfg.Width), uint32(cfg.Height), 32, render.TilingY)
	if err != nil {
		vertexBuffer.Destroy()
		return nil, nil, fmt.Errorf("backend: render target allocation failed: %w", err)
	}

	return vertexBuffer, renderTarget, nil
}

// compileSolidFillShader compiles the embedded solid_fill WGSL fragment shader
// to native GPU machine code.  Returns nil if the GPU generation is unknown or
// compilation fails; callers must handle nil gracefully.
func compileSolidFillShader(drmPath string) []byte {
	gpuGen := render.DetectGPU(drmPath)
	if gpuGen < 0 {
		return nil
	}
	binary, err := render.CompileShader(
		string(solidFillWGSL),
		render.GpuGeneration(gpuGen),
		render.FragmentShader,
	)
	if err != nil {
		log.Printf("backend: solid_fill shader compile skipped (%v); using fixed-function path", err)
		return nil
	}
	return binary.Data
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
	return b.RenderWithDamage(dl, nil)
}

// RenderWithDamage renders a display list with optional damage tracking.
// If damage is nil or empty, renders the full frame.
// Phase 5.4 implementation: damage tracking with scissor rects.
func (b *GPUBackend) RenderWithDamage(dl *displaylist.DisplayList, damage []displaylist.Rect) error {
	if dl == nil {
		return ErrNilDisplayList
	}
	if dl.Len() == 0 {
		return nil
	}

	b.profiler.BeginFrame()

	commands, scissorRects := b.processDamage(dl.Commands(), damage)
	if len(commands) == 0 {
		return nil
	}

	batches := batchCommands(commands)
	vertexData, err := b.packVertices(batches)
	if err != nil {
		return fmt.Errorf("backend: vertex packing failed: %w", err)
	}

	b.profiler.MarkGPUSubmit()

	if err := b.submitBatchesWithScissor(batches, vertexData, scissorRects); err != nil {
		return fmt.Errorf("backend: batch submission failed: %w", err)
	}

	b.profiler.EndFrame()
	return nil
}

// processDamage filters draw commands by damage regions and converts damage rects to scissor rects.
func (b *GPUBackend) processDamage(commands []displaylist.DrawCommand, damage []displaylist.Rect) ([]displaylist.DrawCommand, []ScissorRect) {
	if len(damage) == 0 {
		return commands, nil
	}

	commands = displaylist.FilterCommandsByDamage(commands, damage)
	if len(commands) == 0 {
		return nil, nil
	}

	scissorRects := make([]ScissorRect, len(damage))
	for i, rect := range damage {
		scissorRects[i] = ClampScissorRect(
			ScissorRectFromDamage(rect),
			b.width,
			b.height,
		)
	}
	return commands, scissorRects
}

// RenderTarget returns the render target buffer for display.
func (b *GPUBackend) RenderTarget() *render.BufferHandle {
	return b.renderTarget
}

// GetFrameStats returns current frame profiling statistics.
func (b *GPUBackend) GetFrameStats() FrameStats {
	return b.profiler.GetStats()
}

// ResetFrameStats resets frame profiling statistics.
func (b *GPUBackend) ResetFrameStats() {
	b.profiler.Reset()
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
