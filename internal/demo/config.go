package demo

import (
	"log"

	"github.com/opd-ai/wain"
)

// DefaultWindowConfig returns a standard window configuration for demo applications.
// Uses sensible defaults: 800×600 window dimensions, no size constraints.
func DefaultWindowConfig(title string) wain.WindowConfig {
	return wain.WindowConfig{
		Title:  title,
		Width:  800,
		Height: 600,
	}
}

// LargeWindowConfig returns a larger window configuration suitable for complex demos.
// Uses 1024×768 dimensions with optional min/max constraints.
func LargeWindowConfig(title string) wain.WindowConfig {
	return wain.WindowConfig{
		Title:     title,
		Width:     1024,
		Height:    768,
		MinWidth:  640,
		MinHeight: 480,
		MaxWidth:  1920,
		MaxHeight: 1080,
	}
}

// CreateWindow creates a new window with the given configuration and handles errors.
// On error, it logs a fatal message and terminates the program.
func CreateWindow(app *wain.App, config wain.WindowConfig) *wain.Window {
	win, err := app.NewWindow(config)
	if err != nil {
		log.Fatalf("Failed to create window: %v", err)
	}
	return win
}

// CreateDefaultWindow creates a window with default demo dimensions (800×600).
// This is a convenience wrapper around CreateWindow with DefaultWindowConfig.
func CreateDefaultWindow(app *wain.App, title string) *wain.Window {
	return CreateWindow(app, DefaultWindowConfig(title))
}

// CreateLargeWindow creates a window with large demo dimensions (1024×768).
// This is a convenience wrapper around CreateWindow with LargeWindowConfig.
func CreateLargeWindow(app *wain.App, title string) *wain.Window {
	return CreateWindow(app, LargeWindowConfig(title))
}
