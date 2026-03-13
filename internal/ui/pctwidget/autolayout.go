package pctwidget

// FlowDirection controls the direction of the automatic layout flow.
type FlowDirection int

const (
	// FlowColumn arranges children top-to-bottom (default).
	FlowColumn FlowDirection = iota
	// FlowRow arranges children left-to-right.
	FlowRow
)

// Align controls cross-axis alignment of children within a container.
//
// For Row containers the cross axis is vertical; for Column containers it is
// horizontal.
type Align int

const (
	// AlignStart places children at the start of the cross axis (top for Row, left for Column).
	AlignStart Align = iota
	// AlignCenter centers children on the cross axis.
	AlignCenter
	// AlignEnd places children at the end of the cross axis (bottom for Row, right for Column).
	AlignEnd
	// AlignStretch stretches children to fill the cross axis entirely.
	AlignStretch
)

// AutoLayout computes positions for a slice of panels within a parent region.
//
// The engine resolves each child's percentage-based size against the parent
// dimensions and places children sequentially in the specified flow direction.
// Gaps between children and padding around the edges are derived from the
// supplied [Style] (or [DefaultStyle] if nil).
//
// The align parameter controls cross-axis alignment of children:
//   - [AlignStart] places children at the start of the cross axis (default).
//   - [AlignCenter] centers children on the cross axis.
//   - [AlignEnd] places children at the end of the cross axis.
//   - [AlignStretch] sizes children to fill the cross axis.
//
// Children that have a manual position override ([BaseWidget.IsManuallyPositioned])
// are left untouched and do not consume space in the flow.
//
// AutoLayout recurses into each child using the child's own [FlowDirection] and
// [Align], providing zero-configuration nested layout.
func AutoLayout(panels []*Panel, parentX, parentY, parentW, parentH int, dir FlowDirection, align Align, style Style) {
	if style == nil {
		style = DefaultStyle()
	}
	cx, cy, cw, ch := computeContentArea(parentX, parentY, parentW, parentH, style)
	gap := clampToZero(style.Gap())
	cursor := 0 // running offset along the main axis

	for _, p := range panels {
		if !p.Visible() {
			continue
		}
		if p.IsManuallyPositioned() {
			// Recurse into children using the manually set bounds.
			layoutChildren(p, style)
			continue
		}

		// Resolve percentage-based size against parent content area.
		if err := p.Resolve(cw, ch); err != nil {
			// If resolution fails (e.g., invalid parent size), clear dimensions
			// so we don't reuse stale values from a previous layout pass.
			p.width = 0
			p.height = 0
		}

		cursor = placePanel(p, cx, cy, cw, ch, cursor, gap, dir, align)

		// Recurse into children using the child's own layout settings.
		layoutChildren(p, style)
	}
}

// computeContentArea calculates the content area after applying padding.
func computeContentArea(parentX, parentY, parentW, parentH int, style Style) (cx, cy, cw, ch int) {
	pad := clampToZero(style.Padding())
	cx = parentX + pad
	cy = parentY + pad
	cw = parentW - 2*pad
	ch = parentH - 2*pad
	if cw < 0 {
		cw = 0
	}
	if ch < 0 {
		ch = 0
	}
	return cx, cy, cw, ch
}

// placePanel positions a panel on the main axis and applies cross-axis alignment.
// It returns the updated cursor (offset along the main axis).
func placePanel(p *Panel, cx, cy, cw, ch, cursor, gap int, dir FlowDirection, align Align) int {
	switch dir {
	case FlowRow:
		p.x = cx + cursor
		if align == AlignStretch {
			p.y = cy
			p.height = ch
		} else {
			p.y = crossAxisPosition(cy, p.height, ch, align)
		}
		return cursor + p.width + gap
	default: // FlowColumn or unknown
		p.y = cy + cursor
		if align == AlignStretch {
			p.x = cx
			p.width = cw
		} else {
			p.x = crossAxisPosition(cx, p.width, cw, align)
		}
		return cursor + p.height + gap
	}
}

// crossAxisPosition calculates the offset along the cross axis for a child.
func crossAxisPosition(containerStart, childSize, containerSize int, align Align) int {
	switch align {
	case AlignCenter:
		return containerStart + (containerSize-childSize)/2
	case AlignEnd:
		return containerStart + containerSize - childSize
	default: // AlignStart
		return containerStart
	}
}

// clampToZero returns max(value, 0).
func clampToZero(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

// layoutChildren runs AutoLayout on a panel's children using the panel's own
// resolved position, dimensions, flow direction, and cross-axis alignment.
func layoutChildren(p *Panel, style Style) {
	if len(p.children) == 0 {
		return
	}
	AutoLayout(p.children, p.x, p.y, p.width, p.height, p.flowDirection, p.align, style)
}
