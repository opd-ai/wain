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

func (t *Touch) handleDownEvent(args []wire.Argument) error {
	if len(args) < 6 {
		return fmt.Errorf("touch: down event requires 6 arguments, got %d", len(args))
	}
	serial, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("touch: down serial must be uint32")
	}
	time, ok := args[1].Value.(uint32)
	if !ok {
		return fmt.Errorf("touch: down time must be uint32")
	}
	surfaceID, ok := args[2].Value.(uint32)
	if !ok {
		return fmt.Errorf("touch: down surface must be uint32")
	}
	id, ok := args[3].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: down id must be int32")
	}
	x, ok := args[4].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: down x must be fixed")
	}
	y, ok := args[5].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: down y must be fixed")
	}
	t.HandleDown(serial, time, surfaceID, id, x, y)
	return nil
}

func (t *Touch) handleUpEvent(args []wire.Argument) error {
	if len(args) < 3 {
		return fmt.Errorf("touch: up event requires 3 arguments, got %d", len(args))
	}
	serial, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("touch: up serial must be uint32")
	}
	time, ok := args[1].Value.(uint32)
	if !ok {
		return fmt.Errorf("touch: up time must be uint32")
	}
	id, ok := args[2].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: up id must be int32")
	}
	t.HandleUp(serial, time, id)
	return nil
}

func (t *Touch) handleMotionEvent(args []wire.Argument) error {
	if len(args) < 4 {
		return fmt.Errorf("touch: motion event requires 4 arguments, got %d", len(args))
	}
	time, ok := args[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("touch: motion time must be uint32")
	}
	id, ok := args[1].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: motion id must be int32")
	}
	x, ok := args[2].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: motion x must be fixed")
	}
	y, ok := args[3].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: motion y must be fixed")
	}
	t.HandleMotion(time, id, x, y)
	return nil
}

func (t *Touch) handleShapeEvent(args []wire.Argument) error {
	if len(args) < 3 {
		return fmt.Errorf("touch: shape event requires 3 arguments, got %d", len(args))
	}
	id, ok := args[0].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: shape id must be int32")
	}
	major, ok := args[1].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: shape major must be fixed")
	}
	minor, ok := args[2].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: shape minor must be fixed")
	}
	t.HandleShape(id, major, minor)
	return nil
}

func (t *Touch) handleOrientationEvent(args []wire.Argument) error {
	if len(args) < 2 {
		return fmt.Errorf("touch: orientation event requires 2 arguments, got %d", len(args))
	}
	id, ok := args[0].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: orientation id must be int32")
	}
	orientation, ok := args[1].Value.(int32)
	if !ok {
		return fmt.Errorf("touch: orientation value must be fixed")
	}
	t.HandleOrientation(id, orientation)
	return nil
}
