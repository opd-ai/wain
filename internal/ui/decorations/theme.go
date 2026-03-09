package decorations

import "github.com/opd-ai/wain/internal/raster/primitives"

// Theme defines the visual appearance of window decorations.
type Theme struct {
	// Title bar colors
	TitleBarBackground primitives.Color
	TitleBarBorder     primitives.Color
	TitleTextColor     primitives.Color

	// Button colors
	ButtonBackgroundNormal  primitives.Color
	ButtonBackgroundHover   primitives.Color
	ButtonBackgroundPressed primitives.Color
	ButtonForegroundNormal  primitives.Color
	ButtonForegroundHover   primitives.Color
	ButtonForegroundPressed primitives.Color

	// Sizing
	TitleBarHeight int
	ButtonSpacing  int
	TitlePaddingX  int
	TitleFontSize  float64

	// Resize handle
	ResizeHandleWidth int
	ResizeHandleColor primitives.Color
}

// DefaultDecorationTheme returns the default decoration theme.
func DefaultDecorationTheme() *Theme {
	return &Theme{
		TitleBarBackground: primitives.Color{R: 245, G: 245, B: 245, A: 255},
		TitleBarBorder:     primitives.Color{R: 200, G: 200, B: 200, A: 255},
		TitleTextColor:     primitives.Color{R: 30, G: 30, B: 30, A: 255},

		ButtonBackgroundNormal:  primitives.Color{R: 245, G: 245, B: 245, A: 255},
		ButtonBackgroundHover:   primitives.Color{R: 220, G: 220, B: 220, A: 255},
		ButtonBackgroundPressed: primitives.Color{R: 200, G: 200, B: 200, A: 255},
		ButtonForegroundNormal:  primitives.Color{R: 80, G: 80, B: 80, A: 255},
		ButtonForegroundHover:   primitives.Color{R: 40, G: 40, B: 40, A: 255},
		ButtonForegroundPressed: primitives.Color{R: 20, G: 20, B: 20, A: 255},

		TitleBarHeight:    32,
		ButtonSpacing:     4,
		TitlePaddingX:     12,
		TitleFontSize:     14.0,
		ResizeHandleWidth: 8,
		ResizeHandleColor: primitives.Color{R: 180, G: 180, B: 180, A: 128},
	}
}
