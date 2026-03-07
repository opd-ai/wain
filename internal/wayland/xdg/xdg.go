// Package xdg implements the XDG shell protocol for Wayland.
//
// The XDG shell protocol provides a standard way to create application windows
// with proper window management capabilities including:
//   - Window positioning and sizing
//   - Window decoration negotiation
//   - Popup and menu handling
//   - Surface roles and lifecycle
//
// This implementation includes:
//   - xdg_wm_base: Base window management interface
//   - xdg_surface: Surface role assignment
//   - xdg_toplevel: Top-level window interface
//
// Protocol specification:
// https://wayland.app/protocols/xdg-shell
package xdg

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Conn represents the subset of client.Connection methods needed by XDG objects.
// This interface allows XDG objects to interact with the Wayland connection
// without creating a circular dependency on the client package.
type Conn interface {
	AllocID() uint32
	RegisterObject(obj interface{})
	SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error
}

// objectBase provides common fields for XDG-related Wayland objects.
type objectBase struct {
	id    uint32
	iface string
	conn  Conn
}

func (o *objectBase) ID() uint32 {
	return o.id
}

func (o *objectBase) Interface() string {
	return o.iface
}

// WmBase represents the xdg_wm_base global interface.
//
// The xdg_wm_base interface is the base interface for XDG shell protocol.
// It allows clients to turn their wl_surfaces into windows in a desktop
// environment.
//
// The compositor must send a ping event periodically, and the client must
// respond with a pong to demonstrate liveness.
type WmBase struct {
	objectBase
	version uint32
}

const (
	wmBaseOpcodeDestroy          uint16 = 0
	wmBaseOpcodeCreatePositioner uint16 = 1
	wmBaseOpcodeGetXdgSurface    uint16 = 2
	wmBaseOpcodePong             uint16 = 3
)

const (
	wmBaseEventPing uint16 = 0
)

// NewWmBase creates a new WmBase object from a registry binding.
func NewWmBase(conn Conn, id uint32, version uint32) *WmBase {
	return &WmBase{
		objectBase: objectBase{
			id:    id,
			iface: "xdg_wm_base",
			conn:  conn,
		},
		version: version,
	}
}

// GetXdgSurface creates an xdg_surface for the given wl_surface.
//
// This assigns the XDG surface role to a wl_surface, making it suitable
// for use as a window. The surface must not already have a role assigned.
//
// Parameters:
//   - surfaceID: object ID of the wl_surface to assign the role to
func (w *WmBase) GetXdgSurface(surfaceID uint32) (*Surface, error) {
	xdgSurfaceID := w.conn.AllocID()

	xdgSurface := &Surface{
		objectBase: objectBase{
			id:    xdgSurfaceID,
			iface: "xdg_surface",
			conn:  w.conn,
		},
		wmBase:        w,
		configureChan: make(chan uint32, 8),
	}

	w.conn.RegisterObject(xdgSurface)

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: xdgSurfaceID},
		{Type: wire.ArgTypeObject, Value: surfaceID},
	}

	if err := w.conn.SendRequest(w.id, wmBaseOpcodeGetXdgSurface, args); err != nil {
		return nil, fmt.Errorf("xdg_wm_base: get_xdg_surface failed: %w", err)
	}

	return xdgSurface, nil
}

// Pong responds to a ping event from the compositor.
//
// The compositor sends periodic ping events to verify client responsiveness.
// Clients must respond with pong using the same serial number.
//
// Parameters:
//   - serial: the serial number from the ping event
func (w *WmBase) Pong(serial uint32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: serial},
	}

	if err := w.conn.SendRequest(w.id, wmBaseOpcodePong, args); err != nil {
		return fmt.Errorf("xdg_wm_base: pong failed: %w", err)
	}

	return nil
}

// HandleEvent processes events from the compositor for this WmBase object.
func (w *WmBase) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case wmBaseEventPing:
		if len(args) != 1 {
			return fmt.Errorf("xdg_wm_base ping event: expected 1 argument, got %d", len(args))
		}
		if args[0].Type != wire.ArgTypeUint32 {
			return fmt.Errorf("xdg_wm_base ping event: expected uint32 argument")
		}
		serial := args[0].Value.(uint32)
		// Auto-respond to ping with pong.
		return w.Pong(serial)
	default:
		return fmt.Errorf("unknown xdg_wm_base event opcode: %d", opcode)
	}
}

