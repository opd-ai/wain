package a11y

import (
	"fmt"
	"sync"
)

// AccessibleObject holds the accessibility metadata for a single widget.
// It is exported over D-Bus via four AT-SPI2 interfaces.
type AccessibleObject struct {
	mu sync.RWMutex

	id          uint64
	parentID    uint64
	childIDs    []uint64
	role        Role
	name        string
	description string
	x, y        int32
	width       int32
	height      int32
	focused     bool
	enabled     bool
	text        string
	caretOffset int32
	actions     []objectAction
	manager     *Manager
}

// objectAction represents one activatable action exposed via the Action interface.
type objectAction struct {
	name        string
	description string
	keyBinding  string
	do          func() bool
}

// objectPath returns the D-Bus object path for this accessible object.
func (o *AccessibleObject) objectPath() string {
	return fmt.Sprintf("%s/%d", basePath, o.id)
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

// addChild registers a child object ID in the ordered child list.
func (o *AccessibleObject) addChild(childID uint64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.childIDs = append(o.childIDs, childID)
}

// states builds the current StateSet for this object.
func (o *AccessibleObject) states() StateSet {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return defaultStates(o.focused, o.enabled)
}

// snapshot holds an immutable snapshot of object fields for D-Bus methods.
type snapshot struct {
	id          uint64
	parentID    uint64
	childIDs    []uint64
	role        Role
	name        string
	description string
	x, y        int32
	width       int32
	height      int32
	focused     bool
	enabled     bool
	text        string
	caretOffset int32
	actions     []objectAction
}

// snap returns an immutable copy of the object's current state.
func (o *AccessibleObject) snap() snapshot {
	o.mu.RLock()
	defer o.mu.RUnlock()
	ids := make([]uint64, len(o.childIDs))
	copy(ids, o.childIDs)
	acts := make([]objectAction, len(o.actions))
	copy(acts, o.actions)
	return snapshot{
		id:          o.id,
		parentID:    o.parentID,
		childIDs:    ids,
		role:        o.role,
		name:        o.name,
		description: o.description,
		x:           o.x,
		y:           o.y,
		width:       o.width,
		height:      o.height,
		focused:     o.focused,
		enabled:     o.enabled,
		text:        o.text,
		caretOffset: o.caretOffset,
		actions:     acts,
	}
}
