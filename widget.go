package wain

// Widget is the interface that all UI widgets must implement for event handling.
type Widget interface {
	// Contains returns true if the point (x, y) is inside the widget's bounds.
	Contains(x, y float64) bool

	// Children returns the widget's child widgets.
	Children() []Widget

	// HandlePointer processes a pointer event.
	HandlePointer(evt *PointerEvent)

	// HandleKey processes a keyboard event.
	HandleKey(evt *KeyEvent)

	// HandleTouch processes a touch event.
	HandleTouch(evt *TouchEvent)

	// SetFocused sets the widget's focus state.
	SetFocused(focused bool)

	// IsFocused returns true if the widget has keyboard focus.
	IsFocused() bool
}

// BaseWidget provides default implementations for Widget interface.
type BaseWidget struct {
	x, y          float64
	width, height float64
	children      []Widget
	focused       bool

	// Event callbacks
	onPointer func(*PointerEvent)
	onKey     func(*KeyEvent)
	onTouch   func(*TouchEvent)
}

func (w *BaseWidget) Contains(x, y float64) bool {
	return x >= w.x && x < w.x+w.width &&
		y >= w.y && y < w.y+w.height
}

func (w *BaseWidget) Children() []Widget {
	return w.children
}

func (w *BaseWidget) HandlePointer(evt *PointerEvent) {
	if w.onPointer != nil {
		w.onPointer(evt)
	}
}

func (w *BaseWidget) HandleKey(evt *KeyEvent) {
	if w.onKey != nil {
		w.onKey(evt)
	}
}

func (w *BaseWidget) HandleTouch(evt *TouchEvent) {
	if w.onTouch != nil {
		w.onTouch(evt)
	}
}

func (w *BaseWidget) SetFocused(focused bool) {
	w.focused = focused
}

func (w *BaseWidget) IsFocused() bool {
	return w.focused
}

// SetBounds sets the widget's position and size.
func (w *BaseWidget) SetBounds(x, y, width, height float64) {
	w.x = x
	w.y = y
	w.width = width
	w.height = height
}

// AddChild adds a child widget.
func (w *BaseWidget) AddChild(child Widget) {
	w.children = append(w.children, child)
}

// OnPointer sets the pointer event callback.
func (w *BaseWidget) OnPointer(handler func(*PointerEvent)) {
	w.onPointer = handler
}

// OnKey sets the keyboard event callback.
func (w *BaseWidget) OnKey(handler func(*KeyEvent)) {
	w.onKey = handler
}

// OnTouch sets the touch event callback.
func (w *BaseWidget) OnTouch(handler func(*TouchEvent)) {
	w.onTouch = handler
}
