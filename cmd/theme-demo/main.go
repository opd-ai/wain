package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain"
)

func main() {
	app := wain.NewApp()

	// Demonstrate theme switching
	fmt.Println("=== Theme Demo ===")
	fmt.Println()

	// Display DefaultDark theme
	darkTheme := wain.DefaultDark()
	fmt.Println("DefaultDark Theme:")
	fmt.Printf("  Background: RGB(%d, %d, %d)\n", darkTheme.Background.R, darkTheme.Background.G, darkTheme.Background.B)
	fmt.Printf("  Foreground: RGB(%d, %d, %d)\n", darkTheme.Foreground.R, darkTheme.Foreground.G, darkTheme.Foreground.B)
	fmt.Printf("  Accent:     RGB(%d, %d, %d)\n", darkTheme.Accent.R, darkTheme.Accent.G, darkTheme.Accent.B)
	fmt.Printf("  Border:     RGB(%d, %d, %d)\n", darkTheme.Border.R, darkTheme.Border.G, darkTheme.Border.B)
	fmt.Printf("  FontSize:   %.1f\n", darkTheme.FontSize)
	fmt.Printf("  Padding:    %d\n", darkTheme.Padding)
	fmt.Println()

	// Display DefaultLight theme
	lightTheme := wain.DefaultLight()
	fmt.Println("DefaultLight Theme:")
	fmt.Printf("  Background: RGB(%d, %d, %d)\n", lightTheme.Background.R, lightTheme.Background.G, lightTheme.Background.B)
	fmt.Printf("  Foreground: RGB(%d, %d, %d)\n", lightTheme.Foreground.R, lightTheme.Foreground.G, lightTheme.Foreground.B)
	fmt.Printf("  Accent:     RGB(%d, %d, %d)\n", lightTheme.Accent.R, lightTheme.Accent.G, lightTheme.Accent.B)
	fmt.Printf("  Border:     RGB(%d, %d, %d)\n", lightTheme.Border.R, lightTheme.Border.G, lightTheme.Border.B)
	fmt.Printf("  FontSize:   %.1f\n", lightTheme.FontSize)
	fmt.Printf("  Padding:    %d\n", lightTheme.Padding)
	fmt.Println()

	// Display HighContrast theme
	hcTheme := wain.HighContrast()
	fmt.Println("HighContrast Theme:")
	fmt.Printf("  Background: RGB(%d, %d, %d)\n", hcTheme.Background.R, hcTheme.Background.G, hcTheme.Background.B)
	fmt.Printf("  Foreground: RGB(%d, %d, %d)\n", hcTheme.Foreground.R, hcTheme.Foreground.G, hcTheme.Foreground.B)
	fmt.Printf("  Accent:     RGB(%d, %d, %d)\n", hcTheme.Accent.R, hcTheme.Accent.G, hcTheme.Accent.B)
	fmt.Printf("  Border:     RGB(%d, %d, %d)\n", hcTheme.Border.R, hcTheme.Border.G, hcTheme.Border.B)
	fmt.Printf("  FontSize:   %.1f\n", hcTheme.FontSize)
	fmt.Printf("  Padding:    %d\n", hcTheme.Padding)
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
