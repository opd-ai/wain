package render

import (
	"testing"
)

func TestCommandBuilderBasics(t *testing.T) {
	cb := NewCommandBuilder()
	
	// Test emitting dwords
	cb.EmitDword(0x12345678)
	cb.EmitDword(0xABCDEF01)
	
	data := cb.Data()
	if len(data) != 8 {
		t.Errorf("Expected 8 bytes, got %d", len(data))
	}
	
	// Check little-endian encoding
	if data[0] != 0x78 || data[1] != 0x56 || data[2] != 0x34 || data[3] != 0x12 {
		t.Errorf("Incorrect little-endian encoding for first dword")
	}
}

func TestMiNoop(t *testing.T) {
	cb := NewCommandBuilder()
	cb.MiNoop()
	
	data := cb.Data()
	if len(data) != 4 {
		t.Errorf("MI_NOOP should be 4 bytes, got %d", len(data))
	}
	
	expected := []byte{0x00, 0x00, 0x00, 0x00}
	for i := 0; i < 4; i++ {
		if data[i] != expected[i] {
			t.Errorf("MI_NOOP byte %d: expected 0x%02x, got 0x%02x", i, expected[i], data[i])
		}
	}
}

func TestMiBatchBufferEnd(t *testing.T) {
	cb := NewCommandBuilder()
	cb.MiBatchBufferEnd()
	
	data := cb.Data()
	if len(data) != 4 {
		t.Errorf("MI_BATCH_BUFFER_END should be 4 bytes, got %d", len(data))
	}
	
	// MI_BATCH_BUFFER_END = 0x0A000000
	expected := []byte{0x00, 0x00, 0x00, 0x0A}
	for i := 0; i < 4; i++ {
		if data[i] != expected[i] {
			t.Errorf("MI_BATCH_BUFFER_END byte %d: expected 0x%02x, got 0x%02x", i, expected[i], data[i])
		}
	}
}

func TestPipelineSelect3D(t *testing.T) {
	cb := NewCommandBuilder()
	cb.PipelineSelect3D()
	
	data := cb.Data()
	if len(data) != 4 {
		t.Errorf("PIPELINE_SELECT should be 4 bytes, got %d", len(data))
	}
	
	// Check that opcode is encoded correctly
	// Expected: (1 << 29) | (0x6904 << 16) = 0x20000000 | 0x69040000 = 0x69040000
	// Wait, that's not right. Let me recalculate:
	// 0x6904 << 16 = 0x69040000
	// 1 << 29 = 0x20000000
	// OR them = 0x69040000 | 0x20000000 = 0x69040000 + 0x20000000 = 0x89040000
	dword := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	
	expected := uint32((1 << 29) | (0x6904 << 16))
	if dword != expected {
		t.Errorf("PIPELINE_SELECT dword should be 0x%08x, got 0x%08x", expected, dword)
	}
	
	// Bit 0 should be 0 for 3D mode
	if (dword & 1) != 0 {
		t.Errorf("3D mode bit should be 0, got 1")
	}
}

func TestStateBaseAddress(t *testing.T) {
	cb := NewCommandBuilder()
	cb.StateBaseAddress()
	
	data := cb.Data()
	expectedLen := 16 * 4 // 16 DWords
	if len(data) != expectedLen {
		t.Errorf("STATE_BASE_ADDRESS should be %d bytes, got %d", expectedLen, len(data))
	}
	
	// Check command header
	dword0 := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	
	cmdType := (dword0 >> 29) & 0x7
	if cmdType != 3 {
		t.Errorf("Command type should be 3 (3D), got %d", cmdType)
	}
	
	length := dword0 & 0xFF
	if length != 15 {
		t.Errorf("Length field should be 15, got %d", length)
	}
}

