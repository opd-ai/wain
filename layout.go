package wain

import (
	"github.com/opd-ai/wain/internal/raster/primitives"
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
// Align controls positioning on the cross axis: for Row containers this is
// vertical (top/center/bottom), for Column containers this is horizontal
// (left/center/right).
type Align int

const (
	// AlignStart aligns children to the start of the cross axis.
	// For Row: top edge. For Column: left edge.
	AlignStart Align = Align(pctwidget.AlignStart)

	// AlignCenter centers children on the cross axis.
	// For Row: vertical center. For Column: horizontal center.
	AlignCenter Align = Align(pctwidget.AlignCenter)

	// AlignEnd aligns children to the end of the cross axis.
	// For Row: bottom edge. For Column: right edge.
	AlignEnd Align = Align(pctwidget.AlignEnd)

	// AlignStretch stretches children to fill the cross axis.
	// For Row: children fill container height. For Column: children fill container width.
	AlignStretch Align = Align(pctwidget.AlignStretch)
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
	internal      *pctwidget.Panel
	styleOverride *StyleOverride
	theme         *Theme
}

// NewPanel creates a new Panel with percentage-based dimensions.
// NewPanel takes a size parameter specifying Width and Height as percentages (0-100)
// of the parent container. Values are automatically clamped to the valid range.
// The panel uses the default theme style initially. Use SetStyle to customize
// appearance.
func NewPanel(size Size) *Panel {
	internal := pctwidget.NewPanel(size.Width, size.Height)
	return &Panel{internal: internal}
}

// Add appends a child widget to this panel.
// Add lays out children according to the panel's flow direction (Row or Column).
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
// Children returns widgets in add order, which determines both layout and z-order
// for rendering. Modifying the returned slice does not affect the panel's state.
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
// HandleEvent returns false by default, allowing events to propagate to children.
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
// SetFlowDirection accepts FlowRow (horizontal, left to right) or
// FlowColumn (vertical, top to bottom). The default is FlowColumn.
func (p *Panel) SetFlowDirection(dir FlowDirection) {
	p.internal.SetFlowDirection(pctwidget.FlowDirection(dir))
}

// FlowDirection returns the current flow direction.
func (p *Panel) FlowDirection() FlowDirection {
	return FlowDirection(p.internal.FlowDirection())
}

// SetPadding sets the padding (in pixels) around the panel's content area.
// SetPadding creates space between the panel's border and its children.
func (p *Panel) SetPadding(pixels int) {
	if p.styleOverride == nil {
		p.styleOverride = &StyleOverride{}
	}
	p.styleOverride.Padding = &pixels
	p.syncStyleToInternal()
}

// SetGap sets the spacing (in pixels) between child widgets.
// SetGap applies to both Row and Column flow directions.
func (p *Panel) SetGap(pixels int) {
	if p.styleOverride == nil {
		p.styleOverride = &StyleOverride{}
	}
	p.styleOverride.Gap = &pixels
	p.syncStyleToInternal()
}

// SetAlign sets the cross-axis alignment for children.
// SetAlign controls alignment on the cross axis: vertical for Row (top/center/bottom)
// or horizontal for Column (left/center/right). Default is AlignStart.
func (p *Panel) SetAlign(align Align) {
	p.internal.SetAlign(pctwidget.Align(align))
}

// Align returns the current cross-axis alignment.
func (p *Panel) Align() Align {
	return Align(p.internal.GetAlign())
}

// SetPosition manually overrides the auto-layout position and dimensions.
// SetPosition disables automatic layout for this panel, using absolute positioning.
// Call ClearPosition to return to automatic layout.
func (p *Panel) SetPosition(x, y, width, height int) {
	p.internal.SetPosition(x, y, width, height)
}

// ClearPosition removes the manual position override and returns the panel
// to percentage-based automatic layout.
func (p *Panel) ClearPosition() {
	p.internal.ClearPosition()
}

// SetVisible controls whether this panel is drawn and participates in layout.
// SetVisible(false) hides the panel, which then consumes no space and is not rendered.
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
	p.styleOverride = &override
	p.syncStyleToInternal()
}

// SetTheme applies a theme to this panel and all its children.
// SetTheme controls the visual appearance of widgets that do not have
// a StyleOverride applied. It recursively propagates the theme to all
// descendant panels.
func (p *Panel) SetTheme(theme Theme) {
	p.theme = &theme
	p.syncStyleToInternal()

	// Recursively propagate to children
	for _, child := range p.Children() {
		if childPanel := extractPanel(child); childPanel != nil {
			childPanel.SetTheme(theme)
		}
	}
}

