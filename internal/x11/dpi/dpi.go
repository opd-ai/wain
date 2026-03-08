// Package dpi implements DPI detection for X11 displays.
//
// This package provides functionality to query the X server for DPI settings
// from various sources (XRDB, screen dimensions, etc.) to support HiDPI displays.
package dpi

import (
	"fmt"
)

// Detector provides DPI detection for X11 displays.
type Detector struct {
	conn Connection
}

// Connection interface for X11 protocol operations.
type Connection interface {
	GetScreenDimensions() (widthPx, heightPx, widthMm, heightMm uint32, err error)
}

// New creates a new DPI detector.
func New(conn Connection) *Detector {
	return &Detector{conn: conn}
}

// DetectDPI attempts to detect the display DPI.
// It uses multiple methods in order of preference:
// 1. Calculate from screen physical dimensions
// 2. Default to 96 DPI if calculation fails
func (d *Detector) DetectDPI() (int32, error) {
	// Try to get DPI from screen dimensions
	dpi, err := d.dpiFromScreenDimensions()
	if err == nil && dpi > 0 {
		return dpi, nil
	}

	// Default to 96 DPI (standard desktop DPI)
	return 96, nil
}

// dpiFromScreenDimensions calculates DPI from screen physical size and pixel dimensions.
func (d *Detector) dpiFromScreenDimensions() (int32, error) {
	widthPx, heightPx, widthMm, heightMm, err := d.conn.GetScreenDimensions()
	if err != nil {
		return 0, fmt.Errorf("failed to get screen dimensions: %w", err)
	}

	if widthMm == 0 || heightMm == 0 {
		return 0, fmt.Errorf("invalid physical dimensions")
	}

	// Calculate DPI from width (horizontal DPI)
	// DPI = pixels / inches
	// inches = mm / 25.4
	widthInches := float64(widthMm) / 25.4
	dpiX := float64(widthPx) / widthInches

	// Calculate DPI from height (vertical DPI)
	heightInches := float64(heightMm) / 25.4
	dpiY := float64(heightPx) / heightInches

	// Use average of horizontal and vertical DPI
	avgDPI := (dpiX + dpiY) / 2.0

	return int32(avgDPI + 0.5), nil // Round to nearest integer
}
