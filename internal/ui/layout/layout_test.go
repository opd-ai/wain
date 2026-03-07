package layout

import (
	"testing"
)

func TestNewBox(t *testing.T) {
	box := NewBox(100, 50)
	if box.Width != 100 || box.Height != 50 {
		t.Errorf("NewBox(100, 50) = {%d, %d}, want {100, 50}", box.Width, box.Height)
	}
}

func TestNewContainer(t *testing.T) {
	c := NewContainer(Row, 800, 600)
	if c.Direction != Row {
		t.Errorf("Direction = %v, want Row", c.Direction)
	}
	if c.Width != 800 || c.Height != 600 {
		t.Errorf("Dimensions = {%d, %d}, want {800, 600}", c.Width, c.Height)
	}
	if c.Align != AlignStart {
		t.Errorf("Default Align = %v, want AlignStart", c.Align)
	}
	if c.Justify != JustifyStart {
		t.Errorf("Default Justify = %v, want JustifyStart", c.Justify)
	}
}

func TestContainerSetters(t *testing.T) {
	c := NewContainer(Row, 100, 100)

	c.SetAlign(AlignCenter)
	if c.Align != AlignCenter {
		t.Errorf("SetAlign(AlignCenter): Align = %v, want AlignCenter", c.Align)
	}

	c.SetJustify(JustifyEnd)
	if c.Justify != JustifyEnd {
		t.Errorf("SetJustify(JustifyEnd): Justify = %v, want JustifyEnd", c.Justify)
	}

	c.SetGap(10)
	if c.Gap != 10 {
		t.Errorf("SetGap(10): Gap = %d, want 10", c.Gap)
	}

	padding := Padding{Top: 5, Right: 10, Bottom: 15, Left: 20}
	c.SetPadding(padding)
	if c.Padding != padding {
		t.Errorf("SetPadding: got %+v, want %+v", c.Padding, padding)
	}
}

func TestLayoutRowBasic(t *testing.T) {
	c := NewContainer(Row, 300, 100)
	c.Add(NewBox(100, 50))
	c.Add(NewBox(100, 50))

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// First item at (0, 0)
	if items[0].X != 0 || items[0].Y != 0 {
		t.Errorf("items[0] position = (%d, %d), want (0, 0)", items[0].X, items[0].Y)
	}
	if items[0].Width != 100 || items[0].Height != 50 {
		t.Errorf("items[0] size = (%d, %d), want (100, 50)", items[0].Width, items[0].Height)
	}

	// Second item at (100, 0)
	if items[1].X != 100 || items[1].Y != 0 {
		t.Errorf("items[1] position = (%d, %d), want (100, 0)", items[1].X, items[1].Y)
	}
}

func TestLayoutColumnBasic(t *testing.T) {
	c := NewContainer(Column, 100, 300)
	c.Add(NewBox(50, 100))
	c.Add(NewBox(50, 100))

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// First item at (0, 0)
	if items[0].X != 0 || items[0].Y != 0 {
		t.Errorf("items[0] position = (%d, %d), want (0, 0)", items[0].X, items[0].Y)
	}

	// Second item at (0, 100)
	if items[1].X != 0 || items[1].Y != 100 {
		t.Errorf("items[1] position = (%d, %d), want (0, 100)", items[1].X, items[1].Y)
	}
}

func TestLayoutRowWithGap(t *testing.T) {
	c := NewContainer(Row, 320, 100)
	c.SetGap(10)
	c.Add(NewBox(100, 50))
	c.Add(NewBox(100, 50))
	c.Add(NewBox(100, 50))

	items := c.Layout()
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}

	if items[0].X != 0 {
		t.Errorf("items[0].X = %d, want 0", items[0].X)
	}
	if items[1].X != 110 {
		t.Errorf("items[1].X = %d, want 110", items[1].X)
	}
	if items[2].X != 220 {
		t.Errorf("items[2].X = %d, want 220", items[2].X)
	}
}

func TestLayoutColumnWithGap(t *testing.T) {
	c := NewContainer(Column, 100, 320)
	c.SetGap(10)
	c.Add(NewBox(50, 100))
	c.Add(NewBox(50, 100))
	c.Add(NewBox(50, 100))

	items := c.Layout()
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}

	if items[0].Y != 0 {
		t.Errorf("items[0].Y = %d, want 0", items[0].Y)
	}
	if items[1].Y != 110 {
		t.Errorf("items[1].Y = %d, want 110", items[1].Y)
	}
	if items[2].Y != 220 {
		t.Errorf("items[2].Y = %d, want 220", items[2].Y)
	}
}

