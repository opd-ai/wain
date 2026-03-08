package consumer

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/core"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/ui/widgets"
)

// TestWidgetRenderToDisplayList tests that widgets can render to a display list.
func TestWidgetRenderToDisplayList(t *testing.T) {
	atlas, err := text.NewAtlas()
	if err != nil {
		t.Fatalf("Failed to create atlas: %v", err)
	}

	// Create a button
	btn := widgets.NewButton("Click Me", 100, 30)
	btn.SetAtlas(atlas)

	// Render to display list
	dl := displaylist.New()
	btn.RenderToDisplayList(dl, 10, 10)

	if dl.Len() == 0 {
		t.Error("Expected button to emit display list commands")
	}

	// Commands should include: background, border (4 lines), text
	if dl.Len() < 2 {
		t.Errorf("Expected at least 2 commands, got %d", dl.Len())
	}
}

// TestButtonRenderParity tests that Draw and RenderToDisplayList produce similar output.
func TestButtonRenderParity(t *testing.T) {
	atlas, err := text.NewAtlas()
	if err != nil {
		t.Fatalf("Failed to create atlas: %v", err)
	}

	btn := widgets.NewButton("Test", 100, 30)
	btn.SetAtlas(atlas)

	// Direct rendering
	bufDirect, err := core.NewBuffer(200, 100)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	if err := btn.Draw(bufDirect, 10, 10); err != nil {
		t.Fatalf("Direct draw failed: %v", err)
	}

	// DisplayList rendering
	bufDisplayList, err := core.NewBuffer(200, 100)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	dl := displaylist.New()
	btn.RenderToDisplayList(dl, 10, 10)

	consumer := NewSoftwareConsumer(atlas)
	if err := consumer.Render(dl, bufDisplayList); err != nil {
		t.Fatalf("DisplayList render failed: %v", err)
	}

	// Compare a sample pixel in the center of the button
	// Both should have drawn the background color
	x, y := 60, 25
	directPixel := bufDirect.At(x, y)
	dlPixel := bufDisplayList.At(x, y)

	// Pixels should be non-transparent (button background was drawn)
	if directPixel.A == 0 {
		t.Error("Direct rendering produced transparent pixel")
	}
	if dlPixel.A == 0 {
		t.Error("DisplayList rendering produced transparent pixel")
	}

	// Note: Exact pixel-perfect comparison is complex due to anti-aliasing
	// This is a smoke test to verify both paths produce non-zero output
}

// TestTextInputRenderToDisplayList tests TextInput widget rendering.
func TestTextInputRenderToDisplayList(t *testing.T) {
	atlas, err := text.NewAtlas()
	if err != nil {
		t.Fatalf("Failed to create atlas: %v", err)
	}

	input := widgets.NewTextInput("Type here...", 200, 30)
	input.SetAtlas(atlas)
	input.SetText("Hello")

	dl := displaylist.New()
	input.RenderToDisplayList(dl, 10, 50)

	if dl.Len() == 0 {
		t.Error("Expected input to emit display list commands")
	}
}

// TestScrollContainerRenderToDisplayList tests ScrollContainer widget rendering.
func TestScrollContainerRenderToDisplayList(t *testing.T) {
	atlas, err := text.NewAtlas()
	if err != nil {
		t.Fatalf("Failed to create atlas: %v", err)
	}

	container := widgets.NewScrollContainer(300, 200)

	// Add some buttons as children
	for i := 0; i < 5; i++ {
		btn := widgets.NewButton("Item", 280, 30)
		btn.SetAtlas(atlas)
		container.AddChild(btn)
	}

	dl := displaylist.New()
	container.RenderToDisplayList(dl, 10, 10)

	if dl.Len() == 0 {
		t.Error("Expected container to emit display list commands")
	}

	// Container should emit: background, border, children, scrollbar
	// Each child button emits multiple commands
	if dl.Len() < 10 {
		t.Errorf("Expected many commands from container+children, got %d", dl.Len())
	}
}
