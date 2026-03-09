package decorations

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/ui/widgets"
)

func TestNewWindowButton(t *testing.T) {
	btn := NewWindowButton(ButtonTypeClose, 24)
	if btn == nil {
		t.Fatal("NewWindowButton returned nil")
	}

	w, h := btn.Bounds()
	if w != 24 || h != 24 {
		t.Errorf("expected bounds 24x24, got %dx%d", w, h)
	}

	if btn.buttonType != ButtonTypeClose {
		t.Errorf("expected button type Close, got %d", btn.buttonType)
	}
}

func TestWindowButton_PointerState(t *testing.T) {
	btn := NewWindowButton(ButtonTypeMaximize, 24)

	if btn.state != widgets.PointerStateNormal {
		t.Errorf("expected initial state Normal, got %d", btn.state)
	}

	btn.HandlePointerEnter()
	if btn.state != widgets.PointerStateHover {
		t.Errorf("expected state Hover after enter, got %d", btn.state)
	}

	btn.HandlePointerDown(1)
	if btn.state != widgets.PointerStatePressed {
		t.Errorf("expected state Pressed after down, got %d", btn.state)
	}

	btn.HandlePointerUp(1)
	if btn.state != widgets.PointerStateHover {
		t.Errorf("expected state Hover after up, got %d", btn.state)
	}

	btn.HandlePointerLeave()
	if btn.state != widgets.PointerStateNormal {
		t.Errorf("expected state Normal after leave, got %d", btn.state)
	}
}

func TestWindowButton_Draw(t *testing.T) {
	buf, err := primitives.NewBuffer(100, 100)
	if err != nil {
		t.Fatalf("NewBuffer failed: %v", err)
	}
	btn := NewWindowButton(ButtonTypeClose, 24)

	err = btn.Draw(buf, 10, 10)
	if err != nil {
		t.Fatalf("Draw failed: %v", err)
	}
}

func TestWindowButton_RenderToDisplayList(t *testing.T) {
	dl := displaylist.New()
	btn := NewWindowButton(ButtonTypeMinimize, 24)

	btn.RenderToDisplayList(dl, 10, 10)

	commands := dl.Commands()
	if len(commands) == 0 {
		t.Error("expected display list commands, got none")
	}
}

func TestNewTitleBar(t *testing.T) {
	tb := NewTitleBar("Test Window", 400, 32)
	if tb == nil {
		t.Fatal("NewTitleBar returned nil")
	}

	if tb.title != "Test Window" {
		t.Errorf("expected title 'Test Window', got '%s'", tb.title)
	}

	w, h := tb.Bounds()
	if w != 400 || h != 32 {
		t.Errorf("expected bounds 400x32, got %dx%d", w, h)
	}

	if tb.closeBtn == nil {
		t.Error("close button is nil")
	}
	if tb.maxBtn == nil {
		t.Error("maximize button is nil")
	}
	if tb.minBtn == nil {
		t.Error("minimize button is nil")
	}
}

func TestTitleBar_SetTitle(t *testing.T) {
	tb := NewTitleBar("Old Title", 400, 32)
	tb.SetTitle("New Title")

	if tb.title != "New Title" {
		t.Errorf("expected title 'New Title', got '%s'", tb.title)
	}
}

func TestTitleBar_Resize(t *testing.T) {
	tb := NewTitleBar("Test", 400, 32)
	tb.Resize(600)

	w, _ := tb.Bounds()
	if w != 600 {
		t.Errorf("expected width 600, got %d", w)
	}
}

