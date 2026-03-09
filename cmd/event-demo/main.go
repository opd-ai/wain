package main

import (
	"fmt"
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
	app := wain.NewApp()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutdown signal received, exiting...")
		app.Quit()
	}()

	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Run()
	}()

	time.Sleep(100 * time.Millisecond)

	win := demo.CreateDefaultWindow(app, "Event System Demo")

	setupEventHandlers(win, app)

	fmt.Printf("Event Demo started on %s\n", app.DisplayServer())
	fmt.Println("Move mouse, click, type keys, resize window to see events...")
	fmt.Println("Press Escape or Ctrl+C to exit.")

	select {
	case err := <-errChan:
		if err != nil {
			fmt.Fprintf(os.Stderr, "App error: %v\n", err)
			os.Exit(1)
		}
	case <-sigChan:
		fmt.Println("Exiting...")
	}
}

func setupEventHandlers(win *wain.Window, app *wain.App) {
	win.OnPointer(demo.LogPointerEvent())
	win.OnKeyPress(demo.LogKeyPress(app, true))
	win.OnKeyRelease(demo.LogKeyRelease())
	win.OnTouch(demo.LogTouch())
	win.OnResize(demo.LogResize())
	win.OnClose(demo.LogClose(app))
	win.OnFocus(demo.LogFocus())
	win.OnScaleChange(demo.LogScaleChange())
}
