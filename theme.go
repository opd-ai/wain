package wain

const (
	// DefaultPadding is the default inner padding in pixels.
	DefaultPadding = 8

	// DefaultGap is the default gap between sibling widgets in pixels.
	DefaultGap = 6

	// DefaultBorderRadius is the default border radius for rounded corners in pixels.
	DefaultBorderRadius = 4
)

// Theme defines the global visual appearance of the application.
//
// A Theme specifies colors, fonts, spacing, and scale that can be applied
// application-wide or per-widget. Widgets inherit theme from their parent
// container unless overridden with a StyleOverride.
//
// The framework provides three built-in themes accessible via DefaultDark(),
// DefaultLight(), and HighContrast(). Custom themes can be created by
// constructing a Theme struct with desired values.
//
// Example:
//
//	app.SetTheme(wain.DefaultDark())
//	app.SetTheme(wain.HighContrast())
//
//	custom := wain.Theme{
//	    Background: wain.RGB(20, 20, 25),
//	    Foreground: wain.RGB(230, 230, 230),
//	    Accent:     wain.RGB(100, 180, 255),
//	    // ... other fields
//	}
//	app.SetTheme(custom)
type Theme struct {
	// Background is the primary background color.
	Background Color

	// Foreground is the primary text/foreground color.
	Foreground Color

	// Accent is the accent/highlight color for interactive elements.
	Accent Color

	// Border is the default border color.
	Border Color

	// FontSize is the base font size in pixels.
	FontSize float64

	// Padding is the default inner padding in pixels.
	Padding int

	// Gap is the default gap between sibling widgets in pixels.
	Gap int

	// BorderWidth is the default border width in pixels.
	BorderWidth int

	// BorderRadius is the default border radius for rounded corners in pixels.
	BorderRadius int

	// Scale is the HiDPI scale factor.
	// Auto-detected from the display server but can be overridden.
	// A value of 1.0 is standard DPI, 2.0 is 2x (retina), etc.
	Scale float64
}

// DefaultDark returns the built-in dark theme.
//
// This is the default theme inspired by retro pixel-art aesthetics with
// modern high-resolution colors. It features a dark blue-gray background
// with soft white text and bright blue accents.
//
// Color palette:
//   - Background: dark blue-gray (#1E1E2E)
//   - Foreground: soft white (#CDD6F4)
//   - Accent: bright blue (#89B4FA)
//   - Border: muted gray (#585B70)
func DefaultDark() Theme {
	return Theme{
		Background:   RGB(30, 30, 46),
		Foreground:   RGB(205, 214, 244),
		Accent:       RGB(137, 180, 250),
		Border:       RGB(88, 91, 112),
		FontSize:     14.0,
		Padding:      DefaultPadding,
		Gap:          DefaultGap,
		BorderWidth:  1,
		BorderRadius: DefaultBorderRadius,
		Scale:        1.0,
	}
}

// DefaultLight returns the built-in light theme.
//
// This theme provides a clean, bright appearance suitable for well-lit
// environments. It features a light gray background with dark text and
// blue accents.
//
// Color palette:
//   - Background: light gray (#F5F5F5)
//   - Foreground: dark gray (#1E1E1E)
//   - Accent: medium blue (#4A90E2)
//   - Border: medium gray (#C8C8C8)
func DefaultLight() Theme {
	return Theme{
		Background:   RGB(245, 245, 245),
		Foreground:   RGB(30, 30, 30),
		Accent:       RGB(74, 144, 226),
		Border:       RGB(200, 200, 200),
		FontSize:     14.0,
		Padding:      DefaultPadding,
		Gap:          DefaultGap,
		BorderWidth:  1,
		BorderRadius: DefaultBorderRadius,
		Scale:        1.0,
	}
}

// HighContrast returns a high-contrast theme for accessibility.
//
// This theme provides maximum readability with pure black background,
// pure white text, and bright yellow accents. It meets WCAG AAA contrast
// requirements and is suitable for users with visual impairments.
//
// Color palette:
//   - Background: pure black (#000000)
//   - Foreground: pure white (#FFFFFF)
//   - Accent: bright yellow (#FFFF00)
//   - Border: pure white (#FFFFFF)
func HighContrast() Theme {
	return Theme{
		Background:   RGB(0, 0, 0),
		Foreground:   RGB(255, 255, 255),
		Accent:       RGB(255, 255, 0),
		Border:       RGB(255, 255, 255),
		FontSize:     16.0, // Larger base font for readability
		Padding:      10,   // More padding for easier targeting
		Gap:          8,    // Larger gaps for visual separation
		BorderWidth:  2,    // Thicker borders for visibility
		BorderRadius: 0,    // No rounded corners (sharp edges are clearer)
		Scale:        1.0,
	}
}

// StyleOverride provides per-widget visual customization.
//
// A StyleOverride allows individual widgets to override specific theme
// properties without affecting the global theme. Any field left as zero
// value will inherit from the parent container's theme.
//
// Example:
//
//	panel := wain.NewPanel(wain.Size{Width: 100, Height: 100})
//	panel.SetStyle(wain.StyleOverride{
//	    Background: wain.RGB(40, 40, 60),
//	    BorderRadius: 12,
//	})
//
//	// Only background and border radius are overridden;
//	// foreground, accent, etc. inherit from the theme.
type StyleOverride struct {
	// Background overrides the background color if non-zero.
	Background *Color

	// Foreground overrides the foreground color if non-zero.
	Foreground *Color

	// Accent overrides the accent color if non-zero.
	Accent *Color

	// Border overrides the border color if non-zero.
	Border *Color

	// FontSize overrides the font size if non-zero.
	FontSize *float64

	// Padding overrides the padding if non-zero.
	Padding *int

	// Gap overrides the gap if non-zero.
	Gap *int

	// BorderWidth overrides the border width if non-zero.
	BorderWidth *int

	// BorderRadius overrides the border radius if non-zero.
	BorderRadius *int
}

// applyOverride sets *dst = *src when src is non-nil, leaving *dst unchanged otherwise.
func applyOverride[T any](dst, src *T) {
	if src != nil {
		*dst = *src
	}
}

// applyToTheme applies the style override to a theme, returning a new theme
// with overridden values.
func (s StyleOverride) applyToTheme(base Theme) Theme {
	result := base
	applyOverride(&result.Background, s.Background)
	applyOverride(&result.Foreground, s.Foreground)
	applyOverride(&result.Accent, s.Accent)
	applyOverride(&result.Border, s.Border)
	applyOverride(&result.FontSize, s.FontSize)
	applyOverride(&result.Padding, s.Padding)
	applyOverride(&result.Gap, s.Gap)
	applyOverride(&result.BorderWidth, s.BorderWidth)
	applyOverride(&result.BorderRadius, s.BorderRadius)
	return result
}
