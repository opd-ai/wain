package backend

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"syscall"

	"github.com/opd-ai/wain/internal/render"
)

// submitBatchesWithScissor builds and submits a GPU batch buffer with optional scissor rects.
//
// Phase 5.4 implementation: damage tracking with scissor clipping.
func (b *GPUBackend) submitBatchesWithScissor(batches []Batch, vertexData []byte, scissorRects []ScissorRect) error {
	if len(batches) == 0 {
		return nil
	}

	if len(vertexData) > 0 {
		if err := b.writeVertexData(vertexData); err != nil {
			return fmt.Errorf("write vertex data: %w", err)
		}
	}

	batchBuffer, err := b.allocateBatchBuffer()
	if err != nil {
		return fmt.Errorf("submit: allocate batch buffer: %w", err)
	}
	defer func() { _ = batchBuffer.Destroy() }()

	batchData, relocs := b.buildBatchBuffer(batches, 0, len(vertexData)/20, scissorRects)

	if err := b.writeBatchData(batchBuffer, batchData); err != nil {
		return fmt.Errorf("write batch data: %w", err)
	}

	return b.submitToGPU(batchBuffer, batchData, relocs)
}

// allocateBatchBuffer allocates a 64KB GPU buffer for batch command submission.
func (b *GPUBackend) allocateBatchBuffer() (*render.BufferHandle, error) {
	const batchSize = 64 * 1024
	buffer, err := b.allocator.Allocate(batchSize/4, 1, 32, render.TilingNone)
	if err != nil {
		return nil, fmt.Errorf("allocate batch buffer: %w", err)
	}
	return buffer, nil
}

// submitToGPU submits the batch buffer to the GPU with relocations for execution.
//
// When a compiled solid-fill shader is available (set at GPUBackend init time),
// it also dispatches a shader batch via render_submit_shader_batch so that the
// EU/RDNA kernel is active in the pipeline alongside the fixed-function state.
// The shader batch is best-effort: a failure is logged but does not abort the frame.
func (b *GPUBackend) submitToGPU(buffer *render.BufferHandle, data []byte, relocs []render.Relocation) error {
	b.maybeSubmitShaderBatch()

	if err := render.SubmitBatch(
		b.drmPath,
		buffer.GemHandle(),
		uint32(len(data)),
		relocs,
		b.ctx.ContextID,
	); err != nil {
		return fmt.Errorf("submit batch: %w", err)
	}
	return nil
}

// maybeSubmitShaderBatch submits the pre-compiled solid_fill shader batch when
// the shader binary is available.  Failure is non-fatal and falls through to
// the fixed-function path.
func (b *GPUBackend) maybeSubmitShaderBatch() {
	if len(b.solidFillShader) == 0 {
		return
	}
	if err := render.SubmitShaderBatch(b.drmPath, solidFillWGSL, true, b.ctx.ContextID); err != nil {
		log.Printf("backend: shader batch submission skipped (%v); using fixed-function path", err)
	}
}

// writeVertexData writes vertex data to the vertex buffer via mmap.
func (b *GPUBackend) writeVertexData(data []byte) error {
	return b.writeBufferData(b.vertexBuffer, data, "vertex")
}

// writeBatchData writes batch command data to a buffer via mmap.
func (b *GPUBackend) writeBatchData(buffer *render.BufferHandle, data []byte) error {
	return b.writeBufferData(buffer, data, "batch")
}

// writeBufferData writes data to a GPU buffer via mmap.
func (b *GPUBackend) writeBufferData(buffer *render.BufferHandle, data []byte, bufferType string) error {
	if len(data) == 0 {
		return nil
	}

	bufferSize := buffer.Stride * buffer.Height
	if uint32(len(data)) > bufferSize {
		return fmt.Errorf("%s data size %d exceeds buffer size %d", bufferType, len(data), bufferSize)
	}

	mem, err := syscall.Mmap(
		-1,
		0,
		int(bufferSize),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return fmt.Errorf("mmap %s buffer: %w", bufferType, err)
	}
	defer func() { _ = syscall.Munmap(mem) }()

	copy(mem, data)
	return nil
}

