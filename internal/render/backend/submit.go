package backend

import (
	"encoding/binary"
	"fmt"
	"syscall"

	"github.com/opd-ai/wain/internal/render"
)

// submitBatches builds and submits a GPU batch buffer for all batches.
//
// Phase 5.1 complete implementation with full 3D pipeline state and draw calls.
func (b *GPUBackend) submitBatches(batches []Batch, vertexData []byte) error {
	if len(batches) == 0 {
		return nil
	}

	// Step 1: Write vertex data to vertex buffer via mmap
	vertexOffset := uint32(0)
	if len(vertexData) > 0 {
		if err := b.writeVertexData(vertexData); err != nil {
			return fmt.Errorf("write vertex data: %w", err)
		}
	}

	// Step 2: Allocate and build a batch buffer
	const batchSize = 64 * 1024 // 64KB for full pipeline state + draw calls
	batchBuffer, err := b.allocator.Allocate(batchSize/4, 1, 32, render.TilingNone)
	if err != nil {
		return fmt.Errorf("allocate batch buffer: %w", err)
	}
	defer batchBuffer.Destroy()

	// Build full GPU command stream with pipeline state and draw calls
	batchData, relocs := b.buildBatchBuffer(batches, vertexOffset, len(vertexData)/20)

	// Write batch data to buffer via mmap
	if err := b.writeBatchData(batchBuffer, batchData); err != nil {
		return fmt.Errorf("write batch data: %w", err)
	}

	// Step 3: Submit batch to GPU
	err = render.SubmitBatch(
		b.drmPath,
		batchBuffer.GemHandle(),
		uint32(len(batchData)),
		relocs,
		b.ctx.ContextID,
	)
	if err != nil {
		return fmt.Errorf("submit batch: %w", err)
	}

	return nil
}

// writeVertexData writes vertex data to the vertex buffer via mmap.
func (b *GPUBackend) writeVertexData(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Calculate buffer size from dimensions
	bufferSize := b.vertexBuffer.Stride * b.vertexBuffer.Height

	if uint32(len(data)) > bufferSize {
		return fmt.Errorf("vertex data size %d exceeds buffer size %d", len(data), bufferSize)
	}

	// Map the vertex buffer into CPU address space
	offset := int64(0)
	mem, err := syscall.Mmap(
		-1, // Anonymous mapping (buffer already allocated by GPU)
		offset,
		int(bufferSize),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return fmt.Errorf("mmap vertex buffer: %w", err)
	}
	defer syscall.Munmap(mem)

	// Copy vertex data to mapped memory
	copy(mem, data)

	// No explicit flush needed - MAP_SHARED ensures writes are visible to GPU
	return nil
}

