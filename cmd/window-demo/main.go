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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opd-ai/wain"
	"github.com/opd-ai/wain/internal/demo"
)

func main() {
	log.SetFlags(0)

	// Create app with verbose logging
	cfg := wain.DefaultConfig()
	cfg.Verbose = true
	app := wain.NewAppWithConfig(cfg)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nShutdown signal received, exiting...")
		app.Quit()
	}()

	// Start the application in a goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Println("Starting wain application...")
		errChan <- app.Run()
	}()

	// Wait for initialization
	time.Sleep(100 * time.Millisecond)

	// Create a window with custom configuration
	log.Println("Creating window...")
	win, err := app.NewWindow(wain.WindowConfig{
		Title:     "wain Window Demo",
		Width:     1024,
		Height:    768,
		MinWidth:  640,
		MinHeight: 480,
		MaxWidth:  1920,
		MaxHeight: 1080,
	})
	if err != nil {
		log.Printf("Failed to create window: %v", err)
		app.Quit()
		os.Exit(1)
	}

	log.Printf("Window created: %s", win.Title())
	w, h := win.Size()
	log.Printf("Window size: %dx%d", w, h)
	log.Printf("Window scale: %.1f", win.Scale())

	win.OnResize(demo.LogResize())
	win.OnClose(demo.LogClose(app))
	win.OnFocus(demo.LogFocus())
	win.OnScaleChange(demo.LogScaleChange())

	log.Println("\nDemonstrating window operations:")

	// Change title
	if err := win.SetTitle("Updated Title"); err != nil {
		log.Printf("Failed to set title: %v", err)
	} else {
		log.Printf("Title updated: %s", win.Title())
	}

	// Update size constraints
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

	log.Println("\nWindow demo running. Press Ctrl+C to exit.")

	// Wait for app to finish or signal
	select {
	case err := <-errChan:
		if err != nil {
			log.Fatalf("App error: %v", err)
		}
	case <-sigChan:
		log.Println("Exiting...")
	}

	// Report final state
	log.Printf("Display server: %s", app.DisplayServer())
	log.Printf("Backend type: %s", app.BackendType())
	log.Println("Application exited cleanly")
}
