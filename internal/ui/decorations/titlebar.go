// Package decorations provides client-side window decoration widgets.
package decorations

import (
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/ui/widgets"
)

// ButtonType represents the type of window control button.
type ButtonType int

// Button type constants for window control buttons.
const (
	// ButtonTypeClose represents the window close button.
	ButtonTypeClose ButtonType = iota
	// ButtonTypeMaximize represents the window maximize button.
	ButtonTypeMaximize
	// ButtonTypeMinimize represents the window minimize button.
	ButtonTypeMinimize
)

// WindowButton represents a window control button (close, minimize, maximize).
type WindowButton struct {
	buttonType ButtonType
	size       int
	state      widgets.PointerState
	theme      *Theme
}

// NewWindowButton creates a new window control button.
func NewWindowButton(buttonType ButtonType, size int) *WindowButton {
	return &WindowButton{
		buttonType: buttonType,
		size:       size,
		state:      widgets.PointerStateNormal,
		theme:      DefaultDecorationTheme(),
	}
}

// SetTheme sets the theme for the button.
func (b *WindowButton) SetTheme(theme *Theme) {
	b.theme = theme
}

// Bounds returns the button dimensions.
func (b *WindowButton) Bounds() (int, int) {
	return b.size, b.size
}

// HandlePointerEnter is called when the pointer enters the button.
func (b *WindowButton) HandlePointerEnter() {
	b.state = widgets.PointerStateHover
}

// HandlePointerLeave is called when the pointer leaves the button.
func (b *WindowButton) HandlePointerLeave() {
	b.state = widgets.PointerStateNormal
}

// HandlePointerDown is called when a pointer button is pressed.
func (b *WindowButton) HandlePointerDown(button uint32) {
	b.state = widgets.PointerStatePressed
}

// HandlePointerUp is called when a pointer button is released.
func (b *WindowButton) HandlePointerUp(button uint32) {
	b.state = widgets.PointerStateHover
}

// getStateColors returns background and foreground colors based on button state.
func (b *WindowButton) getStateColors() (bg, fg primitives.Color) {
	bg = b.theme.ButtonBackgroundNormal
	fg = b.theme.ButtonForegroundNormal

	switch b.state {
	case widgets.PointerStateHover:
		bg = b.theme.ButtonBackgroundHover
		fg = b.theme.ButtonForegroundHover
	case widgets.PointerStatePressed:
		bg = b.theme.ButtonBackgroundPressed
		fg = b.theme.ButtonForegroundPressed
	}
	return bg, fg
}

// Draw renders the button to a buffer.
func (b *WindowButton) Draw(buf *primitives.Buffer, x, y int) error {
	bg, fg := b.getStateColors()

	// Draw background
	buf.FillRect(x, y, b.size, b.size, bg)

	// Draw icon based on button type
	iconSize := b.size / 3
	iconX := x + (b.size-iconSize)/2
	iconY := y + (b.size-iconSize)/2

	switch b.buttonType {
	case ButtonTypeClose:
		// Draw X
		buf.DrawLine(iconX, iconY, iconX+iconSize, iconY+iconSize, 2, fg)
		buf.DrawLine(iconX+iconSize, iconY, iconX, iconY+iconSize, 2, fg)
	case ButtonTypeMaximize:
		// Draw square
		buf.DrawLine(iconX, iconY, iconX+iconSize, iconY, 2, fg)
		buf.DrawLine(iconX+iconSize, iconY, iconX+iconSize, iconY+iconSize, 2, fg)
		buf.DrawLine(iconX+iconSize, iconY+iconSize, iconX, iconY+iconSize, 2, fg)
		buf.DrawLine(iconX, iconY+iconSize, iconX, iconY, 2, fg)
	case ButtonTypeMinimize:
		// Draw horizontal line
		lineY := iconY + iconSize/2
		buf.DrawLine(iconX, lineY, iconX+iconSize, lineY, 2, fg)
	}

	return nil
}