func TestLayoutRowWithPadding(t *testing.T) {
	c := NewContainer(Row, 320, 120)
	c.SetPadding(Padding{Top: 10, Right: 10, Bottom: 10, Left: 10})
	c.Add(NewBox(100, 50))
	c.Add(NewBox(100, 50))

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// First item should start at padding.Left
	if items[0].X != 10 || items[0].Y != 10 {
		t.Errorf("items[0] position = (%d, %d), want (10, 10)", items[0].X, items[0].Y)
	}
}

func TestLayoutRowFlexGrow(t *testing.T) {
	c := NewContainer(Row, 400, 100)
	c.AddFlex(NewBox(100, 50), 1, 1, 0)
	c.AddFlex(NewBox(100, 50), 2, 1, 0)

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// Remaining space: 400 - 200 = 200
	// First item gets 1/3 = 66 extra, total 166
	// Second item gets 2/3 = 133 extra, total 233
	if items[0].Width < 150 || items[0].Width > 170 {
		t.Errorf("items[0].Width = %d, want ~166", items[0].Width)
	}
	if items[1].Width < 220 || items[1].Width > 240 {
		t.Errorf("items[1].Width = %d, want ~233", items[1].Width)
	}
}

func TestLayoutColumnFlexGrow(t *testing.T) {
	c := NewContainer(Column, 100, 400)
	c.AddFlex(NewBox(50, 100), 1, 1, 0)
	c.AddFlex(NewBox(50, 100), 2, 1, 0)

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// Remaining space: 400 - 200 = 200
	// First item gets 1/3 = 66 extra
	// Second item gets 2/3 = 133 extra
	if items[0].Height < 150 || items[0].Height > 170 {
		t.Errorf("items[0].Height = %d, want ~166", items[0].Height)
	}
	if items[1].Height < 220 || items[1].Height > 240 {
		t.Errorf("items[1].Height = %d, want ~233", items[1].Height)
	}
}

func TestLayoutRowAlignCenter(t *testing.T) {
	c := NewContainer(Row, 300, 100)
	c.SetAlign(AlignCenter)
	c.Add(NewBox(100, 40))
	c.Add(NewBox(100, 60))

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// First item: (100 - 40) / 2 = 30
	if items[0].Y != 30 {
		t.Errorf("items[0].Y = %d, want 30", items[0].Y)
	}

	// Second item: (100 - 60) / 2 = 20
	if items[1].Y != 20 {
		t.Errorf("items[1].Y = %d, want 20", items[1].Y)
	}
}

func TestLayoutColumnAlignCenter(t *testing.T) {
	c := NewContainer(Column, 100, 300)
	c.SetAlign(AlignCenter)
	c.Add(NewBox(40, 100))
	c.Add(NewBox(60, 100))

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// First item: (100 - 40) / 2 = 30
	if items[0].X != 30 {
		t.Errorf("items[0].X = %d, want 30", items[0].X)
	}

	// Second item: (100 - 60) / 2 = 20
	if items[1].X != 20 {
		t.Errorf("items[1].X = %d, want 20", items[1].X)
	}
}

func TestLayoutRowAlignEnd(t *testing.T) {
	c := NewContainer(Row, 300, 100)
	c.SetAlign(AlignEnd)
	c.Add(NewBox(100, 40))

	items := c.Layout()
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}

	// Item should be at bottom: 100 - 40 = 60
	if items[0].Y != 60 {
		t.Errorf("items[0].Y = %d, want 60", items[0].Y)
	}
}

func TestLayoutRowAlignStretch(t *testing.T) {
	c := NewContainer(Row, 300, 100)
	c.SetAlign(AlignStretch)
	c.Add(NewBox(100, 40))

	items := c.Layout()
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}

	// Height should be stretched to container height
	if items[0].Height != 100 {
		t.Errorf("items[0].Height = %d, want 100", items[0].Height)
	}
}

func TestLayoutRowJustifyCenter(t *testing.T) {
	c := NewContainer(Row, 400, 100)
	c.SetJustify(JustifyCenter)
	c.Add(NewBox(100, 50))
	c.Add(NewBox(100, 50))

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// Total width = 200, remaining = 200, offset = 100
	if items[0].X != 100 {
		t.Errorf("items[0].X = %d, want 100", items[0].X)
	}
	if items[1].X != 200 {
		t.Errorf("items[1].X = %d, want 200", items[1].X)
	}
}

