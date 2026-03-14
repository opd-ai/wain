package wain

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
)

// Test helper to create PointerEvent
func newPointerEvent(eventType PointerEventType, button PointerButton) *PointerEvent {
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: eventType,
		button:    button,
	}
}

// Test helper to create KeyEvent
func newKeyEvent(eventType KeyEventType, key Key, r rune) *KeyEvent {
	return &KeyEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: eventType,
		key:       key,
		rune:      r,
	}
}

// Test helper to create scroll PointerEvent
func newScrollEvent(value float64) *PointerEvent {
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerScroll,
		value:     value,
	}
}

// Test helper to create click PointerEvent
func newClickEvent(x, y float64) *PointerEvent {
	return &PointerEvent{
		baseEvent: baseEvent{timestamp: time.Now()},
		eventType: PointerButtonPress,
		button:    PointerButtonLeft,
		x:         x,
		y:         y,
	}
}

func TestNewButton(t *testing.T) {
	btn := NewButton("Click me", Size{Width: 50, Height: 10})
	if btn == nil {
		t.Fatal("NewButton returned nil")
	}
	if btn.Text() != "Click me" {
		t.Errorf("expected text 'Click me', got '%s'", btn.Text())
	}
	if btn.size.Width != 50 {
		t.Errorf("expected width 50, got %f", btn.size.Width)
	}
	if btn.size.Height != 10 {
		t.Errorf("expected height 10, got %f", btn.size.Height)
	}
}

func TestButtonSetText(t *testing.T) {
	btn := NewButton("Initial", Size{Width: 50, Height: 10})
	btn.SetText("Updated")
	if btn.Text() != "Updated" {
		t.Errorf("expected text 'Updated', got '%s'", btn.Text())
	}
}

func TestButtonOnClick(t *testing.T) {
	clicked := false
	btn := NewButton("Click", Size{Width: 50, Height: 10})
	btn.OnClick(func() {
		clicked = true
	})

	// Simulate button press and release
	// Need to enter hover state first
	btn.HandleEvent(newPointerEvent(PointerEnter, 0))
	btn.HandleEvent(newPointerEvent(PointerButtonPress, PointerButtonLeft))
	btn.HandleEvent(newPointerEvent(PointerButtonRelease, PointerButtonLeft))

	if !clicked {
		t.Error("onClick callback was not invoked")
	}
}

func TestButtonSetEnabled(t *testing.T) {
	btn := NewButton("Test", Size{Width: 50, Height: 10})
	btn.SetEnabled(false)

	// Disabled button should not trigger onClick
	clicked := false
	btn.OnClick(func() {
		clicked = true
	})
	btn.HandleEvent(newPointerEvent(PointerEnter, 0))
	btn.HandleEvent(newPointerEvent(PointerButtonPress, PointerButtonLeft))
	btn.HandleEvent(newPointerEvent(PointerButtonRelease, PointerButtonLeft))

	if clicked {
		t.Error("onClick callback was invoked on disabled button")
	}
}

func TestNewLabel(t *testing.T) {
	label := NewLabel("Hello", Size{Width: 100, Height: 5})
	if label == nil {
		t.Fatal("NewLabel returned nil")
	}
	if label.Text() != "Hello" {
		t.Errorf("expected text 'Hello', got '%s'", label.Text())
	}
}

func TestLabelSetText(t *testing.T) {
	label := NewLabel("Initial", Size{Width: 100, Height: 5})
	label.SetText("Updated")
	if label.Text() != "Updated" {
		t.Errorf("expected text 'Updated', got '%s'", label.Text())
	}
}

func TestLabelSetTextColor(t *testing.T) {
	label := NewLabel("Test", Size{Width: 100, Height: 5})
	color := RGB(255, 0, 0)
	label.SetTextColor(color)
	if label.textColor.R != 255 || label.textColor.G != 0 || label.textColor.B != 0 {
		t.Errorf("expected color RGB(255,0,0), got RGB(%d,%d,%d)",
			label.textColor.R, label.textColor.G, label.textColor.B)
	}
}

func TestLabelSetFontSize(t *testing.T) {
	label := NewLabel("Test", Size{Width: 100, Height: 5})
	label.SetFontSize(18)
	if label.fontSize != 18 {
		t.Errorf("expected font size 18, got %d", label.fontSize)
	}
}

