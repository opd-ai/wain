// Package animation provides a goroutine-free property animation system for the
// wain UI toolkit.
//
// Animations are driven by calling [Animator.Tick] on every frame from the
// render loop, passing the elapsed time since the previous frame. No timers or
// goroutines are used; the system is deterministic and safe to call from a
// single thread.
//
// # Usage
//
// Create an [Animator], schedule [Animation] values via [Animator.Add], and
// call [Animator.Tick] once per frame:
//
//	anim := animation.New()
//	a := &animation.Animation{
//	    From:     0,
//	    To:       1,
//	    Duration: 300 * time.Millisecond,
//	    Easing:   animation.EaseInOut,
//	    OnTick:   func(v float64) { widget.SetOpacity(v) },
//	}
//	anim.Add(a)
//
//	// in the render loop:
//	anim.Tick(dt)
package animation

import "time"

// EasingFunc maps normalised time t ∈ [0, 1] to a progress value.
// The identity (linear) easing is f(t) = t.
type EasingFunc func(t float64) float64

// Built-in easing functions.
var (
	// Linear easing — constant rate of change.
	Linear EasingFunc = func(t float64) float64 { return t }

	// EaseIn — slow start, fast finish (cubic).
	EaseIn EasingFunc = func(t float64) float64 { return t * t * t }

	// EaseOut — fast start, slow finish (cubic).
	EaseOut EasingFunc = func(t float64) float64 {
		u := t - 1
		return u*u*u + 1
	}

	// EaseInOut — slow start and finish, fast middle (cubic Hermite).
	EaseInOut EasingFunc = func(t float64) float64 {
		return t * t * (3 - 2*t)
	}

	// Spring — overshoots the target slightly then settles (back easing).
	Spring EasingFunc = func(t float64) float64 {
		const c1 = 1.70158
		const c3 = c1 + 1
		return 1 + c3*(t-1)*(t-1)*(t-1) + c1*(t-1)*(t-1)
	}
)

// Animation describes a single property animation.
type Animation struct {
	// From is the starting value of the animated property.
	From float64
	// To is the target value of the animated property.
	To float64
	// Duration is how long the animation should run.
	Duration time.Duration
	// Easing controls the rate of change over the animation's lifetime.
	// Defaults to [Linear] when nil.
	Easing EasingFunc
	// OnTick is called on every frame with the current interpolated value.
	OnTick func(v float64)
	// onComplete is called once when the animation finishes or is cancelled.
	onComplete func()

	elapsed  time.Duration
	done     bool
	canceled bool
}

// OnComplete registers a callback to be invoked when the animation finishes.
// Only the most recently registered callback is kept.
func (a *Animation) OnComplete(fn func()) {
	a.onComplete = fn
}

// Cancel stops the animation immediately without calling OnComplete.
func (a *Animation) Cancel() {
	a.canceled = true
	a.done = true
}

// advance moves the animation forward by dt and returns the current value.
// Returns done=true on the frame the animation completes.
func (a *Animation) advance(dt time.Duration) (value float64, done bool) {
	if a.done {
		return a.To, true
	}
	a.elapsed += dt
	easing := a.Easing
	if easing == nil {
		easing = Linear
	}
	if a.Duration <= 0 || a.elapsed >= a.Duration {
		a.done = true
		return a.To, true
	}
	t := float64(a.elapsed) / float64(a.Duration)
	value = a.From + (a.To-a.From)*easing(t)
	return value, false
}

// Animator manages a collection of running animations and drives them via Tick.
type Animator struct {
	animations []*Animation
}

// New creates a new Animator with no running animations.
func New() *Animator {
	return &Animator{}
}

// Add enqueues an animation. The animation starts advancing on the next Tick.
func (a *Animator) Add(anim *Animation) {
	a.animations = append(a.animations, anim)
}

// Tick advances all running animations by dt and calls their OnTick callbacks.
// Completed animations are removed and their OnComplete callbacks are invoked.
//
// Call Tick once per frame from the render loop, passing the elapsed time since
// the previous frame.
func (a *Animator) Tick(dt time.Duration) {
	live := a.animations[:0]
	for _, anim := range a.animations {
		if anim.canceled {
			continue
		}
		value, done := anim.advance(dt)
		if anim.OnTick != nil {
			anim.OnTick(value)
		}
		if done {
			if !anim.canceled && anim.onComplete != nil {
				anim.onComplete()
			}
		} else {
			live = append(live, anim)
		}
	}
	a.animations = live
}

// Running returns the number of currently active animations.
func (a *Animator) Running() int {
	return len(a.animations)
}