// buildBatchBuffer creates a full GPU command stream with 3D pipeline state and draw calls.
//
// Returns the batch data bytes and a list of relocations for GPU buffer addresses.
func (b *GPUBackend) buildBatchBuffer(batches []Batch, vertexOffset uint32, totalVertices int, scissorRects []ScissorRect) ([]byte, []render.Relocation) {
	commands := make([]uint32, 0, 2048)
	relocs := make([]render.Relocation, 0, 16)

	commands, relocs = b.encodePipelineHeader(commands, relocs, scissorRects)
	commands, relocs = b.encodeBatchDrawCalls(commands, relocs, batches, vertexOffset)
	commands = encodePipelineFooter(commands)

	return commandsToBytes(commands), relocs
}

// encodePipelineHeader encodes pipeline setup commands.
func (b *GPUBackend) encodePipelineHeader(commands []uint32, relocs []render.Relocation, scissorRects []ScissorRect) ([]uint32, []render.Relocation) {
	commands = append(commands, 0x69040000) // PIPELINE_SELECT
	commands, relocs = b.encodeStateBaseAddress(commands, relocs)
	commands = append(commands, 0x780B0000) // 3DSTATE_VF_STATISTICS
	if len(scissorRects) > 0 {
		commands = encodeScissorCommands(commands, scissorRects)
	}
	return commands, relocs
}

// encodeBatchDrawCalls encodes pipeline state and draw calls for all batches.
func (b *GPUBackend) encodeBatchDrawCalls(commands []uint32, relocs []render.Relocation, batches []Batch, vertexOffset uint32) ([]uint32, []render.Relocation) {
	vertexStart := vertexOffset / 20
	for _, batch := range batches {
		vertexCount := len(batch.Commands) * 6
		commands, relocs = b.encodeVertexBuffers(commands, relocs)
		commands = encodeVertexElements(commands)
		commands = b.encodePipelineState(commands, batch.Pipeline)
		commands = encodePrimitive(commands, uint32(vertexCount), vertexStart)
		vertexStart += uint32(vertexCount)
	}
	return commands, relocs
}

// encodePipelineFooter encodes pipeline flush and batch termination.
func encodePipelineFooter(commands []uint32) []uint32 {
	commands = append(commands,
		0x7A000004, 0x00100000, 0, 0, 0, 0, // PIPE_CONTROL
	)
	commands = append(commands, 0x0A000000) // MI_BATCH_BUFFER_END
	return commands
}

// commandsToBytes converts uint32 commands to byte array.
func commandsToBytes(commands []uint32) []byte {
	data := make([]byte, len(commands)*4)
	for i, cmd := range commands {
		binary.LittleEndian.PutUint32(data[i*4:], cmd)
	}
	return data
}

// encodeStateBaseAddress configures GPU base pointers for state buffers.
//
// Phase 5.2: adds a relocation for the Surface State Base Address so the GPU
// kernel can patch in the render-target buffer's GPU-virtual address at submission.
func (b *GPUBackend) encodeStateBaseAddress(commands []uint32, relocs []render.Relocation) ([]uint32, []render.Relocation) {
	// DWord 2 holds the Surface State Base Address; the kernel fills it via relocation.
	surfaceAddrOffset := uint64((len(commands) + 2) * 4)
	relocs = append(relocs, render.Relocation{
		Offset:       surfaceAddrOffset,
		TargetHandle: b.targetHandle,
		Delta:        0,
	})
	return append(commands,
		0x61010009, // Opcode: STATE_BASE_ADDRESS, length = 10 (11 DWords)
		0x00000000, // General State Base Address (unused)
		0x00000000, // Surface State Base Address (patched by relocation above)
		0x00000000, // Dynamic State Base Address
		0x00000000, // Indirect Object Base Address
		0x00000000, // Instruction Base Address
		0x00000000, // General State Base Address Modify Enable (0 = keep current)
		0x00000001, // Surface State Base Address Modify Enable (1 = apply relocation)
		0x00000000, // Dynamic State Base Address Modify Enable (0 = keep current)
		0x00000000, // Indirect Object Base Address Modify Enable (0 = keep current)
		0x00000000, // Instruction Base Address Modify Enable (0 = keep current)
	), relocs
}

