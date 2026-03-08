package render

import (
	"encoding/binary"
)

// CommandBuilder helps construct GPU command streams as byte slices.
type CommandBuilder struct {
	data []byte
}

// NewCommandBuilder creates a new command builder.
func NewCommandBuilder() *CommandBuilder {
	return &CommandBuilder{
		data: make([]byte, 0, 4096),
	}
}

// EmitDword appends a 32-bit value to the command stream.
func (cb *CommandBuilder) EmitDword(dword uint32) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, dword)
	cb.data = append(cb.data, buf...)
}

// EmitQword appends a 64-bit value to the command stream.
func (cb *CommandBuilder) EmitQword(qword uint64) {
	cb.EmitDword(uint32(qword & 0xFFFFFFFF))
	cb.EmitDword(uint32(qword >> 32))
}

// Data returns the accumulated command stream bytes.
func (cb *CommandBuilder) Data() []byte {
	return cb.data
}

// Len returns the current size of the command stream in bytes.
func (cb *CommandBuilder) Len() int {
	return len(cb.data)
}

// MiNoop emits a MI_NOOP command.
func (cb *CommandBuilder) MiNoop() {
	cb.EmitDword(0x00000000)
}

// MiBatchBufferEnd emits a MI_BATCH_BUFFER_END command.
func (cb *CommandBuilder) MiBatchBufferEnd() {
	cb.EmitDword(0x0A000000)
}

// PipelineSelect emits a PIPELINE_SELECT command (3D mode).
func (cb *CommandBuilder) PipelineSelect3D() {
	opcode := uint32(0x6904)
	dw0 := (1 << 29) | (opcode << 16)
	cb.EmitDword(dw0)
}

// StateBaseAddress emits a STATE_BASE_ADDRESS command with dummy addresses.
// For the first triangle, we can use zero addresses since we're not using
// dynamic state, surface state, or instruction heaps.
func (cb *CommandBuilder) StateBaseAddress() {
	opcode := uint32(0x7801)
	length := uint32(15) // 16 DWords total
	dw0 := (3 << 29) | (opcode << 16) | length
	
	cb.EmitDword(dw0)
	// General state base (disabled - bit 0 = 0)
	cb.EmitDword(0)
	cb.EmitDword(0)
	// Surface state base (disabled)
	cb.EmitDword(0)
	cb.EmitDword(0)
	// Dynamic state base (disabled)
	cb.EmitDword(0)
	cb.EmitDword(0)
	// Indirect object base (disabled)
	cb.EmitDword(0)
	cb.EmitDword(0)
	// Instruction base (disabled)
	cb.EmitDword(0)
	cb.EmitDword(0)
	// Upper bounds
	cb.EmitDword(0xFFFFF000)
	cb.EmitDword(0xFFFFF000)
	cb.EmitDword(0xFFFFF000)
	cb.EmitDword(0xFFFFF000)
	cb.EmitDword(0)
}

// State3DClip emits a 3DSTATE_CLIP command with default settings.
func (cb *CommandBuilder) State3DClip() {
	opcode := uint32(0x7812)
	length := uint32(3) // 4 DWords total
	dw0 := (3 << 29) | (opcode << 16) | length
	
	dw1 := uint32(0)
	dw1 |= 1 << 31 // Clip enable
	dw1 |= 1 << 28 // Viewport XY clip test enable
	
	cb.EmitDword(dw0)
	cb.EmitDword(dw1)
	cb.EmitDword(0)
	cb.EmitDword(0)
}

// State3DSF emits a 3DSTATE_SF command (rasterization setup).
func (cb *CommandBuilder) State3DSF() {
	opcode := uint32(0x7813)
	length := uint32(3) // 4 DWords total
	dw0 := (3 << 29) | (opcode << 16) | length
	
	dw1 := uint32(0)
	dw1 |= 1 << 0 // CCW front winding
	// Cull mode = 0 (no culling)
	
	cb.EmitDword(dw0)
	cb.EmitDword(dw1)
	cb.EmitDword(0)
	cb.EmitDword(0)
}

