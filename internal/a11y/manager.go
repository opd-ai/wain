package a11y

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
)

// basePath is the root D-Bus object path prefix for all accessible objects.
const basePath = "/org/a11y/atspi/accessible"

// atspiRegistryPath is the AT-SPI2 registry bus name and object path.
const atspiRegistryPath = "org.a11y.atspi.Registry"

// Manager manages the AT-SPI2 D-Bus registration for one wain application.
// It maintains a registry of all accessible objects and handles their D-Bus
// export lifecycle.
//
// All methods are safe to call from multiple goroutines.
type Manager struct {
	conn    *dbus.Conn
	mu      sync.RWMutex
	objects map[uint64]*AccessibleObject
	nextID  uint64
	appName string
}

// NewManager creates a Manager and connects to the D-Bus session bus.
// Returns an error when D-Bus is unavailable; callers should disable
// accessibility gracefully rather than aborting.
func NewManager(appName string) (*Manager, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("a11y: connect to session bus: %w", err)
	}
	m := &Manager{
		conn:    conn,
		objects: make(map[uint64]*AccessibleObject),
		appName: appName,
	}
	if err := m.registerWithRegistry(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("a11y: register with AT-SPI2 registry: %w", err)
	}
	return m, nil
}

// registerWithRegistry announces this application to the AT-SPI2 registry.
func (m *Manager) registerWithRegistry() error {
	busName := fmt.Sprintf("org.a11y.atspi.accessible.%s", m.appName)
	reply, err := m.conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("request bus name %q: %w", busName, err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("bus name %q already taken", busName)
	}
	return nil
}

// Close removes all exported objects and disconnects from D-Bus.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, obj := range m.objects {
		m.unexportObject(obj)
		delete(m.objects, id)
	}
	m.conn.Close()
}

// RegisterPanel adds a generic panel (container) accessible object.
// parentID 0 means this is a root-level object.
func (m *Manager) RegisterPanel(name string, parentID uint64) uint64 {
	return m.register(name, RolePanel, parentID, nil)
}

// RegisterButton adds a button accessible object with a click action.
func (m *Manager) RegisterButton(name string, parentID uint64, onClick func() bool) uint64 {
	acts := []objectAction{
		{name: "click", description: "Activate the button", do: onClick},
	}
	return m.register(name, RolePushButton, parentID, acts)
}

// RegisterLabel adds a static label accessible object.
func (m *Manager) RegisterLabel(text string, parentID uint64) uint64 {
	id := m.register(text, RoleLabel, parentID, nil)
	m.lookupObject(id).SetText(text)
	return id
}

// RegisterEntry adds an editable text field accessible object.
func (m *Manager) RegisterEntry(name string, parentID uint64) uint64 {
	acts := []objectAction{
		{name: "activate", description: "Focus the text field"},
	}
	id := m.register(name, RoleEntry, parentID, acts)
	obj := m.lookupObject(id)
	obj.mu.Lock()
	obj.enabled = true
	obj.mu.Unlock()
	return id
}

// RegisterScrollPane adds a scrollable container accessible object.
func (m *Manager) RegisterScrollPane(name string, parentID uint64) uint64 {
	return m.register(name, RoleScrollPane, parentID, nil)
}

// SetBounds updates the screen rectangle of the accessible object with the given ID.
func (m *Manager) SetBounds(id uint64, x, y, width, height int32) {
	if obj := m.lookupObject(id); obj != nil {
		obj.SetBounds(x, y, width, height)
	}
}

// SetFocused marks the object with the given ID as focused (or unfocused).
func (m *Manager) SetFocused(id uint64, focused bool) {
	if obj := m.lookupObject(id); obj != nil {
		obj.SetFocused(focused)
		if focused {
			m.emitFocusEvent(obj)
		}
	}
}

// SetText updates the text content for an Entry object.
func (m *Manager) SetText(id uint64, text string) {
	if obj := m.lookupObject(id); obj != nil {
		obj.SetText(text)
	}
}

// SetName updates the accessible name for the object with the given ID.
func (m *Manager) SetName(id uint64, name string) {
	if obj := m.lookupObject(id); obj != nil {
		obj.SetName(name)
	}
}

// lookupObject retrieves the AccessibleObject for the given ID, or nil.
func (m *Manager) lookupObject(id uint64) *AccessibleObject {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.objects[id]
}

// register creates, stores, and exports an AccessibleObject.
func (m *Manager) register(name string, role Role, parentID uint64, actions []objectAction) uint64 {
	m.mu.Lock()
	m.nextID++
	id := m.nextID
	obj := &AccessibleObject{
		id:      id,
		parentID: parentID,
		role:    role,
		name:    name,
		enabled: true,
		actions: actions,
		manager: m,
	}
	m.objects[id] = obj
	m.mu.Unlock()

	if parentID != 0 {
		if parent := m.lookupObject(parentID); parent != nil {
			parent.addChild(id)
		}
	}
	m.exportObject(obj)
	return id
}

// exportObject registers all four AT-SPI2 interfaces for obj on D-Bus.
func (m *Manager) exportObject(obj *AccessibleObject) {
	path := dbus.ObjectPath(obj.objectPath())
	m.conn.Export(&accessibleIface{obj}, path, "org.a11y.atspi.Accessible")
	m.conn.Export(&componentIface{obj}, path, "org.a11y.atspi.Component")
	m.conn.Export(&actionIface{obj}, path, "org.a11y.atspi.Action")
	m.conn.Export(&textIface{obj}, path, "org.a11y.atspi.Text")
}

// unexportObject removes D-Bus registrations for obj.
func (m *Manager) unexportObject(obj *AccessibleObject) {
	path := dbus.ObjectPath(obj.objectPath())
	m.conn.Export(nil, path, "org.a11y.atspi.Accessible")
	m.conn.Export(nil, path, "org.a11y.atspi.Component")
	m.conn.Export(nil, path, "org.a11y.atspi.Action")
	m.conn.Export(nil, path, "org.a11y.atspi.Text")
}

// emitFocusEvent emits the AT-SPI2 focus signal for the given object.
func (m *Manager) emitFocusEvent(obj *AccessibleObject) {
	path := dbus.ObjectPath(obj.objectPath())
	m.conn.Emit(path, "org.a11y.atspi.Event.Focus:Focus", uint32(0), uint32(0), "", nil)
}
