package input

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
