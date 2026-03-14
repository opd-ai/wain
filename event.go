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
	// EventTypeDrag identifies drag-and-drop events.
	EventTypeDrag
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
	// KeyEscape is the Escape key.
	KeyEscape Key = 0xFF1B
	// KeyReturn is the Return/Enter key.
	KeyReturn Key = 0xFF0D
	// KeyTab is the Tab key.
	KeyTab Key = 0xFF09
	// KeyBackspace is the Backspace key.
	KeyBackspace Key = 0xFF08
	// KeyDelete is the Delete key.
	KeyDelete Key = 0xFFFF
	// KeyLeft is the Left arrow key.
	KeyLeft Key = 0xFF51
	// KeyUp is the Up arrow key.
	KeyUp Key = 0xFF52
	// KeyRight is the Right arrow key.
	KeyRight Key = 0xFF53
	// KeyDown is the Down arrow key.
	KeyDown Key = 0xFF54
	// KeyHome is the Home key.
	KeyHome Key = 0xFF50
	// KeyEnd is the End key.
	KeyEnd Key = 0xFF57
	// KeyPageUp is the Page Up key.
	KeyPageUp Key = 0xFF55
	// KeyPageDown is the Page Down key.
	KeyPageDown Key = 0xFF56
	// KeySpace is the Space key.
	KeySpace Key = 0x0020
	// KeyShiftL is the Left Shift key.
	KeyShiftL Key = 0xFFE1
	// KeyShiftR is the Right Shift key.
	KeyShiftR Key = 0xFFE2
	// KeyControlL is the Left Control key.
	KeyControlL Key = 0xFFE3
	// KeyControlR is the Right Control key.
	KeyControlR Key = 0xFFE4
	// KeyAltL is the Left Alt key.
	KeyAltL Key = 0xFFE9
	// KeyAltR is the Right Alt key.
	KeyAltR Key = 0xFFEA
	// KeySuperL is the Left Super (Windows/Command) key.
	KeySuperL Key = 0xFFEB
	// KeySuperR is the Right Super (Windows/Command) key.
	KeySuperR Key = 0xFFEC
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

// CustomEventPayload is an opaque payload for application-defined custom events.
// Applications can send any data through the event system by wrapping it in
// a CustomEvent. The receiving handler is responsible for type-asserting the
// payload to the expected concrete type.
type CustomEventPayload interface{}

// CustomEvent represents application-defined events.
type CustomEvent struct {
	baseEvent
	data CustomEventPayload
}

// Type returns EventTypeCustom for custom events.
func (e *CustomEvent) Type() EventType { return EventTypeCustom }

// Data returns the application-defined data attached to the event.
func (e *CustomEvent) Data() CustomEventPayload { return e.data }

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

// translateX11KeyReleaseEvent converts an X11 KeyRelease event to a wain KeyEvent.
func translateX11KeyReleaseEvent(e events.KeyReleaseEvent) *KeyEvent {
	return &KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: KeyRelease,
		key:       Key(e.Detail),
	}
}

// translateX11ButtonPressEvent converts an X11 ButtonPress event to a wain PointerEvent, handling both mouse buttons and scroll wheel.
func translateX11ButtonPressEvent(e events.ButtonPressEvent) Event {
	pe := &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerButtonPress,
		x:         float64(e.EventX),
		y:         float64(e.EventY),
	}
	// X11 button mapping: 1=left, 2=middle, 3=right, 4/5=scroll
	if e.Detail <= 3 {
		// Map X11 button to Linux input event code
		pe.button = PointerButton(0x110 + uint32(e.Detail) - 1)
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

// translateX11ButtonReleaseEvent converts an X11 ButtonRelease event to a wain PointerEvent, filtering scroll button releases.
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
		button:    PointerButton(0x110 + uint32(e.Detail) - 1),
	}
}

// translateX11MotionNotifyEvent converts an X11 MotionNotify event to a wain PointerEvent.
func translateX11MotionNotifyEvent(e events.MotionNotifyEvent) *PointerEvent {
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerMove,
		x:         float64(e.EventX),
		y:         float64(e.EventY),
	}
}

// translateX11ConfigureNotifyEvent converts an X11 ConfigureNotify event to a wain WindowEvent.
func translateX11ConfigureNotifyEvent(e events.ConfigureNotifyEvent) *WindowEvent {
	return &WindowEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: WindowResize,
		width:     int(e.Width),
		height:    int(e.Height),
	}
}

