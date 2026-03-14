// Package events implements X11 event handling and parsing.
//
// This package provides structures and parsers for X11 protocol events. X11
// events are sent asynchronously from the server to notify the client of
// state changes such as keyboard input, mouse motion, window exposure, and
// configuration changes.
//
// # Event Types
//
// The package implements the following core X11 event types:
//   - KeyPress/KeyRelease: Keyboard input events
//   - ButtonPress/ButtonRelease: Mouse button events
//   - MotionNotify: Mouse pointer movement
//   - Expose: Window content needs redrawing
//   - ConfigureNotify: Window configuration changed
//
// # Event Parsing
//
// Events are received as 32-byte messages from the X server. The first byte
// indicates the event type, and the remaining bytes contain event-specific data.
// All multi-byte fields are encoded in little-endian format.
//
// Reference: https://www.x.org/releases/current/doc/xproto/x11protocol.html
package events

import (
	"encoding/binary"
	"fmt"

	"github.com/opd-ai/wain/internal/x11/wire"
)

// EventType represents an X11 event type code.
type EventType uint8

// Core X11 event types.
const (
	// EventTypeKeyPress is a key press event.
	EventTypeKeyPress EventType = 2
	// EventTypeKeyRelease is a key release event.
	EventTypeKeyRelease EventType = 3
	// EventTypeButtonPress is a mouse button press event.
	EventTypeButtonPress EventType = 4
	// EventTypeButtonRelease is a mouse button release event.
	EventTypeButtonRelease EventType = 5
	// EventTypeMotionNotify is a pointer motion event.
	EventTypeMotionNotify EventType = 6
	// EventTypeEnterNotify is a pointer enter window event.
	EventTypeEnterNotify EventType = 7
	// EventTypeLeaveNotify is a pointer leave window event.
	EventTypeLeaveNotify EventType = 8
	// EventTypeFocusIn is a keyboard focus gained event.
	EventTypeFocusIn EventType = 9
	// EventTypeFocusOut is a keyboard focus lost event.
	EventTypeFocusOut EventType = 10
	// EventTypeExpose is a window expose event.
	EventTypeExpose EventType = 12
	// EventTypeGraphicsExposure is a graphics expose event.
	EventTypeGraphicsExposure EventType = 13
	// EventTypeNoExposure is a no-exposure event.
	EventTypeNoExposure EventType = 14
	// EventTypeVisibilityNotify is a visibility change event.
	EventTypeVisibilityNotify EventType = 15
	// EventTypeCreateNotify is a window creation event.
	EventTypeCreateNotify EventType = 16
	// EventTypeDestroyNotify is a window destruction event.
	EventTypeDestroyNotify EventType = 17
	// EventTypeUnmapNotify is a window unmap event.
	EventTypeUnmapNotify EventType = 18
	// EventTypeMapNotify is a window map event.
	EventTypeMapNotify EventType = 19
	// EventTypeMapRequest is a window map request event.
	EventTypeMapRequest EventType = 20
	// EventTypeReparentNotify is a window reparent event.
	EventTypeReparentNotify EventType = 21
	// EventTypeConfigureNotify is a window configure event.
	EventTypeConfigureNotify EventType = 22
	// EventTypeConfigureRequest is a window configure request event.
	EventTypeConfigureRequest EventType = 23
	// EventTypeGravityNotify is a window gravity change event.
	EventTypeGravityNotify EventType = 24
	// EventTypeResizeRequest is a window resize request event.
	EventTypeResizeRequest EventType = 25
	// EventTypeCirculateNotify is a window circulate event.
	EventTypeCirculateNotify EventType = 26
	// EventTypeCirculateRequest is a window circulate request event.
	EventTypeCirculateRequest EventType = 27
	// EventTypePropertyNotify is a property change event.
	EventTypePropertyNotify EventType = 28
	// EventTypeSelectionClear is a selection clear event.
	EventTypeSelectionClear EventType = 29
	// EventTypeSelectionRequest is a selection request event.
	EventTypeSelectionRequest EventType = 30
	// EventTypeSelectionNotify is a selection notify event.
	EventTypeSelectionNotify EventType = 31
	// EventTypeColormapNotify is a colormap change event.
	EventTypeColormapNotify EventType = 32
	// EventTypeClientMessage is a client message event.
	EventTypeClientMessage EventType = 33
	// EventTypeMappingNotify is a mapping change event.
	EventTypeMappingNotify EventType = 34
)

