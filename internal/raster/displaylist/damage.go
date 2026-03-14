package displaylist

// Rect represents a rectangular region.
type Rect struct {
	X, Y          int
	Width, Height int
}

// DamageTracker tracks dirty regions that need re-rendering.
type DamageTracker struct {
	regions []Rect
}

// NewDamageTracker creates a new damage tracker.
func NewDamageTracker() *DamageTracker {
	return &DamageTracker{
		regions: make([]Rect, 0, 8),
	}
}

// AddRect adds a rectangular damage region.
func (dt *DamageTracker) AddRect(x, y, width, height int) {
	dt.regions = append(dt.regions, Rect{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	})
}

// AddRectStruct adds a damage region from a Rect.
func (dt *DamageTracker) AddRectStruct(r Rect) {
	dt.regions = append(dt.regions, r)
}

// Regions returns the list of damage regions.
func (dt *DamageTracker) Regions() []Rect {
	return dt.regions
}

// Clear clears all damage regions.
func (dt *DamageTracker) Clear() {
	dt.regions = dt.regions[:0]
}

// IsEmpty returns true if there are no damage regions.
func (dt *DamageTracker) IsEmpty() bool {
	return len(dt.regions) == 0
}

// growMergedRegion merges region at index i with all overlapping or adjacent
// remaining regions (those not yet consumed), expanding the result rect until
// no further merges are possible.
func growMergedRegion(regions []Rect, used []bool, i, margin int) Rect {
	current := regions[i]
	changed := true
	for changed {
		changed = false
		for j := i + 1; j < len(regions); j++ {
			if used[j] {
				continue
			}
			if rectsOverlapOrClose(current, regions[j], margin) {
				current = mergeRects(current, regions[j])
				used[j] = true
				changed = true
			}
		}
	}
	return current
}

// Coalesce merges overlapping or adjacent damage regions to reduce the number of regions.
// This uses a simple greedy algorithm that merges regions that overlap or are close.
func (dt *DamageTracker) Coalesce(margin int) {
	if len(dt.regions) <= 1 {
		return
	}

	merged := make([]Rect, 0, len(dt.regions))
	used := make([]bool, len(dt.regions))

	for i := range dt.regions {
		if used[i] {
			continue
		}
		used[i] = true
		merged = append(merged, growMergedRegion(dt.regions, used, i, margin))
	}

	dt.regions = merged
}

// Bounds returns a single rect that encompasses all damage regions.
// Returns (0, 0, 0, 0) if there are no regions.
func (dt *DamageTracker) Bounds() Rect {
	if len(dt.regions) == 0 {
		return Rect{}
	}

	bounds := dt.regions[0]
	for i := 1; i < len(dt.regions); i++ {
		bounds = mergeRects(bounds, dt.regions[i])
	}
	return bounds
}

// rectsOverlapOrClose returns true if two rects overlap or are within margin pixels of each other.
func rectsOverlapOrClose(a, b Rect, margin int) bool {
	// Expand each rect by margin in all directions
	aX1 := a.X - margin
	aY1 := a.Y - margin
	aX2 := a.X + a.Width + margin
	aY2 := a.Y + a.Height + margin

	bX1 := b.X - margin
	bY1 := b.Y - margin
	bX2 := b.X + b.Width + margin
	bY2 := b.Y + b.Height + margin

	// Check if rects overlap
	return aX2 >= bX1 && bX2 >= aX1 && aY2 >= bY1 && bY2 >= aY1
}

// mergeRects returns a rect that encompasses both input rects.
func mergeRects(a, b Rect) Rect {
	x1 := min(a.X, b.X)
	y1 := min(a.Y, b.Y)
	x2 := max(a.X+a.Width, b.X+b.Width)
	y2 := max(a.Y+a.Height, b.Y+b.Height)

	return Rect{
		X:      x1,
		Y:      y1,
		Width:  x2 - x1,
		Height: y2 - y1,
	}
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ComputeDamageForCommand computes the damage rect for a single command.
func ComputeDamageForCommand(cmd DrawCommand) Rect {
	switch cmd.Type {
	case CmdFillRect:
		return damageFillRect(cmd.Data.(FillRectData))
	case CmdFillRoundedRect:
		return damageFillRoundedRect(cmd.Data.(FillRoundedRectData))
	case CmdDrawLine:
		return damageDrawLine(cmd.Data.(DrawLineData))
	case CmdDrawText:
		return damageDrawText(cmd.Data.(DrawTextData))
	case CmdLinearGradient:
		return damageLinearGradient(cmd.Data.(LinearGradientData))
	case CmdRadialGradient:
		return damageRadialGradient(cmd.Data.(RadialGradientData))
	case CmdBoxShadow:
		return damageBoxShadow(cmd.Data.(BoxShadowData))
	case CmdDrawImage:
		return damageDrawImage(cmd.Data.(DrawImageData))
	default:
		return Rect{}
	}
}

func damageFillRect(data FillRectData) Rect {
	return Rect{X: data.X, Y: data.Y, Width: data.Width, Height: data.Height}
}

func damageFillRoundedRect(data FillRoundedRectData) Rect {
	return Rect{X: data.X, Y: data.Y, Width: data.Width, Height: data.Height}
}

func damageDrawLine(data DrawLineData) Rect {
	x1 := min(data.X0, data.X1)
	y1 := min(data.Y0, data.Y1)
	x2 := max(data.X0, data.X1)
	y2 := max(data.Y0, data.Y1)
	w := data.Width / 2
	return Rect{X: x1 - w, Y: y1 - w, Width: x2 - x1 + 2*w, Height: y2 - y1 + 2*w}
}

func damageDrawText(data DrawTextData) Rect {
	width := len(data.Text) * data.FontSize / 2
	height := data.FontSize + data.FontSize/4
	return Rect{X: data.X, Y: data.Y - height, Width: width, Height: height}
}

func damageLinearGradient(data LinearGradientData) Rect {
	return Rect{X: data.X, Y: data.Y, Width: data.Width, Height: data.Height}
}

func damageRadialGradient(data RadialGradientData) Rect {
	return Rect{X: data.X, Y: data.Y, Width: data.Width, Height: data.Height}
}

func damageBoxShadow(data BoxShadowData) Rect {
	expand := data.BlurRadius + data.SpreadRadius
	return Rect{
		X:      data.X - expand,
		Y:      data.Y - expand,
		Width:  data.Width + 2*expand,
		Height: data.Height + 2*expand,
	}
}

func damageDrawImage(data DrawImageData) Rect {
	return Rect{X: data.X, Y: data.Y, Width: data.Width, Height: data.Height}
}

// FilterCommandsByDamage filters commands to only those that intersect with damage regions.
// Returns a new slice of commands that need to be rendered.
func FilterCommandsByDamage(commands []DrawCommand, damage []Rect) []DrawCommand {
	if len(damage) == 0 {
		return nil // No damage, nothing to render
	}

	filtered := make([]DrawCommand, 0, len(commands))

	for _, cmd := range commands {
		cmdRect := ComputeDamageForCommand(cmd)

		// Check if command intersects with any damage region
		for _, damageRect := range damage {
			if rectsIntersect(cmdRect, damageRect) {
				filtered = append(filtered, cmd)
				break
			}
		}
	}

	return filtered
}

// rectsIntersect returns true if two rects overlap.
func rectsIntersect(a, b Rect) bool {
	return a.X+a.Width > b.X &&
		b.X+b.Width > a.X &&
		a.Y+a.Height > b.Y &&
		b.Y+b.Height > a.Y
}
