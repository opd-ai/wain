package wain

import (
	"image"
	"image/color"
	"math"

	"github.com/opd-ai/wain/internal/raster/composite"
	"github.com/opd-ai/wain/internal/raster/effects"
	"github.com/opd-ai/wain/internal/raster/primitives"
	textpkg "github.com/opd-ai/wain/internal/raster/text"
	"github.com/opd-ai/wain/internal/ui/widgets"
)

// widgetAdapter bridges PublicWidget to the internal Widget interface.
// This allows PublicWidget instances to be used where internal Widget is expected.
type widgetAdapter struct {
	public PublicWidget
}

// newWidgetAdapter creates an adapter that wraps a PublicWidget.
func newWidgetAdapter(public PublicWidget) *widgetAdapter {
	return &widgetAdapter{
		public: public,
	}
}

// Bounds returns the widget's dimensions.
func (a *widgetAdapter) Bounds() (width, height int) {
	return a.public.Bounds()
}

// HandlePointerEnter is called when the pointer enters the widget.
func (a *widgetAdapter) HandlePointerEnter() {
	evt := &PointerEvent{
		eventType: PointerEnter,
	}
	a.public.HandleEvent(evt)
}

// HandlePointerLeave is called when the pointer leaves the widget.
func (a *widgetAdapter) HandlePointerLeave() {
	evt := &PointerEvent{
		eventType: PointerLeave,
	}
	a.public.HandleEvent(evt)
}

// HandlePointerDown is called when a pointer button is pressed.
func (a *widgetAdapter) HandlePointerDown(button uint32) {
	evt := &PointerEvent{
		eventType: PointerButtonPress,
		button:    PointerButton(button),
	}
	a.public.HandleEvent(evt)
}

// HandlePointerUp is called when a pointer button is released.
func (a *widgetAdapter) HandlePointerUp(button uint32) {
	evt := &PointerEvent{
		eventType: PointerButtonRelease,
		button:    PointerButton(button),
	}
	a.public.HandleEvent(evt)
}

// Draw renders the widget to the buffer at the specified position.
// This bridges the public Canvas-based drawing to internal Buffer-based drawing.
func (a *widgetAdapter) Draw(buf *primitives.Buffer, x, y int) error {
	// Create a buffer-backed Canvas that offsets all drawing by (x, y)
	canvas := newBufferCanvas(buf, x, y)

	// Let the public widget draw to the canvas
	a.public.Draw(canvas)

	return nil
}

// bufferCanvas implements Canvas by drawing directly to a primitives.Buffer.
// It translates the public Canvas API calls into buffer drawing operations.
type bufferCanvas struct {
	buf  *primitives.Buffer
	xOff int
	yOff int
}

// newBufferCanvas creates a Canvas that draws to a buffer with an offset.
func newBufferCanvas(buf *primitives.Buffer, xOff, yOff int) Canvas {
	return &bufferCanvas{
		buf:  buf,
		xOff: xOff,
		yOff: yOff,
	}
}

// FillRect fills a solid rectangle.
func (c *bufferCanvas) FillRect(x, y, width, height int, color Color) {
	c.buf.FillRect(c.xOff+x, c.yOff+y, width, height, color.toInternal())
}

// FillRoundedRect fills a rounded rectangle.
func (c *bufferCanvas) FillRoundedRect(x, y, width, height, radius int, color Color) {
	c.buf.FillRoundedRect(c.xOff+x, c.yOff+y, width, height, float64(radius), color.toInternal())
}

// DrawLine draws a line segment.
func (c *bufferCanvas) DrawLine(x1, y1, x2, y2 int, color Color, thickness int) {
	c.buf.DrawLine(c.xOff+x1, c.yOff+y1, c.xOff+x2, c.yOff+y2, float64(thickness), color.toInternal())
}

// DrawText renders text.
func (c *bufferCanvas) DrawText(txt string, x, y int, font *Font, color Color) {
	if font == nil || font.atlas == nil {
		return
	}

	textpkg.DrawText(
		c.buf,
		txt,
		float64(c.xOff+x),
		float64(c.yOff+y),
		font.size,
		color.toInternal(),
		font.atlas,
	)
}

