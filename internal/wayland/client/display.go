package client

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Display represents the wl_display singleton object.
//
// The wl_display object is the core connection object to the compositor.
// It has object ID 1 and is created automatically when establishing a connection.
//
// wl_display provides:
//   - Sync: synchronization with the compositor
//   - GetRegistry: obtaining the global object registry
//
// Events:
//   - Error: fatal error notification
//   - DeleteID: object ID reuse notification
//
// Reference: https://wayland.freedesktop.org/docs/html/apa.html#protocol-spec-wl_display
type Display struct {
	baseObject
}

const (
	displayOpcodeSync        uint16 = 0
	displayOpcodeGetRegistry uint16 = 1
)

// Callback represents a wl_callback object used for synchronization.
type Callback struct {
	baseObject
	doneChan chan uint32
}

// Done returns a channel that receives the callback data when the compositor
// sends the done event.
func (cb *Callback) Done() <-chan uint32 {
	return cb.doneChan
}

// Sync creates a wl_callback object and sends a sync request to the compositor.
// The compositor will emit a done event when all preceding requests have been processed.
// This provides a synchronization point in the event stream.
func (d *Display) Sync() (*Callback, error) {
	callbackID := d.conn.allocID()

	cb := &Callback{
		baseObject: baseObject{
			id:    callbackID,
			iface: "wl_callback",
			conn:  d.conn,
		},
		doneChan: make(chan uint32, 1),
	}

	d.conn.registerObject(cb)

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: callbackID},
	}

	if err := d.conn.sendRequest(d.id, displayOpcodeSync, args); err != nil {
		return nil, fmt.Errorf("display: sync failed: %w", err)
	}

	return cb, nil
}

// GetRegistry creates a wl_registry object to discover global compositor objects.
func (d *Display) GetRegistry() (*Registry, error) {
	registryID := d.conn.allocID()

	registry := &Registry{
		baseObject: baseObject{
			id:    registryID,
			iface: "wl_registry",
			conn:  d.conn,
		},
		globals: make(map[uint32]*Global),
	}

	d.conn.registerObject(registry)

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: registryID},
	}

	if err := d.conn.sendRequest(d.id, displayOpcodeGetRegistry, args); err != nil {
		return nil, fmt.Errorf("display: get_registry failed: %w", err)
	}

	return registry, nil
}