// Surface represents an xdg_surface object.
//
// An xdg_surface is a surface with an assigned XDG role. It serves as the
// base for toplevel windows and popups. The surface must be configured with
// a specific role (toplevel or popup) before it can be used.
//
// The client must respond to configure events by calling AckConfigure with
// the provided serial number.
type Surface struct {
	objectBase
	wmBase        *WmBase
	configureChan chan uint32
}

const (
	surfaceOpcodeDestroy           uint16 = 0
	surfaceOpcodeGetToplevel       uint16 = 1
	surfaceOpcodeGetPopup          uint16 = 2
	surfaceOpcodeSetWindowGeometry uint16 = 3
	surfaceOpcodeAckConfigure      uint16 = 4
)

const (
	surfaceEventConfigure uint16 = 0
)

// GetToplevel creates a toplevel surface for this xdg_surface.
//
// A toplevel surface represents a top-level window (application window).
// This should be called once to assign the toplevel role to the surface.
func (s *Surface) GetToplevel() (*Toplevel, error) {
	toplevelID := s.conn.AllocID()

	toplevel := &Toplevel{
		objectBase: objectBase{
			id:    toplevelID,
			iface: "xdg_toplevel",
			conn:  s.conn,
		},
		surface:       s,
		configureChan: make(chan *ConfigureEvent, 8),
	}

	s.conn.RegisterObject(toplevel)

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: toplevelID},
	}

	if err := s.conn.SendRequest(s.id, surfaceOpcodeGetToplevel, args); err != nil {
		return nil, fmt.Errorf("xdg_surface: get_toplevel failed: %w", err)
	}

	return toplevel, nil
}

// SetWindowGeometry sets the window geometry for this surface.
//
// The window geometry defines the visible bounds of the window's content,
// excluding any client-side decorations (shadows, borders, etc.).
//
// Parameters:
//   - x, y: offset of the content area from the surface origin
//   - width, height: size of the content area
func (s *Surface) SetWindowGeometry(x, y, width, height int32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeInt32, Value: x},
		{Type: wire.ArgTypeInt32, Value: y},
		{Type: wire.ArgTypeInt32, Value: width},
		{Type: wire.ArgTypeInt32, Value: height},
	}

	if err := s.conn.SendRequest(s.id, surfaceOpcodeSetWindowGeometry, args); err != nil {
		return fmt.Errorf("xdg_surface: set_window_geometry failed: %w", err)
	}

	return nil
}

// AckConfigure acknowledges a configure event.
//
// The client must acknowledge every configure event by calling this method
// with the serial from the configure event. This must be done before the
// next commit on the associated wl_surface.
//
// Parameters:
//   - serial: the serial number from the configure event
func (s *Surface) AckConfigure(serial uint32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: serial},
	}

	if err := s.conn.SendRequest(s.id, surfaceOpcodeAckConfigure, args); err != nil {
		return fmt.Errorf("xdg_surface: ack_configure failed: %w", err)
	}

	return nil
}

// HandleEvent processes events from the compositor for this Surface object.
func (s *Surface) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case surfaceEventConfigure:
		if len(args) != 1 {
			return fmt.Errorf("xdg_surface configure event: expected 1 argument, got %d", len(args))
		}
		if args[0].Type != wire.ArgTypeUint32 {
			return fmt.Errorf("xdg_surface configure event: expected uint32 argument")
		}
		serial := args[0].Value.(uint32)
		if s.configureChan != nil {
			s.configureChan <- serial
		}
		return nil
	default:
		return fmt.Errorf("unknown xdg_surface event opcode: %d", opcode)
	}
}

// Destroy destroys the xdg_surface.
func (s *Surface) Destroy() error {
	if err := s.conn.SendRequest(s.id, surfaceOpcodeDestroy, nil); err != nil {
		return fmt.Errorf("xdg_surface: destroy failed: %w", err)
	}

	return nil
}
