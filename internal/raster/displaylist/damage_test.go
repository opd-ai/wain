package displaylist

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/core"
)

func TestNewDamageTracker(t *testing.T) {
	dt := NewDamageTracker()
	if dt == nil {
		t.Fatal("NewDamageTracker() returned nil")
	}
	if !dt.IsEmpty() {
		t.Error("Expected new damage tracker to be empty")
	}
}

func TestAddRect(t *testing.T) {
	dt := NewDamageTracker()
	dt.AddRect(10, 20, 100, 50)

	regions := dt.Regions()
	if len(regions) != 1 {
		t.Fatalf("Expected 1 region, got %d", len(regions))
	}

	r := regions[0]
	if r.X != 10 || r.Y != 20 || r.Width != 100 || r.Height != 50 {
		t.Errorf("Expected rect (10,20,100,50), got (%d,%d,%d,%d)",
			r.X, r.Y, r.Width, r.Height)
	}
}

func TestAddMultipleRects(t *testing.T) {
	dt := NewDamageTracker()
	dt.AddRect(0, 0, 50, 50)
	dt.AddRect(100, 100, 200, 150)
	dt.AddRect(300, 300, 100, 100)

	if dt.IsEmpty() {
		t.Error("Expected damage tracker to not be empty")
	}

	regions := dt.Regions()
	if len(regions) != 3 {
		t.Fatalf("Expected 3 regions, got %d", len(regions))
	}
}

func TestClear(t *testing.T) {
	dt := NewDamageTracker()
	dt.AddRect(10, 20, 100, 50)
	dt.AddRect(50, 60, 80, 40)

	if dt.IsEmpty() {
		t.Error("Expected damage tracker to have regions")
	}

	dt.Clear()

	if !dt.IsEmpty() {
		t.Error("Expected damage tracker to be empty after Clear()")
	}

	if len(dt.Regions()) != 0 {
		t.Errorf("Expected 0 regions after clear, got %d", len(dt.Regions()))
	}
}

func TestBounds(t *testing.T) {
	dt := NewDamageTracker()
	dt.AddRect(10, 20, 50, 30)   // (10,20) to (60,50)
	dt.AddRect(100, 100, 50, 50) // (100,100) to (150,150)
	dt.AddRect(40, 40, 30, 30)   // (40,40) to (70,70)

	bounds := dt.Bounds()

	// Should encompass all three rects: (10,20) to (150,150)
	if bounds.X != 10 || bounds.Y != 20 {
		t.Errorf("Expected bounds origin (10,20), got (%d,%d)", bounds.X, bounds.Y)
	}

	expectedWidth := 150 - 10
	expectedHeight := 150 - 20

	if bounds.Width != expectedWidth || bounds.Height != expectedHeight {
		t.Errorf("Expected bounds size (%d,%d), got (%d,%d)",
			expectedWidth, expectedHeight, bounds.Width, bounds.Height)
	}
}

func TestBoundsEmpty(t *testing.T) {
	dt := NewDamageTracker()
	bounds := dt.Bounds()

	if bounds.X != 0 || bounds.Y != 0 || bounds.Width != 0 || bounds.Height != 0 {
		t.Errorf("Expected zero bounds for empty tracker, got (%d,%d,%d,%d)",
			bounds.X, bounds.Y, bounds.Width, bounds.Height)
	}
}

func TestCoalesce(t *testing.T) {
	dt := NewDamageTracker()
	// Add two overlapping rects
	dt.AddRect(0, 0, 50, 50)
	dt.AddRect(40, 40, 50, 50) // Overlaps with first

	dt.Coalesce(0)

	regions := dt.Regions()
	if len(regions) != 1 {
		t.Fatalf("Expected 1 region after coalesce, got %d", len(regions))
	}

	// Merged rect should be (0,0) to (90,90)
	r := regions[0]
	if r.X != 0 || r.Y != 0 || r.Width != 90 || r.Height != 90 {
		t.Errorf("Expected merged rect (0,0,90,90), got (%d,%d,%d,%d)",
			r.X, r.Y, r.Width, r.Height)
	}
}

func TestCoalesceWithMargin(t *testing.T) {
	dt := NewDamageTracker()
	// Add two rects that don't overlap but are close
	dt.AddRect(0, 0, 50, 50)
	dt.AddRect(55, 0, 50, 50) // 5 pixels apart

	// Without margin, should not merge
	dt.Coalesce(0)
	if len(dt.Regions()) != 2 {
		t.Errorf("Expected 2 regions with margin=0, got %d", len(dt.Regions()))
	}

	// Reset and try with margin
	dt.Clear()
	dt.AddRect(0, 0, 50, 50)
	dt.AddRect(55, 0, 50, 50)

	// With margin=10, should merge
	dt.Coalesce(10)
	if len(dt.Regions()) != 1 {
		t.Errorf("Expected 1 region with margin=10, got %d", len(dt.Regions()))
	}
}

