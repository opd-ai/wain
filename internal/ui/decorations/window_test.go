package decorations

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
)

func TestWindowFrame_NewWindowFrame(t *testing.T) {
	wf := NewWindowFrame("Test Window", 640, 480)

	if wf.width != 640 {
		t.Errorf("width = %d; want 640", wf.width)
	}
	if wf.height != 480 {
		t.Errorf("height = %d; want 480", wf.height)
	}

	// Verify content dimensions exclude decorations
	theme := DefaultDecorationTheme()
	expectedContentWidth := 640 - 2*theme.ResizeHandleWidth
	expectedContentHeight := 480 - theme.TitleBarHeight - 2*theme.ResizeHandleWidth

	if wf.contentWidth != expectedContentWidth {
		t.Errorf("contentWidth = %d; want %d", wf.contentWidth, expectedContentWidth)
	}
	if wf.contentHeight != expectedContentHeight {
		t.Errorf("contentHeight = %d; want %d", wf.contentHeight, expectedContentHeight)
	}
}

func TestWindowFrame_Bounds(t *testing.T) {
	wf := NewWindowFrame("Test", 800, 600)

	w, h := wf.Bounds()
	if w != 800 || h != 600 {
		t.Errorf("Bounds() = (%d, %d); want (800, 600)", w, h)
	}
}

func TestWindowFrame_ContentBounds(t *testing.T) {
	wf := NewWindowFrame("Test", 800, 600)
	theme := DefaultDecorationTheme()

	w, h := wf.ContentBounds()
	expectedW := 800 - 2*theme.ResizeHandleWidth
	expectedH := 600 - theme.TitleBarHeight - 2*theme.ResizeHandleWidth

	if w != expectedW || h != expectedH {
		t.Errorf("ContentBounds() = (%d, %d); want (%d, %d)", w, h, expectedW, expectedH)
	}
}

func TestWindowFrame_ContentOffset(t *testing.T) {
	wf := NewWindowFrame("Test", 800, 600)
	theme := DefaultDecorationTheme()

	x, y := wf.ContentOffset()
	expectedX := theme.ResizeHandleWidth
	expectedY := theme.TitleBarHeight + theme.ResizeHandleWidth

	if x != expectedX || y != expectedY {
		t.Errorf("ContentOffset() = (%d, %d); want (%d, %d)", x, y, expectedX, expectedY)
	}
}

func TestWindowFrame_Resize(t *testing.T) {
	wf := NewWindowFrame("Test", 640, 480)

	wf.Resize(800, 600)

	if wf.width != 800 {
		t.Errorf("After Resize, width = %d; want 800", wf.width)
	}
	if wf.height != 600 {
		t.Errorf("After Resize, height = %d; want 600", wf.height)
	}

	// Verify content dimensions updated
	theme := DefaultDecorationTheme()
	expectedContentWidth := 800 - 2*theme.ResizeHandleWidth
	expectedContentHeight := 600 - theme.TitleBarHeight - 2*theme.ResizeHandleWidth

	if wf.contentWidth != expectedContentWidth {
		t.Errorf("After Resize, contentWidth = %d; want %d", wf.contentWidth, expectedContentWidth)
	}
	if wf.contentHeight != expectedContentHeight {
		t.Errorf("After Resize, contentHeight = %d; want %d", wf.contentHeight, expectedContentHeight)
	}
}

func TestWindowFrame_SetTitle(t *testing.T) {
	wf := NewWindowFrame("Initial Title", 640, 480)

	wf.SetTitle("New Title")

	// Access title bar to verify
	if wf.titleBar.title != "New Title" {
		t.Errorf("After SetTitle, title = %q; want %q", wf.titleBar.title, "New Title")
	}
}

func TestWindowFrame_HitTestResize(t *testing.T) {
	wf := NewWindowFrame("Test", 640, 480)

	// Test top-left corner
	edge := wf.HitTestResize(4, 4)
	if edge != ResizeEdgeTopLeft {
		t.Errorf("HitTestResize(4, 4) = %v; want ResizeEdgeTopLeft", edge)
	}

	// Test center (should be none)
	edge = wf.HitTestResize(320, 240)
	if edge != ResizeEdgeNone {
		t.Errorf("HitTestResize(320, 240) = %v; want ResizeEdgeNone", edge)
	}
}

