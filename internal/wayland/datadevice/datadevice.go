// Package datadevice implements the Wayland data device protocol for clipboard
// and drag-and-drop operations.
//
// The data device protocol consists of four main interfaces:
//   - wl_data_device_manager: Global factory for creating data devices
//   - wl_data_device: Per-seat interface for clipboard and drag-and-drop
//   - wl_data_source: Represents data we're offering (for copy/drag)
//   - wl_data_offer: Represents data being offered to us (for paste/drop)
//
// Reference: https://wayland.freedesktop.org/docs/html/apa.html#protocol-spec-wl_data_device_manager
package datadevice

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// Conn represents the subset of client.Connection methods needed by data device objects.
type Conn interface {
	AllocID() uint32
	RegisterObject(obj interface{})
	SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error
}

// objectBase provides common fields for data device-related Wayland objects.
type objectBase struct {
	id    uint32
	iface string
	conn  Conn
}

// ID returns the object's unique identifier.
func (o *objectBase) ID() uint32 {
	return o.id
}

// Interface returns the Wayland interface name.
func (o *objectBase) Interface() string {
	return o.iface
}

// Manager represents the wl_data_device_manager global interface.
type Manager struct {
	objectBase
}

// NewManager creates a new Manager from a registry binding.
func NewManager(conn Conn, id uint32) *Manager {
	return &Manager{
		objectBase: objectBase{
			id:    id,
			iface: "wl_data_device_manager",
			conn:  conn,
		},
	}
}

// HandleEvent processes events from the compositor (Manager has no events).
func (m *Manager) HandleEvent(opcode uint16, args []wire.Argument) error {
	return fmt.Errorf("wl_data_device_manager has no events (opcode %d)", opcode)
}

// CreateDataSource creates a new data source for offering data.
func (m *Manager) CreateDataSource() (*Source, error) {
	sourceID := m.conn.AllocID()

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: sourceID},
	}

	if err := m.conn.SendRequest(m.id, 0, args); err != nil {
		return nil, fmt.Errorf("create_data_source failed: %w", err)
	}

	source := NewSource(m.conn, sourceID)
	m.conn.RegisterObject(source)

	return source, nil
}

// GetDataDevice creates a data device for the specified seat.
func (m *Manager) GetDataDevice(seatID uint32) (*Device, error) {
	deviceID := m.conn.AllocID()

	args := []wire.Argument{
		{Type: wire.ArgTypeNewID, Value: deviceID},
		{Type: wire.ArgTypeObject, Value: seatID},
	}

	if err := m.conn.SendRequest(m.id, 1, args); err != nil {
		return nil, fmt.Errorf("get_data_device failed: %w", err)
	}

	device := NewDevice(m.conn, deviceID)
	m.conn.RegisterObject(device)

	return device, nil
}

// Source represents the wl_data_source interface.
type Source struct {
	objectBase
	mimeTypes      []string
	sendRequests   chan SendRequest
	cancelled      chan struct{}
	dndDropPerform chan struct{}
	dndFinished    chan struct{}
	dndAction      uint32
}

// SendRequest represents a request to send clipboard data.
type SendRequest struct {
	MimeType string
	FD       int
}

// NewSource creates a new data source object.
func NewSource(conn Conn, id uint32) *Source {
	return &Source{
		objectBase: objectBase{
			id:    id,
			iface: "wl_data_source",
			conn:  conn,
		},
		sendRequests:   make(chan SendRequest, 1),
		cancelled:      make(chan struct{}, 1),
		dndDropPerform: make(chan struct{}, 1),
		dndFinished:    make(chan struct{}, 1),
	}
}

// Offer announces a MIME type supported by this data source.
func (s *Source) Offer(mimeType string) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeString, Value: mimeType},
	}

	if err := s.conn.SendRequest(s.id, 0, args); err != nil {
		return fmt.Errorf("offer failed: %w", err)
	}

	s.mimeTypes = append(s.mimeTypes, mimeType)
	return nil
}

// Destroy destroys this data source.
func (s *Source) Destroy() error {
	return s.conn.SendRequest(s.id, 1, nil)
}

// SetActions sets the drag-and-drop actions supported by this source.
func (s *Source) SetActions(actions uint32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: actions},
	}
	return s.conn.SendRequest(s.id, 2, args)
}