// String returns a human-readable event type name.
func (t EventType) String() string {
	switch t {
	case EventTypeKeyPress:
		return "KeyPress"
	case EventTypeKeyRelease:
		return "KeyRelease"
	case EventTypeButtonPress:
		return "ButtonPress"
	case EventTypeButtonRelease:
		return "ButtonRelease"
	case EventTypeMotionNotify:
		return "MotionNotify"
	case EventTypeEnterNotify:
		return "EnterNotify"
	case EventTypeLeaveNotify:
		return "LeaveNotify"
	case EventTypeExpose:
		return "Expose"
	case EventTypeConfigureNotify:
		return "ConfigureNotify"
	default:
		return fmt.Sprintf("Event(%d)", t)
	}
}

// KeyPressEvent represents a keyboard key press event.
type KeyPressEvent struct {
	Type       EventType // Event type (KeyPress)
	Detail     uint8     // Keycode of the pressed key
	Sequence   uint16    // Sequence number
	Time       uint32    // Server timestamp in milliseconds
	Root       uint32    // Root window
	Event      uint32    // Event window
	Child      uint32    // Child window (0 if none)
	RootX      int16     // Pointer X coordinate relative to root
	RootY      int16     // Pointer Y coordinate relative to root
	EventX     int16     // Pointer X coordinate relative to event window
	EventY     int16     // Pointer Y coordinate relative to event window
	State      uint16    // Modifier key mask
	SameScreen bool      // True if event and root windows are on same screen
}

// ParseKeyPressEvent decodes a KeyPress event from raw event data.
func ParseKeyPressEvent(header wire.EventHeader, data []byte) (KeyPressEvent, error) {
	if len(data) < 28 {
		return KeyPressEvent{}, fmt.Errorf("events: key press data too short: %d bytes", len(data))
	}

	return KeyPressEvent{
		Type:       EventType(header.Type),
		Detail:     header.Detail,
		Sequence:   header.Sequence,
		Time:       binary.LittleEndian.Uint32(data[0:4]),
		Root:       binary.LittleEndian.Uint32(data[4:8]),
		Event:      binary.LittleEndian.Uint32(data[8:12]),
		Child:      binary.LittleEndian.Uint32(data[12:16]),
		RootX:      int16(binary.LittleEndian.Uint16(data[16:18])),
		RootY:      int16(binary.LittleEndian.Uint16(data[18:20])),
		EventX:     int16(binary.LittleEndian.Uint16(data[20:22])),
		EventY:     int16(binary.LittleEndian.Uint16(data[22:24])),
		State:      binary.LittleEndian.Uint16(data[24:26]),
		SameScreen: data[26] != 0,
	}, nil
}

// KeyReleaseEvent represents a keyboard key release event.
// The structure is identical to KeyPressEvent.
type KeyReleaseEvent KeyPressEvent

// ParseKeyReleaseEvent decodes a KeyRelease event from raw event data.
func ParseKeyReleaseEvent(header wire.EventHeader, data []byte) (KeyReleaseEvent, error) {
	evt, err := ParseKeyPressEvent(header, data)
	return KeyReleaseEvent(evt), err
}

// ButtonPressEvent represents a mouse button press event.
type ButtonPressEvent struct {
	Type       EventType // Event type (ButtonPress)
	Detail     uint8     // Button number (1=left, 2=middle, 3=right, 4=scroll up, 5=scroll down)
	Sequence   uint16    // Sequence number
	Time       uint32    // Server timestamp in milliseconds
	Root       uint32    // Root window
	Event      uint32    // Event window
	Child      uint32    // Child window (0 if none)
	RootX      int16     // Pointer X coordinate relative to root
	RootY      int16     // Pointer Y coordinate relative to root
	EventX     int16     // Pointer X coordinate relative to event window
	EventY     int16     // Pointer Y coordinate relative to event window
	State      uint16    // Modifier key and button mask
	SameScreen bool      // True if event and root windows are on same screen
}

