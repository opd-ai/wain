package input

import (
	"testing"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

type mockConn struct {
	nextID       uint32
	objects      []interface{}
	requests     []mockRequest
	registerCall bool
}

type mockRequest struct {
	objectID uint32
	opcode   uint16
	args     []wire.Argument
}

func (m *mockConn) AllocID() uint32 {
	m.nextID++
	return m.nextID
}

func (m *mockConn) RegisterObject(obj interface{}) {
	m.objects = append(m.objects, obj)
	m.registerCall = true
}

func (m *mockConn) SendRequest(objectID uint32, opcode uint16, args []wire.Argument) error {
	m.requests = append(m.requests, mockRequest{
		objectID: objectID,
		opcode:   opcode,
		args:     args,
	})
	return nil
}

func TestNewSeat(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)

	if seat.ID() != 42 {
		t.Errorf("ID() = %d, want 42", seat.ID())
	}
	if seat.Interface() != "wl_seat" {
		t.Errorf("Interface() = %q, want %q", seat.Interface(), "wl_seat")
	}
	if seat.version != 7 {
		t.Errorf("version = %d, want 7", seat.version)
	}
}

func TestSeatGetPointer(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)

	pointer, err := seat.GetPointer()
	if err != nil {
		t.Fatalf("GetPointer() error = %v", err)
	}

	if pointer == nil {
		t.Fatal("GetPointer() returned nil")
	}
	if pointer.ID() != 2 {
		t.Errorf("pointer.ID() = %d, want 2", pointer.ID())
	}
	if pointer.Interface() != "wl_pointer" {
		t.Errorf("pointer.Interface() = %q, want %q", pointer.Interface(), "wl_pointer")
	}

	if !conn.registerCall {
		t.Error("RegisterObject was not called")
	}

	if len(conn.requests) != 1 {
		t.Fatalf("len(requests) = %d, want 1", len(conn.requests))
	}
	req := conn.requests[0]
	if req.objectID != 42 {
		t.Errorf("request.objectID = %d, want 42", req.objectID)
	}
	if req.opcode != seatOpcodeGetPointer {
		t.Errorf("request.opcode = %d, want %d", req.opcode, seatOpcodeGetPointer)
	}
}

func TestSeatGetKeyboard(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)

	keyboard, err := seat.GetKeyboard()
	if err != nil {
		t.Fatalf("GetKeyboard() error = %v", err)
	}

	if keyboard == nil {
		t.Fatal("GetKeyboard() returned nil")
	}
	if keyboard.ID() != 2 {
		t.Errorf("keyboard.ID() = %d, want 2", keyboard.ID())
	}
	if keyboard.Interface() != "wl_keyboard" {
		t.Errorf("keyboard.Interface() = %q, want %q", keyboard.Interface(), "wl_keyboard")
	}

	if len(conn.requests) != 1 {
		t.Fatalf("len(requests) = %d, want 1", len(conn.requests))
	}
	req := conn.requests[0]
	if req.opcode != seatOpcodeGetKeyboard {
		t.Errorf("request.opcode = %d, want %d", req.opcode, seatOpcodeGetKeyboard)
	}
}

func TestSeatGetTouch(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)

	touch, err := seat.GetTouch()
	if err != nil {
		t.Fatalf("GetTouch() error = %v", err)
	}

	if touch == nil {
		t.Fatal("GetTouch() returned nil")
	}
	if touch.ID() != 2 {
		t.Errorf("touch.ID() = %d, want 2", touch.ID())
	}
	if touch.Interface() != "wl_touch" {
		t.Errorf("touch.Interface() = %q, want %q", touch.Interface(), "wl_touch")
	}

	if len(conn.requests) != 1 {
		t.Fatalf("len(requests) = %d, want 1", len(conn.requests))
	}
	req := conn.requests[0]
	if req.opcode != seatOpcodeGetTouch {
		t.Errorf("request.opcode = %d, want %d", req.opcode, seatOpcodeGetTouch)
	}
}