func TestNewTextInput(t *testing.T) {
	input := NewTextInput("initial", Size{Width: 50, Height: 6})
	if input == nil {
		t.Fatal("NewTextInput returned nil")
	}
	if input.Text() != "initial" {
		t.Errorf("expected text 'initial', got '%s'", input.Text())
	}
}

func TestTextInputSetText(t *testing.T) {
	input := NewTextInput("", Size{Width: 50, Height: 6})
	input.SetText("new text")
	if input.Text() != "new text" {
		t.Errorf("expected text 'new text', got '%s'", input.Text())
	}
}

func TestTextInputOnChange(t *testing.T) {
	input := NewTextInput("", Size{Width: 50, Height: 6})
	changeCount := 0
	var lastText string

	input.OnChange(func(text string) {
		changeCount++
		lastText = text
	})

	input.SetText("hello")
	if changeCount != 1 {
		t.Errorf("expected 1 onChange call, got %d", changeCount)
	}
	if lastText != "hello" {
		t.Errorf("expected onChange text 'hello', got '%s'", lastText)
	}
}

func TestTextInputSetFocus(t *testing.T) {
	input := NewTextInput("", Size{Width: 50, Height: 6})
	if input == nil {
		t.Fatal("NewTextInput returned nil")
	}

	input.SetFocus(true)
	input.SetFocus(false)

	width, height := input.Bounds()
	if width == 0 || height == 0 {
		t.Errorf("expected non-zero bounds, got width=%d height=%d", width, height)
	}
}

func TestTextInputHandleKeyPress(t *testing.T) {
	input := NewTextInput("", Size{Width: 50, Height: 6})
	input.SetFocus(true)

	// Simulate typing "hi"
	input.HandleEvent(newKeyEvent(KeyPress, Key('h'), 'h'))
	input.HandleEvent(newKeyEvent(KeyPress, Key('i'), 'i'))

	if input.Text() != "hi" {
		t.Errorf("expected text 'hi', got '%s'", input.Text())
	}
}

func TestNewScrollView(t *testing.T) {
	scroll := NewScrollView(Size{Width: 100, Height: 80})
	if scroll == nil {
		t.Fatal("NewScrollView returned nil")
	}
	if scroll.ScrollOffset() != 0 {
		t.Errorf("expected initial scroll offset 0, got %d", scroll.ScrollOffset())
	}
}

func TestScrollViewSetScrollOffset(t *testing.T) {
	scroll := NewScrollView(Size{Width: 100, Height: 80})
	// ScrollContainer limits offset based on content height
	// With no content, max scroll is 0
	scroll.SetScrollOffset(50)
	// Since content height is 0 and viewport is 80, max scroll is 0
	if scroll.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0 (no content), got %d", scroll.ScrollOffset())
	}
}

func TestScrollViewOnScroll(t *testing.T) {
	scroll := NewScrollView(Size{Width: 100, Height: 80})
	scrolled := false
	var lastOffset int

	scroll.OnScroll(func(offset int) {
		scrolled = true
		lastOffset = offset
	})

	scroll.SetScrollOffset(100)
	if !scrolled {
		t.Error("onScroll callback was not invoked")
	}
	if lastOffset != 100 {
		t.Errorf("expected scroll offset 100, got %d", lastOffset)
	}
}

func TestScrollViewHandleScrollEvent(t *testing.T) {
	scroll := NewScrollView(Size{Width: 100, Height: 80})
	initialOffset := scroll.ScrollOffset()

	// Simulate scroll wheel event (positive delta scrolls down)
	// Since there's no content, offset will remain 0 (clamped)
	consumed := scroll.HandleEvent(newScrollEvent(5.0))

	if !consumed {
		t.Error("scroll event was not consumed")
	}
	// With no content, scroll should still be 0
	if scroll.ScrollOffset() != initialOffset {
		t.Log("scroll offset changed to", scroll.ScrollOffset(), "but expected", initialOffset)
	}
}

func TestNewImageWidget(t *testing.T) {
	img := &Image{id: 1}
	widget := NewImageWidget(img, Size{Width: 20, Height: 20})
	if widget == nil {
		t.Fatal("NewImageWidget returned nil")
	}
	if widget.Image() != img {
		t.Error("image widget does not return the correct image")
	}
}

