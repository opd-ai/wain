// Package widgets implements core UI widgets including text input, buttons, and scroll containers.
//
// This package provides interactive UI widgets that integrate with the layout system
// and input handling. Widgets are renderer-agnostic and emit drawing commands for
// consumption by the rasterizer.
//
// # Widget Model
//
// Widgets follow a retained-mode architecture:
//   - State is maintained within widget instances
//   - Layout is computed separately via the layout package
//   - Rendering is delegated to the raster package
//   - Input events update widget state
//
// # Coordinate System
//
// Widgets use the same coordinate system as the layout and raster packages:
// origin (0,0) at top-left, X increases right, Y increases down.
//
// # Usage
//
// Create widgets, handle events, render to buffer:
//
//	btn := widgets.NewButton("Click me", 100, 30)
//	btn.HandlePointerClick(x, y)
//	btn.Draw(buffer, x, y)
package widgets

import (
	"errors"

	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/effects"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
)

var (
	// ErrInvalidDimensions is returned when widget dimensions are invalid.
	ErrInvalidDimensions = errors.New("widgets: invalid dimensions")

	// ErrNilBuffer is returned when a nil buffer is provided for rendering.
	ErrNilBuffer = errors.New("widgets: nil buffer")
)

// PointerState represents the state of a pointer device.
type PointerState int

const (
	// PointerStateNormal indicates no interaction.
	PointerStateNormal PointerState = iota

	// PointerStateHover indicates the pointer is hovering over the widget.
	PointerStateHover

	// PointerStatePressed indicates the widget is being pressed.
	PointerStatePressed
)

// Widget defines the common interface for all UI widgets.
type Widget interface {
	// Bounds returns the widget's dimensions.
	Bounds() (width, height int)

	// HandlePointerEnter is called when the pointer enters the widget.
	HandlePointerEnter()

	// HandlePointerLeave is called when the pointer leaves the widget.
	HandlePointerLeave()

	// HandlePointerDown is called when a pointer button is pressed.
	HandlePointerDown(button uint32)

	// HandlePointerUp is called when a pointer button is released.
	HandlePointerUp(button uint32)

	// Draw renders the widget to the buffer at the specified position.
	Draw(buf *primitives.Buffer, x, y int) error
}

// Theme defines the visual appearance of widgets.
type Theme struct {
	// Background colors
	BackgroundNormal   primitives.Color
	BackgroundHover    primitives.Color
	BackgroundPressed  primitives.Color
	BackgroundDisabled primitives.Color

	// Text colors
	TextNormal   primitives.Color
	TextHover    primitives.Color
	TextPressed  primitives.Color
	TextDisabled primitives.Color

	// Border colors
	BorderNormal  primitives.Color
	BorderHover   primitives.Color
	BorderPressed primitives.Color
	BorderFocus   primitives.Color

	// Scale factor for HiDPI displays (1.0 = normal, 2.0 = 2x)
	Scale float32

	// Border radius for rounded corners
	BorderRadius int

	// Shadow properties
	ShadowColor   primitives.Color
	ShadowBlur    int
	ShadowOffsetX int
	ShadowOffsetY int

	// Typography
	FontSize float64
}

// DefaultTheme returns the default widget theme.
func DefaultTheme() *Theme {
	return &Theme{
		BackgroundNormal:   primitives.Color{R: 240, G: 240, B: 240, A: 255},
		BackgroundHover:    primitives.Color{R: 230, G: 230, B: 230, A: 255},
		BackgroundPressed:  primitives.Color{R: 210, G: 210, B: 210, A: 255},
		BackgroundDisabled: primitives.Color{R: 200, G: 200, B: 200, A: 128},

		TextNormal:   primitives.Color{R: 30, G: 30, B: 30, A: 255},
		TextHover:    primitives.Color{R: 0, G: 0, B: 0, A: 255},
		TextPressed:  primitives.Color{R: 0, G: 0, B: 0, A: 255},
		TextDisabled: primitives.Color{R: 150, G: 150, B: 150, A: 255},

		BorderNormal:  primitives.Color{R: 180, G: 180, B: 180, A: 255},
		BorderHover:   primitives.Color{R: 120, G: 120, B: 120, A: 255},
		BorderPressed: primitives.Color{R: 100, G: 100, B: 100, A: 255},
		BorderFocus:   primitives.Color{R: 70, G: 130, B: 220, A: 255},

		BorderRadius: 4,

		ShadowColor:   primitives.Color{R: 0, G: 0, B: 0, A: 40},
		ShadowBlur:    4,
		ShadowOffsetX: 0,
		ShadowOffsetY: 2,

		FontSize: 14.0,
		Scale:    1.0, // Default to 1.0 scale
	}
}