func TestSeatRelease(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)

	if err := seat.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	if len(conn.requests) != 1 {
		t.Fatalf("len(requests) = %d, want 1", len(conn.requests))
	}
	req := conn.requests[0]
	if req.opcode != seatOpcodeRelease {
		t.Errorf("request.opcode = %d, want %d", req.opcode, seatOpcodeRelease)
	}
}

func TestSeatHandleCapabilities(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)

	tests := []struct {
		name string
		caps uint32
		want SeatCapability
	}{
		{"pointer only", 1, SeatCapabilityPointer},
		{"keyboard only", 2, SeatCapabilityKeyboard},
		{"touch only", 4, SeatCapabilityTouch},
		{"all capabilities", 7, SeatCapabilityPointer | SeatCapabilityKeyboard | SeatCapabilityTouch},
		{"pointer and keyboard", 3, SeatCapabilityPointer | SeatCapabilityKeyboard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seat.HandleCapabilities(tt.caps)
			if seat.Capabilities() != tt.want {
				t.Errorf("Capabilities() = %d, want %d", seat.Capabilities(), tt.want)
			}
		})
	}
}

func TestSeatHandleName(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)

	seat.HandleName("seat0")
	if seat.Name() != "seat0" {
		t.Errorf("Name() = %q, want %q", seat.Name(), "seat0")
	}
}

func TestPointerSetCursor(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)
	pointer, _ := seat.GetPointer()

	conn.requests = nil

	err := pointer.SetCursor(123, 456, 10, 20)
	if err != nil {
		t.Fatalf("SetCursor() error = %v", err)
	}

	if len(conn.requests) != 1 {
		t.Fatalf("len(requests) = %d, want 1", len(conn.requests))
	}
	req := conn.requests[0]
	if req.opcode != pointerOpcodeSetCursor {
		t.Errorf("request.opcode = %d, want %d", req.opcode, pointerOpcodeSetCursor)
	}
	if len(req.args) != 4 {
		t.Fatalf("len(args) = %d, want 4", len(req.args))
	}
}

func TestPointerRelease(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)
	pointer, _ := seat.GetPointer()

	conn.requests = nil

	if err := pointer.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	if len(conn.requests) != 1 {
		t.Fatalf("len(requests) = %d, want 1", len(conn.requests))
	}
	req := conn.requests[0]
	if req.opcode != pointerOpcodeRelease {
		t.Errorf("request.opcode = %d, want %d", req.opcode, pointerOpcodeRelease)
	}
}

func TestKeyboardRelease(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)
	keyboard, _ := seat.GetKeyboard()

	conn.requests = nil

	if err := keyboard.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	if len(conn.requests) != 1 {
		t.Fatalf("len(requests) = %d, want 1", len(conn.requests))
	}
	req := conn.requests[0]
	if req.opcode != keyboardOpcodeRelease {
		t.Errorf("request.opcode = %d, want %d", req.opcode, keyboardOpcodeRelease)
	}
}

func TestKeyboardModifiers(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)
	keyboard, _ := seat.GetKeyboard()

	tests := []struct {
		name      string
		depressed uint32
		latched   uint32
		locked    uint32
		wantShift bool
		wantCtrl  bool
		wantAlt   bool
	}{
		{"no modifiers", 0x00, 0x00, 0x00, false, false, false},
		{"shift only", 0x01, 0x00, 0x00, true, false, false},
		{"ctrl only", 0x04, 0x00, 0x00, false, true, false},
		{"shift and ctrl", 0x05, 0x00, 0x00, true, true, false},
		{"all modifiers", 0xFF, 0x00, 0x00, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyboard.HandleModifiers(1, tt.depressed, tt.latched, tt.locked, 0)
			mods := keyboard.Modifiers()
			if mods.Shift != tt.wantShift {
				t.Errorf("Shift = %v, want %v", mods.Shift, tt.wantShift)
			}
			if mods.Ctrl != tt.wantCtrl {
				t.Errorf("Ctrl = %v, want %v", mods.Ctrl, tt.wantCtrl)
			}
			if mods.Alt != tt.wantAlt {
				t.Errorf("Alt = %v, want %v", mods.Alt, tt.wantAlt)
			}
		})
	}
}