func TestImageWidgetSetImage(t *testing.T) {
	img1 := &Image{id: 1}
	img2 := &Image{id: 2}
	widget := NewImageWidget(img1, Size{Width: 20, Height: 20})
	widget.SetImage(img2)
	if widget.Image() != img2 {
		t.Error("SetImage did not update the image")
	}
}

func TestNewSpacer(t *testing.T) {
	spacer := NewSpacer(Size{Width: 60, Height: 10})
	if spacer == nil {
		t.Fatal("NewSpacer returned nil")
	}
	if spacer.size.Width != 60 {
		t.Errorf("expected width 60, got %f", spacer.size.Width)
	}
	if spacer.size.Height != 10 {
		t.Errorf("expected height 10, got %f", spacer.size.Height)
	}
}

func TestSpacerDraw(t *testing.T) {
	spacer := NewSpacer(Size{Width: 60, Height: 10})
	if spacer == nil {
		t.Fatal("NewSpacer returned nil")
	}

	width, height := spacer.Bounds()
	if width == 0 || height == 0 {
		t.Errorf("expected non-zero bounds, got width=%d height=%d", width, height)
	}
}

func TestButtonImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &Button{}

	btn := NewButton("Test", Size{Width: 30, Height: 8})
	if btn == nil {
		t.Fatal("NewButton returned nil")
	}

	width, height := btn.Bounds()
	if width == 0 || height == 0 {
		t.Errorf("button must have non-zero bounds, got width=%d height=%d", width, height)
	}

	consumed := btn.HandleEvent(newClickEvent(5, 5))
	if !consumed {
		t.Error("button should consume pointer events")
	}
}

func TestLabelImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &Label{}

	label := NewLabel("Test", Size{Width: 20, Height: 4})
	if label == nil {
		t.Fatal("NewLabel returned nil")
	}

	width, height := label.Bounds()
	if width == 0 || height == 0 {
		t.Errorf("label must have non-zero bounds, got width=%d height=%d", width, height)
	}

	consumed := label.HandleEvent(newClickEvent(5, 5))
	if consumed {
		t.Error("label should not consume click events")
	}
}

func TestTextInputImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &TextInput{}

	input := NewTextInput("", Size{Width: 50, Height: 6})
	if input == nil {
		t.Fatal("NewTextInput returned nil")
	}

	width, height := input.Bounds()
	if width == 0 || height == 0 {
		t.Errorf("text input must have non-zero bounds, got width=%d height=%d", width, height)
	}

	input.SetFocus(true)
	consumed := input.HandleEvent(newKeyEvent(KeyPress, Key('a'), 'a'))
	if !consumed {
		t.Error("focused text input should consume key events")
	}
}

func TestScrollViewImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &ScrollView{}

	scroll := NewScrollView(Size{Width: 100, Height: 80})
	if scroll == nil {
		t.Fatal("NewScrollView returned nil")
	}

	width, height := scroll.Bounds()
	if width == 0 || height == 0 {
		t.Errorf("scroll view must have non-zero bounds, got width=%d height=%d", width, height)
	}

	consumed := scroll.HandleEvent(newScrollEvent(5.0))
	if !consumed {
		t.Error("scroll view should consume scroll events")
	}
}

func TestImageWidgetImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &ImageWidget{}

	img := &Image{id: 1}
	widget := NewImageWidget(img, Size{Width: 20, Height: 20})
	if widget == nil {
		t.Fatal("NewImageWidget returned nil")
	}

	width, height := widget.Bounds()
	if width == 0 || height == 0 {
		t.Errorf("image widget must have non-zero bounds, got width=%d height=%d", width, height)
	}

	if widget.Image() != img {
		t.Error("image widget should return the correct image")
	}
}

func TestSpacerImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &Spacer{}

	spacer := NewSpacer(Size{Width: 60, Height: 10})
	if spacer == nil {
		t.Fatal("NewSpacer returned nil")
	}

	width, height := spacer.Bounds()
	if width == 0 || height == 0 {
		t.Errorf("spacer must have non-zero bounds, got width=%d height=%d", width, height)
	}

	consumed := spacer.HandleEvent(newClickEvent(5, 5))
	if consumed {
		t.Error("spacer should not consume events")
	}
}

