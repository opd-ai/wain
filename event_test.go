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

// TestDispatcherOnPointer verifies that OnPointer handler is called.
func TestDispatcherOnPointer(t *testing.T) {
	d := NewEventDispatcher()
	called := false
	d.OnPointer(func(e *PointerEvent) { called = true })

	d.Dispatch(&PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerButtonPress,
	})
	if !called {
		t.Error("OnPointer handler not called")
	}
}

// TestDispatcherOnKey verifies that OnKey handler is called for non-Tab keys.
func TestDispatcherOnKey(t *testing.T) {
	d := NewEventDispatcher()
	called := false
	d.OnKey(func(e *KeyEvent) { called = true })

	d.Dispatch(&KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: KeyPress,
		key:       Key(0x61), // 'a'
	})
	if !called {
		t.Error("OnKey handler not called")
	}
}

// TestDispatcherOnTouch verifies that OnTouch handler is called.
func TestDispatcherOnTouch(t *testing.T) {
	d := NewEventDispatcher()
	called := false
	d.OnTouch(func(e *TouchEvent) { called = true })

	d.Dispatch(&TouchEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: TouchDown,
	})
	if !called {
		t.Error("OnTouch handler not called")
	}
}

// TestDispatcherOnWindow verifies that OnWindow handler is called.
func TestDispatcherOnWindow(t *testing.T) {
	d := NewEventDispatcher()
	called := false
	d.OnWindow(func(e *WindowEvent) { called = true })

	d.Dispatch(&WindowEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: WindowResize,
	})
	if !called {
		t.Error("OnWindow handler not called")
	}
}

// TestDispatcherOnCustom verifies that OnCustom handler is called.
func TestDispatcherOnCustom(t *testing.T) {
	d := NewEventDispatcher()
	var received CustomEventPayload
	d.OnCustom(func(e *CustomEvent) { received = e.Data() })

	type msg struct{ text string }
	payload := msg{"hello"}
	d.Dispatch(&CustomEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		data:      payload,
	})
	if received != payload {
		t.Errorf("OnCustom handler received %v, want %v", received, payload)
	}
}

// TestDispatcherDispatchNil verifies that Dispatch(nil) doesn't panic.
func TestDispatcherDispatchNil(t *testing.T) {
	d := NewEventDispatcher()
	d.Dispatch(nil) // must not panic
}

// TestDispatcherKeyToFocusedWidget verifies that key events reach the focused widget.
func TestDispatcherKeyToFocusedWidget(t *testing.T) {
	d := NewEventDispatcher()

	btn := &BaseWidget{}
	received := false
	btn.OnKey(func(e *KeyEvent) { received = true })

	d.focusManager.SetChain([]Widget{btn})
	d.focusManager.Focus(btn)

	d.Dispatch(&KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: KeyPress,
		key:       Key(0x41), // 'A'
	})
	if !received {
		t.Error("focused widget did not receive key event")
	}
}

// TestDispatcherSetFocusChangeHook verifies that FocusChangeHook is triggered.
func TestDispatcherSetFocusChangeHook(t *testing.T) {
	d := NewEventDispatcher()
	var hooked Widget

	d.focusManager.SetFocusChangeHook(func(w Widget) {
		hooked = w
	})

	btn := &BaseWidget{}
	d.focusManager.SetChain([]Widget{btn})
	d.focusManager.Focus(btn)

	if hooked != btn {
		t.Errorf("hook called with %v, want btn", hooked)
	}
}

// TestFocusManagerClearFocus verifies that ClearFocus removes the focused widget.
func TestFocusManagerClearFocus(t *testing.T) {
	fm := NewFocusManager()
	btn := &BaseWidget{}
	fm.SetChain([]Widget{btn})
	fm.Focus(btn)
	fm.ClearFocus()

	if fm.Focused() != nil {
		t.Errorf("after ClearFocus, Focused() = %v, want nil", fm.Focused())
	}
}

// TestDispatcherPointerConsumed verifies that a consumed pointer event stops propagation.
func TestDispatcherPointerConsumed(t *testing.T) {
	d := NewEventDispatcher()

	first := false
	second := false

	d.OnPointer(func(e *PointerEvent) {
		first = true
		e.Consume()
	})
	d.OnPointer(func(e *PointerEvent) {
		second = true
	})

	d.Dispatch(&PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerButtonPress,
	})

	if !first {
		t.Error("first handler was not called")
	}
	if second {
		t.Error("second handler was called despite event being consumed")
	}
}

