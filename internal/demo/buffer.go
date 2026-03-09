package demo

import (
	"fmt"

	"github.com/opd-ai/wain/internal/raster/primitives"
)

// CreateDemoBuffer creates a render buffer with the standard demo dimensions.
func CreateDemoBuffer(width, height int) (*primitives.Buffer, error) {
	buf, err := primitives.NewBuffer(width, height)
	if err != nil {
		return nil, fmt.Errorf("create buffer: %w", err)
	}
	return buf, nil
}