// RenderToDisplayList renders the button to a display list.
func (b *WindowButton) RenderToDisplayList(dl *displaylist.DisplayList, x, y int) {
	bg, fg := b.getStateColors()

	// Background
	dl.AddFillRect(x, y, b.size, b.size, bg)

	// Icon
	iconSize := b.size / 3
	iconX := x + (b.size-iconSize)/2
	iconY := y + (b.size-iconSize)/2

	switch b.buttonType {
	case ButtonTypeClose:
		dl.AddDrawLine(iconX, iconY, iconX+iconSize, iconY+iconSize, 2, fg)
		dl.AddDrawLine(iconX+iconSize, iconY, iconX, iconY+iconSize, 2, fg)
	case ButtonTypeMaximize:
		dl.AddDrawLine(iconX, iconY, iconX+iconSize, iconY, 2, fg)
		dl.AddDrawLine(iconX+iconSize, iconY, iconX+iconSize, iconY+iconSize, 2, fg)
		dl.AddDrawLine(iconX+iconSize, iconY+iconSize, iconX, iconY+iconSize, 2, fg)
		dl.AddDrawLine(iconX, iconY+iconSize, iconX, iconY, 2, fg)
	case ButtonTypeMinimize:
		lineY := iconY + iconSize/2
		dl.AddDrawLine(iconX, lineY, iconX+iconSize, lineY, 2, fg)
	}
}

// TitleBar represents a window title bar with control buttons.
type TitleBar struct {
	title     string
	width     int
	height    int
	theme     *Theme
	atlas     *text.Atlas
	closeBtn  *WindowButton
	maxBtn    *WindowButton
	minBtn    *WindowButton
	dragStart *struct{ x, y int }
	dragging  bool
}

// NewTitleBar creates a new title bar.
func NewTitleBar(title string, width, height int) *TitleBar {
	theme := DefaultDecorationTheme()
	buttonSize := height - 8

	return &TitleBar{
		title:    title,
		width:    width,
		height:   height,
		theme:    theme,
		closeBtn: NewWindowButton(ButtonTypeClose, buttonSize),
		maxBtn:   NewWindowButton(ButtonTypeMaximize, buttonSize),
		minBtn:   NewWindowButton(ButtonTypeMinimize, buttonSize),
	}
}

// SetTheme sets the theme for the title bar and its buttons.
func (t *TitleBar) SetTheme(theme *Theme) {
	t.theme = theme
	t.closeBtn.SetTheme(theme)
	t.maxBtn.SetTheme(theme)
	t.minBtn.SetTheme(theme)
}

// SetAtlas sets the text atlas for rendering text.
func (t *TitleBar) SetAtlas(atlas *text.Atlas) {
	t.atlas = atlas
}

// SetTitle updates the title text.
func (t *TitleBar) SetTitle(title string) {
	t.title = title
}

// Bounds returns the title bar dimensions.
func (t *TitleBar) Bounds() (int, int) {
	return t.width, t.height
}

// Resize updates the title bar width.
func (t *TitleBar) Resize(width int) {
	t.width = width
}

// HandlePointerEnter is called when the pointer enters the title bar.
func (t *TitleBar) HandlePointerEnter() {}

// HandlePointerLeave is called when the pointer leaves the title bar.
func (t *TitleBar) HandlePointerLeave() {}

// HandlePointerDown is called when a pointer button is pressed.
func (t *TitleBar) HandlePointerDown(button uint32) {}

// HandlePointerUp is called when a pointer button is released.
func (t *TitleBar) HandlePointerUp(button uint32) {}

// HandlePointerMotion handles pointer motion for dragging.
func (t *TitleBar) HandlePointerMotion(x, y int) (bool, int, int) {
	if t.dragging && t.dragStart != nil {
		dx := x - t.dragStart.x
		dy := y - t.dragStart.y
		t.dragStart.x = x
		t.dragStart.y = y
		return true, dx, dy
	}
	return false, 0, 0
}

