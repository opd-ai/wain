// Package consumer provides display list consumers for different rendering backends.
//
// A consumer takes a DisplayList and renders it to a target surface using either
// software (CPU) rasterization or GPU acceleration.
//
// Software Rasterizer Limitations:
//
// The SoftwareConsumer does not implement the CmdDrawImage display list command.
// Image compositing is available through the composite package's Blit and BlitScaled
// functions, but DrawImage command execution requires a GPU backend. This is a
// deliberate design decision to keep the software rasterizer focused on vector
// primitives while GPU-accelerated texture sampling handles image operations.
package consumer

import (
	"fmt"

	"github.com/opd-ai/wain/internal/raster/primitives"
	"github.com/opd-ai/wain/internal/raster/displaylist"
	"github.com/opd-ai/wain/internal/raster/effects"
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
			return err
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
		// DrawImage not yet implemented in software rasterizer
		// Skip for now - GPU backend handles this

	default:
		return fmt.Errorf("consumer: unknown command type %d", cmd.Type)
	}

	return nil
}