func TestTouchRelease(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)
	touch, _ := seat.GetTouch()

	conn.requests = nil

	if err := touch.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	if len(conn.requests) != 1 {
		t.Fatalf("len(requests) = %d, want 1", len(conn.requests))
	}
	req := conn.requests[0]
	if req.opcode != touchOpcodeRelease {
		t.Errorf("request.opcode = %d, want %d", req.opcode, touchOpcodeRelease)
	}
}

func TestKeymapKeycodeToKeysym(t *testing.T) {
	km := &Keymap{}

	tests := []struct {
		name    string
		keycode uint32
		mods    ModifierState
		want    Keysym
	}{
		{"escape", 1, ModifierState{}, KeysymEscape},
		{"backspace", 14, ModifierState{}, KeysymBackSpace},
		{"tab", 15, ModifierState{}, KeysymTab},
		{"return", 28, ModifierState{}, KeysymReturn},
		{"a unshifted", 30, ModifierState{}, Keysym('a')},
		{"a shifted", 30, ModifierState{Shift: true}, Keysym('A')},
		{"z unshifted", 44, ModifierState{}, Keysym('z')},
		{"z shifted", 44, ModifierState{Shift: true}, Keysym('Z')},
		{"space", 57, ModifierState{}, Keysym(' ')},
		{"1 unshifted", 2, ModifierState{}, Keysym('1')},
		{"1 shifted", 2, ModifierState{Shift: true}, Keysym('!')},
		{"0 unshifted", 11, ModifierState{}, Keysym('0')},
		{"0 shifted", 11, ModifierState{Shift: true}, Keysym(')')},
		{"up arrow", 103, ModifierState{}, KeysymUp},
		{"down arrow", 108, ModifierState{}, KeysymDown},
		{"left arrow", 105, ModifierState{}, KeysymLeft},
		{"right arrow", 106, ModifierState{}, KeysymRight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := km.KeycodeToKeysym(tt.keycode, tt.mods)
			if got != tt.want {
				t.Errorf("KeycodeToKeysym(%d, %+v) = 0x%X, want 0x%X", tt.keycode, tt.mods, got, tt.want)
			}
		})
	}
}

func TestPointerEventHandlers(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)
	pointer, _ := seat.GetPointer()

	pointer.HandleEnter(1, 100, 256, 512)
	pointer.HandleLeave(2, 100)
	pointer.HandleMotion(3, 128, 256)
	pointer.HandleButton(4, 5, 272, uint32(ButtonStatePressed))
	pointer.HandleAxis(6, uint32(AxisVerticalScroll), -120)
	pointer.HandleFrame()
	pointer.HandleAxisSource(1)
	pointer.HandleAxisStop(7, uint32(AxisVerticalScroll))
	pointer.HandleAxisDiscrete(uint32(AxisVerticalScroll), -1)
}

func TestKeyboardEventHandlers(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)
	keyboard, _ := seat.GetKeyboard()

	keyboard.HandleKeymap(1, 5, 1024)
	keyboard.HandleEnter(1, 100, []uint32{30, 31, 32})
	keyboard.HandleLeave(2, 100)
	keyboard.HandleKey(3, 4, 30, uint32(KeyStatePressed))
	keyboard.HandleModifiers(5, 0x01, 0, 0, 0)
	keyboard.HandleRepeatInfo(25, 600)
}

func TestTouchEventHandlers(t *testing.T) {
	conn := &mockConn{nextID: 1}
	seat := NewSeat(conn, 42, 7)
	touch, _ := seat.GetTouch()

	touch.HandleDown(1, 100, 200, 0, 512, 768)
	touch.HandleMotion(101, 0, 520, 770)
	touch.HandleUp(2, 102, 0)
	touch.HandleFrame()
	touch.HandleCancel()
	touch.HandleShape(0, 10, 8)
	touch.HandleOrientation(0, 45)
}