// Button represents a clickable button widget.
type Button struct {
	text    string
	width   int
	height  int
	state   PointerState
	enabled bool
	theme   *Theme
	atlas   *text.Atlas
	onClick func()
}

// NewButton creates a new button with the specified text and dimensions.
func NewButton(text string, width, height int) *Button {
	return &Button{
		text:    text,
		width:   width,
		height:  height,
		state:   PointerStateNormal,
		enabled: true,
		theme:   DefaultTheme(),
		onClick: nil,
	}
}

// SetAtlas sets the font atlas for text rendering.
func (b *Button) SetAtlas(atlas *text.Atlas) {
	b.atlas = atlas
}

// SetTheme sets a custom theme for the button.
func (b *Button) SetTheme(theme *Theme) {
	b.theme = theme
}

// SetOnClick sets the callback function for click events.
func (b *Button) SetOnClick(onClick func()) {
	b.onClick = onClick
}

// SetEnabled enables or disables the button.
func (b *Button) SetEnabled(enabled bool) {
	b.enabled = enabled
	if !enabled {
		b.state = PointerStateNormal
	}
}

// Bounds returns the button's dimensions.
func (b *Button) Bounds() (width, height int) {
	return b.width, b.height
}

// HandlePointerEnter is called when the pointer enters the button.
func (b *Button) HandlePointerEnter() {
	if b.enabled && b.state == PointerStateNormal {
		b.state = PointerStateHover
	}
}

// HandlePointerLeave is called when the pointer leaves the button.
func (b *Button) HandlePointerLeave() {
	if b.enabled && b.state != PointerStatePressed {
		b.state = PointerStateNormal
	}
}

// HandlePointerDown is called when a pointer button is pressed.
func (b *Button) HandlePointerDown(button uint32) {
	if b.enabled && button == 1 { // Left button
		b.state = PointerStatePressed
	}
}

// HandlePointerUp is called when a pointer button is released.
func (b *Button) HandlePointerUp(button uint32) {
	if b.enabled && button == 1 && b.state == PointerStatePressed {
		b.state = PointerStateHover
		if b.onClick != nil {
			b.onClick()
		}
	}
}

// buttonColors holds the color scheme for a button in its current state.
type buttonColors struct {
	background primitives.Color
	text       primitives.Color
	border     primitives.Color
}

// getButtonColors returns the appropriate colors based on button state.
func (b *Button) getButtonColors() buttonColors {
	if !b.enabled {
		return buttonColors{
			background: b.theme.BackgroundDisabled,
			text:       b.theme.TextDisabled,
			border:     b.theme.BorderNormal,
		}
	}
	switch b.state {
	case PointerStatePressed:
		return buttonColors{
			background: b.theme.BackgroundPressed,
			text:       b.theme.TextPressed,
			border:     b.theme.BorderPressed,
		}
	case PointerStateHover:
		return buttonColors{
			background: b.theme.BackgroundHover,
			text:       b.theme.TextHover,
			border:     b.theme.BorderHover,
		}
	default:
		return buttonColors{
			background: b.theme.BackgroundNormal,
			text:       b.theme.TextNormal,
			border:     b.theme.BorderNormal,
		}
	}
}

// Draw renders the button to the buffer at the specified position.
// Draw renders the button to the buffer at the specified position.
func (b *Button) Draw(buf *primitives.Buffer, x, y int) error {
	if buf == nil {
		return ErrNilBuffer
	}

	colors := b.getButtonColors()

	// Draw shadow if enabled
	if b.enabled && b.theme.ShadowBlur > 0 {
		shadowX := x + b.theme.ShadowOffsetX
		shadowY := y + b.theme.ShadowOffsetY
		effects.BoxShadow(buf, shadowX, shadowY, b.width, b.height,
			b.theme.ShadowBlur, b.theme.ShadowColor)
	}

	// Draw background with rounded corners
	buf.FillRoundedRect(x, y, b.width, b.height, float64(b.theme.BorderRadius), colors.background)

	// Draw border
	drawRectBorder(buf, x, y, b.width, b.height, 1, colors.border)

	// Draw text centered
	if b.atlas != nil && b.text != "" {
		textWidth := b.measureTextWidth(b.text)
		textX := float64(x) + float64(b.width-textWidth)/2
		textY := float64(y) + float64(b.height)/2 + b.theme.FontSize/3
		text.DrawText(buf, b.text, textX, textY, b.theme.FontSize, colors.text, b.atlas)
	}

	return nil
}

