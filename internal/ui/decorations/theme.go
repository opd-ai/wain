package decorations

import "github.com/opd-ai/wain/internal/raster/core"

// Theme defines the visual appearance of window decorations.
type Theme struct {
	// Title bar colors
	TitleBarBackground core.Color
	TitleBarBorder     core.Color
	TitleTextColor     core.Color

	// Button colors
	ButtonBackgroundNormal  core.Color
	ButtonBackgroundHover   core.Color
	ButtonBackgroundPressed core.Color
	ButtonForegroundNormal  core.Color
	ButtonForegroundHover   core.Color
	ButtonForegroundPressed core.Color

	// Sizing
	TitleBarHeight int
	ButtonSpacing  int
	TitlePaddingX  int
	TitleFontSize  float64

	// Resize handle
	ResizeHandleWidth int
	ResizeHandleColor core.Color
}

// DefaultDecorationTheme returns the default decoration theme.
func DefaultDecorationTheme() *Theme {
	return &Theme{
		TitleBarBackground: core.Color{R: 245, G: 245, B: 245, A: 255},
		TitleBarBorder:     core.Color{R: 200, G: 200, B: 200, A: 255},
		TitleTextColor:     core.Color{R: 30, G: 30, B: 30, A: 255},

		ButtonBackgroundNormal:  core.Color{R: 245, G: 245, B: 245, A: 255},
		ButtonBackgroundHover:   core.Color{R: 220, G: 220, B: 220, A: 255},
		ButtonBackgroundPressed: core.Color{R: 200, G: 200, B: 200, A: 255},
		ButtonForegroundNormal:  core.Color{R: 80, G: 80, B: 80, A: 255},
		ButtonForegroundHover:   core.Color{R: 40, G: 40, B: 40, A: 255},
		ButtonForegroundPressed: core.Color{R: 20, G: 20, B: 20, A: 255},

		TitleBarHeight:    32,
		ButtonSpacing:     4,
		TitlePaddingX:     12,
		TitleFontSize:     14.0,
		ResizeHandleWidth: 8,
		ResizeHandleColor: core.Color{R: 180, G: 180, B: 180, A: 128},
	}
}
