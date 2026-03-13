// Package layout implements a flexbox-like layout system for UI elements.
//
// This package provides a renderer-agnostic layout engine that computes positions
// and dimensions for UI elements based on constraints and flex rules. The layout
// system emits a display list of positioned elements rather than rendering pixels
// directly.
//
// # Layout Model
//
// The layout model is inspired by CSS Flexbox:
//
//   - Containers arrange children in rows or columns (main axis)
//   - Children can grow/shrink to fill available space
//   - Alignment controls positioning on the cross axis
//   - Gaps define spacing between children
//
// # Coordinate System
//
// The coordinate system is standard 2D: origin (0,0) at top-left, X increases
// right, Y increases down. All dimensions are in pixels.
//
// # Usage
//
// Create a container, add children with flex properties, then call Layout to
// compute final positions:
//
//	container := layout.NewContainer(Direction.Row, 800, 600)
//	container.Add(layout.NewBox(100, 100), FlexGrow(1))
//	container.Add(layout.NewBox(100, 100), FlexGrow(2))
//	items := container.Layout()
package layout

import (
	"errors"
)

var (
	// ErrInvalidDimensions is returned when width or height is negative.
	ErrInvalidDimensions = errors.New("layout: invalid dimensions")

	// ErrInvalidFlex is returned when flex values are negative.
	ErrInvalidFlex = errors.New("layout: invalid flex value")
)

// Direction specifies the main axis direction for layout.
type Direction int

const (
	// Row arranges children horizontally (left to right).
	Row Direction = iota

	// Column arranges children vertically (top to bottom).
	Column
)

// Align specifies alignment on the cross axis.
type Align int

const (
	// AlignStart aligns to the start of the cross axis.
	AlignStart Align = iota

	// AlignCenter aligns to the center of the cross axis.
	AlignCenter

	// AlignEnd aligns to the end of the cross axis.
	AlignEnd

	// AlignStretch stretches to fill the cross axis.
	AlignStretch
)

// Justify specifies alignment on the main axis.
type Justify int

const (
	// JustifyStart aligns to the start of the main axis.
	JustifyStart Justify = iota

	// JustifyCenter aligns to the center of the main axis.
	JustifyCenter

	// JustifyEnd aligns to the end of the main axis.
	JustifyEnd

	// JustifySpaceBetween distributes space between children.
	JustifySpaceBetween

	// JustifySpaceAround distributes space around children.
	JustifySpaceAround
)

// Box represents a rectangular UI element with dimensions.
type Box struct {
	Width  int
	Height int
	Data   interface{} // User-defined data for this box
}

// NewBox creates a box with the specified dimensions.
func NewBox(width, height int) *Box {
	return &Box{Width: width, Height: height}
}

// FlexItem represents a child element with flex properties.
type FlexItem struct {
	Box        *Box
	FlexGrow   float64
	FlexShrink float64
	FlexBasis  int
}

// LayoutItem represents a positioned element after layout computation.
type LayoutItem struct {
	Box    *Box
	X, Y   int
	Width  int
	Height int
}

// Container represents a flex container that arranges children.
type Container struct {
	Direction Direction
	Align     Align
	Justify   Justify
	Gap       int
	Width     int
	Height    int
	Padding   Padding
	children  []FlexItem
}

// Padding represents padding around a container.
type Padding struct {
	Top, Right, Bottom, Left int
}

// NewContainer creates a flex container with the specified direction and dimensions.
func NewContainer(dir Direction, width, height int) *Container {
	return &Container{
		Direction: dir,
		Align:     AlignStart,
		Justify:   JustifyStart,
		Gap:       0,
		Width:     width,
		Height:    height,
		Padding:   Padding{},
		children:  make([]FlexItem, 0),
	}
}

// SetAlign sets the cross-axis alignment.
func (c *Container) SetAlign(align Align) {
	c.Align = align
}

// SetJustify sets the main-axis justification.
func (c *Container) SetJustify(justify Justify) {
	c.Justify = justify
}

// SetGap sets the gap between children.
func (c *Container) SetGap(gap int) {
	c.Gap = gap
}

// SetPadding sets the padding around the container.
func (c *Container) SetPadding(padding Padding) {
	c.Padding = padding
}

// Add adds a child element to the container with default flex properties.
func (c *Container) Add(box *Box) {
	c.AddFlex(box, 0, 1, 0)
}

// AddFlex adds a child element with explicit flex properties.
func (c *Container) AddFlex(box *Box, grow, shrink float64, basis int) {
	c.children = append(c.children, FlexItem{
		Box:        box,
		FlexGrow:   grow,
		FlexShrink: shrink,
		FlexBasis:  basis,
	})
}

// Layout computes positions for all children and returns positioned items.
func (c *Container) Layout() []LayoutItem {
	if len(c.children) == 0 {
		return nil
	}

	contentWidth := c.Width - c.Padding.Left - c.Padding.Right
	contentHeight := c.Height - c.Padding.Top - c.Padding.Bottom

	if contentWidth < 0 {
		contentWidth = 0
	}
	if contentHeight < 0 {
		contentHeight = 0
	}

	if c.Direction == Row {
		return c.layoutRow(contentWidth, contentHeight)
	}
	return c.layoutColumn(contentWidth, contentHeight)
}

// flexMeasurement holds computed flex layout metrics for a child item.
type flexMeasurement struct {
	basis  int
	grow   float64
	shrink float64
}