// RenderToDisplayList emits draw commands for the button to a display list.
func (b *Button) RenderToDisplayList(dl *displaylist.DisplayList, x, y int) {
	if dl == nil {
		return
	}

	colors := b.getButtonColors()

	// Emit shadow command if enabled
	if b.enabled && b.theme.ShadowBlur > 0 {
		shadowX := x + b.theme.ShadowOffsetX
		shadowY := y + b.theme.ShadowOffsetY
		dl.AddBoxShadow(shadowX, shadowY, b.width, b.height,
			b.theme.ShadowBlur, 0, b.theme.ShadowColor)
	}

	// Emit background with rounded corners
	dl.AddFillRoundedRect(x, y, b.width, b.height, b.theme.BorderRadius, colors.background)

	// Emit border
	emitRectBorder(dl, x, y, b.width, b.height, 1, colors.border)

	// Emit text centered
	if b.atlas != nil && b.text != "" {
		textWidth := b.measureTextWidth(b.text)
		textX := x + (b.width-textWidth)/2
		textY := y + b.height/2 + int(b.theme.FontSize/3)
		dl.AddDrawText(b.text, textX, textY, int(b.theme.FontSize), colors.text, 0)
	}
}

// Text returns the button's current text.
func (b *Button) Text() string {
	return b.text
}

// SetText changes the button's text.
func (b *Button) SetText(text string) {
	b.text = text
}

// Theme returns the button's current theme.
func (b *Button) Theme() *Theme {
	return b.theme
}

// measureTextWidth estimates the width of text in pixels.
func (b *Button) measureTextWidth(s string) int {
	if b.atlas == nil {
		return 0
	}
	width := 0.0
	scale := b.theme.FontSize / b.atlas.Baseline
	for _, r := range s {
		if glyph, err := b.atlas.GetGlyph(r); err == nil {
			width += glyph.Advance * scale
		}
	}
	return int(width)
}

// drawRectBorder draws a rectangle border using lines.
func drawRectBorder(buf *primitives.Buffer, x, y, width, height, lineWidth int, color primitives.Color) {
	w := float64(lineWidth)
	// Top edge
	buf.DrawLine(x, y, x+width, y, w, color)
	// Right edge
	buf.DrawLine(x+width, y, x+width, y+height, w, color)
	// Bottom edge
	buf.DrawLine(x+width, y+height, x, y+height, w, color)
	// Left edge
	buf.DrawLine(x, y+height, x, y, w, color)
}

// emitRectBorder emits border draw commands to a display list.
func emitRectBorder(dl *displaylist.DisplayList, x, y, width, height, lineWidth int, color primitives.Color) {
	// Top edge
	dl.AddDrawLine(x, y, x+width, y, lineWidth, color)
	// Right edge
	dl.AddDrawLine(x+width, y, x+width, y+height, lineWidth, color)
	// Bottom edge
	dl.AddDrawLine(x+width, y+height, x, y+height, lineWidth, color)
	// Left edge
	dl.AddDrawLine(x, y+height, x, y, lineWidth, color)
}

// TextInput represents a single-line text input field.
type TextInput struct {
	text        string
	placeholder string
	width       int
	height      int
	cursorPos   int
	focused     bool
	enabled     bool
	theme       *Theme
	atlas       *text.Atlas
	onChange    func(string)
}

// NewTextInput creates a new text input field.
func NewTextInput(placeholder string, width, height int) *TextInput {
	return &TextInput{
		text:        "",
		placeholder: placeholder,
		width:       width,
		height:      height,
		cursorPos:   0,
		focused:     false,
		enabled:     true,
		theme:       DefaultTheme(),
		onChange:    nil,
	}
}

// SetAtlas sets the font atlas for text rendering.
func (t *TextInput) SetAtlas(atlas *text.Atlas) {
	t.atlas = atlas
}

// SetTheme sets a custom theme for the text input.
func (t *TextInput) SetTheme(theme *Theme) {
	t.theme = theme
}

// SetOnChange sets the callback function for text change events.
func (t *TextInput) SetOnChange(onChange func(string)) {
	t.onChange = onChange
}

// SetEnabled enables or disables the text input.
func (t *TextInput) SetEnabled(enabled bool) {
	t.enabled = enabled
	if !enabled {
		t.focused = false
	}
}