// encodeScissorCommands emits 3DSTATE_SCISSOR_STATE_POINTERS with inline scissor data.
//
// Phase 5.2: the scissor rectangle descriptors are embedded inline immediately
// after the 2-DWord command, and the pointer is computed as a batch-buffer offset.
func encodeScissorCommands(commands []uint32, scissorRects []ScissorRect) []uint32 {
	// Inline scissor data starts after the 2-DWord 3DSTATE_SCISSOR_STATE_POINTERS command.
	scissorDataOffset := uint32((len(commands) + 2) * 4)
	commands = append(commands,
		0x780F0001,        // 3DSTATE_SCISSOR_STATE_POINTERS (length = 1, 2 DWords)
		scissorDataOffset, // offset to inline scissor descriptor(s)
	)
	for _, rect := range scissorRects {
		commands = append(commands, encodeScissorState(rect)...)
	}
	return commands
}

// encodeVertexBuffers emits vertex buffer configuration and adds relocations.
func (b *GPUBackend) encodeVertexBuffers(commands []uint32, relocs []render.Relocation) ([]uint32, []render.Relocation) {
	commands = append(commands,
		0x78080003, // Opcode: 3DSTATE_VERTEX_BUFFERS, length=4 dwords
		0x00000000, // VB0: Binding 0, stride=20 bytes
		0x00000014, // Stride: 20 bytes (5 floats/bytes)
	)
	// Add relocation for vertex buffer address (will be filled by kernel)
	bufferAddrOffset := uint64(len(commands) * 4)
	relocs = append(relocs, render.Relocation{
		Offset:       bufferAddrOffset,
		TargetHandle: b.vertexHandle,
		Delta:        0,
	})
	commands = append(commands,
		0x00000000, // Buffer address (relocation)
		0x00000000, // Buffer size
	)
	return commands, relocs
}

// encodeVertexElements defines the vertex attribute layout for the shader.
func encodeVertexElements(commands []uint32) []uint32 {
	return append(commands,
		0x78090005, // Opcode: 3DSTATE_VERTEX_ELEMENTS, length=6 dwords
		// Element 0: Position (x, y) - 2 floats at offset 0
		0x00000000, // Buffer 0, offset 0, format R32G32_FLOAT
		0x00000000, // Component 0,1 from input, 0,1 to output
		// Element 1: UV (u, v) - 2 floats at offset 8
		0x00080001, // Buffer 0, offset 8, format R32G32_FLOAT
		0x00000000, // Component 0,1 from input, 2,3 to output
		// Element 2: Color (r, g, b, a) - 4 bytes at offset 16
		0x00100002, // Buffer 0, offset 16, format R8G8B8A8_UNORM
		0x00000000, // Component 0,1,2,3 from input, 4,5,6,7 to output
	)
}

// encodePrimitive emits a 3DPRIMITIVE command to issue a draw call.
func encodePrimitive(commands []uint32, vertexCount, vertexStart uint32) []uint32 {
	return append(commands,
		0x7A000005,          // Opcode: 3DPRIMITIVE, length=6 dwords
		0x00000004,          // Topology: Triangle list
		uint32(vertexCount), // Vertex count
		uint32(vertexStart), // Start vertex
		0x00000001,          // Instance count
		0x00000000,          // Start instance
		0x00000000,          // Base vertex location
	)
}

