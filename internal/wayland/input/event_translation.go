package input

import (
	"fmt"

	"github.com/opd-ai/wain/internal/wayland/wire"
)

// parseEvent validates the minimum argument count, creates an ArgDecoder,
// calls fn to decode each argument field, then returns any accumulated decode error.
// It is the canonical implementation of the boilerplate shared by every
// handleXxxEvent method in keyboard.go, pointer.go, and touch.go.
func parseEvent(args []wire.Argument, minArgs int, ctx string, fn func(*wire.ArgDecoder)) error {
	if err := wire.ParseArgMinLen(args, minArgs, ctx); err != nil {
		return fmt.Errorf("wayland/input: parse event: %w", err)
	}
	d := wire.NewArgDecoder(args)
	fn(d)
	return d.Err()
}
