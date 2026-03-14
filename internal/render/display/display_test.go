package display_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/opd-ai/wain/internal/render/display"
)

// ---------------------------------------------------------------------------
// FramebufferState tests
// ---------------------------------------------------------------------------

func TestFramebufferStateString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		state display.FramebufferState
		want  string
	}{
		{display.FramebufferAvailable, "available"},
		{display.FramebufferRendering, "rendering"},
		{display.FramebufferDisplaying, "displaying"},
		{display.FramebufferState(99), "unknown(99)"},
	}
	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.want {
			t.Errorf("FramebufferState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// FramebufferPool construction
// ---------------------------------------------------------------------------

func TestNewFramebufferPool_Basic(t *testing.T) {
	t.Parallel()

	pool, err := display.NewFramebufferPool(3)
	if err != nil {
		t.Fatalf("NewFramebufferPool(3): %v", err)
	}
	if pool.Size() != 3 {
		t.Errorf("Size = %d, want 3", pool.Size())
	}
}

func TestNewFramebufferPool_InvalidSize(t *testing.T) {
	t.Parallel()

	if _, err := display.NewFramebufferPool(0); err == nil {
		t.Error("expected error for size 0")
	}
	if _, err := display.NewFramebufferPool(-1); err == nil {
		t.Error("expected error for negative size")
	}
}

// ---------------------------------------------------------------------------
// Acquire / MarkDisplaying / OnRelease round-trip
// ---------------------------------------------------------------------------

func TestFramebufferPool_AcquireAndMark(t *testing.T) {
	t.Parallel()

	pool, err := display.NewFramebufferPool(2)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	defer pool.Close() //nolint:errcheck

	ctx := context.Background()

	fb1, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if fb1 == nil {
		t.Fatal("first Acquire returned nil")
	}
	if fb1.State() != display.FramebufferRendering {
		t.Errorf("State after Acquire = %v, want rendering", fb1.State())
	}

	if err := pool.MarkDisplaying(fb1); err != nil {
		t.Fatalf("MarkDisplaying: %v", err)
	}
	if fb1.State() != display.FramebufferDisplaying {
		t.Errorf("State after MarkDisplaying = %v, want displaying", fb1.State())
	}
}

func TestFramebufferPool_ReleaseAndReuse(t *testing.T) {
	t.Parallel()

	pool, err := display.NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	defer pool.Close() //nolint:errcheck

	ctx := context.Background()

	fb, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if err := pool.MarkDisplaying(fb); err != nil {
		t.Fatalf("MarkDisplaying: %v", err)
	}

	// Register so OnRelease can find the buffer by ID.
	pool.Register(fb, 42)
	if err := pool.OnRelease(42); err != nil {
		t.Fatalf("OnRelease: %v", err)
	}
	if fb.State() != display.FramebufferAvailable {
		t.Errorf("State after OnRelease = %v, want available", fb.State())
	}

	// Should be able to acquire the same slot again.
	fb2, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("second Acquire after release: %v", err)
	}
	if fb2 == nil {
		t.Fatal("second Acquire returned nil")
	}
}

func TestFramebufferPool_OnRelease_UnknownID(t *testing.T) {
	t.Parallel()

	pool, err := display.NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	defer pool.Close() //nolint:errcheck

	err = pool.OnRelease(999)
	if !errors.Is(err, display.ErrFramebufferNotFound) {
		t.Errorf("OnRelease with unknown ID: expected ErrFramebufferNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Pool closed behaviour
// ---------------------------------------------------------------------------

func TestFramebufferPool_ClosedAcquire(t *testing.T) {
	t.Parallel()

	pool, err := display.NewFramebufferPool(2)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}

	if err := pool.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Second Close should be a no-op.
	if err := pool.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}

	_, err = pool.Acquire(context.Background())
	if !errors.Is(err, display.ErrPoolClosed) {
		t.Errorf("Acquire on closed pool: expected ErrPoolClosed, got %v", err)
	}
}

func TestFramebufferPool_ClosedMarkDisplaying(t *testing.T) {
	t.Parallel()

	pool, err := display.NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}

	fb, _ := pool.Acquire(context.Background())
	pool.Close() //nolint:errcheck

	err = pool.MarkDisplaying(fb)
	if !errors.Is(err, display.ErrPoolClosed) {
		t.Errorf("MarkDisplaying on closed pool: expected ErrPoolClosed, got %v", err)
	}
}

func TestFramebufferPool_ClosedOnRelease(t *testing.T) {
	t.Parallel()

	pool, err := display.NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}

	fb, _ := pool.Acquire(context.Background())
	pool.Register(fb, 7)
	pool.Close() //nolint:errcheck

	err = pool.OnRelease(7)
	if !errors.Is(err, display.ErrPoolClosed) {
		t.Errorf("OnRelease on closed pool: expected ErrPoolClosed, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Context cancellation during Acquire
// ---------------------------------------------------------------------------

func TestFramebufferPool_AcquireContextCancel(t *testing.T) {
	t.Parallel()

	pool, err := display.NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	defer pool.Close() //nolint:errcheck

	// Exhaust the single slot.
	_, _ = pool.Acquire(context.Background())

	// Now acquiring again should respect context cancellation.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = pool.Acquire(ctx)
	if err == nil {
		t.Error("expected error when context is cancelled, got nil")
	}
}

// ---------------------------------------------------------------------------
// WaitRelease / signalRelease (via Framebuffer.WaitRelease)
// ---------------------------------------------------------------------------

func TestFramebuffer_WaitRelease_AlreadyAvailable(t *testing.T) {
	t.Parallel()

	pool, err := display.NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	defer pool.Close() //nolint:errcheck

	// All slots start as available and pre-signalled.
	fb, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Release back to available via OnRelease.
	pool.Register(fb, 1)
	if err := pool.OnRelease(1); err != nil {
		t.Fatalf("OnRelease: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := fb.WaitRelease(ctx); err != nil {
		t.Errorf("WaitRelease on released framebuffer: %v", err)
	}
}

func TestFramebuffer_WaitRelease_ContextCancel(t *testing.T) {
	t.Parallel()

	// Pool with a single slot: once acquired, WaitRelease blocks until OnRelease.
	// We cancel the context to unblock it.
	pool, err := display.NewFramebufferPool(1)
	if err != nil {
		t.Fatalf("NewFramebufferPool: %v", err)
	}
	defer pool.Close() //nolint:errcheck

	fb, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	// Drain the pre-signalled release channel by waiting with a short deadline.
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer drainCancel()
	_ = fb.WaitRelease(drainCtx)

	// Now the channel is empty and fb is in FramebufferRendering state.
	// WaitRelease should block until context expires.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	if err := fb.WaitRelease(ctx); err == nil {
		t.Error("expected context error when no release is pending, got nil")
	}
}