func TestWindowFrame_HitTestTitleBarButton(t *testing.T) {
	wf := NewWindowFrame("Test", 640, 480)
	theme := DefaultDecorationTheme()
	hw := theme.ResizeHandleWidth

	// Calculate button position (rightmost button is close)
	buttonSize := theme.TitleBarHeight - 8
	spacing := theme.ButtonSpacing
	rightEdge := 640 - spacing
	closeX := rightEdge - buttonSize
	closeY := hw + spacing + buttonSize/2

	// Hit test the close button area
	button := wf.HitTestTitleBarButton(closeX+hw, closeY)
	if button == nil {
		t.Error("HitTestTitleBarButton on close button returned nil")
	} else if button.buttonType != ButtonTypeClose {
		t.Errorf("HitTestTitleBarButton returned button type %v; want ButtonTypeClose", button.buttonType)
	}
}

func TestWindowFrame_IsTitleBarDragArea(t *testing.T) {
	wf := NewWindowFrame("Test", 640, 480)
	theme := DefaultDecorationTheme()
	hw := theme.ResizeHandleWidth

	// Test middle of title bar (should be drag area)
	isDrag := wf.IsTitleBarDragArea(320, hw+theme.TitleBarHeight/2)
	if !isDrag {
		t.Error("IsTitleBarDragArea(center) = false; want true")
	}

	// Test over a button (should not be drag area)
	buttonSize := theme.TitleBarHeight - 8
	spacing := theme.ButtonSpacing
	rightEdge := 640 - spacing
	closeX := rightEdge - buttonSize
	closeY := hw + spacing + buttonSize/2

	isDrag = wf.IsTitleBarDragArea(closeX+hw, closeY)
	if isDrag {
		t.Error("IsTitleBarDragArea(over button) = true; want false")
	}

	// Test outside title bar (should not be drag area)
	isDrag = wf.IsTitleBarDragArea(320, 5)
	if isDrag {
		t.Error("IsTitleBarDragArea(outside) = true; want false")
	}
}

func TestWindowFrame_HandlePointerMotion(t *testing.T) {
	wf := NewWindowFrame("Test", 640, 480)

	// Test over resize handle
	isResize, edge := wf.HandlePointerMotion(5, 5)
	if !isResize {
		t.Error("HandlePointerMotion(5, 5) isResize = false; want true")
	}
	if edge != ResizeEdgeTopLeft {
		t.Errorf("HandlePointerMotion(5, 5) edge = %v; want ResizeEdgeTopLeft", edge)
	}

	// Test in center (not resize area)
	isResize, edge = wf.HandlePointerMotion(320, 240)
	if isResize {
		t.Error("HandlePointerMotion(320, 240) isResize = true; want false")
	}
	if edge != ResizeEdgeNone {
		t.Errorf("HandlePointerMotion(320, 240) edge = %v; want ResizeEdgeNone", edge)
	}
}

func TestWindowFrame_Draw(t *testing.T) {
	wf := NewWindowFrame("Test", 640, 480)
	buf, err := primitives.NewBuffer(640, 480)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	err = wf.Draw(buf, 0, 0)
	if err != nil {
		t.Errorf("Draw failed: %v", err)
	}
}

func TestWindowFrame_RenderToDisplayList(t *testing.T) {
	wf := NewWindowFrame("Test", 640, 480)
	dl := displaylist.New()

	wf.RenderToDisplayList(dl, 0, 0)

	if len(dl.Commands()) == 0 {
		t.Error("RenderToDisplayList produced no commands")
	}
}

func TestWindowFrame_TitleBar(t *testing.T) {
	wf := NewWindowFrame("Test", 640, 480)

	tb := wf.TitleBar()
	if tb == nil {
		t.Error("TitleBar() returned nil")
	}
	if tb != wf.titleBar {
		t.Error("TitleBar() did not return internal title bar")
	}
}

func TestWindowFrame_SetTheme(t *testing.T) {
	wf := NewWindowFrame("Test", 640, 480)

	newTheme := &Theme{
		TitleBarHeight:    40,
		ResizeHandleWidth: 10,
	}

	wf.SetTheme(newTheme)

	if wf.theme != newTheme {
		t.Error("SetTheme did not update frame theme")
	}
	if wf.titleBar.theme != newTheme {
		t.Error("SetTheme did not update title bar theme")
	}
	if wf.resizeHandles.theme != newTheme {
		t.Error("SetTheme did not update resize handles theme")
	}
}
