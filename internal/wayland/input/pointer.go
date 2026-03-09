package input

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// ButtonState represents the state of a pointer button.
type ButtonState uint32

const (
	// ButtonStateReleased indicates the button is released.
	ButtonStateReleased ButtonState = 0

	// ButtonStatePressed indicates the button is pressed.
	ButtonStatePressed ButtonState = 1
)

// Axis represents a pointer scroll axis.
type Axis uint32

const (
	// AxisVerticalScroll is the vertical scroll axis.
	AxisVerticalScroll Axis = 0

	// AxisHorizontalScroll is the horizontal scroll axis.
	AxisHorizontalScroll Axis = 1
)

// Pointer represents the wl_pointer interface.
//
// The wl_pointer interface represents one or more input devices, such as mice,
// which control the pointer location and pointer focus of a seat.
//
// The pointer has a location and a focus surface. Enter and leave events are
// generated whenever the pointer location crosses the boundary of a surface.
type Pointer struct {
	objectBase
	focusedSurface uint32
	surfaceX       float64
	surfaceY       float64
	onButton       func(surfaceID, button, state uint32, x, y float64)
	onMotion       func(surfaceID uint32, x, y float64)
	onAxis         func(surfaceID, axis uint32, value, x, y float64)
	onEnter        func(surfaceID uint32, x, y float64)
	onLeave        func(surfaceID uint32)
}

const (
	pointerOpcodeSetCursor uint16 = 0
	pointerOpcodeRelease   uint16 = 1
)

const (
	pointerEventEnter        uint16 = 0
	pointerEventLeave        uint16 = 1
	pointerEventMotion       uint16 = 2
	pointerEventButton       uint16 = 3
	pointerEventAxis         uint16 = 4
	pointerEventFrame        uint16 = 5
	pointerEventAxisSource   uint16 = 6
	pointerEventAxisStop     uint16 = 7
	pointerEventAxisDiscrete uint16 = 8
)

// SetCursor sets the cursor image for this pointer.
//
// The hotspot coordinates specify the location in the cursor image that
// corresponds to the pointer location in surface-local coordinates.
//
// Parameters:
//   - serial: Serial number from the enter event
//   - surfaceID: Surface containing the pointer image, or 0 to hide cursor
//   - hotspotX: X coordinate of the cursor hotspot
//   - hotspotY: Y coordinate of the cursor hotspot
func (p *Pointer) SetCursor(serial, surfaceID uint32, hotspotX, hotspotY int32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: serial},
		{Type: wire.ArgTypeObject, Value: surfaceID},
		{Type: wire.ArgTypeInt32, Value: hotspotX},
		{Type: wire.ArgTypeInt32, Value: hotspotY},
	}
	if err := p.conn.SendRequest(p.id, pointerOpcodeSetCursor, args); err != nil {
		return fmt.Errorf("SetCursor: %w", err)
	}
	return nil
}

// Release destroys the pointer object.
func (p *Pointer) Release() error {
	return p.conn.SendRequest(p.id, pointerOpcodeRelease, nil)
}

// SetButtonCallback sets the callback function for button events.
func (p *Pointer) SetButtonCallback(fn func(surfaceID, button, state uint32, x, y float64)) {
	p.onButton = fn
}

// SetMotionCallback sets the callback function for motion events.
func (p *Pointer) SetMotionCallback(fn func(surfaceID uint32, x, y float64)) {
	p.onMotion = fn
}

// SetAxisCallback sets the callback function for axis (scroll) events.
func (p *Pointer) SetAxisCallback(fn func(surfaceID, axis uint32, value, x, y float64)) {
	p.onAxis = fn
}

// SetEnterCallback sets the callback function for pointer enter events.
func (p *Pointer) SetEnterCallback(fn func(surfaceID uint32, x, y float64)) {
	p.onEnter = fn
}

// SetLeaveCallback sets the callback function for pointer leave events.
func (p *Pointer) SetLeaveCallback(fn func(surfaceID uint32)) {
	p.onLeave = fn
}

