package wain

import (
	"github.com/opd-ai/wain/internal/ui/pctwidget"
)

// Size represents percentage-based dimensions for a widget.
//
// Width and Height are specified as percentages (0-100) of the parent container.
// Values outside the valid range are automatically clamped to [0, 100].
//
// The percentage-based sizing model eliminates manual pixel calculations and
// enables responsive layouts that adapt to window resizes and HiDPI scaling
// automatically.
//
// Example usage:
//
//	sidebar := wain.NewPanel(wain.Size{Width: 25, Height: 100})  // 25% wide, full height
//	content := wain.NewPanel(wain.Size{Width: 75, Height: 100})  // 75% wide, full height
//	header := wain.NewPanel(wain.Size{Width: 100, Height: 10})   // full width, 10% tall
type Size struct {
	Width  float64 // Width as percentage of parent (0-100)
	Height float64 // Height as percentage of parent (0-100)
}

// FlowDirection controls how a container arranges its child widgets.
type FlowDirection int

const (
	// FlowRow arranges children horizontally, left to right.
	// Children's Width percentages determine their share of the horizontal space.
	// All children span the full height of the container.
	FlowRow FlowDirection = FlowDirection(pctwidget.FlowRow)

	// FlowColumn arranges children vertically, top to bottom.
	// Children's Height percentages determine their share of the vertical space.
	// All children span the full width of the container.
	FlowColumn FlowDirection = FlowDirection(pctwidget.FlowColumn)
)

// Align specifies alignment on the cross axis of a container.
//
// For Row containers, cross axis is vertical (controls top/center/bottom alignment).
// For Column containers, cross axis is horizontal (controls left/center/right alignment).
type Align int

const (
	// AlignStart aligns children to the start of the cross axis.
	// For Row: top edge. For Column: left edge.
	AlignStart Align = iota

	// AlignCenter centers children on the cross axis.
	// For Row: vertical center. For Column: horizontal center.
	AlignCenter

	// AlignEnd aligns children to the end of the cross axis.
	// For Row: bottom edge. For Column: right edge.
	AlignEnd

	// AlignStretch stretches children to fill the cross axis.
	// For Row: children fill container height. For Column: children fill container width.
	AlignStretch
)

// Panel is a styled rectangular container that holds child widgets.
//
// Panel supports percentage-based sizing and automatic layout. It can be used
// as a building block for complex UIs by nesting panels and setting their
// flow direction.
//
// Panel uses the current theme's style by default, but can be customized
// via SetStyle.
//
// Example usage:
//
//	panel := wain.NewPanel(wain.Size{Width: 50, Height: 100})
//	panel.SetFlowDirection(wain.FlowColumn)
//	panel.SetPadding(10)
//	panel.SetGap(5)
//	panel.Add(header)
//	panel.Add(content)
type Panel struct {
	internal *pctwidget.Panel
}

// NewPanel creates a new Panel with percentage-based dimensions.
//
// The size parameter specifies Width and Height as percentages (0-100) of
// the parent container. Values are automatically clamped to the valid range.
//
// The panel uses the default theme style initially. Use SetStyle to customize
// appearance.
func NewPanel(size Size) *Panel {
	internal := pctwidget.NewPanel(size.Width, size.Height)
	return &Panel{internal: internal}
}

// Add appends a child widget to this panel.
//
// Children are laid out according to the panel's flow direction (Row or Column).
// Each child's percentage size is relative to this panel's resolved pixel dimensions.
func (p *Panel) Add(child PublicWidget) {
	// Unwrap the public widget to get the internal panel.
	// Currently only supports Panel, Row, Column, Stack, and Grid types.
	// Phase 10.3 will add support for all concrete widget types.
	switch c := child.(type) {
	case *Panel:
		p.internal.Add(c.internal)
	case *Row:
		p.internal.Add(c.Panel.internal)
	case *Column:
		p.internal.Add(c.Panel.internal)
	case *Stack:
		p.internal.Add(c.Panel.internal)
	case *Grid:
		p.internal.Add(c.Panel.internal)
	}
}