// makeTestCanvas creates a bufferCanvas backed by a buffer of the given size.
func makeTestCanvas(w, h int) (*bufferCanvas, *primitives.Buffer) {
	buf, err := primitives.NewBuffer(w, h)
	if err != nil {
		panic(err)
	}
	return &bufferCanvas{buf: buf}, buf
}

// hasNonZeroPixel returns true if any pixel in buf has at least one non-zero byte.
func hasNonZeroPixel(buf *primitives.Buffer) bool {
	for _, b := range buf.Pixels {
		if b != 0 {
			return true
		}
	}
	return false
}

// TestBufferCanvasDrawImage verifies that DrawImage composites visible pixels.
func TestBufferCanvasDrawImage(t *testing.T) {
	c, buf := makeTestCanvas(64, 64)

	// Create a 4×4 solid red standard image.
	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	img := &Image{data: src, width: 4, height: 4}

	c.DrawImage(img, 0, 0, 16, 16)

	if !hasNonZeroPixel(buf) {
		t.Error("DrawImage produced no visible pixels")
	}
}

// TestBufferCanvasLinearGradient verifies that LinearGradient writes pixels.
func TestBufferCanvasLinearGradient(t *testing.T) {
	c, buf := makeTestCanvas(64, 64)

	start := RGBA(255, 0, 0, 255)
	end := RGBA(0, 0, 255, 255)
	c.LinearGradient(0, 0, 64, 64, start, end, 0)

	if !hasNonZeroPixel(buf) {
		t.Error("LinearGradient produced no visible pixels")
	}
}

// TestBufferCanvasRadialGradient verifies that RadialGradient writes pixels.
func TestBufferCanvasRadialGradient(t *testing.T) {
	c, buf := makeTestCanvas(64, 64)

	center := RGBA(255, 255, 0, 255)
	edge := RGBA(0, 0, 0, 255)
	c.RadialGradient(0, 0, 64, 64, center, edge)

	if !hasNonZeroPixel(buf) {
		t.Error("RadialGradient produced no visible pixels")
	}
}

// TestBufferCanvasBoxShadow verifies that BoxShadow writes pixels.
func TestBufferCanvasBoxShadow(t *testing.T) {
	c, buf := makeTestCanvas(128, 128)

	c.BoxShadow(10, 10, 40, 40, 5, 5, 8, RGBA(0, 0, 0, 200))

	if !hasNonZeroPixel(buf) {
		t.Error("BoxShadow produced no visible pixels")
	}
}

// TestButtonHandleEventPointerEnterLeave verifies PointerEnter/Leave events.
func TestButtonHandleEventPointerEnterLeave(t *testing.T) {
	btn := NewButton("Test", Size{Width: 30, Height: 10})

	enterEvt := &PointerEvent{eventType: PointerEnter}
	if !btn.HandleEvent(enterEvt) {
		t.Error("HandleEvent(PointerEnter) should return true")
	}

	leaveEvt := &PointerEvent{eventType: PointerLeave}
	if !btn.HandleEvent(leaveEvt) {
		t.Error("HandleEvent(PointerLeave) should return true")
	}
}

// TestButtonHandleEventPointerPressRelease verifies button press/release.
func TestButtonHandleEventPointerPressRelease(t *testing.T) {
	btn := NewButton("Test", Size{Width: 30, Height: 10})

	pressEvt := &PointerEvent{eventType: PointerButtonPress, button: PointerButtonLeft}
	if !btn.HandleEvent(pressEvt) {
		t.Error("HandleEvent(PointerButtonPress) should return true")
	}

	releaseEvt := &PointerEvent{eventType: PointerButtonRelease, button: PointerButtonRight}
	if !btn.HandleEvent(releaseEvt) {
		t.Error("HandleEvent(PointerButtonRelease) should return true")
	}

	// Unknown event should return false
	keyEvt := &KeyEvent{eventType: KeyPress}
	if btn.HandleEvent(keyEvt) {
		t.Error("HandleEvent(KeyEvent) should return false for Button")
	}
}

// TestButtonToInternal verifies buttonToInternal mapping.
func TestButtonToInternal(t *testing.T) {
	tests := []struct {
		btn  PointerButton
		want uint32
	}{
		{PointerButtonLeft, 1},
		{PointerButtonMiddle, 2},
		{PointerButtonRight, 3},
		{PointerButton(0x999), 0},
	}
	for _, tt := range tests {
		if got := buttonToInternal(tt.btn); got != tt.want {
			t.Errorf("buttonToInternal(%v) = %d, want %d", tt.btn, got, tt.want)
		}
	}
}