// SetText sets the text content.
func (t *TextInput) SetText(text string) {
	t.text = text
	if t.cursorPos > len(t.text) {
		t.cursorPos = len(t.text)
	}
}

// Text returns the current text content.
func (t *TextInput) Text() string {
	return t.text
}

// Bounds returns the text input's dimensions.
func (t *TextInput) Bounds() (width, height int) {
	return t.width, t.height
}

// HandlePointerEnter is called when the pointer enters the text input.
func (t *TextInput) HandlePointerEnter() {
}

// HandlePointerLeave is called when the pointer leaves the text input.
func (t *TextInput) HandlePointerLeave() {
}

// HandlePointerDown is called when a pointer button is pressed.
func (t *TextInput) HandlePointerDown(button uint32) {
	if t.enabled && button == 1 {
		t.focused = true
	}
}

// HandlePointerUp is called when a pointer button is released.
func (t *TextInput) HandlePointerUp(button uint32) {
}

// HandleKeyPress processes keyboard input.
// key is the key code, text is the printable character (if any).
func (t *TextInput) HandleKeyPress(key int, text string) {
	if !t.enabled || !t.focused {
		return
	}

	// Insert character at cursor position if text is provided
	if text != "" {
		t.text = t.text[:t.cursorPos] + text + t.text[t.cursorPos:]
		t.cursorPos += len(text)
		if t.onChange != nil {
			t.onChange(t.text)
		}
	}
}

// HandleBackspace processes backspace key.
func (t *TextInput) HandleBackspace() {
	if !t.enabled || !t.focused || t.cursorPos == 0 {
		return
	}

	t.text = t.text[:t.cursorPos-1] + t.text[t.cursorPos:]
	t.cursorPos--
	if t.onChange != nil {
		t.onChange(t.text)
	}
}

// HandleDelete processes delete key.
func (t *TextInput) HandleDelete() {
	if !t.enabled || !t.focused || t.cursorPos >= len(t.text) {
		return
	}

	t.text = t.text[:t.cursorPos] + t.text[t.cursorPos+1:]
	if t.onChange != nil {
		t.onChange(t.text)
	}
}

// HandleCursorMove moves the cursor position.
func (t *TextInput) HandleCursorMove(delta int) {
	if !t.enabled || !t.focused {
		return
	}

	t.cursorPos += delta
	if t.cursorPos < 0 {
		t.cursorPos = 0
	}
	if t.cursorPos > len(t.text) {
		t.cursorPos = len(t.text)
	}
}

// HandleFocus sets the input to focused state.
func (t *TextInput) HandleFocus() {
	t.focused = true
}

// HandleBlur removes focus from the input.
func (t *TextInput) HandleBlur() {
	t.focused = false
}

// textInputDisplay holds the display text and color for rendering.
type textInputDisplay struct {
	text  string
	color primitives.Color
}

// getDisplayText returns the text to display and its color.
func (t *TextInput) getDisplayText() textInputDisplay {
	if t.text != "" {
		color := t.theme.TextNormal
		if !t.enabled {
			color = t.theme.TextDisabled
		}
		return textInputDisplay{text: t.text, color: color}
	}
	if t.placeholder != "" {
		return textInputDisplay{
			text:  t.placeholder,
			color: primitives.Color{R: 150, G: 150, B: 150, A: 255},
		}
	}
	return textInputDisplay{text: "", color: t.theme.TextNormal}
}

// getBackgroundColor returns the background color based on enabled state.
func (t *TextInput) getBackgroundColor() primitives.Color {
	if t.enabled {
		return t.theme.BackgroundNormal
	}
	return t.theme.BackgroundDisabled
}

// getBorderColor returns the border color based on focus state.
func (t *TextInput) getBorderColor() primitives.Color {
	if t.focused {
		return t.theme.BorderFocus
	}
	return t.theme.BorderNormal
}

