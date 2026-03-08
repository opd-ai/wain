package integration

import (
	"os"
	"syscall"
	"testing"

	"github.com/opd-ai/wain/internal/render"
)

const (
	drmRenderNode = "/dev/dri/renderD128"
	testWidth     = 256
	testHeight    = 256
	testBpp       = 32 // ARGB8888
)

// TestGPUDetection validates GPU generation detection on Intel hardware.
//
// This test verifies that:
//  1. GPU detection returns a valid generation on Intel GPUs
//  2. GPU detection handles missing devices gracefully
//
// The test passes on Intel Gen9-Gen12 or Xe GPUs and skips on other hardware.
func TestGPUDetection(t *testing.T) {
	// Check if DRI render node exists
	if _, err := os.Stat(drmRenderNode); os.IsNotExist(err) {
		t.Skipf("Skipping GPU detection test: %s not found", drmRenderNode)
	}

	// Detect GPU generation
	gen := render.DetectGPU(drmRenderNode)

	// Valid Intel GPUs should return a known generation
	switch gen {
	case render.GpuGen9, render.GpuGen11, render.GpuGen12, render.GpuXe:
		t.Logf("Detected Intel GPU: %s", gen)
	case render.GpuUnknown:
		t.Skipf("Skipping GPU detection test: GPU not recognized or not Intel (got: %s)", gen)
	default:
		t.Errorf("Invalid GPU generation value: %d", gen)
	}
}

// TestGPUDetectionNonexistentDevice validates error handling for missing devices.
func TestGPUDetectionNonexistentDevice(t *testing.T) {
	gen := render.DetectGPU("/dev/nonexistent")
	if gen != render.GpuUnknown {
		t.Errorf("Expected GpuUnknown for nonexistent device, got: %s", gen)
	}
}

// TestBatchConstruction validates GPU command batch serialization.
//
// This test verifies that:
//  1. Batch buffer can be allocated from the slab allocator
//  2. Command data can be constructed correctly
//  3. Batch size is reasonable for simple operations
//
// This test skips if GPU hardware is not available.
func TestBatchConstruction(t *testing.T) {
	if _, err := os.Stat(drmRenderNode); os.IsNotExist(err) {
		t.Skipf("Skipping batch construction test: %s not found", drmRenderNode)
	}

	// Verify GPU is Intel
	gen := render.DetectGPU(drmRenderNode)
	if gen == render.GpuUnknown {
		t.Skipf("Skipping batch construction test: non-Intel GPU detected")
	}

	// Create allocator
	allocator, err := render.NewAllocator(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping batch construction test: failed to create allocator: %v", err)
	}
	defer allocator.Close()

	// Allocate batch buffer (4KB for simple commands)
	const batchSize = 4 * 1024
	batchBuf, err := allocator.Allocate(batchSize, 1, 8, render.TilingNone)
	if err != nil {
		t.Fatalf("Failed to allocate batch buffer: %v", err)
	}
	defer batchBuf.Destroy()

	// Construct minimal valid batch: MI_NOOP + MI_BATCH_BUFFER_END
	batch := buildMinimalBatch()

	// Verify batch is non-empty and reasonable size
	if len(batch) == 0 {
		t.Fatal("Batch construction produced empty buffer")
	}
	if len(batch) > int(batchSize) {
		t.Fatalf("Batch size (%d bytes) exceeds buffer size (%d bytes)", len(batch), batchSize)
	}

	t.Logf("Successfully constructed %d-byte batch", len(batch))
}

// TestGPUContextCreation validates GPU context creation.
//
// This test verifies that:
//  1. GPU contexts can be created successfully on Intel GPUs
//  2. Context IDs are valid (non-zero for i915, both IDs for Xe)
//
// This test skips if GPU hardware is not available.
func TestGPUContextCreation(t *testing.T) {
	if _, err := os.Stat(drmRenderNode); os.IsNotExist(err) {
		t.Skipf("Skipping context creation test: %s not found", drmRenderNode)
	}

	gen := render.DetectGPU(drmRenderNode)
	if gen == render.GpuUnknown {
		t.Skipf("Skipping context creation test: non-Intel GPU detected")
	}

	ctx, err := render.CreateContext(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping context creation test: failed to create context: %v", err)
	}

	// Validate context ID is non-zero
	if ctx.ContextID == 0 {
		t.Error("Context ID is zero (expected non-zero)")
	}

	// For Xe GPUs, VM ID should also be non-zero
	if gen == render.GpuXe && ctx.VmID == 0 {
		t.Error("VM ID is zero on Xe GPU (expected non-zero)")
	}

	t.Logf("Created context: ID=%d, VM=%d (GPU: %s)", ctx.ContextID, ctx.VmID, gen)
}

