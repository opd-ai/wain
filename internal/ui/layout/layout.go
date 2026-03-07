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

func (c *Container) layoutRow(contentWidth, contentHeight int) []LayoutItem {
	totalGaps := c.Gap * (len(c.children) - 1)
	if totalGaps < 0 {
		totalGaps = 0
	}

	// Compute base sizes
	var baseWidth int
	var totalGrow float64
	var totalShrink float64

	for _, item := range c.children {
		if item.FlexBasis > 0 {
			baseWidth += item.FlexBasis
		} else {
			baseWidth += item.Box.Width
		}
		totalGrow += item.FlexGrow
		totalShrink += item.FlexShrink
	}

	availableWidth := contentWidth - totalGaps
	remaining := availableWidth - baseWidth

	// Compute final widths
	widths := make([]int, len(c.children))
	for i, item := range c.children {
		basis := item.FlexBasis
		if basis == 0 {
			basis = item.Box.Width
		}

		if remaining > 0 && totalGrow > 0 {
			grow := int(float64(remaining) * (item.FlexGrow / totalGrow))
			widths[i] = basis + grow
		} else if remaining < 0 && totalShrink > 0 {
			shrink := int(float64(-remaining) * (item.FlexShrink / totalShrink))
			widths[i] = basis - shrink
			if widths[i] < 0 {
				widths[i] = 0
			}
		} else {
			widths[i] = basis
		}
	}

	// Compute starting position based on justification
	x := c.Padding.Left
	switch c.Justify {
	case JustifyCenter:
		totalWidth := 0
		for _, w := range widths {
			totalWidth += w
		}
		totalWidth += totalGaps
		x += (contentWidth - totalWidth) / 2
	case JustifyEnd:
		totalWidth := 0
		for _, w := range widths {
			totalWidth += w
		}
		totalWidth += totalGaps
		x += contentWidth - totalWidth
	case JustifySpaceBetween:
		if len(c.children) > 1 {
			totalWidth := 0
			for _, w := range widths {
				totalWidth += w
			}
			c.Gap = (contentWidth - totalWidth) / (len(c.children) - 1)
		}
	case JustifySpaceAround:
		totalWidth := 0
		for _, w := range widths {
			totalWidth += w
		}
		space := (contentWidth - totalWidth) / len(c.children)
		c.Gap = space
		x += space / 2
	}

	// Position children
	items := make([]LayoutItem, 0, len(c.children))
	for i, item := range c.children {
		height := item.Box.Height
		y := c.Padding.Top

		switch c.Align {
		case AlignCenter:
			y += (contentHeight - height) / 2
		case AlignEnd:
			y += contentHeight - height
		case AlignStretch:
			height = contentHeight
		}

		items = append(items, LayoutItem{
			Box:    item.Box,
			X:      x,
			Y:      y,
			Width:  widths[i],
			Height: height,
		})

		x += widths[i] + c.Gap
	}

	return items
}

func (c *Container) layoutColumn(contentWidth, contentHeight int) []LayoutItem {
	totalGaps := c.Gap * (len(c.children) - 1)
	if totalGaps < 0 {
		totalGaps = 0
	}

	// Compute base sizes
	var baseHeight int
	var totalGrow float64
	var totalShrink float64

	for _, item := range c.children {
		if item.FlexBasis > 0 {
			baseHeight += item.FlexBasis
		} else {
			baseHeight += item.Box.Height
		}
		totalGrow += item.FlexGrow
		totalShrink += item.FlexShrink
	}

	availableHeight := contentHeight - totalGaps
	remaining := availableHeight - baseHeight

	// Compute final heights
	heights := make([]int, len(c.children))
	for i, item := range c.children {
		basis := item.FlexBasis
		if basis == 0 {
			basis = item.Box.Height
		}

		if remaining > 0 && totalGrow > 0 {
			grow := int(float64(remaining) * (item.FlexGrow / totalGrow))
			heights[i] = basis + grow
		} else if remaining < 0 && totalShrink > 0 {
			shrink := int(float64(-remaining) * (item.FlexShrink / totalShrink))
			heights[i] = basis - shrink
			if heights[i] < 0 {
				heights[i] = 0
			}
		} else {
			heights[i] = basis
		}
	}

	// Compute starting position based on justification
	y := c.Padding.Top
	switch c.Justify {
	case JustifyCenter:
		totalHeight := 0
		for _, h := range heights {
			totalHeight += h
		}
		totalHeight += totalGaps
		y += (contentHeight - totalHeight) / 2
	case JustifyEnd:
		totalHeight := 0
		for _, h := range heights {
			totalHeight += h
		}
		totalHeight += totalGaps
		y += contentHeight - totalHeight
	case JustifySpaceBetween:
		if len(c.children) > 1 {
			totalHeight := 0
			for _, h := range heights {
				totalHeight += h
			}
			c.Gap = (contentHeight - totalHeight) / (len(c.children) - 1)
		}
	case JustifySpaceAround:
		totalHeight := 0
		for _, h := range heights {
			totalHeight += h
		}
		space := (contentHeight - totalHeight) / len(c.children)
		c.Gap = space
		y += space / 2
	}

	// Position children
	items := make([]LayoutItem, 0, len(c.children))
	for i, item := range c.children {
		width := item.Box.Width
		x := c.Padding.Left

		switch c.Align {
		case AlignCenter:
			x += (contentWidth - width) / 2
		case AlignEnd:
			x += contentWidth - width
		case AlignStretch:
			width = contentWidth
		}

		items = append(items, LayoutItem{
			Box:    item.Box,
			X:      x,
			Y:      y,
			Width:  width,
			Height: heights[i],
		})

		y += heights[i] + c.Gap
	}

	return items
}