// Children returns a slice of this panel's child widgets.
//
// The order matches the add order and determines both layout order and z-order
// for rendering. Modifying the returned slice does not affect the panel's
// internal state.
func (p *Panel) Children() []PublicWidget {
	internalChildren := p.internal.Children()
	children := make([]PublicWidget, len(internalChildren))
	for i, child := range internalChildren {
		// Wrap each internal panel in a public Panel.
		// When we add support for other widget types, we'll need to
		// track the original widget type to wrap correctly.
		children[i] = &Panel{internal: child}
	}
	return children
}

// Bounds returns the current pixel dimensions of the panel after layout resolution.
func (p *Panel) Bounds() (width, height int) {
	_, _, width, height = p.internal.ResolvedBounds()
	return width, height
}

// HandleEvent processes a user interaction event.
//
// Panels do not consume events by default, allowing them to propagate to children.
func (p *Panel) HandleEvent(Event) bool {
	// Panels are passive containers - they don't handle events directly.
	// Event handling will be implemented in Phase 9.3 for interactive widgets.
	return false
}

// Draw renders the panel and its children to the provided canvas.
func (p *Panel) Draw(Canvas) {
	// Drawing implementation will be integrated in Phase 9.4 (Render Integration Bridge).
	// The panel's visual appearance is controlled by its Style.
}

// SetFlowDirection sets how this panel arranges its children.
//
// FlowRow arranges children horizontally (left to right).
// FlowColumn arranges children vertically (top to bottom).
//
// The default flow direction is FlowColumn.
func (p *Panel) SetFlowDirection(dir FlowDirection) {
	p.internal.SetFlowDirection(pctwidget.FlowDirection(dir))
}

// FlowDirection returns the current flow direction.
func (p *Panel) FlowDirection() FlowDirection {
	return FlowDirection(p.internal.FlowDirection())
}

// SetPadding sets the padding (in pixels) around the panel's content area.
//
// Padding creates space between the panel's border and its children.
// This is applied before children are laid out.
func (p *Panel) SetPadding(pixels int) {
	// TODO: Implement via internal panel's style customization.
	// The internal pctwidget.Panel uses Style.Padding().
	// This will be completed when we expose StyleOverride in Phase 10.5.
	_ = pixels
}

// SetGap sets the spacing (in pixels) between child widgets.
//
// Gap is the space inserted between children during layout, both for
// Row and Column flow directions.
func (p *Panel) SetGap(pixels int) {
	// TODO: Implement via internal panel's style customization.
	// The internal pctwidget.Panel uses Style.Gap().
	// This will be completed when we expose StyleOverride in Phase 10.5.
	_ = pixels
}

// SetAlign sets the cross-axis alignment for children.
//
// For Row containers, controls vertical alignment (top/center/bottom).
// For Column containers, controls horizontal alignment (left/center/right).
func (p *Panel) SetAlign(align Align) {
	// TODO: Implement cross-axis alignment.
	// This requires extending the internal pctwidget.AutoLayout to support
	// alignment modes beyond the current start-aligned behavior.
	// Will be implemented in Phase 10.4.
	_ = align
}

// SetPosition manually overrides the auto-layout position and dimensions.
//
// After calling SetPosition, this panel will not be repositioned by the
// auto-layout engine. Use this for absolute positioning when percentage-based
// layout is insufficient.
//
// To return to automatic layout, call ClearPosition.
func (p *Panel) SetPosition(x, y, width, height int) {
	p.internal.SetPosition(x, y, width, height)
}

// ClearPosition removes the manual position override and returns the panel
// to percentage-based automatic layout.
func (p *Panel) ClearPosition() {
	p.internal.ClearPosition()
}

// SetVisible controls whether this panel is drawn and participates in layout.
//
// Hidden panels (visible = false) do not consume space in their parent's layout
// and are not rendered.
func (p *Panel) SetVisible(visible bool) {
	p.internal.SetVisible(visible)
}

// Visible reports whether the panel is currently visible.
func (p *Panel) Visible() bool {
	return p.internal.Visible()
}