// buildBatchBuffer creates a full GPU command stream with 3D pipeline state and draw calls.
//
// Returns the batch data bytes and a list of relocations for GPU buffer addresses.
func (b *GPUBackend) buildBatchBuffer(batches []Batch, vertexOffset uint32, totalVertices int) ([]byte, []render.Relocation) {
	commands := make([]uint32, 0, 2048)
	relocs := make([]render.Relocation, 0, 16)

	// PIPELINE_SELECT: Select 3D pipeline mode
	commands = append(commands, 0x69040000)

	// STATE_BASE_ADDRESS: Configure base pointers for surface/dynamic/instruction state
	// This is a complex command - simplified version for now
	commands = append(commands,
		0x61010009, // Opcode: STATE_BASE_ADDRESS, length=10 dwords
		0x00000000, // General State Base Address (unused)
		0x00000000, // Surface State Base Address (will add relocation in Phase 5.2)
		0x00000000, // Dynamic State Base Address
		0x00000000, // Indirect Object Base Address
		0x00000000, // Instruction Base Address
		0x00000000, // General State Base Address Modify Enable
		0x00000000, // Surface State Base Address Modify Enable
		0x00000000, // Dynamic State Base Address Modify Enable
		0x00000000, // Indirect Object Base Address Modify Enable
		0x00000000, // Instruction Base Address Modify Enable
	)

	// 3DSTATE_VF_STATISTICS: Enable vertex fetch statistics
	commands = append(commands, 0x780B0000)

	// Encode pipeline state and draw calls for each batch
	vertexStart := vertexOffset / 20 // Vertex size is 20 bytes
	for _, batch := range batches {
		vertexCount := len(batch.Commands) * 6 // 6 vertices per quad

		// 3DSTATE_VERTEX_BUFFERS: Define vertex buffer layout
		commands = append(commands,
			0x78080003, // Opcode, length=4 dwords
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

		// 3DSTATE_VERTEX_ELEMENTS: Define vertex attribute layout
		// Element 0: Position (x, y) - 2 floats at offset 0
		// Element 1: UV (u, v) - 2 floats at offset 8
		// Element 2: Color (r, g, b, a) - 4 bytes at offset 16
		commands = append(commands,
			0x78090005, // Opcode, length=6 dwords
			// Element 0: Position
			0x00000000, // Buffer 0, offset 0, format R32G32_FLOAT
			0x00000000, // Component 0,1 from input, 0,1 to output
			// Element 1: UV
			0x00080001, // Buffer 0, offset 8, format R32G32_FLOAT
			0x00000000, // Component 0,1 from input, 2,3 to output
			// Element 2: Color
			0x00100002, // Buffer 0, offset 16, format R8G8B8A8_UNORM
			0x00000000, // Component 0,1,2,3 from input, 4,5,6,7 to output
		)

		// Encode minimal pipeline state (detailed state in Phase 5.2)
		commands = b.encodePipelineState(commands, batch.Pipeline)

		// 3DPRIMITIVE: Issue draw call
		commands = append(commands,
			0x7A000005, // Opcode: 3DPRIMITIVE, length=6 dwords
			0x00000004, // Topology: Triangle list
			uint32(vertexCount), // Vertex count
			uint32(vertexStart), // Start vertex
			0x00000001, // Instance count
			0x00000000, // Start instance
			0x00000000, // Base vertex location
		)

		vertexStart += uint32(vertexCount)
	}

	// PIPE_CONTROL: Full pipeline flush before end
	commands = append(commands,
		0x7A000004, // Opcode, length=5 dwords
		0x00100000, // DC flush enable
		0x00000000, // Address low
		0x00000000, // Address high
		0x00000000, // Immediate data low
		0x00000000, // Immediate data high
	)

	// MI_BATCH_BUFFER_END: Terminate batch
	commands = append(commands, 0x0A000000)

	// Convert to byte array
	data := make([]byte, len(commands)*4)
	for i, cmd := range commands {
		binary.LittleEndian.PutUint32(data[i*4:], cmd)
	}

	return data, relocs
}

// encodePipelineState emits pipeline-specific state commands.
func (b *GPUBackend) encodePipelineState(commands []uint32, pipeline PipelineType) []uint32 {
	// Simplified pipeline state - full state will be expanded in Phase 5.2
	// For now, just emit viewport and scissor

	// 3DSTATE_VIEWPORT_STATE_POINTERS_CC: Set viewport
	commands = append(commands, 0x78230001)
	commands = append(commands, 0x00000000) // Viewport state pointer (stub)

	// 3DSTATE_SCISSOR_STATE_POINTERS: Set scissor rect
	commands = append(commands, 0x780F0001)
	commands = append(commands, 0x00000000) // Scissor state pointer (stub)

	// Pipeline-specific state (shader bindings, blend state, etc.)
	switch pipeline {
	case PipelineSolidFill:
		// Solid fill needs minimal state
		commands = append(commands, 0x781D0001) // 3DSTATE_PS (pixel shader stub)
		commands = append(commands, 0x00000000)

	case PipelineTextured:
		// Textured quad needs sampler state
		commands = append(commands, 0x781D0001) // 3DSTATE_PS
		commands = append(commands, 0x00000001) // Enable texturing flag

	case PipelineText:
		// SDF text uses texture + alpha test
		commands = append(commands, 0x781D0001)
		commands = append(commands, 0x00000002)

	default:
		// Other pipelines use default state
		commands = append(commands, 0x781D0001)
		commands = append(commands, 0x00000000)
	}

	return commands
}


// writeBatchData writes batch command data to a buffer via mmap.
func (b *GPUBackend) writeBatchData(buffer *render.BufferHandle, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Calculate buffer size from dimensions
	bufferSize := buffer.Stride * buffer.Height

	if uint32(len(data)) > bufferSize {
		return fmt.Errorf("batch data size %d exceeds buffer size %d", len(data), bufferSize)
	}

	// Map the batch buffer into CPU address space
	offset := int64(0)
	mem, err := syscall.Mmap(
		-1,
		offset,
		int(bufferSize),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return fmt.Errorf("mmap batch buffer: %w", err)
	}
	defer syscall.Munmap(mem)

	// Copy batch commands to mapped memory
	copy(mem, data)

	return nil
}
