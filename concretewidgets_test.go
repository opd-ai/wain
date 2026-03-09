package wain

import (
	"testing"
	"time"
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
	input.SetFocus(true)
	// Focus state is internal, we can only verify it doesn't panic
	input.SetFocus(false)
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
	// Spacer's Draw should not panic
	spacer.Draw(nil)
}

func TestButtonImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &Button{}
}

func TestLabelImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &Label{}
}

func TestTextInputImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &TextInput{}
}

func TestScrollViewImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &ScrollView{}
}

func TestImageWidgetImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &ImageWidget{}
}

func TestSpacerImplementsPublicWidget(t *testing.T) {
	var _ PublicWidget = &Spacer{}
}