// linuxKeycodeMap maps Linux input event keycodes to X11 keysyms.
var linuxKeycodeMap = map[uint32]Key{
	1:   KeyEscape,
	28:  KeyReturn,
	15:  KeyTab,
	14:  KeyBackspace,
	111: KeyDelete,
	105: KeyLeft,
	103: KeyUp,
	106: KeyRight,
	108: KeyDown,
	102: KeyHome,
	107: KeyEnd,
	104: KeyPageUp,
	109: KeyPageDown,
	57:  KeySpace,
	42:  KeyShiftL,
	54:  KeyShiftR,
	29:  KeyControlL,
	97:  KeyControlR,
	56:  KeyAltL,
	100: KeyAltR,
	125: KeySuperL,
	126: KeySuperR,
	// Number keys (1-0)
	2: Key(0x0031), 3: Key(0x0032), 4: Key(0x0033), 5: Key(0x0034), 6: Key(0x0035),
	7: Key(0x0036), 8: Key(0x0037), 9: Key(0x0038), 10: Key(0x0039), 11: Key(0x0030),
	// QWERTY row
	16: Key('q'), 17: Key('w'), 18: Key('e'), 19: Key('r'), 20: Key('t'),
	21: Key('y'), 22: Key('u'), 23: Key('i'), 24: Key('o'), 25: Key('p'),
	// ASDF row
	30: Key('a'), 31: Key('s'), 32: Key('d'), 33: Key('f'), 34: Key('g'),
	35: Key('h'), 36: Key('j'), 37: Key('k'), 38: Key('l'),
	// ZXCV row
	44: Key('z'), 45: Key('x'), 46: Key('c'), 47: Key('v'), 48: Key('b'),
	49: Key('n'), 50: Key('m'),
}

// linuxToKeysym converts Linux input event keycodes to X11 keysyms (simplified).
func linuxToKeysym(code uint32) Key {
	if key, ok := linuxKeycodeMap[code]; ok {
		return key
	}
	return Key(code)
}

// translateWaylandKeyEvent converts Wayland key event data to a wain KeyEvent.
func translateWaylandKeyEvent(key, state uint32) *KeyEvent {
	eventType := KeyRelease
	if state == 1 { // KeyStatePressed
		eventType = KeyPress
	}
	return &KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: eventType,
		key:       linuxToKeysym(key),
	}
}

// translateWaylandPointerButtonEvent converts Wayland pointer button event data to a wain PointerEvent.
func translateWaylandPointerButtonEvent(button, state uint32, x, y float64) *PointerEvent {
	eventType := PointerButtonRelease
	if state == 1 { // ButtonStatePressed
		eventType = PointerButtonPress
	}
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: eventType,
		x:         x,
		y:         y,
		button:    PointerButton(button),
	}
}

// translateWaylandPointerMotionEvent converts Wayland pointer motion event data to a wain PointerEvent.
func translateWaylandPointerMotionEvent(x, y float64) *PointerEvent {
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerMove,
		x:         x,
		y:         y,
	}
}

// translateWaylandPointerAxisEvent converts Wayland pointer axis (scroll) event data to a wain PointerEvent.
func translateWaylandPointerAxisEvent(axis uint32, value, x, y float64) *PointerEvent {
	scrollAxis := ScrollAxisVertical
	if axis == 1 { // AxisHorizontalScroll
		scrollAxis = ScrollAxisHorizontal
	}
	// Wayland axis values are in pixels, convert to scroll steps
	// Negative value = scroll up/left, positive = scroll down/right
	scrollValue := value / 10.0
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerScroll,
		x:         x,
		y:         y,
		axis:      scrollAxis,
		value:     scrollValue,
	}
}

// DragEventKind specifies the kind of drag-and-drop event.
type DragEventKind int

// Drag event kind constants.
const (
	// DragEnter is dispatched when a drag enters the window.
	DragEnter DragEventKind = iota
	// DragMove is dispatched as the drag moves within the window.
	DragMove
	// DragDrop is dispatched when the user drops onto the window.
	DragDrop
	// DragLeave is dispatched when a drag leaves the window without dropping.
	DragLeave
)

// DragEvent represents a drag-and-drop interaction.
type DragEvent struct {
	baseEvent
	kind      DragEventKind
	x, y      float64
	mimeTypes []string
}

// Type returns EventTypeDrag for drag-and-drop events.
func (e *DragEvent) Type() EventType { return EventTypeDrag }

// Kind returns the specific kind of drag event.
func (e *DragEvent) Kind() DragEventKind { return e.kind }

// X returns the horizontal coordinate of the drag point, in window pixels.
func (e *DragEvent) X() float64 { return e.x }

// Y returns the vertical coordinate of the drag point, in window pixels.
func (e *DragEvent) Y() float64 { return e.y }

// MimeTypes returns the MIME types offered by the drag source.
// The slice is non-nil only for [DragEnter] events.
func (e *DragEvent) MimeTypes() []string { return e.mimeTypes }

// newDragEvent constructs a DragEvent.
func newDragEvent(kind DragEventKind, x, y float64, mimeTypes []string) *DragEvent {
	return &DragEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		kind:      kind,
		x:         x,
		y:         y,
		mimeTypes: mimeTypes,
	}
}

// DragDropHandler is called when a drop event is received on a window that has
// registered itself as a drop target via [Window.SetDropTarget].
//
// mimeType is the negotiated MIME type; data contains the dropped bytes.
type DragDropHandler func(mimeType string, data []byte)

// DragDataProvider is called when the toolkit needs to supply drag data for a
// MIME type, used when starting a drag via [Window.StartDrag].
//
// The function must write the data for mimeType into the returned byte slice.
type DragDataProvider func(mimeType string) []byte