// HandleEnter processes an enter event from the compositor.
//
// This event is sent when the pointer enters a surface. The surface-local
// coordinates are provided in fixed-point format (multiply by 1/256).
func (p *Pointer) HandleEnter(serial, surfaceID uint32, surfaceX, surfaceY int32) {
	p.focusedSurface = surfaceID
	p.surfaceX = float64(surfaceX) / 256.0
	p.surfaceY = float64(surfaceY) / 256.0
	if p.onEnter != nil {
		p.onEnter(surfaceID, p.surfaceX, p.surfaceY)
	}
}

// HandleLeave processes a leave event from the compositor.
//
// This event is sent when the pointer leaves a surface.
func (p *Pointer) HandleLeave(serial, surfaceID uint32) {
	p.focusedSurface = 0
	if p.onLeave != nil {
		p.onLeave(surfaceID)
	}
}

// HandleMotion processes a motion event from the compositor.
//
// This event is sent when the pointer moves. The coordinates are in
// surface-local coordinates in fixed-point format.
func (p *Pointer) HandleMotion(time uint32, surfaceX, surfaceY int32) {
	p.surfaceX = float64(surfaceX) / 256.0
	p.surfaceY = float64(surfaceY) / 256.0
	if p.onMotion != nil && p.focusedSurface != 0 {
		p.onMotion(p.focusedSurface, p.surfaceX, p.surfaceY)
	}
}

// HandleButton processes a button event from the compositor.
//
// This event is sent when a pointer button is pressed or released.
func (p *Pointer) HandleButton(serial, time, button, state uint32) {
	if p.onButton != nil && p.focusedSurface != 0 {
		p.onButton(p.focusedSurface, button, state, p.surfaceX, p.surfaceY)
	}
}

// HandleAxis processes an axis event from the compositor.
//
// This event is sent when a scroll or other axis event occurs. The value
// is in surface-local coordinates.
func (p *Pointer) HandleAxis(time, axis uint32, value int32) {
	if p.onAxis != nil && p.focusedSurface != 0 {
		p.onAxis(p.focusedSurface, axis, float64(value)/256.0, p.surfaceX, p.surfaceY)
	}
}

// HandleFrame processes a frame event from the compositor.
//
// This event groups related pointer events together. Applications should
// process all events in a frame atomically.
func (p *Pointer) HandleFrame() {
}

// HandleAxisSource processes an axis source event from the compositor.
//
// This event describes the physical source of axis events.
func (p *Pointer) HandleAxisSource(axisSource uint32) {
}

// HandleAxisStop processes an axis stop event from the compositor.
//
// This event indicates that scrolling has stopped on the given axis.
func (p *Pointer) HandleAxisStop(time, axis uint32) {
}

// HandleAxisDiscrete processes an axis discrete event from the compositor.
//
// This event carries discrete step information for scroll wheels.
func (p *Pointer) HandleAxisDiscrete(axis uint32, discrete int32) {
}

// HandleEvent implements the EventHandler interface for wl_pointer events.
func (p *Pointer) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case pointerEventEnter:
		return p.handleEnterEvent(args)
	case pointerEventLeave:
		return p.handleLeaveEvent(args)
	case pointerEventMotion:
		return p.handleMotionEvent(args)
	case pointerEventButton:
		return p.handleButtonEvent(args)
	case pointerEventAxis:
		return p.handleAxisEvent(args)
	case pointerEventFrame:
		p.HandleFrame()
		return nil
	case pointerEventAxisSource:
		return p.handleAxisSourceEvent(args)
	case pointerEventAxisStop:
		return p.handleAxisStopEvent(args)
	case pointerEventAxisDiscrete:
		return p.handleAxisDiscreteEvent(args)
	default:
		return fmt.Errorf("pointer: unknown event opcode %d", opcode)
	}
}

