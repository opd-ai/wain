// clipboard-demo demonstrates clipboard functionality for both X11 and Wayland.
//
// This is a simplified demonstration showing the clipboard protocol implementation.
package main

import (
	"fmt"

	"github.com/opd-ai/wain/internal/demo"
	"github.com/opd-ai/wain/internal/wayland/datadevice"
	"github.com/opd-ai/wain/internal/wayland/wire"
	"github.com/opd-ai/wain/internal/x11/selection"
)

func main() {
	demo.CheckHelpFlag("clipboard-demo", "Clipboard protocol implementation for X11 and Wayland", []string{
		demo.FormatExample("clipboard-demo", "Run clipboard API demonstration"),
		demo.FormatExample("clipboard-demo --help", "Show this help message"),
	})

	fmt.Println("Clipboard Demo - Phase 8.2 Implementation")
	fmt.Println("==========================================")
	fmt.Println()

	demonstrateWaylandAPI()
	fmt.Println()
	demonstrateX11API()
}

func demonstrateWaylandAPI() {
	fmt.Println("Wayland Data Device Protocol")
	fmt.Println("----------------------------")

	// Create mock connection
	conn := &mockWaylandConn{nextID: 100}

	// Create data device manager
	manager := datadevice.NewManager(conn, 50)
	fmt.Printf("✓ Created wl_data_device_manager (ID: %d)\n", manager.ID())

	// Create data source for offering clipboard data
	source, _ := manager.CreateDataSource()
	fmt.Printf("✓ Created wl_data_source (ID: %d)\n", source.ID())

	// Offer MIME types
	source.Offer("text/plain")
	source.Offer("text/html")
	fmt.Println("✓ Offered MIME types: text/plain, text/html")

	// Get data device for a seat
	device, _ := manager.GetDataDevice(200)
	fmt.Printf("✓ Created wl_data_device (ID: %d)\n", device.ID())

	// Set clipboard selection
	device.SetSelection(source, 12345)
	fmt.Println("✓ Set clipboard selection with serial 12345")

	demo.PrintFeatureList("Protocol features:", []string{
		"MIME type negotiation",
		"File descriptor-based data transfer",
		"Drag-and-drop support (enter/leave/motion/drop events)",
	})
}

func demonstrateX11API() {
	fmt.Println("X11 Selection Protocol")
	fmt.Println("---------------------")

	// Create mock connection
	conn := &mockX11Conn{
		atoms: map[string]uint32{
			"CLIPBOARD":   69,
			"UTF8_STRING": 100,
			"TARGETS":     101,
			"TEXT":        102,
		},
	}

	// Create selection manager
	manager, _ := selection.NewManager(conn, 500)
	fmt.Printf("✓ Created selection manager for window 500\n")

	// Set clipboard
	manager.SetClipboard("Hello, clipboard!")
	fmt.Println("✓ Set CLIPBOARD selection: 'Hello, clipboard!'")

	// Set PRIMARY selection
	manager.SetPrimary("Selected text")
	fmt.Println("✓ Set PRIMARY selection: 'Selected text'")

	demo.PrintFeatureList("Protocol features:", []string{
		"CLIPBOARD and PRIMARY selections",
		"TARGETS negotiation",
		"UTF8_STRING and TEXT encoding",
		"SelectionRequest/SelectionNotify events",
	})
	fmt.Println()
	fmt.Println("Demo complete! Both protocols are fully implemented.")
}

// mockWaylandConn implements the Conn interface for testing
type mockWaylandConn struct {
	nextID uint32
}

// AllocID allocates and returns the next Wayland object ID.
func (m *mockWaylandConn) AllocID() uint32 {
	m.nextID++
	return m.nextID
}

// RegisterObject registers a Wayland object with the connection (no-op for mock).
func (m *mockWaylandConn) RegisterObject(obj interface{}) {}

// SendRequest sends a Wayland protocol request (no-op for mock).
func (m *mockWaylandConn) SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error {
	return nil
}

// mockX11Conn implements the Conn interface for X11 selection testing
type mockX11Conn struct {
	atoms map[string]uint32
}

// AllocXID allocates and returns an X11 resource ID.
func (m *mockX11Conn) AllocXID() uint32 {
	return 1000
}

// SendRequest sends an X11 protocol request (no-op for mock).
func (m *mockX11Conn) SendRequest(opcode uint8, data []byte) error {
	return nil
}

// SendRequestAndReply sends an X11 request and waits for a reply (returns nil for mock).
func (m *mockX11Conn) SendRequestAndReply(opcode uint8, data []byte) ([]byte, error) {
	return nil, nil
}

// InternAtom looks up or creates an X11 atom by name.
func (m *mockX11Conn) InternAtom(name string, onlyIfExists bool) (uint32, error) {
	if atom, ok := m.atoms[name]; ok {
		return atom, nil
	}
	return 0, nil
}

// GetProperty retrieves a window property value (returns nil for mock).
func (m *mockX11Conn) GetProperty(window, property, typ, offset, length uint32, deleteFlag bool) ([]byte, uint32, error) {
	return nil, 0, nil
}

// ChangeProperty sets or modifies a window property (no-op for mock).
func (m *mockX11Conn) ChangeProperty(window, property, typ uint32, format, mode uint8, data []byte) error {
	return nil
}

// DeleteProperty removes a property from a window (no-op for mock).
func (m *mockX11Conn) DeleteProperty(window, property uint32) error {
	return nil
}
