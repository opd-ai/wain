package wain

import "github.com/opd-ai/wain/internal/raster/displaylist"

// PublicWidget is the stable public interface for all UI widgets in wain.
//
// Unlike the internal widget system which exposes pointer-specific handlers
// and low-level buffer drawing, PublicWidget provides a simplified,
// stable contract for application developers. All concrete widget types
// implement this interface.
//
// PublicWidget abstracts over:
//   - Platform-specific event details (the internal system uses separate
//     HandlePointer/HandleKey/HandleTouch methods; PublicWidget uses a
//     unified HandleEvent method)
//   - Low-level rendering details (the internal system draws directly to
//     a core.Buffer; PublicWidget draws to a Canvas abstraction)
//   - Layout resolution (the internal system exposes Resolve methods and
//     manual position overrides; PublicWidget hides these)
//
// Example usage:
//
//	type MyWidget struct {
//	    wain.BasePublicWidget
//	    // ... custom fields
//	}
//
//	func (w *MyWidget) Draw(c Canvas) {
//	    // Custom drawing logic
//	}
type PublicWidget interface {
	// Bounds returns the current pixel dimensions of the widget.
	// The returned width and height are the resolved pixel size after
	// layout, scaling, and HiDPI adjustments.
	Bounds() (width, height int)

	// HandleEvent processes a user interaction event and returns true
	// if the event was consumed (stopping propagation to other widgets).
	// Returning false allows the event to continue propagating.
	HandleEvent(Event) bool

	// Draw renders the widget to the provided canvas.
	// The canvas provides a high-level drawing API that abstracts over
	// GPU and software rendering backends.
	Draw(Canvas)
}

// Container extends PublicWidget for widgets that can contain child widgets.
//
// Containers manage layout and rendering of their children. Examples include
// Row, Column, Stack, and Grid layout containers, as well as ScrollView and
// custom composite widgets.
//
// Example usage:
//
//	panel := wain.NewPanel(wain.Size{Width: 50, Height: 100})
//	button := wain.NewButton("Click me", wain.Size{Width: 30, Height: 10})
//	panel.Add(button)
type Container interface {
	PublicWidget

	// Add appends a child widget to this container.
	// The container takes ownership of layout computation for the child.
	// Children are laid out according to the container's layout algorithm
	// (e.g., Row distributes horizontally, Column vertically).
	Add(child PublicWidget)

	// Children returns a slice of the container's child widgets.
	// The order matches the add order and determines layout and z-order.
	// Modifying the returned slice does not affect the container's
	// internal state.
	Children() []PublicWidget
}

// Canvas provides a high-level drawing API for widget rendering.
//
// Canvas abstracts over the internal displaylist.DisplayList, providing
// a simpler, more stable API surface for application developers. Methods
// accept pixel coordinates and automatically handle GPU/software backend
// selection, HiDPI scaling, and display list emission.
//
// Canvas instances are provided by the framework during widget rendering;
// application code does not create Canvas instances directly.
//
// Example usage:
//
//	func (w *MyWidget) Draw(c Canvas) {
//	    x, y := w.Position()
//	    width, height := w.Bounds()
//	    c.FillRect(x, y, width, height, wain.RGB(40, 40, 60))
//	    c.DrawText("Hello", x+10, y+10, wain.DefaultFont(), wain.RGB(255, 255, 255))
//	}
type Canvas interface {
	// FillRect fills a solid rectangle at the given position and size.
	FillRect(x, y, width, height int, color Color)

	// FillRoundedRect fills a rounded rectangle with the specified corner radius.
	FillRoundedRect(x, y, width, height, radius int, color Color)

	// DrawLine draws a line segment from (x1, y1) to (x2, y2).
	DrawLine(x1, y1, x2, y2 int, color Color, thickness int)

	// DrawText renders text at the given position using the specified font and color.
	DrawText(text string, x, y int, font *Font, color Color)

	// DrawImage renders an image at the given position.
	// The image is scaled to fit the specified width and height.
	DrawImage(img *Image, x, y, width, height int)

	// LinearGradient fills a rectangle with a linear gradient from startColor to endColor.
	// The angle is in degrees (0 = left-to-right, 90 = top-to-bottom).
	LinearGradient(x, y, width, height int, startColor, endColor Color, angle float64)

	// RadialGradient fills a rectangle with a radial gradient from centerColor to edgeColor.
	RadialGradient(x, y, width, height int, centerColor, edgeColor Color)

	// BoxShadow renders a box shadow around the given rectangle.
	// offsetX and offsetY specify the shadow offset, blur is the blur radius.
	BoxShadow(x, y, width, height, offsetX, offsetY, blur int, color Color)
}

