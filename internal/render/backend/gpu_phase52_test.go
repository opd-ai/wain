package backend

import (
	"encoding/binary"
	"testing"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
)

// TestPhase52ViewportScissorPointers verifies that the batch buffer produced by
// buildBatchBuffer contains no zero-filled state pointer DWords for the
// 3DSTATE_VIEWPORT_STATE_POINTERS_CC and 3DSTATE_SCISSOR_STATE_POINTERS commands.
//
// This is the acceptance gate for Phase 5.2: replacing zero-filled stubs with
// computed inline-state offsets.
func TestPhase52ViewportScissorPointers(t *testing.T) {
	b := &GPUBackend{width: 800, height: 600}
	red := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	dl := displaylist.New()
	dl.AddFillRect(0, 0, 100, 100, red)
	batches := batchCommands(dl.Commands())
	data, _ := b.buildBatchBuffer(batches, 0, 6, nil)

	pointerOffsets := scanStatePointerOffsets(data)
	for _, off := range pointerOffsets {
		val := binary.LittleEndian.Uint32(data[off:])
		if val == 0 {
			t.Errorf("state pointer at batch offset %d is zero (stub not replaced)", off)
		}
	}
}

// TestPhase52PixelShaderExpanded verifies that 3DSTATE_PS is emitted as a
// full 12-DWord command (opcode 0x7820, length = 11), not as the 2-DWord stub.
func TestPhase52PixelShaderExpanded(t *testing.T) {
	b := &GPUBackend{width: 800, height: 600}
	red := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	dl := displaylist.New()
	dl.AddFillRect(0, 0, 100, 100, red)
	batches := batchCommands(dl.Commands())
	data, _ := b.buildBatchBuffer(batches, 0, 6, nil)

	const ps3DStateOpcode = uint32(0x7820000B) // opcode 0x7820, length = 11
	found := false
	for i := 0; i+4 <= len(data); i += 4 {
		dw := binary.LittleEndian.Uint32(data[i:])
		if dw == ps3DStateOpcode {
			found = true
			break
		}
	}
	if !found {
		t.Error("3DSTATE_PS (0x7820000B) not found in batch buffer; stub may not be replaced")
	}
}

// TestPhase52StateBaseAddressRelocation verifies that buildBatchBuffer emits a
// relocation entry for the Surface State Base Address slot.
func TestPhase52StateBaseAddressRelocation(t *testing.T) {
	b := &GPUBackend{width: 800, height: 600, targetHandle: 42}
	red := primitives.Color{R: 255, G: 0, B: 0, A: 255}
	dl := displaylist.New()
	dl.AddFillRect(0, 0, 100, 100, red)
	batches := batchCommands(dl.Commands())
	_, relocs := b.buildBatchBuffer(batches, 0, 6, nil)

	found := false
	for _, r := range relocs {
		if r.TargetHandle == b.targetHandle {
			found = true
			break
		}
	}
	if !found {
		t.Error("no relocation for render target (targetHandle) found in relocation list")
	}
}

// scanStatePointerOffsets scans a serialised command buffer and returns the byte
// offsets of the pointer DWord in each 3DSTATE_VIEWPORT_STATE_POINTERS_CC and
// 3DSTATE_SCISSOR_STATE_POINTERS command.
func scanStatePointerOffsets(data []byte) []int {
	var offsets []int
	for i := 0; i+8 <= len(data); i += 4 {
		dw := binary.LittleEndian.Uint32(data[i:])
		switch dw {
		case 0x78230001: // 3DSTATE_VIEWPORT_STATE_POINTERS_CC
			offsets = append(offsets, i+4)
		case 0x780F0001: // 3DSTATE_SCISSOR_STATE_POINTERS
			offsets = append(offsets, i+4)
		}
	}
	return offsets
}
