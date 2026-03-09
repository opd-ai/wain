// Package input implements Wayland input protocol objects.
//
// This package provides client-side implementation of the Wayland input protocol,
// including seat, pointer, keyboard, and touch interfaces. The seat is the central
// input management object that provides access to input devices.
//
// Implemented interfaces:
//   - wl_seat: Input device capability discovery and management
//   - wl_pointer: Mouse/pointing device events
//   - wl_keyboard: Keyboard input and key events
//   - wl_touch: Touch screen input events
//
// Protocol specification:
// https://wayland.app/protocols/wayland#wl_seat
//
// # Event Handling
//
// Input objects emit events that applications handle via callbacks. Each input
// device type has its own event handler interface that applications implement.
//
// # Thread Safety
//
// This implementation is not thread-safe. All operations must be performed from
// a single goroutine.
package input

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Conn represents the subset of client.Connection methods needed by input objects.
type Conn interface {
	AllocID() uint32
	RegisterObject(obj interface{})
	SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error
}

// objectBase provides common fields for input-related Wayland objects.
type objectBase struct {
	id    uint32
	iface string
	conn  Conn
}

// ID returns the object's unique identifier.
func (o *objectBase) ID() uint32 {
	return o.id
}

// Interface returns the Wayland interface name.
func (o *objectBase) Interface() string {
	return o.iface
}

// SeatCapability represents input capabilities that a seat may have.
type SeatCapability uint32

const (
	// SeatCapabilityPointer indicates the seat has pointer (mouse) capability.
	SeatCapabilityPointer SeatCapability = 1

	// SeatCapabilityKeyboard indicates the seat has keyboard capability.
	SeatCapabilityKeyboard SeatCapability = 2

	// SeatCapabilityTouch indicates the seat has touch capability.
	SeatCapabilityTouch SeatCapability = 4
)

// Seat represents the wl_seat interface.
//
// A seat is a group of keyboards, pointers and touch devices. This object is
// published as a global during start up, or when such a device is hot plugged.
// A seat typically has a pointer and maintains a keyboard focus and a pointer
// focus.
type Seat struct {
	objectBase
	version      uint32
	capabilities SeatCapability
	name         string
}

const (
	seatOpcodeGetPointer  uint16 = 0
	seatOpcodeGetKeyboard uint16 = 1
	seatOpcodeGetTouch    uint16 = 2
	seatOpcodeRelease     uint16 = 3
)

const (
	seatEventCapabilities uint16 = 0
	seatEventName         uint16 = 1
)

// NewSeat creates a new Seat object from a registry binding.
func NewSeat(conn Conn, id, version uint32) *Seat {
	return &Seat{
		objectBase: objectBase{
			id:    id,
			iface: "wl_seat",
			conn:  conn,
		},
		version: version,
	}
}

// GetPointer returns a pointer object for this seat.
//
// The pointer object will send events when the pointer enters or leaves a
// surface, when it moves, and when buttons are pressed or released.
func (s *Seat) GetPointer() (*Pointer, error) {
	pointerID := s.conn.AllocID()
	pointer := &Pointer{
		objectBase: objectBase{
			id:    pointerID,
			iface: "wl_pointer",
			conn:  s.conn,
		},
	}
	s.conn.RegisterObject(pointer)

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: pointerID},
	}
	if err := s.conn.SendRequest(s.id, seatOpcodeGetPointer, args); err != nil {
		return nil, fmt.Errorf("GetPointer: %w", err)
	}

	return pointer, nil
}

// GetKeyboard returns a keyboard object for this seat.
//
// The keyboard object will send events for key presses and releases, as well
// as keyboard focus changes and modifier state updates.
func (s *Seat) GetKeyboard() (*Keyboard, error) {
	keyboardID := s.conn.AllocID()
	keyboard := &Keyboard{
		objectBase: objectBase{
			id:    keyboardID,
			iface: "wl_keyboard",
			conn:  s.conn,
		},
	}
	s.conn.RegisterObject(keyboard)

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: keyboardID},
	}
	if err := s.conn.SendRequest(s.id, seatOpcodeGetKeyboard, args); err != nil {
		return nil, fmt.Errorf("GetKeyboard: %w", err)
	}

	return keyboard, nil
}

// GetTouch returns a touch object for this seat.
//
// The touch object will send events for touch down, up, motion, and other
// touch-related interactions.
func (s *Seat) GetTouch() (*Touch, error) {
	touchID := s.conn.AllocID()
	touch := &Touch{
		objectBase: objectBase{
			id:    touchID,
			iface: "wl_touch",
			conn:  s.conn,
		},
	}
	s.conn.RegisterObject(touch)

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: touchID},
	}
	if err := s.conn.SendRequest(s.id, seatOpcodeGetTouch, args); err != nil {
		return nil, fmt.Errorf("GetTouch: %w", err)
	}

	return touch, nil
}

// Release destroys the seat object.
func (s *Seat) Release() error {
	return s.conn.SendRequest(s.id, seatOpcodeRelease, nil)
}

// HandleCapabilities processes a capabilities event from the compositor.
//
// This event is sent when the seat's capabilities change. The capabilities
// bitfield indicates which input devices are available.
func (s *Seat) HandleCapabilities(caps uint32) {
	s.capabilities = SeatCapability(caps)
}

// HandleName processes a name event from the compositor.
//
// This event is sent to give the seat a human-readable name.
func (s *Seat) HandleName(name string) {
	s.name = name
}

// HandleEvent implements the EventHandler interface for wl_seat events.
func (s *Seat) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case seatEventCapabilities:
		if len(args) < 1 {
			return fmt.Errorf("seat: capabilities event requires 1 argument, got %d", len(args))
		}
		caps, ok := args[0].Value.(uint32)
		if !ok {
			return fmt.Errorf("seat: capabilities must be uint32")
		}
		s.HandleCapabilities(caps)
		return nil

	case seatEventName:
		if len(args) < 1 {
			return fmt.Errorf("seat: name event requires 1 argument, got %d", len(args))
		}
		name, ok := args[0].Value.(string)
		if !ok {
			return fmt.Errorf("seat: name must be string")
		}
		s.HandleName(name)
		return nil

	default:
		return fmt.Errorf("seat: unknown event opcode %d", opcode)
	}
}

// Capabilities returns the current seat capabilities.
func (s *Seat) Capabilities() SeatCapability {
	return s.capabilities
}

// Name returns the seat name.
func (s *Seat) Name() string {
	return s.name
}