// DrawImage renders an image by converting it to a primitives.Buffer
// and compositing it into the canvas buffer using bilinear scaling.
func (c *bufferCanvas) DrawImage(img *Image, x, y, width, height int) {
	if img == nil || img.data == nil || c.buf == nil {
		return
	}
	src := imageToBuffer(img.data)
	if src == nil {
		return
	}
	srcW, srcH := img.width, img.height
	composite.BlitScaled(c.buf, c.xOff+x, c.yOff+y, width, height, src, 0, 0, srcW, srcH)
}

// LinearGradient fills a rectangle with a linear gradient.
func (c *bufferCanvas) LinearGradient(x, y, width, height int, startColor, endColor Color, angle float64) {
	rad := angle * math.Pi / 180
	cx, cy := float64(c.xOff+x+width/2), float64(c.yOff+y+height/2)
	hw, hh := float64(width)/2, float64(height)/2
	x0 := int(cx - hw*math.Cos(rad))
	y0 := int(cy - hh*math.Sin(rad))
	x1 := int(cx + hw*math.Cos(rad))
	y1 := int(cy + hh*math.Sin(rad))
	effects.LinearGradient(c.buf, c.xOff+x, c.yOff+y, width, height,
		x0, y0, startColor.toInternal(),
		x1, y1, endColor.toInternal())
}

// RadialGradient fills a rectangle with a radial gradient.
func (c *bufferCanvas) RadialGradient(x, y, width, height int, centerColor, edgeColor Color) {
	cx := c.xOff + x + width/2
	cy := c.yOff + y + height/2
	r := width / 2
	if height < width {
		r = height / 2
	}
	effects.RadialGradient(c.buf, c.xOff+x, c.yOff+y, width, height, cx, cy, r,
		centerColor.toInternal(), edgeColor.toInternal())
}

// BoxShadow renders a box shadow around the given rectangle.
func (c *bufferCanvas) BoxShadow(x, y, width, height, offsetX, offsetY, blur int, clr Color) {
	effects.BoxShadow(c.buf, c.xOff+x+offsetX, c.yOff+y+offsetY, width, height, blur, clr.toInternal())
}

// imageToBuffer converts a standard Go image.Image to a primitives.Buffer in
// ARGB8888 format (little-endian byte order: B, G, R, A per pixel).
func imageToBuffer(img image.Image) *primitives.Buffer {
	b := img.Bounds()
	w, h := b.Max.X-b.Min.X, b.Max.Y-b.Min.Y
	if w <= 0 || h <= 0 {
		return nil
	}
	buf, err := primitives.NewBuffer(w, h)
	if err != nil {
		return nil
	}
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+px, b.Min.Y+py)).(color.NRGBA)
			idx := py*buf.Stride + px*4
			buf.Pixels[idx] = c.B
			buf.Pixels[idx+1] = c.G
			buf.Pixels[idx+2] = c.R
			buf.Pixels[idx+3] = c.A
		}
	}
	return buf
}

// Button is a clickable button widget with text and onClick callback.
//
// Button provides visual feedback for hover and press states, and supports
// custom styling via themes. Text is centered within the button bounds.
//
// Example usage:
//
//	btn := wain.NewButton("Submit", wain.Size{Width: 30, Height: 8})
//	btn.OnClick(func() {
//	    fmt.Println("Button clicked!")
//	})
//	panel.Add(btn)
type Button struct {
	BasePublicWidget
	internal *widgets.Button
	size     Size
	onClick  func()
}

// NewButton creates a new button with the specified text and percentage-based size.
//
// The size parameter specifies Width and Height as percentages (0-100) of
// the parent container. The button uses the default theme initially.
func NewButton(text string, size Size) *Button {
	internal := widgets.NewButton(text, 100, 30)
	btn := &Button{
		BasePublicWidget: NewBasePublicWidget(100, 30),
		internal:         internal,
		size:             size,
	}
	return btn
}

