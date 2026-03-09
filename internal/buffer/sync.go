package buffer

import (
	"context"
	"fmt"
	"sync"
)

// Synchronizer coordinates buffer release events from Wayland/X11 compositors
// with the buffer ring state machine. It acts as a bridge between protocol-level
// event handlers and buffer lifecycle management.
type Synchronizer struct {
	ring *Ring

	// indexMap maps external buffer IDs (wl_buffer ID, X11 Pixmap XID) to ring slot indices.
	indexMap map[uint32]int
	mu       sync.RWMutex
}

// NewSynchronizer creates a synchronizer for the given buffer ring.
func NewSynchronizer(ring *Ring) *Synchronizer {
	return &Synchronizer{
		ring:     ring,
		indexMap: make(map[uint32]int),
	}
}

// RegisterBuffer associates an external buffer ID with a ring slot index.
// Call this after creating a protocol buffer object (wl_buffer, Pixmap).
func (s *Synchronizer) RegisterBuffer(externalID uint32, slotIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if slotIndex < 0 || slotIndex >= s.ring.Size() {
		return fmt.Errorf("slot index %d out of range [0, %d)", slotIndex, s.ring.Size())
	}

	if existing, exists := s.indexMap[externalID]; exists {
		return fmt.Errorf("external ID %d already registered to slot %d", externalID, existing)
	}

	s.indexMap[externalID] = slotIndex
	return nil
}

// UnregisterBuffer removes the association for an external buffer ID.
// Call this when destroying a protocol buffer object.
func (s *Synchronizer) UnregisterBuffer(externalID uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.indexMap, externalID)
}

// OnReleaseEvent handles a buffer release event from the compositor.
// For Wayland: call from wl_buffer.release event handler.
// For X11: call from PresentIdleNotify event handler.
func (s *Synchronizer) OnReleaseEvent(externalID uint32) error {
	s.mu.RLock()
	slotIndex, exists := s.indexMap[externalID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("release event for unknown buffer ID %d", externalID)
	}

	return s.ring.MarkReleased(slotIndex)
}

// AcquireForWriting is a convenience wrapper around ring.AcquireForWriting.
func (s *Synchronizer) AcquireForWriting(ctx context.Context) (*Slot, error) {
	return s.ring.AcquireForWriting(ctx)
}

// MarkDisplaying is a convenience wrapper around ring.MarkDisplaying.
func (s *Synchronizer) MarkDisplaying(slotIndex int) error {
	return s.ring.MarkDisplaying(slotIndex)
}

// GetSlot is a convenience wrapper around ring.GetSlot.
func (s *Synchronizer) GetSlot(index int) (*Slot, error) {
	return s.ring.GetSlot(index)
}

// Reset is a convenience wrapper around ring.Reset.
func (s *Synchronizer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ring.Reset()
	// Clear all buffer registrations on reset
	s.indexMap = make(map[uint32]int)
}

// Stats returns diagnostic information including mapping stats.
func (s *Synchronizer) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"ring_stats":       s.ring.Stats(),
		"registered_count": len(s.indexMap),
	}
}
