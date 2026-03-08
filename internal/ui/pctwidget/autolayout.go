package pctwidget

// FlowDirection controls the direction of the automatic layout flow.
type FlowDirection int

const (
	// FlowColumn arranges children top-to-bottom (default).
	FlowColumn FlowDirection = iota
	// FlowRow arranges children left-to-right.
	FlowRow
)

// AutoLayout computes positions for a slice of panels within a parent region.
//
// The engine resolves each child's percentage-based size against the parent
// dimensions and places children sequentially in the specified flow direction.
// Gaps between children and padding around the edges are derived from the
// supplied [Style] (or [DefaultStyle] if nil).
//
// Children that have a manual position override ([BaseWidget.IsManuallyPositioned])
// are left untouched and do not consume space in the flow.
//
// AutoLayout also recurses into each child's own children, providing
// zero-configuration nested layout.
func AutoLayout(panels []*Panel, parentX, parentY, parentW, parentH int, dir FlowDirection, style Style) {
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
			_, _, pw, ph := p.ResolvedBounds()
			layoutChildren(p, pw, ph, dir, style)
			continue
		}

		// Resolve percentage-based size against parent content area.
		if err := p.Resolve(cw, ch); err != nil {
			// If resolution fails (e.g., invalid parent size), clear dimensions
			// so we don't reuse stale values from a previous layout pass.
			p.width = 0
			p.height = 0
		}

		cursor = placePanel(p, cx, cy, cursor, gap, dir)

		// Recurse into children.
		layoutChildren(p, p.width, p.height, dir, style)
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
	return
}

// placePanel positions a panel based on flow direction and returns the updated cursor.
func placePanel(p *Panel, cx, cy, cursor, gap int, dir FlowDirection) int {
	switch dir {
	case FlowRow:
		p.x = cx + cursor
		p.y = cy
		return cursor + p.width + gap
	default: // FlowColumn or unknown
		p.x = cx
		p.y = cy + cursor
		return cursor + p.height + gap
	}
}

// clampToZero returns max(value, 0).
func clampToZero(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

// layoutChildren is a helper that runs AutoLayout on a panel's children using
// the panel's resolved position and dimensions as the parent region.
func layoutChildren(p *Panel, parentW, parentH int, dir FlowDirection, style Style) {
	if len(p.children) == 0 {
		return
	}
	AutoLayout(p.children, p.x, p.y, parentW, parentH, dir, style)
}
