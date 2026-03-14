// Animation types and public API for the wain UI toolkit.
//
// This file exposes the internal animation package as part of the wain public API.
package wain

import (
	"time"

	"github.com/opd-ai/wain/internal/ui/animation"
)

// EasingFunc maps a normalised time value t ∈ [0, 1] to an animation progress
// value. The identity (linear) easing is f(t) = t.
type EasingFunc = animation.EasingFunc

// Animation describes a single running property animation.
// Use [App.Animate] to schedule an animation, and [Animation.OnComplete] or
// [Animation.Cancel] to control it.
type Animation = animation.Animation

// Easing function constants — pass these to [App.Animate].
var (
	// AnimateLinear is a constant-rate easing function.
	AnimateLinear EasingFunc = animation.Linear
	// AnimateEaseIn starts slowly and accelerates (cubic).
	AnimateEaseIn EasingFunc = animation.EaseIn
	// AnimateEaseOut starts fast and decelerates (cubic).
	AnimateEaseOut EasingFunc = animation.EaseOut
	// AnimateEaseInOut is slow at both ends and fast in the middle (cubic Hermite).
	AnimateEaseInOut EasingFunc = animation.EaseInOut
	// AnimateSpring overshoots the target and then settles.
	AnimateSpring EasingFunc = animation.Spring
)

// Animate schedules a property animation driven by the app's frame loop.
//
// from and to are the start and end values; dur is the animation duration;
// easing controls the rate of change (pass nil for [AnimateLinear]). onTick is
// called every frame with the current interpolated value.
//
// The returned [*Animation] can be used to register an [Animation.OnComplete]
// callback or to [Animation.Cancel] the animation early.
//
// Example — fade a widget in over 300 ms:
//
//	a := app.Animate(0, 1, 300*time.Millisecond, wain.AnimateEaseInOut,
//	    func(v float64) { widget.SetOpacity(v) })
//	a.OnComplete(func() { fmt.Println("fade done") })
func (a *App) Animate(from, to float64, dur time.Duration, easing EasingFunc, onTick func(float64)) *Animation {
	anim := &animation.Animation{
		From:     from,
		To:       to,
		Duration: dur,
		Easing:   easing,
		OnTick:   onTick,
	}
	a.animator.Add(anim)
	return anim
}
