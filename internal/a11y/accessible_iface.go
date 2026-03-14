package a11y

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

// accessibleIface exports org.a11y.atspi.Accessible for an AccessibleObject.
type accessibleIface struct{ obj *AccessibleObject }

// GetName returns the accessible name of the widget.
func (a *accessibleIface) GetName() (string, *dbus.Error) {
	s := a.obj.snap()
	return s.name, nil
}

// GetDescription returns the accessible description of the widget.
func (a *accessibleIface) GetDescription() (string, *dbus.Error) {
	s := a.obj.snap()
	return s.description, nil
}

// GetRole returns the AT-SPI2 role constant for this widget.
func (a *accessibleIface) GetRole() (uint32, *dbus.Error) {
	s := a.obj.snap()
	return uint32(s.role), nil
}

// GetParent returns the D-Bus path of the parent object.
// Returns the base path if the object is the root.
func (a *accessibleIface) GetParent() (dbus.ObjectPath, *dbus.Error) {
	s := a.obj.snap()
	if s.parentID == 0 {
		return dbus.ObjectPath(basePath), nil
	}
	return dbus.ObjectPath(fmt.Sprintf("%s/%d", basePath, s.parentID)), nil
}

// GetChildCount returns the number of child accessible objects.
func (a *accessibleIface) GetChildCount() (int32, *dbus.Error) {
	s := a.obj.snap()
	return int32(len(s.childIDs)), nil
}

// GetChildAtIndex returns the D-Bus path of the child at the given index.
func (a *accessibleIface) GetChildAtIndex(index int32) (dbus.ObjectPath, *dbus.Error) {
	s := a.obj.snap()
	if index < 0 || int(index) >= len(s.childIDs) {
		return dbus.ObjectPath("/"), dbus.MakeFailedError(fmt.Errorf("index %d out of range", index))
	}
	return dbus.ObjectPath(fmt.Sprintf("%s/%d", basePath, s.childIDs[index])), nil
}

// GetChildren returns D-Bus paths for all child objects.
func (a *accessibleIface) GetChildren() ([]dbus.ObjectPath, *dbus.Error) {
	s := a.obj.snap()
	paths := make([]dbus.ObjectPath, len(s.childIDs))
	for i, id := range s.childIDs {
		paths[i] = dbus.ObjectPath(fmt.Sprintf("%s/%d", basePath, id))
	}
	return paths, nil
}

// GetIndexInParent returns this object's index within its parent's child list.
func (a *accessibleIface) GetIndexInParent() (int32, *dbus.Error) {
	s := a.obj.snap()
	if s.parentID == 0 {
		return 0, nil
	}
	parent := a.obj.manager.lookupObject(s.parentID)
	if parent == nil {
		return 0, nil
	}
	ps := parent.snap()
	for i, id := range ps.childIDs {
		if id == s.id {
			return int32(i), nil
		}
	}
	return 0, nil
}

// GetRelationSet returns an empty relation set (no relations defined).
func (a *accessibleIface) GetRelationSet() ([]interface{}, *dbus.Error) {
	return []interface{}{}, nil
}

// GetState returns the two-word AT-SPI2 state bitfield.
func (a *accessibleIface) GetState() ([]uint32, *dbus.Error) {
	return a.obj.states().Uint32s(), nil
}

// GetAttributes returns an empty attribute map.
func (a *accessibleIface) GetAttributes() (map[string]string, *dbus.Error) {
	return map[string]string{}, nil
}

// GetApplication returns the D-Bus path of the root application object.
func (a *accessibleIface) GetApplication() (dbus.ObjectPath, *dbus.Error) {
	return dbus.ObjectPath(basePath + "/root"), nil
}
