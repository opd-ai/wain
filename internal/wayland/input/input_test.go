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

// ---------------------------------------------------------------------------
// Helper: build wire.Argument slices for event testing
// ---------------------------------------------------------------------------

func uint32Arg(v uint32) wire.Argument   { return wire.Argument{Type: wire.ArgTypeUint, Value: v} }
func int32Arg(v int32) wire.Argument     { return wire.Argument{Type: wire.ArgTypeInt, Value: v} }
func intArg(v int) wire.Argument         { return wire.Argument{Type: wire.ArgTypeFD, Value: v} }
func bytesArg(v []byte) wire.Argument    { return wire.Argument{Type: wire.ArgTypeArray, Value: v} }

// ---------------------------------------------------------------------------
// Keyboard HandleEvent routing
// ---------------------------------------------------------------------------

func newTestKeyboard(conn *mockConn) *Keyboard {
seat := NewSeat(conn, 42, 7)
kb, _ := seat.GetKeyboard()
return kb
}

func TestKeyboard_HandleEvent_Enter(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
kb := newTestKeyboard(conn)

var gotSurface uint32
kb.SetEnterCallback(func(surfaceID uint32) { gotSurface = surfaceID })

args := []wire.Argument{
uint32Arg(1),     // serial
uint32Arg(99),    // surfaceID
bytesArg(nil),    // keys array (empty)
}
if err := kb.HandleEvent(keyboardEventEnter, args); err != nil {
t.Fatalf("HandleEvent enter: %v", err)
}
if gotSurface != 99 {
t.Errorf("enter callback surfaceID = %d, want 99", gotSurface)
}
}

func TestKeyboard_HandleEvent_Leave(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
kb := newTestKeyboard(conn)

var gotSurface uint32
kb.SetLeaveCallback(func(surfaceID uint32) { gotSurface = surfaceID })

args := []wire.Argument{
uint32Arg(2), // serial
uint32Arg(77), // surfaceID
}
if err := kb.HandleEvent(keyboardEventLeave, args); err != nil {
t.Fatalf("HandleEvent leave: %v", err)
}
if gotSurface != 77 {
t.Errorf("leave callback surfaceID = %d, want 77", gotSurface)
}
}

func TestKeyboard_HandleEvent_Key(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
kb := newTestKeyboard(conn)

var gotKey, gotState uint32
kb.SetKeyCallback(func(surfaceID, key, state uint32) {
gotKey = key
gotState = state
})
kb.focusedSurface = 10

args := []wire.Argument{
uint32Arg(3),                       // serial
uint32Arg(100),                     // time
uint32Arg(30),                      // key code
uint32Arg(uint32(KeyStatePressed)), // state
}
if err := kb.HandleEvent(keyboardEventKey, args); err != nil {
t.Fatalf("HandleEvent key: %v", err)
}
if gotKey != 30 {
t.Errorf("key = %d, want 30", gotKey)
}
if gotState != uint32(KeyStatePressed) {
t.Errorf("state = %d, want pressed", gotState)
}
}

func TestKeyboard_HandleEvent_Modifiers(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
kb := newTestKeyboard(conn)

var gotDepressed uint32
kb.SetModifiersCallback(func(dep, lat, lock uint32) { gotDepressed = dep })

args := []wire.Argument{
uint32Arg(4),    // serial
uint32Arg(0x01), // depressed (Shift)
uint32Arg(0),    // latched
uint32Arg(0),    // locked
uint32Arg(0),    // group
}
if err := kb.HandleEvent(keyboardEventModifiers, args); err != nil {
t.Fatalf("HandleEvent modifiers: %v", err)
}
if gotDepressed != 0x01 {
t.Errorf("depressed = %d, want 1", gotDepressed)
}
mods := kb.Modifiers()
if !mods.Shift {
t.Error("expected Shift modifier set")
}
}

func TestKeyboard_HandleEvent_RepeatInfo(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
kb := newTestKeyboard(conn)

args := []wire.Argument{
int32Arg(25),  // rate
int32Arg(600), // delay
}
if err := kb.HandleEvent(keyboardEventRepeatInfo, args); err != nil {
t.Fatalf("HandleEvent repeat_info: %v", err)
}
}

func TestKeyboard_HandleEvent_Keymap(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
kb := newTestKeyboard(conn)

// Format 0 (unknown) — should not crash.
args := []wire.Argument{
uint32Arg(0), // format (unknown, no keymap loaded)
intArg(-1),   // fd (invalid, but not used for format 0)
uint32Arg(0), // size
}
if err := kb.HandleEvent(keyboardEventKeymap, args); err != nil {
t.Fatalf("HandleEvent keymap: %v", err)
}
}

func TestKeyboard_HandleEvent_UnknownOpcode(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
kb := newTestKeyboard(conn)

err := kb.HandleEvent(99, nil)
if err == nil {
t.Error("expected error for unknown opcode")
}
}

func TestKeyboard_HandleEvent_TooFewArgs(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
kb := newTestKeyboard(conn)

// enter event requires 3 args; provide 1.
err := kb.HandleEvent(keyboardEventEnter, []wire.Argument{uint32Arg(1)})
if err == nil {
t.Error("expected error for too few args")
}
}

// ---------------------------------------------------------------------------
// Pointer HandleEvent routing
// ---------------------------------------------------------------------------

func newTestPointer(conn *mockConn) *Pointer {
seat := NewSeat(conn, 42, 7)
p, _ := seat.GetPointer()
return p
}