func TestCoalesceNoRegions(t *testing.T) {
	dt := NewDamageTracker()
	dt.Coalesce(0) // Should not panic
	if !dt.IsEmpty() {
		t.Error("Expected empty tracker to remain empty after coalesce")
	}
}

func TestCoalesceSingleRegion(t *testing.T) {
	dt := NewDamageTracker()
	dt.AddRect(10, 20, 100, 50)
	dt.Coalesce(0)

	regions := dt.Regions()
	if len(regions) != 1 {
		t.Fatalf("Expected 1 region, got %d", len(regions))
	}

	// Should be unchanged
	r := regions[0]
	if r.X != 10 || r.Y != 20 || r.Width != 100 || r.Height != 50 {
		t.Errorf("Expected unchanged rect, got (%d,%d,%d,%d)",
			r.X, r.Y, r.Width, r.Height)
	}
}

func TestComputeDamageForFillRect(t *testing.T) {
	cmd := DrawCommand{
		Type: CmdFillRect,
		Data: FillRectData{X: 10, Y: 20, Width: 100, Height: 50, Color: core.Color{}},
	}

	rect := ComputeDamageForCommand(cmd)

	if rect.X != 10 || rect.Y != 20 || rect.Width != 100 || rect.Height != 50 {
		t.Errorf("Expected rect (10,20,100,50), got (%d,%d,%d,%d)",
			rect.X, rect.Y, rect.Width, rect.Height)
	}
}

func TestComputeDamageForDrawLine(t *testing.T) {
	cmd := DrawCommand{
		Type: CmdDrawLine,
		Data: DrawLineData{X0: 10, Y0: 20, X1: 110, Y1: 120, Width: 4, Color: core.Color{}},
	}

	rect := ComputeDamageForCommand(cmd)

	// Line from (10,20) to (110,120) with width 4
	// Should be expanded by width/2 = 2 on each side
	// Expected: (8,18,104,104)
	if rect.X != 8 || rect.Y != 18 {
		t.Errorf("Expected origin (8,18), got (%d,%d)", rect.X, rect.Y)
	}

	if rect.Width != 104 || rect.Height != 104 {
		t.Errorf("Expected size (104,104), got (%d,%d)", rect.Width, rect.Height)
	}
}

func TestComputeDamageForBoxShadow(t *testing.T) {
	cmd := DrawCommand{
		Type: CmdBoxShadow,
		Data: BoxShadowData{
			X: 10, Y: 20, Width: 100, Height: 50,
			BlurRadius: 5, SpreadRadius: 2, Color: core.Color{},
		},
	}

	rect := ComputeDamageForCommand(cmd)

	// Shadow extends by blur + spread = 7 pixels in all directions
	// Expected: (3,13,114,64)
	if rect.X != 3 || rect.Y != 13 {
		t.Errorf("Expected origin (3,13), got (%d,%d)", rect.X, rect.Y)
	}

	if rect.Width != 114 || rect.Height != 64 {
		t.Errorf("Expected size (114,64), got (%d,%d)", rect.Width, rect.Height)
	}
}

func TestComputeDamageForDrawText(t *testing.T) {
	cmd := DrawCommand{
		Type: CmdDrawText,
		Data: DrawTextData{
			Text: "Hello", X: 10, Y: 50, FontSize: 16, Color: core.Color{}, AtlasID: 0,
		},
	}

	rect := ComputeDamageForCommand(cmd)

	// Text bounds are estimated
	// Width: len("Hello") * 16/2 = 5 * 8 = 40
	// Height: 16 + 16/4 = 20
	// Y origin adjusted: 50 - 20 = 30
	if rect.X != 10 {
		t.Errorf("Expected X=10, got %d", rect.X)
	}

	if rect.Y != 30 {
		t.Errorf("Expected Y=30, got %d", rect.Y)
	}

	if rect.Width != 40 {
		t.Errorf("Expected Width=40, got %d", rect.Width)
	}

	if rect.Height != 20 {
		t.Errorf("Expected Height=20, got %d", rect.Height)
	}
}

