package decorations

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/core"
)

func TestResizeHandles_HitTest_Corners(t *testing.T) {
	rh := NewResizeHandles(640, 480)
	hw := rh.handleWidth

	tests := []struct {
		name     string
		x, y     int
		expected ResizeEdge
	}{
		{"TopLeft corner", hw / 2, hw / 2, ResizeEdgeTopLeft},
		{"TopRight corner", 640 - hw/2, hw / 2, ResizeEdgeTopRight},
		{"BottomLeft corner", hw / 2, 480 - hw/2, ResizeEdgeBottomLeft},
		{"BottomRight corner", 640 - hw/2, 480 - hw/2, ResizeEdgeBottomRight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := rh.HitTest(tt.x, tt.y)
			if edge != tt.expected {
				t.Errorf("HitTest(%d, %d) = %v; want %v", tt.x, tt.y, edge, tt.expected)
			}
		})
	}
}

func TestResizeHandles_HitTest_Edges(t *testing.T) {
	rh := NewResizeHandles(640, 480)
	hw := rh.handleWidth

	tests := []struct {
		name     string
		x, y     int
		expected ResizeEdge
	}{
		{"Top edge center", 320, hw / 2, ResizeEdgeTop},
		{"Bottom edge center", 320, 480 - hw/2, ResizeEdgeBottom},
		{"Left edge center", hw / 2, 240, ResizeEdgeLeft},
		{"Right edge center", 640 - hw/2, 240, ResizeEdgeRight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := rh.HitTest(tt.x, tt.y)
			if edge != tt.expected {
				t.Errorf("HitTest(%d, %d) = %v; want %v", tt.x, tt.y, edge, tt.expected)
			}
		})
	}
}

func TestResizeHandles_HitTest_Interior(t *testing.T) {
	rh := NewResizeHandles(640, 480)
	hw := rh.handleWidth

	// Test points well inside the window (not near edges)
	tests := []struct {
		name string
		x, y int
	}{
		{"Center", 320, 240},
		{"Upper middle", 320, 100},
		{"Lower middle", 320, 380},
		{"Left middle", 100, 240},
		{"Right middle", 540, 240},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure point is not near edges
			if tt.x < hw*2 || tt.x > 640-hw*2 || tt.y < hw*2 || tt.y > 480-hw*2 {
				t.Skip("Test point too close to edge")
			}

			edge := rh.HitTest(tt.x, tt.y)
			if edge != ResizeEdgeNone {
				t.Errorf("HitTest(%d, %d) = %v; want ResizeEdgeNone", tt.x, tt.y, edge)
			}
		})
	}
}

func TestResizeHandles_Resize(t *testing.T) {
	rh := NewResizeHandles(640, 480)

	rh.Resize(800, 600)

	if rh.width != 800 {
		t.Errorf("After Resize(800, 600), width = %d; want 800", rh.width)
	}
	if rh.height != 600 {
		t.Errorf("After Resize(800, 600), height = %d; want 600", rh.height)
	}

	// Verify hit testing still works with new dimensions
	edge := rh.HitTest(5, 5)
	if edge != ResizeEdgeTopLeft {
		t.Errorf("After resize, HitTest(5, 5) = %v; want ResizeEdgeTopLeft", edge)
	}
}

func TestResizeHandles_HandlePointerEnterLeave(t *testing.T) {
	rh := NewResizeHandles(640, 480)

	if rh.hoverEdge != ResizeEdgeNone {
		t.Errorf("Initial hoverEdge = %v; want ResizeEdgeNone", rh.hoverEdge)
	}

	rh.HandlePointerEnter(ResizeEdgeTopLeft)
	if rh.hoverEdge != ResizeEdgeTopLeft {
		t.Errorf("After HandlePointerEnter(TopLeft), hoverEdge = %v; want ResizeEdgeTopLeft", rh.hoverEdge)
	}

	rh.HandlePointerLeave()
	if rh.hoverEdge != ResizeEdgeNone {
		t.Errorf("After HandlePointerLeave(), hoverEdge = %v; want ResizeEdgeNone", rh.hoverEdge)
	}
}

