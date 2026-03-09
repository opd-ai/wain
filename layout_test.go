package wain

import (
	"testing"
)

func TestSize(t *testing.T) {
	tests := []struct {
		name  string
		size  Size
		wantW float64
		wantH float64
	}{
		{"normal values", Size{Width: 50, Height: 75}, 50, 75},
		{"zero values", Size{Width: 0, Height: 0}, 0, 0},
		{"full size", Size{Width: 100, Height: 100}, 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.size.Width != tt.wantW {
				t.Errorf("Width = %v, want %v", tt.size.Width, tt.wantW)
			}
			if tt.size.Height != tt.wantH {
				t.Errorf("Height = %v, want %v", tt.size.Height, tt.wantH)
			}
		})
	}
}

func TestNewPanel(t *testing.T) {
	tests := []struct {
		name string
		size Size
	}{
		{"half width full height", Size{Width: 50, Height: 100}},
		{"quarter size", Size{Width: 25, Height: 25}},
		{"full size", Size{Width: 100, Height: 100}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := NewPanel(tt.size)
			if panel == nil {
				t.Fatal("NewPanel returned nil")
			}
			if panel.internal == nil {
				t.Fatal("Panel.internal is nil")
			}
		})
	}
}

func TestPanelChildren(t *testing.T) {
	parent := NewPanel(Size{Width: 100, Height: 100})
	child1 := NewPanel(Size{Width: 50, Height: 50})
	child2 := NewPanel(Size{Width: 50, Height: 50})

	parent.Add(child1)
	parent.Add(child2)

	children := parent.Children()
	if len(children) != 2 {
		t.Errorf("Children() returned %d children, want 2", len(children))
	}
}

func TestPanelFlowDirection(t *testing.T) {
	panel := NewPanel(Size{Width: 100, Height: 100})

	// Default should be FlowColumn
	if dir := panel.FlowDirection(); dir != FlowColumn {
		t.Errorf("Default flow direction = %v, want FlowColumn", dir)
	}

	// Set to FlowRow
	panel.SetFlowDirection(FlowRow)
	if dir := panel.FlowDirection(); dir != FlowRow {
		t.Errorf("After SetFlowDirection(FlowRow), got %v, want FlowRow", dir)
	}

	// Set back to FlowColumn
	panel.SetFlowDirection(FlowColumn)
	if dir := panel.FlowDirection(); dir != FlowColumn {
		t.Errorf("After SetFlowDirection(FlowColumn), got %v, want FlowColumn", dir)
	}
}

func TestPanelVisibility(t *testing.T) {
	panel := NewPanel(Size{Width: 100, Height: 100})

	// Default should be visible
	if !panel.Visible() {
		t.Error("Default Visible() = false, want true")
	}

	// Hide the panel
	panel.SetVisible(false)
	if panel.Visible() {
		t.Error("After SetVisible(false), Visible() = true, want false")
	}

	// Show the panel again
	panel.SetVisible(true)
	if !panel.Visible() {
		t.Error("After SetVisible(true), Visible() = false, want true")
	}
}

func TestPanelPosition(t *testing.T) {
	panel := NewPanel(Size{Width: 50, Height: 50})

	// Set manual position
	panel.SetPosition(100, 200, 300, 400)

	// Bounds should reflect the manual dimensions
	width, height := panel.Bounds()
	if width != 300 || height != 400 {
		t.Errorf("Bounds() = (%d, %d), want (300, 400)", width, height)
	}

	// Clear manual position
	panel.ClearPosition()

	// After clearing, bounds depend on layout (which hasn't run yet)
	// so we just verify the operation doesn't panic
}

func TestPanelBounds(t *testing.T) {
	panel := NewPanel(Size{Width: 50, Height: 50})

	// Before any layout resolution, bounds should be zero
	width, height := panel.Bounds()
	if width != 0 || height != 0 {
		t.Errorf("Bounds() before layout = (%d, %d), want (0, 0)", width, height)
	}
}

func TestPanelHandleEvent(t *testing.T) {
	panel := NewPanel(Size{Width: 100, Height: 100})

	// Create a dummy event (using an empty struct for now)
	var event Event

	// Panels should not consume events by default
	consumed := panel.HandleEvent(event)
	if consumed {
		t.Error("Panel.HandleEvent() = true, want false (panels don't consume events)")
	}
}

func TestNewRow(t *testing.T) {
	row := NewRow()
	if row == nil {
		t.Fatal("NewRow() returned nil")
	}
	if row.Panel == nil {
		t.Fatal("Row.Panel is nil")
	}

	// Row should have FlowRow direction
	if dir := row.FlowDirection(); dir != FlowRow {
		t.Errorf("Row.FlowDirection() = %v, want FlowRow", dir)
	}
}

func TestNewColumn(t *testing.T) {
	col := NewColumn()
	if col == nil {
		t.Fatal("NewColumn() returned nil")
	}
	if col.Panel == nil {
		t.Fatal("Column.Panel is nil")
	}

	// Column should have FlowColumn direction
	if dir := col.FlowDirection(); dir != FlowColumn {
		t.Errorf("Column.FlowDirection() = %v, want FlowColumn", dir)
	}
}

func TestRowColumnNesting(t *testing.T) {
	// Create a layout: Row containing two columns
	row := NewRow()
	leftCol := NewColumn()
	rightCol := NewColumn()

	row.Add(leftCol)
	row.Add(rightCol)

	children := row.Children()
	if len(children) != 2 {
		t.Errorf("Row has %d children, want 2", len(children))
	}

	// Add panels to columns
	leftCol.Add(NewPanel(Size{Width: 100, Height: 50}))
	leftCol.Add(NewPanel(Size{Width: 100, Height: 50}))

	leftChildren := leftCol.Children()
	if len(leftChildren) != 2 {
		t.Errorf("Left column has %d children, want 2", len(leftChildren))
	}
}

