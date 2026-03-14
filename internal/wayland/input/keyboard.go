package input

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// KeyState represents the state of a key.
type KeyState uint32

const (
	// KeyStateReleased indicates the key is released.
	KeyStateReleased KeyState = 0

	// KeyStatePressed indicates the key is pressed.
	KeyStatePressed KeyState = 1
)

// Keyboard represents the wl_keyboard interface.
//
// The wl_keyboard interface represents one or more keyboards associated with
// a seat. It provides events for key presses, releases, and modifier state.
type Keyboard struct {
	objectBase
	keymap         *Keymap
	modifiers      ModifierState
	focusedSurface uint32
	onKey          func(surfaceID, key, state uint32)
	onEnter        func(surfaceID uint32)
	onLeave        func(surfaceID uint32)
	onModifiers    func(modsDepressed, modsLatched, modsLocked uint32)
}

const (
	keyboardOpcodeRelease uint16 = 0
)

const (
	keyboardEventKeymap     uint16 = 0
	keyboardEventEnter      uint16 = 1
	keyboardEventLeave      uint16 = 2
	keyboardEventKey        uint16 = 3
	keyboardEventModifiers  uint16 = 4
	keyboardEventRepeatInfo uint16 = 5
)

// ModifierState represents the state of keyboard modifiers.
type ModifierState struct {
	Shift    bool
	CapsLock bool
	Ctrl     bool
	Alt      bool
	NumLock  bool
	Meta     bool
}

// Release destroys the keyboard object.
func (k *Keyboard) Release() error {
	return k.conn.SendRequest(k.id, keyboardOpcodeRelease, nil)
}

// SetKeyCallback sets the callback function for key events.
func (k *Keyboard) SetKeyCallback(fn func(surfaceID, key, state uint32)) {
	k.onKey = fn
}

// SetEnterCallback sets the callback function for focus enter events.
func (k *Keyboard) SetEnterCallback(fn func(surfaceID uint32)) {
	k.onEnter = fn
}

// SetLeaveCallback sets the callback function for focus leave events.
func (k *Keyboard) SetLeaveCallback(fn func(surfaceID uint32)) {
	k.onLeave = fn
}

// SetModifiersCallback sets the callback function for modifier state changes.
func (k *Keyboard) SetModifiersCallback(fn func(modsDepressed, modsLatched, modsLocked uint32)) {
	k.onModifiers = fn
}

// HandleKeymap processes a keymap event from the compositor.
//
// This event provides a file descriptor containing the keymap in XKB format.
// The format parameter indicates the keymap format (1 = XKB v1).
func (k *Keyboard) HandleKeymap(format, fd, size uint32) {
	if format == 1 {
		k.keymap = NewKeymap(int(fd), int(size))
	}
}

// HandleEnter processes an enter event from the compositor.
//
// This event is sent when keyboard focus enters a surface.
func (k *Keyboard) HandleEnter(serial, surfaceID uint32, keys []uint32) {
	k.focusedSurface = surfaceID
	if k.onEnter != nil {
		k.onEnter(surfaceID)
	}
}

// HandleLeave processes a leave event from the compositor.
//
// This event is sent when keyboard focus leaves a surface.
func (k *Keyboard) HandleLeave(serial, surfaceID uint32) {
	k.focusedSurface = 0
	if k.onLeave != nil {
		k.onLeave(surfaceID)
	}
}

// HandleKey processes a key event from the compositor.
//
// This event is sent when a key is pressed or released. The key parameter
// is a Linux evdev keycode.
func (k *Keyboard) HandleKey(serial, time, key, state uint32) {
	if k.onKey != nil && k.focusedSurface != 0 {
		k.onKey(k.focusedSurface, key, state)
	}
}

// HandleModifiers processes a modifiers event from the compositor.
//
// This event is sent when the modifier state changes. The parameters are
// XKB modifier indices.
func (k *Keyboard) HandleModifiers(serial, modsDepressed, modsLatched, modsLocked, group uint32) {
	k.modifiers = k.decodeModifiers(modsDepressed, modsLatched, modsLocked)
	if k.onModifiers != nil {
		k.onModifiers(modsDepressed, modsLatched, modsLocked)
	}
}

// HandleRepeatInfo processes a repeat info event from the compositor.
//
// This event provides keyboard repeat rate and delay information.
func (k *Keyboard) HandleRepeatInfo(rate, delay int32) {
}

// HandleEvent implements the EventHandler interface for wl_keyboard events.
func (k *Keyboard) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case keyboardEventKeymap:
		return k.handleKeymapEvent(args)
	case keyboardEventEnter:
		return k.handleEnterEvent(args)
	case keyboardEventLeave:
		return k.handleLeaveEvent(args)
	case keyboardEventKey:
		return k.handleKeyEvent(args)
	case keyboardEventModifiers:
		return k.handleModifiersEvent(args)
	case keyboardEventRepeatInfo:
		return k.handleRepeatInfoEvent(args)
	default:
		return fmt.Errorf("keyboard: unknown event opcode %d", opcode)
	}
}

