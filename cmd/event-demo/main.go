package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opd-ai/wain"
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

	win, err := app.NewWindow(wain.WindowConfig{
		Title:  "Event System Demo",
		Width:  800,
		Height: 600,
	})
	if err != nil {
		log.Fatalf("Failed to create window: %v", err)
	}

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
	win.OnPointer(func(e *wain.PointerEvent) {
		switch e.EventType() {
		case wain.PointerMove:
			fmt.Printf("PointerMove: x=%.2f y=%.2f\n", e.X(), e.Y())
		case wain.PointerButtonPress:
			fmt.Printf("PointerButtonPress: button=%d x=%.2f y=%.2f\n",
				e.Button(), e.X(), e.Y())
		case wain.PointerButtonRelease:
			fmt.Printf("PointerButtonRelease: button=%d x=%.2f y=%.2f\n",
				e.Button(), e.X(), e.Y())
		case wain.PointerScroll:
			fmt.Printf("PointerScroll: axis=%d value=%.2f\n",
				e.Axis(), e.Value())
		case wain.PointerEnter:
			fmt.Printf("PointerEnter: x=%.2f y=%.2f\n", e.X(), e.Y())
		case wain.PointerLeave:
			fmt.Printf("PointerLeave: x=%.2f y=%.2f\n", e.X(), e.Y())
		}
	})

	win.OnKeyPress(func(e *wain.KeyEvent) {
		keyName := keyToString(e.Key())
		modifiers := modifiersToString(e.Modifiers())
		if modifiers != "" {
			fmt.Printf("KeyPress: %s+%s (rune='%c')\n", modifiers, keyName, e.Rune())
		} else {
			fmt.Printf("KeyPress: %s (rune='%c')\n", keyName, e.Rune())
		}

		if e.Key() == wain.KeyEscape {
			fmt.Println("Escape pressed, quitting...")
			app.Quit()
		}
	})

	win.OnKeyRelease(func(e *wain.KeyEvent) {
		keyName := keyToString(e.Key())
		modifiers := modifiersToString(e.Modifiers())
		if modifiers != "" {
			fmt.Printf("KeyRelease: %s+%s\n", modifiers, keyName)
		} else {
			fmt.Printf("KeyRelease: %s\n", keyName)
		}
	})

	win.OnTouch(func(e *wain.TouchEvent) {
		switch e.EventType() {
		case wain.TouchDown:
			fmt.Printf("TouchDown: id=%d x=%.2f y=%.2f\n", e.ID(), e.X(), e.Y())
		case wain.TouchUp:
			fmt.Printf("TouchUp: id=%d x=%.2f y=%.2f\n", e.ID(), e.X(), e.Y())
		case wain.TouchMotion:
			fmt.Printf("TouchMotion: id=%d x=%.2f y=%.2f\n", e.ID(), e.X(), e.Y())
		case wain.TouchCancel:
			fmt.Printf("TouchCancel: id=%d\n", e.ID())
		}
	})

	win.OnResize(func(width, height int) {
		fmt.Printf("WindowResize: %dx%d\n", width, height)
	})

	win.OnClose(func() {
		fmt.Println("WindowClose: user requested close")
		app.Quit()
	})

	win.OnFocus(func(focused bool) {
		if focused {
			fmt.Println("WindowFocus: gained focus")
		} else {
			fmt.Println("WindowFocus: lost focus")
		}
	})

	win.OnScaleChange(func(scale float64) {
		fmt.Printf("WindowScaleChange: scale=%.2f\n", scale)
	})
}

func keyToString(key wain.Key) string {
	switch key {
	case wain.KeyEscape:
		return "Escape"
	case wain.KeyReturn:
		return "Return"
	case wain.KeyTab:
		return "Tab"
	case wain.KeyBackspace:
		return "Backspace"
	case wain.KeyDelete:
		return "Delete"
	case wain.KeyLeft:
		return "Left"
	case wain.KeyUp:
		return "Up"
	case wain.KeyRight:
		return "Right"
	case wain.KeyDown:
		return "Down"
	case wain.KeyHome:
		return "Home"
	case wain.KeyEnd:
		return "End"
	case wain.KeyPageUp:
		return "PageUp"
	case wain.KeyPageDown:
		return "PageDown"
	case wain.KeySpace:
		return "Space"
	case wain.KeyShiftL:
		return "ShiftL"
	case wain.KeyShiftR:
		return "ShiftR"
	case wain.KeyControlL:
		return "ControlL"
	case wain.KeyControlR:
		return "ControlR"
	case wain.KeyAltL:
		return "AltL"
	case wain.KeyAltR:
		return "AltR"
	case wain.KeySuperL:
		return "SuperL"
	case wain.KeySuperR:
		return "SuperR"
	default:
		if key >= 32 && key < 127 {
			return fmt.Sprintf("%c", rune(key))
		}
		return fmt.Sprintf("0x%X", key)
	}
}

func modifiersToString(mods wain.Modifier) string {
	parts := []string{}
	if mods&wain.ModShift != 0 {
		parts = append(parts, "Shift")
	}
	if mods&wain.ModControl != 0 {
		parts = append(parts, "Ctrl")
	}
	if mods&wain.ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	if mods&wain.ModSuper != 0 {
		parts = append(parts, "Super")
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += "+"
		}
		result += part
	}
	return result
}
