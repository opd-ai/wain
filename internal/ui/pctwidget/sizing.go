// Package pctwidget implements a percentage-based widget system with automatic layout.
//
// This package provides a high-level widget abstraction layer that sits alongside
// the existing widgets and raster packages. Widgets are sized using percentages
// (0–100) of their parent container, enabling responsive layouts that adapt to
// window resizes. It includes its own auto-layout engine for zero-configuration
// positioning and a Style interface for pluggable visual customization.
//
// # Sizing Model
//
// All widget dimensions are expressed as percentages of the parent container:
//
//   - Width and Height range from 0.0 to 100.0 (percent of parent)
//   - Percentage values are converted to pixels at layout time
//   - The conversion depends on the current parent/window dimensions
//
// # Manual Override
//
// Widgets can optionally specify absolute pixel positions and sizes to override
// the automatic percentage-based layout when precise control is needed.
//
// # Coordinate System
//
// Same as the raster packages: origin (0,0) at top-left,
// X increases right, Y increases down.
package pctwidget

import "errors"

var (
	// ErrInvalidPercentage is returned when a percentage value is out of range.
	ErrInvalidPercentage = errors.New("widget: percentage must be between 0 and 100")

	// ErrInvalidParentSize is returned when the parent dimensions are non-positive.
	ErrInvalidParentSize = errors.New("widget: parent dimensions must be positive")
)

// Percent represents a percentage value between 0 and 100.
type Percent float64

// Clamp returns the percentage clamped to the valid [0, 100] range.
func (p Percent) Clamp() Percent {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

// ToPixels converts a percentage to an absolute pixel value given a parent dimension.
// The parent dimension must be positive. The result is rounded to the nearest integer.
func (p Percent) ToPixels(parentDimension int) (int, error) {
	if parentDimension <= 0 {
		return 0, ErrInvalidParentSize
	}
	clamped := p.Clamp()
	pixels := float64(clamped) / 100.0 * float64(parentDimension)
	return int(pixels + 0.5), nil
}

// ValidatePercentage checks whether a value is a valid percentage (0–100).
func ValidatePercentage(v float64) error {
	if v < 0 || v > 100 {
		return ErrInvalidPercentage
	}
	return nil
}

// PercentToPixels converts a percentage value to pixels given the parent dimension.
// This is a convenience function that clamps the percentage to [0, 100].
func PercentToPixels(percent float64, parentDimension int) (int, error) {
	return Percent(percent).ToPixels(parentDimension)
}

// Size represents percentage-based dimensions for a widget.
type Size struct {
	Width  Percent // Width as percentage of parent (0–100).
	Height Percent // Height as percentage of parent (0–100).
}

// Resolve converts percentage-based dimensions to absolute pixel values.
func (s Size) Resolve(parentWidth, parentHeight int) (width, height int, err error) {
	width, err = s.Width.ToPixels(parentWidth)
	if err != nil {
		return 0, 0, err
	}
	height, err = s.Height.ToPixels(parentHeight)
	if err != nil {
		return 0, 0, err
	}
	return width, height, nil
}