// BasePublicWidget provides default implementations for the PublicWidget interface.
//
// Application developers can embed BasePublicWidget in custom widget types
// to get default event handling and bounds management. Override Draw() to
// provide custom rendering.
//
// Example:
//
//	type ColoredPanel struct {
//	    wain.BasePublicWidget
//	    Color wain.Color
//	}
//
//	func (p *ColoredPanel) Draw(c wain.Canvas) {
//	    w, h := p.Bounds()
//	    x, y := p.Position()
//	    c.FillRect(x, y, w, h, p.Color)
//	}
type BasePublicWidget struct {
	x, y          int
	width, height int
	children      []PublicWidget
	visible       bool

	// Event handlers
	onEvent func(Event) bool
}

// NewBasePublicWidget creates a BasePublicWidget with the given pixel dimensions.
func NewBasePublicWidget(width, height int) BasePublicWidget {
	return BasePublicWidget{
		width:   width,
		height:  height,
		visible: true,
	}
}

// Bounds returns the current pixel dimensions of the widget.
func (w *BasePublicWidget) Bounds() (width, height int) {
	return w.width, w.height
}

// Position returns the current position of the widget in pixels.
// Position is a convenience method not required by the PublicWidget interface.
func (w *BasePublicWidget) Position() (x, y int) {
	return w.x, w.y
}

// HandleEvent processes an event. The default implementation invokes the
// registered event handler if one is set, otherwise returns false.
func (w *BasePublicWidget) HandleEvent(evt Event) bool {
	if w.onEvent != nil {
		return w.onEvent(evt)
	}
	return false
}

// Draw is a no-op. Override this in concrete widget types.
func (w *BasePublicWidget) Draw(c Canvas) {
	// Default implementation does nothing
}

// Add appends a child widget. This makes BasePublicWidget compatible with
// the Container interface when embedded in a container type.
func (w *BasePublicWidget) Add(child PublicWidget) {
	w.children = append(w.children, child)
}

// Children returns the list of child widgets.
func (w *BasePublicWidget) Children() []PublicWidget {
	return w.children
}

// SetBounds updates the widget's pixel dimensions and position.
// SetBounds is called by the layout engine and is not typically called by
// application code.
func (w *BasePublicWidget) SetBounds(x, y, width, height int) {
	w.x = x
	w.y = y
	w.width = width
	w.height = height
}

// SetVisible controls whether the widget participates in layout and rendering.
func (w *BasePublicWidget) SetVisible(visible bool) {
	w.visible = visible
}

// IsVisible returns true if the widget is visible.
func (w *BasePublicWidget) IsVisible() bool {
	return w.visible
}

// OnEvent registers a callback to handle events for this widget.
// OnEvent expects a callback that returns true if it consumes the event.
func (w *BasePublicWidget) OnEvent(handler func(Event) bool) {
	w.onEvent = handler
}

// displayListCanvas implements the Canvas interface by emitting commands
// to a displaylist.DisplayList. This is the bridge between the public
// Canvas API and the internal rendering infrastructure.
type displayListCanvas struct {
	dl *displaylist.DisplayList
}

// newDisplayListCanvas creates a Canvas backed by a DisplayList.
func newDisplayListCanvas(dl *displaylist.DisplayList) Canvas {
	return &displayListCanvas{dl: dl}
}

// FillRect fills a solid rectangle.
func (c *displayListCanvas) FillRect(x, y, width, height int, color Color) {
	c.dl.AddFillRect(x, y, width, height, color.toInternal())
}

// FillRoundedRect fills a rounded rectangle.
func (c *displayListCanvas) FillRoundedRect(x, y, width, height, radius int, color Color) {
	c.dl.AddFillRoundedRect(x, y, width, height, radius, color.toInternal())
}

