package a11y

import (
	"fmt"
	"sync"
)

// basePath is the root D-Bus object path prefix for all accessible objects.
// Used by atspi build-tagged code in accessible_iface.go and manager.go.
const basePath = "/org/a11y/atspi/accessible" //nolint:unused // used with -tags=atspi

// AccessibleObject holds the accessibility metadata for a single widget.
// It is exported over D-Bus via four AT-SPI2 interfaces.
type AccessibleObject struct {
	mu sync.RWMutex

	id          uint64         //nolint:unused // used with -tags=atspi
	parentID    uint64         //nolint:unused // used with -tags=atspi
	childIDs    []uint64       //nolint:unused // used with -tags=atspi
	role        Role           //nolint:unused // used with -tags=atspi
	name        string
	description string         //nolint:unused // used with -tags=atspi
	x, y        int32
	width       int32
	height      int32
	focused     bool
	enabled     bool           //nolint:unused // used with -tags=atspi
	text        string
	caretOffset int32          //nolint:unused // used with -tags=atspi
	actions     []objectAction //nolint:unused // used with -tags=atspi
	manager     *Manager       //nolint:unused // used with -tags=atspi
}

// objectAction represents one activatable action exposed via the Action interface.
type objectAction struct { //nolint:unused // used with -tags=atspi
	name        string       //nolint:unused // used with -tags=atspi
	description string       //nolint:unused // used with -tags=atspi
	keyBinding  string       //nolint:unused // used with -tags=atspi
	do          func() bool  //nolint:unused // used with -tags=atspi
}

// objectPath returns the D-Bus object path for this accessible object.
func (o *AccessibleObject) objectPath() string { //nolint:unused // used with -tags=atspi
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
func (o *AccessibleObject) addChild(childID uint64) { //nolint:unused // used with -tags=atspi
	o.mu.Lock()
	defer o.mu.Unlock()
	o.childIDs = append(o.childIDs, childID)
}

// states builds the current StateSet for this object.
func (o *AccessibleObject) states() StateSet { //nolint:unused // used with -tags=atspi
	o.mu.RLock()
	defer o.mu.RUnlock()
	return defaultStates(o.focused, o.enabled)
}

// snapshot holds an immutable snapshot of object fields for D-Bus methods.
type snapshot struct { //nolint:unused // used with -tags=atspi
	id          uint64         //nolint:unused // used with -tags=atspi
	parentID    uint64         //nolint:unused // used with -tags=atspi
	childIDs    []uint64       //nolint:unused // used with -tags=atspi
	role        Role           //nolint:unused // used with -tags=atspi
	name        string         //nolint:unused // used with -tags=atspi
	description string         //nolint:unused // used with -tags=atspi
	x, y        int32          //nolint:unused // used with -tags=atspi
	width       int32          //nolint:unused // used with -tags=atspi
	height      int32          //nolint:unused // used with -tags=atspi
	focused     bool           //nolint:unused // used with -tags=atspi
	enabled     bool           //nolint:unused // used with -tags=atspi
	text        string         //nolint:unused // used with -tags=atspi
	caretOffset int32          //nolint:unused // used with -tags=atspi
	actions     []objectAction //nolint:unused // used with -tags=atspi
}

// snap returns an immutable copy of the object's current state.
func (o *AccessibleObject) snap() snapshot { //nolint:unused // used with -tags=atspi
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
