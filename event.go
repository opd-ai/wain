package wain

import (
	"time"

	"github.com/opd-ai/wain/internal/x11/events"
)

// EventType identifies the type of event.
type EventType int

// Event type constants identify the category of an event.
const (
	// EventTypePointer identifies mouse/touchpad pointer events.
	EventTypePointer EventType = iota
	// EventTypeKey identifies keyboard events.
	EventTypeKey
	// EventTypeTouch identifies touch screen events.
	EventTypeTouch
	// EventTypeWindow identifies window state events.
	EventTypeWindow
	// EventTypeCustom identifies application-defined events.
	EventTypeCustom
)

// Event is the common interface for all events.
type Event interface {
	Type() EventType
	Timestamp() time.Time
	Consumed() bool
	Consume()
}

// baseEvent provides common fields for all events.
type baseEvent struct {
	timestamp time.Time
	consumed  bool
}

// Timestamp returns the time when the event occurred.
func (e *baseEvent) Timestamp() time.Time { return e.timestamp }

// Consumed returns whether the event has been consumed by an event handler.
func (e *baseEvent) Consumed() bool { return e.consumed }

// Consume marks the event as consumed, preventing further processing.
func (e *baseEvent) Consume() { e.consumed = true }

// PointerEventType specifies the type of pointer event.
type PointerEventType int

// Pointer event type constants.
const (
	// PointerMove indicates the pointer has moved.
	PointerMove PointerEventType = iota
	// PointerButtonPress indicates a mouse button was pressed.
	PointerButtonPress
	// PointerButtonRelease indicates a mouse button was released.
	PointerButtonRelease
	// PointerScroll indicates a scroll wheel event.
	PointerScroll
	// PointerEnter indicates the pointer entered the window.
	PointerEnter
	// PointerLeave indicates the pointer left the window.
	PointerLeave
)

// PointerButton represents a mouse button.
type PointerButton uint32

// Pointer button constants (Linux input event codes).
const (
	// PointerButtonLeft is the left mouse button (BTN_LEFT).
	PointerButtonLeft PointerButton = 0x110
	// PointerButtonRight is the right mouse button (BTN_RIGHT).
	PointerButtonRight PointerButton = 0x111
	// PointerButtonMiddle is the middle mouse button (BTN_MIDDLE).
	PointerButtonMiddle PointerButton = 0x112
)

// ScrollAxis represents the scroll direction.
type ScrollAxis int

// Scroll axis constants.
const (
	// ScrollAxisVertical indicates vertical scrolling (up/down).
	ScrollAxisVertical ScrollAxis = iota
	// ScrollAxisHorizontal indicates horizontal scrolling (left/right).
	ScrollAxisHorizontal
)

// PointerEvent represents mouse/touchpad pointer events.
type PointerEvent struct {
	baseEvent
	eventType PointerEventType
	x, y      float64
	button    PointerButton
	axis      ScrollAxis
	value     float64
}

// Type returns EventTypePointer for pointer events.
func (e *PointerEvent) Type() EventType { return EventTypePointer }

// EventType returns the specific type of pointer event.
func (e *PointerEvent) EventType() PointerEventType { return e.eventType }

// X returns the pointer's X coordinate within the window.
func (e *PointerEvent) X() float64 { return e.x }

// Y returns the pointer's Y coordinate within the window.
func (e *PointerEvent) Y() float64 { return e.y }

// Button returns which mouse button triggered the event.
func (e *PointerEvent) Button() PointerButton { return e.button }

// Axis returns the scroll axis for scroll events.
func (e *PointerEvent) Axis() ScrollAxis { return e.axis }

// Value returns the scroll delta for scroll events.
func (e *PointerEvent) Value() float64 { return e.value }

// KeyEventType specifies the type of keyboard event.
type KeyEventType int

// Key event type constants.
const (
	// KeyPress indicates a key was pressed down.
	KeyPress KeyEventType = iota
	// KeyRelease indicates a key was released.
	KeyRelease
	// KeyRepeat indicates a key is being held and repeating.
	KeyRepeat
)

// Key represents a keyboard key symbol.
type Key uint32

