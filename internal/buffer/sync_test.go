package buffer

import (
	"context"
	"testing"
)

func TestNewSynchronizer(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	sync := NewSynchronizer(ring)
	if sync == nil {
		t.Fatal("NewSynchronizer() returned nil")
	}

	stats := sync.Stats()
	ringStats := stats["ring_stats"].(map[string]int)
	if ringStats["available"] != 2 {
		t.Errorf("initial ring_stats[available] = %d, want 2", ringStats["available"])
	}
	if stats["registered_count"] != 0 {
		t.Errorf("initial registered_count = %d, want 0", stats["registered_count"])
	}
}

func TestSynchronizer_RegisterBuffer(t *testing.T) {
	ring, err := NewRing(3)
	if err != nil {
		t.Fatalf("NewRing(3) failed: %v", err)
	}

	sync := NewSynchronizer(ring)

	// Register buffer ID 100 to slot 0
	if err := sync.RegisterBuffer(100, 0); err != nil {
		t.Fatalf("RegisterBuffer(100, 0) failed: %v", err)
	}

	// Register buffer ID 200 to slot 1
	if err := sync.RegisterBuffer(200, 1); err != nil {
		t.Fatalf("RegisterBuffer(200, 1) failed: %v", err)
	}

	// Verify registration count
	stats := sync.Stats()
	if stats["registered_count"] != 2 {
		t.Errorf("registered_count = %d, want 2", stats["registered_count"])
	}

	// Try to register duplicate external ID
	err = sync.RegisterBuffer(100, 2)
	if err == nil {
		t.Error("RegisterBuffer() should fail for duplicate external ID")
	}

	// Try to register invalid slot index
	err = sync.RegisterBuffer(300, 99)
	if err == nil {
		t.Error("RegisterBuffer() should fail for out-of-range slot index")
	}
}

func TestSynchronizer_UnregisterBuffer(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	sync := NewSynchronizer(ring)

	// Register and then unregister
	sync.RegisterBuffer(100, 0)
	stats := sync.Stats()
	if stats["registered_count"] != 1 {
		t.Errorf("registered_count = %d, want 1", stats["registered_count"])
	}

	sync.UnregisterBuffer(100)
	stats = sync.Stats()
	if stats["registered_count"] != 0 {
		t.Errorf("after unregister, registered_count = %d, want 0", stats["registered_count"])
	}

	// Unregistering non-existent ID is a no-op
	sync.UnregisterBuffer(999)
	stats = sync.Stats()
	if stats["registered_count"] != 0 {
		t.Errorf("registered_count = %d, want 0", stats["registered_count"])
	}
}

func TestSynchronizer_OnReleaseEvent(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	sync := NewSynchronizer(ring)
	ctx := context.Background()

	// Acquire slot 0
	slot, err := sync.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() failed: %v", err)
	}

	// Register external buffer ID
	externalID := uint32(12345)
	if err := sync.RegisterBuffer(externalID, slot.Index); err != nil {
		t.Fatalf("RegisterBuffer(%d, %d) failed: %v", externalID, slot.Index, err)
	}

	// Mark as displaying
	if err := sync.MarkDisplaying(slot.Index); err != nil {
		t.Fatalf("MarkDisplaying(%d) failed: %v", slot.Index, err)
	}

	// Simulate compositor release event
	if err := sync.OnReleaseEvent(externalID); err != nil {
		t.Fatalf("OnReleaseEvent(%d) failed: %v", externalID, err)
	}

	// Verify slot is now released
	if slot.State() != StateReleased {
		t.Errorf("slot.State() = %v, want %v", slot.State(), StateReleased)
	}
}

func TestSynchronizer_OnReleaseEvent_UnknownID(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	sync := NewSynchronizer(ring)

	// Release event for unknown buffer ID should fail
	err = sync.OnReleaseEvent(99999)
	if err == nil {
		t.Error("OnReleaseEvent() should fail for unknown buffer ID")
	}
}

