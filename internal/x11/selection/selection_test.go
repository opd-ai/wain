package selection

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

// mockConn is a mock X11 connection for testing.
type mockConn struct {
	nextXID    uint32
	requests   []mockRequest
	atoms      map[string]uint32
	properties map[propertyKey]propertyValue
}

type mockRequest struct {
	opcode uint8
	data   []byte
}

type propertyKey struct {
	window   uint32
	property uint32
}

type propertyValue struct {
	data []byte
	typ  uint32
}

func newMockConn() *mockConn {
	return &mockConn{
		nextXID: 1000,
		requests: make([]mockRequest, 0),
		atoms: map[string]uint32{
			"CLIPBOARD":    69,
			"UTF8_STRING":  100,
			"TARGETS":      101,
			"TEXT":         102,
			"_WAIN_SELECTION": 103,
		},
		properties: make(map[propertyKey]propertyValue),
	}
}

func (m *mockConn) AllocXID() uint32 {
	xid := m.nextXID
	m.nextXID++
	return xid
}

func (m *mockConn) SendRequest(opcode uint8, data []byte) error {
	m.requests = append(m.requests, mockRequest{opcode: opcode, data: data})
	return nil
}

func (m *mockConn) SendRequestAndReply(opcode uint8, data []byte) ([]byte, error) {
	m.requests = append(m.requests, mockRequest{opcode: opcode, data: data})
	return nil, nil
}

func (m *mockConn) InternAtom(name string, onlyIfExists bool) (uint32, error) {
	if atom, ok := m.atoms[name]; ok {
		return atom, nil
	}
	if !onlyIfExists {
		atom := m.nextXID
		m.nextXID++
		m.atoms[name] = atom
		return atom, nil
	}
	return 0, nil
}

func (m *mockConn) GetProperty(window, property, typ uint32, offset, length uint32, deleteFlag bool) ([]byte, uint32, error) {
	key := propertyKey{window: window, property: property}
	if prop, ok := m.properties[key]; ok {
		if deleteFlag {
			delete(m.properties, key)
		}
		return prop.data, prop.typ, nil
	}
	return nil, 0, nil
}

func (m *mockConn) ChangeProperty(window, property, typ uint32, format uint8, mode uint8, data []byte) error {
	key := propertyKey{window: window, property: property}
	m.properties[key] = propertyValue{data: data, typ: typ}
	return nil
}

func (m *mockConn) DeleteProperty(window, property uint32) error {
	key := propertyKey{window: window, property: property}
	delete(m.properties, key)
	return nil
}

func (m *mockConn) lastRequest() mockRequest {
	if len(m.requests) == 0 {
		return mockRequest{}
	}
	return m.requests[len(m.requests)-1]
}

func TestNewManager(t *testing.T) {
	conn := newMockConn()

	mgr, err := NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr.window != 500 {
		t.Errorf("Expected window 500, got %d", mgr.window)
	}

	if mgr.clipboardAtom != 69 {
		t.Errorf("Expected clipboard atom 69, got %d", mgr.clipboardAtom)
	}

	if mgr.utf8Atom != 100 {
		t.Errorf("Expected UTF8_STRING atom 100, got %d", mgr.utf8Atom)
	}
}

func TestSetClipboard(t *testing.T) {
	conn := newMockConn()
	mgr, err := NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = mgr.SetClipboard("Hello, World!")
	if err != nil {
		t.Fatalf("SetClipboard failed: %v", err)
	}

	if !mgr.OwnsClipboard() {
		t.Error("Expected to own clipboard")
	}

	if string(mgr.clipboardData) != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got %s", string(mgr.clipboardData))
	}

	// Check SetSelectionOwner request was sent
	req := conn.lastRequest()
	if req.opcode != 22 {
		t.Errorf("Expected opcode 22 (SetSelectionOwner), got %d", req.opcode)
	}

	window := binary.LittleEndian.Uint32(req.data[0:4])
	if window != 500 {
		t.Errorf("Expected window 500, got %d", window)
	}

	selection := binary.LittleEndian.Uint32(req.data[4:8])
	if selection != 69 {
		t.Errorf("Expected selection 69 (CLIPBOARD), got %d", selection)
	}
}

func TestSetPrimary(t *testing.T) {
	conn := newMockConn()
	mgr, err := NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	err = mgr.SetPrimary("Primary text")
	if err != nil {
		t.Fatalf("SetPrimary failed: %v", err)
	}

	if !mgr.OwnsPrimary() {
		t.Error("Expected to own primary selection")
	}

	if string(mgr.primaryData) != "Primary text" {
		t.Errorf("Expected 'Primary text', got %s", string(mgr.primaryData))
	}

	req := conn.lastRequest()
	selection := binary.LittleEndian.Uint32(req.data[4:8])
	if selection != AtomPRIMARY {
		t.Errorf("Expected selection %d (PRIMARY), got %d", AtomPRIMARY, selection)
	}
}

