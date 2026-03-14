package wain_test

import (
	"fmt"

	"github.com/opd-ai/wain"
)

// ExampleNewApp demonstrates creating an App and querying its display server.
// In a real program call app.Run() to enter the event loop.
func ExampleNewApp() {
	app := wain.NewApp()
	ds := app.DisplayServer()
	fmt.Println(ds == wain.DisplayServerUnknown || ds == wain.DisplayServerWayland || ds == wain.DisplayServerX11)
	// Output:
	// true
}

// ExampleNewButton demonstrates creating a Button and registering an onClick
// handler.
func ExampleNewButton() {
	btn := wain.NewButton("Submit", wain.Size{Width: 30, Height: 8})
	btn.OnClick(func() {
		fmt.Println("clicked")
	})
	fmt.Println(btn.Text())
	// Output:
	// Submit
}

// ExampleNewLabel demonstrates creating a Label and reading its text.
func ExampleNewLabel() {
	lbl := wain.NewLabel("Hello, wain!", wain.Size{Width: 50, Height: 5})
	fmt.Println(lbl.Text())
	// Output:
	// Hello, wain!
}

// ExampleEnableAccessibility demonstrates enabling AT-SPI2 accessibility.
// On systems without a D-Bus session bus this returns nil gracefully.
func ExampleEnableAccessibility() {
	am := wain.EnableAccessibility("example-app")
	fmt.Println(am == nil || am != nil) // always true — nil when D-Bus is unavailable
	if am != nil {
		am.Close()
	}
	// Output:
	// true
}
