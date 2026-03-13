package demo

import (
	"fmt"
	"strings"

	"github.com/opd-ai/wain"
)

// LogPointerEvent creates an OnPointer handler that logs pointer events to stdout.
func LogPointerEvent() func(*wain.PointerEvent) {
	return func(e *wain.PointerEvent) {
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
	}
}

// LogKeyPress creates an OnKeyPress handler that logs key press events with optional app quit on Escape.
func LogKeyPress(app *wain.App, quitOnEscape bool) func(*wain.KeyEvent) {
	return func(e *wain.KeyEvent) {
		keyName := KeyToString(e.Key())
		modifiers := ModifiersToString(e.Modifiers())
		if modifiers != "" {
			fmt.Printf("KeyPress: %s+%s (rune='%c')\n", modifiers, keyName, e.Rune())
		} else {
			fmt.Printf("KeyPress: %s (rune='%c')\n", keyName, e.Rune())
		}

		if quitOnEscape && e.Key() == wain.KeyEscape {
			fmt.Println("Escape pressed, quitting...")
			app.Quit()
		}
	}
}

// LogKeyRelease creates an OnKeyRelease handler that logs key release events to stdout.
func LogKeyRelease() func(*wain.KeyEvent) {
	return func(e *wain.KeyEvent) {
		keyName := KeyToString(e.Key())
		modifiers := ModifiersToString(e.Modifiers())
		if modifiers != "" {
			fmt.Printf("KeyRelease: %s+%s\n", modifiers, keyName)
		} else {
			fmt.Printf("KeyRelease: %s\n", keyName)
		}
	}
}

// LogTouch creates an OnTouch handler that logs touch events to stdout.
func LogTouch() func(*wain.TouchEvent) {
	return func(e *wain.TouchEvent) {
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
	}
}

// LogResize creates an OnResize handler that logs window resize events to stdout.
func LogResize() func(int, int) {
	return func(width, height int) {
		fmt.Printf("WindowResize: %dx%d\n", width, height)
	}
}

// LogClose creates an OnClose handler that logs window close events and quits the app.
func LogClose(app *wain.App) func() {
	return func() {
		fmt.Println("WindowClose: user requested close")
		app.Quit()
	}
}

// LogFocus creates an OnFocus handler that logs window focus events to stdout.
func LogFocus() func(bool) {
	return func(focused bool) {
		if focused {
			fmt.Println("WindowFocus: gained focus")
		} else {
			fmt.Println("WindowFocus: lost focus")
		}
	}
}

// LogScaleChange creates an OnScaleChange handler that logs window scale changes to stdout.
func LogScaleChange() func(float64) {
	return func(scale float64) {
		fmt.Printf("WindowScaleChange: scale=%.2f\n", scale)
	}
}

// KeyToString converts a wain.Key to a human-readable string.
func KeyToString(key wain.Key) string {
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

// modifierBits maps each modifier bitmask to its display name.
var modifierBits = []struct {
	mask wain.Modifier
	name string
}{
	{wain.ModShift, "Shift"},
	{wain.ModControl, "Ctrl"},
	{wain.ModAlt, "Alt"},
	{wain.ModSuper, "Super"},
}

// ModifiersToString converts wain.Modifier flags to a human-readable string.
func ModifiersToString(mods wain.Modifier) string {
	parts := make([]string, 0, len(modifierBits))
	for _, mb := range modifierBits {
		if mods&mb.mask != 0 {
			parts = append(parts, mb.name)
		}
	}
	return strings.Join(parts, "+")
}
