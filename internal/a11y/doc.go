// Package a11y implements AT-SPI2 accessibility support for wain applications.
//
// AT-SPI2 (Assistive Technology Service Provider Interface 2) is the standard
// Linux accessibility protocol. It enables screen readers (Orca), magnifiers,
// and other assistive tools to interact with applications over D-Bus.
//
// # Architecture
//
// A Manager connects to the D-Bus session bus and registers the application
// with the AT-SPI2 registry. Each widget is represented by an AccessibleObject
// exported as a D-Bus object at a unique path.
//
// Four AT-SPI2 interfaces are implemented per object:
//   - org.a11y.atspi.Accessible — name, role, parent/child navigation, state
//   - org.a11y.atspi.Component — bounds and hit testing
//   - org.a11y.atspi.Action — activatable actions (click, focus)
//   - org.a11y.atspi.Text — text content and caret position
//
// # Graceful Degradation
//
// If D-Bus is unavailable (headless server, missing dbus-daemon), NewManager
// returns an error and the application continues normally without accessibility.
//
// # Usage
//
//	mgr, err := a11y.NewManager("my-app")
//	if err != nil {
//	    log.Printf("a11y: disabled: %v", err)
//	    return // accessibility not available
//	}
//	defer mgr.Close()
//
//	rootID := mgr.RegisterPanel("root", 0)
//	btnID  := mgr.RegisterButton("OK", rootID, func() bool { return true })
//	mgr.SetBounds(btnID, 10, 10, 80, 30)
package a11y