// OnClick registers a callback function to be invoked when the button is clicked.
func (b *Button) OnClick(handler func()) {
	b.onClick = handler
	b.internal.SetOnClick(handler)
}

// SetText changes the button's displayed text.
func (b *Button) SetText(text string) {
	b.internal.SetText(text)
}

// Text returns the button's current text.
func (b *Button) Text() string {
	return b.internal.Text()
}

// SetEnabled enables or disables the button.
// Disabled buttons do not respond to clicks and display with a disabled appearance.
func (b *Button) SetEnabled(enabled bool) {
	b.internal.SetEnabled(enabled)
}

// Draw renders the button to the canvas.
func (b *Button) Draw(c Canvas) {
	// Drawing is handled by the render bridge which calls the internal widget's
	// RenderToDisplayList method
	_ = c
}

// HandleEvent processes pointer events for button interaction.
func (b *Button) HandleEvent(evt Event) bool {
	switch e := evt.(type) {
	case *PointerEvent:
		switch e.EventType() {
		case PointerEnter:
			b.internal.HandlePointerEnter()
			return true
		case PointerLeave:
			b.internal.HandlePointerLeave()
			return true
		case PointerButtonPress:
			// Convert PointerButton to internal button number (1=left, 2=middle, 3=right)
			btn := buttonToInternal(e.Button())
			b.internal.HandlePointerDown(btn)
			return true
		case PointerButtonRelease:
			btn := buttonToInternal(e.Button())
			b.internal.HandlePointerUp(btn)
			// onClick is already called by internal button if enabled
			return true
		}
	}
	return false
}

// SetStyle applies a style override to this button.
//
// The override allows customizing specific visual properties while inheriting
// others from the theme. Any field left nil in the override will use the
// theme's value.
//
// Example:
//
//	accent := wain.RGB(100, 200, 100)
//	btn.SetStyle(wain.StyleOverride{Accent: &accent})
func (b *Button) SetStyle(override StyleOverride) {
	// Placeholder - actual implementation will be added when integrating
	// theme system with internal widgets
	_ = override
}

// buttonToInternal converts PointerButton to internal button number.
func buttonToInternal(btn PointerButton) uint32 {
	switch btn {
	case PointerButtonLeft:
		return 1
	case PointerButtonMiddle:
		return 2
	case PointerButtonRight:
		return 3
	default:
		return 0
	}
}

// Label is a static text display widget.
//
// Label renders text with optional styling. It does not respond to user input.
// Use Label for headings, descriptions, and other static text elements.
//
// Example usage:
//
//	label := wain.NewLabel("Welcome", wain.Size{Width: 100, Height: 5})
//	label.SetTextColor(wain.RGB(255, 255, 255))
//	panel.Add(label)
type Label struct {
	BasePublicWidget
	text      string
	size      Size
	textColor Color
	fontSize  int
}

// NewLabel creates a new label with the specified text and percentage-based size.
func NewLabel(text string, size Size) *Label {
	return &Label{
		BasePublicWidget: NewBasePublicWidget(100, 20),
		text:             text,
		size:             size,
		textColor:        RGB(30, 30, 30),
		fontSize:         14,
	}
}

// SetText changes the label's displayed text.
func (l *Label) SetText(text string) {
	l.text = text
}

// Text returns the label's current text.
func (l *Label) Text() string {
	return l.text
}

// SetTextColor sets the color of the label's text.
func (l *Label) SetTextColor(color Color) {
	l.textColor = color
}

// SetFontSize sets the font size in pixels.
func (l *Label) SetFontSize(size int) {
	l.fontSize = size
}

// Draw renders the label text to the canvas.
func (l *Label) Draw(c Canvas) {
	if l.text == "" {
		return
	}
	x, y := l.Position()
	// Label rendering will be integrated with the render bridge
	_ = c
	_ = x
	_ = y
}

// SetStyle applies a style override to this label.
//
// The override allows customizing specific visual properties while inheriting
// others from the theme.
func (l *Label) SetStyle(override StyleOverride) {
	_ = override
}