func TestTitleBar_HitTest(t *testing.T) {
	tb := NewTitleBar("Test", 400, 32)
	theme := DefaultDecorationTheme()
	tb.SetTheme(theme)

	buttonSize := 24
	spacing := theme.ButtonSpacing

	// Test close button (rightmost)
	closeX := 400 - spacing - buttonSize/2
	btn := tb.HitTest(closeX, theme.ButtonSpacing+buttonSize/2)
	if btn != tb.closeBtn {
		t.Error("expected close button hit")
	}

	// Test maximize button
	maxX := closeX - buttonSize - spacing
	btn = tb.HitTest(maxX, theme.ButtonSpacing+buttonSize/2)
	if btn != tb.maxBtn {
		t.Error("expected maximize button hit")
	}

	// Test minimize button
	minX := maxX - buttonSize - spacing
	btn = tb.HitTest(minX, theme.ButtonSpacing+buttonSize/2)
	if btn != tb.minBtn {
		t.Error("expected minimize button hit")
	}

	// Test miss (title area)
	btn = tb.HitTest(50, 16)
	if btn != nil {
		t.Error("expected no button hit in title area")
	}

	// Test miss (out of bounds)
	btn = tb.HitTest(50, -5)
	if btn != nil {
		t.Error("expected no button hit out of bounds")
	}
}

func TestTitleBar_Dragging(t *testing.T) {
	tb := NewTitleBar("Test", 400, 32)

	// Initially not dragging
	dragging, _, _ := tb.HandlePointerMotion(100, 16)
	if dragging {
		t.Error("should not be dragging initially")
	}

	// Start drag
	tb.StartDrag(100, 16)
	if !tb.dragging {
		t.Error("expected dragging flag to be true")
	}

	// Motion while dragging
	dragging, dx, dy := tb.HandlePointerMotion(110, 20)
	if !dragging {
		t.Error("expected dragging to be true")
	}
	if dx != 10 || dy != 4 {
		t.Errorf("expected delta (10, 4), got (%d, %d)", dx, dy)
	}

	// Stop drag
	tb.StopDrag()
	if tb.dragging {
		t.Error("expected dragging flag to be false")
	}

	// Motion after stop
	dragging, _, _ = tb.HandlePointerMotion(120, 24)
	if dragging {
		t.Error("should not be dragging after stop")
	}
}

func TestTitleBar_Draw(t *testing.T) {
	buf, err := primitives.NewBuffer(400, 32)
	if err != nil {
		t.Fatalf("NewBuffer failed: %v", err)
	}
	tb := NewTitleBar("Test Window", 400, 32)

	err = tb.Draw(buf, 0, 0)
	if err != nil {
		t.Fatalf("Draw failed: %v", err)
	}
}

func TestTitleBar_RenderToDisplayList(t *testing.T) {
	dl := displaylist.New()
	tb := NewTitleBar("Test Window", 400, 32)

	tb.RenderToDisplayList(dl, 0, 0)

	commands := dl.Commands()
	if len(commands) == 0 {
		t.Error("expected display list commands, got none")
	}
}

func TestDefaultDecorationTheme(t *testing.T) {
	theme := DefaultDecorationTheme()
	if theme == nil {
		t.Fatal("DefaultDecorationTheme returned nil")
	}

	if theme.TitleBarHeight != 32 {
		t.Errorf("expected title bar height 32, got %d", theme.TitleBarHeight)
	}

	if theme.ButtonSpacing < 0 {
		t.Error("button spacing should be non-negative")
	}

	if theme.TitleFontSize <= 0 {
		t.Error("title font size should be positive")
	}
}

func TestButtonTypes(t *testing.T) {
	types := []ButtonType{ButtonTypeClose, ButtonTypeMaximize, ButtonTypeMinimize}

	for _, bt := range types {
		btn := NewWindowButton(bt, 24)
		if btn.buttonType != bt {
			t.Errorf("expected button type %d, got %d", bt, btn.buttonType)
		}
	}
}

func TestWindowButton_SetTheme(t *testing.T) {
	btn := NewWindowButton(ButtonTypeClose, 24)
	theme := DefaultDecorationTheme()

	btn.SetTheme(theme)

	if btn.theme != theme {
		t.Error("theme not set correctly")
	}
}

func TestTitleBar_SetTheme(t *testing.T) {
	tb := NewTitleBar("Test", 400, 32)
	theme := DefaultDecorationTheme()
	theme.TitleBarHeight = 40

	tb.SetTheme(theme)

	if tb.theme.TitleBarHeight != 40 {
		t.Errorf("expected title bar height 40, got %d", tb.theme.TitleBarHeight)
	}
}
