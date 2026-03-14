package a11y

import "sync"

// AccessibleObject holds the accessibility metadata for a single widget.
type AccessibleObject struct {
	mu      sync.RWMutex
	name    string
	x, y    int32
	width   int32
	height  int32
	focused bool
	text    string
}

// SetBounds updates the widget's screen coordinates and dimensions.
func (o *AccessibleObject) SetBounds(x, y, width, height int32) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.x, o.y, o.width, o.height = x, y, width, height
}

// SetFocused updates the widget's keyboard focus state.
func (o *AccessibleObject) SetFocused(focused bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.focused = focused
}

// SetText updates the text content for Entry widgets.
func (o *AccessibleObject) SetText(text string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.text = text
}

// SetName updates the accessible name exposed to screen readers.
func (o *AccessibleObject) SetName(name string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.name = name
}
