// Command wain is the entry point for the wain UI toolkit.
//
// At this stage (Phase 0) it exercises the Go → Rust static-library link by
// calling the trivial render_add function and printing the result.
//
// Usage:
//
//	./bin/wain              # Run validation test
//	./bin/wain --help       # Show help message
//	./bin/wain --version    # Show version information
package main

import (
	"fmt"
	"os"

	"github.com/opd-ai/wain/internal/render"
)

const usage = `wain - A statically-compiled Go UI toolkit with GPU rendering via Rust

Usage:
  wain              Run validation test (render.Add and render.Version)
  wain --help       Show this help message
  wain --version    Show version information

At this stage (Phase 0), wain exercises the Go → Rust static-library link
by calling the render_add function and printing the result.

For more information, see: https://github.com/opd-ai/wain`

// main exercises the Go → Rust static-library link by calling render.Add
// and render.Version, then prints the results to demonstrate successful linkage.
func main() {
	// Handle flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h", "help":
			fmt.Println(usage)
			return
		case "--version", "-v", "version":
			fmt.Printf("wain version: %s\n", render.Version())
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n\n", os.Args[1])
			fmt.Println(usage)
			os.Exit(1)
		}
	}

	// Run validation test
	a, b := int32(6), int32(7)
	result := render.Add(a, b)
	fmt.Printf("render.Add(%d, %d) = %d\n", a, b, result)
	fmt.Printf("render library version: %s\n", render.Version())
}
