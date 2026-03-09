package wain

import (
	"github.com/opd-ai/wain/internal/ui/widgets"
)

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
	// TODO: Add placeholder support to internal TextInput
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
	// TODO: Implement proper child management for ScrollView
	// This requires extending widgets.ScrollContainer to support PublicWidget children
	_ = child
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
