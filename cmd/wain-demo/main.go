// Command wain-demo demonstrates the public wain API (Phase 9.1).
//
// This demo validates the APPLICATION LIFECYCLE milestone:
//   - Display server auto-detection (Wayland preferred, X11 fallback)
//   - Renderer auto-detection (Intel GPU → AMD GPU → software fallback)
//   - Event loop management (single-goroutine event dispatch)
//   - Graceful shutdown and resource cleanup
//
// The demo opens a blank window using wain.NewApp().Run().
package main

import (
"log"
"os"
"os/signal"
"syscall"

"github.com/opd-ai/wain"
)

func main() {
log.SetFlags(0)

// Create app with default configuration
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

// Run the application (blocks until Quit() is called)
log.Println("Starting wain application...")
if err := app.Run(); err != nil {
log.Fatalf("App error: %v", err)
}

// Report final state
log.Printf("Display server: %s", app.DisplayServer())
log.Printf("Backend type: %s", app.BackendType())
w, h := app.Dimensions()
log.Printf("Window size: %dx%d", w, h)

log.Println("Application exited cleanly")
}
