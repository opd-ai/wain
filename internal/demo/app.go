package demo

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/opd-ai/wain"
)

// SetupApp creates a verbose wain.App and registers SIGINT/SIGTERM signal handlers
// that call app.Quit() for graceful shutdown. log.SetFlags(0) is called to
// suppress the timestamp prefix that clutters demo output.
func SetupApp() *wain.App {
	log.SetFlags(0)
	cfg := wain.DefaultConfig()
	cfg.Verbose = true
	app := wain.NewAppWithConfig(cfg)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("\nShutdown signal received, exiting...")
		app.Quit()
	}()
	return app
}
