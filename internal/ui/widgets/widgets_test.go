package widgets

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/core"
)

func TestNewButton(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		width  int
		height int
	}{
		{"basic button", "Click me", 100, 30},
		{"wide button", "Submit", 200, 40},
		{"empty text", "", 50, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			btn := NewButton(tt.text, tt.width, tt.height)
			if btn == nil {
				t.Fatal("NewButton returned nil")
			}
			if btn.text != tt.text {
				t.Errorf("text = %q, want %q", btn.text, tt.text)
			}
			if w, h := btn.Bounds(); w != tt.width || h != tt.height {
				t.Errorf("Bounds() = (%d, %d), want (%d, %d)", w, h, tt.width, tt.height)
			}
			if !btn.enabled {
				t.Error("button should be enabled by default")
			}
			if btn.state != PointerStateNormal {
				t.Errorf("state = %v, want %v", btn.state, PointerStateNormal)
			}
		})
	}
}

func TestButtonPointerInteraction(t *testing.T) {
	btn := NewButton("Test", 100, 30)

	// Initial state
	if btn.state != PointerStateNormal {
		t.Errorf("initial state = %v, want %v", btn.state, PointerStateNormal)
	}

	// Hover
	btn.HandlePointerEnter()
	if btn.state != PointerStateHover {
		t.Errorf("after enter: state = %v, want %v", btn.state, PointerStateHover)
	}

	// Press
	btn.HandlePointerDown(1) // Left button
	if btn.state != PointerStatePressed {
		t.Errorf("after down: state = %v, want %v", btn.state, PointerStatePressed)
	}

	// Release
	clicked := false
	btn.SetOnClick(func() { clicked = true })
	btn.HandlePointerUp(1)
	if btn.state != PointerStateHover {
		t.Errorf("after up: state = %v, want %v", btn.state, PointerStateHover)
	}
	if !clicked {
		t.Error("onClick callback not called")
	}

	// Leave
	btn.HandlePointerLeave()
	if btn.state != PointerStateNormal {
		t.Errorf("after leave: state = %v, want %v", btn.state, PointerStateNormal)
	}
}

func TestButtonDisabled(t *testing.T) {
	btn := NewButton("Test", 100, 30)
	btn.SetEnabled(false)

	if btn.enabled {
		t.Error("button should be disabled")
	}

	// Interactions should be ignored
	btn.HandlePointerEnter()
	if btn.state != PointerStateNormal {
		t.Error("disabled button should not respond to hover")
	}

	btn.HandlePointerDown(1)
	if btn.state != PointerStateNormal {
		t.Error("disabled button should not respond to press")
	}
}

func TestButtonDraw(t *testing.T) {
	buf, err := core.NewBuffer(200, 100)
	if err != nil {
		t.Fatalf("NewBuffer failed: %v", err)
	}

	btn := NewButton("Test", 100, 30)

	// Draw normal state
	if err := btn.Draw(buf, 10, 10); err != nil {
		t.Errorf("Draw failed: %v", err)
	}

	// Draw with nil buffer
	if err := btn.Draw(nil, 10, 10); err == nil {
		t.Error("Draw should fail with nil buffer")
	}
}

func TestNewTextInput(t *testing.T) {
	tests := []struct {
		name        string
		placeholder string
		width       int
		height      int
	}{
		{"basic input", "Enter text...", 200, 30},
		{"wide input", "Search", 300, 40},
		{"no placeholder", "", 150, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := NewTextInput(tt.placeholder, tt.width, tt.height)
			if input == nil {
				t.Fatal("NewTextInput returned nil")
			}
			if input.placeholder != tt.placeholder {
				t.Errorf("placeholder = %q, want %q", input.placeholder, tt.placeholder)
			}
			if w, h := input.Bounds(); w != tt.width || h != tt.height {
				t.Errorf("Bounds() = (%d, %d), want (%d, %d)", w, h, tt.width, tt.height)
			}
			if input.focused {
				t.Error("input should not be focused by default")
			}
			if !input.enabled {
				t.Error("input should be enabled by default")
			}
		})
	}
}

func TestTextInputFocus(t *testing.T) {
	input := NewTextInput("Test", 200, 30)

	if input.focused {
		t.Error("initial focused state should be false")
	}

	input.HandlePointerDown(1)
	if !input.focused {
		t.Error("should be focused after pointer down")
	}

	input.SetEnabled(false)
	if input.focused {
		t.Error("should lose focus when disabled")
	}
}

