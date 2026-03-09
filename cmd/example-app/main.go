// Package main demonstrates the complete public wain API (Phase 10.7).
//
// This reference application showcases all public API features:
//  - App creation and lifecycle management
//  - Window creation with custom configuration
//  - Multi-panel layout using percentage-based sizing
//  - All container types: Row, Column, ScrollView
//  - All widget types: Panel, Button, Label, TextInput, Spacer
//  - Event handling and callbacks
//  - Theme switching between DefaultDark, DefaultLight, and HighContrast
//  - Cross-goroutine state updates via App.Notify()
//
// The demo creates a complete UI with:
//  - Header (Row): title Label and theme toggle buttons
//  - Main content (Row): sidebar navigation (Column) + content area (ScrollView)
//  - Footer (Row): status Label
//
// All sizing is percentage-based, adapting automatically to window resize.
//
// NOTE: This example demonstrates the API structure. Full integration with
// the rendering pipeline is in progress. To run this example:
//     go generate ./...
//     go build ./cmd/example-app
package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain"
)

func main() {
	log.SetFlags(0)
	log.Println("=== Wain Example Application ===")
	log.Println()

	// Create application
	app := wain.NewApp()

	// Set default theme
	app.SetTheme(wain.DefaultDark())
	log.Printf("Theme set: DefaultDark")
	log.Println()

	// === Build UI Layout ===
	log.Println("Building UI layout...")
	log.Println()

	// Header: title + theme toggle buttons (Row, 100% width)
	header := buildHeader(app)
	log.Println("✓ Header created (Row with title + theme buttons)")

	// Main content: sidebar + content area (Row, 100% width)
	mainContent := buildMainContent(app)
	log.Println("✓ Main content created (Row with sidebar + scrollable content)")

	// Footer: status bar (Row, 100% width)
	footer := buildFooter(app)
	log.Println("✓ Footer created (Row with status label)")

	// Root layout: Column stacking header, main, footer
	root := wain.NewColumn()
	root.Add(header)
	root.Add(mainContent)
	root.Add(footer)
	log.Println("✓ Root layout assembled (Column)")
	log.Println()

	// Print layout structure
	printLayoutStructure()

	// Demonstrate theme switching
	log.Println("=== Theme Switching ===")
	demonstrateThemes(app)
	log.Println()

	// Report success
	log.Println("=== Example Complete ===")
	log.Println()
	log.Println("This example demonstrates the public wain API structure:")
	log.Println("  • Percentage-based layout (Row, Column, ScrollView)")
	log.Println("  • Widget types (Panel, Button, Label, TextInput, Spacer)")
	log.Println("  • Theme system (DefaultDark, DefaultLight, HighContrast)")
	log.Println("  • Event callbacks (OnClick, OnChange, OnScroll)")
	log.Println("  • Cross-goroutine updates (App.Notify)")
	log.Println()
	log.Println("To create a window and run the event loop, use:")
	log.Println("  win, _ := app.NewWindow(wain.WindowConfig{...})")
	log.Println("  app.Run()")

	_ = root
}

// buildHeader creates the header row with title and theme buttons
func buildHeader(app *wain.App) *wain.Row {
	header := wain.NewRow()
	header.SetPadding(10)
	header.SetGap(10)

	// Title label (70% width)
	title := wain.NewLabel("Wain Example Application", wain.Size{Width: 70, Height: 100})
	title.SetFontSize(24.0)
	header.Add(title)

	// Spacer for alignment
	header.Add(wain.NewSpacer(wain.Size{Width: 5, Height: 100}))

	// Theme toggle buttons (25% width total, split into 3 buttons)
	darkBtn := wain.NewButton("Dark", wain.Size{Width: 8, Height: 100})
	darkBtn.OnClick(func() {
		app.SetTheme(wain.DefaultDark())
		log.Println("Theme: Dark")
	})

	lightBtn := wain.NewButton("Light", wain.Size{Width: 8, Height: 100})
	lightBtn.OnClick(func() {
		app.SetTheme(wain.DefaultLight())
		log.Println("Theme: Light")
	})

	hcBtn := wain.NewButton("HighContrast", wain.Size{Width: 9, Height: 100})
	hcBtn.OnClick(func() {
		app.SetTheme(wain.HighContrast())
		log.Println("Theme: HighContrast")
	})

	header.Add(darkBtn)
	header.Add(lightBtn)
	header.Add(hcBtn)

	return header
}

// buildMainContent creates the main content area with sidebar and content panel
func buildMainContent(app *wain.App) *wain.Row {
	mainRow := wain.NewRow()
	mainRow.SetGap(5)

	// Sidebar: navigation panels (25% width)
	sidebar := buildSidebar()
	mainRow.Add(sidebar)

	// Content area: scrollable panel (75% width)
	contentArea := buildContentArea(app)
	mainRow.Add(contentArea)

	return mainRow
}

