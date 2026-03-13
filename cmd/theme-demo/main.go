package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain"
)

// printThemeInfo prints the common fields of a named theme.
func printThemeInfo(label string, t wain.Theme) {
	fmt.Printf("%s Theme:\n", label)
	fmt.Printf("  Background: RGB(%d, %d, %d)\n", t.Background.R, t.Background.G, t.Background.B)
	fmt.Printf("  Foreground: RGB(%d, %d, %d)\n", t.Foreground.R, t.Foreground.G, t.Foreground.B)
	fmt.Printf("  Accent:     RGB(%d, %d, %d)\n", t.Accent.R, t.Accent.G, t.Accent.B)
	fmt.Printf("  Border:     RGB(%d, %d, %d)\n", t.Border.R, t.Border.G, t.Border.B)
	fmt.Printf("  FontSize:   %.1f\n", t.FontSize)
	fmt.Printf("  Padding:    %d\n", t.Padding)
}

func main() {
	app := wain.NewApp()

	// Demonstrate theme switching
	fmt.Println("=== Theme Demo ===")
	fmt.Println()

	printThemeInfo("DefaultDark", wain.DefaultDark())
	fmt.Println()

	printThemeInfo("DefaultLight", wain.DefaultLight())
	fmt.Println()

	// Display HighContrast theme — also shows extra border fields.
	hcTheme := wain.HighContrast()
	printThemeInfo("HighContrast", hcTheme)
	fmt.Printf("  BorderWidth:%d\n", hcTheme.BorderWidth)
	fmt.Printf("  BorderRadius:%d\n", hcTheme.BorderRadius)
	fmt.Println()

	// Demonstrate theme switching on app
	fmt.Println("Setting theme to DefaultDark...")
	app.SetTheme(wain.DefaultDark())
	currentTheme := app.GetTheme()
	fmt.Printf("Current theme background: RGB(%d, %d, %d)\n", currentTheme.Background.R, currentTheme.Background.G, currentTheme.Background.B)
	fmt.Println()

	fmt.Println("Setting theme to DefaultLight...")
	app.SetTheme(wain.DefaultLight())
	currentTheme = app.GetTheme()
	fmt.Printf("Current theme background: RGB(%d, %d, %d)\n", currentTheme.Background.R, currentTheme.Background.G, currentTheme.Background.B)
	fmt.Println()

	fmt.Println("Setting theme to HighContrast...")
	app.SetTheme(wain.HighContrast())
	currentTheme = app.GetTheme()
	fmt.Printf("Current theme background: RGB(%d, %d, %d)\n", currentTheme.Background.R, currentTheme.Background.G, currentTheme.Background.B)
	fmt.Println()

	// Demonstrate StyleOverride
	fmt.Println("Testing StyleOverride:")
	baseTheme := wain.DefaultDark()
	customBg := wain.RGB(100, 100, 150)
	fontSize := 18.0
	override := wain.StyleOverride{
		Background: &customBg,
		FontSize:   &fontSize,
	}

	// Note: applyToTheme is private, so we can't demonstrate it here.
	// In actual usage, widgets would use it internally.
	fmt.Printf("  Created override with custom background RGB(%d, %d, %d) and font size %.1f\n", customBg.R, customBg.G, customBg.B, fontSize)
	fmt.Printf("  Base theme background: RGB(%d, %d, %d), fontSize: %.1f\n", baseTheme.Background.R, baseTheme.Background.G, baseTheme.Background.B, baseTheme.FontSize)
	fmt.Println()

	fmt.Println("=== Theme Demo Complete ===")

	// Note: We're not running the full app event loop since this is just a theme demo
	_ = app
	_ = override
	log.Println("Theme demo completed successfully")
}
