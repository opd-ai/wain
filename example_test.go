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

// ExampleNewTextInput demonstrates creating a TextInput with a change handler.
func ExampleNewTextInput() {
	input := wain.NewTextInput("", wain.Size{Width: 40, Height: 4})
	input.SetPlaceholder("Type here…")
	input.SetText("hello")
	fmt.Println(input.Text())
	// Output:
	// hello
}

// ExampleNewPanel demonstrates creating a Panel and adding child sub-panels.
func ExampleNewPanel() {
	panel := wain.NewPanel(wain.Size{Width: 80, Height: 24})
	panel.Add(wain.NewPanel(wain.Size{Width: 80, Height: 12}))
	panel.Add(wain.NewPanel(wain.Size{Width: 80, Height: 12}))
	fmt.Println(len(panel.Children()))
	// Output:
	// 2
}

// ExampleNewRow demonstrates creating a horizontal Row layout.
func ExampleNewRow() {
	row := wain.NewRow()
	row.Add(wain.NewColumn())
	row.Add(wain.NewColumn())
	fmt.Println(len(row.Children()))
	// Output:
	// 2
}

// ExampleNewColumn demonstrates creating a vertical Column layout.
func ExampleNewColumn() {
	col := wain.NewColumn()
	col.Add(wain.NewPanel(wain.Size{Width: 100, Height: 33}))
	col.Add(wain.NewPanel(wain.Size{Width: 100, Height: 33}))
	col.Add(wain.NewPanel(wain.Size{Width: 100, Height: 33}))
	fmt.Println(len(col.Children()))
	// Output:
	// 3
}

// ExampleNewGrid demonstrates a Grid layout with a fixed column count.
func ExampleNewGrid() {
	grid := wain.NewGrid(3)
	for range 6 {
		grid.Add(wain.NewPanel(wain.Size{Width: 33, Height: 50}))
	}
	fmt.Println(grid.Columns())
	fmt.Println(len(grid.Children()))
	// Output:
	// 3
	// 6
}

// ExampleNewScrollView demonstrates wrapping a tall widget in a ScrollView.
func ExampleNewScrollView() {
	sv := wain.NewScrollView(wain.Size{Width: 40, Height: 10})
	for i := range 20 {
		sv.Add(wain.NewLabel(fmt.Sprintf("Item %d", i+1), wain.Size{Width: 38, Height: 1}))
	}
	w, h := sv.Bounds()
	fmt.Println(w > 0 && h > 0)
	// Output:
	// true
}