// DrawLine draws a line segment.
func (c *displayListCanvas) DrawLine(x1, y1, x2, y2 int, color Color, thickness int) {
	c.dl.AddDrawLine(x1, y1, x2, y2, thickness, color.toInternal())
}

// DrawText renders text.
func (c *displayListCanvas) DrawText(text string, x, y int, font *Font, color Color) {
	if font == nil || font.atlas == nil {
		return
	}
	// Use a default font size if not specified
	fontSize := 14
	if font.size > 0 {
		fontSize = int(font.size)
	}
	c.dl.AddDrawText(text, x, y, fontSize, color.toInternal(), font.id)
}

// DrawImage renders an image.
func (c *displayListCanvas) DrawImage(img *Image, x, y, width, height int) {
	if img == nil {
		return
	}
	c.dl.AddDrawImage(x, y, width, height, img.id, img.data, 0.0, 0.0, 1.0, 1.0)
}

// LinearGradient fills a rectangle with a linear gradient.
func (c *displayListCanvas) LinearGradient(x, y, width, height int, startColor, endColor Color, angle float64) {
	// Convert angle to start/end points
	// For simplicity, assume 0 degrees = horizontal left-to-right
	x0, y0 := x, y+height/2
	x1, y1 := x+width, y+height/2
	c.dl.AddLinearGradient(x, y, width, height, x0, y0, x1, y1, startColor.toInternal(), endColor.toInternal())
}

// RadialGradient fills a rectangle with a radial gradient.
func (c *displayListCanvas) RadialGradient(x, y, width, height int, centerColor, edgeColor Color) {
	centerX := x + width/2
	centerY := y + height/2
	radius := width / 2
	if height < width {
		radius = height / 2
	}
	c.dl.AddRadialGradient(x, y, width, height, centerX, centerY, radius, centerColor.toInternal(), edgeColor.toInternal())
}

// BoxShadow renders a box shadow.
func (c *displayListCanvas) BoxShadow(x, y, width, height, offsetX, offsetY, blur int, color Color) {
	// Offset the shadow position
	shadowX := x + offsetX
	shadowY := y + offsetY
	c.dl.AddBoxShadow(shadowX, shadowY, width, height, blur, 0, color.toInternal())
}

// layoutAdapter wraps a PublicWidget tree and implements the Widget interface
// used by the render bridge and event dispatcher. This is the bridge that
// Window.SetLayout uses to attach a public-API widget tree to the window.
type layoutAdapter struct {
	pub     PublicWidget
	focused bool
}

// newLayoutAdapter wraps a PublicWidget so it satisfies the Widget interface.
func newLayoutAdapter(pub PublicWidget) *layoutAdapter {
	return &layoutAdapter{pub: pub}
}

// Contains reports whether the point (x, y) lies within the widget's bounds.
func (a *layoutAdapter) Contains(x, y float64) bool {
	w, h := a.pub.Bounds()
	return x >= 0 && y >= 0 && x < float64(w) && y < float64(h)
}

// Children returns child widgets as Widget values, each wrapped in a layoutAdapter.
func (a *layoutAdapter) Children() []Widget {
	c, ok := a.pub.(Container)
	if !ok {
		return nil
	}
	children := c.Children()
	out := make([]Widget, len(children))
	for i, child := range children {
		out[i] = newLayoutAdapter(child)
	}
	return out
}

// HandlePointer converts the high-level PointerEvent and forwards it.
func (a *layoutAdapter) HandlePointer(evt *PointerEvent) {
	a.pub.HandleEvent(evt)
}

// HandleKey converts the high-level KeyEvent and forwards it.
func (a *layoutAdapter) HandleKey(evt *KeyEvent) {
	a.pub.HandleEvent(evt)
}

// HandleTouch converts the high-level TouchEvent and forwards it.
func (a *layoutAdapter) HandleTouch(evt *TouchEvent) {
	a.pub.HandleEvent(evt)
}

// SetFocused sets the widget's focus state.
func (a *layoutAdapter) SetFocused(focused bool) {
	a.focused = focused
}

// IsFocused reports whether the widget currently has keyboard focus.
func (a *layoutAdapter) IsFocused() bool {
	return a.focused
}