// HandleEvent processes events from the compositor.
func (s *Source) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case 0: // target event
		// Sent during drag-and-drop to indicate accepted MIME type
		return nil
	case 1: // send event
		if len(args) != 2 {
			return fmt.Errorf("send event: expected 2 arguments, got %d", len(args))
		}
		mimeType := args[0].Value.(string)
		fd := int(args[1].Value.(int32))

		select {
		case s.sendRequests <- SendRequest{MimeType: mimeType, FD: fd}:
		default:
		}
		return nil
	case 2: // cancelled event
		select {
		case s.cancelled <- struct{}{}:
		default:
		}
		return nil
	case 3: // dnd_drop_performed event
		select {
		case s.dndDropPerform <- struct{}{}:
		default:
		}
		return nil
	case 4: // dnd_finished event
		select {
		case s.dndFinished <- struct{}{}:
		default:
		}
		return nil
	case 5: // action event
		if len(args) != 1 {
			return fmt.Errorf("action event: expected 1 argument, got %d", len(args))
		}
		s.dndAction = args[0].Value.(uint32)
		return nil
	default:
		return fmt.Errorf("unknown wl_data_source event opcode: %d", opcode)
	}
}

// SendRequests returns the channel for send requests.
func (s *Source) SendRequests() <-chan SendRequest {
	return s.sendRequests
}

// Cancelled returns the channel for cancellation events.
func (s *Source) Cancelled() <-chan struct{} {
	return s.cancelled
}

// Device represents the wl_data_device interface.
type Device struct {
	objectBase
	dataOffers     map[uint32]*Offer
	selection      *Offer
	enterSerial    uint32
	selectionChan  chan *Offer
	dragEnterChan  chan DragEnterEvent
	dragLeaveChan  chan struct{}
	dragMotionChan chan DragMotionEvent
	dropChan       chan struct{}
}

// DragEnterEvent contains drag-and-drop enter event data.
type DragEnterEvent struct {
	Serial  uint32
	Surface uint32
	X       float64
	Y       float64
	Offer   *Offer
}

// DragMotionEvent contains drag-and-drop motion event data.
type DragMotionEvent struct {
	Time uint32
	X    float64
	Y    float64
}

// NewDevice creates a new data device object.
func NewDevice(conn Conn, id uint32) *Device {
	return &Device{
		objectBase: objectBase{
			id:    id,
			iface: "wl_data_device",
			conn:  conn,
		},
		dataOffers:     make(map[uint32]*Offer),
		selectionChan:  make(chan *Offer, 1),
		dragEnterChan:  make(chan DragEnterEvent, 1),
		dragLeaveChan:  make(chan struct{}, 1),
		dragMotionChan: make(chan DragMotionEvent, 16),
		dropChan:       make(chan struct{}, 1),
	}
}

// SetSelection sets the clipboard selection to the given data source.
func (d *Device) SetSelection(source *Source, serial uint32) error {
	var sourceID uint32
	if source != nil {
		sourceID = source.id
	}

	args := []wire.Argument{
		{Type: wire.ArgTypeObject, Value: sourceID},
		{Type: wire.ArgTypeUint32, Value: serial},
	}

	return d.conn.SendRequest(d.id, 1, args)
}

// Release destroys this data device.
func (d *Device) Release() error {
	return d.conn.SendRequest(d.id, 2, nil)
}

// HandleEvent processes events from the compositor.
func (d *Device) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case 0: // data_offer event
		if len(args) != 1 {
			return fmt.Errorf("data_offer event: expected 1 argument, got %d", len(args))
		}
		offerID := args[0].Value.(uint32)
		offer := NewOffer(d.conn, offerID)
		d.dataOffers[offerID] = offer
		d.conn.RegisterObject(offer)
		return nil

	case 1: // enter event (drag-and-drop)
		if len(args) != 5 {
			return fmt.Errorf("enter event: expected 5 arguments, got %d", len(args))
		}
		serial := args[0].Value.(uint32)
		surface := args[1].Value.(uint32)
		x := args[2].Value.(float64)
		y := args[3].Value.(float64)
		offerID := args[4].Value.(uint32)

		d.enterSerial = serial
		event := DragEnterEvent{
			Serial:  serial,
			Surface: surface,
			X:       x,
			Y:       y,
			Offer:   d.dataOffers[offerID],
		}
		select {
		case d.dragEnterChan <- event:
		default:
		}
		return nil

	case 2: // leave event (drag-and-drop)
		select {
		case d.dragLeaveChan <- struct{}{}:
		default:
		}
		return nil

	case 3: // motion event (drag-and-drop)
		if len(args) != 3 {
			return fmt.Errorf("motion event: expected 3 arguments, got %d", len(args))
		}
		time := args[0].Value.(uint32)
		x := args[1].Value.(float64)
		y := args[2].Value.(float64)

		event := DragMotionEvent{
			Time: time,
			X:    x,
			Y:    y,
		}
		select {
		case d.dragMotionChan <- event:
		default:
		}
		return nil

	case 4: // drop event (drag-and-drop)
		select {
		case d.dropChan <- struct{}{}:
		default:
		}
		return nil

	case 5: // selection event (clipboard)
		var offer *Offer
		if len(args) == 1 && args[0].Value.(uint32) != 0 {
			offerID := args[0].Value.(uint32)
			offer = d.dataOffers[offerID]
		}
		d.selection = offer
		select {
		case d.selectionChan <- offer:
		default:
		}
		return nil

	default:
		return fmt.Errorf("unknown wl_data_device event opcode: %d", opcode)
	}
}

