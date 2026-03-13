package input

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Touch represents the wl_touch interface.
//
// The wl_touch interface represents a touchscreen device. It provides events
// for touch down, up, motion, and frame synchronization.
//
// Touch events are grouped into frames. Each frame may contain multiple touch
// point updates, and the frame event signals the end of a logical group.
type Touch struct {
	objectBase
}

const (
	touchOpcodeRelease uint16 = 0
)

const (
	touchEventDown        uint16 = 0
	touchEventUp          uint16 = 1
	touchEventMotion      uint16 = 2
	touchEventFrame       uint16 = 3
	touchEventCancel      uint16 = 4
	touchEventShape       uint16 = 5
	touchEventOrientation uint16 = 6
)

// Release destroys the touch object.
func (t *Touch) Release() error {
	return t.conn.SendRequest(t.id, touchOpcodeRelease, nil)
}

// HandleDown processes a touch down event from the compositor.
//
// This event is sent when a touch point first makes contact with the surface.
// The position is given in surface-local coordinates.
func (t *Touch) HandleDown(serial, time, surfaceID uint32, id, x, y int32) {
}

// HandleUp processes a touch up event from the compositor.
//
// This event is sent when a touch point is removed from the surface.
func (t *Touch) HandleUp(serial, time uint32, id int32) {
}

// HandleMotion processes a touch motion event from the compositor.
//
// This event is sent when a touch point moves across the surface.
func (t *Touch) HandleMotion(time uint32, id, x, y int32) {
}

// HandleFrame processes a touch frame event from the compositor.
//
// This event signals the end of a logical group of touch events. Applications
// should process all touch events in a frame atomically.
func (t *Touch) HandleFrame() {
}

// HandleCancel processes a touch cancel event from the compositor.
//
// This event is sent when a touch session is cancelled, for example when
// the compositor takes over touch handling.
func (t *Touch) HandleCancel() {
}

// HandleShape processes a touch shape event from the compositor.
//
// This event describes the shape of a touch point as an ellipse.
func (t *Touch) HandleShape(id, major, minor int32) {
}

// HandleOrientation processes a touch orientation event from the compositor.
//
// This event describes the orientation of a touch point in degrees.
func (t *Touch) HandleOrientation(id, orientation int32) {
}

// HandleEvent implements the EventHandler interface for wl_touch events.
func (t *Touch) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case touchEventDown:
		return t.handleDownEvent(args)
	case touchEventUp:
		return t.handleUpEvent(args)
	case touchEventMotion:
		return t.handleMotionEvent(args)
	case touchEventFrame:
		t.HandleFrame()
		return nil
	case touchEventCancel:
		t.HandleCancel()
		return nil
	case touchEventShape:
		return t.handleShapeEvent(args)
	case touchEventOrientation:
		return t.handleOrientationEvent(args)
	default:
		return fmt.Errorf("touch: unknown event opcode %d", opcode)
	}
}

// handleDownEvent processes wl_touch.down event (opcode 0).
func (t *Touch) handleDownEvent(args []wire.Argument) error {
	var serial, time, surfaceID uint32
	var id, x, y int32
	if err := parseEvent(args, 6, "touch: down event", func(d *wire.ArgDecoder) {
		serial = d.Uint32("touch: down serial")
		time = d.Uint32("touch: down time")
		surfaceID = d.Uint32("touch: down surface")
		id = d.Int32("touch: down id")
		x = d.Int32("touch: down x")
		y = d.Int32("touch: down y")
	}); err != nil {
		return err
	}
	t.HandleDown(serial, time, surfaceID, id, x, y)
	return nil
}

// handleUpEvent processes wl_touch.up event (opcode 1).
func (t *Touch) handleUpEvent(args []wire.Argument) error {
	var serial, time uint32
	var id int32
	if err := parseEvent(args, 3, "touch: up event", func(d *wire.ArgDecoder) {
		serial = d.Uint32("touch: up serial")
		time = d.Uint32("touch: up time")
		id = d.Int32("touch: up id")
	}); err != nil {
		return err
	}
	t.HandleUp(serial, time, id)
	return nil
}

// handleMotionEvent processes wl_touch.motion event (opcode 2).
func (t *Touch) handleMotionEvent(args []wire.Argument) error {
	var time uint32
	var id, x, y int32
	if err := parseEvent(args, 4, "touch: motion event", func(d *wire.ArgDecoder) {
		time = d.Uint32("touch: motion time")
		id = d.Int32("touch: motion id")
		x = d.Int32("touch: motion x")
		y = d.Int32("touch: motion y")
	}); err != nil {
		return err
	}
	t.HandleMotion(time, id, x, y)
	return nil
}

// handleShapeEvent processes wl_touch.shape event (opcode 4).
func (t *Touch) handleShapeEvent(args []wire.Argument) error {
	var id, major, minor int32
	if err := parseEvent(args, 3, "touch: shape event", func(d *wire.ArgDecoder) {
		id = d.Int32("touch: shape id")
		major = d.Int32("touch: shape major")
		minor = d.Int32("touch: shape minor")
	}); err != nil {
		return err
	}
	t.HandleShape(id, major, minor)
	return nil
}

// handleOrientationEvent processes wl_touch.orientation event (opcode 5).
func (t *Touch) handleOrientationEvent(args []wire.Argument) error {
	var id, orientation int32
	if err := parseEvent(args, 2, "touch: orientation event", func(d *wire.ArgDecoder) {
		id = d.Int32("touch: orientation id")
		orientation = d.Int32("touch: orientation value")
	}); err != nil {
		return err
	}
	t.HandleOrientation(id, orientation)
	return nil
}