// TextInput is a single-line editable text field with cursor.
//
// TextInput supports text entry, selection, cursor positioning, and onChange
// callbacks. It provides visual feedback for focus state.
//
// Example usage:
//
//	input := wain.NewTextInput("", wain.Size{Width: 50, Height: 6})
//	input.SetPlaceholder("Enter your name...")
//	input.OnChange(func(text string) {
//	    fmt.Println("Input changed:", text)
//	})
//	panel.Add(input)
type TextInput struct {
	BasePublicWidget
	internal    *widgets.TextInput
	size        Size
	onChange    func(string)
	placeholder string
}

// NewTextInput creates a new text input field with initial text and percentage-based size.
func NewTextInput(initialText string, size Size) *TextInput {
	internal := widgets.NewTextInput("", 100, 30)
	internal.SetText(initialText)
	return &TextInput{
		BasePublicWidget: NewBasePublicWidget(100, 30),
		internal:         internal,
		size:             size,
	}
}

// OnChange registers a callback to be invoked when the text changes.
func (t *TextInput) OnChange(handler func(string)) {
	t.onChange = handler
	t.internal.SetOnChange(handler)
}

// SetText changes the input's text content.
func (t *TextInput) SetText(text string) {
	t.internal.SetText(text)
	if t.onChange != nil {
		t.onChange(text)
	}
}

// Text returns the current text content.
func (t *TextInput) Text() string {
	return t.internal.Text()
}

// SetPlaceholder sets the placeholder text shown when the input is empty.
func (t *TextInput) SetPlaceholder(placeholder string) {
	t.placeholder = placeholder
	t.internal.SetPlaceholder(placeholder)
}

// SetFocus sets the keyboard focus state of the text input.
func (t *TextInput) SetFocus(focused bool) {
	if focused {
		t.internal.HandleFocus()
	} else {
		t.internal.HandleBlur()
	}
}

// Draw renders the text input to the canvas.
func (t *TextInput) Draw(c Canvas) {
	// Drawing is handled by the render bridge
	_ = c
}

// HandleEvent processes keyboard and pointer events for text input.
func (t *TextInput) HandleEvent(evt Event) bool {
	switch e := evt.(type) {
	case *KeyEvent:
		if e.EventType() == KeyPress {
			// Get the character from Rune() accessor
			text := string(e.Rune())
			t.internal.HandleKeyPress(int(e.Key()), text)
			return true
		}
	case *PointerEvent:
		if e.EventType() == PointerButtonPress {
			btn := buttonToInternal(e.Button())
			t.internal.HandlePointerDown(btn)
			return true
		}
	}
	return false
}

// SetStyle applies a style override to this text input.
func (t *TextInput) SetStyle(override StyleOverride) {
	_ = override
}

// ScrollView is a scrollable container for overflow content.
//
// ScrollView displays a subset of its content area and provides vertical
// scrolling when content exceeds the visible height. Scrolling can be
// controlled via mouse wheel, touch gestures, or programmatically.
//
// Example usage:
//
//	scroll := wain.NewScrollView(wain.Size{Width: 100, Height: 80})
//	for i := 0; i < 50; i++ {
//	    scroll.Add(wain.NewLabel(fmt.Sprintf("Item %d", i), wain.Size{Width: 100, Height: 5}))
//	}
//	panel.Add(scroll)
type ScrollView struct {
	BasePublicWidget
	internal *widgets.ScrollContainer
	size     Size
	onScroll func(offset int)
}

// NewScrollView creates a new scrollable container with percentage-based size.
func NewScrollView(size Size) *ScrollView {
	internal := widgets.NewScrollContainer(100, 200)
	return &ScrollView{
		BasePublicWidget: NewBasePublicWidget(100, 200),
		internal:         internal,
		size:             size,
	}
}

// OnScroll registers a callback invoked when the scroll offset changes.
func (s *ScrollView) OnScroll(handler func(offset int)) {
	s.onScroll = handler
}

