package events

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/opd-ai/wain/internal/x11/wire"
)

// Helper function to create event data buffer
func makeEventData(values ...interface{}) []byte {
	var buf bytes.Buffer
	for _, v := range values {
		switch val := v.(type) {
		case uint32:
			binary.Write(&buf, binary.LittleEndian, val)
		case int32:
			binary.Write(&buf, binary.LittleEndian, val)
		case uint16:
			binary.Write(&buf, binary.LittleEndian, val)
		case int16:
			binary.Write(&buf, binary.LittleEndian, val)
		case uint8:
			buf.WriteByte(val)
		case bool:
			if val {
				buf.WriteByte(1)
			} else {
				buf.WriteByte(0)
			}
		}
	}
	// Pad to 28 bytes
	for buf.Len() < 28 {
		buf.WriteByte(0)
	}
	return buf.Bytes()
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventTypeKeyPress, "KeyPress"},
		{EventTypeKeyRelease, "KeyRelease"},
		{EventTypeButtonPress, "ButtonPress"},
		{EventTypeButtonRelease, "ButtonRelease"},
		{EventTypeMotionNotify, "MotionNotify"},
		{EventTypeExpose, "Expose"},
		{EventTypeConfigureNotify, "ConfigureNotify"},
		{EventType(99), "Event(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.eventType.String()
			if result != tt.expected {
				t.Errorf("EventType.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseKeyPressEvent(t *testing.T) {
	header := wire.EventHeader{
		Type:     uint8(EventTypeKeyPress),
		Detail:   65, // 'a' keycode
		Sequence: 100,
	}

	data := makeEventData(
		uint32(12345),  // Time
		uint32(1),      // Root
		uint32(2),      // Event
		uint32(3),      // Child
		int16(100),     // RootX
		int16(200),     // RootY
		int16(50),      // EventX
		int16(75),      // EventY
		uint16(1),      // State (Shift)
		true,           // SameScreen
	)

	evt, err := ParseKeyPressEvent(header, data)
	if err != nil {
		t.Fatalf("ParseKeyPressEvent failed: %v", err)
	}

	if evt.Type != EventTypeKeyPress {
		t.Errorf("Type = %v, want %v", evt.Type, EventTypeKeyPress)
	}
	if evt.Detail != 65 {
		t.Errorf("Detail = %d, want 65", evt.Detail)
	}
	if evt.Sequence != 100 {
		t.Errorf("Sequence = %d, want 100", evt.Sequence)
	}
	if evt.Time != 12345 {
		t.Errorf("Time = %d, want 12345", evt.Time)
	}
	if evt.Root != 1 {
		t.Errorf("Root = %d, want 1", evt.Root)
	}
	if evt.Event != 2 {
		t.Errorf("Event = %d, want 2", evt.Event)
	}
	if evt.Child != 3 {
		t.Errorf("Child = %d, want 3", evt.Child)
	}
	if evt.RootX != 100 {
		t.Errorf("RootX = %d, want 100", evt.RootX)
	}
	if evt.RootY != 200 {
		t.Errorf("RootY = %d, want 200", evt.RootY)
	}
	if evt.EventX != 50 {
		t.Errorf("EventX = %d, want 50", evt.EventX)
	}
	if evt.EventY != 75 {
		t.Errorf("EventY = %d, want 75", evt.EventY)
	}
	if evt.State != 1 {
		t.Errorf("State = %d, want 1", evt.State)
	}
	if !evt.SameScreen {
		t.Error("SameScreen = false, want true")
	}
}

func TestParseKeyPressEventShortData(t *testing.T) {
	header := wire.EventHeader{
		Type:     uint8(EventTypeKeyPress),
		Detail:   65,
		Sequence: 100,
	}

	data := []byte{1, 2, 3} // Too short

	_, err := ParseKeyPressEvent(header, data)
	if err == nil {
		t.Error("ParseKeyPressEvent should fail with short data")
	}
}

func TestParseKeyReleaseEvent(t *testing.T) {
	header := wire.EventHeader{
		Type:     uint8(EventTypeKeyRelease),
		Detail:   66,
		Sequence: 101,
	}

	data := makeEventData(
		uint32(12346),  // Time
		uint32(1),      // Root
		uint32(2),      // Event
		uint32(0),      // Child (none)
		int16(101),     // RootX
		int16(201),     // RootY
		int16(51),      // EventX
		int16(76),      // EventY
		uint16(0),      // State (no modifiers)
		true,           // SameScreen
	)

	evt, err := ParseKeyReleaseEvent(header, data)
	if err != nil {
		t.Fatalf("ParseKeyReleaseEvent failed: %v", err)
	}

	if EventType(evt.Type) != EventTypeKeyRelease {
		t.Errorf("Type = %v, want %v", evt.Type, EventTypeKeyRelease)
	}
	if evt.Detail != 66 {
		t.Errorf("Detail = %d, want 66", evt.Detail)
	}
}

func TestParseButtonPressEvent(t *testing.T) {
	header := wire.EventHeader{
		Type:     uint8(EventTypeButtonPress),
		Detail:   1, // Left button
		Sequence: 102,
	}

	data := makeEventData(
		uint32(12347),  // Time
		uint32(1),      // Root
		uint32(2),      // Event
		uint32(4),      // Child
		int16(150),     // RootX
		int16(250),     // RootY
		int16(60),      // EventX
		int16(80),      // EventY
		uint16(4),      // State (Control)
		true,           // SameScreen
	)

	evt, err := ParseButtonPressEvent(header, data)
	if err != nil {
		t.Fatalf("ParseButtonPressEvent failed: %v", err)
	}

	if evt.Type != EventTypeButtonPress {
		t.Errorf("Type = %v, want %v", evt.Type, EventTypeButtonPress)
	}
	if evt.Detail != 1 {
		t.Errorf("Detail = %d, want 1", evt.Detail)
	}
	if evt.Time != 12347 {
		t.Errorf("Time = %d, want 12347", evt.Time)
	}
	if evt.RootX != 150 || evt.RootY != 250 {
		t.Errorf("Root coordinates = (%d, %d), want (150, 250)", evt.RootX, evt.RootY)
	}
	if evt.EventX != 60 || evt.EventY != 80 {
		t.Errorf("Event coordinates = (%d, %d), want (60, 80)", evt.EventX, evt.EventY)
	}
}

func TestParseButtonReleaseEvent(t *testing.T) {
	header := wire.EventHeader{
		Type:     uint8(EventTypeButtonRelease),
		Detail:   3, // Right button
		Sequence: 103,
	}

	data := makeEventData(
		uint32(12348),  // Time
		uint32(1),      // Root
		uint32(2),      // Event
		uint32(0),      // Child
		int16(160),     // RootX
		int16(260),     // RootY
		int16(65),      // EventX
		int16(85),      // EventY
		uint16(0x104),  // State (Control + Button1)
		true,           // SameScreen
	)

	evt, err := ParseButtonReleaseEvent(header, data)
	if err != nil {
		t.Fatalf("ParseButtonReleaseEvent failed: %v", err)
	}

	if EventType(evt.Type) != EventTypeButtonRelease {
		t.Errorf("Type = %v, want %v", evt.Type, EventTypeButtonRelease)
	}
	if evt.Detail != 3 {
		t.Errorf("Detail = %d, want 3", evt.Detail)
	}
}

func TestParseMotionNotifyEvent(t *testing.T) {
	header := wire.EventHeader{
		Type:     uint8(EventTypeMotionNotify),
		Detail:   0, // Normal mode
		Sequence: 104,
	}

	data := makeEventData(
		uint32(12349),  // Time
		uint32(1),      // Root
		uint32(2),      // Event
		uint32(5),      // Child
		int16(170),     // RootX
		int16(270),     // RootY
		int16(70),      // EventX
		int16(90),      // EventY
		uint16(0x101),  // State (Shift + Button1)
		true,           // SameScreen
	)

	evt, err := ParseMotionNotifyEvent(header, data)
	if err != nil {
		t.Fatalf("ParseMotionNotifyEvent failed: %v", err)
	}

	if evt.Type != EventTypeMotionNotify {
		t.Errorf("Type = %v, want %v", evt.Type, EventTypeMotionNotify)
	}
	if evt.Detail != 0 {
		t.Errorf("Detail = %d, want 0", evt.Detail)
	}
	if evt.RootX != 170 || evt.RootY != 270 {
		t.Errorf("Root coordinates = (%d, %d), want (170, 270)", evt.RootX, evt.RootY)
	}
	if evt.EventX != 70 || evt.EventY != 90 {
		t.Errorf("Event coordinates = (%d, %d), want (70, 90)", evt.EventX, evt.EventY)
	}
}

func TestParseExposeEvent(t *testing.T) {
	header := wire.EventHeader{
		Type:     uint8(EventTypeExpose),
		Detail:   0,
		Sequence: 105,
	}

	data := makeEventData(
		uint32(100),  // Window
		uint16(10),   // X
		uint16(20),   // Y
		uint16(300),  // Width
		uint16(200),  // Height
		uint16(2),    // Count
	)

	evt, err := ParseExposeEvent(header, data)
	if err != nil {
		t.Fatalf("ParseExposeEvent failed: %v", err)
	}

	if evt.Type != EventTypeExpose {
		t.Errorf("Type = %v, want %v", evt.Type, EventTypeExpose)
	}
	if evt.Sequence != 105 {
		t.Errorf("Sequence = %d, want 105", evt.Sequence)
	}
	if evt.Window != 100 {
		t.Errorf("Window = %d, want 100", evt.Window)
	}
	if evt.X != 10 || evt.Y != 20 {
		t.Errorf("Position = (%d, %d), want (10, 20)", evt.X, evt.Y)
	}
	if evt.Width != 300 || evt.Height != 200 {
		t.Errorf("Size = (%d, %d), want (300, 200)", evt.Width, evt.Height)
	}
	if evt.Count != 2 {
		t.Errorf("Count = %d, want 2", evt.Count)
	}
}

func TestParseConfigureNotifyEvent(t *testing.T) {
	header := wire.EventHeader{
		Type:     uint8(EventTypeConfigureNotify),
		Detail:   0,
		Sequence: 106,
	}

	data := makeEventData(
		uint32(10),   // Event
		uint32(11),   // Window
		uint32(12),   // AboveSibling
		int16(100),   // X
		int16(150),   // Y
		uint16(800),  // Width
		uint16(600),  // Height
		uint16(2),    // BorderWidth
		true,         // OverrideRedirect
	)

	evt, err := ParseConfigureNotifyEvent(header, data)
	if err != nil {
		t.Fatalf("ParseConfigureNotifyEvent failed: %v", err)
	}

	if evt.Type != EventTypeConfigureNotify {
		t.Errorf("Type = %v, want %v", evt.Type, EventTypeConfigureNotify)
	}
	if evt.Sequence != 106 {
		t.Errorf("Sequence = %d, want 106", evt.Sequence)
	}
	if evt.Event != 10 {
		t.Errorf("Event = %d, want 10", evt.Event)
	}
	if evt.Window != 11 {
		t.Errorf("Window = %d, want 11", evt.Window)
	}
	if evt.AboveSibling != 12 {
		t.Errorf("AboveSibling = %d, want 12", evt.AboveSibling)
	}
	if evt.X != 100 || evt.Y != 150 {
		t.Errorf("Position = (%d, %d), want (100, 150)", evt.X, evt.Y)
	}
	if evt.Width != 800 || evt.Height != 600 {
		t.Errorf("Size = (%d, %d), want (800, 600)", evt.Width, evt.Height)
	}
	if evt.BorderWidth != 2 {
		t.Errorf("BorderWidth = %d, want 2", evt.BorderWidth)
	}
	if !evt.OverrideRedirect {
		t.Error("OverrideRedirect = false, want true")
	}
}

func TestHasModifier(t *testing.T) {
	tests := []struct {
		name     string
		state    uint16
		modifier ModifierMask
		expected bool
	}{
		{"Shift set", 0x01, ModifierShift, true},
		{"Shift not set", 0x00, ModifierShift, false},
		{"Control set", 0x04, ModifierControl, true},
		{"Control not set", 0x01, ModifierControl, false},
		{"Multiple modifiers", 0x05, ModifierShift, true},
		{"Multiple modifiers 2", 0x05, ModifierControl, true},
		{"Button1 set", 0x100, ModifierButton1, true},
		{"Button1 not set", 0x01, ModifierButton1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasModifier(tt.state, tt.modifier)
			if result != tt.expected {
				t.Errorf("HasModifier(%#x, %v) = %v, want %v",
					tt.state, tt.modifier, result, tt.expected)
			}
		})
	}
}

func TestParseEventsWithNegativeCoordinates(t *testing.T) {
	header := wire.EventHeader{
		Type:     uint8(EventTypeMotionNotify),
		Detail:   0,
		Sequence: 200,
	}

	data := makeEventData(
		uint32(1000),   // Time
		uint32(1),      // Root
		uint32(2),      // Event
		uint32(0),      // Child
		int16(-50),     // RootX (negative)
		int16(-100),    // RootY (negative)
		int16(-25),     // EventX (negative)
		int16(-75),     // EventY (negative)
		uint16(0),      // State
		true,           // SameScreen
	)

	evt, err := ParseMotionNotifyEvent(header, data)
	if err != nil {
		t.Fatalf("ParseMotionNotifyEvent failed: %v", err)
	}

	if evt.RootX != -50 || evt.RootY != -100 {
		t.Errorf("Root coordinates = (%d, %d), want (-50, -100)", evt.RootX, evt.RootY)
	}
	if evt.EventX != -25 || evt.EventY != -75 {
		t.Errorf("Event coordinates = (%d, %d), want (-25, -75)", evt.EventX, evt.EventY)
	}
}

func TestParseExposeEventZeroCount(t *testing.T) {
	header := wire.EventHeader{
		Type:     uint8(EventTypeExpose),
		Sequence: 300,
	}

	data := makeEventData(
		uint32(100),  // Window
		uint16(0),    // X
		uint16(0),    // Y
		uint16(800),  // Width
		uint16(600),  // Height
		uint16(0),    // Count (no more expose events)
	)

	evt, err := ParseExposeEvent(header, data)
	if err != nil {
		t.Fatalf("ParseExposeEvent failed: %v", err)
	}

	if evt.Count != 0 {
		t.Errorf("Count = %d, want 0 (last expose event)", evt.Count)
	}
}

func TestModifierMaskConstants(t *testing.T) {
	// Verify modifier mask values are correct
	if ModifierShift != 1<<0 {
		t.Errorf("ModifierShift = %d, want %d", ModifierShift, 1<<0)
	}
	if ModifierControl != 1<<2 {
		t.Errorf("ModifierControl = %d, want %d", ModifierControl, 1<<2)
	}
	if ModifierButton1 != 1<<8 {
		t.Errorf("ModifierButton1 = %d, want %d", ModifierButton1, 1<<8)
	}
	if ModifierButton3 != 1<<10 {
		t.Errorf("ModifierButton3 = %d, want %d", ModifierButton3, 1<<10)
	}
}

func TestButtonNumbers(t *testing.T) {
	// Test common button codes
	tests := []struct {
		detail   uint8
		name     string
	}{
		{1, "Left button"},
		{2, "Middle button"},
		{3, "Right button"},
		{4, "Scroll up"},
		{5, "Scroll down"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := wire.EventHeader{
				Type:   uint8(EventTypeButtonPress),
				Detail: tt.detail,
			}

			data := makeEventData(
				uint32(1000), uint32(1), uint32(2), uint32(0),
				int16(0), int16(0), int16(0), int16(0),
				uint16(0), true,
			)

			evt, err := ParseButtonPressEvent(header, data)
			if err != nil {
				t.Fatalf("ParseButtonPressEvent failed for %s: %v", tt.name, err)
			}

			if evt.Detail != tt.detail {
				t.Errorf("%s: Detail = %d, want %d", tt.name, evt.Detail, tt.detail)
			}
		})
	}
}
