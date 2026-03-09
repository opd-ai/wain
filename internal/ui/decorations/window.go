// Package decorations provides client-side window decoration widgets.
package decorations

import (
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
)

// WindowFrame combines a title bar and resize handles for complete window decorations.
type WindowFrame struct {
	width         int
	height        int
	contentWidth  int
	contentHeight int
	theme         *Theme
	titleBar      *TitleBar
	resizeHandles *ResizeHandles
}

// NewWindowFrame creates a new window frame with title bar and resize handles.
// width and height include the decorations (title bar + resize handles).
func NewWindowFrame(title string, width, height int) *WindowFrame {
	theme := DefaultDecorationTheme()
	titleBarHeight := theme.TitleBarHeight
	handleWidth := theme.ResizeHandleWidth

	// Content area excludes title bar and resize handles
	contentWidth := width - 2*handleWidth
	contentHeight := height - titleBarHeight - 2*handleWidth

	titleBar := NewTitleBar(title, width, titleBarHeight)
	titleBar.SetTheme(theme)

	resizeHandles := NewResizeHandles(width, height)
	resizeHandles.SetTheme(theme)

	return &WindowFrame{
		width:         width,
		height:        height,
		contentWidth:  contentWidth,
		contentHeight: contentHeight,
		theme:         theme,
		titleBar:      titleBar,
		resizeHandles: resizeHandles,
	}
}

// SetTheme sets the theme for all decoration components.
func (w *WindowFrame) SetTheme(theme *Theme) {
	w.theme = theme
	w.titleBar.SetTheme(theme)
	w.resizeHandles.SetTheme(theme)
}

// SetAtlas sets the text atlas for title rendering.
func (w *WindowFrame) SetAtlas(atlas *text.Atlas) {
	w.titleBar.SetAtlas(atlas)
}

// SetTitle updates the window title.
func (w *WindowFrame) SetTitle(title string) {
	w.titleBar.SetTitle(title)
}

// Bounds returns the total frame dimensions (including decorations).
func (w *WindowFrame) Bounds() (int, int) {
	return w.width, w.height
}

// ContentBounds returns the usable content area dimensions (excluding decorations).
func (w *WindowFrame) ContentBounds() (int, int) {
	return w.contentWidth, w.contentHeight
}

// ContentOffset returns the x, y offset of the content area.
func (w *WindowFrame) ContentOffset() (int, int) {
	handleWidth := w.theme.ResizeHandleWidth
	titleBarHeight := w.theme.TitleBarHeight
	return handleWidth, titleBarHeight + handleWidth
}

// Resize updates the frame dimensions.
func (w *WindowFrame) Resize(width, height int) {
	w.width = width
	w.height = height

	handleWidth := w.theme.ResizeHandleWidth
	titleBarHeight := w.theme.TitleBarHeight

	w.contentWidth = width - 2*handleWidth
	w.contentHeight = height - titleBarHeight - 2*handleWidth

	w.titleBar.Resize(width)
	w.resizeHandles.Resize(width, height)
}

// HitTestResize determines if the pointer is over a resize handle.
func (w *WindowFrame) HitTestResize(x, y int) ResizeEdge {
	return w.resizeHandles.HitTest(x, y)
}

// HitTestTitleBarButton determines if the pointer is over a title bar button.
// Coordinates are relative to the window frame.
func (w *WindowFrame) HitTestTitleBarButton(x, y int) *WindowButton {
	handleWidth := w.theme.ResizeHandleWidth

	// Adjust coordinates to title bar space
	tbX := x - handleWidth
	tbY := y - handleWidth

	return w.titleBar.HitTest(tbX, tbY)
}

// IsTitleBarDragArea returns true if the pointer is in the title bar drag area.
// Coordinates are relative to the window frame.
func (w *WindowFrame) IsTitleBarDragArea(x, y int) bool {
	handleWidth := w.theme.ResizeHandleWidth
	titleBarHeight := w.theme.TitleBarHeight

	// Check if within title bar bounds
	if y < handleWidth || y >= handleWidth+titleBarHeight {
		return false
	}
	if x < handleWidth || x >= w.width-handleWidth {
		return false
	}

	// Check if not over a button
	button := w.HitTestTitleBarButton(x, y)
	return button == nil
}

// HandlePointerMotion handles pointer motion events.
// Returns (isResize, edge) if over resize handle, (isDrag, none) if in drag area.
func (w *WindowFrame) HandlePointerMotion(x, y int) (bool, ResizeEdge) {
	edge := w.HitTestResize(x, y)
	if edge != ResizeEdgeNone {
		w.resizeHandles.HandlePointerEnter(edge)
		return true, edge
	}

	w.resizeHandles.HandlePointerLeave()
	return false, ResizeEdgeNone
}

// TitleBar returns the title bar widget for direct event handling.
func (w *WindowFrame) TitleBar() *TitleBar {
	return w.titleBar
}

// Draw renders the window frame to a buffer.
func (w *WindowFrame) Draw(buf *primitives.Buffer, x, y int) error {
	handleWidth := w.theme.ResizeHandleWidth

	// Draw resize handles
	if err := w.resizeHandles.Draw(buf, x, y); err != nil {
		return err
	}

	// Draw title bar (offset by handle width)
	if err := w.titleBar.Draw(buf, x+handleWidth, y+handleWidth); err != nil {
		return err
	}

	return nil
}

// RenderToDisplayList renders the window frame to a display list.
func (w *WindowFrame) RenderToDisplayList(dl *displaylist.DisplayList, x, y int) {
	handleWidth := w.theme.ResizeHandleWidth

	// Render resize handles
	w.resizeHandles.RenderToDisplayList(dl, x, y)

	// Render title bar (offset by handle width)
	w.titleBar.RenderToDisplayList(dl, x+handleWidth, y+handleWidth)
}
