package widgets

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/primitives"
)

// TestScrollVelocity verifies scroll offset updates with mouse wheel events.
func TestScrollVelocity(t *testing.T) {
	tests := []struct {
		name          string
		width         int
		height        int
		contentHeight int
		scrollDeltas  []int
		wantOffset    int
	}{
		{
			name:          "scroll down within bounds",
			width:         400,
			height:        200,
			contentHeight: 1000,
			scrollDeltas:  []int{5, 5, 5},
			wantOffset:    300,
		},
		{
			name:          "scroll up from middle",
			width:         400,
			height:        200,
			contentHeight: 1000,
			scrollDeltas:  []int{20, -10},
			wantOffset:    200,
		},
		{
			name:          "scroll past bottom clamps to max",
			width:         400,
			height:        200,
			contentHeight: 600,
			scrollDeltas:  []int{100},
			wantOffset:    400,
		},
		{
			name:          "scroll past top clamps to zero",
			width:         400,
			height:        200,
			contentHeight: 1000,
			scrollDeltas:  []int{10, -20},
			wantOffset:    0,
		},
		{
			name:          "no scroll when content fits",
			width:         400,
			height:        500,
			contentHeight: 400,
			scrollDeltas:  []int{5, 5},
			wantOffset:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewScrollContainer(tt.width, tt.height)

			// Add mock children to simulate content height
			mockChild := &mockWidget{width: tt.width, height: tt.contentHeight}
			sc.AddChild(mockChild)

			// Simulate scroll events
			offset := 0
			for _, delta := range tt.scrollDeltas {
				offset += delta * 20
				sc.SetScrollOffset(offset)
				offset = sc.ScrollOffset()
			}

			if offset != tt.wantOffset {
				t.Errorf("ScrollOffset() = %d, want %d", offset, tt.wantOffset)
			}
		})
	}
}

// TestScrollBounds verifies scroll bounds prevent over-scrolling.
func TestScrollBounds(t *testing.T) {
	sc := NewScrollContainer(400, 200)

	// Add 2000px tall content (10 children x 200px each)
	for i := 0; i < 10; i++ {
		sc.AddChild(&mockWidget{width: 380, height: 200})
	}

	// Attempt to scroll beyond bottom
	sc.SetScrollOffset(5000)
	if got := sc.ScrollOffset(); got != 1800 {
		t.Errorf("ScrollOffset() beyond max = %d, want 1800", got)
	}

	// Attempt to scroll beyond top
	sc.SetScrollOffset(-100)
	if got := sc.ScrollOffset(); got != 0 {
		t.Errorf("ScrollOffset() below zero = %d, want 0", got)
	}
}

// TestScrollRendering verifies only visible children are rendered.
func TestScrollRendering(t *testing.T) {
	sc := NewScrollContainer(400, 200)

	// Add children
	for i := 0; i < 5; i++ {
		sc.AddChild(&mockWidget{width: 380, height: 100})
	}

	// Create buffer for rendering
	buf, err := primitives.NewBuffer(400, 200)
	if err != nil {
		t.Fatalf("NewBuffer() error = %v", err)
	}

	// Scroll to middle
	sc.SetScrollOffset(150)

	// Render (should clip to visible area)
	if err := sc.Draw(buf, 0, 0); err != nil {
		t.Errorf("Draw() error = %v", err)
	}
}

// TestScrollWithNoContent verifies behavior with empty container.
func TestScrollWithNoContent(t *testing.T) {
	sc := NewScrollContainer(400, 200)

	// Attempt to scroll with no content
	sc.SetScrollOffset(100)

	if got := sc.ScrollOffset(); got != 0 {
		t.Errorf("ScrollOffset() with no content = %d, want 0", got)
	}
}

// mockWidget is a test helper that implements the Widget interface.
type mockWidget struct {
	width  int
	height int
}

func (m *mockWidget) Bounds() (int, int) {
	return m.width, m.height
}

func (m *mockWidget) HandlePointerEnter() {}

func (m *mockWidget) HandlePointerLeave() {}

func (m *mockWidget) HandlePointerDown(button uint32) {}

func (m *mockWidget) HandlePointerUp(button uint32) {}

func (m *mockWidget) Draw(buf *primitives.Buffer, x, y int) error {
	return nil
}