// encodePipelineState emits pipeline-specific state commands.
//
// Phase 5.2: viewport and scissor state pointers reference inline descriptors
// appended immediately after the command headers.  The 3DSTATE_PS is expanded
// to its full 12-DWord Gen9 format (opcode 0x7820).
func (b *GPUBackend) encodePipelineState(commands []uint32, pipeline PipelineType) []uint32 {
	// The pipeline header is: vpCC(2) + scissor(2) + PS(12) = 16 DWords.
	// Inline state immediately follows at offset (len(commands) + 16) × 4.
	const pipelineHeaderDWords = 16
	ccVPOffset := uint32((len(commands) + pipelineHeaderDWords) * 4)
	// SCISSOR_RECT follows CC_VIEWPORT (2 DWords = 8 bytes).
	scissorOffset := ccVPOffset + 8

	// 3DSTATE_VIEWPORT_STATE_POINTERS_CC (opcode 0x7823, length = 1 → 2 DWords)
	commands = append(commands, 0x78230001, ccVPOffset)
	// 3DSTATE_SCISSOR_STATE_POINTERS (opcode 0x780F, length = 1 → 2 DWords)
	commands = append(commands, 0x780F0001, scissorOffset)
	// 3DSTATE_PS: full 12-DWord Gen9 form (opcode 0x7820, length = 11).
	commands = b.encodePixelShaderState(commands, pipeline)

	// Inline CC_VIEWPORT descriptor (2 DWords): minDepth = 0.0, maxDepth = 1.0.
	commands = appendCCViewport(commands)
	// Inline SCISSOR_RECT descriptor (2 DWords): full render-target coverage.
	commands = appendScissorDescriptor(commands, b.width, b.height)
	return commands
}

// encodePixelShaderState emits a full 12-DWord 3DSTATE_PS command (Gen9 Phase 5.2).
//
// Opcode: 0x7820, length = 11 (12 DWords total).
// The kernel start pointer DWords are zero here; on real hardware a relocation
// would patch them to the compiled shader binary's GPU virtual address.
func (b *GPUBackend) encodePixelShaderState(commands []uint32, pipeline PipelineType) []uint32 {
	// DW3: dispatch settings — bit 0 enables SIMD8 dispatch.
	dw3 := uint32(1) // SIMD8 dispatch
	switch pipeline {
	case PipelineTextured:
		dw3 |= 1 << 8 // sampler count = 1
	case PipelineText:
		dw3 |= (1 << 8) | (1 << 9) // sampler count = 1, alpha-test enable
	}
	return append(commands,
		0x7820000B, // 3DSTATE_PS: opcode 0x7820, length = 11 (12 DWords)
		0x00000000, // kernel start pointer low  (relocation fills on real HW)
		0x00000000, // kernel start pointer high
		dw3,        // dispatch settings (SIMD8 + pipeline-specific flags)
		0x00000000, // per-thread scratch space
		0x001F0000, // max threads = 31, push-constant enable
		0x00000000, // reserved
		0x00000000, // reserved
		0x00000000, // reserved
		0x00000000, // reserved
		0x00000000, // reserved
		0x00000000, // reserved
	)
}

// appendCCViewport encodes a CC_VIEWPORT descriptor (2 DWords: minDepth, maxDepth).
func appendCCViewport(commands []uint32) []uint32 {
	return append(commands,
		0x00000000,            // minDepth = 0.0 (IEEE 754)
		math.Float32bits(1.0), // maxDepth = 1.0
	)
}

// appendScissorDescriptor encodes a SCISSOR_RECT descriptor (2 DWords).
// The scissor covers the full render target at (0,0) to (width, height).
func appendScissorDescriptor(commands []uint32, width, height int) []uint32 {
	return append(commands,
		0x00000000,                         // minX = 0, minY = 0
		(uint32(height)<<16)|uint32(width), // maxY << 16 | maxX
	)
}
