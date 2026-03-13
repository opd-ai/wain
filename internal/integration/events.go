package integration

import (
	"time"

	"github.com/opd-ai/wain/internal/wayland/input"
	"github.com/opd-ai/wain/internal/x11/events"
)

// WaylandPointerEvent represents a Wayland pointer event with all necessary state.
type WaylandPointerEvent struct {
	Type     WaylandPointerEventType
	Serial   uint32
	Time     uint32
	SurfaceX int32
	SurfaceY int32
	Button   uint32
	State    input.ButtonState
	Axis     input.Axis
	Value    int32
}

// WaylandPointerEventType identifies the specific pointer event type.
type WaylandPointerEventType int

const (
	WaylandPointerEnter WaylandPointerEventType = iota
	WaylandPointerLeave
	WaylandPointerMotion
	WaylandPointerButton
	WaylandPointerAxis
)

// WaylandKeyboardEvent represents a Wayland keyboard event with all necessary state.
type WaylandKeyboardEvent struct {
	Type      WaylandKeyboardEventType
	Serial    uint32
	Time      uint32
	Key       uint32
	State     input.KeyState
	Modifiers input.ModifierState
}

// WaylandKeyboardEventType identifies the specific keyboard event type.
type WaylandKeyboardEventType int

const (
	WaylandKeyboardEnter WaylandKeyboardEventType = iota
	WaylandKeyboardLeave
	WaylandKeyboardKey
)

// PointerEvent is the public pointer event interface.
type PointerEvent interface {
	EventType() PointerEventType
	X() float64
	Y() float64
	Button() uint32
	Axis() uint32
	Value() float64
	Timestamp() time.Time
}

// PointerEventType identifies pointer event types.
type PointerEventType int

const (
	PointerMove PointerEventType = iota
	PointerButtonPress
	PointerButtonRelease
	PointerScroll
	PointerEnter
	PointerLeave
)

// KeyEvent is the public key event interface.
type KeyEvent interface {
	EventType() KeyEventType
	Key() uint32
	Modifiers() uint32
	Timestamp() time.Time
}

// KeyEventType identifies key event types.
type KeyEventType int

const (
	KeyPress KeyEventType = iota
	KeyRelease
)

// pointerEvent implements the PointerEvent interface.
type pointerEvent struct {
	eventType PointerEventType
	x, y      float64
	button    uint32
	axis      uint32
	value     float64
	timestamp time.Time
}

// EventType indicates the type of pointer event (motion, button press/release, axis scroll).
func (e *pointerEvent) EventType() PointerEventType { return e.eventType }

// X is the horizontal pointer coordinate in surface-local space.
func (e *pointerEvent) X() float64 { return e.x }

// Y is the vertical pointer coordinate in surface-local space.
func (e *pointerEvent) Y() float64 { return e.y }

// Button is the button number for button press/release events (BTN_LEFT=0x110, BTN_RIGHT=0x111, BTN_MIDDLE=0x112).
func (e *pointerEvent) Button() uint32 { return e.button }

// Axis is the axis type for scroll events (vertical=0, horizontal=1).
func (e *pointerEvent) Axis() uint32 { return e.axis }

// Value is the scroll amount in surface-local coordinates (positive=down/right, negative=up/left).
func (e *pointerEvent) Value() float64 { return e.value }

// Timestamp is the event timestamp from the display server.
func (e *pointerEvent) Timestamp() time.Time { return e.timestamp }

// keyEvent implements the KeyEvent interface.
type keyEvent struct {
	eventType KeyEventType
	key       uint32
	modifiers uint32
	timestamp time.Time
}

// EventType indicates the type of key event (press or release).
func (e *keyEvent) EventType() KeyEventType { return e.eventType }

// Key is the Linux keycode (evdev scancode) for the key that was pressed or released.
func (e *keyEvent) Key() uint32 { return e.key }

// Modifiers is a bitmask of active modifier keys (Shift, Ctrl, Alt, Super).
func (e *keyEvent) Modifiers() uint32 { return e.modifiers }

// Timestamp is the event timestamp from the display server.
func (e *keyEvent) Timestamp() time.Time { return e.timestamp }

// TranslateWaylandPointer converts Wayland pointer events to public pointer events.
func TranslateWaylandPointer(evt WaylandPointerEvent) PointerEvent {
	pe := &pointerEvent{
		x:         float64(evt.SurfaceX) / 256.0, // Wayland uses fixed-point 24.8
		y:         float64(evt.SurfaceY) / 256.0,
		timestamp: time.Now(),
	}

	switch evt.Type {
	case WaylandPointerEnter:
		pe.eventType = PointerEnter
	case WaylandPointerLeave:
		pe.eventType = PointerLeave
	case WaylandPointerMotion:
		pe.eventType = PointerMove
	case WaylandPointerButton:
		if evt.State == input.ButtonStatePressed {
			pe.eventType = PointerButtonPress
		} else {
			pe.eventType = PointerButtonRelease
		}
		pe.button = evt.Button
	case WaylandPointerAxis:
		pe.eventType = PointerScroll
		pe.axis = uint32(evt.Axis)
		pe.value = float64(evt.Value) / 256.0
	}

	return pe
}

