package pctwidget

import (
	"testing"

	"github.com/opd-ai/wain/internal/raster/primitives"
)

// ---------------------------------------------------------------------------
// Sizing tests
// ---------------------------------------------------------------------------

func TestPercentClamp(t *testing.T) {
	tests := []struct {
		input Percent
		want  Percent
	}{
		{-10, 0},
		{0, 0},
		{50, 50},
		{100, 100},
		{150, 100},
	}
	for _, tt := range tests {
		got := tt.input.Clamp()
		if got != tt.want {
			t.Errorf("Percent(%v).Clamp() = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestPercentToPixels(t *testing.T) {
	tests := []struct {
		pct    Percent
		parent int
		want   int
	}{
		{50, 800, 400},
		{100, 600, 600},
		{0, 600, 0},
		{25, 200, 50},
		{33, 300, 99},
	}
	for _, tt := range tests {
		got, err := tt.pct.ToPixels(tt.parent)
		if err != nil {
			t.Fatalf("Percent(%v).ToPixels(%d) unexpected error: %v", tt.pct, tt.parent, err)
		}
		if got != tt.want {
			t.Errorf("Percent(%v).ToPixels(%d) = %d, want %d", tt.pct, tt.parent, got, tt.want)
		}
	}
}

func TestPercentToPixelsInvalidParent(t *testing.T) {
	_, err := Percent(50).ToPixels(0)
	if err != ErrInvalidParentSize {
		t.Errorf("expected ErrInvalidParentSize, got %v", err)
	}
	_, err = Percent(50).ToPixels(-1)
	if err != ErrInvalidParentSize {
		t.Errorf("expected ErrInvalidParentSize, got %v", err)
	}
}

func TestValidatePercentage(t *testing.T) {
	if err := ValidatePercentage(50); err != nil {
		t.Errorf("ValidatePercentage(50) unexpected error: %v", err)
	}
	if err := ValidatePercentage(-1); err != ErrInvalidPercentage {
		t.Errorf("expected ErrInvalidPercentage for -1, got %v", err)
	}
	if err := ValidatePercentage(101); err != ErrInvalidPercentage {
		t.Errorf("expected ErrInvalidPercentage for 101, got %v", err)
	}
}

func TestPercentToPixelsConvenience(t *testing.T) {
	got, err := PercentToPixels(50.0, 800)
	if err != nil {
		t.Fatalf("PercentToPixels: %v", err)
	}
	if got != 400 {
		t.Errorf("PercentToPixels(50, 800) = %d, want 400", got)
	}
}

func TestSizeResolve(t *testing.T) {
	s := Size{Width: 50, Height: 25}
	w, h, err := s.Resolve(800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if w != 400 || h != 150 {
		t.Errorf("Size.Resolve(800,600) = (%d,%d), want (400,150)", w, h)
	}
}

// ---------------------------------------------------------------------------
// Style tests
// ---------------------------------------------------------------------------

func TestDefaultStyle(t *testing.T) {
	s := DefaultStyle()
	if s == nil {
		t.Fatal("DefaultStyle() returned nil")
	}
	if s.FontSize() <= 0 {
		t.Error("FontSize should be positive")
	}
	if s.Padding() < 0 {
		t.Error("Padding should be non-negative")
	}
	if s.Gap() < 0 {
		t.Error("Gap should be non-negative")
	}
	// Ensure Background has full opacity.
	if s.Background().A != 255 {
		t.Error("Background alpha should be 255")
	}
}

func TestRetroStyleImplementsStyle(t *testing.T) {
	var _ Style = (*RetroStyle)(nil)
}

func TestCustomStyle(t *testing.T) {
	custom := &RetroStyle{
		BgColor:      primitives.Color{R: 255, G: 0, B: 0, A: 255},
		FgColor:      primitives.Color{R: 0, G: 255, B: 0, A: 255},
		AccentColor:  primitives.Color{R: 0, G: 0, B: 255, A: 255},
		BorderColor:  primitives.Color{R: 128, G: 128, B: 128, A: 255},
		BaseFontSize: 16.0,
		BasePadding:  10,
		BaseGap:      8,
		BaseBorderW:  2,
	}
	if custom.Background() != custom.BgColor {
		t.Error("Background mismatch")
	}
	if custom.Foreground() != custom.FgColor {
		t.Error("Foreground mismatch")
	}
	if custom.Accent() != custom.AccentColor {
		t.Error("Accent mismatch")
	}
	if custom.Border() != custom.BorderColor {
		t.Error("Border mismatch")
	}
	if custom.FontSize() != 16.0 {
		t.Error("FontSize mismatch")
	}
	if custom.Padding() != 10 {
		t.Error("Padding mismatch")
	}
	if custom.Gap() != 8 {
		t.Error("Gap mismatch")
	}
	if custom.BorderWidth() != 2 {
		t.Error("BorderWidth mismatch")
	}
}

// ---------------------------------------------------------------------------
// Widget / Panel tests
// ---------------------------------------------------------------------------

func TestNewBaseWidget(t *testing.T) {
	bw := NewBaseWidget(50, 25)
	if bw.size.Width != 50 || bw.size.Height != 25 {
		t.Errorf("unexpected size: %+v", bw.size)
	}
	if !bw.Visible() {
		t.Error("new widget should be visible by default")
	}
}

func TestBaseWidgetClamps(t *testing.T) {
	bw := NewBaseWidget(150, -10)
	if bw.size.Width != 100 || bw.size.Height != 0 {
		t.Errorf("expected clamped size (100, 0), got (%v, %v)", bw.size.Width, bw.size.Height)
	}
}

func TestBaseWidgetManualPosition(t *testing.T) {
	bw := NewBaseWidget(50, 50)
	if bw.IsManuallyPositioned() {
		t.Error("should not be manually positioned by default")
	}
	bw.SetPosition(10, 20, 100, 200)
	if !bw.IsManuallyPositioned() {
		t.Error("should be manually positioned after SetPosition")
	}
	x, y, w, h := bw.ResolvedBounds()
	if x != 10 || y != 20 || w != 100 || h != 200 {
		t.Errorf("bounds = (%d,%d,%d,%d), want (10,20,100,200)", x, y, w, h)
	}
	bw.ClearPosition()
	if bw.IsManuallyPositioned() {
		t.Error("should not be manually positioned after ClearPosition")
	}
}

func TestBaseWidgetResolve(t *testing.T) {
	bw := NewBaseWidget(50, 25)
	if err := bw.Resolve(800, 600); err != nil {
		t.Fatal(err)
	}
	_, _, w, h := bw.ResolvedBounds()
	if w != 400 || h != 150 {
		t.Errorf("resolved = (%d,%d), want (400,150)", w, h)
	}
}

func TestBaseWidgetEffectiveStyle(t *testing.T) {
	bw := NewBaseWidget(50, 50)
	// Default should return DefaultStyle.
	s := bw.EffectiveStyle()
	if s == nil {
		t.Fatal("EffectiveStyle returned nil")
	}
	// Custom style.
	custom := &RetroStyle{BaseFontSize: 20}
	bw.SetStyle(custom)
	if bw.EffectiveStyle().FontSize() != 20 {
		t.Error("custom style not applied")
	}
	// Reset to default.
	bw.SetStyle(nil)
	if bw.EffectiveStyle().FontSize() != DefaultStyle().FontSize() {
		t.Error("resetting to nil should return default style")
	}
}

func TestNewPanel(t *testing.T) {
	p := NewPanel(100, 50)
	if p.size.Width != 100 || p.size.Height != 50 {
		t.Errorf("unexpected panel size: %+v", p.size)
	}
}

func TestPanelAddChild(t *testing.T) {
	parent := NewPanel(100, 100)
	child := NewPanel(50, 50)
	parent.AddChild(child)
	if len(parent.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(parent.Children()))
	}
	if parent.Children()[0] != child {
		t.Error("child mismatch")
	}
}

func TestPanelDraw(t *testing.T) {
	buf, err := primitives.NewBuffer(200, 200)
	if err != nil {
		t.Fatal(err)
	}
	p := NewPanel(100, 100)
	p.SetPosition(0, 0, 200, 200)
	if err := p.Draw(buf); err != nil {
		t.Fatal(err)
	}
}

func TestPanelDrawNilBuffer(t *testing.T) {
	p := NewPanel(100, 100)
	if err := p.Draw(nil); err != ErrNilBuffer {
		t.Errorf("expected ErrNilBuffer, got %v", err)
	}
}

func TestPanelDrawInvisible(t *testing.T) {
	buf, _ := primitives.NewBuffer(200, 200)
	p := NewPanel(100, 100)
	p.SetVisible(false)
	if err := p.Draw(buf); err != nil {
		t.Fatal("drawing invisible panel should succeed silently")
	}
}

// ---------------------------------------------------------------------------
// AutoLayout tests
// ---------------------------------------------------------------------------

func TestAutoLayoutColumn(t *testing.T) {
	parent := NewPanel(100, 100)
	c1 := NewPanel(100, 30)
	c2 := NewPanel(100, 30)
	parent.AddChild(c1)
	parent.AddChild(c2)

	parentW, parentH := 400, 400
	AutoLayout(parent.Children(), 0, 0, parentW, parentH, FlowColumn, AlignStart, DefaultStyle())

	style := DefaultStyle()
	pad := style.Padding()
	gap := style.Gap()

	// c1 should be at (pad, pad).
	x1, y1, _, h1 := c1.ResolvedBounds()
	if x1 != pad || y1 != pad {
		t.Errorf("c1 position = (%d,%d), want (%d,%d)", x1, y1, pad, pad)
	}
	// c2 should be below c1 + gap.
	x2, y2, _, _ := c2.ResolvedBounds()
	if x2 != pad {
		t.Errorf("c2.x = %d, want %d", x2, pad)
	}
	expectedY2 := pad + h1 + gap
	if y2 != expectedY2 {
		t.Errorf("c2.y = %d, want %d", y2, expectedY2)
	}
}

func TestAutoLayoutRow(t *testing.T) {
	c1 := NewPanel(30, 100)
	c2 := NewPanel(30, 100)

	AutoLayout([]*Panel{c1, c2}, 0, 0, 400, 400, FlowRow, AlignStart, DefaultStyle())

	style := DefaultStyle()
	pad := style.Padding()
	gap := style.Gap()

	x1, y1, w1, _ := c1.ResolvedBounds()
	if x1 != pad || y1 != pad {
		t.Errorf("c1 position = (%d,%d), want (%d,%d)", x1, y1, pad, pad)
	}
	x2, y2, _, _ := c2.ResolvedBounds()
	if y2 != pad {
		t.Errorf("c2.y = %d, want %d", y2, pad)
	}
	expectedX2 := pad + w1 + gap
	if x2 != expectedX2 {
		t.Errorf("c2.x = %d, want %d", x2, expectedX2)
	}
}

func TestAutoLayoutSkipsManual(t *testing.T) {
	c1 := NewPanel(30, 30)
	c1.SetPosition(100, 100, 50, 50) // manual override
	c2 := NewPanel(30, 30)

	AutoLayout([]*Panel{c1, c2}, 0, 0, 400, 400, FlowColumn, AlignStart, DefaultStyle())

	// c1 should keep its manual position.
	x1, y1, _, _ := c1.ResolvedBounds()
	if x1 != 100 || y1 != 100 {
		t.Errorf("manually positioned widget moved: (%d,%d)", x1, y1)
	}

	style := DefaultStyle()
	pad := style.Padding()
	// c2 should start at the beginning of the flow (pad, pad) since c1 didn't consume space.
	x2, y2, _, _ := c2.ResolvedBounds()
	if x2 != pad || y2 != pad {
		t.Errorf("c2 position = (%d,%d), want (%d,%d)", x2, y2, pad, pad)
	}
}

func TestAutoLayoutSkipsInvisible(t *testing.T) {
	c1 := NewPanel(50, 30)
	c1.SetVisible(false)
	c2 := NewPanel(50, 30)

	AutoLayout([]*Panel{c1, c2}, 0, 0, 400, 400, FlowColumn, AlignStart, DefaultStyle())

	style := DefaultStyle()
	pad := style.Padding()
	// c2 should start at (pad, pad) since c1 is invisible.
	x2, y2, _, _ := c2.ResolvedBounds()
	if x2 != pad || y2 != pad {
		t.Errorf("c2 position = (%d,%d), want (%d,%d)", x2, y2, pad, pad)
	}
}

func TestAutoLayoutNested(t *testing.T) {
	parent := NewPanel(100, 100)
	child := NewPanel(100, 50)
	grandchild := NewPanel(100, 50)
	child.AddChild(grandchild)
	parent.AddChild(child)

	AutoLayout(parent.Children(), 0, 0, 400, 400, FlowColumn, AlignStart, DefaultStyle())

	// Grandchild should be positioned inside the child's resolved bounds.
	gx, gy, _, _ := grandchild.ResolvedBounds()
	cx, cy, _, _ := child.ResolvedBounds()
	pad := DefaultStyle().Padding()
	if gx != cx+pad || gy != cy+pad {
		t.Errorf("grandchild position = (%d,%d), expected (%d,%d)", gx, gy, cx+pad, cy+pad)
	}
}

func TestAutoLayoutNilStyle(t *testing.T) {
	c1 := NewPanel(50, 30)
	// Should not panic — AutoLayout falls back to DefaultStyle.
	AutoLayout([]*Panel{c1}, 0, 0, 400, 400, FlowColumn, AlignStart, nil)
	x, y, _, _ := c1.ResolvedBounds()
	pad := DefaultStyle().Padding()
	if x != pad || y != pad {
		t.Errorf("position = (%d,%d), want (%d,%d)", x, y, pad, pad)
	}
}

func TestAutoLayoutPercentageConsistency(t *testing.T) {
	// Verify that resizing the parent correctly updates child pixel sizes.
	c := NewPanel(50, 50)

	AutoLayout([]*Panel{c}, 0, 0, 800, 600, FlowColumn, AlignStart, DefaultStyle())
	_, _, w1, h1 := c.ResolvedBounds()

	AutoLayout([]*Panel{c}, 0, 0, 400, 300, FlowColumn, AlignStart, DefaultStyle())
	_, _, w2, h2 := c.ResolvedBounds()

	// At 50% sizing, halving the parent should halve the child.
	// Account for padding: content width = parent - 2*pad
	pad := DefaultStyle().Padding()
	cw1 := 800 - 2*pad
	cw2 := 400 - 2*pad
	expectedW1, _ := PercentToPixels(50, cw1)
	expectedW2, _ := PercentToPixels(50, cw2)
	ch1 := 600 - 2*pad
	ch2 := 300 - 2*pad
	expectedH1, _ := PercentToPixels(50, ch1)
	expectedH2, _ := PercentToPixels(50, ch2)

	if w1 != expectedW1 || h1 != expectedH1 {
		t.Errorf("800x600: got (%d,%d), want (%d,%d)", w1, h1, expectedW1, expectedH1)
	}
	if w2 != expectedW2 || h2 != expectedH2 {
		t.Errorf("400x300: got (%d,%d), want (%d,%d)", w2, h2, expectedW2, expectedH2)
	}
}

func TestAutoLayoutDefaultFlowDirection(t *testing.T) {
	c1 := NewPanel(100, 30)
	c2 := NewPanel(100, 30)

	// Use an invalid FlowDirection value; should behave like FlowColumn.
	AutoLayout([]*Panel{c1, c2}, 0, 0, 400, 400, FlowDirection(99), AlignStart, DefaultStyle())

	style := DefaultStyle()
	pad := style.Padding()
	gap := style.Gap()

	x1, y1, _, h1 := c1.ResolvedBounds()
	if x1 != pad || y1 != pad {
		t.Errorf("c1 position = (%d,%d), want (%d,%d)", x1, y1, pad, pad)
	}
	x2, y2, _, _ := c2.ResolvedBounds()
	expectedY2 := pad + h1 + gap
	if x2 != pad || y2 != expectedY2 {
		t.Errorf("c2 position = (%d,%d), want (%d,%d)", x2, y2, pad, expectedY2)
	}
}

func TestDefaultStyleSingleton(t *testing.T) {
	s1 := DefaultStyle()
	s2 := DefaultStyle()
	if s1 != s2 {
		t.Error("DefaultStyle should return the same singleton instance")
	}
}

func TestPanelDrawBorderClamped(t *testing.T) {
	buf, err := primitives.NewBuffer(200, 200)
	if err != nil {
		t.Fatal(err)
	}
	// Create a tiny panel where border width exceeds half the dimensions.
	p := NewPanel(100, 100)
	p.SetPosition(0, 0, 4, 4)
	bigBorder := &RetroStyle{
		BgColor:     primitives.Color{R: 30, G: 30, B: 46, A: 255},
		BorderColor: primitives.Color{R: 255, G: 0, B: 0, A: 255},
		BaseBorderW: 10, // much larger than the 4x4 panel
	}
	p.SetStyle(bigBorder)
	// Should not panic or draw outside bounds.
	if err := p.Draw(buf); err != nil {
		t.Fatal(err)
	}
}

func TestAutoLayoutZeroParent(t *testing.T) {
	c := NewPanel(50, 50)
	// Parent area smaller than 2*padding should not panic.
	AutoLayout([]*Panel{c}, 0, 0, 2, 2, FlowColumn, AlignStart, DefaultStyle())
	_, _, w, h := c.ResolvedBounds()
	if w != 0 || h != 0 {
		t.Errorf("expected zero dimensions for tiny parent, got (%d,%d)", w, h)
	}
}

func TestPanelSetAlign(t *testing.T) {
	p := NewPanel(100, 100)
	if p.GetAlign() != AlignStart {
		t.Errorf("default align = %v, want AlignStart", p.GetAlign())
	}
	for _, a := range []Align{AlignStart, AlignCenter, AlignEnd, AlignStretch} {
		p.SetAlign(a)
		if got := p.GetAlign(); got != a {
			t.Errorf("SetAlign(%v): GetAlign() = %v", a, got)
		}
	}
}

func TestAutoLayoutAlignCenterRow(t *testing.T) {
	// Container: 400 wide, 200 tall (no padding/gap via custom style)
	noStyle := &RetroStyle{}
	// Child: 100% wide, 50% tall → 400×100
	child := NewPanel(100, 50)
	AutoLayout([]*Panel{child}, 0, 0, 400, 200, FlowRow, AlignCenter, noStyle)
	_, y, _, h := child.ResolvedBounds()
	// cross axis is vertical; center of 200 with child height 100 → y = 50
	if h != 100 {
		t.Fatalf("child height = %d, want 100", h)
	}
	if y != 50 {
		t.Errorf("AlignCenter row: child.y = %d, want 50", y)
	}
}

func TestAutoLayoutAlignEndRow(t *testing.T) {
	noStyle := &RetroStyle{}
	child := NewPanel(100, 50)
	AutoLayout([]*Panel{child}, 0, 0, 400, 200, FlowRow, AlignEnd, noStyle)
	_, y, _, h := child.ResolvedBounds()
	// end of 200 with child height 100 → y = 100
	if h != 100 {
		t.Fatalf("child height = %d, want 100", h)
	}
	if y != 100 {
		t.Errorf("AlignEnd row: child.y = %d, want 100", y)
	}
}

func TestAutoLayoutAlignStretchRow(t *testing.T) {
	noStyle := &RetroStyle{}
	child := NewPanel(100, 50)
	AutoLayout([]*Panel{child}, 0, 0, 400, 200, FlowRow, AlignStretch, noStyle)
	_, y, _, h := child.ResolvedBounds()
	if h != 200 {
		t.Errorf("AlignStretch row: child height = %d, want 200", h)
	}
	if y != 0 {
		t.Errorf("AlignStretch row: child.y = %d, want 0", y)
	}
}

func TestAutoLayoutAlignCenterColumn(t *testing.T) {
	noStyle := &RetroStyle{}
	// Child: 50% wide, 100% tall → 200×400 in a 400×400 container
	child := NewPanel(50, 100)
	AutoLayout([]*Panel{child}, 0, 0, 400, 400, FlowColumn, AlignCenter, noStyle)
	x, _, w, _ := child.ResolvedBounds()
	if w != 200 {
		t.Fatalf("child width = %d, want 200", w)
	}
	if x != 100 {
		t.Errorf("AlignCenter column: child.x = %d, want 100", x)
	}
}

func TestAutoLayoutAlignEndColumn(t *testing.T) {
	noStyle := &RetroStyle{}
	child := NewPanel(50, 100)
	AutoLayout([]*Panel{child}, 0, 0, 400, 400, FlowColumn, AlignEnd, noStyle)
	x, _, w, _ := child.ResolvedBounds()
	if w != 200 {
		t.Fatalf("child width = %d, want 200", w)
	}
	if x != 200 {
		t.Errorf("AlignEnd column: child.x = %d, want 200", x)
	}
}

func TestAutoLayoutAlignStretchColumn(t *testing.T) {
	noStyle := &RetroStyle{}
	child := NewPanel(50, 100)
	AutoLayout([]*Panel{child}, 0, 0, 400, 400, FlowColumn, AlignStretch, noStyle)
	x, _, w, _ := child.ResolvedBounds()
	if w != 400 {
		t.Errorf("AlignStretch column: child width = %d, want 400", w)
	}
	if x != 0 {
		t.Errorf("AlignStretch column: child.x = %d, want 0", x)
	}
}

func TestAutoLayoutPanelFlowDirectionPropagation(t *testing.T) {
	// Verify that when recursing, the child's own flowDirection is used,
	// not the parent's direction passed to the top-level AutoLayout call.
	noStyle := &RetroStyle{}
	parent := NewPanel(100, 100)
	// Child uses FlowRow even though parent uses FlowColumn.
	child := NewPanel(100, 100)
	child.SetFlowDirection(FlowRow)
	gc1 := NewPanel(50, 100)
	gc2 := NewPanel(50, 100)
	child.AddChild(gc1)
	child.AddChild(gc2)
	parent.AddChild(child)

	AutoLayout(parent.Children(), 0, 0, 400, 400, FlowColumn, AlignStart, noStyle)

	// gc1 and gc2 should be laid out side-by-side (FlowRow), not top-to-bottom.
	x1, _, w1, _ := gc1.ResolvedBounds()
	x2, _, _, _ := gc2.ResolvedBounds()
	if x1 != 0 {
		t.Errorf("gc1.x = %d, want 0", x1)
	}
	if x2 != w1 {
		t.Errorf("gc2.x = %d, want %d (gc1.width)", x2, w1)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkAutoLayout(b *testing.B) {
	panels := make([]*Panel, 20)
	for i := range panels {
		panels[i] = NewPanel(100, 5)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AutoLayout(panels, 0, 0, 800, 600, FlowColumn, AlignStart, DefaultStyle())
	}
}

func BenchmarkPanelDraw(b *testing.B) {
	buf, _ := primitives.NewBuffer(800, 600)
	p := NewPanel(100, 100)
	p.SetPosition(0, 0, 800, 600)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Draw(buf)
	}
}
