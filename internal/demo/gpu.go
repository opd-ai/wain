package demo

import (
	"fmt"

	"github.com/opd-ai/wain/internal/render"
)

// SetupGPUContext creates a GPU execution context and prints formatted status.
// The caller is responsible for closing the allocator on failure.
func SetupGPUContext(allocator *render.Allocator, drmPath string) (*render.GpuContext, error) {
	gpuCtx, err := render.CreateContext(drmPath)
	if err != nil {
		allocator.Close()
		return nil, fmt.Errorf("create GPU context: %w", err)
	}
	fmt.Printf("       ✓ Created context ID: %d", gpuCtx.ContextID)
	if gpuCtx.VmID != 0 {
		fmt.Printf(", VM ID: %d", gpuCtx.VmID)
	}
	fmt.Println()
	return gpuCtx, nil
}

// SetupGPUAllocator creates and initializes a GPU buffer allocator and context.
// It displays formatted progress messages indicating the current step number.
// Returns the allocator and GPU context, or an error if setup fails.
// The caller is responsible for closing the allocator when done.
func SetupGPUAllocator(drmPath string, stepNum, totalSteps int) (*render.Allocator, *render.GpuContext, error) {
	fmt.Printf("\n[%d/%d] Creating GPU buffer allocator...\n", stepNum, totalSteps)
	allocator, err := render.NewAllocator(drmPath)
	if err != nil {
		return nil, nil, fmt.Errorf("create allocator: %w (is %s accessible?)", err, drmPath)
	}
	fmt.Printf("       ✓ Opened %s\n", drmPath)

	fmt.Printf("\n[%d/%d] Detecting GPU generation...\n", stepNum+1, totalSteps)
	gpuGen := render.DetectGPU(drmPath)
	if gpuGen == render.GpuUnknown {
		allocator.Close()
		return nil, nil, fmt.Errorf("GPU detection failed or unsupported GPU")
	}
	fmt.Printf("       ✓ Detected: %s\n", gpuGen)

	fmt.Printf("\n[%d/%d] Creating GPU context...\n", stepNum+2, totalSteps)
	gpuCtx, err := SetupGPUContext(allocator, drmPath)
	if err != nil {
		return nil, nil, err
	}

	return allocator, gpuCtx, nil
}

// AllocateBuffer allocates a GPU buffer of the given dimensions with no tiling.
// Returns the buffer handle, a cleanup function that destroys the buffer, and
// any error encountered.
func AllocateBuffer(allocator *render.Allocator, width, height, bpp uint32) (*render.BufferHandle, func(), error) {
	buffer, err := allocator.Allocate(width, height, bpp, render.TilingNone)
	if err != nil {
		return nil, nil, fmt.Errorf("allocate buffer: %w", err)
	}
	return buffer, func() { buffer.Destroy() }, nil
}

// It displays a simple progress message with the given step number.
// Returns the allocator or an error if setup fails.
// The caller is responsible for closing the allocator when done.
func SetupGPUAllocatorSimple(drmPath string, stepNum, totalSteps int) (*render.Allocator, error) {
	fmt.Printf("\n[%d/%d] Creating GPU buffer allocator...\n", stepNum, totalSteps)
	allocator, err := render.NewAllocator(drmPath)
	if err != nil {
		return nil, fmt.Errorf("create allocator: %w (is %s accessible?)", err, drmPath)
	}
	fmt.Printf("      ✓ Opened %s\n", drmPath)
	return allocator, nil
}
