package wain

import (
	"testing"
	"time"
)

// mockEvent is a simple event implementation for testing.
type mockEvent struct {
	baseEvent
}

func (m *mockEvent) Type() EventType { return EventTypeCustom }

// TestPublicWidgetInterface verifies that BasePublicWidget implements PublicWidget.
func TestPublicWidgetInterface(t *testing.T) {
	var _ PublicWidget = &BasePublicWidget{}
}

// TestContainerInterface verifies that BasePublicWidget implements Container.
func TestContainerInterface(t *testing.T) {
	var _ Container = &BasePublicWidget{}
}

// TestBasePublicWidget_Bounds verifies the Bounds method.
func TestBasePublicWidget_Bounds(t *testing.T) {
	w := NewBasePublicWidget(100, 200)
	width, height := w.Bounds()
	if width != 100 || height != 200 {
		t.Errorf("Bounds() = (%d, %d), want (100, 200)", width, height)
	}
}

// TestBasePublicWidget_Position verifies the Position method.
func TestBasePublicWidget_Position(t *testing.T) {
	w := NewBasePublicWidget(100, 200)
	w.SetBounds(10, 20, 100, 200)
	x, y := w.Position()
	if x != 10 || y != 20 {
		t.Errorf("Position() = (%d, %d), want (10, 20)", x, y)
	}
}

// TestBasePublicWidget_SetBounds verifies the SetBounds method.
func TestBasePublicWidget_SetBounds(t *testing.T) {
	w := NewBasePublicWidget(0, 0)
	w.SetBounds(50, 60, 150, 250)
	x, y := w.Position()
	width, height := w.Bounds()
	if x != 50 || y != 60 {
		t.Errorf("Position() = (%d, %d), want (50, 60)", x, y)
	}
	if width != 150 || height != 250 {
		t.Errorf("Bounds() = (%d, %d), want (150, 250)", width, height)
	}
}

// TestBasePublicWidget_Visibility verifies visibility control.
func TestBasePublicWidget_Visibility(t *testing.T) {
	w := NewBasePublicWidget(100, 200)
	if !w.IsVisible() {
		t.Error("NewBasePublicWidget should be visible by default")
	}

	w.SetVisible(false)
	if w.IsVisible() {
		t.Error("SetVisible(false) should make widget invisible")
	}

	w.SetVisible(true)
	if !w.IsVisible() {
		t.Error("SetVisible(true) should make widget visible")
	}
}

// TestBasePublicWidget_Children verifies child widget management.
func TestBasePublicWidget_Children(t *testing.T) {
	parent := NewBasePublicWidget(200, 300)
	child1 := NewBasePublicWidget(50, 50)
	child2 := NewBasePublicWidget(50, 50)

	if len(parent.Children()) != 0 {
		t.Error("New widget should have no children")
	}

	parent.Add(&child1)
	if len(parent.Children()) != 1 {
		t.Errorf("Add() resulted in %d children, want 1", len(parent.Children()))
	}

	parent.Add(&child2)
	if len(parent.Children()) != 2 {
		t.Errorf("Add() resulted in %d children, want 2", len(parent.Children()))
	}
}

// TestBasePublicWidget_HandleEvent verifies event handling.
func TestBasePublicWidget_HandleEvent(t *testing.T) {
	w := NewBasePublicWidget(100, 100)

	// Create a simple mock event
	evt := &mockEvent{}
	evt.timestamp = time.Now()

	// Default should return false
	if w.HandleEvent(evt) {
		t.Error("HandleEvent should return false when no handler is set")
	}

	// With handler
	eventHandled := false
	w.OnEvent(func(e Event) bool {
		eventHandled = true
		return true
	})

	if !w.HandleEvent(evt) {
		t.Error("HandleEvent should return true when handler returns true")
	}
	if !eventHandled {
		t.Error("Event handler should have been called")
	}
}

// TestColor verifies color creation and conversion.
func TestColor(t *testing.T) {
	// RGB
	red := RGB(255, 0, 0)
	if red.R != 255 || red.G != 0 || red.B != 0 || red.A != 255 {
		t.Errorf("RGB(255, 0, 0) = %+v, want R=255, G=0, B=0, A=255", red)
	}

	// RGBA
	transparentBlue := RGBA(0, 0, 255, 128)
	if transparentBlue.R != 0 || transparentBlue.G != 0 || transparentBlue.B != 255 || transparentBlue.A != 128 {
		t.Errorf("RGBA(0, 0, 255, 128) = %+v, want R=0, G=0, B=255, A=128", transparentBlue)
	}

	// WithAlpha
	semiRed := red.WithAlpha(128)
	if semiRed.R != 255 || semiRed.A != 128 {
		t.Errorf("WithAlpha(128) = %+v, want R=255, A=128", semiRed)
	}
}

// TestColorConstants verifies predefined color constants.
func TestColorConstants(t *testing.T) {
	if Black.R != 0 || Black.G != 0 || Black.B != 0 || Black.A != 255 {
		t.Errorf("Black = %+v, want R=0, G=0, B=0, A=255", Black)
	}

	if White.R != 255 || White.G != 255 || White.B != 255 || White.A != 255 {
		t.Errorf("White = %+v, want R=255, G=255, B=255, A=255", White)
	}

	if Transparent.A != 0 {
		t.Errorf("Transparent.A = %d, want 0", Transparent.A)
	}
}