// Draw renders the text input to the buffer at the specified position.
// Draw renders the text input to the buffer at the specified position.
func (t *TextInput) Draw(buf *primitives.Buffer, x, y int) error {
	if buf == nil {
		return ErrNilBuffer
	}

	// Draw background
	buf.FillRoundedRect(x, y, t.width, t.height, float64(t.theme.BorderRadius), t.getBackgroundColor())

	// Draw border
	drawRectBorder(buf, x, y, t.width, t.height, 1, t.getBorderColor())

	// Draw text or placeholder
	display := t.getDisplayText()
	if t.atlas != nil && display.text != "" {
		padding := 8
		textX := float64(x + padding)
		textY := float64(y) + float64(t.height)/2 + t.theme.FontSize/3
		text.DrawText(buf, display.text, textX, textY, t.theme.FontSize, display.color, t.atlas)
	}

	// Draw cursor if focused
	if t.focused && t.enabled {
		cursorX := t.getCursorX(x)
		cursorY0 := y + 4
		cursorY1 := y + t.height - 4
		buf.DrawLine(cursorX, cursorY0, cursorX, cursorY1, 1.0, t.theme.TextNormal)
	}

	return nil
}

// RenderToDisplayList emits draw commands for the text input to a display list.
func (t *TextInput) RenderToDisplayList(dl *displaylist.DisplayList, x, y int) {
	if dl == nil {
		return
	}

	// Emit background
	dl.AddFillRoundedRect(x, y, t.width, t.height, t.theme.BorderRadius, t.getBackgroundColor())

	// Emit border
	emitRectBorder(dl, x, y, t.width, t.height, 1, t.getBorderColor())

	// Emit text or placeholder
	display := t.getDisplayText()
	if t.atlas != nil && display.text != "" {
		padding := 8
		textX := x + padding
		textY := y + t.height/2 + int(t.theme.FontSize/3)
		dl.AddDrawText(display.text, textX, textY, int(t.theme.FontSize), display.color, 0)
	}

	// Emit cursor if focused
	if t.focused && t.enabled {
		cursorX := t.getCursorX(x)
		cursorY0 := y + 4
		cursorY1 := y + t.height - 4
		dl.AddDrawLine(cursorX, cursorY0, cursorX, cursorY1, 1, t.theme.TextNormal)
	}
}

// getCursorX calculates the X position of the cursor.
func (t *TextInput) getCursorX(baseX int) int {
	if t.atlas == nil {
		return baseX + 8
	}

	padding := 8
	x := float64(baseX + padding)
	scale := t.theme.FontSize / t.atlas.Baseline

	for i, r := range t.text {
		if i >= t.cursorPos {
			break
		}
		if glyph, err := t.atlas.GetGlyph(r); err == nil {
			x += glyph.Advance * scale
		}
	}

	return int(x)
}

// ScrollContainer represents a scrollable container widget.
type ScrollContainer struct {
	width         int
	height        int
	contentHeight int
	scrollOffset  int
	theme         *Theme
	children      []Widget
}

// NewScrollContainer creates a new scroll container.
func NewScrollContainer(width, height int) *ScrollContainer {
	return &ScrollContainer{
		width:         width,
		height:        height,
		contentHeight: 0,
		scrollOffset:  0,
		theme:         DefaultTheme(),
		children:      make([]Widget, 0),
	}
}

// SetTheme sets a custom theme for the scroll container.
func (s *ScrollContainer) SetTheme(theme *Theme) {
	s.theme = theme
}

// AddChild adds a widget to the container.
func (s *ScrollContainer) AddChild(child Widget) {
	s.children = append(s.children, child)
	s.updateContentHeight()
}

// updateContentHeight recalculates the total content height.
func (s *ScrollContainer) updateContentHeight() {
	totalHeight := 0
	for _, child := range s.children {
		_, h := child.Bounds()
		totalHeight += h
	}
	s.contentHeight = totalHeight
}

// Bounds returns the scroll container's dimensions.
func (s *ScrollContainer) Bounds() (width, height int) {
	return s.width, s.height
}

// HandlePointerEnter is called when the pointer enters the container.
func (s *ScrollContainer) HandlePointerEnter() {
}

// HandlePointerLeave is called when the pointer leaves the container.
func (s *ScrollContainer) HandlePointerLeave() {
}

// HandlePointerDown is called when a pointer button is pressed.
func (s *ScrollContainer) HandlePointerDown(button uint32) {
}

// HandlePointerUp is called when a pointer button is released.
func (s *ScrollContainer) HandlePointerUp(button uint32) {
}

// HandleScroll processes scroll events.
func (s *ScrollContainer) HandleScroll(delta int) {
	s.scrollOffset += delta

	maxScroll := s.contentHeight - s.height
	if maxScroll < 0 {
		maxScroll = 0
	}

	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
	if s.scrollOffset > maxScroll {
		s.scrollOffset = maxScroll
	}
}

// ScrollOffset returns the current scroll position.
func (s *ScrollContainer) ScrollOffset() int {
	return s.scrollOffset
}