// ParseButtonPressEvent decodes a ButtonPress event from raw event data.
func ParseButtonPressEvent(header wire.EventHeader, data []byte) (ButtonPressEvent, error) {
	if len(data) < 28 {
		return ButtonPressEvent{}, fmt.Errorf("events: button press data too short: %d bytes", len(data))
	}

	return ButtonPressEvent{
		Type:       EventType(header.Type),
		Detail:     header.Detail,
		Sequence:   header.Sequence,
		Time:       binary.LittleEndian.Uint32(data[0:4]),
		Root:       binary.LittleEndian.Uint32(data[4:8]),
		Event:      binary.LittleEndian.Uint32(data[8:12]),
		Child:      binary.LittleEndian.Uint32(data[12:16]),
		RootX:      int16(binary.LittleEndian.Uint16(data[16:18])),
		RootY:      int16(binary.LittleEndian.Uint16(data[18:20])),
		EventX:     int16(binary.LittleEndian.Uint16(data[20:22])),
		EventY:     int16(binary.LittleEndian.Uint16(data[22:24])),
		State:      binary.LittleEndian.Uint16(data[24:26]),
		SameScreen: data[26] != 0,
	}, nil
}

// ButtonReleaseEvent represents a mouse button release event.
// The structure is identical to ButtonPressEvent.
type ButtonReleaseEvent ButtonPressEvent

// ParseButtonReleaseEvent decodes a ButtonRelease event from raw event data.
func ParseButtonReleaseEvent(header wire.EventHeader, data []byte) (ButtonReleaseEvent, error) {
	evt, err := ParseButtonPressEvent(header, data)
	return ButtonReleaseEvent(evt), err
}

// MotionNotifyEvent represents a pointer motion event.
type MotionNotifyEvent struct {
	Type       EventType // Event type (MotionNotify)
	Detail     uint8     // Hint mode (0=normal, 1=hint)
	Sequence   uint16    // Sequence number
	Time       uint32    // Server timestamp in milliseconds
	Root       uint32    // Root window
	Event      uint32    // Event window
	Child      uint32    // Child window (0 if none)
	RootX      int16     // Pointer X coordinate relative to root
	RootY      int16     // Pointer Y coordinate relative to root
	EventX     int16     // Pointer X coordinate relative to event window
	EventY     int16     // Pointer Y coordinate relative to event window
	State      uint16    // Modifier key and button mask
	SameScreen bool      // True if event and root windows are on same screen
}

// ParseMotionNotifyEvent decodes a MotionNotify event from raw event data.
func ParseMotionNotifyEvent(header wire.EventHeader, data []byte) (MotionNotifyEvent, error) {
	if len(data) < 28 {
		return MotionNotifyEvent{}, fmt.Errorf("events: motion notify data too short: %d bytes", len(data))
	}

	return MotionNotifyEvent{
		Type:       EventType(header.Type),
		Detail:     header.Detail,
		Sequence:   header.Sequence,
		Time:       binary.LittleEndian.Uint32(data[0:4]),
		Root:       binary.LittleEndian.Uint32(data[4:8]),
		Event:      binary.LittleEndian.Uint32(data[8:12]),
		Child:      binary.LittleEndian.Uint32(data[12:16]),
		RootX:      int16(binary.LittleEndian.Uint16(data[16:18])),
		RootY:      int16(binary.LittleEndian.Uint16(data[18:20])),
		EventX:     int16(binary.LittleEndian.Uint16(data[20:22])),
		EventY:     int16(binary.LittleEndian.Uint16(data[22:24])),
		State:      binary.LittleEndian.Uint16(data[24:26]),
		SameScreen: data[26] != 0,
	}, nil
}

// ExposeEvent represents a window exposure event.
//
// This event is sent when a window region needs to be redrawn, typically
// because it was previously obscured and is now visible.
type ExposeEvent struct {
	Type     EventType // Event type (Expose)
	Sequence uint16    // Sequence number
	Window   uint32    // Window that needs redrawing
	X        uint16    // X coordinate of exposed region
	Y        uint16    // Y coordinate of exposed region
	Width    uint16    // Width of exposed region
	Height   uint16    // Height of exposed region
	Count    uint16    // Number of Expose events to follow (0 if this is the last)
}

