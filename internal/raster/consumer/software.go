// Package consumer provides display list consumers for different rendering backends.
//
// A consumer takes a DisplayList and renders it to a target surface using either
// software (CPU) rasterization or GPU acceleration.
package consumer

import (
	"fmt"
	"image"
	"image/color"

	"github.com/opd-ai/wain/internal/raster/composite"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/effects"
	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/text"
)

// SoftwareConsumer renders display lists using software rasterization.
type SoftwareConsumer struct {
	atlas *text.Atlas
}

// NewSoftwareConsumer creates a new software display list consumer.
func NewSoftwareConsumer(atlas *text.Atlas) *SoftwareConsumer {
	return &SoftwareConsumer{
		atlas: atlas,
	}
}

// Render executes all commands in the display list using software rasterization.
func (sc *SoftwareConsumer) Render(dl *displaylist.DisplayList, buf *primitives.Buffer) error {
	if dl == nil {
		return fmt.Errorf("consumer: nil display list")
	}
	if buf == nil {
		return fmt.Errorf("consumer: nil buffer")
	}

	for _, cmd := range dl.Commands() {
		if err := sc.renderCommand(cmd, buf); err != nil {
			return fmt.Errorf("consumer: render command %v: %w", cmd.Type, err)
		}
	}

	return nil
}

// renderCommand executes a single draw command.
func (sc *SoftwareConsumer) renderCommand(cmd displaylist.DrawCommand, buf *primitives.Buffer) error {
	switch cmd.Type {
	case displaylist.CmdFillRect:
		data := cmd.Data.(displaylist.FillRectData)
		buf.FillRect(data.X, data.Y, data.Width, data.Height, data.Color)

	case displaylist.CmdFillRoundedRect:
		data := cmd.Data.(displaylist.FillRoundedRectData)
		buf.FillRoundedRect(data.X, data.Y, data.Width, data.Height, float64(data.Radius), data.Color)

	case displaylist.CmdDrawLine:
		data := cmd.Data.(displaylist.DrawLineData)
		buf.DrawLine(data.X0, data.Y0, data.X1, data.Y1, float64(data.Width), data.Color)

	case displaylist.CmdDrawText:
		data := cmd.Data.(displaylist.DrawTextData)
		if sc.atlas != nil {
			text.DrawText(buf, data.Text, float64(data.X), float64(data.Y), float64(data.FontSize), data.Color, sc.atlas)
		}

	case displaylist.CmdLinearGradient:
		data := cmd.Data.(displaylist.LinearGradientData)
		effects.LinearGradient(buf, data.X, data.Y, data.Width, data.Height,
			data.X0, data.Y0, data.Color0,
			data.X1, data.Y1, data.Color1)

	case displaylist.CmdRadialGradient:
		data := cmd.Data.(displaylist.RadialGradientData)
		effects.RadialGradient(buf, data.X, data.Y, data.Width, data.Height,
			data.CenterX, data.CenterY, data.Radius, data.Color0, data.Color1)

	case displaylist.CmdBoxShadow:
		data := cmd.Data.(displaylist.BoxShadowData)
		effects.BoxShadow(buf, data.X, data.Y, data.Width, data.Height, data.BlurRadius, data.Color)

	case displaylist.CmdDrawImage:
		data := cmd.Data.(displaylist.DrawImageData)
		sc.renderDrawImage(data, buf)

	default:
		return fmt.Errorf("consumer: unknown command type %d", cmd.Type)
	}

	return nil
}

// renderDrawImage blits a software image into the destination buffer using
// bilinear scaling. If the command carries no source image (GPU-only path),
// the call is silently ignored.
func (sc *SoftwareConsumer) renderDrawImage(data displaylist.DrawImageData, buf *primitives.Buffer) {
	if data.Src == nil || data.Width <= 0 || data.Height <= 0 {
		return
	}
	src := imageToBuffer(data.Src)
	if src == nil {
		return
	}
	composite.BlitScaled(buf, data.X, data.Y, data.Width, data.Height,
		src, 0, 0, src.Width, src.Height)
}

// imageToBuffer converts a standard image.Image to a primitives.Buffer in
// ARGB8888 format (B, G, R, A byte order) for use by the software rasterizer.
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