// SetScrollOffset sets the scroll position.
func (s *ScrollContainer) SetScrollOffset(offset int) {
	maxScroll := s.contentHeight - s.height
	if maxScroll < 0 {
		maxScroll = 0
	}

	s.scrollOffset = offset
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
	if s.scrollOffset > maxScroll {
		s.scrollOffset = maxScroll
	}
}

// Draw renders the scroll container to the buffer at the specified position.
func (s *ScrollContainer) Draw(buf *primitives.Buffer, x, y int) error {
	if buf == nil {
		return ErrNilBuffer
	}

	// Draw background
	bgColor := s.theme.BackgroundNormal
	buf.FillRect(x, y, s.width, s.height, bgColor)

	// Draw border
	drawRectBorder(buf, x, y, s.width, s.height, 1, s.theme.BorderNormal)

	// Draw children with clipping
	childY := y - s.scrollOffset
	for _, child := range s.children {
		_, h := child.Bounds()

		// Only draw if visible
		if childY+h > y && childY < y+s.height {
			child.Draw(buf, x, childY)
		}

		childY += h
	}

	// Draw scrollbar if content overflows
	if s.contentHeight > s.height {
		s.drawScrollbar(buf, x, y)
	}

	return nil
}

// drawScrollbar draws a vertical scrollbar.
func (s *ScrollContainer) drawScrollbar(buf *primitives.Buffer, x, y int) {
	barWidth := 8
	barX := x + s.width - barWidth - 2

	// Calculate scrollbar thumb size and position
	thumbRatio := float64(s.height) / float64(s.contentHeight)
	thumbHeight := int(float64(s.height) * thumbRatio)
	if thumbHeight < 20 {
		thumbHeight = 20
	}

	scrollRatio := float64(s.scrollOffset) / float64(s.contentHeight-s.height)
	thumbY := y + int(scrollRatio*float64(s.height-thumbHeight))

	// Draw scrollbar track
	trackColor := primitives.Color{R: 230, G: 230, B: 230, A: 255}
	buf.FillRect(barX, y, barWidth, s.height, trackColor)

	// Draw scrollbar thumb
	thumbColor := primitives.Color{R: 180, G: 180, B: 180, A: 255}
	buf.FillRoundedRect(barX, thumbY, barWidth, thumbHeight, 4.0, thumbColor)
}

// RenderToDisplayList emits draw commands for the scroll container to a display list.
func (s *ScrollContainer) RenderToDisplayList(dl *displaylist.DisplayList, x, y int) {
	if dl == nil {
		return
	}

	// Emit background
	bgColor := s.theme.BackgroundNormal
	dl.AddFillRect(x, y, s.width, s.height, bgColor)

	// Emit border
	emitRectBorder(dl, x, y, s.width, s.height, 1, s.theme.BorderNormal)

	// Emit children with offset
	childY := y - s.scrollOffset
	for _, child := range s.children {
		_, h := child.Bounds()

		// Only emit if visible
		if childY+h > y && childY < y+s.height {
			// Check if child supports RenderToDisplayList
			if renderer, ok := child.(interface {
				RenderToDisplayList(*displaylist.DisplayList, int, int)
			}); ok {
				renderer.RenderToDisplayList(dl, x, childY)
			}
		}

		childY += h
	}

	// Emit scrollbar if content overflows
	if s.contentHeight > s.height {
		s.emitScrollbar(dl, x, y)
	}
}

// emitScrollbar emits scrollbar draw commands to a display list.
func (s *ScrollContainer) emitScrollbar(dl *displaylist.DisplayList, x, y int) {
	barWidth := 8
	barX := x + s.width - barWidth - 2

	// Calculate scrollbar thumb size and position
	thumbRatio := float64(s.height) / float64(s.contentHeight)
	thumbHeight := int(float64(s.height) * thumbRatio)
	if thumbHeight < 20 {
		thumbHeight = 20
	}

	scrollRatio := float64(s.scrollOffset) / float64(s.contentHeight-s.height)
	thumbY := y + int(scrollRatio*float64(s.height-thumbHeight))

	// Emit scrollbar track
	trackColor := primitives.Color{R: 230, G: 230, B: 230, A: 255}
	dl.AddFillRect(barX, y, barWidth, s.height, trackColor)

	// Emit scrollbar thumb
	thumbColor := primitives.Color{R: 180, G: 180, B: 180, A: 255}
	dl.AddFillRoundedRect(barX, thumbY, barWidth, thumbHeight, 4, thumbColor)
}