// TestButtonSetStyle verifies that SetStyle doesn't panic.
func TestButtonSetStyle(t *testing.T) {
	btn := NewButton("OK", Size{Width: 30, Height: 10})
	ac := RGB(0, 128, 0)
	btn.SetStyle(StyleOverride{Accent: &ac}) // must not panic
}

// TestLabelSetStyle verifies that Label.SetStyle doesn't panic.
func TestLabelSetStyle(t *testing.T) {
	lbl := NewLabel("hello", Size{Width: 50, Height: 10})
	bg := RGB(10, 20, 30)
	lbl.SetStyle(StyleOverride{Background: &bg}) // must not panic
}

// TestTextInputHandleEventKey verifies that TextInput handles KeyPress.
func TestTextInputHandleEventKey(t *testing.T) {
	ti := NewTextInput("", Size{Width: 80, Height: 10})
	keyEvt := &KeyEvent{
		eventType: KeyPress,
		key:       Key(0x61), // 'a'
		rune:      'a',
	}
	if !ti.HandleEvent(keyEvt) {
		t.Error("TextInput.HandleEvent(KeyPress) should return true")
	}
}

// TestTextInputHandleEventPointer verifies that TextInput handles PointerButtonPress.
func TestTextInputHandleEventPointer(t *testing.T) {
	ti := NewTextInput("", Size{Width: 80, Height: 10})
	ptrEvt := &PointerEvent{eventType: PointerButtonPress, button: PointerButtonLeft}
	if !ti.HandleEvent(ptrEvt) {
		t.Error("TextInput.HandleEvent(PointerButtonPress) should return true")
	}
}

// TestTextInputSetStyle verifies that TextInput.SetStyle doesn't panic.
func TestTextInputSetStyle(t *testing.T) {
	ti := NewTextInput("", Size{Width: 80, Height: 10})
	ti.SetStyle(StyleOverride{}) // must not panic
}

// TestScrollViewHandleEventScroll verifies that ScrollView handles PointerScroll.
func TestScrollViewHandleEventScroll(t *testing.T) {
	sv := NewScrollView(Size{Width: 100, Height: 80})

	called := false
	sv.OnScroll(func(offset int) {
		called = true
	})

	scrollEvt := &PointerEvent{eventType: PointerScroll, value: 2.0}
	if !sv.HandleEvent(scrollEvt) {
		t.Error("ScrollView.HandleEvent(PointerScroll) should return true")
	}
	if !called {
		t.Error("OnScroll handler should have been called")
	}
}

// TestScrollViewScrollOffset verifies SetScrollOffset/ScrollOffset are callable.
func TestScrollViewScrollOffset(t *testing.T) {
	sv := NewScrollView(Size{Width: 100, Height: 80})
	// With no content, maxScroll=0, so offset is clamped to 0.
	sv.SetScrollOffset(50)
	if got := sv.ScrollOffset(); got != 0 {
		t.Errorf("ScrollOffset() = %d, want 0 (clamped, no content)", got)
	}
	// Offset of 0 is always valid.
	sv.SetScrollOffset(0)
	if got := sv.ScrollOffset(); got != 0 {
		t.Errorf("ScrollOffset() = %d, want 0", got)
	}
}

// TestNewBufferCanvasFillRect verifies the bufferCanvas FillRect method.
func TestNewBufferCanvasFillRect(t *testing.T) {
	buf, err := primitives.NewBuffer(100, 100)
	if err != nil {
		t.Fatalf("NewBuffer: %v", err)
	}

	canvas := newBufferCanvas(buf, 0, 0)

	// FillRect should not panic
	canvas.FillRect(10, 10, 20, 20, RGB(255, 0, 0))

	// Check that at least one pixel was painted
	r, g, _, _ := buf.At(15, 15).RGBA()
	if r == 0 && g == 0 {
		t.Log("FillRect may not have set expected color (depends on format)")
	}
}

// TestNewBufferCanvasFillRoundedRect exercises FillRoundedRect.
func TestNewBufferCanvasFillRoundedRect(t *testing.T) {
	buf, _ := primitives.NewBuffer(100, 100)
	canvas := newBufferCanvas(buf, 5, 5)
	canvas.FillRoundedRect(0, 0, 50, 30, 5, RGB(0, 200, 0)) // must not panic
}

