// Package integration provides high-level integration of buffer synchronization
// with Wayland and X11 protocol handlers.
package integration

import (
	"github.com/opd-ai/wain/internal/buffer"
	"github.com/opd-ai/wain/internal/wayland/wire"
)

// WaylandBufferHandler wraps buffer release event handling with synchronizer integration.
type WaylandBufferHandler struct {
	bufferID  uint32
	sync      *buffer.Synchronizer
	slotIndex int
}

// NewWaylandBufferHandler creates a handler for Wayland buffer release events.
func NewWaylandBufferHandler(bufferID uint32, sync *buffer.Synchronizer, slotIndex int) (*WaylandBufferHandler, error) {
	// Register the buffer ID with the synchronizer
	if err := sync.RegisterBuffer(bufferID, slotIndex); err != nil {
		return nil, err
	}

	return &WaylandBufferHandler{
		bufferID:  bufferID,
		sync:      sync,
		slotIndex: slotIndex,
	}, nil
}

// HandleEvent processes buffer events and integrates with the synchronizer.
// Call this from your buffer event handler.
func (w *WaylandBufferHandler) HandleEvent(opcode uint16, args []wire.Argument) error {
	// If this is a release event (opcode 0), notify the synchronizer
	if opcode == 0 {
		return w.sync.OnReleaseEvent(w.bufferID)
	}
	return nil
}

// Cleanup unregisters the buffer from the synchronizer.
// Call this before destroying the buffer.
func (w *WaylandBufferHandler) Cleanup() {
	w.sync.UnregisterBuffer(w.bufferID)
}

// BufferID returns the buffer ID.
func (w *WaylandBufferHandler) BufferID() uint32 {
	return w.bufferID
}

// SlotIndex returns the ring slot index for this buffer.
func (w *WaylandBufferHandler) SlotIndex() int {
	return w.slotIndex
}
