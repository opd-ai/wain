package animation

import (
	"math"
	"testing"
	"time"
)

func TestAnimationLinear(t *testing.T) {
	var got []float64
	a := &Animation{
		From:     0,
		To:       100,
		Duration: 100 * time.Millisecond,
		Easing:   Linear,
		OnTick:   func(v float64) { got = append(got, v) },
	}

	anim := New()
	anim.Add(a)

	// Advance in 10 ms steps; 10 ticks should reach the end.
	for i := 0; i < 10; i++ {
		anim.Tick(10 * time.Millisecond)
	}

	if len(got) < 10 {
		t.Fatalf("expected ≥10 ticks, got %d", len(got))
	}
	// After 10 × 10 ms = 100 ms, animation should complete at 100.
	if got[len(got)-1] != 100 {
		t.Errorf("final value = %v, want 100", got[len(got)-1])
	}
	// No more animations running.
	if anim.Running() != 0 {
		t.Errorf("Running = %d after completion, want 0", anim.Running())
	}
}

func TestAnimationEaseIn(t *testing.T) {
	var midpoint float64
	a := &Animation{
		From:     0,
		To:       1,
		Duration: 100 * time.Millisecond,
		Easing:   EaseIn,
		OnTick:   func(v float64) { midpoint = v },
	}
	anim := New()
	anim.Add(a)
	anim.Tick(50 * time.Millisecond) // t=0.5
	// EaseIn(0.5) = 0.5^3 = 0.125 — slower than linear at the start
	if midpoint >= 0.5 {
		t.Errorf("EaseIn at t=0.5 should be < 0.5, got %v", midpoint)
	}
}

func TestAnimationEaseOut(t *testing.T) {
	var midpoint float64
	a := &Animation{
		From:     0,
		To:       1,
		Duration: 100 * time.Millisecond,
		Easing:   EaseOut,
		OnTick:   func(v float64) { midpoint = v },
	}
	anim := New()
	anim.Add(a)
	anim.Tick(50 * time.Millisecond) // t=0.5
	// EaseOut(0.5) = 0.875 — faster than linear at the start
	if midpoint <= 0.5 {
		t.Errorf("EaseOut at t=0.5 should be > 0.5, got %v", midpoint)
	}
}

func TestAnimationSpring(t *testing.T) {
	var peak float64
	a := &Animation{
		From:     0,
		To:       1,
		Duration: 100 * time.Millisecond,
		Easing:   Spring,
		OnTick: func(v float64) {
			if v > peak {
				peak = v
			}
		},
	}
	anim := New()
	anim.Add(a)
	for i := 0; i < 10; i++ {
		anim.Tick(10 * time.Millisecond)
	}
	// Spring should overshoot past 1.0 at some point.
	if peak <= 1.0 {
		t.Errorf("Spring easing peak=%v — expected overshoot > 1.0", peak)
	}
}

func TestAnimationOnComplete(t *testing.T) {
	called := false
	a := &Animation{
		From:     0,
		To:       1,
		Duration: 50 * time.Millisecond,
	}
	a.OnComplete(func() { called = true })

	anim := New()
	anim.Add(a)
	anim.Tick(100 * time.Millisecond) // complete in one tick

	if !called {
		t.Error("OnComplete was not called after animation finished")
	}
}

func TestAnimationCancel(t *testing.T) {
	completeCalled := false
	tickCalled := false
	a := &Animation{
		From:     0,
		To:       1,
		Duration: 200 * time.Millisecond,
		OnTick:   func(_ float64) { tickCalled = true },
	}
	a.OnComplete(func() { completeCalled = true })

	anim := New()
	anim.Add(a)
	anim.Tick(10 * time.Millisecond) // advance once
	a.Cancel()
	anim.Tick(10 * time.Millisecond) // should be removed

	if completeCalled {
		t.Error("OnComplete must NOT be called after Cancel")
	}
	if anim.Running() != 0 {
		t.Errorf("Running = %d after Cancel, want 0", anim.Running())
	}
	_ = tickCalled
}

func TestAnimationZeroDuration(t *testing.T) {
	var got float64
	a := &Animation{
		From:   5,
		To:     10,
		OnTick: func(v float64) { got = v },
	}
	anim := New()
	anim.Add(a)
	anim.Tick(1 * time.Millisecond)

	if got != 10 {
		t.Errorf("zero-duration animation: got=%v, want 10", got)
	}
	if anim.Running() != 0 {
		t.Errorf("Running = %d after zero-duration animation, want 0", anim.Running())
	}
}

func TestAnimatorMultiple(t *testing.T) {
	var v1, v2 float64
	anim := New()
	anim.Add(&Animation{From: 0, To: 10, Duration: 50 * time.Millisecond, OnTick: func(v float64) { v1 = v }})
	anim.Add(&Animation{From: 0, To: 20, Duration: 100 * time.Millisecond, OnTick: func(v float64) { v2 = v }})

	anim.Tick(50 * time.Millisecond)

	// First animation should be done (v1==10), second still running.
	if v1 != 10 {
		t.Errorf("v1=%v, want 10", v1)
	}
	if v2 >= 20 {
		t.Errorf("v2=%v should be < 20 at halfway", v2)
	}
	if anim.Running() != 1 {
		t.Errorf("Running=%d, want 1 (second animation still live)", anim.Running())
	}
}

func TestEasingFunctions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		easing EasingFunc
	}{
		{"Linear", Linear},
		{"EaseIn", EaseIn},
		{"EaseOut", EaseOut},
		{"EaseInOut", EaseInOut},
		{"Spring", Spring},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// f(0) must be 0 for non-Spring easings.
			if tc.name != "Spring" {
				if got := tc.easing(0); math.Abs(got) > 1e-9 {
					t.Errorf("%s(0) = %v, want 0", tc.name, got)
				}
			}
			// f(1) must be 1 for all easings.
			if got := tc.easing(1); math.Abs(got-1) > 1e-9 {
				t.Errorf("%s(1) = %v, want 1", tc.name, got)
			}
		})
	}
}
