package demo

import (
	"fmt"

	"github.com/opd-ai/wain/internal/raster/core"
)

// CreateDemoBuffer creates a render buffer with the standard demo dimensions.
func CreateDemoBuffer(width, height int) (*core.Buffer, error) {
	buf, err := core.NewBuffer(width, height)
	if err != nil {
		return nil, fmt.Errorf("create buffer: %w", err)
	}
	return buf, nil
}
