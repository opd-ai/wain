package xdg

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Toplevel represents an xdg_toplevel object.
//
// A toplevel provides the semantics of a desktop application window.
// It manages window properties like title, app ID, and handles window
// state changes from the compositor (resize, maximize, fullscreen, etc.).
//
// The toplevel receives configure events to notify the client of requested
// state changes. The client must respond by configuring its surface appropriately.
type Toplevel struct {
	objectBase
	surface       *Surface
	configureChan chan *ConfigureEvent
}

const (
	toplevelOpcodeDestroy         uint16 = 0
	toplevelOpcodeSetParent       uint16 = 1
	toplevelOpcodeSetTitle        uint16 = 2
	toplevelOpcodeSetAppID        uint16 = 3
	toplevelOpcodeShowWindowMenu  uint16 = 4
	toplevelOpcodeMove            uint16 = 5
	toplevelOpcodeResize          uint16 = 6
	toplevelOpcodeSetMaxSize      uint16 = 7
	toplevelOpcodeSetMinSize      uint16 = 8
	toplevelOpcodeSetMaximized    uint16 = 9
	toplevelOpcodeUnsetMaximized  uint16 = 10
	toplevelOpcodeSetFullscreen   uint16 = 11
	toplevelOpcodeUnsetFullscreen uint16 = 12
	toplevelOpcodeSetMinimized    uint16 = 13
)

const (
	toplevelEventConfigure uint16 = 0
	toplevelEventClose     uint16 = 1
)

// State represents the state of a toplevel window.
type State uint32

const (
	StateMaximized   State = 1
	StateFullscreen  State = 2
	StateResizing    State = 3
	StateActivated   State = 4
	StateTiledLeft   State = 5
	StateTiledRight  State = 6
	StateTiledTop    State = 7
	StateTiledBottom State = 8
)

// ConfigureEvent represents a configure event from the compositor.
//
// Configure events notify the client of the compositor's desired window
// state and size. The client should adjust its surface accordingly and
// acknowledge the configure.
type ConfigureEvent struct {
	Width  int32   // Requested width (0 means client chooses)
	Height int32   // Requested height (0 means client chooses)
	States []State // Array of state flags
}

// SetTitle sets the window title.
//
// The title is typically displayed in the window's title bar and task list.
//
// Parameters:
//   - title: the window title string
func (t *Toplevel) SetTitle(title string) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeString, Value: title},
	}

	if err := t.conn.SendRequest(t.id, toplevelOpcodeSetTitle, args); err != nil {
		return fmt.Errorf("xdg_toplevel: set_title failed: %w", err)
	}

	return nil
}

// SetAppID sets the application ID.
//
// The app ID is used to identify the application. It's typically a reverse
// domain name (e.g., "org.example.myapp") and is used by the compositor
// for grouping windows and accessing .desktop files.
//
// Parameters:
//   - appID: the application identifier string
func (t *Toplevel) SetAppID(appID string) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeString, Value: appID},
	}

	if err := t.conn.SendRequest(t.id, toplevelOpcodeSetAppID, args); err != nil {
		return fmt.Errorf("xdg_toplevel: set_app_id failed: %w", err)
	}

	return nil
}

// SetMinSize sets the minimum window size.
//
// This hints to the compositor the minimum size the window can be resized to.
// A value of 0 means no minimum.
//
// Parameters:
//   - width, height: minimum window dimensions in surface-local coordinates
func (t *Toplevel) SetMinSize(width, height int32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeInt32, Value: width},
		{Type: wire.ArgTypeInt32, Value: height},
	}

	if err := t.conn.SendRequest(t.id, toplevelOpcodeSetMinSize, args); err != nil {
		return fmt.Errorf("xdg_toplevel: set_min_size failed: %w", err)
	}

	return nil
}

// SetMaxSize sets the maximum window size.
//
// This hints to the compositor the maximum size the window can be resized to.
// A value of 0 means no maximum.
//
// Parameters:
//   - width, height: maximum window dimensions in surface-local coordinates
func (t *Toplevel) SetMaxSize(width, height int32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeInt32, Value: width},
		{Type: wire.ArgTypeInt32, Value: height},
	}

	if err := t.conn.SendRequest(t.id, toplevelOpcodeSetMaxSize, args); err != nil {
		return fmt.Errorf("xdg_toplevel: set_max_size failed: %w", err)
	}

	return nil
}