// TestEventTimestamp verifies that baseEvent.Timestamp() returns the set value.
func TestEventTimestamp(t *testing.T) {
	now := time.Now()
	e := &PointerEvent{baseEvent: baseEvent{timestamp: now}}
	if got := e.Timestamp(); !got.Equal(now) {
		t.Errorf("Timestamp() = %v, want %v", got, now)
	}
}

// TestPointerEventAccessors verifies X(), Y(), Button(), Type() accessors.
func TestPointerEventAccessors(t *testing.T) {
	e := &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerButtonPress,
		x:         12.5,
		y:         34.5,
		button:    PointerButtonLeft,
	}
	if e.X() != 12.5 {
		t.Errorf("X() = %v, want 12.5", e.X())
	}
	if e.Y() != 34.5 {
		t.Errorf("Y() = %v, want 34.5", e.Y())
	}
	if e.Button() != PointerButtonLeft {
		t.Errorf("Button() = %v, want ButtonLeft", e.Button())
	}
	if e.Type() != EventTypePointer {
		t.Errorf("Type() = %v, want EventTypePointer", e.Type())
	}
}

// TestKeyEventAccessors verifies Key(), Modifiers(), EventType() accessors.
func TestKeyEventAccessors(t *testing.T) {
	e := &KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: KeyRelease,
		key:       KeyTab,
		modifiers: ModShift,
	}
	if e.Key() != KeyTab {
		t.Errorf("Key() = %v, want KeyTab", e.Key())
	}
	if e.Modifiers() != ModShift {
		t.Errorf("Modifiers() = %v, want ModShift", e.Modifiers())
	}
	if e.EventType() != KeyRelease {
		t.Errorf("EventType() = %v, want KeyRelease", e.EventType())
	}
	if e.Type() != EventTypeKey {
		t.Errorf("Type() = %v, want EventTypeKey", e.Type())
	}
}

// TestTouchEventAccessors verifies Touch event accessors.
func TestTouchEventAccessors(t *testing.T) {
	e := &TouchEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: TouchDown,
		id:        3,
		x:         5.0,
		y:         7.0,
	}
	if e.ID() != 3 {
		t.Errorf("ID() = %d, want 3", e.ID())
	}
	if e.X() != 5.0 {
		t.Errorf("X() = %v, want 5.0", e.X())
	}
	if e.Y() != 7.0 {
		t.Errorf("Y() = %v, want 7.0", e.Y())
	}
	if e.Type() != EventTypeTouch {
		t.Errorf("Type() = %v, want EventTypeTouch", e.Type())
	}
	if e.EventType() != TouchDown {
		t.Errorf("EventType() = %v, want TouchDown", e.EventType())
	}
}

// TestWindowEventAccessors verifies WindowEvent accessors.
func TestWindowEventAccessors(t *testing.T) {
	e := &WindowEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: WindowResize,
		width:     800,
		height:    600,
	}
	if e.Width() != 800 {
		t.Errorf("Width() = %d, want 800", e.Width())
	}
	if e.Height() != 600 {
		t.Errorf("Height() = %d, want 600", e.Height())
	}
	if e.EventType() != WindowResize {
		t.Errorf("EventType() = %v, want WindowResize", e.EventType())
	}
	if e.Type() != EventTypeWindow {
		t.Errorf("Type() = %v, want EventTypeWindow", e.Type())
	}
}

// TestDispatchTouchHitTestPath exercises dispatchTouch with a widgetRoot.
func TestDispatchTouchHitTestPath(t *testing.T) {
	d := NewEventDispatcher()

	root := &BaseWidget{}
	root.SetBounds(0, 0, 200, 200)

	touchHandled := false
	root.OnTouch(func(e *TouchEvent) {
		touchHandled = true
	})

	d.SetWidgetRoot(root)

	evt := &TouchEvent{eventType: TouchDown, x: 100, y: 100}
	d.Dispatch(evt)
	if !touchHandled {
		t.Error("touch handler not called via hit-test path")
	}
}