func TestTextInputKeyPress(t *testing.T) {
	input := NewTextInput("Test", 200, 30)
	input.focused = true

	// Type some text
	input.HandleKeyPress('H', "H")
	input.HandleKeyPress('e', "e")
	input.HandleKeyPress('l', "l")
	input.HandleKeyPress('l', "l")
	input.HandleKeyPress('o', "o")

	if input.Text() != "Hello" {
		t.Errorf("text = %q, want %q", input.Text(), "Hello")
	}

	if input.cursorPos != 5 {
		t.Errorf("cursorPos = %d, want 5", input.cursorPos)
	}
}

func TestTextInputBackspace(t *testing.T) {
	input := NewTextInput("Test", 200, 30)
	input.focused = true
	input.SetText("Hello")
	input.cursorPos = 5

	input.HandleBackspace()
	if input.Text() != "Hell" {
		t.Errorf("text = %q, want %q", input.Text(), "Hell")
	}

	if input.cursorPos != 4 {
		t.Errorf("cursorPos = %d, want 4", input.cursorPos)
	}

	// Backspace at position 0 should do nothing
	input.cursorPos = 0
	input.HandleBackspace()
	if input.Text() != "Hell" {
		t.Error("backspace at position 0 should not change text")
	}
}

func TestTextInputDelete(t *testing.T) {
	input := NewTextInput("Test", 200, 30)
	input.focused = true
	input.SetText("Hello")
	input.cursorPos = 1

	input.HandleDelete()
	if input.Text() != "Hllo" {
		t.Errorf("text = %q, want %q", input.Text(), "Hllo")
	}

	if input.cursorPos != 1 {
		t.Errorf("cursorPos = %d, want 1", input.cursorPos)
	}

	// Delete at end should do nothing
	input.cursorPos = len(input.text)
	input.HandleDelete()
	if input.Text() != "Hllo" {
		t.Error("delete at end should not change text")
	}
}

func TestTextInputCursorMove(t *testing.T) {
	input := NewTextInput("Test", 200, 30)
	input.focused = true
	input.SetText("Hello")
	input.cursorPos = 2

	// Move right
	input.HandleCursorMove(1)
	if input.cursorPos != 3 {
		t.Errorf("cursorPos = %d, want 3", input.cursorPos)
	}

	// Move left
	input.HandleCursorMove(-2)
	if input.cursorPos != 1 {
		t.Errorf("cursorPos = %d, want 1", input.cursorPos)
	}

	// Clamp to bounds
	input.HandleCursorMove(-10)
	if input.cursorPos != 0 {
		t.Errorf("cursorPos = %d, want 0 (clamped)", input.cursorPos)
	}

	input.HandleCursorMove(100)
	if input.cursorPos != len(input.text) {
		t.Errorf("cursorPos = %d, want %d (clamped)", input.cursorPos, len(input.text))
	}
}

func TestTextInputOnChange(t *testing.T) {
	input := NewTextInput("Test", 200, 30)
	input.focused = true

	changeCount := 0
	var lastText string
	input.SetOnChange(func(text string) {
		changeCount++
		lastText = text
	})

	input.HandleKeyPress('A', "A")
	if changeCount != 1 || lastText != "A" {
		t.Errorf("onChange not called correctly: count=%d, text=%q", changeCount, lastText)
	}

	input.HandleBackspace()
	if changeCount != 2 || lastText != "" {
		t.Errorf("onChange not called on backspace: count=%d, text=%q", changeCount, lastText)
	}
}

func TestTextInputDraw(t *testing.T) {
	buf, err := core.NewBuffer(300, 100)
	if err != nil {
		t.Fatalf("NewBuffer failed: %v", err)
	}

	input := NewTextInput("Placeholder", 200, 30)

	// Draw normal state
	if err := input.Draw(buf, 10, 10); err != nil {
		t.Errorf("Draw failed: %v", err)
	}

	// Draw focused state
	input.focused = true
	if err := input.Draw(buf, 10, 50); err != nil {
		t.Errorf("Draw failed: %v", err)
	}

	// Draw with nil buffer
	if err := input.Draw(nil, 10, 10); err == nil {
		t.Error("Draw should fail with nil buffer")
	}
}

func TestNewScrollContainer(t *testing.T) {
	container := NewScrollContainer(400, 300)
	if container == nil {
		t.Fatal("NewScrollContainer returned nil")
	}

	if w, h := container.Bounds(); w != 400 || h != 300 {
		t.Errorf("Bounds() = (%d, %d), want (400, 300)", w, h)
	}

	if container.scrollOffset != 0 {
		t.Error("scrollOffset should be 0 initially")
	}

	if container.contentHeight != 0 {
		t.Error("contentHeight should be 0 initially")
	}
}

