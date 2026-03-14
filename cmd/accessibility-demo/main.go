// accessibility-demo demonstrates AT-SPI2 accessibility integration in wain.
//
// This binary registers a simple form UI with the AT-SPI2 registry, making
// it visible to Orca screen reader and Accerciser accessibility inspector.
//
// # Running
//
//	./accessibility-demo
//
// # Testing with Accerciser
//
//	accerciser &       # open inspector
//	./accessibility-demo
//
// The application will appear in Accerciser's tree as
// "org.a11y.atspi.accessible.accessibility-demo" and expose:
//   - Root panel containing a form
//   - Name label, name text entry
//   - Submit button (activate via Accerciser → Actions → click)
//
// # Testing with Orca
//
// Launch Orca before running this demo. Navigate the UI with Tab and arrow
// keys; Orca will announce each widget's role and name.
package main

import (
	"log"
	"time"

	"github.com/opd-ai/wain"
)

func main() {
	am := wain.EnableAccessibility("accessibility-demo")
	if am == nil {
		log.Println("accessibility-demo: D-Bus not available; running without AT-SPI2")
		runHeadless()
		return
	}
	defer am.Close()

	buildAccessibleTree(am)

	log.Println("accessibility-demo: AT-SPI2 tree registered; press Ctrl-C to exit")
	log.Println("  Inspect with: accerciser or orca")

	// Keep the process alive so assistive tools can introspect the tree.
	select {}
}

// buildAccessibleTree creates and registers the demo UI as accessible objects.
func buildAccessibleTree(am *wain.AccessibilityManager) {
	// Root panel — the application window.
	rootID := am.RegisterPanel("accessibility-demo", 0)
	am.SetBounds(rootID, 0, 0, 400, 300)

	// Heading label.
	headingID := am.RegisterLabel("Simple Form", rootID)
	am.SetBounds(headingID, 10, 10, 380, 30)

	// Name field row.
	nameLabelID := am.RegisterLabel("Name", rootID)
	am.SetBounds(nameLabelID, 10, 60, 80, 25)

	nameEntryID := am.RegisterEntry("Name", rootID)
	am.SetBounds(nameEntryID, 100, 60, 280, 25)
	am.SetText(nameEntryID, "")

	// Email field row.
	emailLabelID := am.RegisterLabel("Email", rootID)
	am.SetBounds(emailLabelID, 10, 100, 80, 25)

	emailEntryID := am.RegisterEntry("Email", rootID)
	am.SetBounds(emailEntryID, 100, 100, 280, 25)
	am.SetText(emailEntryID, "")

	// Submit button.
	submitted := false
	submitID := am.RegisterButton("Submit", rootID, func() bool {
		submitted = true
		log.Println("accessibility-demo: Submit activated via AT-SPI2")
		return true
	})
	am.SetBounds(submitID, 10, 150, 100, 35)

	// Cancel button.
	cancelID := am.RegisterButton("Cancel", rootID, func() bool {
		log.Println("accessibility-demo: Cancel activated via AT-SPI2")
		return true
	})
	am.SetBounds(cancelID, 120, 150, 100, 35)

	// Simulate focus cycling through fields (visible in Accerciser/Orca).
	go simulateFocusCycle(am, nameEntryID, emailEntryID, submitID, cancelID, &submitted)

	_ = headingID
	_ = nameLabelID
	_ = emailLabelID
}

// simulateFocusCycle cycles keyboard focus across fields to demonstrate
// AT-SPI2 focus events being fired. Stops when Submit is activated.
func simulateFocusCycle(
	am *wain.AccessibilityManager,
	nameID, emailID, submitID, cancelID uint64,
	done *bool,
) {
	fields := []uint64{nameID, emailID, submitID, cancelID}
	for !*done {
		for _, id := range fields {
			if *done {
				return
			}
			am.SetFocused(id, true)
			time.Sleep(2 * time.Second)
			am.SetFocused(id, false)
		}
	}
}

// runHeadless runs the demo for 5 s when D-Bus is unavailable,
// demonstrating that the application continues without accessibility.
func runHeadless() {
	log.Println("accessibility-demo: running for 5 s in headless mode")
	time.Sleep(5 * time.Second)
	log.Println("accessibility-demo: done")
}
