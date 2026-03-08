package datadevice

import (
	"testing"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// mockConn is a mock implementation of the Conn interface for testing.
type mockConn struct {
	nextID       uint32
	requests     []mockRequest
	objects      map[uint32]interface{}
}

type mockRequest struct {
	objectID uint32
	opcode   uint16
	args     []wire.Argument
}

func newMockConn() *mockConn {
	return &mockConn{
		nextID:   100,
		requests: make([]mockRequest, 0),
		objects:  make(map[uint32]interface{}),
	}
}

func (m *mockConn) AllocID() uint32 {
	id := m.nextID
	m.nextID++
	return id
}

func (m *mockConn) RegisterObject(obj interface{}) {
	if o, ok := obj.(interface{ ID() uint32 }); ok {
		m.objects[o.ID()] = obj
	}
}

func (m *mockConn) SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error {
	m.requests = append(m.requests, mockRequest{
		objectID: objectID,
		opcode:   opcode,
		args:     args,
	})
	return nil
}

func (m *mockConn) lastRequest() mockRequest {
	if len(m.requests) == 0 {
		return mockRequest{}
	}
	return m.requests[len(m.requests)-1]
}

func TestManagerCreateDataSource(t *testing.T) {
	conn := newMockConn()
	manager := NewManager(conn, 50)

	source, err := manager.CreateDataSource()
	if err != nil {
		t.Fatalf("CreateDataSource failed: %v", err)
	}

	if source.ID() != 100 {
		t.Errorf("Expected source ID 100, got %d", source.ID())
	}

	if source.Interface() != "wl_data_source" {
		t.Errorf("Expected interface wl_data_source, got %s", source.Interface())
	}

	req := conn.lastRequest()
	if req.objectID != 50 {
		t.Errorf("Expected request on object 50, got %d", req.objectID)
	}
	if req.opcode != 0 {
		t.Errorf("Expected opcode 0, got %d", req.opcode)
	}
}

func TestManagerGetDataDevice(t *testing.T) {
	conn := newMockConn()
	manager := NewManager(conn, 50)

	device, err := manager.GetDataDevice(25)
	if err != nil {
		t.Fatalf("GetDataDevice failed: %v", err)
	}

	if device.ID() != 100 {
		t.Errorf("Expected device ID 100, got %d", device.ID())
	}

	if device.Interface() != "wl_data_device" {
		t.Errorf("Expected interface wl_data_device, got %s", device.Interface())
	}

	req := conn.lastRequest()
	if req.opcode != 1 {
		t.Errorf("Expected opcode 1, got %d", req.opcode)
	}
	if len(req.args) != 2 {
		t.Fatalf("Expected 2 arguments, got %d", len(req.args))
	}
	if req.args[1].Value.(uint32) != 25 {
		t.Errorf("Expected seat ID 25, got %v", req.args[1].Value)
	}
}

func TestSourceOffer(t *testing.T) {
	conn := newMockConn()
	source := NewSource(conn, 100)

	err := source.Offer("text/plain")
	if err != nil {
		t.Fatalf("Offer failed: %v", err)
	}

	if len(source.mimeTypes) != 1 {
		t.Errorf("Expected 1 MIME type, got %d", len(source.mimeTypes))
	}

	if source.mimeTypes[0] != "text/plain" {
		t.Errorf("Expected text/plain, got %s", source.mimeTypes[0])
	}

	req := conn.lastRequest()
	if req.opcode != 0 {
		t.Errorf("Expected opcode 0, got %d", req.opcode)
	}
}

func TestSourceHandleEvent(t *testing.T) {
	conn := newMockConn()
	source := NewSource(conn, 100)

	// Test send event
	args := []wire.Argument{
		{Type: wire.ArgTypeString, Value: "text/plain"},
		{Type: wire.ArgTypeFD, Value: int32(5)},
	}

	err := source.HandleEvent(1, args)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	select {
	case req := <-source.SendRequests():
		if req.MimeType != "text/plain" {
			t.Errorf("Expected text/plain, got %s", req.MimeType)
		}
		if req.FD != 5 {
			t.Errorf("Expected FD 5, got %d", req.FD)
		}
	default:
		t.Error("Expected send request")
	}

	// Test cancelled event
	err = source.HandleEvent(2, nil)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	select {
	case <-source.Cancelled():
	default:
		t.Error("Expected cancellation event")
	}
}

func TestDeviceSetSelection(t *testing.T) {
	conn := newMockConn()
	device := NewDevice(conn, 100)
	source := NewSource(conn, 200)

	err := device.SetSelection(source, 12345)
	if err != nil {
		t.Fatalf("SetSelection failed: %v", err)
	}

	req := conn.lastRequest()
	if req.opcode != 1 {
		t.Errorf("Expected opcode 1, got %d", req.opcode)
	}
	if len(req.args) != 2 {
		t.Fatalf("Expected 2 arguments, got %d", len(req.args))
	}
	if req.args[0].Value.(uint32) != 200 {
		t.Errorf("Expected source ID 200, got %v", req.args[0].Value)
	}
	if req.args[1].Value.(uint32) != 12345 {
		t.Errorf("Expected serial 12345, got %v", req.args[1].Value)
	}
}

func TestDeviceHandleDataOffer(t *testing.T) {
	conn := newMockConn()
	device := NewDevice(conn, 100)

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: uint32(200)},
	}

	err := device.HandleEvent(0, args)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	offer := device.dataOffers[200]
	if offer == nil {
		t.Fatal("Expected offer to be registered")
	}

	if offer.ID() != 200 {
		t.Errorf("Expected offer ID 200, got %d", offer.ID())
	}
}