func TestSynchronizer_OnReleaseEvent_InvalidState(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	sync := NewSynchronizer(ring)
	ctx := context.Background()

	// Acquire slot but don't mark as displaying
	slot, err := sync.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() failed: %v", err)
	}

	// Register external buffer ID
	externalID := uint32(12345)
	sync.RegisterBuffer(externalID, slot.Index)

	// Try to release while still in rendering state (should fail)
	err = sync.OnReleaseEvent(externalID)
	if err == nil {
		t.Error("OnReleaseEvent() should fail when slot is not in displaying state")
	}
}

func TestSynchronizer_Reset(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	sync := NewSynchronizer(ring)
	ctx := context.Background()

	// Acquire slots and register buffers
	slot1, _ := sync.AcquireForWriting(ctx)
	sync.RegisterBuffer(100, slot1.Index)
	sync.MarkDisplaying(slot1.Index)

	slot2, _ := sync.AcquireForWriting(ctx)
	sync.RegisterBuffer(200, slot2.Index)

	stats := sync.Stats()
	if stats["registered_count"] != 2 {
		t.Errorf("registered_count = %d, want 2", stats["registered_count"])
	}

	// Reset
	sync.Reset()

	// Verify all slots are available
	stats = sync.Stats()
	ringStats := stats["ring_stats"].(map[string]int)
	if ringStats["available"] != 2 {
		t.Errorf("after reset, available slots = %d, want 2", ringStats["available"])
	}

	// Verify all buffer registrations are cleared
	if stats["registered_count"] != 0 {
		t.Errorf("after reset, registered_count = %d, want 0", stats["registered_count"])
	}
}

func TestSynchronizer_FullCycle(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	sync := NewSynchronizer(ring)
	ctx := context.Background()

	const (
		wlBufferID1 = uint32(1001)
		wlBufferID2 = uint32(1002)
	)

	// Frame 1: acquire, register, display, release
	slot1, err := sync.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() frame 1 failed: %v", err)
	}
	sync.RegisterBuffer(wlBufferID1, slot1.Index)
	sync.MarkDisplaying(slot1.Index)
	sync.OnReleaseEvent(wlBufferID1)

	// Frame 2: acquire, register, display (still showing)
	slot2, err := sync.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() frame 2 failed: %v", err)
	}
	sync.RegisterBuffer(wlBufferID2, slot2.Index)
	sync.MarkDisplaying(slot2.Index)

	// Frame 3: should reacquire slot1 (it was released)
	slot3, err := sync.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() frame 3 failed: %v", err)
	}
	if slot3.Index != slot1.Index {
		t.Errorf("expected to reacquire slot %d, got %d", slot1.Index, slot3.Index)
	}

	// Unregister old buffer from slot1, register new buffer
	sync.UnregisterBuffer(wlBufferID1)
	newBufferID := uint32(1003)
	sync.RegisterBuffer(newBufferID, slot3.Index)
	sync.MarkDisplaying(slot3.Index)

	// Verify state
	stats := sync.Stats()
	if stats["registered_count"] != 2 {
		t.Errorf("registered_count = %d, want 2", stats["registered_count"])
	}

	ringStats := stats["ring_stats"].(map[string]int)
	if ringStats["displaying"] != 2 {
		t.Errorf("displaying slots = %d, want 2", ringStats["displaying"])
	}
}

func TestSynchronizer_ConcurrentAccess(t *testing.T) {
	ring, err := NewRing(3)
	if err != nil {
		t.Fatalf("NewRing(3) failed: %v", err)
	}

	sync := NewSynchronizer(ring)
	ctx := context.Background()

	const goroutines = 5
	const iterations = 50

	errChan := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			for i := 0; i < iterations; i++ {
				slot, err := sync.AcquireForWriting(ctx)
				if err != nil {
					errChan <- err
					return
				}

				externalID := uint32((id * 1000) + i)
				if err := sync.RegisterBuffer(externalID, slot.Index); err != nil {
					errChan <- err
					return
				}

				if err := sync.MarkDisplaying(slot.Index); err != nil {
					errChan <- err
					return
				}

				if err := sync.OnReleaseEvent(externalID); err != nil {
					errChan <- err
					return
				}

				sync.UnregisterBuffer(externalID)
			}
		}(g)
	}

	// Collect errors
	for i := 0; i < goroutines*iterations; i++ {
		select {
		case err := <-errChan:
			t.Errorf("concurrent access error: %v", err)
		default:
			continue
		}
	}
}