// Selection returns the current clipboard selection offer.
func (d *Device) Selection() *Offer {
	return d.selection
}

// SelectionChannel returns the channel for selection change events.
func (d *Device) SelectionChannel() <-chan *Offer {
	return d.selectionChan
}

// Offer represents the wl_data_offer interface.
type Offer struct {
	objectBase
	mimeTypes    []string
	mimeTypeChan chan string
	sourceChan   chan uint32
	actionChan   chan uint32
}

// NewOffer creates a new data offer object.
func NewOffer(conn Conn, id uint32) *Offer {
	return &Offer{
		objectBase: objectBase{
			id:    id,
			iface: "wl_data_offer",
			conn:  conn,
		},
		mimeTypeChan: make(chan string, 16),
		sourceChan:   make(chan uint32, 1),
		actionChan:   make(chan uint32, 1),
	}
}

// Accept announces we accept the drag-and-drop offer with the given MIME type.
func (o *Offer) Accept(serial uint32, mimeType string) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: serial},
		{Type: wire.ArgTypeString, Value: mimeType},
	}
	return o.conn.SendRequest(o.id, 0, args)
}

// Receive requests data for the given MIME type, returns an FD to read from.
func (o *Offer) Receive(mimeType string, fd int) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeString, Value: mimeType},
		{Type: wire.ArgTypeFD, Value: int32(fd)},
	}
	return o.conn.SendRequest(o.id, 1, args)
}

// Destroy destroys this offer.
func (o *Offer) Destroy() error {
	return o.conn.SendRequest(o.id, 2, nil)
}

// Finish indicates the drag-and-drop operation is complete.
func (o *Offer) Finish() error {
	return o.conn.SendRequest(o.id, 3, nil)
}

// SetActions sets the drag-and-drop actions we support.
func (o *Offer) SetActions(actions, preferredAction uint32) error {
	args := []wire.Argument{
		{Type: wire.ArgTypeUint32, Value: actions},
		{Type: wire.ArgTypeUint32, Value: preferredAction},
	}
	return o.conn.SendRequest(o.id, 4, args)
}

// HandleEvent processes events from the compositor.
func (o *Offer) HandleEvent(opcode uint16, args []wire.Argument) error {
	switch opcode {
	case 0: // offer event
		if len(args) != 1 {
			return fmt.Errorf("offer event: expected 1 argument, got %d", len(args))
		}
		mimeType := args[0].Value.(string)
		o.mimeTypes = append(o.mimeTypes, mimeType)
		select {
		case o.mimeTypeChan <- mimeType:
		default:
		}
		return nil
	case 1: // source_actions event
		if len(args) != 1 {
			return fmt.Errorf("source_actions event: expected 1 argument, got %d", len(args))
		}
		actions := args[0].Value.(uint32)
		select {
		case o.sourceChan <- actions:
		default:
		}
		return nil
	case 2: // action event
		if len(args) != 1 {
			return fmt.Errorf("action event: expected 1 argument, got %d", len(args))
		}
		action := args[0].Value.(uint32)
		select {
		case o.actionChan <- action:
		default:
		}
		return nil
	default:
		return fmt.Errorf("unknown wl_data_offer event opcode: %d", opcode)
	}
}

// MimeTypes returns the list of MIME types offered.
func (o *Offer) MimeTypes() []string {
	return o.mimeTypes
}

// MimeTypeChannel returns the channel for MIME type announcements.
func (o *Offer) MimeTypeChannel() <-chan string {
	return o.mimeTypeChan
}

// ReadData reads data from this offer for the given MIME type.
func (o *Offer) ReadData(mimeType string) ([]byte, error) {
	fds := make([]int, 2)
	if err := syscall.Pipe(fds); err != nil {
		return nil, fmt.Errorf("pipe failed: %w", err)
	}

	readFD := fds[0]
	writeFD := fds[1]

	if err := o.Receive(mimeType, writeFD); err != nil {
		syscall.Close(readFD)
		syscall.Close(writeFD)
		return nil, fmt.Errorf("receive failed: %w", err)
	}

	syscall.Close(writeFD)

	r := os.NewFile(uintptr(readFD), "pipe")
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	return data, nil
}