func TestThreePanelLayout(t *testing.T) {
	// Test the milestone layout: header 100×10%, sidebar 25×90%, content 75×90%
	root := NewColumn()

	header := NewPanel(Size{Width: 100, Height: 10})
	contentRow := NewRow()
	sidebar := NewPanel(Size{Width: 25, Height: 90})
	content := NewPanel(Size{Width: 75, Height: 90})

	contentRow.Add(sidebar)
	contentRow.Add(content)

	root.Add(header)
	root.Add(contentRow)

	// Verify the structure
	rootChildren := root.Children()
	if len(rootChildren) != 2 {
		t.Errorf("Root has %d children, want 2", len(rootChildren))
	}

	rowChildren := contentRow.Children()
	if len(rowChildren) != 2 {
		t.Errorf("Content row has %d children, want 2", len(rowChildren))
	}
}

func TestAlignConstants(t *testing.T) {
	// Verify all alignment constants are defined
	alignments := []Align{AlignStart, AlignCenter, AlignEnd, AlignStretch}
	if len(alignments) != 4 {
		t.Errorf("Expected 4 alignment constants, got %d", len(alignments))
	}
}

func TestFlowDirectionConstants(t *testing.T) {
	// Verify flow direction constants are distinct
	if FlowRow == FlowColumn {
		t.Error("FlowRow and FlowColumn have the same value")
	}
}

func TestNewStack(t *testing.T) {
	stack := NewStack()
	if stack == nil {
		t.Fatal("NewStack() returned nil")
	}
	if stack.Panel == nil {
		t.Fatal("Stack.Panel is nil")
	}
}

func TestStackLayering(t *testing.T) {
	stack := NewStack()

	// Add three layers
	background := NewPanel(Size{Width: 100, Height: 100})
	content := NewPanel(Size{Width: 80, Height: 80})
	overlay := NewPanel(Size{Width: 50, Height: 50})

	stack.Add(background)
	stack.Add(content)
	stack.Add(overlay)

	children := stack.Children()
	if len(children) != 3 {
		t.Errorf("Stack has %d children, want 3", len(children))
	}
}

func TestNewGrid(t *testing.T) {
	tests := []struct {
		name    string
		columns int
		want    int
	}{
		{"3 columns", 3, 3},
		{"1 column", 1, 1},
		{"5 columns", 5, 5},
		{"zero columns defaults to 1", 0, 1},
		{"negative columns defaults to 1", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grid := NewGrid(tt.columns)
			if grid == nil {
				t.Fatal("NewGrid() returned nil")
			}
			if grid.Panel == nil {
				t.Fatal("Grid.Panel is nil")
			}
			if grid.Columns() != tt.want {
				t.Errorf("Grid.Columns() = %d, want %d", grid.Columns(), tt.want)
			}
		})
	}
}

func TestGridSetColumns(t *testing.T) {
	grid := NewGrid(3)

	// Change to 4 columns
	grid.SetColumns(4)
	if grid.Columns() != 4 {
		t.Errorf("After SetColumns(4), Columns() = %d, want 4", grid.Columns())
	}

	// Invalid values should clamp to 1
	grid.SetColumns(0)
	if grid.Columns() != 1 {
		t.Errorf("After SetColumns(0), Columns() = %d, want 1", grid.Columns())
	}

	grid.SetColumns(-5)
	if grid.Columns() != 1 {
		t.Errorf("After SetColumns(-5), Columns() = %d, want 1", grid.Columns())
	}
}

func TestGrid3x3Layout(t *testing.T) {
	grid := NewGrid(3)

	// Add 9 panels to create a 3x3 grid
	for i := 0; i < 9; i++ {
		panel := NewPanel(Size{Width: 100, Height: 100})
		grid.Add(panel)
	}

	children := grid.Children()
	if len(children) != 9 {
		t.Errorf("Grid has %d children, want 9", len(children))
	}
}

func TestGridPartialRow(t *testing.T) {
	grid := NewGrid(3)

	// Add 7 panels (2 full rows + 1 partial row)
	for i := 0; i < 7; i++ {
		panel := NewPanel(Size{Width: 100, Height: 100})
		grid.Add(panel)
	}

	children := grid.Children()
	if len(children) != 7 {
		t.Errorf("Grid has %d children, want 7", len(children))
	}
}

func TestContainerTypes(t *testing.T) {
	// Verify all container types can be created and used together
	root := NewColumn()

	header := NewRow()
	header.Add(NewPanel(Size{Width: 100, Height: 100}))

	content := NewRow()
	sidebar := NewColumn()
	sidebar.Add(NewPanel(Size{Width: 100, Height: 50}))
	sidebar.Add(NewPanel(Size{Width: 100, Height: 50}))

	mainArea := NewStack()
	mainArea.Add(NewPanel(Size{Width: 100, Height: 100}))
	mainArea.Add(NewPanel(Size{Width: 80, Height: 80}))

	content.Add(sidebar)
	content.Add(mainArea)

	footer := NewGrid(4)
	for i := 0; i < 4; i++ {
		footer.Add(NewPanel(Size{Width: 100, Height: 100}))
	}

	root.Add(header)
	root.Add(content)
	root.Add(footer)

	// Verify structure
	rootChildren := root.Children()
	if len(rootChildren) != 3 {
		t.Errorf("Root has %d children, want 3", len(rootChildren))
	}
}
