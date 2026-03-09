package wain

import (
	"sync"
	"testing"
	"time"
)

// TestNotifyBasic verifies that Notify queues and executes callbacks.
func TestNotifyBasic(t *testing.T) {
	app := NewApp()

	executed := false
	callback := func() {
		executed = true
	}

	app.Notify(callback)

	// Process notifications
	app.processNotifications()

	if !executed {
		t.Error("Notify callback was not executed")
	}
}

// TestNotifyNil verifies that Notify handles nil callbacks gracefully.
func TestNotifyNil(t *testing.T) {
	app := NewApp()

	// Should not panic or block
	app.Notify(nil)
	app.processNotifications()
}

// TestNotifyMultiple verifies that multiple notifications are processed in order.
func TestNotifyMultiple(t *testing.T) {
	app := NewApp()

	var order []int
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		val := i
		app.Notify(func() {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, val)
		})
	}

	app.processNotifications()

	mu.Lock()
	defer mu.Unlock()

	if len(order) != 10 {
		t.Errorf("Expected 10 callbacks executed, got %d", len(order))
	}

	for i := 0; i < 10; i++ {
		if order[i] != i {
			t.Errorf("Expected callback %d at position %d, got %d", i, i, order[i])
		}
	}
}

// TestNotifyCrossGoroutine verifies safe cross-goroutine UI updates.
func TestNotifyCrossGoroutine(t *testing.T) {
	app := NewApp()

	var wg sync.WaitGroup
	const numGoroutines = 50

	counter := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Simulate background work
			time.Sleep(time.Millisecond)
			// Schedule UI update via Notify
			app.Notify(func() {
				mu.Lock()
				counter++
				mu.Unlock()
			})
		}()
	}

	// Wait for all goroutines to schedule notifications
	wg.Wait()

	// Process all notifications (simulating event loop)
	app.processNotifications()

	mu.Lock()
	finalCount := counter
	mu.Unlock()

	if finalCount != numGoroutines {
		t.Errorf("Expected %d callbacks executed, got %d", numGoroutines, finalCount)
	}
}

// TestNotifyWidgetUpdate verifies that widget updates via Notify are safe.
func TestNotifyWidgetUpdate(t *testing.T) {
	app := NewApp()

	// Create widgets
	btn := NewButton("Initial", Size{Width: 30, Height: 10})
	label := NewLabel("Initial Label", Size{Width: 50, Height: 5})
	input := NewTextInput("", Size{Width: 40, Height: 6})

	// Simulate background goroutine updating widgets
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		app.Notify(func() {
			btn.SetText("Updated Button")
		})
	}()

	go func() {
		defer wg.Done()
		app.Notify(func() {
			label.SetText("Updated Label")
		})
	}()

	go func() {
		defer wg.Done()
		app.Notify(func() {
			input.SetText("Updated Input")
		})
	}()

	wg.Wait()
	app.processNotifications()

	// Verify updates
	if btn.Text() != "Updated Button" {
		t.Errorf("Button text not updated: %s", btn.Text())
	}
	if label.Text() != "Updated Label" {
		t.Errorf("Label text not updated: %s", label.Text())
	}
	if input.Text() != "Updated Input" {
		t.Errorf("TextInput text not updated: %s", input.Text())
	}
}

// TestNotifyChannelCapacity verifies that the notification channel has proper capacity.
func TestNotifyChannelCapacity(t *testing.T) {
	app := NewApp()

	// Queue exactly the channel capacity (100) without processing
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			app.Notify(func() {})
		}
		done <- true
	}()

	select {
	case <-done:
		// Success - 100 notifications queued without blocking
	case <-time.After(time.Second):
		t.Fatal("Notify blocked before reaching channel capacity")
	}

	// Verify all were queued
	app.processNotifications()
}

// TestNotifyWithCallbacks verifies integration with widget callbacks.
func TestNotifyWithCallbacks(t *testing.T) {
	app := NewApp()

	clickCount := 0
	btn := NewButton("Click Me", Size{Width: 30, Height: 10})
	btn.OnClick(func() {
		clickCount++
	})

	// Simulate background goroutine triggering clicks via Notify
	for i := 0; i < 5; i++ {
		app.Notify(func() {
			if btn.onClick != nil {
				btn.onClick()
			}
		})
	}

	app.processNotifications()

	if clickCount != 5 {
		t.Errorf("Expected 5 button clicks, got %d", clickCount)
	}
}
