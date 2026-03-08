package demo

import (
	"fmt"
	"log"
)

// RunDemoWithSetup provides a common main() pattern for demo binaries.
// It checks for help flags, prints a banner, runs the demo function, and handles errors.
func RunDemoWithSetup(name, description string, examples []string, bannerText string, runFn func() error) {
	CheckHelpFlag(name, description, examples)

	fmt.Println("==============================================")
	fmt.Println(bannerText)
	fmt.Println("==============================================")
	fmt.Println()

	if err := runFn(); err != nil {
		log.Fatalf("Demo failed: %v", err)
	}

	fmt.Println("\n✓ Demo completed successfully!")
}