// extractPanel returns the underlying Panel from a PublicWidget, if any.
func extractPanel(w PublicWidget) *Panel {
	switch widget := w.(type) {
	case *Panel:
		return widget
	case *Row:
		return widget.Panel
	case *Column:
		return widget.Panel
	case *Stack:
		return widget.Panel
	case *Grid:
		return widget.Panel
	default:
		return nil
	}
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
// NewRow fills 100% width and height of its parent by default.
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
// NewColumn fills 100% width and height of its parent by default.
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
// NewStack fills 100% width and height of its parent by default.
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
// NewGrid fills 100% width and height of its parent by default.
// Children are arranged left-to-right, top-to-bottom, and rows are computed
// automatically. If columns is less than 1, it defaults to 1.
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
// SetColumns triggers a layout recomputation on the next frame.
// If columns is less than 1, it is set to 1.
func (g *Grid) SetColumns(columns int) {
	if columns < 1 {
		columns = 1
	}
	g.columns = columns
}

// syncStyleToInternal applies the panel's style override to the internal widget.
// This method creates a themeAdapter that wraps the current theme with the override.
func (p *Panel) syncStyleToInternal() {
	if p.styleOverride == nil && p.theme == nil {
		p.internal.SetStyle(nil)
		return
	}

	baseTheme := DefaultDark()
	if p.theme != nil {
		baseTheme = *p.theme
	}

	style := &themeAdapter{
		base:     baseTheme,
		override: p.styleOverride,
	}
	p.internal.SetStyle(style)
}

// themeAdapter adapts Theme + StyleOverride to pctwidget.Style interface.
type themeAdapter struct {
	base     Theme
	override *StyleOverride
}

// Background returns the background color, using the override if present or the base theme otherwise.
// This implements pctwidget.Style.Background.
func (s *themeAdapter) Background() primitives.Color {
	if s.override != nil && s.override.Background != nil {
		return s.override.Background.toInternal()
	}
	return s.base.Background.toInternal()
}

// Foreground returns the foreground (text) color, using the override if present or the base theme otherwise.
// This implements pctwidget.Style.Foreground.
func (s *themeAdapter) Foreground() primitives.Color {
	if s.override != nil && s.override.Foreground != nil {
		return s.override.Foreground.toInternal()
	}
	return s.base.Foreground.toInternal()
}

// Accent returns the accent/highlight color, using the override if present or the base theme otherwise.
// This implements pctwidget.Style.Accent.
func (s *themeAdapter) Accent() primitives.Color {
	if s.override != nil && s.override.Accent != nil {
		return s.override.Accent.toInternal()
	}
	return s.base.Accent.toInternal()
}

// Border returns the border color, using the override if present or the base theme otherwise.
// This implements pctwidget.Style.Border.
func (s *themeAdapter) Border() primitives.Color {
	if s.override != nil && s.override.Border != nil {
		return s.override.Border.toInternal()
	}
	return s.base.Border.toInternal()
}

// FontSize returns the base font size in pixels, using the override if present or the base theme otherwise.
// This implements pctwidget.Style.FontSize.
func (s *themeAdapter) FontSize() float64 {
	if s.override != nil && s.override.FontSize != nil {
		return *s.override.FontSize
	}
	return s.base.FontSize
}

// Padding returns the default inner padding in pixels, using the override if present or the base theme otherwise.
// This implements pctwidget.Style.Padding.
func (s *themeAdapter) Padding() int {
	if s.override != nil && s.override.Padding != nil {
		return *s.override.Padding
	}
	return s.base.Padding
}

// Gap returns the default gap between sibling widgets in pixels, using the override if present or the base theme otherwise.
// This implements pctwidget.Style.Gap.
func (s *themeAdapter) Gap() int {
	if s.override != nil && s.override.Gap != nil {
		return *s.override.Gap
	}
	return s.base.Gap
}

// BorderWidth returns the default border width in pixels, using the override if present or the base theme otherwise.
// This implements pctwidget.Style.BorderWidth.
func (s *themeAdapter) BorderWidth() int {
	if s.override != nil && s.override.BorderWidth != nil {
		return *s.override.BorderWidth
	}
	return s.base.BorderWidth
}