// SetScrollOffset sets the current scroll position in pixels.
func (s *ScrollView) SetScrollOffset(offset int) {
	s.internal.SetScrollOffset(offset)
	if s.onScroll != nil {
		s.onScroll(offset)
	}
}

// ScrollOffset returns the current scroll position in pixels.
func (s *ScrollView) ScrollOffset() int {
	return s.internal.ScrollOffset()
}

// Draw renders the visible portion of the scroll view to the canvas.
func (s *ScrollView) Draw(c Canvas) {
	// Drawing is handled by the render bridge with clipping
	_ = c
}

// HandleEvent processes scroll events (mouse wheel, touch gestures).
func (s *ScrollView) HandleEvent(evt Event) bool {
	switch e := evt.(type) {
	case *PointerEvent:
		if e.EventType() == PointerScroll {
			currentOffset := s.internal.ScrollOffset()
			newOffset := currentOffset + int(e.Value()*20)
			s.SetScrollOffset(newOffset)
			return true
		}
	}
	return false
}

// Add appends a child widget to the scroll view's content area.
func (s *ScrollView) Add(child PublicWidget) {
	// Wrap the public widget in an adapter that implements the internal Widget interface
	adapter := newWidgetAdapter(child)
	s.internal.AddChild(adapter)

	// Also track in the public widget's children list for consistency
	s.children = append(s.children, child)
}

// SetStyle applies a style override to this scroll view.
func (s *ScrollView) SetStyle(override StyleOverride) {
	_ = override
}

// ImageWidget displays an image resource.
//
// ImageWidget renders an image loaded via LoadImage. The image is scaled
// to fit the widget's dimensions while maintaining aspect ratio.
//
// Example usage:
//
//	img := wain.LoadImage("icon.png")
//	imageWidget := wain.NewImageWidget(img, wain.Size{Width: 20, Height: 20})
//	panel.Add(imageWidget)
type ImageWidget struct {
	BasePublicWidget
	image *Image
	size  Size
}

// NewImageWidget creates a new image display widget with percentage-based size.
func NewImageWidget(img *Image, size Size) *ImageWidget {
	return &ImageWidget{
		BasePublicWidget: NewBasePublicWidget(100, 100),
		image:            img,
		size:             size,
	}
}

// SetImage changes the displayed image.
func (iw *ImageWidget) SetImage(img *Image) {
	iw.image = img
}

// Image returns the currently displayed image.
func (iw *ImageWidget) Image() *Image {
	return iw.image
}

// Draw renders the image to the canvas, scaled to fit the widget bounds.
func (iw *ImageWidget) Draw(c Canvas) {
	if iw.image == nil {
		return
	}
	x, y := iw.Position()
	w, h := iw.Bounds()
	c.DrawImage(iw.image, x, y, w, h)
}

// SetStyle applies a style override to this image widget.
func (iw *ImageWidget) SetStyle(override StyleOverride) {
	_ = override
}

// Spacer is an invisible widget that consumes percentage space.
//
// Spacer is used for layout alignment and spacing. It participates in
// layout computation but does not render anything. Use Spacer to create
// flexible spacing between widgets.
//
// Example usage:
//
//	row := wain.NewRow()
//	row.Add(wain.NewButton("Left", wain.Size{Width: 20, Height: 10}))
//	row.Add(wain.NewSpacer(wain.Size{Width: 60, Height: 10}))  // Push next button to the right
//	row.Add(wain.NewButton("Right", wain.Size{Width: 20, Height: 10}))
type Spacer struct {
	BasePublicWidget
	size Size
}

// NewSpacer creates a new invisible spacer widget with percentage-based size.
func NewSpacer(size Size) *Spacer {
	return &Spacer{
		BasePublicWidget: NewBasePublicWidget(100, 10),
		size:             size,
	}
}

// Draw does nothing for spacers (they are invisible).
func (s *Spacer) Draw(c Canvas) {
	// Spacers are invisible
}

// SetStyle applies a style override to this spacer (does nothing - spacers are invisible).
func (s *Spacer) SetStyle(override StyleOverride) {
	_ = override
}