// ParseExposeEvent decodes an Expose event from raw event data.
//
// Currently unused - reserved for future damage tracking optimization.
// Expose events notify the client when window regions need repainting.
// Active event handling uses polling-based damage detection instead.
func ParseExposeEvent(header wire.EventHeader, data []byte) (ExposeEvent, error) {
	if len(data) < 28 {
		return ExposeEvent{}, fmt.Errorf("events: expose data too short: %d bytes", len(data))
	}

	return ExposeEvent{
		Type:     EventType(header.Type),
		Sequence: header.Sequence,
		Window:   binary.LittleEndian.Uint32(data[0:4]),
		X:        binary.LittleEndian.Uint16(data[4:6]),
		Y:        binary.LittleEndian.Uint16(data[6:8]),
		Width:    binary.LittleEndian.Uint16(data[8:10]),
		Height:   binary.LittleEndian.Uint16(data[10:12]),
		Count:    binary.LittleEndian.Uint16(data[12:14]),
	}, nil
}

// ConfigureNotifyEvent represents a window configuration change notification.
//
// This event is sent when a window's size, position, border width, or stacking
// order changes. It is sent to both the reconfigured window and its parent.
type ConfigureNotifyEvent struct {
	Type             EventType // Event type (ConfigureNotify)
	Sequence         uint16    // Sequence number
	Event            uint32    // Window receiving the event
	Window           uint32    // Window that was configured
	AboveSibling     uint32    // Sibling window above this one (0 if bottom)
	X                int16     // New X coordinate relative to parent
	Y                int16     // New Y coordinate relative to parent
	Width            uint16    // New width
	Height           uint16    // New height
	BorderWidth      uint16    // New border width
	OverrideRedirect bool      // True if override-redirect flag is set
}

// ParseConfigureNotifyEvent decodes a ConfigureNotify event from raw event data.
func ParseConfigureNotifyEvent(header wire.EventHeader, data []byte) (ConfigureNotifyEvent, error) {
	if len(data) < 28 {
		return ConfigureNotifyEvent{}, fmt.Errorf("events: configure notify data too short: %d bytes", len(data))
	}

	return ConfigureNotifyEvent{
		Type:             EventType(header.Type),
		Sequence:         header.Sequence,
		Event:            binary.LittleEndian.Uint32(data[0:4]),
		Window:           binary.LittleEndian.Uint32(data[4:8]),
		AboveSibling:     binary.LittleEndian.Uint32(data[8:12]),
		X:                int16(binary.LittleEndian.Uint16(data[12:14])),
		Y:                int16(binary.LittleEndian.Uint16(data[14:16])),
		Width:            binary.LittleEndian.Uint16(data[16:18]),
		Height:           binary.LittleEndian.Uint16(data[18:20]),
		BorderWidth:      binary.LittleEndian.Uint16(data[20:22]),
		OverrideRedirect: data[22] != 0,
	}, nil
}

// ModifierMask represents the modifier key and button state mask.
type ModifierMask uint16

// Modifier mask constants for keyboard and pointer state.
const (
	// ModifierShift indicates the Shift key is pressed.
	ModifierShift ModifierMask = 1 << 0
	// ModifierLock indicates Caps Lock is active.
	ModifierLock ModifierMask = 1 << 1
	// ModifierControl indicates the Control key is pressed.
	ModifierControl ModifierMask = 1 << 2
	// ModifierMod1 indicates Mod1 (typically Alt) is pressed.
	ModifierMod1 ModifierMask = 1 << 3
	// ModifierMod2 indicates Mod2 (typically Num Lock) is active.
	ModifierMod2 ModifierMask = 1 << 4
	// ModifierMod3 indicates Mod3 is pressed.
	ModifierMod3 ModifierMask = 1 << 5
	// ModifierMod4 indicates Mod4 (typically Super/Windows) is pressed.
	ModifierMod4 ModifierMask = 1 << 6
	// ModifierMod5 indicates Mod5 is pressed.
	ModifierMod5 ModifierMask = 1 << 7
	// ModifierButton1 indicates the left mouse button is pressed.
	ModifierButton1 ModifierMask = 1 << 8
	// ModifierButton2 indicates the middle mouse button is pressed.
	ModifierButton2 ModifierMask = 1 << 9
	// ModifierButton3 indicates the right mouse button is pressed.
	ModifierButton3 ModifierMask = 1 << 10
	// ModifierButton4 indicates scroll up button is pressed.
	ModifierButton4 ModifierMask = 1 << 11
	// ModifierButton5 indicates scroll down button is pressed.
	ModifierButton5 ModifierMask = 1 << 12
)

// HasModifier checks if a specific modifier is set in the state mask.
func HasModifier(state uint16, modifier ModifierMask) bool {
	return state&uint16(modifier) != 0
}
