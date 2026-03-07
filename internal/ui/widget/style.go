package widget

import "github.com/opd-ai/wain/internal/raster/core"

// Style defines the visual appearance contract for the widget system.
//
// Consumers customize the look of the toolkit by implementing this interface
// and passing it to widgets or the auto-layout engine. The interface exposes
// color properties for backgrounds, text, borders, and accents, plus font and
// spacing parameters.
//
// A default retro-pixel-art-inspired implementation is provided by
// [DefaultStyle].
type Style interface {
	// Background returns the primary background color.
	Background() core.Color
	// Foreground returns the primary text / foreground color.
	Foreground() core.Color
	// Accent returns the accent / highlight color.
	Accent() core.Color
	// Border returns the border color.
	Border() core.Color
	// FontSize returns the base font size in pixels.
	FontSize() float64
	// Padding returns the default inner padding in pixels.
	Padding() int
	// Gap returns the default gap between sibling widgets in pixels.
	Gap() int
	// BorderWidth returns the default border width in pixels.
	BorderWidth() int
}

// RetroStyle is the default [Style] implementation.
// It combines pixel-art aesthetics with modern high-resolution colors.
type RetroStyle struct {
	BgColor       core.Color
	FgColor       core.Color
	AccentColor   core.Color
	BorderColor   core.Color
	BaseFontSize  float64
	BasePadding   int
	BaseGap       int
	BaseBorderW   int
}

// Background implements [Style].
func (s *RetroStyle) Background() core.Color { return s.BgColor }

// Foreground implements [Style].
func (s *RetroStyle) Foreground() core.Color { return s.FgColor }

// Accent implements [Style].
func (s *RetroStyle) Accent() core.Color { return s.AccentColor }

// Border implements [Style].
func (s *RetroStyle) Border() core.Color { return s.BorderColor }

// FontSize implements [Style].
func (s *RetroStyle) FontSize() float64 { return s.BaseFontSize }

// Padding implements [Style].
func (s *RetroStyle) Padding() int { return s.BasePadding }

// Gap implements [Style].
func (s *RetroStyle) Gap() int { return s.BaseGap }

// BorderWidth implements [Style].
func (s *RetroStyle) BorderWidth() int { return s.BaseBorderW }

// DefaultStyle returns the built-in retro-pixel-art style.
//
// Colors are chosen for a dark-background, high-contrast aesthetic reminiscent
// of classic pixel-art UIs but rendered at modern resolutions.
func DefaultStyle() Style {
	return &RetroStyle{
		BgColor:      core.Color{R: 30, G: 30, B: 46, A: 255},    // dark blue-gray
		FgColor:      core.Color{R: 205, G: 214, B: 244, A: 255}, // soft white
		AccentColor:  core.Color{R: 137, G: 180, B: 250, A: 255}, // bright blue
		BorderColor:  core.Color{R: 88, G: 91, B: 112, A: 255},   // muted gray
		BaseFontSize: 14.0,
		BasePadding:  8,
		BaseGap:      6,
		BaseBorderW:  1,
	}
}
