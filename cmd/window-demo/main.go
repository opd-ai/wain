// Command window-demo demonstrates the public Window API (Phase 9.2).
//
// This demo validates the WINDOW ABSTRACTION milestone:
//   - Create a public Window type wrapping internal Wayland/X11 windows
//   - Support window properties: title, initial size, min/max constraints, fullscreen toggle
//   - Window dimensions specified as pixel defaults with automatic HiDPI scaling
//
// The demo creates a titled window with custom initial dimensions.
package main

import (
	"log"
	"time"

	"github.com/opd-ai/wain"
	"github.com/opd-ai/wain/internal/demo"
)

func main() {
	app := demo.SetupApp()

	errChan := make(chan error, 1)
	go func() {
		log.Println("Starting wain application...")
		errChan <- app.Run()
	}()

	time.Sleep(100 * time.Millisecond)

	log.Println("Creating window...")
	win := demo.CreateLargeWindow(app, "wain Window Demo")

	log.Printf("Window created: %s", win.Title())
	w, h := win.Size()
	log.Printf("Window size: %dx%d", w, h)
	log.Printf("Window scale: %.1f", win.Scale())

	win.OnResize(demo.LogResize())
	win.OnClose(demo.LogClose(app))
	win.OnFocus(demo.LogFocus())
	win.OnScaleChange(demo.LogScaleChange())

	demonstrateWindowOperations(win)

	log.Println("\nWindow demo running. Press Ctrl+C to exit.")

	waitForAppExit(errChan, app)
}

// demonstrateWindowOperations exercises title, min-size, and max-size mutations,
// logging the outcome of each call.
func demonstrateWindowOperations(win *wain.Window) {
	log.Println("\nDemonstrating window operations:")

	if err := win.SetTitle("Updated Title"); err != nil {
		log.Printf("Failed to set title: %v", err)
	} else {
		log.Printf("Title updated: %s", win.Title())
	}

	if err := win.SetMinSize(800, 600); err != nil {
		log.Printf("Failed to set min size: %v", err)
	} else {
		log.Println("Min size updated to 800x600")
	}

	if err := win.SetMaxSize(1600, 1200); err != nil {
		log.Printf("Failed to set max size: %v", err)
	} else {
		log.Println("Max size updated to 1600x1200")
	}
}

// waitForAppExit blocks until the app exits cleanly or a signal is received,
// then logs the final display-server and backend state.
func waitForAppExit(errChan <-chan error, app *wain.App) {
	if err := <-errChan; err != nil {
		log.Fatalf("App error: %v", err)
	}

	log.Printf("Display server: %s", app.DisplayServer())
	log.Printf("Backend type: %s", app.BackendType())
	log.Println("Application exited cleanly")
}
