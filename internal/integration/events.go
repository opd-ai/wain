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

func (e *pointerEvent) EventType() PointerEventType { return e.eventType }
func (e *pointerEvent) X() float64                  { return e.x }
func (e *pointerEvent) Y() float64                  { return e.y }
func (e *pointerEvent) Button() uint32              { return e.button }
func (e *pointerEvent) Axis() uint32                { return e.axis }
func (e *pointerEvent) Value() float64              { return e.value }
func (e *pointerEvent) Timestamp() time.Time        { return e.timestamp }

// keyEvent implements the KeyEvent interface.
type keyEvent struct {
	eventType KeyEventType
	key       uint32
	modifiers uint32
	timestamp time.Time
}

func (e *keyEvent) EventType() KeyEventType { return e.eventType }
func (e *keyEvent) Key() uint32             { return e.key }
func (e *keyEvent) Modifiers() uint32       { return e.modifiers }
func (e *keyEvent) Timestamp() time.Time    { return e.timestamp }

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
		// X11 button mapping: 1=left, 2=middle, 3=right, 4/5=scroll
		if e.Detail <= 3 {
			pe.eventType = PointerButtonPress
			// Map X11 buttons to Linux input event codes
			pe.button = 0x110 + uint32(e.Detail) - 1
		} else if e.Detail == 4 {
			pe.eventType = PointerScroll
			pe.axis = 0 // vertical
			pe.value = -1.0
		} else if e.Detail == 5 {
			pe.eventType = PointerScroll
			pe.axis = 0 // vertical
			pe.value = 1.0
		}

	case events.ButtonReleaseEvent:
		// Skip scroll button releases
		if e.Detail >= 4 && e.Detail <= 5 {
			return nil
		}
		pe.eventType = PointerButtonRelease
		pe.x = float64(e.EventX)
		pe.y = float64(e.EventY)
		pe.button = 0x110 + uint32(e.Detail) - 1

	case events.MotionNotifyEvent:
		pe.eventType = PointerMove
		pe.x = float64(e.EventX)
		pe.y = float64(e.EventY)

	default:
		return nil
	}

	return pe
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

// linuxToKeysym converts Linux input event keycodes to X11 keysyms.
// This is a simplified mapping for common keys.
func linuxToKeysym(code uint32) uint32 {
	switch code {
	case 1:
		return 0xFF1B // Escape
	case 28:
		return 0xFF0D // Return
	case 15:
		return 0xFF09 // Tab
	case 14:
		return 0xFF08 // Backspace
	case 111:
		return 0xFFFF // Delete
	case 105:
		return 0xFF51 // Left
	case 103:
		return 0xFF52 // Up
	case 106:
		return 0xFF53 // Right
	case 108:
		return 0xFF54 // Down
	case 102:
		return 0xFF50 // Home
	case 107:
		return 0xFF57 // End
	case 104:
		return 0xFF55 // PageUp
	case 109:
		return 0xFF56 // PageDown
	case 57:
		return 0x0020 // Space
	case 42:
		return 0xFFE1 // ShiftL
	case 54:
		return 0xFFE2 // ShiftR
	case 29:
		return 0xFFE3 // ControlL
	case 97:
		return 0xFFE4 // ControlR
	case 56:
		return 0xFFE9 // AltL
	case 100:
		return 0xFFEA // AltR
	case 125:
		return 0xFFEB // SuperL
	case 126:
		return 0xFFEC // SuperR
	default:
		// For printable characters, attempt simple mapping
		if code >= 2 && code <= 11 {
			return 0x0030 + (code-1)%10 // 1-0 keys
		}
		if code >= 16 && code <= 25 {
			qwerty := "qwertyuiop"
			return uint32(qwerty[code-16])
		}
		if code >= 30 && code <= 38 {
			asdf := "asdfghjkl"
			return uint32(asdf[code-30])
		}
		if code >= 44 && code <= 50 {
			zxcv := "zxcvbnm"
			return uint32(zxcv[code-44])
		}
		return code
	}
}