// StartDrag begins a window drag operation.
func (t *TitleBar) StartDrag(x, y int) {
	t.dragging = true
	t.dragStart = &struct{ x, y int }{x, y}
}

// StopDrag ends the window drag operation.
func (t *TitleBar) StopDrag() {
	t.dragging = false
	t.dragStart = nil
}

// buttonPositions calculates the x-coordinates for minimize, maximize, and close buttons.
func (t *TitleBar) buttonPositions() (minX, maxX, closeX int) {
	buttonSize, _ := t.closeBtn.Bounds()
	spacing := t.theme.ButtonSpacing
	rightEdge := t.width - spacing

	closeX = rightEdge - buttonSize
	maxX = closeX - buttonSize - spacing
	minX = maxX - buttonSize - spacing
	return minX, maxX, closeX
}

// HitTest returns which button (if any) was hit at the given coordinates.
func (t *TitleBar) HitTest(x, y int) *WindowButton {
	if y < 0 || y >= t.height {
		return nil
	}

	buttonSize, _ := t.closeBtn.Bounds()
	spacing := t.theme.ButtonSpacing
	minX, maxX, closeX := t.buttonPositions()

	// Close button (rightmost)
	if x >= closeX && x < t.width-spacing && y >= spacing && y < spacing+buttonSize {
		return t.closeBtn
	}

	// Maximize button
	if x >= maxX && x < closeX-spacing && y >= spacing && y < spacing+buttonSize {
		return t.maxBtn
	}

	// Minimize button
	if x >= minX && x < maxX-spacing && y >= spacing && y < spacing+buttonSize {
		return t.minBtn
	}

	return nil
}

// Draw renders the title bar to a buffer.
func (t *TitleBar) Draw(buf *primitives.Buffer, x, y int) error {
	// Draw title bar background
	buf.FillRect(x, y, t.width, t.height, t.theme.TitleBarBackground)

	// Draw separator line at bottom
	separatorY := y + t.height - 1
	buf.DrawLine(x, separatorY, x+t.width, separatorY, 1, t.theme.TitleBarBorder)

	// Draw title text
	if t.atlas != nil && t.title != "" {
		textX := x + t.theme.TitlePaddingX
		textY := y + (t.height-int(t.theme.TitleFontSize))/2
		text.DrawText(buf, t.title, float64(textX), float64(textY), t.theme.TitleFontSize, t.theme.TitleTextColor, t.atlas)
	}

	// Draw buttons
	minX, maxX, closeX := t.buttonPositions()
	spacing := t.theme.ButtonSpacing

	if err := t.minBtn.Draw(buf, x+minX, y+spacing); err != nil {
		return err
	}
	if err := t.maxBtn.Draw(buf, x+maxX, y+spacing); err != nil {
		return err
	}
	if err := t.closeBtn.Draw(buf, x+closeX, y+spacing); err != nil {
		return err
	}

	return nil
}

// RenderToDisplayList renders the title bar to a display list.
func (t *TitleBar) RenderToDisplayList(dl *displaylist.DisplayList, x, y int) {
	// Background
	dl.AddFillRect(x, y, t.width, t.height, t.theme.TitleBarBackground)

	// Separator line
	separatorY := y + t.height - 1
	dl.AddDrawLine(x, separatorY, x+t.width, separatorY, 1, t.theme.TitleBarBorder)

	// Title text
	if t.atlas != nil && t.title != "" {
		textX := x + t.theme.TitlePaddingX
		textY := y + (t.height-int(t.theme.TitleFontSize))/2
		dl.AddDrawText(t.title, textX, textY, int(t.theme.TitleFontSize), t.theme.TitleTextColor, 0)
	}

	// Buttons
	minX, maxX, closeX := t.buttonPositions()
	spacing := t.theme.ButtonSpacing

	t.minBtn.RenderToDisplayList(dl, x+minX, y+spacing)
	t.maxBtn.RenderToDisplayList(dl, x+maxX, y+spacing)
	t.closeBtn.RenderToDisplayList(dl, x+closeX, y+spacing)
}