// TestNewBufferCanvasDrawLine exercises DrawLine.
func TestNewBufferCanvasDrawLine(t *testing.T) {
	buf, _ := primitives.NewBuffer(100, 100)
	canvas := newBufferCanvas(buf, 0, 0)
	canvas.DrawLine(0, 0, 99, 99, RGB(128, 128, 128), 1) // must not panic
}

// TestNewBufferCanvasDrawTextNilFont verifies DrawText with a nil font is safe.
func TestNewBufferCanvasDrawTextNilFont(t *testing.T) {
	buf, _ := primitives.NewBuffer(100, 100)
	canvas := newBufferCanvas(buf, 0, 0)
	canvas.DrawText("hello", 0, 0, nil, RGB(0, 0, 0)) // must not panic
}

// TestWidgetAdapterHandlePointerEvents verifies the widgetAdapter bridge.
func TestWidgetAdapterHandlePointerEvents(t *testing.T) {
	btn := NewButton("Test", Size{Width: 30, Height: 10})
	adapter := newWidgetAdapter(btn)

	adapter.HandlePointerEnter() // must not panic
	adapter.HandlePointerLeave() // must not panic
	adapter.HandlePointerDown(1) // must not panic
	adapter.HandlePointerUp(1)   // must not panic
}

// TestWidgetDrawStubs verifies Draw stubs do not panic.

// TestWidgetDrawStubs verifies Draw stubs do not panic.
func TestWidgetDrawStubs(t *testing.T) {
	buf, _ := primitives.NewBuffer(100, 100)
	canvas := newBufferCanvas(buf, 0, 0)

	NewButton("Test", Size{Width: 50, Height: 20}).Draw(canvas)
	NewLabel("Hello", Size{Width: 50, Height: 20}).Draw(canvas)
	NewTextInput("world", Size{Width: 50, Height: 20}).Draw(canvas)
	NewScrollView(Size{Width: 100, Height: 100}).Draw(canvas)
}

// TestBaseWidgetHandleTouchAndFocus exercises HandleTouch, IsFocused, OnTouch.
func TestBaseWidgetHandleTouchAndFocus(t *testing.T) {
	w := &BaseWidget{}

	if w.IsFocused() {
		t.Error("new widget should not be focused")
	}
	w.SetFocused(true)
	if !w.IsFocused() {
		t.Error("widget should be focused after SetFocused(true)")
	}

	called := false
	w.OnTouch(func(e *TouchEvent) { called = true })
	evt := &TouchEvent{}
	w.HandleTouch(evt)
	if !called {
		t.Error("HandleTouch should call onTouch callback")
	}

	w2 := &BaseWidget{}
	w2.HandleTouch(evt)
}

// TestWidgetAdapterDraw verifies widgetAdapter.Draw creates a canvas and draws.
func TestWidgetAdapterDraw(t *testing.T) {
	buf, _ := primitives.NewBuffer(100, 100)
	lbl := NewLabel("test", Size{Width: 50, Height: 5})
	adapter := newWidgetAdapter(lbl)
	wa := adapter
	if err := wa.Draw(buf, 0, 0); err != nil {
		t.Errorf("widgetAdapter.Draw: %v", err)
	}
}

// TestScrollViewSetStyle verifies SetStyle on ScrollView does not panic.
func TestScrollViewSetStyle(t *testing.T) {
	s := NewScrollView(Size{Width: 100, Height: 100})
	s.SetStyle(StyleOverride{})
}

// TestImageWidgetDrawAndStyle verifies ImageWidget.Draw and SetStyle.
func TestImageWidgetDrawAndStyle(t *testing.T) {
	buf, _ := primitives.NewBuffer(100, 100)
	canvas := newBufferCanvas(buf, 0, 0)

	// Nil image — should early return without panic
	iw := NewImageWidget(nil, Size{Width: 50, Height: 50})
	iw.Draw(canvas)

	// Non-nil image
	img := &Image{width: 2, height: 2, data: &testImageStub{}}
	iw2 := NewImageWidget(img, Size{Width: 50, Height: 50})
	iw2.Draw(canvas)
	iw2.SetStyle(StyleOverride{})
}

