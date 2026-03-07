// Package buffer provides frame buffer ring management for double/triple buffering.
//
// The ring buffer manages a fixed set of buffers that cycle through states:
// available → rendering → displaying → released → available. This enables
// smooth presentation without blocking the render thread.
//
// Supports both Wayland (wl_buffer.release events) and X11 (Present events)
// synchronization mechanisms.
package buffer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// SlotState represents the lifecycle state of a buffer slot in the ring.
type SlotState int

const (
	// StateAvailable indicates the slot is ready for writing.
	StateAvailable SlotState = iota
	// StateRendering indicates rendering is in progress on this slot.
	StateRendering
	// StateDisplaying indicates the slot is currently being displayed.
	StateDisplaying
	// StateReleased indicates the compositor/server has released the slot.
	StateReleased
)

// String returns the string representation of the slot state.
func (s SlotState) String() string {
	switch s {
	case StateAvailable:
		return "available"
	case StateRendering:
		return "rendering"
	case StateDisplaying:
		return "displaying"
	case StateReleased:
		return "released"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// Slot represents a single buffer in the ring with its metadata and synchronization.
type Slot struct {
	// Index is the slot's position in the ring (0-based).
	Index int

	// UserData allows the caller to associate arbitrary data with the slot
	// (e.g., wl_buffer, GPU buffer handle, X11 pixmap XID).
	UserData interface{}

	// state tracks the current lifecycle state.
	state SlotState

	// releaseChan signals when the slot transitions to released state.
	// Buffered to prevent blocking the event handler.
	releaseChan chan struct{}

	// mu protects state transitions.
	mu sync.Mutex
}

// State returns the current state of the slot.
func (s *Slot) State() SlotState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// setState transitions the slot to a new state.
func (s *Slot) setState(newState SlotState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = newState
}

// WaitRelease blocks until the slot is released or the context is canceled.
// Returns nil on success, context error on cancellation/timeout.
func (s *Slot) WaitRelease(ctx context.Context) error {
	select {
	case <-s.releaseChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// signalRelease sends a non-blocking release notification.
// Safe to call multiple times; subsequent calls are no-ops.
func (s *Slot) signalRelease() {
	select {
	case s.releaseChan <- struct{}{}:
		// Successfully signaled
	default:
		// Already signaled, don't block
	}
}

// Ring manages a ring of buffer slots for double/triple buffering.
type Ring struct {
	// slots holds all buffer slots in the ring.
	slots []*Slot

	// size is the number of slots (typically 2 or 3).
	size int

	// nextAcquire is the index to try next for acquisition.
	nextAcquire int

	// mu protects ring-level state.
	mu sync.Mutex
}

// NewRing creates a ring buffer with the specified number of slots.
// size must be >= 2 (for double buffering) and is typically 2 or 3.
func NewRing(size int) (*Ring, error) {
	if size < 2 {
		return nil, errors.New("ring size must be at least 2 for double buffering")
	}

	slots := make([]*Slot, size)
	for i := 0; i < size; i++ {
		slots[i] = &Slot{
			Index:       i,
			state:       StateAvailable,
			releaseChan: make(chan struct{}, 1),
		}
	}

	return &Ring{
		slots:       slots,
		size:        size,
		nextAcquire: 0,
	}, nil
}

// Size returns the number of slots in the ring.
func (r *Ring) Size() int {
	return r.size
}

// GetSlot returns the slot at the given index.
// Panics if index is out of bounds.
func (r *Ring) GetSlot(index int) *Slot {
	if index < 0 || index >= r.size {
		panic(fmt.Sprintf("slot index %d out of bounds [0, %d)", index, r.size))
	}
	return r.slots[index]
}

// AcquireForWriting acquires the next available slot for rendering.
// Blocks until a slot becomes available or the context is canceled.
// The caller must call Release(slot.Index) when done presenting.
func (r *Ring) AcquireForWriting(ctx context.Context) (*Slot, error) {
	const pollInterval = 5 * time.Millisecond

	for {
		r.mu.Lock()
		// Try to find an available or released slot
		for i := 0; i < r.size; i++ {
			idx := (r.nextAcquire + i) % r.size
			slot := r.slots[idx]

			slot.mu.Lock()
			if slot.state == StateAvailable || slot.state == StateReleased {
				slot.state = StateRendering
				r.nextAcquire = (idx + 1) % r.size
				slot.mu.Unlock()
				r.mu.Unlock()
				return slot, nil
			}
			slot.mu.Unlock()
		}
		r.mu.Unlock()

		// No slots available, wait briefly and retry
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("acquire canceled: %w", ctx.Err())
		case <-time.After(pollInterval):
			// Retry
		}
	}
}

// MarkDisplaying transitions a slot from rendering to displaying state.
// Call this after presenting the buffer to the compositor/server.
func (r *Ring) MarkDisplaying(index int) error {
	slot := r.GetSlot(index)

	slot.mu.Lock()
	defer slot.mu.Unlock()

	if slot.state != StateRendering {
		return fmt.Errorf("cannot mark slot %d as displaying: current state is %s, expected rendering", index, slot.state)
	}

	slot.state = StateDisplaying
	return nil
}

// MarkReleased transitions a slot from displaying to released state and signals waiters.
// Call this from the event handler when receiving a release/idle notification.
func (r *Ring) MarkReleased(index int) error {
	slot := r.GetSlot(index)

	slot.mu.Lock()
	defer slot.mu.Unlock()

	if slot.state != StateDisplaying {
		return fmt.Errorf("cannot mark slot %d as released: current state is %s, expected displaying", index, slot.state)
	}

	slot.state = StateReleased
	slot.signalRelease()
	return nil
}

// MarkAvailable immediately transitions a slot to available state.
// Use this for cleanup or reset scenarios (not typical flow).
func (r *Ring) MarkAvailable(index int) {
	slot := r.GetSlot(index)
	slot.setState(StateAvailable)
}

// Reset transitions all slots to available state.
// Use this when reinitializing the display (e.g., after a VT switch).
func (r *Ring) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, slot := range r.slots {
		slot.setState(StateAvailable)
		// Drain release channel to prevent stale signals
		select {
		case <-slot.releaseChan:
		default:
		}
	}
	r.nextAcquire = 0
}

// Stats returns diagnostic information about the ring state.
func (r *Ring) Stats() map[string]int {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := make(map[string]int)
	for _, slot := range r.slots {
		state := slot.State()
		stats[state.String()]++
	}
	return stats
}