// Common key constants (compatible with X11 keysyms and Linux input event codes).
const (
	KeyEscape    Key = 0xFF1B
	KeyReturn    Key = 0xFF0D
	KeyTab       Key = 0xFF09
	KeyBackspace Key = 0xFF08
	KeyDelete    Key = 0xFFFF
	KeyLeft      Key = 0xFF51
	KeyUp        Key = 0xFF52
	KeyRight     Key = 0xFF53
	KeyDown      Key = 0xFF54
	KeyHome      Key = 0xFF50
	KeyEnd       Key = 0xFF57
	KeyPageUp    Key = 0xFF55
	KeyPageDown  Key = 0xFF56
	KeySpace     Key = 0x0020
	KeyShiftL    Key = 0xFFE1
	KeyShiftR    Key = 0xFFE2
	KeyControlL  Key = 0xFFE3
	KeyControlR  Key = 0xFFE4
	KeyAltL      Key = 0xFFE9
	KeyAltR      Key = 0xFFEA
	KeySuperL    Key = 0xFFEB
	KeySuperR    Key = 0xFFEC
)

// Modifier represents keyboard modifiers.
type Modifier uint32

// Keyboard modifier constants.
const (
	// ModShift indicates the Shift key is held.
	ModShift Modifier = 1 << 0
	// ModControl indicates the Control key is held.
	ModControl Modifier = 1 << 1
	// ModAlt indicates the Alt key is held.
	ModAlt Modifier = 1 << 2
	// ModSuper indicates the Super (Windows/Command) key is held.
	ModSuper Modifier = 1 << 3
)

// KeyEvent represents keyboard events.
type KeyEvent struct {
	baseEvent
	eventType KeyEventType
	key       Key
	modifiers Modifier
	rune      rune
}

// Type returns EventTypeKey for keyboard events.
func (e *KeyEvent) Type() EventType { return EventTypeKey }

// EventType returns the specific type of keyboard event.
func (e *KeyEvent) EventType() KeyEventType { return e.eventType }

// Key returns the key symbol that triggered the event.
func (e *KeyEvent) Key() Key { return e.key }

// Modifiers returns the active keyboard modifiers.
func (e *KeyEvent) Modifiers() Modifier { return e.modifiers }

// Rune returns the character produced by the key, if any.
func (e *KeyEvent) Rune() rune { return e.rune }

// TouchEventType specifies the type of touch event.
type TouchEventType int

// Touch event type constants.
const (
	// TouchDown indicates a touch point was pressed.
	TouchDown TouchEventType = iota
	// TouchUp indicates a touch point was released.
	TouchUp
	// TouchMotion indicates a touch point has moved.
	TouchMotion
	// TouchCancel indicates a touch sequence was cancelled.
	TouchCancel
)

// TouchEvent represents touch screen events.
type TouchEvent struct {
	baseEvent
	eventType TouchEventType
	id        int32
	x, y      float64
}

// Type returns EventTypeTouch for touch events.
func (e *TouchEvent) Type() EventType { return EventTypeTouch }

// EventType returns the specific type of touch event.
func (e *TouchEvent) EventType() TouchEventType { return e.eventType }

// ID returns the unique identifier for this touch point.
func (e *TouchEvent) ID() int32 { return e.id }

// X returns the touch point's X coordinate within the window.
func (e *TouchEvent) X() float64 { return e.x }

// Y returns the touch point's Y coordinate within the window.
func (e *TouchEvent) Y() float64 { return e.y }

// WindowEventType specifies the type of window event.
type WindowEventType int

// Window event type constants.
const (
	// WindowResize indicates the window was resized.
	WindowResize WindowEventType = iota
	// WindowClose indicates the window close was requested.
	WindowClose
	// WindowFocus indicates the window gained focus.
	WindowFocus
	// WindowUnfocus indicates the window lost focus.
	WindowUnfocus
	// WindowScaleChange indicates the window's scale factor changed.
	WindowScaleChange
)

// WindowEvent represents window state events.
type WindowEvent struct {
	baseEvent
	eventType WindowEventType
	width     int
	height    int
	scale     float64
}

// Type returns EventTypeWindow for window events.
func (e *WindowEvent) Type() EventType { return EventTypeWindow }

// EventType returns the specific type of window event.
func (e *WindowEvent) EventType() WindowEventType { return e.eventType }

// Width returns the new window width after a resize event.
func (e *WindowEvent) Width() int { return e.width }

// Height returns the new window height after a resize event.
func (e *WindowEvent) Height() int { return e.height }

// Scale returns the new scale factor after a scale change event.
func (e *WindowEvent) Scale() float64 { return e.scale }

// CustomEvent represents application-defined events.
type CustomEvent struct {
	baseEvent
	data interface{}
}

// Type returns EventTypeCustom for custom events.
func (e *CustomEvent) Type() EventType { return EventTypeCustom }