func TestState3DClip(t *testing.T) {
	cb := NewCommandBuilder()
	cb.State3DClip()
	
	data := cb.Data()
	expectedLen := 4 * 4 // 4 DWords
	if len(data) != expectedLen {
		t.Errorf("3DSTATE_CLIP should be %d bytes, got %d", expectedLen, len(data))
	}
	
	// Check DWord 1 for clip enable bit
	dword1 := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
	
	if (dword1 & (1 << 31)) == 0 {
		t.Errorf("Clip enable bit should be set")
	}
	
	if (dword1 & (1 << 28)) == 0 {
		t.Errorf("Viewport XY clip test enable bit should be set")
	}
}

func TestState3DVertexBuffers(t *testing.T) {
	cb := NewCommandBuilder()
	cb.State3DVertexBuffers(0, 0x1000, 128, 16)
	
	data := cb.Data()
	expectedLen := 5 * 4 // 5 DWords (header + 4 per buffer)
	if len(data) != expectedLen {
		t.Errorf("3DSTATE_VERTEX_BUFFERS should be %d bytes, got %d", expectedLen, len(data))
	}
}

func TestPrimitive3D(t *testing.T) {
	cb := NewCommandBuilder()
	cb.Primitive3D(3)
	
	data := cb.Data()
	expectedLen := 7 * 4 // 7 DWords
	if len(data) != expectedLen {
		t.Errorf("3DPRIMITIVE should be %d bytes, got %d", expectedLen, len(data))
	}
	
	// Check vertex count
	vertexCount := uint32(data[8]) | uint32(data[9])<<8 | uint32(data[10])<<16 | uint32(data[11])<<24
	if vertexCount != 3 {
		t.Errorf("Vertex count should be 3, got %d", vertexCount)
	}
	
	// Check topology (should be triangle list = 0x04)
	dword1 := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
	topology := dword1 & 0x3F
	if topology != 0x04 {
		t.Errorf("Topology should be 0x04 (triangle list), got 0x%02x", topology)
	}
}

func TestPipeControl(t *testing.T) {
	cb := NewCommandBuilder()
	cb.PipeControl()
	
	data := cb.Data()
	expectedLen := 5 * 4 // 5 DWords
	if len(data) != expectedLen {
		t.Errorf("PIPE_CONTROL should be %d bytes, got %d", expectedLen, len(data))
	}
	
	// Check DWord 1 for flush flags
	dword1 := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
	
	if (dword1 & (1 << 1)) == 0 {
		t.Errorf("Stall at pixel scoreboard should be set")
	}
	if (dword1 & (1 << 12)) == 0 {
		t.Errorf("Render target cache flush should be set")
	}
	if (dword1 & (1 << 20)) == 0 {
		t.Errorf("CS stall should be set")
	}
}

func TestFullBatchConstruction(t *testing.T) {
	// Test a minimal batch similar to buildTriangleBatch
	cb := NewCommandBuilder()
	
	cb.MiNoop()
	cb.MiNoop()
	cb.PipelineSelect3D()
	cb.StateBaseAddress()
	cb.State3DClip()
	cb.Primitive3D(3)
	cb.PipeControl()
	cb.MiBatchBufferEnd()
	
	data := cb.Data()
	
	// Should have reasonable size
	if len(data) < 50 {
		t.Errorf("Batch seems too small: %d bytes", len(data))
	}
	
	// Should be aligned to 4-byte boundary
	if len(data)%4 != 0 {
		t.Errorf("Batch should be 4-byte aligned, got %d bytes", len(data))
	}
	
	// Last command should be MI_BATCH_BUFFER_END (0x0A000000)
	lastDword := uint32(data[len(data)-4]) | uint32(data[len(data)-3])<<8 |
		uint32(data[len(data)-2])<<16 | uint32(data[len(data)-1])<<24
	
	if lastDword != 0x0A000000 {
		t.Errorf("Last command should be MI_BATCH_BUFFER_END (0x0A000000), got 0x%08x", lastDword)
	}
}