// State3DWM emits a 3DSTATE_WM command.
func (cb *CommandBuilder) State3DWM() {
	opcode := uint32(0x7814)
	length := uint32(1) // 2 DWords total
	dw0 := (3 << 29) | (opcode << 16) | length
	
	dw1 := uint32(0)
	dw1 |= 1 << 25 // Pixel shader kill enable
	
	cb.EmitDword(dw0)
	cb.EmitDword(dw1)
}

// State3DPS emits a 3DSTATE_PS command with shader kernel address.
func (cb *CommandBuilder) State3DPS(kernelAddr uint64) {
	opcode := uint32(0x7820)
	length := uint32(11) // 12 DWords total
	dw0 := (3 << 29) | (opcode << 16) | length
	
	dw1 := uint32(kernelAddr & 0xFFFFFFFF)
	dw2 := uint32(kernelAddr >> 32)
	
	dw3 := uint32(0)
	dw3 |= 1 << 0 // 8-pixel dispatch enable
	
	cb.EmitDword(dw0)
	cb.EmitDword(dw1)
	cb.EmitDword(dw2)
	cb.EmitDword(dw3)
	// DWords 4-11 (shader parameters - zeros for now)
	for i := 0; i < 8; i++ {
		cb.EmitDword(0)
	}
}

// State3DVertexBuffers emits a 3DSTATE_VERTEX_BUFFERS command.
func (cb *CommandBuilder) State3DVertexBuffers(index uint32, address uint64, size uint32, stride uint32) {
	opcode := uint32(0x7808)
	length := uint32(3) // 4 DWords per buffer
	dw0 := (3 << 29) | (opcode << 16) | length
	
	dw1 := (index << 26) | (stride & 0x7FF)
	dw2 := uint32(address & 0xFFFFFFFF)
	dw3 := uint32(address >> 32)
	dw4 := size
	
	cb.EmitDword(dw0)
	cb.EmitDword(dw1)
	cb.EmitDword(dw2)
	cb.EmitDword(dw3)
	cb.EmitDword(dw4)
}

// State3DVertexElements emits a 3DSTATE_VERTEX_ELEMENTS command.
func (cb *CommandBuilder) State3DVertexElements(bufferIndex uint32, offset uint32, format uint32) {
	opcode := uint32(0x7809)
	length := uint32(1) // 2 DWords per element
	dw0 := (3 << 29) | (opcode << 16) | length
	
	dw1 := (bufferIndex << 26) | (offset & 0x7FF)
	dw2 := format
	
	cb.EmitDword(dw0)
	cb.EmitDword(dw1)
	cb.EmitDword(dw2)
}

// Primitive3D emits a 3DPRIMITIVE command for drawing triangles.
func (cb *CommandBuilder) Primitive3D(vertexCount uint32) {
	opcode := uint32(0x7A00)
	length := uint32(6) // 7 DWords total
	dw0 := (3 << 29) | (opcode << 16) | length
	
	topology := uint32(0x04) // Triangle list
	dw1 := topology & 0x3F
	
	cb.EmitDword(dw0)
	cb.EmitDword(dw1)
	cb.EmitDword(vertexCount)
	cb.EmitDword(0) // Start vertex
	cb.EmitDword(1) // Instance count
	cb.EmitDword(0) // Start instance
	cb.EmitDword(0) // Base vertex
}

// PipeControl emits a PIPE_CONTROL command with full flush.
func (cb *CommandBuilder) PipeControl() {
	opcode := uint32(0x7A00)
	length := uint32(4) // 5 DWords total
	dw0 := (3 << 29) | (opcode << 16) | length
	
	dw1 := uint32(0)
	dw1 |= 1 << 1  // Stall at pixel scoreboard
	dw1 |= 1 << 12 // Render target cache flush
	dw1 |= 1 << 0  // Depth cache flush
	dw1 |= 1 << 10 // Texture cache invalidate
	dw1 |= 1 << 20 // CS stall
	
	cb.EmitDword(dw0)
	cb.EmitDword(dw1)
	cb.EmitDword(0) // Address low
	cb.EmitDword(0) // Address high
	cb.EmitDword(0) // Data
}
