package wain

import (
	"testing"
	"time"
)

// TestFocusTraversal verifies that Tab and Shift-Tab move focus through
// the focus chain registered with the EventDispatcher.
func TestFocusTraversal(t *testing.T) {
	d := NewEventDispatcher()

	btn1 := &BaseWidget{}
	btn2 := &BaseWidget{}
	btn3 := &BaseWidget{}

	d.focusManager.SetChain([]Widget{btn1, btn2, btn3})
	d.focusManager.Focus(btn1)

	tab := &KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: KeyPress,
		key:       KeyTab,
	}

	// Tab → btn2
	d.Dispatch(tab)
	if got := d.focusManager.Focused(); got != btn2 {
		t.Errorf("after first Tab: focused = %v, want btn2", got)
	}

	// Tab → btn3
	d.Dispatch(tab)
	if got := d.focusManager.Focused(); got != btn3 {
		t.Errorf("after second Tab: focused = %v, want btn3", got)
	}

	// Tab wraps → btn1
	d.Dispatch(tab)
	if got := d.focusManager.Focused(); got != btn1 {
		t.Errorf("after wrap Tab: focused = %v, want btn1", got)
	}

	shiftTab := &KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: KeyPress,
		key:       KeyTab,
		modifiers: ModShift,
	}

	// Shift-Tab from btn1 → wraps to btn3
	d.Dispatch(shiftTab)
	if got := d.focusManager.Focused(); got != btn3 {
		t.Errorf("after Shift-Tab: focused = %v, want btn3", got)
	}
}

// TestEventBubbling verifies that pointer events are dispatched to the
// widget under the pointer via the EventDispatcher.
func TestEventBubbling(t *testing.T) {
	d := NewEventDispatcher()

	root := &BaseWidget{}
	root.SetBounds(0, 0, 100, 100)

	child := &BaseWidget{}
	child.SetBounds(10, 10, 30, 30)
	root.AddChild(child)

	d.SetWidgetRoot(root)

	childReceived := false
	child.OnPointer(func(evt *PointerEvent) {
		childReceived = true
	})

	click := &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerButtonPress,
		x:         20,
		y:         20,
	}

	d.Dispatch(click)

	if !childReceived {
		t.Error("child widget did not receive pointer event in its bounds")
	}
}

// TestEventDispatcherNoHandlers verifies that Dispatch does not panic when no
// handlers are registered.
func TestEventDispatcherNoHandlers(t *testing.T) {
	d := NewEventDispatcher()
	evt := &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerButtonPress,
	}
	// Must not panic
	d.Dispatch(evt)
}