func TestDeviceHandleSelection(t *testing.T) {
	conn := newMockConn()
	device := NewDevice(conn, 100)

	// First create an offer
	device.dataOffers[200] = NewOffer(conn, 200)

	// Now send selection event
	args := []wire.Argument{
		{Type: wire.ArgTypeObject, Value: uint32(200)},
	}

	err := device.HandleEvent(5, args)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	if device.Selection() == nil {
		t.Error("Expected selection to be set")
	}

	if device.Selection().ID() != 200 {
		t.Errorf("Expected selection ID 200, got %d", device.Selection().ID())
	}

	select {
	case offer := <-device.SelectionChannel():
		if offer.ID() != 200 {
			t.Errorf("Expected offer ID 200, got %d", offer.ID())
		}
	default:
		t.Error("Expected selection event")
	}
}

func TestOfferHandleEvent(t *testing.T) {
	conn := newMockConn()
	offer := NewOffer(conn, 100)

	args := []wire.Argument{
		{Type: wire.ArgTypeString, Value: "text/plain"},
	}

	err := offer.HandleEvent(0, args)
	if err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	if len(offer.MimeTypes()) != 1 {
		t.Errorf("Expected 1 MIME type, got %d", len(offer.MimeTypes()))
	}

	if offer.MimeTypes()[0] != "text/plain" {
		t.Errorf("Expected text/plain, got %s", offer.MimeTypes()[0])
	}

	select {
	case mime := <-offer.MimeTypeChannel():
		if mime != "text/plain" {
			t.Errorf("Expected text/plain, got %s", mime)
		}
	default:
		t.Error("Expected MIME type event")
	}
}

func TestOfferAccept(t *testing.T) {
	conn := newMockConn()
	offer := NewOffer(conn, 100)

	err := offer.Accept(12345, "text/plain")
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}

	req := conn.lastRequest()
	if req.opcode != 0 {
		t.Errorf("Expected opcode 0, got %d", req.opcode)
	}
	if len(req.args) != 2 {
		t.Fatalf("Expected 2 arguments, got %d", len(req.args))
	}
}

func TestManagerInterface(t *testing.T) {
	conn := newMockConn()
	manager := NewManager(conn, 50)

	if manager.Interface() != "wl_data_device_manager" {
		t.Errorf("Expected wl_data_device_manager, got %s", manager.Interface())
	}

	if manager.ID() != 50 {
		t.Errorf("Expected ID 50, got %d", manager.ID())
	}
}