// SetStyle applies a style override to this panel.
//
// The override allows customizing specific visual properties while inheriting
// others from the theme. Any field left nil in the override will use the
// theme's value.
//
// Example:
//
//	bg := wain.RGB(40, 40, 60)
//	panel.SetStyle(wain.StyleOverride{Background: &bg})
func (p *Panel) SetStyle(override StyleOverride) {
	// This is a placeholder - actual implementation will be added when
	// we integrate the theme system with internal widgets
	_ = override
}

// Row is a convenience container that arranges children horizontally.
//
// Row is equivalent to a Panel with FlowDirection set to FlowRow.
// Children are laid out left to right, with each child's Width percentage
// determining its share of the horizontal space.
type Row struct {
	*Panel
}

// NewRow creates a new horizontal container.
//
// The row automatically fills 100% width and height of its parent.
// Add children with custom Size values to control their dimensions.
func NewRow() *Row {
	panel := NewPanel(Size{Width: 100, Height: 100})
	panel.SetFlowDirection(FlowRow)
	return &Row{Panel: panel}
}

// Column is a convenience container that arranges children vertically.
//
// Column is equivalent to a Panel with FlowDirection set to FlowColumn.
// Children are laid out top to bottom, with each child's Height percentage
// determining its share of the vertical space.
type Column struct {
	*Panel
}

// NewColumn creates a new vertical container.
//
// The column automatically fills 100% width and height of its parent.
// Add children with custom Size values to control their dimensions.
func NewColumn() *Column {
	panel := NewPanel(Size{Width: 100, Height: 100})
	panel.SetFlowDirection(FlowColumn)
	return &Column{Panel: panel}
}

// Stack is a layering container that places children on top of each other.
//
// Stack is useful for overlays, modals, tooltips, and other UI elements
// that need to be layered. All children are positioned at the same location
// and rendered in the order they were added (first child on bottom, last on top).
//
// Each child's percentage-based size is resolved against the stack's dimensions.
// Children can have different sizes - they do not need to fill the entire stack.
//
// Example usage:
//
//	stack := wain.NewStack()
//	stack.Add(background)  // drawn first (bottom layer)
//	stack.Add(content)     // drawn second (middle layer)
//	stack.Add(tooltip)     // drawn last (top layer)
type Stack struct {
	*Panel
}

// NewStack creates a new layering container.
//
// The stack automatically fills 100% width and height of its parent.
// Children are layered in the order added (first = bottom, last = top).
func NewStack() *Stack {
	panel := NewPanel(Size{Width: 100, Height: 100})
	// Stack uses a special flow direction that we'll handle in autolayout
	return &Stack{Panel: panel}
}

// Grid is a fixed-column grid container.
//
// Grid arranges children in a grid with a fixed number of columns. Each cell
// is evenly divided, and children's percentage sizes are relative to their cell.
// Children are added left-to-right, top-to-bottom.
//
// The grid automatically calculates the number of rows needed based on the
// number of children and the column count.
//
// Example usage:
//
//	grid := wain.NewGrid(3)  // 3 columns
//	for i := 0; i < 9; i++ {
//	    grid.Add(wain.NewPanel(wain.Size{Width: 100, Height: 100}))
//	}
//	// Creates a 3x3 grid where each cell is equal size
type Grid struct {
	*Panel
	columns int
}

// NewGrid creates a new grid container with the specified number of columns.
//
// The grid automatically fills 100% width and height of its parent.
// Children are arranged in cells, left-to-right, top-to-bottom.
// The number of rows is calculated based on the number of children.
//
// The columns parameter must be positive. If columns is less than 1,
// it defaults to 1.
func NewGrid(columns int) *Grid {
	if columns < 1 {
		columns = 1
	}
	panel := NewPanel(Size{Width: 100, Height: 100})
	return &Grid{
		Panel:   panel,
		columns: columns,
	}
}

// Columns returns the number of columns in the grid.
func (g *Grid) Columns() int {
	return g.columns
}

// SetColumns changes the number of columns in the grid.
//
// The layout will be recomputed on the next frame.
// The columns parameter must be positive. If columns is less than 1,
// it is set to 1.
func (g *Grid) SetColumns(columns int) {
	if columns < 1 {
		columns = 1
	}
	g.columns = columns
}