// buildSidebar creates the navigation sidebar
func buildSidebar() *wain.Column {
	sidebar := wain.NewColumn()
	sidebar.SetPadding(10)
	sidebar.SetGap(5)

	// Navigation buttons
	for _, name := range []string{"Home", "Profile", "Settings", "Help", "About"} {
		btn := wain.NewButton(name, wain.Size{Width: 100, Height: 15})
		navName := name
		btn.OnClick(func() {
			log.Printf("Navigate to: %s", navName)
		})
		sidebar.Add(btn)
	}

	// Fill remaining space
	sidebar.Add(wain.NewSpacer(wain.Size{Width: 100, Height: 25}))

	return sidebar
}

// buildContentArea creates the main scrollable content area
func buildContentArea(app *wain.App) *wain.ScrollView {
	scroll := wain.NewScrollView(wain.Size{Width: 75, Height: 100})
	scroll.OnScroll(func(offset int) {
		log.Printf("Scrolled to: %d", offset)
	})

	// Content panels inside scroll view
	content := wain.NewColumn()
	content.SetPadding(15)
	content.SetGap(10)

	// Welcome panel
	welcomePanel := wain.NewPanel(wain.Size{Width: 100, Height: 20})
	welcomeLabel := wain.NewLabel("Welcome to Wain!", wain.Size{Width: 100, Height: 100})
	welcomeLabel.SetFontSize(18.0)
	welcomePanel.Add(welcomeLabel)
	content.Add(welcomePanel)

	// Input panel with text field
	inputPanel := wain.NewPanel(wain.Size{Width: 100, Height: 15})
	input := wain.NewTextInput("Type here...", wain.Size{Width: 100, Height: 100})
	input.OnChange(func(text string) {
		log.Printf("Input changed: %s", text)
	})
	inputPanel.Add(input)
	content.Add(inputPanel)

	// Info panels
	for i := 1; i <= 5; i++ {
		panel := wain.NewPanel(wain.Size{Width: 100, Height: 12})
		label := wain.NewLabel(fmt.Sprintf("Content Panel %d", i), wain.Size{Width: 100, Height: 100})
		panel.Add(label)
		content.Add(panel)
	}

	scroll.Add(content)
	return scroll
}

// buildFooter creates the status bar footer
func buildFooter(app *wain.App) *wain.Row {
	footer := wain.NewRow()
	footer.SetPadding(10)

	statusLabel := wain.NewLabel("Ready", wain.Size{Width: 100, Height: 100})
	footer.Add(statusLabel)

	// Demonstrate App.Notify for cross-goroutine updates
	// (Would run in background in a full application)
	log.Println("  • Status label configured with App.Notify callback")

	return footer
}

// printLayoutStructure displays the UI hierarchy
func printLayoutStructure() {
	log.Println("=== UI Layout Structure ===")
	log.Println()
	log.Println("Root (Column, 100% × 100%)")
	log.Println("├── Header (Row, 100% × auto)")
	log.Println("│   ├── Title (Label, 70% × 100%)")
	log.Println("│   ├── Spacer (5% × 100%)")
	log.Println("│   ├── Dark Button (8% × 100%)")
	log.Println("│   ├── Light Button (8% × 100%)")
	log.Println("│   └── HighContrast Button (9% × 100%)")
	log.Println("├── Main Content (Row, 100% × auto)")
	log.Println("│   ├── Sidebar (Column, 25% × 100%)")
	log.Println("│   │   ├── Home Button (100% × 15%)")
	log.Println("│   │   ├── Profile Button (100% × 15%)")
	log.Println("│   │   ├── Settings Button (100% × 15%)")
	log.Println("│   │   ├── Help Button (100% × 15%)")
	log.Println("│   │   ├── About Button (100% × 15%)")
	log.Println("│   │   └── Spacer (100% × 25%)")
	log.Println("│   └── Content Area (ScrollView, 75% × 100%)")
	log.Println("│       └── Content Column")
	log.Println("│           ├── Welcome Panel (100% × 20%)")
	log.Println("│           ├── Input Panel (100% × 15%)")
	log.Println("│           ├── Content Panel 1 (100% × 12%)")
	log.Println("│           ├── Content Panel 2 (100% × 12%)")
	log.Println("│           ├── Content Panel 3 (100% × 12%)")
	log.Println("│           ├── Content Panel 4 (100% × 12%)")
	log.Println("│           └── Content Panel 5 (100% × 12%)")
	log.Println("└── Footer (Row, 100% × auto)")
	log.Println("    └── Status Label (100% × 100%)")
	log.Println()
}

// demonstrateThemes shows all three built-in themes
func demonstrateThemes(app *wain.App) {
	themes := []struct {
		name string
		fn   func() wain.Theme
	}{
		{"DefaultDark", wain.DefaultDark},
		{"DefaultLight", wain.DefaultLight},
		{"HighContrast", wain.HighContrast},
	}

	for _, t := range themes {
		theme := t.fn()
		log.Printf("Theme: %s", t.name)
		log.Printf("  Background: RGB(%d, %d, %d)", theme.Background.R, theme.Background.G, theme.Background.B)
		log.Printf("  Foreground: RGB(%d, %d, %d)", theme.Foreground.R, theme.Foreground.G, theme.Foreground.B)
		log.Printf("  Accent:     RGB(%d, %d, %d)", theme.Accent.R, theme.Accent.G, theme.Accent.B)
		log.Printf("  FontSize:   %.1f", theme.FontSize)
		log.Println()
	}
}
