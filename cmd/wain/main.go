// Command wain is the entry point for the wain UI toolkit.
//
// At this stage (Phase 0) it exercises the Go → Rust static-library link by
// calling the trivial render_add function and printing the result.
package main

import (
	"fmt"

	"github.com/opd-ai/wain/internal/render"
)

// main exercises the Go → Rust static-library link by calling render.Add
// and render.Version, then prints the results to demonstrate successful linkage.
func main() {
	a, b := int32(6), int32(7)
	result := render.Add(a, b)
	fmt.Printf("render.Add(%d, %d) = %d\n", a, b, result)
	fmt.Printf("render library version: %s\n", render.Version())
}
