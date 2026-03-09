// Package main demonstrates the callback and cross-goroutine notification features.
//
// This demo shows:
//  1. Button.OnClick() callbacks
//  2. TextInput.OnChange() callbacks
//  3. ScrollView.OnScroll() callbacks
//  4. App.Notify() for safe cross-goroutine UI updates
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/opd-ai/wain"
)

func main() {
	// Create application
	app := wain.NewApp()

	// Note: This demo requires Run() to be called to start the event loop
	// For now, we demonstrate the callback API structure

	// 1. Button with onClick callback
	btn := wain.NewButton("Click me!", wain.Size{Width: 30, Height: 8})
	clickCount := 0
	btn.OnClick(func() {
		clickCount++
		fmt.Printf("Button clicked! Count: %d\n", clickCount)
	})

	// 2. TextInput with onChange callback
	input := wain.NewTextInput("", wain.Size{Width: 50, Height: 6})
	input.SetPlaceholder("Type something...")
	input.OnChange(func(text string) {
		fmt.Printf("Input changed: %s\n", text)
	})

	// 3. ScrollView with onScroll callback
	scroll := wain.NewScrollView(wain.Size{Width: 100, Height: 80})
	scroll.OnScroll(func(offset int) {
		fmt.Printf("Scrolled to offset: %d\n", offset)
	})

	// 4. Cross-goroutine notification example
	// Simulate a background worker that updates UI via Notify()
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(time.Second)
			counter := i + 1

			// Safe cross-goroutine UI update
			app.Notify(func() {
				statusText := fmt.Sprintf("Background task: %d/5", counter)
				log.Printf("UI update from background goroutine: %s", statusText)
			})
		}
	}()

	// Demonstrate immediate callback execution
	fmt.Println("=== Callback Demonstration ===")
	fmt.Println()

	fmt.Println("1. Button callbacks are registered")
	fmt.Println("   (In a real app, callbacks fire on user interaction)")
	fmt.Println()

	fmt.Println("2. Simulating text input changes:")
	input.SetText("Hello")
	input.SetText("Hello, World!")
	fmt.Println()

	fmt.Println("3. Simulating scroll events:")
	scroll.SetScrollOffset(50)
	scroll.SetScrollOffset(100)
	fmt.Println()

	fmt.Println("4. Cross-goroutine notifications:")
	fmt.Println("   Background goroutine will send 5 notifications over 5 seconds")

	// Process pending notifications for demo purposes
	// In a real app, this happens automatically in the event loop
	time.Sleep(6 * time.Second)

	fmt.Println()
	fmt.Println("=== Demo Complete ===")
	fmt.Println()
	fmt.Println("Key Features:")
	fmt.Println("  • Button.OnClick(func()) - respond to clicks")
	fmt.Println("  • TextInput.OnChange(func(text string)) - track input changes")
	fmt.Println("  • ScrollView.OnScroll(func(offset int)) - monitor scrolling")
	fmt.Println("  • App.Notify(func()) - safe cross-goroutine UI updates")
	fmt.Println()
	fmt.Println("To see this in a real UI, call app.Run() which starts the event loop")
}
