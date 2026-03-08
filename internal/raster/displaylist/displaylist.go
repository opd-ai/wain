// Package displaylist provides a display list abstraction for GPU rendering.
//
// A display list is a sequence of draw commands that can be consumed by either
// the software rasterizer or the GPU backend. This allows the UI layer to be
// renderer-agnostic.
package displaylist

import (
	"github.com/opd-ai/wain/internal/raster/core"
)

// CommandType represents the type of draw command.
type CommandType uint8

const (
	// CmdFillRect fills a solid rectangle.
	CmdFillRect CommandType = iota

	// CmdFillRoundedRect fills a rounded rectangle.
	CmdFillRoundedRect

	// CmdDrawLine draws a line segment.
	CmdDrawLine

	// CmdDrawText draws text using SDF rendering.
	CmdDrawText

	// CmdLinearGradient fills a rectangle with a linear gradient.
	CmdLinearGradient

	// CmdRadialGradient fills a rectangle with a radial gradient.
	CmdRadialGradient

	// CmdBoxShadow renders a box shadow effect.
	CmdBoxShadow

	// CmdDrawImage draws a textured image.
	CmdDrawImage
)

// DrawCommand represents a single draw operation.
type DrawCommand struct {
	Type CommandType
	Data interface{}
}

// FillRectData contains parameters for a filled rectangle.
type FillRectData struct {
	X, Y          int
	Width, Height int
	Color         core.Color
}

// FillRoundedRectData contains parameters for a filled rounded rectangle.
type FillRoundedRectData struct {
	X, Y          int
	Width, Height int
	Radius        int
	Color         core.Color
}

// DrawLineData contains parameters for a line segment.
type DrawLineData struct {
	X0, Y0, X1, Y1 int
	Width          int
	Color          core.Color
}

// DrawTextData contains parameters for text rendering.
type DrawTextData struct {
	Text     string
	X, Y     int
	FontSize int
	Color    core.Color
	AtlasID  int // Reference to font atlas texture (for GPU backend)
}

// LinearGradientData contains parameters for linear gradient.
type LinearGradientData struct {
	X, Y          int
	Width, Height int
	X0, Y0        int // Start point
	X1, Y1        int // End point
	Color0        core.Color
	Color1        core.Color
}

// RadialGradientData contains parameters for radial gradient.
type RadialGradientData struct {
	X, Y          int
	Width, Height int
	CenterX       int
	CenterY       int
	Radius        int
	Color0        core.Color
	Color1        core.Color
}

// BoxShadowData contains parameters for box shadow.
type BoxShadowData struct {
	X, Y          int
	Width, Height int
	BlurRadius    int
	SpreadRadius  int
	Color         core.Color
}

// DrawImageData contains parameters for image rendering.
type DrawImageData struct {
	X, Y          int
	Width, Height int
	TextureID     int // Reference to texture atlas entry
	U0, V0        float32
	U1, V1        float32
}

// DisplayList is a sequence of draw commands.
type DisplayList struct {
	commands []DrawCommand
}

// New creates a new empty display list.
func New() *DisplayList {
	return &DisplayList{
		commands: make([]DrawCommand, 0, 256),
	}
}

// Reset clears the display list for reuse.
func (dl *DisplayList) Reset() {
	dl.commands = dl.commands[:0]
}

// AddFillRect adds a filled rectangle command.
func (dl *DisplayList) AddFillRect(x, y, width, height int, color core.Color) {
	dl.commands = append(dl.commands, DrawCommand{
		Type: CmdFillRect,
		Data: FillRectData{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
			Color:  color,
		},
	})
}

// AddFillRoundedRect adds a filled rounded rectangle command.
func (dl *DisplayList) AddFillRoundedRect(x, y, width, height, radius int, color core.Color) {
	dl.commands = append(dl.commands, DrawCommand{
		Type: CmdFillRoundedRect,
		Data: FillRoundedRectData{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
			Radius: radius,
			Color:  color,
		},
	})
}

// AddDrawLine adds a line drawing command.
func (dl *DisplayList) AddDrawLine(x0, y0, x1, y1, width int, color core.Color) {
	dl.commands = append(dl.commands, DrawCommand{
		Type: CmdDrawLine,
		Data: DrawLineData{
			X0:    x0,
			Y0:    y0,
			X1:    x1,
			Y1:    y1,
			Width: width,
			Color: color,
		},
	})
}

// AddDrawText adds a text drawing command.
func (dl *DisplayList) AddDrawText(text string, x, y, fontSize int, color core.Color, atlasID int) {
	dl.commands = append(dl.commands, DrawCommand{
		Type: CmdDrawText,
		Data: DrawTextData{
			Text:     text,
			X:        x,
			Y:        y,
			FontSize: fontSize,
			Color:    color,
			AtlasID:  atlasID,
		},
	})
}

// AddLinearGradient adds a linear gradient command.
func (dl *DisplayList) AddLinearGradient(x, y, width, height, x0, y0, x1, y1 int, color0, color1 core.Color) {
	dl.commands = append(dl.commands, DrawCommand{
		Type: CmdLinearGradient,
		Data: LinearGradientData{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
			X0:     x0,
			Y0:     y0,
			X1:     x1,
			Y1:     y1,
			Color0: color0,
			Color1: color1,
		},
	})
}

// AddRadialGradient adds a radial gradient command.
func (dl *DisplayList) AddRadialGradient(x, y, width, height, centerX, centerY, radius int, color0, color1 core.Color) {
	dl.commands = append(dl.commands, DrawCommand{
		Type: CmdRadialGradient,
		Data: RadialGradientData{
			X:       x,
			Y:       y,
			Width:   width,
			Height:  height,
			CenterX: centerX,
			CenterY: centerY,
			Radius:  radius,
			Color0:  color0,
			Color1:  color1,
		},
	})
}

// AddBoxShadow adds a box shadow command.
func (dl *DisplayList) AddBoxShadow(x, y, width, height, blurRadius, spreadRadius int, color core.Color) {
	dl.commands = append(dl.commands, DrawCommand{
		Type: CmdBoxShadow,
		Data: BoxShadowData{
			X:            x,
			Y:            y,
			Width:        width,
			Height:       height,
			BlurRadius:   blurRadius,
			SpreadRadius: spreadRadius,
			Color:        color,
		},
	})
}

// AddDrawImage adds an image drawing command.
func (dl *DisplayList) AddDrawImage(x, y, width, height, textureID int, u0, v0, u1, v1 float32) {
	dl.commands = append(dl.commands, DrawCommand{
		Type: CmdDrawImage,
		Data: DrawImageData{
			X:         x,
			Y:         y,
			Width:     width,
			Height:    height,
			TextureID: textureID,
			U0:        u0,
			V0:        v0,
			U1:        u1,
			V1:        v1,
		},
	})
}

// Commands returns the slice of draw commands.
func (dl *DisplayList) Commands() []DrawCommand {
	return dl.commands
}

// Len returns the number of commands in the display list.
func (dl *DisplayList) Len() int {
	return len(dl.commands)
}
