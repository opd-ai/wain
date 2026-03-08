package pctwidget

import (
	"errors"

	"github.com/opd-ai/wain/internal/raster/core"
)

// BaseWidget provides the common foundation for all percentage-based widgets.
//
// It stores percentage-based size, an optional manual position override,
// a reference to a [Style], and drawing state. Concrete widgets embed
// BaseWidget and add type-specific behaviour.
type BaseWidget struct {
	// Percentage-based size relative to parent.
	size Size

	// Resolved absolute position and dimensions (set by the layout engine).
	x, y          int
	width, height int

	// manualPos, when true, indicates the widget has a manually-set position
	// that overrides auto-layout.
	manualPos bool

	// style holds the visual style; nil means use DefaultStyle.
	style Style

	// visible controls whether the widget participates in layout and drawing.
	visible bool
}

// NewBaseWidget creates a BaseWidget with the given percentage-based dimensions.
// Width and height are clamped to the valid [0, 100] range.
func NewBaseWidget(widthPct, heightPct float64) BaseWidget {
	return BaseWidget{
		size: Size{
			Width:  Percent(widthPct).Clamp(),
			Height: Percent(heightPct).Clamp(),
		},
		visible: true,
	}
}

// SetSize updates the percentage-based dimensions. Values are clamped.
func (w *BaseWidget) SetSize(widthPct, heightPct float64) {
	w.size.Width = Percent(widthPct).Clamp()
	w.size.Height = Percent(heightPct).Clamp()
}

// PercentSize returns the current percentage-based size.
func (w *BaseWidget) PercentSize() Size { return w.size }

// SetPosition manually overrides the auto-layout position and pixel dimensions.
// After calling this, the widget will not be repositioned by the auto-layout engine.
func (w *BaseWidget) SetPosition(x, y, width, height int) {
	w.x = x
	w.y = y
	w.width = width
	w.height = height
	w.manualPos = true
}

// ClearPosition removes the manual override and returns the widget to auto-layout.
func (w *BaseWidget) ClearPosition() {
	w.manualPos = false
}

// IsManuallyPositioned reports whether the widget has a manual position override.
func (w *BaseWidget) IsManuallyPositioned() bool { return w.manualPos }

// ResolvedBounds returns the resolved absolute position and pixel dimensions.
func (w *BaseWidget) ResolvedBounds() (x, y, width, height int) {
	return w.x, w.y, w.width, w.height
}

// SetStyle assigns a custom [Style] to this widget. Pass nil to revert to default.
func (w *BaseWidget) SetStyle(s Style) { w.style = s }

// EffectiveStyle returns the widget's style, falling back to [DefaultStyle].
func (w *BaseWidget) EffectiveStyle() Style {
	if w.style != nil {
		return w.style
	}
	return DefaultStyle()
}

// SetVisible controls whether this widget is drawn and laid out.
func (w *BaseWidget) SetVisible(v bool) { w.visible = v }

// Visible reports whether the widget is visible.
func (w *BaseWidget) Visible() bool { return w.visible }

// Resolve computes pixel dimensions from the percentage size and the given
// parent dimensions, storing the result internally. It is called by the
// auto-layout engine; consumers rarely need to call it directly.
func (w *BaseWidget) Resolve(parentWidth, parentHeight int) error {
	pw, ph, err := w.size.Resolve(parentWidth, parentHeight)
	if err != nil {
		return err
	}
	w.width = pw
	w.height = ph
	return nil
}

// Panel is a concrete widget that draws a styled rectangular region.
// It can hold child widgets that are laid out automatically within it.
type Panel struct {
	BaseWidget
	children []*Panel
}

// NewPanel creates a panel with percentage-based dimensions.
func NewPanel(widthPct, heightPct float64) *Panel {
	return &Panel{
		BaseWidget: NewBaseWidget(widthPct, heightPct),
	}
}

// AddChild appends a child panel. Children are laid out within the parent's
// resolved bounds by the auto-layout engine.
func (p *Panel) AddChild(child *Panel) {
	p.children = append(p.children, child)
}

// Children returns the list of child panels.
func (p *Panel) Children() []*Panel { return p.children }

// Draw renders the panel (and its children recursively) into the buffer.
func (p *Panel) Draw(buf *core.Buffer) error {
	if buf == nil {
		return ErrNilBuffer
	}
	if !p.visible {
		return nil
	}
	s := p.EffectiveStyle()
	buf.FillRect(p.x, p.y, p.width, p.height, s.Background())
	p.drawBorder(buf, s)
	return p.drawChildren(buf)
}

// drawBorder renders the panel border if border width is non-zero.
func (p *Panel) drawBorder(buf *core.Buffer, s Style) {
	bw := s.BorderWidth()
	if bw <= 0 || p.width <= 0 || p.height <= 0 {
		return
	}
	// Clamp border width so it doesn't exceed half the panel dimension.
	if bw > p.width/2 {
		bw = p.width / 2
	}
	if bw > p.height/2 {
		bw = p.height / 2
	}
	bc := s.Border()
	buf.FillRect(p.x, p.y, p.width, bw, bc)                       // top
	buf.FillRect(p.x, p.y+p.height-bw, p.width, bw, bc)           // bottom
	buf.FillRect(p.x, p.y, bw, p.height, bc)                      // left
	buf.FillRect(p.x+p.width-bw, p.y, bw, p.height, bc)           // right
}

// drawChildren renders all child panels recursively.
func (p *Panel) drawChildren(buf *core.Buffer) error {
	for _, c := range p.children {
		if err := c.Draw(buf); err != nil {
			return err
		}
	}
	return nil
}

// ErrNilBuffer is returned when a nil buffer is provided for rendering.
var ErrNilBuffer = errors.New("widget: nil buffer")
