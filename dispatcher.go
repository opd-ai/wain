package wain

import (
	"sync"
)

// EventDispatcher manages event routing from platform sources to handlers.
type EventDispatcher struct {
	mu sync.RWMutex

	// Event handlers by type
	pointerHandlers []func(*PointerEvent)
	keyHandlers     []func(*KeyEvent)
	touchHandlers   []func(*TouchEvent)
	windowHandlers  []func(*WindowEvent)
	customHandlers  []func(*CustomEvent)

	// Focus management
	focusManager *FocusManager

	// Widget tree for hit-testing
	widgetRoot Widget
}

// NewEventDispatcher creates a new event dispatcher.
func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		focusManager: NewFocusManager(),
	}
}

// SetWidgetRoot sets the root widget for hit-testing.
func (d *EventDispatcher) SetWidgetRoot(root Widget) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.widgetRoot = root
}

// OnPointer registers a pointer event handler.
func (d *EventDispatcher) OnPointer(handler func(*PointerEvent)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pointerHandlers = append(d.pointerHandlers, handler)
}

// OnKey registers a keyboard event handler.
func (d *EventDispatcher) OnKey(handler func(*KeyEvent)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.keyHandlers = append(d.keyHandlers, handler)
}

// OnTouch registers a touch event handler.
func (d *EventDispatcher) OnTouch(handler func(*TouchEvent)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.touchHandlers = append(d.touchHandlers, handler)
}

// OnWindow registers a window event handler.
func (d *EventDispatcher) OnWindow(handler func(*WindowEvent)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.windowHandlers = append(d.windowHandlers, handler)
}

// OnCustom registers a custom event handler.
func (d *EventDispatcher) OnCustom(handler func(*CustomEvent)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.customHandlers = append(d.customHandlers, handler)
}

// Dispatch routes an event to appropriate handlers.
func (d *EventDispatcher) Dispatch(evt Event) {
	if evt == nil {
		return
	}

	switch e := evt.(type) {
	case *PointerEvent:
		d.dispatchPointer(e)
	case *KeyEvent:
		d.dispatchKey(e)
	case *TouchEvent:
		d.dispatchTouch(e)
	case *WindowEvent:
		d.dispatchWindow(e)
	case *CustomEvent:
		d.dispatchCustom(e)
	}
}

func (d *EventDispatcher) dispatchPointer(evt *PointerEvent) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Hit-test against widget tree
	if d.widgetRoot != nil && (evt.eventType == PointerButtonPress ||
		evt.eventType == PointerButtonRelease || evt.eventType == PointerMove) {
		if target := d.hitTest(d.widgetRoot, evt.x, evt.y); target != nil {
			target.HandlePointer(evt)
			if evt.Consumed() {
				return
			}
		}
	}

	// Dispatch to registered handlers
	for _, handler := range d.pointerHandlers {
		if evt.Consumed() {
			return
		}
		handler(evt)
	}
}

func (d *EventDispatcher) dispatchKey(evt *KeyEvent) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Handle Tab navigation for focus
	if evt.eventType == KeyPress {
		if evt.key == KeyTab {
			if evt.modifiers&ModShift != 0 {
				d.focusManager.FocusPrev()
			} else {
				d.focusManager.FocusNext()
			}
			evt.Consume()
			return
		}
	}

	// Dispatch to focused widget first
	if focused := d.focusManager.Focused(); focused != nil {
		focused.HandleKey(evt)
		if evt.Consumed() {
			return
		}
	}

	// Dispatch to registered handlers
	for _, handler := range d.keyHandlers {
		if evt.Consumed() {
			return
		}
		handler(evt)
	}
}

func (d *EventDispatcher) dispatchTouch(evt *TouchEvent) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Hit-test against widget tree
	if d.widgetRoot != nil && (evt.eventType == TouchDown || evt.eventType == TouchMotion) {
		if target := d.hitTest(d.widgetRoot, evt.x, evt.y); target != nil {
			target.HandleTouch(evt)
			if evt.Consumed() {
				return
			}
		}
	}

	// Dispatch to registered handlers
	for _, handler := range d.touchHandlers {
		if evt.Consumed() {
			return
		}
		handler(evt)
	}
}

func (d *EventDispatcher) dispatchWindow(evt *WindowEvent) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Dispatch to registered handlers
	for _, handler := range d.windowHandlers {
		if evt.Consumed() {
			return
		}
		handler(evt)
	}
}

func (d *EventDispatcher) dispatchCustom(evt *CustomEvent) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Dispatch to registered handlers
	for _, handler := range d.customHandlers {
		if evt.Consumed() {
			return
		}
		handler(evt)
	}
}

// hitTest performs recursive hit-testing on widget tree.
func (d *EventDispatcher) hitTest(w Widget, x, y float64) Widget {
	if !w.Contains(x, y) {
		return nil
	}

	// Check children first (front-to-back order)
	for i := len(w.Children()) - 1; i >= 0; i-- {
		if child := d.hitTest(w.Children()[i], x, y); child != nil {
			return child
		}
	}

	// No child hit, return this widget
	return w
}

// FocusManager manages keyboard focus and tab order.
type FocusManager struct {
	mu sync.RWMutex

	// Focus chain (tab order)
	chain []Widget

	// Currently focused widget index
	focusedIdx int
}

// NewFocusManager creates a new focus manager.
func NewFocusManager() *FocusManager {
	return &FocusManager{
		focusedIdx: -1,
	}
}

// SetChain sets the focus chain (tab order).
func (fm *FocusManager) SetChain(widgets []Widget) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.chain = widgets
	fm.focusedIdx = -1
}

// Focus sets focus to a specific widget.
func (fm *FocusManager) Focus(w Widget) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	for i, widget := range fm.chain {
		if widget == w {
			fm.focusedIdx = i
			widget.SetFocused(true)
			return
		}
	}
}

// FocusNext moves focus to the next widget in the chain.
func (fm *FocusManager) FocusNext() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if len(fm.chain) == 0 {
		return
	}

	// Clear current focus
	if fm.focusedIdx >= 0 && fm.focusedIdx < len(fm.chain) {
		fm.chain[fm.focusedIdx].SetFocused(false)
	}

	// Move to next
	fm.focusedIdx = (fm.focusedIdx + 1) % len(fm.chain)
	fm.chain[fm.focusedIdx].SetFocused(true)
}

// FocusPrev moves focus to the previous widget in the chain.
func (fm *FocusManager) FocusPrev() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if len(fm.chain) == 0 {
		return
	}

	// Clear current focus
	if fm.focusedIdx >= 0 && fm.focusedIdx < len(fm.chain) {
		fm.chain[fm.focusedIdx].SetFocused(false)
	}

	// Move to previous
	fm.focusedIdx--
	if fm.focusedIdx < 0 {
		fm.focusedIdx = len(fm.chain) - 1
	}
	fm.chain[fm.focusedIdx].SetFocused(true)
}

// Focused returns the currently focused widget.
func (fm *FocusManager) Focused() Widget {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if fm.focusedIdx >= 0 && fm.focusedIdx < len(fm.chain) {
		return fm.chain[fm.focusedIdx]
	}
	return nil
}

// ClearFocus removes focus from all widgets.
func (fm *FocusManager) ClearFocus() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if fm.focusedIdx >= 0 && fm.focusedIdx < len(fm.chain) {
		fm.chain[fm.focusedIdx].SetFocused(false)
	}
	fm.focusedIdx = -1
}