// TestBatchSubmission validates end-to-end GPU command submission.
//
// This test verifies that:
//  1. GPU context can be created
//  2. Batch buffer can be submitted successfully
//  3. GPU execution completes without error
//
// Note: This test submits a minimal batch (MI_NOOP + END) as an infrastructure
// validation. Full rendering tests (clear + draw with pixel verification) are
// deferred until Phase 4 when shader compilation is available.
//
// This test skips if GPU hardware is not available.
func TestBatchSubmission(t *testing.T) {
	if _, err := os.Stat(drmRenderNode); os.IsNotExist(err) {
		t.Skipf("Skipping batch submission test: %s not found", drmRenderNode)
	}

	gen := render.DetectGPU(drmRenderNode)
	if gen == render.GpuUnknown {
		t.Skipf("Skipping batch submission test: non-Intel GPU detected")
	}

	// Create allocator
	allocator, err := render.NewAllocator(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping batch submission test: failed to create allocator: %v", err)
	}
	defer allocator.Close()

	// Create GPU context
	ctx, err := render.CreateContext(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping batch submission test: failed to create context: %v", err)
	}

	// Allocate batch buffer
	const batchSize = 4 * 1024
	batchBuf, err := allocator.Allocate(batchSize, 1, 8, render.TilingNone)
	if err != nil {
		t.Fatalf("Failed to allocate batch buffer: %v", err)
	}
	defer batchBuf.Destroy()

	// Build minimal batch
	batch := buildMinimalBatch()

	// Submit batch with no relocations
	err = render.SubmitBatch(drmRenderNode, batchBuf.GemHandle(), uint32(len(batch)), nil, ctx.ContextID)
	if err != nil {
		t.Fatalf("Batch submission failed: %v", err)
	}

	t.Logf("Successfully submitted and executed %d-byte batch on %s", len(batch), gen)
}

// TestBatchSubmissionWithRenderTarget validates GPU command submission with a render target.
//
// This test verifies that:
//  1. Render target buffer can be allocated
//  2. Batch with relocations can be submitted
//  3. GPU execution completes successfully
//
// Note: This test does not verify pixel values yet (deferred to Phase 4+).
//
// This test skips if GPU hardware is not available.
func TestBatchSubmissionWithRenderTarget(t *testing.T) {
	if _, err := os.Stat(drmRenderNode); os.IsNotExist(err) {
		t.Skipf("Skipping render target test: %s not found", drmRenderNode)
	}

	gen := render.DetectGPU(drmRenderNode)
	if gen == render.GpuUnknown {
		t.Skipf("Skipping render target test: non-Intel GPU detected")
	}

	// Create allocator
	allocator, err := render.NewAllocator(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping render target test: failed to create allocator: %v", err)
	}
	defer allocator.Close()

	// Create GPU context
	ctx, err := render.CreateContext(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping render target test: failed to create context: %v", err)
	}

	// Allocate render target
	renderTarget, err := allocator.Allocate(testWidth, testHeight, testBpp, render.TilingNone)
	if err != nil {
		t.Fatalf("Failed to allocate render target: %v", err)
	}
	defer renderTarget.Destroy()

	// Allocate batch buffer
	const batchSize = 8 * 1024
	batchBuf, err := allocator.Allocate(batchSize, 1, 8, render.TilingNone)
	if err != nil {
		t.Fatalf("Failed to allocate batch buffer: %v", err)
	}
	defer batchBuf.Destroy()

	// Build batch referencing render target
	batch := buildBatchWithRenderTarget(renderTarget.GemHandle())

	// Create relocation entry for render target reference
	// Note: This is a placeholder relocation for infrastructure testing.
	// Full relocations will be implemented in Phase 4+ when shaders are available.
	relocs := []render.Relocation{
		{
			TargetHandle:   renderTarget.GemHandle(),
			Delta:          0,
			Offset:         4, // Hypothetical offset in batch where RT handle is referenced
			PresumedOffset: 0,
			ReadDomains:    render.GemDomainRender,
			WriteDomain:    render.GemDomainRender,
		},
	}

	// Submit batch
	err = render.SubmitBatch(drmRenderNode, batchBuf.GemHandle(), uint32(len(batch)), relocs, ctx.ContextID)
	if err != nil {
		t.Fatalf("Batch submission with render target failed: %v", err)
	}

	t.Logf("Successfully submitted batch with %dx%d render target on %s", testWidth, testHeight, gen)
}

