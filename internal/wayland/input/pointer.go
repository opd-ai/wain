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

// handleEnterEvent processes wl_pointer.enter event (opcode 0).
func (p *Pointer) handleEnterEvent(args []wire.Argument) error {
	if err := wire.ParseArgMinLen(args, 4, "pointer: enter event"); err != nil {
		return err
	}
	d := wire.NewArgDecoder(args)
	serial := d.Uint32("pointer: enter serial")
	surfaceID := d.Uint32("pointer: enter surface")
	surfaceX := d.Int32("pointer: enter surface_x")
	surfaceY := d.Int32("pointer: enter surface_y")
	if err := d.Err(); err != nil {
		return err
	}
	p.HandleEnter(serial, surfaceID, surfaceX, surfaceY)
	return nil
}

// handleLeaveEvent processes wl_pointer.leave event (opcode 1).
func (p *Pointer) handleLeaveEvent(args []wire.Argument) error {
	if err := wire.ParseArgMinLen(args, 2, "pointer: leave event"); err != nil {
		return err
	}
	d := wire.NewArgDecoder(args)
	serial := d.Uint32("pointer: leave serial")
	surfaceID := d.Uint32("pointer: leave surface")
	if err := d.Err(); err != nil {
		return err
	}
	p.HandleLeave(serial, surfaceID)
	return nil
}

// handleMotionEvent processes wl_pointer.motion event (opcode 2).
func (p *Pointer) handleMotionEvent(args []wire.Argument) error {
	if err := wire.ParseArgMinLen(args, 3, "pointer: motion event"); err != nil {
		return err
	}
	d := wire.NewArgDecoder(args)
	time := d.Uint32("pointer: motion time")
	surfaceX := d.Int32("pointer: motion surface_x")
	surfaceY := d.Int32("pointer: motion surface_y")
	if err := d.Err(); err != nil {
		return err
	}
	p.HandleMotion(time, surfaceX, surfaceY)
	return nil
}

// handleButtonEvent processes wl_pointer.button event (opcode 3).
func (p *Pointer) handleButtonEvent(args []wire.Argument) error {
	if err := wire.ParseArgMinLen(args, 4, "pointer: button event"); err != nil {
		return err
	}
	d := wire.NewArgDecoder(args)
	serial := d.Uint32("pointer: button serial")
	time := d.Uint32("pointer: button time")
	button := d.Uint32("pointer: button code")
	state := d.Uint32("pointer: button state")
	if err := d.Err(); err != nil {
		return err
	}
	p.HandleButton(serial, time, button, state)
	return nil
}

// handleAxisEvent processes wl_pointer.axis event (opcode 4).
func (p *Pointer) handleAxisEvent(args []wire.Argument) error {
	if err := wire.ParseArgMinLen(args, 3, "pointer: axis event"); err != nil {
		return err
	}
	d := wire.NewArgDecoder(args)
	time := d.Uint32("pointer: axis time")
	axis := d.Uint32("pointer: axis type")
	value := d.Int32("pointer: axis value")
	if err := d.Err(); err != nil {
		return err
	}
	p.HandleAxis(time, axis, value)
	return nil
}

// handleAxisSourceEvent processes wl_pointer.axis_source event (opcode 5).
func (p *Pointer) handleAxisSourceEvent(args []wire.Argument) error {
	if err := wire.ParseArgMinLen(args, 1, "pointer: axis_source event"); err != nil {
		return err
	}
	d := wire.NewArgDecoder(args)
	axisSource := d.Uint32("pointer: axis_source")
	if err := d.Err(); err != nil {
		return err
	}
	p.HandleAxisSource(axisSource)
	return nil
}

// handleAxisStopEvent processes wl_pointer.axis_stop event (opcode 6).
func (p *Pointer) handleAxisStopEvent(args []wire.Argument) error {
	if err := wire.ParseArgMinLen(args, 2, "pointer: axis_stop event"); err != nil {
		return err
	}
	d := wire.NewArgDecoder(args)
	time := d.Uint32("pointer: axis_stop time")
	axis := d.Uint32("pointer: axis_stop axis")
	if err := d.Err(); err != nil {
		return err
	}
	p.HandleAxisStop(time, axis)
	return nil
}

// handleAxisDiscreteEvent processes wl_pointer.axis_discrete event (opcode 7).
func (p *Pointer) handleAxisDiscreteEvent(args []wire.Argument) error {
	if err := wire.ParseArgMinLen(args, 2, "pointer: axis_discrete event"); err != nil {
		return err
	}
	d := wire.NewArgDecoder(args)
	axis := d.Uint32("pointer: axis_discrete axis")
	discrete := d.Int32("pointer: axis_discrete discrete")
	if err := d.Err(); err != nil {
		return err
	}
	p.HandleAxisDiscrete(axis, discrete)
	return nil
}
