//go:build atspi
// +build atspi

package a11y

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

// actionIface exports org.a11y.atspi.Action for an AccessibleObject.
// Action allows assistive tools to trigger widget interactions programmatically.
type actionIface struct{ obj *AccessibleObject }

// GetNActions returns the number of actions available on this widget.
func (a *actionIface) GetNActions() (int32, *dbus.Error) {
	s := a.obj.snap()
	return int32(len(s.actions)), nil
}

// DoAction executes the action at the given index.
// Returns false if the index is out of range or the action fails.
func (a *actionIface) DoAction(index int32) (bool, *dbus.Error) {
	s := a.obj.snap()
	if index < 0 || int(index) >= len(s.actions) {
		return false, dbus.MakeFailedError(fmt.Errorf("action index %d out of range", index))
	}
	act := s.actions[index]
	if act.do == nil {
		return false, nil
	}
	return act.do(), nil
}

// GetName returns the name of the action at the given index.
func (a *actionIface) GetName(index int32) (string, *dbus.Error) {
	s := a.obj.snap()
	if index < 0 || int(index) >= len(s.actions) {
		return "", dbus.MakeFailedError(fmt.Errorf("action index %d out of range", index))
	}
	return s.actions[index].name, nil
}

// GetDescription returns the description of the action at the given index.
func (a *actionIface) GetDescription(index int32) (string, *dbus.Error) {
	s := a.obj.snap()
	if index < 0 || int(index) >= len(s.actions) {
		return "", dbus.MakeFailedError(fmt.Errorf("action index %d out of range", index))
	}
	return s.actions[index].description, nil
}

// GetKeyBinding returns the keyboard shortcut for the action at the given index.
func (a *actionIface) GetKeyBinding(index int32) (string, *dbus.Error) {
	s := a.obj.snap()
	if index < 0 || int(index) >= len(s.actions) {
		return "", dbus.MakeFailedError(fmt.Errorf("action index %d out of range", index))
	}
	return s.actions[index].keyBinding, nil
}