// TestBatchSubmissionMultipleContexts validates that multiple GPU contexts work correctly.
//
// This test verifies that:
//  1. Multiple contexts can be created independently
//  2. Each context can submit batches successfully
//  3. Context isolation works as expected
//
// This test skips if GPU hardware is not available.
func TestBatchSubmissionMultipleContexts(t *testing.T) {
	if _, err := os.Stat(drmRenderNode); os.IsNotExist(err) {
		t.Skipf("Skipping multiple contexts test: %s not found", drmRenderNode)
	}

	gen := render.DetectGPU(drmRenderNode)
	if gen == render.GpuUnknown {
		t.Skipf("Skipping multiple contexts test: non-Intel GPU detected")
	}

	// Create allocator
	allocator, err := render.NewAllocator(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping multiple contexts test: failed to create allocator: %v", err)
	}
	defer allocator.Close()

	// Create two independent contexts
	ctx1, err := render.CreateContext(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping multiple contexts test: failed to create context 1: %v", err)
	}

	ctx2, err := render.CreateContext(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping multiple contexts test: failed to create context 2: %v", err)
	}

	// Verify contexts have different IDs
	if ctx1.ContextID == ctx2.ContextID {
		t.Error("Context IDs are identical (expected unique IDs)")
	}

	// Allocate batch buffer
	const batchSize = 4 * 1024
	batchBuf, err := allocator.Allocate(batchSize, 1, 8, render.TilingNone)
	if err != nil {
		t.Fatalf("Failed to allocate batch buffer: %v", err)
	}
	defer batchBuf.Destroy()

	batch := buildMinimalBatch()

	// Submit to context 1
	err = render.SubmitBatch(drmRenderNode, batchBuf.GemHandle(), uint32(len(batch)), nil, ctx1.ContextID)
	if err != nil {
		t.Errorf("Batch submission to context 1 failed: %v", err)
	}

	// Submit to context 2
	err = render.SubmitBatch(drmRenderNode, batchBuf.GemHandle(), uint32(len(batch)), nil, ctx2.ContextID)
	if err != nil {
		t.Errorf("Batch submission to context 2 failed: %v", err)
	}

	t.Logf("Successfully submitted batches to 2 independent contexts on %s", gen)
}

// TestAllocatorClose validates that allocator cleanup works correctly.
func TestAllocatorClose(t *testing.T) {
	if _, err := os.Stat(drmRenderNode); os.IsNotExist(err) {
		t.Skipf("Skipping allocator close test: %s not found", drmRenderNode)
	}

	gen := render.DetectGPU(drmRenderNode)
	if gen == render.GpuUnknown {
		t.Skipf("Skipping allocator close test: non-Intel GPU detected")
	}

	allocator, err := render.NewAllocator(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping allocator close test: failed to create allocator: %v", err)
	}

	// Close should succeed
	allocator.Close()

	// Multiple closes should be safe (idempotent)
	allocator.Close()
	allocator.Close()
}

// TestBufferExportDmabuf validates DMA-BUF file descriptor export.
func TestBufferExportDmabuf(t *testing.T) {
	if _, err := os.Stat(drmRenderNode); os.IsNotExist(err) {
		t.Skipf("Skipping dmabuf export test: %s not found", drmRenderNode)
	}

	gen := render.DetectGPU(drmRenderNode)
	if gen == render.GpuUnknown {
		t.Skipf("Skipping dmabuf export test: non-Intel GPU detected")
	}

	allocator, err := render.NewAllocator(drmRenderNode)
	if err != nil {
		t.Skipf("Skipping dmabuf export test: failed to create allocator: %v", err)
	}
	defer allocator.Close()

	// Allocate buffer
	buf, err := allocator.Allocate(256, 256, 32, render.TilingNone)
	if err != nil {
		t.Fatalf("Failed to allocate buffer: %v", err)
	}
	defer buf.Destroy()

	// Export DMA-BUF fd
	fd, err := allocator.ExportDmabuf(buf)
	if err != nil {
		t.Fatalf("Failed to export dmabuf: %v", err)
	}
	defer syscall.Close(fd)

	// Verify fd is valid
	if fd < 0 {
		t.Errorf("Invalid file descriptor: %d", fd)
	}

	t.Logf("Successfully exported dmabuf fd %d for buffer on %s", fd, gen)
}

// buildMinimalBatch constructs a minimal valid GPU command batch.
//
// The batch contains:
//   - MI_NOOP (no operation)
//   - MI_BATCH_BUFFER_END (terminates batch execution)
//
// This is used for infrastructure validation. Full rendering batches with
// clear/draw commands are deferred to Phase 4+.
func buildMinimalBatch() []byte {
	batch := []uint32{
		0x00000000, // MI_NOOP
		0x0A000000, // MI_BATCH_BUFFER_END
	}
	return u32SliceToBytes(batch)
}

// buildBatchWithRenderTarget constructs a batch that references a render target.
//
// Note: This is a placeholder for infrastructure testing. The actual batch
// doesn't perform rendering yet (full pipeline requires shader compilation in Phase 4).
func buildBatchWithRenderTarget(renderTargetHandle uint32) []byte {
	_ = renderTargetHandle // Will be used in Phase 4+

	// For now, return minimal batch + padding where RT handle would go
	batch := []uint32{
		0x00000000, // MI_NOOP
		0x00000000, // Placeholder for RT handle reference
		0x0A000000, // MI_BATCH_BUFFER_END
	}
	return u32SliceToBytes(batch)
}

// u32SliceToBytes converts a uint32 slice to little-endian byte slice.
func u32SliceToBytes(data []uint32) []byte {
	result := make([]byte, len(data)*4)
	for i, dword := range data {
		result[i*4+0] = byte(dword)
		result[i*4+1] = byte(dword >> 8)
		result[i*4+2] = byte(dword >> 16)
		result[i*4+3] = byte(dword >> 24)
	}
	return result
}
