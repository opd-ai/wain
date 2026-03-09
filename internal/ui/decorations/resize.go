// Package decorations provides client-side window decoration widgets.
package decorations

import (
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/displaylist"
)

// ResizeEdge indicates which edge or corner is being resized.
type ResizeEdge int

// Resize edge constants identify which edge or corner is being resized.
const (
	// ResizeEdgeNone indicates no resize in progress.
	ResizeEdgeNone ResizeEdge = iota
	// ResizeEdgeTop indicates resizing from the top edge.
	ResizeEdgeTop
	// ResizeEdgeBottom indicates resizing from the bottom edge.
	ResizeEdgeBottom
	// ResizeEdgeLeft indicates resizing from the left edge.
	ResizeEdgeLeft
	// ResizeEdgeRight indicates resizing from the right edge.
	ResizeEdgeRight
	// ResizeEdgeTopLeft indicates resizing from the top-left corner.
	ResizeEdgeTopLeft
	// ResizeEdgeTopRight indicates resizing from the top-right corner.
	ResizeEdgeTopRight
	// ResizeEdgeBottomLeft indicates resizing from the bottom-left corner.
	ResizeEdgeBottomLeft
	// ResizeEdgeBottomRight indicates resizing from the bottom-right corner.
	ResizeEdgeBottomRight
)

// String returns a human-readable description of the resize edge.
func (e ResizeEdge) String() string {
	switch e {
	case ResizeEdgeNone:
		return "none"
	case ResizeEdgeTop:
		return "top"
	case ResizeEdgeBottom:
		return "bottom"
	case ResizeEdgeLeft:
		return "left"
	case ResizeEdgeRight:
		return "right"
	case ResizeEdgeTopLeft:
		return "top-left"
	case ResizeEdgeTopRight:
		return "top-right"
	case ResizeEdgeBottomLeft:
		return "bottom-left"
	case ResizeEdgeBottomRight:
		return "bottom-right"
	default:
		return "unknown"
	}
}

// ResizeHandles provides resize handle zones for window edges and corners.
type ResizeHandles struct {
	width       int
	height      int
	handleWidth int
	theme       *Theme
	hoverEdge   ResizeEdge
}

// NewResizeHandles creates a new resize handles object.
func NewResizeHandles(width, height int) *ResizeHandles {
	theme := DefaultDecorationTheme()
	return &ResizeHandles{
		width:       width,
		height:      height,
		handleWidth: theme.ResizeHandleWidth,
		theme:       theme,
		hoverEdge:   ResizeEdgeNone,
	}
}

// SetTheme sets the theme for the resize handles.
func (r *ResizeHandles) SetTheme(theme *Theme) {
	r.theme = theme
	r.handleWidth = theme.ResizeHandleWidth
}

// Resize updates the dimensions.
func (r *ResizeHandles) Resize(width, height int) {
	r.width = width
	r.height = height
}

// HitTest determines which resize edge (if any) is at the given coordinates.
// Coordinates are relative to the window frame (including decorations).
func (r *ResizeHandles) HitTest(x, y int) ResizeEdge {
	if edge := r.checkCorner(x, y); edge != ResizeEdgeNone {
		return edge
	}
	return r.checkEdge(x, y)
}

// checkCorner tests if coordinates hit a corner resize handle.
func (r *ResizeHandles) checkCorner(x, y int) ResizeEdge {
	hw := r.handleWidth
	if x < hw && y < hw {
		return ResizeEdgeTopLeft
	}
	if x >= r.width-hw && y < hw {
		return ResizeEdgeTopRight
	}
	if x < hw && y >= r.height-hw {
		return ResizeEdgeBottomLeft
	}
	if x >= r.width-hw && y >= r.height-hw {
		return ResizeEdgeBottomRight
	}
	return ResizeEdgeNone
}

// checkEdge tests if coordinates hit a straight edge resize handle.
func (r *ResizeHandles) checkEdge(x, y int) ResizeEdge {
	hw := r.handleWidth
	if y < hw {
		return ResizeEdgeTop
	}
	if y >= r.height-hw {
		return ResizeEdgeBottom
	}
	if x < hw {
		return ResizeEdgeLeft
	}
	if x >= r.width-hw {
		return ResizeEdgeRight
	}
	return ResizeEdgeNone
}

// HandlePointerEnter is called when the pointer enters a resize zone.
func (r *ResizeHandles) HandlePointerEnter(edge ResizeEdge) {
	r.hoverEdge = edge
}

// HandlePointerLeave is called when the pointer leaves the resize zone.
func (r *ResizeHandles) HandlePointerLeave() {
	r.hoverEdge = ResizeEdgeNone
}

// Draw renders the resize handles to a buffer.
// Only draws visible handles in hover state for visual feedback.
func (r *ResizeHandles) Draw(buf *primitives.Buffer, x, y int) error {
	// Resize handles are typically invisible unless hovered.
	// We draw them only in hover state for visual feedback.
	if r.hoverEdge == ResizeEdgeNone {
		return nil
	}

	color := r.theme.ResizeHandleColor
	hw := r.handleWidth

	switch r.hoverEdge {
	case ResizeEdgeTop:
		buf.FillRect(x, y, r.width, hw, color)
	case ResizeEdgeBottom:
		buf.FillRect(x, y+r.height-hw, r.width, hw, color)
	case ResizeEdgeLeft:
		buf.FillRect(x, y, hw, r.height, color)
	case ResizeEdgeRight:
		buf.FillRect(x+r.width-hw, y, hw, r.height, color)
	case ResizeEdgeTopLeft:
		buf.FillRect(x, y, hw, hw, color)
	case ResizeEdgeTopRight:
		buf.FillRect(x+r.width-hw, y, hw, hw, color)
	case ResizeEdgeBottomLeft:
		buf.FillRect(x, y+r.height-hw, hw, hw, color)
	case ResizeEdgeBottomRight:
		buf.FillRect(x+r.width-hw, y+r.height-hw, hw, hw, color)
	}

	return nil
}

// RenderToDisplayList renders the resize handles to a display list.
func (r *ResizeHandles) RenderToDisplayList(dl *displaylist.DisplayList, x, y int) {
	// Only render in hover state
	if r.hoverEdge == ResizeEdgeNone {
		return
	}

	color := r.theme.ResizeHandleColor
	hw := r.handleWidth

	switch r.hoverEdge {
	case ResizeEdgeTop:
		dl.AddFillRect(x, y, r.width, hw, color)
	case ResizeEdgeBottom:
		dl.AddFillRect(x, y+r.height-hw, r.width, hw, color)
	case ResizeEdgeLeft:
		dl.AddFillRect(x, y, hw, r.height, color)
	case ResizeEdgeRight:
		dl.AddFillRect(x+r.width-hw, y, hw, r.height, color)
	case ResizeEdgeTopLeft:
		dl.AddFillRect(x, y, hw, hw, color)
	case ResizeEdgeTopRight:
		dl.AddFillRect(x+r.width-hw, y, hw, hw, color)
	case ResizeEdgeBottomLeft:
		dl.AddFillRect(x, y+r.height-hw, hw, hw, color)
	case ResizeEdgeBottomRight:
		dl.AddFillRect(x+r.width-hw, y+r.height-hw, hw, hw, color)
	}
}
