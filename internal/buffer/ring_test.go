package buffer

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewRing(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{name: "double buffering", size: 2, wantErr: false},
		{name: "triple buffering", size: 3, wantErr: false},
		{name: "quad buffering", size: 4, wantErr: false},
		{name: "invalid size 0", size: 0, wantErr: true},
		{name: "invalid size 1", size: 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring, err := NewRing(tt.size)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewRing(%d) error = %v, wantErr %v", tt.size, err, tt.wantErr)
			}
			if err == nil {
				if ring.Size() != tt.size {
					t.Errorf("ring.Size() = %d, want %d", ring.Size(), tt.size)
				}
				// Verify all slots are initially available
				stats := ring.Stats()
				if stats["available"] != tt.size {
					t.Errorf("initial available slots = %d, want %d", stats["available"], tt.size)
				}
			}
		})
	}
}

func TestSlotState_String(t *testing.T) {
	tests := []struct {
		state SlotState
		want  string
	}{
		{StateAvailable, "available"},
		{StateRendering, "rendering"},
		{StateDisplaying, "displaying"},
		{StateReleased, "released"},
		{SlotState(999), "unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("SlotState(%d).String() = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestRing_BasicFlow(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	ctx := context.Background()

	// Acquire first slot
	slot1, err := ring.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() failed: %v", err)
	}
	if slot1.State() != StateRendering {
		t.Errorf("slot1.State() = %v, want %v", slot1.State(), StateRendering)
	}

	// Mark first slot as displaying
	if err := ring.MarkDisplaying(slot1.Index); err != nil {
		t.Fatalf("MarkDisplaying(%d) failed: %v", slot1.Index, err)
	}
	if slot1.State() != StateDisplaying {
		t.Errorf("slot1.State() = %v, want %v", slot1.State(), StateDisplaying)
	}

	// Acquire second slot (should succeed immediately)
	slot2, err := ring.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("second AcquireForWriting() failed: %v", err)
	}
	if slot2.Index == slot1.Index {
		t.Errorf("got same slot twice: %d", slot1.Index)
	}

	// Mark first slot as released
	if err := ring.MarkReleased(slot1.Index); err != nil {
		t.Fatalf("MarkReleased(%d) failed: %v", slot1.Index, err)
	}
	if slot1.State() != StateReleased {
		t.Errorf("slot1.State() = %v, want %v", slot1.State(), StateReleased)
	}

	// Complete second slot cycle
	if err := ring.MarkDisplaying(slot2.Index); err != nil {
		t.Fatalf("MarkDisplaying(%d) failed: %v", slot2.Index, err)
	}

	// Should be able to acquire the released slot
	slot3, err := ring.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("third AcquireForWriting() failed: %v", err)
	}
	if slot3.Index != slot1.Index {
		t.Errorf("expected to reacquire slot %d, got %d", slot1.Index, slot3.Index)
	}
}

func TestRing_AcquireTimeout(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	ctx := context.Background()

	// Acquire both slots
	slot1, err := ring.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() failed: %v", err)
	}
	_ = ring.MarkDisplaying(slot1.Index)

	slot2, err := ring.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("second AcquireForWriting() failed: %v", err)
	}
	_ = ring.MarkDisplaying(slot2.Index)

	// Try to acquire with timeout (should fail)
	ctxTimeout, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	_, err = ring.AcquireForWriting(ctxTimeout)
	if err == nil {
		t.Error("AcquireForWriting() should have timed out")
	}
	if ctxTimeout.Err() != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestRing_WaitRelease(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	ctx := context.Background()
	slot, err := ring.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() failed: %v", err)
	}
	_ = ring.MarkDisplaying(slot.Index)

	// Start a goroutine that waits for release
	done := make(chan error, 1)
	go func() {
		waitCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		done <- slot.WaitRelease(waitCtx)
	}()

	// Wait a bit to ensure goroutine is blocked
	time.Sleep(20 * time.Millisecond)

	// Release the slot
	if err := ring.MarkReleased(slot.Index); err != nil {
		t.Fatalf("MarkReleased(%d) failed: %v", slot.Index, err)
	}

	// Check that WaitRelease completed
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("WaitRelease() failed: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("WaitRelease() did not complete")
	}
}

func TestRing_WaitReleaseTimeout(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	ctx := context.Background()
	slot, err := ring.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() failed: %v", err)
	}
	_ = ring.MarkDisplaying(slot.Index)

	// Wait with short timeout (should fail)
	ctxTimeout, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()

	err = slot.WaitRelease(ctxTimeout)
	if err != context.DeadlineExceeded {
		t.Errorf("WaitRelease() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestRing_ConcurrentAcquire(t *testing.T) {
	ring, err := NewRing(3)
	if err != nil {
		t.Fatalf("NewRing(3) failed: %v", err)
	}

	ctx := context.Background()
	const goroutines = 10
	const iterations = 100

	acquired := make(map[int]int)
	var mu sync.Mutex

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				slot, err := ring.AcquireForWriting(ctx)
				if err != nil {
					t.Errorf("AcquireForWriting() failed: %v", err)
					return
				}

				mu.Lock()
				acquired[slot.Index]++
				mu.Unlock()

				// Simulate work
				time.Sleep(time.Microsecond)

				_ = ring.MarkDisplaying(slot.Index)
				_ = ring.MarkReleased(slot.Index)
			}
		}()
	}

	wg.Wait()

	// Verify total acquisitions
	total := 0
	for _, count := range acquired {
		total += count
	}
	expected := goroutines * iterations
	if total != expected {
		t.Errorf("total acquisitions = %d, want %d", total, expected)
	}

	// Verify distribution is reasonable (no slot should be starved)
	for idx, count := range acquired {
		if count < expected/10 {
			t.Errorf("slot %d was acquired only %d times (seems starved)", idx, count)
		}
	}
}