func TestLayoutRowJustifyEnd(t *testing.T) {
	c := NewContainer(Row, 400, 100)
	c.SetJustify(JustifyEnd)
	c.Add(NewBox(100, 50))
	c.Add(NewBox(100, 50))

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// Should start at 400 - 200 = 200
	if items[0].X != 200 {
		t.Errorf("items[0].X = %d, want 200", items[0].X)
	}
	if items[1].X != 300 {
		t.Errorf("items[1].X = %d, want 300", items[1].X)
	}
}

func TestLayoutRowJustifySpaceBetween(t *testing.T) {
	c := NewContainer(Row, 500, 100)
	c.SetJustify(JustifySpaceBetween)
	c.Add(NewBox(100, 50))
	c.Add(NewBox(100, 50))
	c.Add(NewBox(100, 50))

	items := c.Layout()
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}

	// Total width = 300, remaining = 200, gaps = 100 each
	if items[0].X != 0 {
		t.Errorf("items[0].X = %d, want 0", items[0].X)
	}
	if items[2].X != 400 {
		t.Errorf("items[2].X = %d, want 400", items[2].X)
	}
}

func TestLayoutEmptyContainer(t *testing.T) {
	c := NewContainer(Row, 100, 100)
	items := c.Layout()
	if items != nil {
		t.Errorf("Layout() on empty container = %v, want nil", items)
	}
}

func TestLayoutWithFlexBasis(t *testing.T) {
	c := NewContainer(Row, 400, 100)
	c.AddFlex(NewBox(50, 50), 1, 1, 100) // basis = 100
	c.AddFlex(NewBox(50, 50), 1, 1, 150) // basis = 150

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// Remaining: 400 - 250 = 150
	// Each gets 75 extra
	if items[0].Width < 160 || items[0].Width > 180 {
		t.Errorf("items[0].Width = %d, want ~175", items[0].Width)
	}
	if items[1].Width < 210 || items[1].Width > 230 {
		t.Errorf("items[1].Width = %d, want ~225", items[1].Width)
	}
}

func TestLayoutRowFlexShrink(t *testing.T) {
	c := NewContainer(Row, 150, 100)
	c.AddFlex(NewBox(100, 50), 0, 1, 0)
	c.AddFlex(NewBox(100, 50), 0, 2, 0)

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// Over by 50 pixels
	// First shrinks by 1/3 = 16, final = 84
	// Second shrinks by 2/3 = 33, final = 67
	if items[0].Width < 80 || items[0].Width > 88 {
		t.Errorf("items[0].Width = %d, want ~84", items[0].Width)
	}
	if items[1].Width < 63 || items[1].Width > 71 {
		t.Errorf("items[1].Width = %d, want ~67", items[1].Width)
	}
}

func TestLayoutWithNegativePadding(t *testing.T) {
	c := NewContainer(Row, 100, 100)
	c.SetPadding(Padding{Top: 60, Right: 60, Bottom: 60, Left: 60})
	c.Add(NewBox(50, 50))

	items := c.Layout()
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}

	// Should handle negative content dimensions gracefully
	if items[0].X != 60 {
		t.Errorf("items[0].X = %d, want 60", items[0].X)
	}
}

func TestLayoutBoxData(t *testing.T) {
	c := NewContainer(Row, 300, 100)
	box1 := NewBox(100, 50)
	box1.Data = "first"
	box2 := NewBox(100, 50)
	box2.Data = "second"

	c.Add(box1)
	c.Add(box2)

	items := c.Layout()
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	if items[0].Box.Data != "first" {
		t.Errorf("items[0].Box.Data = %v, want 'first'", items[0].Box.Data)
	}
	if items[1].Box.Data != "second" {
		t.Errorf("items[1].Box.Data = %v, want 'second'", items[1].Box.Data)
	}
}

func BenchmarkLayoutRow(b *testing.B) {
	c := NewContainer(Row, 1000, 100)
	for i := 0; i < 10; i++ {
		c.AddFlex(NewBox(50, 50), 1, 1, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Layout()
	}
}

func BenchmarkLayoutColumn(b *testing.B) {
	c := NewContainer(Column, 100, 1000)
	for i := 0; i < 10; i++ {
		c.AddFlex(NewBox(50, 50), 1, 1, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Layout()
	}
}