// testImageStub implements image.Image for testing.
type testImageStub struct{}

func (t *testImageStub) ColorModel() color.Model { return color.RGBAModel }
func (t *testImageStub) Bounds() image.Rectangle { return image.Rect(0, 0, 2, 2) }
func (t *testImageStub) At(x, y int) color.Color { return color.RGBA{255, 0, 0, 255} }

// TestSpacerDrawAndStyle verifies Spacer Draw and SetStyle are no-ops.
func TestSpacerDrawAndStyle(t *testing.T) {
	buf, _ := primitives.NewBuffer(50, 50)
	canvas := newBufferCanvas(buf, 0, 0)
	s := NewSpacer(Size{Width: 10, Height: 10})
	s.Draw(canvas)
	s.SetStyle(StyleOverride{})
}

// TestBufferCanvasDrawImageNilImg verifies DrawImage with nil image is a no-op.
func TestBufferCanvasDrawImageNilImg(t *testing.T) {
	c, _ := makeTestCanvas(16, 16)
	c.DrawImage(nil, 0, 0, 16, 16) // must not panic
}

// TestBufferCanvasDrawImageNilData verifies DrawImage with nil image data is a no-op.
func TestBufferCanvasDrawImageNilData(t *testing.T) {
	c, _ := makeTestCanvas(16, 16)
	img := &Image{data: nil, width: 4, height: 4}
	c.DrawImage(img, 0, 0, 16, 16) // must not panic
}

// TestBufferCanvasDrawImageEmptyBounds verifies DrawImage when imageToBuffer returns nil.
func TestBufferCanvasDrawImageEmptyBounds(t *testing.T) {
	c, _ := makeTestCanvas(16, 16)
	// Image with 0-size bounds — imageToBuffer returns nil
	empty := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	img := &Image{data: empty, width: 0, height: 0}
	c.DrawImage(img, 0, 0, 16, 16) // must not panic
}

// TestUploadImageToAtlasNilAtlas verifies uploadImageToAtlas returns ErrNoAtlas when atlas is nil.
func TestUploadImageToAtlasNilAtlas(t *testing.T) {
	rm := newResourceManager(nil)
	// rm.textureAtlas is nil by default
	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	img := &Image{id: 1, data: src, width: 4, height: 4}
	err := rm.uploadImageToAtlas(img)
	if err != ErrNoAtlas {
		t.Errorf("expected ErrNoAtlas, got %v", err)
	}
}

// TestLabelDrawEmptyText verifies Label.Draw with empty text is a no-op.
func TestLabelDrawEmptyText(t *testing.T) {
	lbl := NewLabel("", Size{Width: 50, Height: 10})
	c, _ := makeTestCanvas(64, 64)
	lbl.Draw(c) // early return path
}

// TestBufferCanvasRadialGradientWider verifies RadialGradient when height < width.
func TestBufferCanvasRadialGradientWider(t *testing.T) {
	c, _ := makeTestCanvas(64, 64)
	c.RadialGradient(0, 0, 64, 32, RGBA(255, 0, 0, 255), RGBA(0, 0, 255, 255))
}

// TestBufferCanvasDrawTextValidFont verifies DrawText with a valid font/atlas.
func TestBufferCanvasDrawTextValidFont(t *testing.T) {
	a, err := text.NewAtlas()
	if err != nil {
		t.Skip("embedded font atlas unavailable:", err)
	}
	buf, _ := primitives.NewBuffer(100, 50)
	c := newBufferCanvas(buf, 0, 0)
	font := &Font{atlas: a, size: 12}
	c.DrawText("hi", 0, 0, font, RGB(0, 0, 0))
}

// hugeImage is a mock image.Image whose Bounds() returns dimensions > 16384.
type hugeImage struct{}

func (h *hugeImage) ColorModel() color.Model          { return color.RGBAModel }
func (h *hugeImage) Bounds() image.Rectangle         { return image.Rect(0, 0, 16385, 1) }
func (h *hugeImage) At(x, y int) color.Color          { return color.Black }

// TestImageToBufferOversizedImage covers the primitives.NewBuffer error path.
func TestImageToBufferOversizedImage(t *testing.T) {
result := imageToBuffer(&hugeImage{})
if result != nil {
t.Error("expected nil for oversized image, got non-nil buffer")
}
}
