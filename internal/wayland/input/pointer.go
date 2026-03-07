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

// HandleEnter processes an enter event from the compositor.
//
// This event is sent when the pointer enters a surface. The surface-local
// coordinates are provided in fixed-point format (multiply by 1/256).
func (p *Pointer) HandleEnter(serial, surfaceID uint32, surfaceX, surfaceY int32) {
}

// HandleLeave processes a leave event from the compositor.
//
// This event is sent when the pointer leaves a surface.
func (p *Pointer) HandleLeave(serial, surfaceID uint32) {
}

// HandleMotion processes a motion event from the compositor.
//
// This event is sent when the pointer moves. The coordinates are in
// surface-local coordinates in fixed-point format.
func (p *Pointer) HandleMotion(time uint32, surfaceX, surfaceY int32) {
}

// HandleButton processes a button event from the compositor.
//
// This event is sent when a pointer button is pressed or released.
func (p *Pointer) HandleButton(serial, time, button, state uint32) {
}

// HandleAxis processes an axis event from the compositor.
//
// This event is sent when a scroll or other axis event occurs. The value
// is in surface-local coordinates.
func (p *Pointer) HandleAxis(time, axis uint32, value int32) {
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