// TestDispatchKeyToFocusedWidgetCallback exercises key dispatch to focused widget.
func TestDispatchKeyToFocusedWidgetCallback(t *testing.T) {
	d := NewEventDispatcher()

	root := &BaseWidget{}
	root.SetBounds(0, 0, 200, 200)

	var receivedKey *KeyEvent
	root.OnKey(func(e *KeyEvent) {
		receivedKey = e
	})

	d.SetWidgetRoot(root)
	d.focusManager.SetChain([]Widget{root})
	d.focusManager.Focus(root)

	evt := &KeyEvent{eventType: KeyPress, key: KeyReturn}
	d.Dispatch(evt)

	if receivedKey == nil {
		t.Error("key event not dispatched to focused widget")
	}
}

// TestPointerEventAxis verifies Axis() and Value() on scroll events.
func TestPointerEventAxis(t *testing.T) {
	evt := &PointerEvent{
		eventType: PointerScroll,
		axis:      ScrollAxisVertical,
		value:     3.0,
	}
	if evt.Axis() != ScrollAxisVertical {
		t.Errorf("Axis() = %v, want ScrollAxisVertical", evt.Axis())
	}
	if evt.Value() != 3.0 {
		t.Errorf("Value() = %v, want 3.0", evt.Value())
	}
}

// TestWindowEventScale verifies Scale() on a scale-change event.
func TestWindowEventScale(t *testing.T) {
	evt := &WindowEvent{eventType: WindowScaleChange, scale: 2.0}
	if evt.Scale() != 2.0 {
		t.Errorf("Scale() = %v, want 2.0", evt.Scale())
	}
}

// TestCustomEventType verifies CustomEvent.Type() returns EventTypeCustom.
func TestCustomEventType(t *testing.T) {
	evt := &CustomEvent{data: "hello"}
	if evt.Type() != EventTypeCustom {
		t.Errorf("Type() = %v, want EventTypeCustom", evt.Type())
	}
}

// TestDragEventType verifies DragEvent.Type() returns EventTypeDrag.
func TestDragEventType(t *testing.T) {
	evt := newDragEvent(DragEnter, 1.0, 2.0, []string{"text/plain"})
	if evt.Type() != EventTypeDrag {
		t.Errorf("Type() = %v, want EventTypeDrag", evt.Type())
	}
	if evt.MimeTypes()[0] != "text/plain" {
		t.Errorf("MimeTypes() = %v", evt.MimeTypes())
	}
}

// TestFocusNextEmptyChain verifies FocusNext is a no-op with empty chain.
func TestFocusNextEmptyChain(t *testing.T) {
	fm := NewFocusManager()
	fm.FocusNext() // must not panic
}

// TestFocusPrevEmptyChain verifies FocusPrev is a no-op with empty chain.
func TestFocusPrevEmptyChain(t *testing.T) {
	fm := NewFocusManager()
	fm.FocusPrev() // must not panic
}

// TestDispatchPointerConsumedByWidget verifies dispatchPointer stops when widget consumes.
func TestDispatchPointerConsumedByWidget(t *testing.T) {
	d := NewEventDispatcher()

	root := &BaseWidget{}
	root.SetBounds(0, 0, 100, 100)
	d.SetWidgetRoot(root)

	handlerCalled := false
	root.OnPointer(func(evt *PointerEvent) {
		evt.Consume()
	})

	d.OnPointer(func(evt *PointerEvent) {
		handlerCalled = true
	})

	click := &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerButtonPress,
		x:         50,
		y:         50,
	}
	d.Dispatch(click)
	if handlerCalled {
		t.Error("registered handler should not be called when widget consumed event")
	}
}

// TestDispatchKeyHandlerConsumed verifies dispatchToKeyHandlers stops when consumed.
func TestDispatchKeyHandlerConsumed(t *testing.T) {
	d := NewEventDispatcher()

	secondCalled := false
	d.OnKey(func(evt *KeyEvent) {
		evt.Consume()
	})
	d.OnKey(func(evt *KeyEvent) {
		secondCalled = true
	})

	keyEvt := &KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: KeyPress,
		key:       KeyReturn,
	}
	d.Dispatch(keyEvt)
	if secondCalled {
		t.Error("second key handler should not be called after event consumed")
	}
}