func TestPointer_HandleEvent_Enter(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
p := newTestPointer(conn)

var gotSurface uint32
p.SetEnterCallback(func(surfaceID uint32, x, y float64) { gotSurface = surfaceID })

args := []wire.Argument{
uint32Arg(1),    // serial
uint32Arg(55),   // surfaceID
int32Arg(512),   // surfaceX (fixed-point: 512/256 = 2.0)
int32Arg(1024),  // surfaceY (fixed-point: 1024/256 = 4.0)
}
if err := p.HandleEvent(pointerEventEnter, args); err != nil {
t.Fatalf("HandleEvent enter: %v", err)
}
if gotSurface != 55 {
t.Errorf("enter surfaceID = %d, want 55", gotSurface)
}
}

func TestPointer_HandleEvent_Leave(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
p := newTestPointer(conn)

var gotSurface uint32
p.SetLeaveCallback(func(surfaceID uint32) { gotSurface = surfaceID })
p.focusedSurface = 55

args := []wire.Argument{
uint32Arg(2), // serial
uint32Arg(55), // surfaceID
}
if err := p.HandleEvent(pointerEventLeave, args); err != nil {
t.Fatalf("HandleEvent leave: %v", err)
}
if gotSurface != 55 {
t.Errorf("leave surfaceID = %d, want 55", gotSurface)
}
}

func TestPointer_HandleEvent_Motion(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
p := newTestPointer(conn)

p.focusedSurface = 55
var gotX, gotY float64
p.SetMotionCallback(func(surfaceID uint32, x, y float64) { gotX = x; gotY = y })

args := []wire.Argument{
uint32Arg(200),  // time
int32Arg(512),   // surfaceX
int32Arg(256),   // surfaceY
}
if err := p.HandleEvent(pointerEventMotion, args); err != nil {
t.Fatalf("HandleEvent motion: %v", err)
}
if gotX != 2.0 {
t.Errorf("motion x = %f, want 2.0", gotX)
}
if gotY != 1.0 {
t.Errorf("motion y = %f, want 1.0", gotY)
}
}

func TestPointer_HandleEvent_Button(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
p := newTestPointer(conn)

p.focusedSurface = 55
var gotButton, gotState uint32
p.SetButtonCallback(func(surfaceID, button, state uint32, x, y float64) {
gotButton = button
gotState = state
})

args := []wire.Argument{
uint32Arg(3),   // serial
uint32Arg(200), // time
uint32Arg(272), // button (BTN_LEFT)
uint32Arg(1),   // state (pressed)
}
if err := p.HandleEvent(pointerEventButton, args); err != nil {
t.Fatalf("HandleEvent button: %v", err)
}
if gotButton != 272 {
t.Errorf("button = %d, want 272", gotButton)
}
if gotState != 1 {
t.Errorf("state = %d, want 1", gotState)
}
}

func TestPointer_HandleEvent_Axis(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
p := newTestPointer(conn)

p.focusedSurface = 55
var gotAxis uint32
p.SetAxisCallback(func(surfaceID, axis uint32, value, x, y float64) { gotAxis = axis })

args := []wire.Argument{
uint32Arg(200), // time
uint32Arg(0),   // axis (vertical)
int32Arg(256),  // value (fixed-point)
}
if err := p.HandleEvent(pointerEventAxis, args); err != nil {
t.Fatalf("HandleEvent axis: %v", err)
}
if gotAxis != 0 {
t.Errorf("axis = %d, want 0", gotAxis)
}
}

func TestPointer_HandleEvent_PassThroughOpcodes(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
p := newTestPointer(conn)

// Frame, AxisSource, AxisStop, AxisDiscrete — must not error.
for _, op := range []uint16{pointerEventFrame, pointerEventAxisSource, pointerEventAxisStop, pointerEventAxisDiscrete} {
if err := p.HandleEvent(op, nil); err != nil {
t.Errorf("HandleEvent opcode %d: %v", op, err)
}
}
}

func TestPointer_HandleEvent_UnknownOpcode(t *testing.T) {
t.Parallel()
conn := &mockConn{nextID: 1}
p := newTestPointer(conn)

if err := p.HandleEvent(99, nil); err == nil {
t.Error("expected error for unknown pointer opcode")
}
}

// ---------------------------------------------------------------------------
// parseEvent error path
// ---------------------------------------------------------------------------

func TestParseEvent_TooFewArgs(t *testing.T) {
t.Parallel()
err := parseEvent([]wire.Argument{uint32Arg(1)}, 3, "test", func(_ *wire.ArgDecoder) {})
if err == nil {
t.Error("expected error for too few args")
}
}

func TestParseEvent_EnoughArgs(t *testing.T) {
t.Parallel()
called := false
err := parseEvent(
[]wire.Argument{uint32Arg(1), uint32Arg(2), uint32Arg(3)},
3, "test",
func(d *wire.ArgDecoder) {
called = true
_ = d.Uint32("a")
_ = d.Uint32("b")
_ = d.Uint32("c")
},
)
if err != nil {
t.Errorf("unexpected error: %v", err)
}
if !called {
t.Error("decode func not called")
}
}

// ---------------------------------------------------------------------------
// Keymap helper
// ---------------------------------------------------------------------------

func TestKeymap_MapQwertyRow(t *testing.T) {
t.Parallel()
km := &Keymap{}

// mapQwertyRow tests: Q key on QWERTY layout is keycode 16.
ks := km.mapQwertyRow(16)
if ks == 0 {
t.Error("mapQwertyRow(16) returned 0 (unmapped)")
}
}

func TestKeymap_Close(t *testing.T) {
t.Parallel()
// Close on a nil Keymap.Fd should be safe.
km := &Keymap{}
km.Close() // must not panic
}