// computeFlexMeasurements extracts flex metrics for all children in the specified axis.
// For rows, it measures widths; for columns, it measures heights.
func computeFlexMeasurements(children []FlexItem, isRow bool) ([]flexMeasurement, int, float64, float64) {
	measurements := make([]flexMeasurement, len(children))
	var baseSize int
	var totalGrow float64
	var totalShrink float64

	for i, item := range children {
		basis := item.FlexBasis
		if basis == 0 {
			if isRow {
				basis = item.Box.Width
			} else {
				basis = item.Box.Height
			}
		}

		measurements[i] = flexMeasurement{
			basis:  basis,
			grow:   item.FlexGrow,
			shrink: item.FlexShrink,
		}
		baseSize += basis
		totalGrow += item.FlexGrow
		totalShrink += item.FlexShrink
	}

	return measurements, baseSize, totalGrow, totalShrink
}

// distributeFlex computes final sizes by distributing remaining space according to flex rules.
func distributeFlex(measurements []flexMeasurement, availableSpace, baseSize int, totalGrow, totalShrink float64) []int {
	remaining := availableSpace - baseSize
	sizes := make([]int, len(measurements))

	for i, m := range measurements {
		if remaining > 0 && totalGrow > 0 {
			grow := int(float64(remaining) * (m.grow / totalGrow))
			sizes[i] = m.basis + grow
		} else if remaining < 0 && totalShrink > 0 {
			shrink := int(float64(-remaining) * (m.shrink / totalShrink))
			sizes[i] = m.basis - shrink
			if sizes[i] < 0 {
				sizes[i] = 0
			}
		} else {
			sizes[i] = m.basis
		}
	}

	return sizes
}

// computeJustifyOffset calculates the starting offset and gap adjustment for justification.
// Returns the starting offset and the (possibly modified) gap.
func computeJustifyOffset(justify Justify, sizes []int, contentSize, currentGap, padding int) (int, int) {
	offset := padding
	gap := currentGap
	totalSize := 0
	for _, s := range sizes {
		totalSize += s
	}
	totalGaps := gap * (len(sizes) - 1)
	if totalGaps < 0 {
		totalGaps = 0
	}
	totalSize += totalGaps

	switch justify {
	case JustifyCenter:
		offset += (contentSize - totalSize) / 2
	case JustifyEnd:
		offset += contentSize - totalSize
	case JustifySpaceBetween:
		if len(sizes) > 1 {
			gap = (contentSize - (totalSize - totalGaps)) / (len(sizes) - 1)
		}
	case JustifySpaceAround:
		if len(sizes) > 0 {
			space := (contentSize - (totalSize - totalGaps)) / len(sizes)
			gap = space
			offset += space / 2
		}
	}

	return offset, gap
}

// computeCrossAlign calculates the cross-axis position and size for a child.
// For rows, this computes Y and Height; for columns, X and Width.
func computeCrossAlign(align Align, childSize, contentSize, padding int) (int, int) {
	position := padding
	size := childSize

	switch align {
	case AlignCenter:
		position += (contentSize - childSize) / 2
	case AlignEnd:
		position += contentSize - childSize
	case AlignStretch:
		size = contentSize
	}

	return position, size
}

// layoutAxis implements the shared flex algorithm for both row and column layouts.
// isRow=true distributes along the X axis; isRow=false distributes along the Y axis.
func (c *Container) layoutAxis(isRow bool, contentWidth, contentHeight int) []LayoutItem {
	totalGaps := c.Gap * (len(c.children) - 1)
	if totalGaps < 0 {
		totalGaps = 0
	}

	var mainContent, crossContent int
	if isRow {
		mainContent, crossContent = contentWidth, contentHeight
	} else {
		mainContent, crossContent = contentHeight, contentWidth
	}

	measurements, base, totalGrow, totalShrink := computeFlexMeasurements(c.children, isRow)
	mainSizes := distributeFlex(measurements, mainContent-totalGaps, base, totalGrow, totalShrink)

	var mainPad, crossPad int
	if isRow {
		mainPad, crossPad = c.Padding.Left, c.Padding.Top
	} else {
		mainPad, crossPad = c.Padding.Top, c.Padding.Left
	}
	main, gap := computeJustifyOffset(c.Justify, mainSizes, mainContent, c.Gap, mainPad)

	items := make([]LayoutItem, 0, len(c.children))
	for i, item := range c.children {
		var crossItemSize int
		if isRow {
			crossItemSize = item.Box.Height
		} else {
			crossItemSize = item.Box.Width
		}
		cross, crossSize := computeCrossAlign(c.Align, crossItemSize, crossContent, crossPad)

		var li LayoutItem
		li.Box = item.Box
		if isRow {
			li.X, li.Y = main, cross
			li.Width, li.Height = mainSizes[i], crossSize
		} else {
			li.X, li.Y = cross, main
			li.Width, li.Height = crossSize, mainSizes[i]
		}
		items = append(items, li)
		main += mainSizes[i] + gap
	}

	return items
}

// layoutRow computes the layout for horizontally arranged children using flexbox-like behavior.
// It distributes available width among children based on their flex-grow/shrink properties and gaps.
func (c *Container) layoutRow(contentWidth, contentHeight int) []LayoutItem {
	return c.layoutAxis(true, contentWidth, contentHeight)
}

// layoutColumn computes the layout for vertically arranged children using flexbox-like behavior.
// It distributes available height among children based on their flex-grow/shrink properties and gaps.
func (c *Container) layoutColumn(contentWidth, contentHeight int) []LayoutItem {
	return c.layoutAxis(false, contentWidth, contentHeight)
}
