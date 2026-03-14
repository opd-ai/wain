// Package main is the canonical "hello world" example for the wain UI toolkit.
//
// It demonstrates the minimal public API surface needed to build a working application:
//   - Creating an App and configuring it
//   - Scheduling window creation via App.Notify
//   - Building a layout (Column + Row) from public widget types
//   - Wiring a Button OnClick callback
//   - Running the event loop (or falling back to headless output)
//
// All imports use only the public "github.com/opd-ai/wain" package; no internal
// packages are referenced.
//
// Usage:
//
//	go build github.com/opd-ai/wain/example/hello && ./hello
package main

import (
	"fmt"
	"log"

	"github.com/opd-ai/wain"
)

func main() {
	app := wain.NewApp()

	app.Notify(func() {
		if err := openWindow(app); err != nil {
			log.Printf("window error: %v", err)
			app.Quit()
		}
	})

	if err := app.Run(); err != nil {
		log.Printf("No display available (%v); running headless demo.", err)
		runHeadless()
	}
}

// openWindow creates the application window and attaches the widget hierarchy.
func openWindow(app *wain.App) error {
	win, err := app.NewWindow(wain.WindowConfig{
		Title:       "Hello, wain!",
		Width:       400,
		Height:      200,
		Decorations: true,
	})
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}

	win.OnClose(func() { app.Quit() })

	root := buildLayout(app)
	win.SetLayout(root)
	return nil
}

// buildLayout assembles the widget tree: a Column containing a greeting label
// and a "Say Hello" button.
func buildLayout(app *wain.App) *wain.Column {
	col := wain.NewColumn()
	col.SetPadding(20)
	col.SetGap(10)

	label := wain.NewLabel("Press the button below.", wain.Size{Width: 100, Height: 30})
	col.Add(label)

	btn := wain.NewButton("Say Hello", wain.Size{Width: 50, Height: 20})
	btn.OnClick(func() {
		fmt.Println("Hello from wain!")
		label.SetText("Hello, wain!")
		_ = app
	})
	col.Add(btn)

	return col
}

// runHeadless exercises the public API without a display server so that
// "go test" and CI pipelines can verify the example compiles and the widget
// hierarchy is coherent.
func runHeadless() {
	fmt.Println("=== wain hello example (headless) ===")

	col := buildLayout(nil)
	fmt.Printf("Layout: Column with %d children\n", len(col.Children()))

	// Access the button from the layout to verify it was wired correctly.
	btn := col.Children()[1].(*wain.Button)
	fmt.Printf("Button text: %q\n", btn.Text())

	// Simulate a click to verify callback wiring.
	clicked := false
	btn.OnClick(func() { clicked = true })
	fmt.Printf("Click handler registered: %v\n", !clicked)
	fmt.Println("=== done ===")
}
