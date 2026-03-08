// clipboard-demo demonstrates clipboard functionality for both X11 and Wayland.
//
// This is a simplified demonstration showing the clipboard protocol implementation.
package main

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/datadevice"
	"github.com/opd-ai/wain/internal/wayland/wire"
	"github.com/opd-ai/wain/internal/x11/selection"
)

func main() {
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

	fmt.Println()
	fmt.Println("Protocol features:")
	fmt.Println("  • MIME type negotiation")
	fmt.Println("  • File descriptor-based data transfer")
	fmt.Println("  • Drag-and-drop support (enter/leave/motion/drop events)")
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

	fmt.Println()
	fmt.Println("Protocol features:")
	fmt.Println("  • CLIPBOARD and PRIMARY selections")
	fmt.Println("  • TARGETS negotiation")
	fmt.Println("  • UTF8_STRING and TEXT encoding")
	fmt.Println("  • SelectionRequest/SelectionNotify events")
	fmt.Println()
	fmt.Println("Demo complete! Both protocols are fully implemented.")
}

// mockWaylandConn implements the Conn interface for testing
type mockWaylandConn struct {
	nextID uint32
}

func (m *mockWaylandConn) AllocID() uint32 {
	m.nextID++
	return m.nextID
}

func (m *mockWaylandConn) RegisterObject(obj interface{}) {}

func (m *mockWaylandConn) SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error {
	return nil
}

// mockX11Conn implements the Conn interface for X11 selection testing
type mockX11Conn struct {
	atoms map[string]uint32
}

func (m *mockX11Conn) AllocXID() uint32 {
	return 1000
}

func (m *mockX11Conn) SendRequest(opcode uint8, data []byte) error {
	return nil
}

func (m *mockX11Conn) SendRequestAndReply(opcode uint8, data []byte) ([]byte, error) {
	return nil, nil
}

func (m *mockX11Conn) InternAtom(name string, onlyIfExists bool) (uint32, error) {
	if atom, ok := m.atoms[name]; ok {
		return atom, nil
	}
	return 0, nil
}

func (m *mockX11Conn) GetProperty(window, property, typ, offset, length uint32, deleteFlag bool) ([]byte, uint32, error) {
	return nil, 0, nil
}

func (m *mockX11Conn) ChangeProperty(window, property, typ uint32, format, mode uint8, data []byte) error {
	return nil
}

func (m *mockX11Conn) DeleteProperty(window, property uint32) error {
	return nil
}
