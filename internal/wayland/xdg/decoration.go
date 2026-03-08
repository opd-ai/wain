// Package xdg provides client-side Wayland XDG shell protocol support.
package xdg

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// DecorationMode indicates whether decorations are client-side or server-side.
type DecorationMode uint32

const (
	DecorationModeClientSide DecorationMode = 1
	DecorationModeServerSide DecorationMode = 2
)

// DecorationManager manages the zxdg_decoration_manager_v1 protocol.
type DecorationManager struct {
	objectBase
	version uint32
}

// NewDecorationManager creates a decoration manager object.
func NewDecorationManager(conn Conn, id, version uint32) *DecorationManager {
	return &DecorationManager{
		objectBase: objectBase{
			id:    id,
			iface: "zxdg_decoration_manager_v1",
			conn:  conn,
		},
		version: version,
	}
}

// Destroy releases the decoration manager.
func (d *DecorationManager) Destroy() error {
	return d.conn.SendRequest(d.id, 0, nil)
}

// GetToplevelDecoration creates a decoration object for a toplevel.
func (d *DecorationManager) GetToplevelDecoration(toplevel *Toplevel) (*ToplevelDecoration, error) {
	newID := d.conn.AllocID()
	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: newID},
		{Type: wire.ArgTypeObject, Value: toplevel.ID()},
	}
	if err := d.conn.SendRequest(d.id, 1, args); err != nil {
		return nil, err
	}
	return NewToplevelDecoration(d.conn, newID, d.version), nil
}

// ToplevelDecoration represents a zxdg_toplevel_decoration_v1 object.
type ToplevelDecoration struct {
	objectBase
	version uint32
	mode    DecorationMode
}

// NewToplevelDecoration creates a toplevel decoration object.
func NewToplevelDecoration(conn Conn, id, version uint32) *ToplevelDecoration {
	return &ToplevelDecoration{
		objectBase: objectBase{
			id:    id,
			iface: "zxdg_toplevel_decoration_v1",
			conn:  conn,
		},
		version: version,
	}
}

// Destroy releases the toplevel decoration.
func (t *ToplevelDecoration) Destroy() error {
	return t.conn.SendRequest(t.id, 0, nil)
}

// SetMode requests a specific decoration mode.
func (t *ToplevelDecoration) SetMode(mode DecorationMode) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: uint32(mode)},
	}
	return t.conn.SendRequest(t.id, 1, args)
}

// UnsetMode requests the compositor choose the decoration mode.
func (t *ToplevelDecoration) UnsetMode() error {
	return t.conn.SendRequest(t.id, 2, nil)
}

// Mode returns the current decoration mode.
func (t *ToplevelDecoration) Mode() DecorationMode {
	return t.mode
}

// HandleEvent processes events from the compositor.
func (t *ToplevelDecoration) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case 0: // configure
		if len(args) < 1 {
			return fmt.Errorf("configure event missing mode argument")
		}
		mode := args[0].Value.(uint32)
		t.mode = DecorationMode(mode)
		return nil
	default:
		return fmt.Errorf("unknown toplevel decoration event opcode: %d", opcode)
	}
}