// handleKeymapEvent processes wl_keyboard.keymap event (opcode 0).
func (k *Keyboard) handleKeymapEvent(args []wire.Argument) error {
	var format, size uint32
	var fd int
	if err := parseEvent(args, 3, "keyboard: keymap event", func(d *wire.ArgDecoder) {
		format = d.Uint32("keyboard: keymap format")
		fd = d.Int("keyboard: keymap fd")
		size = d.Uint32("keyboard: keymap size")
	}); err != nil {
		return fmt.Errorf("wayland/input: keyboard handle keymap: %w", err)
	}
	k.HandleKeymap(format, uint32(fd), size)
	return nil
}

// handleEnterEvent processes wl_keyboard.enter event (opcode 1).
func (k *Keyboard) handleEnterEvent(args []wire.Argument) error {
	var serial, surfaceID uint32
	var keysArray []byte
	if err := parseEvent(args, 3, "keyboard: enter event", func(d *wire.ArgDecoder) {
		serial = d.Uint32("keyboard: enter serial")
		surfaceID = d.Uint32("keyboard: enter surface")
		keysArray = d.Bytes("keyboard: enter keys")
	}); err != nil {
		return fmt.Errorf("wayland/input: keyboard handle enter: %w", err)
	}
	keys := make([]uint32, len(keysArray)/4)
	for i := range keys {
		offset := i * 4
		keys[i] = uint32(keysArray[offset]) | uint32(keysArray[offset+1])<<8 |
			uint32(keysArray[offset+2])<<16 | uint32(keysArray[offset+3])<<24
	}
	k.HandleEnter(serial, surfaceID, keys)
	return nil
}

// handleLeaveEvent processes wl_keyboard.leave event (opcode 2).
func (k *Keyboard) handleLeaveEvent(args []wire.Argument) error {
	var serial, surfaceID uint32
	if err := parseEvent(args, 2, "keyboard: leave event", func(d *wire.ArgDecoder) {
		serial = d.Uint32("keyboard: leave serial")
		surfaceID = d.Uint32("keyboard: leave surface")
	}); err != nil {
		return fmt.Errorf("wayland/input: keyboard handle leave: %w", err)
	}
	k.HandleLeave(serial, surfaceID)
	return nil
}

// handleKeyEvent processes wl_keyboard.key event (opcode 3).
func (k *Keyboard) handleKeyEvent(args []wire.Argument) error {
	var serial, time, key, state uint32
	if err := parseEvent(args, 4, "keyboard: key event", func(d *wire.ArgDecoder) {
		serial = d.Uint32("keyboard: key serial")
		time = d.Uint32("keyboard: key time")
		key = d.Uint32("keyboard: key code")
		state = d.Uint32("keyboard: key state")
	}); err != nil {
		return fmt.Errorf("wayland/input: keyboard handle key: %w", err)
	}
	k.HandleKey(serial, time, key, state)
	return nil
}

// handleModifiersEvent processes wl_keyboard.modifiers event (opcode 4).
func (k *Keyboard) handleModifiersEvent(args []wire.Argument) error {
	var serial, modsDepressed, modsLatched, modsLocked, group uint32
	if err := parseEvent(args, 5, "keyboard: modifiers event", func(d *wire.ArgDecoder) {
		serial = d.Uint32("keyboard: modifiers serial")
		modsDepressed = d.Uint32("keyboard: modifiers depressed")
		modsLatched = d.Uint32("keyboard: modifiers latched")
		modsLocked = d.Uint32("keyboard: modifiers locked")
		group = d.Uint32("keyboard: modifiers group")
	}); err != nil {
		return fmt.Errorf("wayland/input: keyboard handle modifiers: %w", err)
	}
	k.HandleModifiers(serial, modsDepressed, modsLatched, modsLocked, group)
	return nil
}

// handleRepeatInfoEvent processes wl_keyboard.repeat_info event (opcode 5).
func (k *Keyboard) handleRepeatInfoEvent(args []wire.Argument) error {
	var rate, delay int32
	if err := parseEvent(args, 2, "keyboard: repeat_info event", func(d *wire.ArgDecoder) {
		rate = d.Int32("keyboard: repeat_info rate")
		delay = d.Int32("keyboard: repeat_info delay")
	}); err != nil {
		return fmt.Errorf("wayland/input: keyboard handle repeat info: %w", err)
	}
	k.HandleRepeatInfo(rate, delay)
	return nil
}

// decodeModifiers converts Wayland modifier bitmasks (depressed, latched, locked) into a ModifierState.
// Combines all three masks and extracts individual modifier flags (Shift, Ctrl, Alt, etc.).
func (k *Keyboard) decodeModifiers(depressed, latched, locked uint32) ModifierState {
	mask := depressed | latched | locked
	return ModifierState{
		Shift:    (mask & 0x01) != 0,
		CapsLock: (mask & 0x02) != 0,
		Ctrl:     (mask & 0x04) != 0,
		Alt:      (mask & 0x08) != 0,
		NumLock:  (mask & 0x10) != 0,
		Meta:     (mask & 0x40) != 0,
	}
}

// Modifiers returns the current modifier state.
func (k *Keyboard) Modifiers() ModifierState {
	return k.modifiers
}
