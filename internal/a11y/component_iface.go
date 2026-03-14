package a11y

import (
	"github.com/godbus/dbus/v5"
)

// componentIface exports org.a11y.atspi.Component for an AccessibleObject.
// Component provides spatial information about a widget's on-screen position.
type componentIface struct{ obj *AccessibleObject }

// extents holds the screen rectangle returned by Component.GetExtents.
type extents struct {
	X, Y, Width, Height int32
}

// point holds the screen position returned by Component.GetPosition.
type point struct{ X, Y int32 }

// size holds the dimensions returned by Component.GetSize.
type size struct{ Width, Height int32 }

// Contains reports whether screen coordinate (x, y) falls within the widget.
// coordType 0 = screen coordinates, 1 = window-relative coordinates.
func (c *componentIface) Contains(x, y int32, _ uint32) (bool, *dbus.Error) {
	s := c.obj.snap()
	return x >= s.x && x < s.x+s.width &&
		y >= s.y && y < s.y+s.height, nil
}

// GetAccessibleAtPoint returns the deepest accessible object at (x, y).
// Returns the object's own path when the point is inside its bounds.
func (c *componentIface) GetAccessibleAtPoint(x, y int32, _ uint32) (dbus.ObjectPath, *dbus.Error) {
	inside, _ := c.Contains(x, y, 0)
	if inside {
		return dbus.ObjectPath(c.obj.objectPath()), nil
	}
	return dbus.ObjectPath("/"), nil
}

// GetExtents returns the widget's bounding rectangle as {x, y, width, height}.
func (c *componentIface) GetExtents(_ uint32) (extents, *dbus.Error) {
	s := c.obj.snap()
	return extents{s.x, s.y, s.width, s.height}, nil
}

// GetPosition returns the widget's top-left corner as {x, y}.
func (c *componentIface) GetPosition(_ uint32) (point, *dbus.Error) {
	s := c.obj.snap()
	return point{s.x, s.y}, nil
}

// GetSize returns the widget's dimensions as {width, height}.
func (c *componentIface) GetSize() (size, *dbus.Error) {
	s := c.obj.snap()
	return size{s.width, s.height}, nil
}

// GrabFocus requests keyboard focus for this widget.
func (c *componentIface) GrabFocus() (bool, *dbus.Error) {
	c.obj.SetFocused(true)
	return true, nil
}

// ScrollTo is a no-op for non-scrollable widgets and always reports success.
func (c *componentIface) ScrollTo(_ uint32) (bool, *dbus.Error) {
	return true, nil
}
