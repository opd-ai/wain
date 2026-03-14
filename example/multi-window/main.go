// Package main demonstrates opening multiple windows with the wain UI toolkit.
//
// It shows:
//   - Creating a second window from inside a Notify callback
//   - Using App.Windows() to enumerate all open windows
//   - Independent layouts and close callbacks per window
//
// Usage:
//
//	go build github.com/opd-ai/wain/example/multi-window && ./multi-window
package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain"
)

func main() {
	app := wain.NewApp()

	app.Notify(func() {
		if err := openWindows(app); err != nil {
			log.Printf("window error: %v", err)
			app.Quit()
		}
	})

	if err := app.Run(); err != nil {
		log.Printf("No display available (%v); running headless demo.", err)
		runHeadless()
	}
}

// openWindows creates two independent windows.
func openWindows(app *wain.App) error {
	open := 2 // track how many windows remain open
	quit := func() {
		open--
		if open <= 0 {
			app.Quit()
		}
	}

	if err := openWindow(app, "Window 1", quit); err != nil {
		return fmt.Errorf("window 1: %w", err)
	}
	if err := openWindow(app, "Window 2", quit); err != nil {
		return fmt.Errorf("window 2: %w", err)
	}

	fmt.Printf("Opened %d windows\n", len(app.Windows()))
	return nil
}

// openWindow creates one window with a label showing its title.
func openWindow(app *wain.App, title string, onClose func()) error {
	win, err := app.NewWindow(wain.WindowConfig{
		Title:       title,
		Width:       320,
		Height:      160,
		Decorations: true,
	})
	if err != nil {
		return err
	}

	win.OnClose(onClose)

	col := wain.NewColumn()
	col.SetPadding(20)
	col.SetGap(10)
	col.Add(wain.NewLabel(title, wain.Size{Width: 200, Height: 30}))
	col.Add(wain.NewLabel("Close this window to quit.", wain.Size{Width: 200, Height: 20}))
	win.SetLayout(col)
	return nil
}

// runHeadless exercises the multi-window API without a display server.
func runHeadless() {
	fmt.Println("=== wain multi-window example (headless) ===")

	app := wain.NewApp()

	col1 := wain.NewColumn()
	col1.Add(wain.NewLabel("Window 1", wain.Size{Width: 200, Height: 30}))

	col2 := wain.NewColumn()
	col2.Add(wain.NewLabel("Window 2", wain.Size{Width: 200, Height: 30}))

	// Verify Windows() returns an empty slice before Run().
	wins := app.Windows()
	fmt.Printf("Windows before Run: %d\n", len(wins))
	fmt.Println("=== done ===")
}