// TranslateWaylandKeyboard converts Wayland keyboard events to public key events.
func TranslateWaylandKeyboard(evt WaylandKeyboardEvent) KeyEvent {
	ke := &keyEvent{
		key:       evt.Key,
		timestamp: time.Now(),
	}

	if evt.State == input.KeyStatePressed {
		ke.eventType = KeyPress
	} else {
		ke.eventType = KeyRelease
	}

	// Convert Wayland modifier state to public modifier bitmask
	ke.modifiers = encodeModifiers(evt.Modifiers)

	return ke
}

// TranslateX11Pointer converts X11 pointer events to public pointer events.
func TranslateX11Pointer(evt interface{}) PointerEvent {
	pe := &pointerEvent{
		timestamp: time.Now(),
	}

	switch e := evt.(type) {
	case events.ButtonPressEvent:
		pe.x = float64(e.EventX)
		pe.y = float64(e.EventY)
		applyX11ButtonPress(pe, e.Detail)

	case events.ButtonReleaseEvent:
		if !applyX11ButtonRelease(pe, e) {
			return nil
		}

	case events.MotionNotifyEvent:
		pe.eventType = PointerMove
		pe.x = float64(e.EventX)
		pe.y = float64(e.EventY)

	default:
		return nil
	}

	return pe
}

// applyX11ButtonPress populates pe with the event type, button, and scroll
// direction for an X11 ButtonPress event. X11 button mapping: 1=left,
// 2=middle, 3=right, 4=scroll up, 5=scroll down.
func applyX11ButtonPress(pe *pointerEvent, detail uint8) {
	if detail <= 3 {
		pe.eventType = PointerButtonPress
		pe.button = 0x110 + uint32(detail) - 1
	} else if detail == 4 {
		pe.eventType = PointerScroll
		pe.axis = 0
		pe.value = -1.0
	} else if detail == 5 {
		pe.eventType = PointerScroll
		pe.axis = 0
		pe.value = 1.0
	}
}

// applyX11ButtonRelease populates pe with button release data and returns
// false when the event should be discarded (scroll-wheel button releases).
func applyX11ButtonRelease(pe *pointerEvent, e events.ButtonReleaseEvent) bool {
	if e.Detail >= 4 && e.Detail <= 5 {
		return false
	}
	pe.eventType = PointerButtonRelease
	pe.x = float64(e.EventX)
	pe.y = float64(e.EventY)
	pe.button = 0x110 + uint32(e.Detail) - 1
	return true
}

// TranslateX11Keyboard converts X11 keyboard events to public key events.
func TranslateX11Keyboard(evt interface{}) KeyEvent {
	ke := &keyEvent{
		timestamp: time.Now(),
	}

	switch e := evt.(type) {
	case events.KeyPressEvent:
		ke.eventType = KeyPress
		ke.key = linuxToKeysym(uint32(e.Detail))
		// X11 state field contains modifier mask
		ke.modifiers = uint32(e.State)

	case events.KeyReleaseEvent:
		ke.eventType = KeyRelease
		ke.key = linuxToKeysym(uint32(e.Detail))
		ke.modifiers = uint32(e.State)

	default:
		return nil
	}

	return ke
}

// encodeModifiers converts Wayland ModifierState to a public modifier bitmask.
func encodeModifiers(mods input.ModifierState) uint32 {
	var mask uint32
	if mods.Shift {
		mask |= 1 << 0
	}
	if mods.Ctrl {
		mask |= 1 << 1
	}
	if mods.Alt {
		mask |= 1 << 2
	}
	if mods.Meta {
		mask |= 1 << 3
	}
	return mask
}

// linuxKeycodeMapInternal maps Linux input event keycodes to X11 keysyms.
var linuxKeycodeMapInternal = map[uint32]uint32{
	1:   0xFF1B, // Escape
	28:  0xFF0D, // Return
	15:  0xFF09, // Tab
	14:  0xFF08, // Backspace
	111: 0xFFFF, // Delete
	105: 0xFF51, // Left
	103: 0xFF52, // Up
	106: 0xFF53, // Right
	108: 0xFF54, // Down
	102: 0xFF50, // Home
	107: 0xFF57, // End
	104: 0xFF55, // PageUp
	109: 0xFF56, // PageDown
	57:  0x0020, // Space
	42:  0xFFE1, // ShiftL
	54:  0xFFE2, // ShiftR
	29:  0xFFE3, // ControlL
	97:  0xFFE4, // ControlR
	56:  0xFFE9, // AltL
	100: 0xFFEA, // AltR
	125: 0xFFEB, // SuperL
	126: 0xFFEC, // SuperR
	// Number keys (1-0)
	2: 0x0031, 3: 0x0032, 4: 0x0033, 5: 0x0034, 6: 0x0035,
	7: 0x0036, 8: 0x0037, 9: 0x0038, 10: 0x0039, 11: 0x0030,
	// QWERTY row
	16: 'q', 17: 'w', 18: 'e', 19: 'r', 20: 't',
	21: 'y', 22: 'u', 23: 'i', 24: 'o', 25: 'p',
	// ASDF row
	30: 'a', 31: 's', 32: 'd', 33: 'f', 34: 'g',
	35: 'h', 36: 'j', 37: 'k', 38: 'l',
	// ZXCV row
	44: 'z', 45: 'x', 46: 'c', 47: 'v', 48: 'b',
	49: 'n', 50: 'm',
}

// linuxToKeysym converts Linux input event keycodes to X11 keysyms.
func linuxToKeysym(code uint32) uint32 {
	if keysym, ok := linuxKeycodeMapInternal[code]; ok {
		return keysym
	}
	return code
}