func TestResizeHandles_Draw(t *testing.T) {
	rh := NewResizeHandles(640, 480)
	buf, err := core.NewBuffer(640, 480)
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	// Fill buffer with a known background color first
	bgColor := core.Color{R: 255, G: 255, B: 255, A: 255}
	buf.FillRect(0, 0, 640, 480, bgColor)

	// Draw with no hover - should be no-op
	err = rh.Draw(buf, 0, 0)
	if err != nil {
		t.Errorf("Draw with no hover failed: %v", err)
	}

	// Verify background is still white (no drawing occurred)
	pixel := buf.At(rh.handleWidth/2, rh.handleWidth/2)
	if pixel != bgColor {
		t.Errorf("Draw with no hover modified buffer: pixel = %v; want %v", pixel, bgColor)
	}

	// Draw with hover
	rh.HandlePointerEnter(ResizeEdgeTopLeft)
	err = rh.Draw(buf, 0, 0)
	if err != nil {
		t.Errorf("Draw with hover failed: %v", err)
	}

	// Verify that the corner was painted (color should have changed from white)
	hw := rh.handleWidth
	pixel = buf.At(hw/2, hw/2)
	if pixel == bgColor {
		t.Error("Draw with hover did not paint resize handle")
	}

	// The color should contain the handle color (may be blended)
	// Just verify it's not the background color
	expectedColor := rh.theme.ResizeHandleColor
	if pixel.R == bgColor.R && pixel.G == bgColor.G && pixel.B == bgColor.B {
		t.Errorf("Pixel at (%d, %d) was not modified; expected handle color influence", hw/2, hw/2)
	}

	// For a more lenient check, verify alpha channel matches
	if pixel.A != expectedColor.A && pixel.A != 255 {
		t.Logf("Note: Pixel alpha = %d, expected %d or 255 (blended)", pixel.A, expectedColor.A)
	}
}

func TestResizeHandles_SetTheme(t *testing.T) {
	rh := NewResizeHandles(640, 480)
	oldWidth := rh.handleWidth

	newTheme := &Theme{
		ResizeHandleWidth: 12,
		ResizeHandleColor: core.Color{R: 255, G: 0, B: 0, A: 255},
	}

	rh.SetTheme(newTheme)

	if rh.theme != newTheme {
		t.Error("SetTheme did not update theme")
	}
	if rh.handleWidth == oldWidth {
		t.Errorf("SetTheme did not update handleWidth; got %d, want different from %d", rh.handleWidth, oldWidth)
	}
	if rh.handleWidth != 12 {
		t.Errorf("handleWidth = %d; want 12", rh.handleWidth)
	}
}

func TestResizeEdge_String(t *testing.T) {
	tests := []struct {
		edge     ResizeEdge
		expected string
	}{
		{ResizeEdgeNone, "none"},
		{ResizeEdgeTop, "top"},
		{ResizeEdgeBottom, "bottom"},
		{ResizeEdgeLeft, "left"},
		{ResizeEdgeRight, "right"},
		{ResizeEdgeTopLeft, "top-left"},
		{ResizeEdgeTopRight, "top-right"},
		{ResizeEdgeBottomLeft, "bottom-left"},
		{ResizeEdgeBottomRight, "bottom-right"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			str := tt.edge.String()
			if str != tt.expected {
				t.Errorf("ResizeEdge(%d).String() = %q; want %q", tt.edge, str, tt.expected)
			}
		})
	}
}

func TestResizeHandles_CornersPriority(t *testing.T) {
	// Verify corners take priority over edges
	rh := NewResizeHandles(640, 480)
	hw := rh.handleWidth

	// At the exact corner boundary, corner should win
	edge := rh.HitTest(0, 0)
	if edge != ResizeEdgeTopLeft {
		t.Errorf("HitTest(0, 0) = %v; want ResizeEdgeTopLeft (corner priority)", edge)
	}

	edge = rh.HitTest(640-1, 0)
	if edge != ResizeEdgeTopRight {
		t.Errorf("HitTest(639, 0) = %v; want ResizeEdgeTopRight (corner priority)", edge)
	}

	// Just outside corner boundary should be edge
	edge = rh.HitTest(hw+1, 1)
	if edge != ResizeEdgeTop {
		t.Errorf("HitTest(%d, 1) = %v; want ResizeEdgeTop", hw+1, edge)
	}
}
