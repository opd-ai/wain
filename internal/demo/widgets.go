package demo

import (
	"github.com/opd-ai/wain/internal/ui/widgets"
)

// StandardWidgets creates the standard demo widgets used across demo applications.
// Returns a Button and TextInput widget with consistent sizes and labels.
func StandardWidgets() (*widgets.Button, *widgets.TextInput) {
	btn := widgets.NewButton("Click Me!", 120, 40)
	input := widgets.NewTextInput("Type here...", 200, 30)
	return btn, input
}
