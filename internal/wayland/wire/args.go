package wire

import "fmt"

// ParseArgMinLen returns an error if args has fewer than minLen elements.
func ParseArgMinLen(args []Argument, minLen int, context string) error {
	if len(args) < minLen {
		return fmt.Errorf("%s: requires %d arguments, got %d", context, minLen, len(args))
	}
	return nil
}

// ParseArgUint32 type-asserts args[idx].Value to uint32 or returns an error with label.
func ParseArgUint32(args []Argument, idx int, label string) (uint32, error) {
	v, ok := args[idx].Value.(uint32)
	if !ok {
		return 0, fmt.Errorf("%s must be uint32", label)
	}
	return v, nil
}

// ParseArgInt32 type-asserts args[idx].Value to int32 or returns an error with label.
func ParseArgInt32(args []Argument, idx int, label string) (int32, error) {
	v, ok := args[idx].Value.(int32)
	if !ok {
		return 0, fmt.Errorf("%s must be int32", label)
	}
	return v, nil
}

// ParseArgInt type-asserts args[idx].Value to int or returns an error with label.
func ParseArgInt(args []Argument, idx int, label string) (int, error) {
	v, ok := args[idx].Value.(int)
	if !ok {
		return 0, fmt.Errorf("%s must be int", label)
	}
	return v, nil
}

// ParseArgBytes type-asserts args[idx].Value to []byte or returns an error with label.
func ParseArgBytes(args []Argument, idx int, label string) ([]byte, error) {
	v, ok := args[idx].Value.([]byte)
	if !ok {
		return nil, fmt.Errorf("%s must be []byte", label)
	}
	return v, nil
}

// ParseArgString type-asserts args[idx].Value to string or returns an error with label.
func ParseArgString(args []Argument, idx int, label string) (string, error) {
	v, ok := args[idx].Value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be string", label)
	}
	return v, nil
}

// ArgDecoder decodes Argument slices sequentially, accumulating the first error.
// If a decode call fails, subsequent calls become no-ops and the error is
// retrievable via Err(). This eliminates per-argument error boilerplate.
type ArgDecoder struct {
	args []Argument
	idx  int
	err  error
}

// NewArgDecoder creates an ArgDecoder for args starting at index 0.
func NewArgDecoder(args []Argument) *ArgDecoder {
	return &ArgDecoder{args: args}
}

// Err returns the first decoding error, or nil if all decodes succeeded.
func (d *ArgDecoder) Err() error { return d.err }

// Uint32 reads the next argument as uint32, recording any type mismatch.
func (d *ArgDecoder) Uint32(label string) uint32 {
	if d.err != nil {
		return 0
	}
	v, err := ParseArgUint32(d.args, d.idx, label)
	d.idx++
	d.err = err
	return v
}

// Int32 reads the next argument as int32, recording any type mismatch.
func (d *ArgDecoder) Int32(label string) int32 {
	if d.err != nil {
		return 0
	}
	v, err := ParseArgInt32(d.args, d.idx, label)
	d.idx++
	d.err = err
	return v
}

// Int reads the next argument as int, recording any type mismatch.
func (d *ArgDecoder) Int(label string) int {
	if d.err != nil {
		return 0
	}
	v, err := ParseArgInt(d.args, d.idx, label)
	d.idx++
	d.err = err
	return v
}

// Bytes reads the next argument as []byte, recording any type mismatch.
func (d *ArgDecoder) Bytes(label string) []byte {
	if d.err != nil {
		return nil
	}
	v, err := ParseArgBytes(d.args, d.idx, label)
	d.idx++
	d.err = err
	return v
}

// String reads the next argument as string, recording any type mismatch.
func (d *ArgDecoder) String(label string) string {
	if d.err != nil {
		return ""
	}
	v, err := ParseArgString(d.args, d.idx, label)
	d.idx++
	d.err = err
	return v
}
