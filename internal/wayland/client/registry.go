package client

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Registry represents a wl_registry object used to discover global compositor objects.
//
// The registry advertises global objects that the compositor supports, such as
// wl_compositor, wl_seat, wl_output, etc. Clients bind to these globals to
// obtain typed interface objects.
//
// Events:
//   - Global: announces a new global object
//   - GlobalRemove: announces removal of a global object
//
// Reference: https://wayland.freedesktop.org/docs/html/apa.html#protocol-spec-wl_registry
type Registry struct {
	baseObject
	globals map[uint32]*Global
}

const (
	registryOpcodeBind uint16 = 0
)

// Global represents a global compositor object advertised by the registry.
type Global struct {
	Name      uint32 // Unique numeric name for this global
	Interface string // Interface name (e.g., "wl_compositor")
	Version   uint32 // Interface version supported by the compositor
}

// Globals returns a copy of all currently known globals.
func (r *Registry) Globals() map[uint32]*Global {
	result := make(map[uint32]*Global, len(r.globals))
	for k, v := range r.globals {
		global := *v
		result[k] = &global
	}
	return result
}

// FindGlobal searches for a global by interface name.
// Returns the first matching global, or nil if not found.
func (r *Registry) FindGlobal(iface string) *Global {
	for _, global := range r.globals {
		if global.Interface == iface {
			return global
		}
	}
	return nil
}

// Bind binds to a global object and creates a typed client-side object.
//
// Parameters:
//   - name: the global's unique numeric name (from Global.Name)
//   - iface: the interface name to bind to
//   - version: the interface version to use
//
// The returned object ID can be used to create typed objects (compositor, seat, etc.).
func (r *Registry) Bind(name uint32, iface string, version uint32) (uint32, error) {
	objectID := r.conn.allocID()

	// Note: The interface name is sent as a string in the bind request.
	// The actual interface type must be known to properly construct the
	// typed object (Compositor, Seat, etc.).
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: name},
		{Type: wire.ArgTypeString, Value: iface},
		{Type: wire.ArgTypeUint32, Value: version},
		{Type: wire.ArgTypeNewID, Value: objectID},
	}

	if err := r.conn.sendRequest(r.id, registryOpcodeBind, args); err != nil {
		return 0, fmt.Errorf("registry: bind failed: %w", err)
	}

	return objectID, nil
}

// BindCompositor is a helper that binds to the wl_compositor interface.
func (r *Registry) BindCompositor(global *Global) (*Compositor, error) {
	if global.Interface != "wl_compositor" {
		return nil, fmt.Errorf("registry: not a compositor: %s", global.Interface)
	}

	objectID, err := r.Bind(global.Name, global.Interface, global.Version)
	if err != nil {
		return nil, err
	}

	compositor := &Compositor{
		baseObject: baseObject{
			id:    objectID,
			iface: "wl_compositor",
			conn:  r.conn,
		},
		version: global.Version,
	}

	r.conn.registerObject(compositor)

	return compositor, nil
}

// addGlobal is called internally when a global event is received.
func (r *Registry) addGlobal(name uint32, iface string, version uint32) {
	r.globals[name] = &Global{
		Name:      name,
		Interface: iface,
		Version:   version,
	}
}

// removeGlobal is called internally when a global_remove event is received.
func (r *Registry) removeGlobal(name uint32) {
	delete(r.globals, name)
}
