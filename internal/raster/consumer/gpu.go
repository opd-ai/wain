package consumer

import (
	"fmt"
	"syscall"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
)

// GPURenderer is the interface that GPU backends must implement.
// This interface avoids an import cycle with backend package.
type GPURenderer interface {
	Render(dl *displaylist.DisplayList) error
	RenderWithDamage(dl *displaylist.DisplayList, damage []displaylist.Rect) error
	Present() (int, error)
	Dimensions() (width, height int)
	Destroy() error
}

// GPUConsumer renders display lists using GPU acceleration.
// It wraps a GPURenderer implementation to provide a consumer interface
// matching the SoftwareConsumer pattern.
type GPUConsumer struct {
	renderer GPURenderer
}

// NewGPUConsumer creates a new GPU display list consumer from a GPU renderer.
// The renderer parameter should be a *backend.GPUBackend instance.
func NewGPUConsumer(renderer GPURenderer) (*GPUConsumer, error) {
	if renderer == nil {
		return nil, fmt.Errorf("consumer: nil GPU renderer")
	}

	return &GPUConsumer{
		renderer: renderer,
	}, nil
}

// Render executes all commands in the display list using GPU acceleration.
// The buf parameter is not used for GPU rendering, as the GPU backend
// maintains its own render target. It's kept for interface compatibility
// with SoftwareConsumer.
func (gc *GPUConsumer) Render(dl *displaylist.DisplayList, buf *primitives.Buffer) error {
	if dl == nil {
		return fmt.Errorf("consumer: nil display list")
	}

	if err := gc.renderer.Render(dl); err != nil {
		return fmt.Errorf("consumer: GPU render failed: %w", err)
	}

	// If a buffer was provided, copy the GPU render target to it
	// This enables hybrid CPU/GPU workflows
	if buf != nil {
		if err := gc.copyToBuffer(buf); err != nil {
			return fmt.Errorf("consumer: GPU to CPU copy failed: %w", err)
		}
	}

	return nil
}

// RenderWithDamage executes commands with damage tracking for incremental updates.
// Only renders regions that have changed, using GPU scissor rectangles for clipping.
func (gc *GPUConsumer) RenderWithDamage(dl *displaylist.DisplayList, damage []displaylist.Rect) error {
	if dl == nil {
		return fmt.Errorf("consumer: nil display list")
	}

	if err := gc.renderer.RenderWithDamage(dl, damage); err != nil {
		return fmt.Errorf("consumer: GPU render with damage failed: %w", err)
	}

	return nil
}

// Present exports the GPU render target as a DMA-BUF file descriptor for display.
// The caller is responsible for closing the returned file descriptor.
func (gc *GPUConsumer) Present() (int, error) {
	fd, err := gc.renderer.Present()
	if err != nil {
		return -1, fmt.Errorf("consumer: GPU present failed: %w", err)
	}
	return fd, nil
}

// Dimensions returns the render target width and height in pixels.
func (gc *GPUConsumer) Dimensions() (width, height int) {
	return gc.renderer.Dimensions()
}

// Destroy frees all GPU resources.
func (gc *GPUConsumer) Destroy() error {
	if gc.renderer != nil {
		if err := gc.renderer.Destroy(); err != nil {
			return fmt.Errorf("consumer: GPU cleanup failed: %w", err)
		}
		gc.renderer = nil
	}
	return nil
}

// copyToBuffer copies the GPU render target to a CPU buffer via DMA-BUF mmap.
// It calls Present to obtain the render target's DMA-BUF file descriptor,
// maps it read-only into the process address space, and bulk-copies the
// ARGB8888 pixel rows into buf. The buf's dimensions and stride are updated
// to match the render target if they differ.
func (gc *GPUConsumer) copyToBuffer(buf *primitives.Buffer) error {
	fd, err := gc.renderer.Present()
	if err != nil {
		return fmt.Errorf("readback: export render target: %w", err)
	}
	if fd < 0 {
		return fmt.Errorf("readback: invalid DMA-BUF fd %d", fd)
	}
	defer syscall.Close(fd)

	width, height := gc.renderer.Dimensions()
	stride := width * 4
	size := stride * height
	if size == 0 {
		return nil
	}

	mem, err := syscall.Mmap(fd, 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("readback: mmap render target: %w", err)
	}
	defer syscall.Munmap(mem)

	if len(buf.Pixels) < size {
		buf.Pixels = make([]byte, size)
	}
	buf.Width = width
	buf.Height = height
	buf.Stride = stride
	copy(buf.Pixels, mem)
	return nil
}