func (p *Pointer) handleEnterEvent(args []wire.Argument) error {
	if len(args) < 4 {
		return fmt.Errorf("pointer: enter event requires 4 arguments, got %d", len(args))
	}
	serial, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: enter serial must be uint32")
	}
	surfaceID, ok := args[1].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: enter surface must be uint32")
	}
	surfaceX, ok := args[2].Value.(int32)
	if !ok {
		return fmt.Errorf("pointer: enter surface_x must be fixed")
	}
	surfaceY, ok := args[3].Value.(int32)
	if !ok {
		return fmt.Errorf("pointer: enter surface_y must be fixed")
	}
	p.HandleEnter(serial, surfaceID, surfaceX, surfaceY)
	return nil
}

func (p *Pointer) handleLeaveEvent(args []wire.Argument) error {
	if len(args) < 2 {
		return fmt.Errorf("pointer: leave event requires 2 arguments, got %d", len(args))
	}
	serial, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: leave serial must be uint32")
	}
	surfaceID, ok := args[1].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: leave surface must be uint32")
	}
	p.HandleLeave(serial, surfaceID)
	return nil
}

func (p *Pointer) handleMotionEvent(args []wire.Argument) error {
	if len(args) < 3 {
		return fmt.Errorf("pointer: motion event requires 3 arguments, got %d", len(args))
	}
	time, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: motion time must be uint32")
	}
	surfaceX, ok := args[1].Value.(int32)
	if !ok {
		return fmt.Errorf("pointer: motion surface_x must be fixed")
	}
	surfaceY, ok := args[2].Value.(int32)
	if !ok {
		return fmt.Errorf("pointer: motion surface_y must be fixed")
	}
	p.HandleMotion(time, surfaceX, surfaceY)
	return nil
}

func (p *Pointer) handleButtonEvent(args []wire.Argument) error {
	if len(args) < 4 {
		return fmt.Errorf("pointer: button event requires 4 arguments, got %d", len(args))
	}
	serial, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: button serial must be uint32")
	}
	time, ok := args[1].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: button time must be uint32")
	}
	button, ok := args[2].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: button code must be uint32")
	}
	state, ok := args[3].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: button state must be uint32")
	}
	p.HandleButton(serial, time, button, state)
	return nil
}

func (p *Pointer) handleAxisEvent(args []wire.Argument) error {
	if len(args) < 3 {
		return fmt.Errorf("pointer: axis event requires 3 arguments, got %d", len(args))
	}
	time, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: axis time must be uint32")
	}
	axis, ok := args[1].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: axis type must be uint32")
	}
	value, ok := args[2].Value.(int32)
	if !ok {
		return fmt.Errorf("pointer: axis value must be fixed")
	}
	p.HandleAxis(time, axis, value)
	return nil
}

func (p *Pointer) handleAxisSourceEvent(args []wire.Argument) error {
	if len(args) < 1 {
		return fmt.Errorf("pointer: axis_source event requires 1 argument, got %d", len(args))
	}
	axisSource, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: axis_source must be uint32")
	}
	p.HandleAxisSource(axisSource)
	return nil
}

func (p *Pointer) handleAxisStopEvent(args []wire.Argument) error {
	if len(args) < 2 {
		return fmt.Errorf("pointer: axis_stop event requires 2 arguments, got %d", len(args))
	}
	time, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: axis_stop time must be uint32")
	}
	axis, ok := args[1].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: axis_stop axis must be uint32")
	}
	p.HandleAxisStop(time, axis)
	return nil
}

func (p *Pointer) handleAxisDiscreteEvent(args []wire.Argument) error {
	if len(args) < 2 {
		return fmt.Errorf("pointer: axis_discrete event requires 2 arguments, got %d", len(args))
	}
	axis, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("pointer: axis_discrete axis must be uint32")
	}
	discrete, ok := args[1].Value.(int32)
	if !ok {
		return fmt.Errorf("pointer: axis_discrete discrete must be int32")
	}
	p.HandleAxisDiscrete(axis, discrete)
	return nil
}