func TestScrollContainerAddChild(t *testing.T) {
	container := NewScrollContainer(400, 300)

	btn1 := NewButton("Button 1", 100, 30)
	btn2 := NewButton("Button 2", 100, 30)

	container.AddChild(btn1)
	if len(container.children) != 1 {
		t.Errorf("children count = %d, want 1", len(container.children))
	}
	if container.contentHeight != 30 {
		t.Errorf("contentHeight = %d, want 30", container.contentHeight)
	}

	container.AddChild(btn2)
	if len(container.children) != 2 {
		t.Errorf("children count = %d, want 2", len(container.children))
	}
	if container.contentHeight != 60 {
		t.Errorf("contentHeight = %d, want 60", container.contentHeight)
	}
}

func TestScrollContainerHandleScroll(t *testing.T) {
	container := NewScrollContainer(400, 200)

	// Add children that exceed container height
	for i := 0; i < 10; i++ {
		container.AddChild(NewButton("Button", 100, 30))
	}

	// contentHeight should be 300 (10 * 30)
	if container.contentHeight != 300 {
		t.Errorf("contentHeight = %d, want 300", container.contentHeight)
	}

	// Scroll down
	container.HandleScroll(50)
	if container.scrollOffset != 50 {
		t.Errorf("scrollOffset = %d, want 50", container.scrollOffset)
	}

	// Scroll down more (will be clamped to max = 100)
	container.HandleScroll(100)
	if container.scrollOffset != 100 {
		t.Errorf("scrollOffset = %d, want 100 (clamped to max)", container.scrollOffset)
	}

	// Scroll down again - should stay at max (contentHeight - height = 300 - 200 = 100)
	container.HandleScroll(100)
	if container.scrollOffset != 100 {
		t.Errorf("scrollOffset = %d, want 100 (clamped)", container.scrollOffset)
	}

	// Scroll up
	container.HandleScroll(-50)
	if container.scrollOffset != 50 {
		t.Errorf("scrollOffset = %d, want 50", container.scrollOffset)
	}

	// Clamp to 0
	container.HandleScroll(-100)
	if container.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0 (clamped)", container.scrollOffset)
	}
}

func TestScrollContainerNoOverflow(t *testing.T) {
	container := NewScrollContainer(400, 300)

	// Add children that don't exceed container height
	container.AddChild(NewButton("Button 1", 100, 30))
	container.AddChild(NewButton("Button 2", 100, 30))

	// Scroll should be clamped to 0
	container.HandleScroll(50)
	if container.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0 (no overflow)", container.scrollOffset)
	}
}

func TestScrollContainerDraw(t *testing.T) {
	buf, err := core.NewBuffer(500, 400)
	if err != nil {
		t.Fatalf("NewBuffer failed: %v", err)
	}

	container := NewScrollContainer(400, 300)
	container.AddChild(NewButton("Button 1", 100, 30))
	container.AddChild(NewButton("Button 2", 100, 30))

	// Draw normal state
	if err := container.Draw(buf, 10, 10); err != nil {
		t.Errorf("Draw failed: %v", err)
	}

	// Draw with scroll offset
	container.HandleScroll(10)
	if err := container.Draw(buf, 10, 10); err != nil {
		t.Errorf("Draw failed: %v", err)
	}

	// Draw with nil buffer
	if err := container.Draw(nil, 10, 10); err == nil {
		t.Error("Draw should fail with nil buffer")
	}
}

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	if theme == nil {
		t.Fatal("DefaultTheme returned nil")
	}

	if theme.FontSize <= 0 {
		t.Error("FontSize should be positive")
	}

	if theme.BorderRadius < 0 {
		t.Error("BorderRadius should be non-negative")
	}
}

func TestWidgetInterface(t *testing.T) {
	var _ Widget = (*Button)(nil)
	var _ Widget = (*TextInput)(nil)
	var _ Widget = (*ScrollContainer)(nil)
}

func BenchmarkButtonDraw(b *testing.B) {
	buf, _ := core.NewBuffer(800, 600)
	btn := NewButton("Click me", 100, 30)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		btn.Draw(buf, 10, 10)
	}
}

func BenchmarkTextInputDraw(b *testing.B) {
	buf, _ := core.NewBuffer(800, 600)
	input := NewTextInput("Placeholder", 200, 30)
	input.SetText("Some text content")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input.Draw(buf, 10, 10)
	}
}