func TestHandleSelectionRequest(t *testing.T) {
	conn := newMockConn()
	mgr, err := NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.SetClipboard("Test data")

	// Simulate a SelectionRequest event
	requestor := uint32(600)
	property := uint32(200)
	timestamp := uint32(time.Now().Unix())

	err = mgr.HandleSelectionRequest(requestor, mgr.clipboardAtom, mgr.utf8Atom, property, timestamp)
	if err != nil {
		t.Fatalf("HandleSelectionRequest failed: %v", err)
	}

	// Check that property was set on requestor window
	key := propertyKey{window: requestor, property: property}
	prop, ok := conn.properties[key]
	if !ok {
		t.Fatal("Expected property to be set on requestor window")
	}

	if string(prop.data) != "Test data" {
		t.Errorf("Expected 'Test data', got %s", string(prop.data))
	}

	// Check SendEvent request was sent
	found := false
	for _, req := range conn.requests {
		if req.opcode == 25 { // SendEvent
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected SendEvent request")
	}
}

func TestHandleSelectionRequestTargets(t *testing.T) {
	conn := newMockConn()
	mgr, err := NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	requestor := uint32(600)
	property := uint32(200)
	timestamp := uint32(time.Now().Unix())

	err = mgr.HandleSelectionRequest(requestor, mgr.clipboardAtom, mgr.targetsAtom, property, timestamp)
	if err != nil {
		t.Fatalf("HandleSelectionRequest failed: %v", err)
	}

	// Check that TARGETS property was set
	key := propertyKey{window: requestor, property: property}
	prop, ok := conn.properties[key]
	if !ok {
		t.Fatal("Expected property to be set")
	}

	// Should contain at least UTF8_STRING and TEXT atoms
	if len(prop.data) < 8 {
		t.Errorf("Expected at least 2 atoms (8 bytes), got %d bytes", len(prop.data))
	}
}

func TestHandleSelectionClear(t *testing.T) {
	conn := newMockConn()
	mgr, err := NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.SetClipboard("Test")
	mgr.SetPrimary("Primary")

	if !mgr.OwnsClipboard() || !mgr.OwnsPrimary() {
		t.Fatal("Expected to own both selections")
	}

	mgr.HandleSelectionClear(mgr.clipboardAtom)

	if mgr.OwnsClipboard() {
		t.Error("Expected not to own clipboard after clear")
	}

	if !mgr.OwnsPrimary() {
		t.Error("Expected to still own primary selection")
	}

	mgr.HandleSelectionClear(AtomPRIMARY)

	if mgr.OwnsPrimary() {
		t.Error("Expected not to own primary after clear")
	}
}

func TestGetClipboard(t *testing.T) {
	conn := newMockConn()
	mgr, err := NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Set up mock property data
	propertyAtom := conn.atoms["_WAIN_SELECTION"]
	key := propertyKey{window: 500, property: propertyAtom}
	conn.properties[key] = propertyValue{
		data: []byte("Retrieved data"),
		typ:  100,
	}

	// This will send ConvertSelection and then read the property
	text, err := mgr.GetClipboard()
	if err != nil {
		t.Fatalf("GetClipboard failed: %v", err)
	}

	if text != "Retrieved data" {
		t.Errorf("Expected 'Retrieved data', got %s", text)
	}

	// Check ConvertSelection request was sent
	found := false
	for _, req := range conn.requests {
		if req.opcode == 24 { // ConvertSelection
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected ConvertSelection request")
	}
}

func TestManagerAtoms(t *testing.T) {
	conn := newMockConn()
	mgr, err := NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Verify atoms were interned correctly
	if mgr.clipboardAtom == 0 {
		t.Error("CLIPBOARD atom not interned")
	}
	if mgr.utf8Atom == 0 {
		t.Error("UTF8_STRING atom not interned")
	}
	if mgr.targetsAtom == 0 {
		t.Error("TARGETS atom not interned")
	}
	if mgr.textAtom == 0 {
		t.Error("TEXT atom not interned")
	}
}

func TestConvertSelectionData(t *testing.T) {
	conn := newMockConn()
	mgr, err := NewManager(conn, 500)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	testData := "Test clipboard data with unicode: ñ,ü,é,中文"
	mgr.SetClipboard(testData)

	if !bytes.Equal(mgr.clipboardData, []byte(testData)) {
		t.Errorf("Clipboard data mismatch")
	}
}