func TestRing_InvalidStateTransitions(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	ctx := context.Background()
	slot, err := ring.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() failed: %v", err)
	}

	// Try to mark as released without marking as displaying first
	err = ring.MarkReleased(slot.Index)
	if err == nil {
		t.Error("MarkReleased() should fail when slot is not displaying")
	}

	// Mark as displaying
	_ = ring.MarkDisplaying(slot.Index)

	// Try to mark as displaying again
	err = ring.MarkDisplaying(slot.Index)
	if err == nil {
		t.Error("MarkDisplaying() should fail when slot is already displaying")
	}
}

func TestRing_Reset(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	ctx := context.Background()

	// Acquire and mark slots in various states
	slot1, _ := ring.AcquireForWriting(ctx)
	_ = ring.MarkDisplaying(slot1.Index)

	slot2, _ := ring.AcquireForWriting(ctx)
	// Leave slot2 in rendering state

	// Reset
	ring.Reset()

	// Verify all slots are available
	stats := ring.Stats()
	if stats["available"] != 2 {
		t.Errorf("after reset, available slots = %d, want 2", stats["available"])
	}
	if slot1.State() != StateAvailable {
		t.Errorf("slot1.State() = %v, want %v", slot1.State(), StateAvailable)
	}
	if slot2.State() != StateAvailable {
		t.Errorf("slot2.State() = %v, want %v", slot2.State(), StateAvailable)
	}
}

func TestRing_UserData(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	ctx := context.Background()
	slot, err := ring.AcquireForWriting(ctx)
	if err != nil {
		t.Fatalf("AcquireForWriting() failed: %v", err)
	}

	// Attach user data
	type testData struct {
		bufferID uint32
		pixmap   uint32
	}
	data := &testData{bufferID: 12345, pixmap: 67890}
	slot.UserData = data

	// Verify we can retrieve it
	retrieved, ok := slot.UserData.(*testData)
	if !ok {
		t.Fatal("failed to retrieve UserData as *testData")
	}
	if retrieved.bufferID != 12345 || retrieved.pixmap != 67890 {
		t.Errorf("UserData = %+v, want {bufferID:12345 pixmap:67890}", retrieved)
	}
}

func TestRing_GetSlot_Bounds(t *testing.T) {
	ring, err := NewRing(3)
	if err != nil {
		t.Fatalf("NewRing(3) failed: %v", err)
	}

	// Valid indices
	for i := 0; i < 3; i++ {
		slot, err := ring.GetSlot(i)
		if err != nil {
			t.Errorf("GetSlot(%d) unexpected error: %v", i, err)
		}
		if slot.Index != i {
			t.Errorf("GetSlot(%d).Index = %d, want %d", i, slot.Index, i)
		}
	}

	// Out of bounds (should return error)
	_, err = ring.GetSlot(-1)
	if err == nil {
		t.Error("GetSlot(-1) should return error")
	}
}

func TestRing_GetSlot_OutOfBoundsUpper(t *testing.T) {
	ring, err := NewRing(2)
	if err != nil {
		t.Fatalf("NewRing(2) failed: %v", err)
	}

	_, err = ring.GetSlot(2)
	if err == nil {
		t.Error("GetSlot(2) should return error for size=2 ring")
	}
}

func TestRing_Stats(t *testing.T) {
	ring, err := NewRing(4)
	if err != nil {
		t.Fatalf("NewRing(4) failed: %v", err)
	}

	ctx := context.Background()

	// Initial state: all available
	stats := ring.Stats()
	if stats["available"] != 4 {
		t.Errorf("initial stats[available] = %d, want 4", stats["available"])
	}

	// Acquire one
	slot1, _ := ring.AcquireForWriting(ctx)
	stats = ring.Stats()
	if stats["rendering"] != 1 || stats["available"] != 3 {
		t.Errorf("after acquire: stats = %v, want {rendering:1 available:3}", stats)
	}

	// Mark displaying
	_ = ring.MarkDisplaying(slot1.Index)
	stats = ring.Stats()
	if stats["displaying"] != 1 || stats["available"] != 3 {
		t.Errorf("after displaying: stats = %v, want {displaying:1 available:3}", stats)
	}

	// Mark released
	_ = ring.MarkReleased(slot1.Index)
	stats = ring.Stats()
	if stats["released"] != 1 || stats["available"] != 3 {
		t.Errorf("after released: stats = %v, want {released:1 available:3}", stats)
	}
}

func BenchmarkRing_AcquireRelease(b *testing.B) {
	ring, err := NewRing(3)
	if err != nil {
		b.Fatalf("NewRing(3) failed: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slot, err := ring.AcquireForWriting(ctx)
		if err != nil {
			b.Fatalf("AcquireForWriting() failed: %v", err)
		}
		_ = ring.MarkDisplaying(slot.Index)
		_ = ring.MarkReleased(slot.Index)
	}
}

func BenchmarkRing_ConcurrentAcquireRelease(b *testing.B) {
	ring, err := NewRing(3)
	if err != nil {
		b.Fatalf("NewRing(3) failed: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			slot, err := ring.AcquireForWriting(ctx)
			if err != nil {
				b.Fatalf("AcquireForWriting() failed: %v", err)
			}
			_ = ring.MarkDisplaying(slot.Index)
			_ = ring.MarkReleased(slot.Index)
		}
	})
}