// Data returns the application-defined data attached to the event.
func (e *CustomEvent) Data() interface{} { return e.data }

// EventHandler is a function that processes events.
type EventHandler func(Event)

// translateX11Event converts X11 events to public events.
func translateX11KeyPressEvent(e events.KeyPressEvent) *KeyEvent {
	return &KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: KeyPress,
		key:       Key(e.Detail), // X11 keycode needs proper keysym translation
	}
}

func translateX11KeyReleaseEvent(e events.KeyReleaseEvent) *KeyEvent {
	return &KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: KeyRelease,
		key:       Key(e.Detail),
	}
}

func translateX11ButtonPressEvent(e events.ButtonPressEvent) Event {
	pe := &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerButtonPress,
		x:         float64(e.EventX),
		y:         float64(e.EventY),
	}
	// X11 button mapping: 1=left, 2=middle, 3=right, 4/5=scroll
	if e.Detail <= 3 {
		pe.button = PointerButton(0x110 + e.Detail - 1)
	} else if e.Detail == 4 {
		pe.eventType = PointerScroll
		pe.axis = ScrollAxisVertical
		pe.value = -1.0
	} else if e.Detail == 5 {
		pe.eventType = PointerScroll
		pe.axis = ScrollAxisVertical
		pe.value = 1.0
	}
	return pe
}

func translateX11ButtonReleaseEvent(e events.ButtonReleaseEvent) Event {
	// Skip release events for scroll buttons
	if e.Detail >= 4 && e.Detail <= 5 {
		return nil
	}
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerButtonRelease,
		x:         float64(e.EventX),
		y:         float64(e.EventY),
		button:    PointerButton(0x110 + e.Detail - 1),
	}
}

func translateX11MotionNotifyEvent(e events.MotionNotifyEvent) *PointerEvent {
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerMove,
		x:         float64(e.EventX),
		y:         float64(e.EventY),
	}
}

func translateX11EnterNotifyEvent(e events.EnterNotifyEvent) *PointerEvent {
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerEnter,
		x:         float64(e.EventX),
		y:         float64(e.EventY),
	}
}

func translateX11LeaveNotifyEvent(e events.LeaveNotifyEvent) *PointerEvent {
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerLeave,
		x:         float64(e.EventX),
		y:         float64(e.EventY),
	}
}

func translateX11ConfigureNotifyEvent(e events.ConfigureNotifyEvent) *WindowEvent {
	return &WindowEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: WindowResize,
		width:     int(e.Width),
		height:    int(e.Height),
	}
}

func translateX11FocusInEvent(e events.FocusInEvent) *WindowEvent {
	return &WindowEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: WindowFocus,
	}
}

func translateX11FocusOutEvent(e events.FocusOutEvent) *WindowEvent {
	return &WindowEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: WindowUnfocus,
	}
}

// linuxToKeysym converts Linux input event keycodes to X11 keysyms (simplified).
func linuxToKeysym(code uint32) Key {
	// Simplified mapping for common keys
	// Full mapping requires xkbcommon or lookup table
	switch code {
	case 1:
		return KeyEscape
	case 28:
		return KeyReturn
	case 15:
		return KeyTab
	case 14:
		return KeyBackspace
	case 111:
		return KeyDelete
	case 105:
		return KeyLeft
	case 103:
		return KeyUp
	case 106:
		return KeyRight
	case 108:
		return KeyDown
	case 102:
		return KeyHome
	case 107:
		return KeyEnd
	case 104:
		return KeyPageUp
	case 109:
		return KeyPageDown
	case 57:
		return KeySpace
	case 42:
		return KeyShiftL
	case 54:
		return KeyShiftR
	case 29:
		return KeyControlL
	case 97:
		return KeyControlR
	case 56:
		return KeyAltL
	case 100:
		return KeyAltR
	case 125:
		return KeySuperL
	case 126:
		return KeySuperR
	default:
		// For printable characters, use the code directly
		if code >= 2 && code <= 11 { // 1-0 keys
			return Key(0x0030 + (code-1)%10)
		}
		if code >= 16 && code <= 25 { // QWERTY keys
			qwerty := "qwertyuiop"
			return Key(qwerty[code-16])
		}
		if code >= 30 && code <= 38 { // ASDF keys
			asdf := "asdfghjkl"
			return Key(asdf[code-30])
		}
		if code >= 44 && code <= 50 { // ZXCV keys
			zxcv := "zxcvbnm"
			return Key(zxcv[code-44])
		}
		return Key(code)
	}
}
