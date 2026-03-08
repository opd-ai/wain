package wain

import (
	"time"

	"github.com/opd-ai/wain/internal/x11/events"
)

// EventType identifies the type of event.
type EventType int

const (
	EventTypePointer EventType = iota
	EventTypeKey
	EventTypeTouch
	EventTypeWindow
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

func (e *baseEvent) Timestamp() time.Time { return e.timestamp }
func (e *baseEvent) Consumed() bool       { return e.consumed }
func (e *baseEvent) Consume()             { e.consumed = true }

// PointerEventType specifies the type of pointer event.
type PointerEventType int

const (
	PointerMove PointerEventType = iota
	PointerButtonPress
	PointerButtonRelease
	PointerScroll
	PointerEnter
	PointerLeave
)

// PointerButton represents a mouse button.
type PointerButton uint32

const (
	PointerButtonLeft   PointerButton = 0x110 // BTN_LEFT
	PointerButtonRight  PointerButton = 0x111 // BTN_RIGHT
	PointerButtonMiddle PointerButton = 0x112 // BTN_MIDDLE
)

// ScrollAxis represents the scroll direction.
type ScrollAxis int

const (
	ScrollAxisVertical ScrollAxis = iota
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

func (e *PointerEvent) Type() EventType             { return EventTypePointer }
func (e *PointerEvent) EventType() PointerEventType { return e.eventType }
func (e *PointerEvent) X() float64                  { return e.x }
func (e *PointerEvent) Y() float64                  { return e.y }
func (e *PointerEvent) Button() PointerButton       { return e.button }
func (e *PointerEvent) Axis() ScrollAxis            { return e.axis }
func (e *PointerEvent) Value() float64              { return e.value }

// KeyEventType specifies the type of keyboard event.
type KeyEventType int

const (
	KeyPress KeyEventType = iota
	KeyRelease
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

const (
	ModShift   Modifier = 1 << 0
	ModControl Modifier = 1 << 1
	ModAlt     Modifier = 1 << 2
	ModSuper   Modifier = 1 << 3
)

// KeyEvent represents keyboard events.
type KeyEvent struct {
	baseEvent
	eventType KeyEventType
	key       Key
	modifiers Modifier
	rune      rune
}

func (e *KeyEvent) Type() EventType         { return EventTypeKey }
func (e *KeyEvent) EventType() KeyEventType { return e.eventType }
func (e *KeyEvent) Key() Key                { return e.key }
func (e *KeyEvent) Modifiers() Modifier     { return e.modifiers }
func (e *KeyEvent) Rune() rune              { return e.rune }

// TouchEventType specifies the type of touch event.
type TouchEventType int

const (
	TouchDown TouchEventType = iota
	TouchUp
	TouchMotion
	TouchCancel
)

// TouchEvent represents touch screen events.
type TouchEvent struct {
	baseEvent
	eventType TouchEventType
	id        int32
	x, y      float64
}

func (e *TouchEvent) Type() EventType           { return EventTypeTouch }
func (e *TouchEvent) EventType() TouchEventType { return e.eventType }
func (e *TouchEvent) ID() int32                 { return e.id }
func (e *TouchEvent) X() float64                { return e.x }
func (e *TouchEvent) Y() float64                { return e.y }

// WindowEventType specifies the type of window event.
type WindowEventType int

const (
	WindowResize WindowEventType = iota
	WindowClose
	WindowFocus
	WindowUnfocus
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

func (e *WindowEvent) Type() EventType            { return EventTypeWindow }
func (e *WindowEvent) EventType() WindowEventType { return e.eventType }
func (e *WindowEvent) Width() int                 { return e.width }
func (e *WindowEvent) Height() int                { return e.height }
func (e *WindowEvent) Scale() float64             { return e.scale }

// CustomEvent represents application-defined events.
type CustomEvent struct {
	baseEvent
	data interface{}
}

func (e *CustomEvent) Type() EventType   { return EventTypeCustom }
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