func TestFilterCommandsByDamage(t *testing.T) {
	dl := New()
	red := core.Color{R: 255, G: 0, B: 0, A: 255}
	green := core.Color{R: 0, G: 255, B: 0, A: 255}
	blue := core.Color{R: 0, G: 0, B: 255, A: 255}

	// Add commands at different locations
	dl.AddFillRect(0, 0, 50, 50, red)       // Top-left
	dl.AddFillRect(200, 200, 50, 50, green) // Bottom-right
	dl.AddFillRect(400, 400, 50, 50, blue)  // Far bottom-right

	// Damage only the top-left area
	damage := []Rect{{X: 0, Y: 0, Width: 100, Height: 100}}

	filtered := FilterCommandsByDamage(dl.Commands(), damage)

	// Should only include the first rect
	if len(filtered) != 1 {
		t.Fatalf("Expected 1 filtered command, got %d", len(filtered))
	}

	if filtered[0].Type != CmdFillRect {
		t.Errorf("Expected CmdFillRect, got %v", filtered[0].Type)
	}

	data := filtered[0].Data.(FillRectData)
	if data.Color != red {
		t.Errorf("Expected red rect, got different color")
	}
}

func TestFilterCommandsByMultipleDamageRegions(t *testing.T) {
	dl := New()
	red := core.Color{R: 255, G: 0, B: 0, A: 255}
	green := core.Color{R: 0, G: 255, B: 0, A: 255}
	blue := core.Color{R: 0, G: 0, B: 255, A: 255}

	dl.AddFillRect(0, 0, 50, 50, red)
	dl.AddFillRect(200, 200, 50, 50, green)
	dl.AddFillRect(400, 400, 50, 50, blue)

	// Damage top-left and bottom-right, but not far bottom-right
	damage := []Rect{
		{X: 0, Y: 0, Width: 100, Height: 100},
		{X: 180, Y: 180, Width: 100, Height: 100},
	}

	filtered := FilterCommandsByDamage(dl.Commands(), damage)

	// Should include first and second rect, but not third
	if len(filtered) != 2 {
		t.Fatalf("Expected 2 filtered commands, got %d", len(filtered))
	}
}

func TestFilterCommandsByDamageEmpty(t *testing.T) {
	dl := New()
	dl.AddFillRect(0, 0, 100, 100, core.Color{})

	// No damage regions
	filtered := FilterCommandsByDamage(dl.Commands(), []Rect{})

	// Should return nil/empty since there's no damage
	if len(filtered) != 0 {
		t.Errorf("Expected 0 filtered commands with no damage, got %d", len(filtered))
	}
}

func TestRectsIntersect(t *testing.T) {
	tests := []struct {
		name     string
		a        Rect
		b        Rect
		expected bool
	}{
		{
			name:     "overlapping",
			a:        Rect{X: 0, Y: 0, Width: 50, Height: 50},
			b:        Rect{X: 25, Y: 25, Width: 50, Height: 50},
			expected: true,
		},
		{
			name:     "touching edges",
			a:        Rect{X: 0, Y: 0, Width: 50, Height: 50},
			b:        Rect{X: 50, Y: 0, Width: 50, Height: 50},
			expected: false, // Edges touching but not overlapping
		},
		{
			name:     "completely separate",
			a:        Rect{X: 0, Y: 0, Width: 50, Height: 50},
			b:        Rect{X: 100, Y: 100, Width: 50, Height: 50},
			expected: false,
		},
		{
			name:     "one contains other",
			a:        Rect{X: 0, Y: 0, Width: 100, Height: 100},
			b:        Rect{X: 25, Y: 25, Width: 50, Height: 50},
			expected: true,
		},
		{
			name:     "identical",
			a:        Rect{X: 10, Y: 20, Width: 30, Height: 40},
			b:        Rect{X: 10, Y: 20, Width: 30, Height: 40},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rectsIntersect(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("rectsIntersect(%+v, %+v) = %v, expected %v",
					tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMergeRects(t *testing.T) {
	a := Rect{X: 10, Y: 20, Width: 50, Height: 30}
	b := Rect{X: 40, Y: 40, Width: 60, Height: 40}

	merged := mergeRects(a, b)

	// Should encompass both: (10,20) to (100,80)
	if merged.X != 10 || merged.Y != 20 {
		t.Errorf("Expected merged origin (10,20), got (%d,%d)", merged.X, merged.Y)
	}

	if merged.Width != 90 || merged.Height != 60 {
		t.Errorf("Expected merged size (90,60), got (%d,%d)", merged.Width, merged.Height)
	}
}
