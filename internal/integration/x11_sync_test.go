package integration

import (
	"context"
	"testing"
	"time"

	"github.com/opd-ai/wain/internal/buffer"
)

func TestX11BufferHandler(t *testing.T) {
	// Create a ring with 2 slots
	ring, err := buffer.NewRing(2)
	if err != nil {
		t.Fatalf("NewRing() failed: %v", err)
	}

	sync := buffer.NewSynchronizer(ring)

	// Create X11 buffer handler for slot 0
	pixmapXID := uint32(1001)
	handler, err := NewX11BufferHandler(pixmapXID, sync, 0)
	if err != nil {
		t.Fatalf("NewX11BufferHandler() failed: %v", err)
	}
	defer handler.Cleanup()

	// Verify handler properties
	if handler.PixmapXID() != pixmapXID {
		t.Errorf("PixmapXID() = %d, want %d", handler.PixmapXID(), pixmapXID)
	}
	if handler.SlotIndex() != 0 {
		t.Errorf("SlotIndex() = %d, want 0", handler.SlotIndex())
	}

	// Acquire the slot and mark it as displaying
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	slot, err := sync.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() failed: %v", err)
	}

	if err := sync.MarkDisplaying(slot.Index); err != nil {
		t.Fatalf("MarkDisplaying() failed: %v", err)
	}

	// Verify slot is in displaying state
	if slot.State() != buffer.StateDisplaying {
		t.Errorf("slot.State() = %v, want StateDisplaying", slot.State())
	}

	// Simulate PresentIdleNotify event
	if err := handler.HandleIdleNotify(pixmapXID); err != nil {
		t.Fatalf("HandleIdleNotify() failed: %v", err)
	}

	// Verify slot transitioned to released state
	if slot.State() != buffer.StateReleased {
		t.Errorf("slot.State() = %v, want StateReleased after IdleNotify", slot.State())
	}
}

func TestX11BufferHandler_WrongPixmap(t *testing.T) {
	// Create a ring with 2 slots
	ring, err := buffer.NewRing(2)
	if err != nil {
		t.Fatalf("NewRing() failed: %v", err)
	}

	sync := buffer.NewSynchronizer(ring)

	// Create X11 buffer handler for slot 0
	pixmapXID := uint32(1001)
	handler, err := NewX11BufferHandler(pixmapXID, sync, 0)
	if err != nil {
		t.Fatalf("NewX11BufferHandler() failed: %v", err)
	}
	defer handler.Cleanup()

	// Acquire the slot and mark it as displaying
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	slot, err := sync.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() failed: %v", err)
	}

	if err := sync.MarkDisplaying(slot.Index); err != nil {
		t.Fatalf("MarkDisplaying() failed: %v", err)
	}

	// Simulate PresentIdleNotify event for a different pixmap
	wrongPixmapXID := uint32(9999)
	if err := handler.HandleIdleNotify(wrongPixmapXID); err != nil {
		t.Fatalf("HandleIdleNotify() with wrong pixmap should not error: %v", err)
	}

	// Verify slot is still in displaying state (not released)
	if slot.State() != buffer.StateDisplaying {
		t.Errorf("slot.State() = %v, want StateDisplaying (event ignored for wrong pixmap)", slot.State())
	}
}

func TestX11BufferHandler_MultipleHandlers(t *testing.T) {
	// Create a ring with 3 slots
	ring, err := buffer.NewRing(3)
	if err != nil {
		t.Fatalf("NewRing() failed: %v", err)
	}

	sync := buffer.NewSynchronizer(ring)

	// Create handlers for each slot
	handlers := make([]*X11BufferHandler, 3)
	for i := 0; i < 3; i++ {
		pixmapXID := uint32(2000 + i)
		h, err := NewX11BufferHandler(pixmapXID, sync, i)
		if err != nil {
			t.Fatalf("NewX11BufferHandler(%d) failed: %v", i, err)
		}
		handlers[i] = h
		defer h.Cleanup()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Acquire all slots and mark as displaying
	for i := 0; i < 3; i++ {
		slot, err := sync.AcquireForWriting(ctx)
		if err != nil {
			t.Fatalf("AcquireForWriting() for slot %d failed: %v", i, err)
		}

		if err := sync.MarkDisplaying(slot.Index); err != nil {
			t.Fatalf("MarkDisplaying(%d) failed: %v", slot.Index, err)
		}
	}

	// Release them in different order: 1, 0, 2
	releaseOrder := []int{1, 0, 2}
	for _, idx := range releaseOrder {
		if err := handlers[idx].HandleIdleNotify(handlers[idx].PixmapXID()); err != nil {
			t.Fatalf("HandleIdleNotify() for slot %d failed: %v", idx, err)
		}

		slot := ring.GetSlot(idx)
		if slot.State() != buffer.StateReleased {
			t.Errorf("slot %d: State() = %v, want StateReleased", idx, slot.State())
		}
	}

	// Verify stats
	stats := sync.Stats()
	ringStats := stats["ring_stats"].(map[string]int)
	if ringStats["released"] != 3 {
		t.Errorf("released count = %d, want 3", ringStats["released"])
	}
}

func TestX11BufferHandler_DuplicateRegistration(t *testing.T) {
	// Create a ring with 2 slots
	ring, err := buffer.NewRing(2)
	if err != nil {
		t.Fatalf("NewRing() failed: %v", err)
	}

	sync := buffer.NewSynchronizer(ring)

	// Create first handler
	pixmapXID := uint32(3001)
	handler1, err := NewX11BufferHandler(pixmapXID, sync, 0)
	if err != nil {
		t.Fatalf("NewX11BufferHandler() (first) failed: %v", err)
	}
	defer handler1.Cleanup()

	// Try to register the same pixmap XID again - should fail
	_, err = NewX11BufferHandler(pixmapXID, sync, 1)
	if err == nil {
		t.Error("NewX11BufferHandler() with duplicate pixmapXID should fail")
	}
}
