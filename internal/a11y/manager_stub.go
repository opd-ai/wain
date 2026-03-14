//go:build !atspi
// +build !atspi

package a11y

import "errors"

// Manager is a no-op accessibility manager used when the atspi build tag is absent.
// To enable real AT-SPI2 D-Bus support, rebuild with -tags=atspi.
type Manager struct{}

// NewManager returns an error indicating that D-Bus support is not compiled in.
// Callers should treat this gracefully (disable accessibility).
func NewManager(_ string) (*Manager, error) {
	return nil, errors.New("a11y: AT-SPI2 support not compiled; rebuild with -tags=atspi")
}

// Close is a no-op in the stub build.
func (m *Manager) Close() {}

// RegisterPanel is a no-op stub.
func (m *Manager) RegisterPanel(_ string, _ uint64) uint64 { return 0 }

// RegisterButton is a no-op stub.
func (m *Manager) RegisterButton(_ string, _ uint64, _ func() bool) uint64 { return 0 }

// RegisterLabel is a no-op stub.
func (m *Manager) RegisterLabel(_ string, _ uint64) uint64 { return 0 }

// RegisterEntry is a no-op stub.
func (m *Manager) RegisterEntry(_ string, _ uint64) uint64 { return 0 }

// RegisterScrollPane is a no-op stub.
func (m *Manager) RegisterScrollPane(_ string, _ uint64) uint64 { return 0 }

// SetBounds is a no-op stub.
func (m *Manager) SetBounds(_ uint64, _, _, _, _ int32) {}

// SetFocused is a no-op stub.
func (m *Manager) SetFocused(_ uint64, _ bool) {}

// SetText is a no-op stub.
func (m *Manager) SetText(_ uint64, _ string) {}

// SetName is a no-op stub.
func (m *Manager) SetName(_ uint64, _ string) {}