// SetMaximized requests the window to be maximized.
//
// This requests that the compositor maximize the window. The compositor
// will send a configure event if it accepts the request.
func (t *Toplevel) SetMaximized() error {
	if err := t.conn.SendRequest(t.id, toplevelOpcodeSetMaximized, nil); err != nil {
		return fmt.Errorf("xdg_toplevel: set_maximized failed: %w", err)
	}

	return nil
}

// UnsetMaximized requests the window to be unmaximized.
//
// This requests that the compositor restore the window from maximized state.
func (t *Toplevel) UnsetMaximized() error {
	if err := t.conn.SendRequest(t.id, toplevelOpcodeUnsetMaximized, nil); err != nil {
		return fmt.Errorf("xdg_toplevel: unset_maximized failed: %w", err)
	}

	return nil
}

// SetFullscreen requests the window to be fullscreen.
//
// This requests that the compositor make the window fullscreen. If output
// is non-zero, the window should be made fullscreen on that specific output.
//
// Parameters:
//   - output: object ID of the wl_output (0 for compositor choice)
func (t *Toplevel) SetFullscreen(output uint32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeObject, Value: output},
	}

	if err := t.conn.SendRequest(t.id, toplevelOpcodeSetFullscreen, args); err != nil {
		return fmt.Errorf("xdg_toplevel: set_fullscreen failed: %w", err)
	}

	return nil
}

// UnsetFullscreen requests the window to exit fullscreen.
func (t *Toplevel) UnsetFullscreen() error {
	if err := t.conn.SendRequest(t.id, toplevelOpcodeUnsetFullscreen, nil); err != nil {
		return fmt.Errorf("xdg_toplevel: unset_fullscreen failed: %w", err)
	}

	return nil
}

// SetMinimized requests the window to be minimized.
func (t *Toplevel) SetMinimized() error {
	if err := t.conn.SendRequest(t.id, toplevelOpcodeSetMinimized, nil); err != nil {
		return fmt.Errorf("xdg_toplevel: set_minimized failed: %w", err)
	}

	return nil
}

// HandleEvent processes events from the compositor for this Toplevel object.
func (t *Toplevel) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case toplevelEventConfigure:
		return t.handleConfigureEvent(args)
	case toplevelEventClose:
		// Close event has no arguments. Application should clean up.
		return nil
	default:
		return fmt.Errorf("unknown xdg_toplevel event opcode: %d", opcode)
	}
}

func (t *Toplevel) handleConfigureEvent(args []wire.Argument) error {
	if len(args) != 3 {
		return fmt.Errorf("configure event: expected 3 arguments, got %d", len(args))
	}

	if args[0].Type != wire.ArgTypeInt32 || args[1].Type != wire.ArgTypeInt32 || args[2].Type != wire.ArgTypeArray {
		return fmt.Errorf("configure event: invalid argument types")
	}

	width := args[0].Value.(int32)
	height := args[1].Value.(int32)
	statesData := args[2].Value.([]byte)

	// Parse states array (each state is a uint32).
	var states []State
	for i := 0; i+3 < len(statesData); i += 4 {
		stateVal := uint32(statesData[i]) |
			uint32(statesData[i+1])<<8 |
			uint32(statesData[i+2])<<16 |
			uint32(statesData[i+3])<<24
		states = append(states, State(stateVal))
	}

	event := &ConfigureEvent{
		Width:  width,
		Height: height,
		States: states,
	}

	if t.configureChan != nil {
		t.configureChan <- event
	}

	return nil
}

// Destroy destroys the toplevel.
//
// This should be called before destroying the associated xdg_surface.
func (t *Toplevel) Destroy() error {
	if err := t.conn.SendRequest(t.id, toplevelOpcodeDestroy, nil); err != nil {
		return fmt.Errorf("xdg_toplevel: destroy failed: %w", err)
	}

	return nil
}
