package dpi

import (
	"errors"
	"testing"
)

// mockConnection implements Connection for testing.
type mockConnection struct {
	widthPx  uint32
	heightPx uint32
	widthMm  uint32
	heightMm uint32
	err      error
}

func (m *mockConnection) GetScreenDimensions() (widthPx, heightPx, widthMm, heightMm uint32, err error) {
	return m.widthPx, m.heightPx, m.widthMm, m.heightMm, m.err
}

func TestDetectDPI_FromScreenDimensions(t *testing.T) {
	tests := []struct {
		name     string
		widthPx  uint32
		heightPx uint32
		widthMm  uint32
		heightMm uint32
		expected int32
	}{
		{
			name:     "Standard 96 DPI (1920x1080, 508x285mm)",
			widthPx:  1920,
			heightPx: 1080,
			widthMm:  508,
			heightMm: 285,
			expected: 96,
		},
		{
			name:     "HiDPI 192 DPI (3840x2160, 508x285mm)",
			widthPx:  3840,
			heightPx: 2160,
			widthMm:  508,
			heightMm: 285,
			expected: 192,
		},
		{
			name:     "MacBook Retina ~220 DPI (2560x1600, 286x179mm)",
			widthPx:  2560,
			heightPx: 1600,
			widthMm:  286,
			heightMm: 179,
			expected: 227,
		},
		{
			name:     "1.5x scale ~144 DPI (2880x1620, 508x285mm)",
			widthPx:  2880,
			heightPx: 1620,
			widthMm:  508,
			heightMm: 285,
			expected: 144,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn := &mockConnection{
				widthPx:  tc.widthPx,
				heightPx: tc.heightPx,
				widthMm:  tc.widthMm,
				heightMm: tc.heightMm,
			}
			detector := New(conn)
			dpi, err := detector.DetectDPI()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Allow ±1 DPI tolerance due to rounding
			if dpi < tc.expected-1 || dpi > tc.expected+1 {
				t.Errorf("expected DPI %d (±1), got %d", tc.expected, dpi)
			}
		})
	}
}

func TestDetectDPI_FallbackTo96(t *testing.T) {
	tests := []struct {
		name     string
		conn     *mockConnection
		expected int32
	}{
		{
			name: "Error from GetScreenDimensions",
			conn: &mockConnection{
				err: errors.New("connection error"),
			},
			expected: 96,
		},
		{
			name: "Zero width_mm",
			conn: &mockConnection{
				widthPx:  1920,
				heightPx: 1080,
				widthMm:  0,
				heightMm: 285,
			},
			expected: 96,
		},
		{
			name: "Zero height_mm",
			conn: &mockConnection{
				widthPx:  1920,
				heightPx: 1080,
				widthMm:  508,
				heightMm: 0,
			},
			expected: 96,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			detector := New(tc.conn)
			dpi, err := detector.DetectDPI()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dpi != tc.expected {
				t.Errorf("expected fallback DPI %d, got %d", tc.expected, dpi)
			}
		})
	}
}

func TestNew(t *testing.T) {
	conn := &mockConnection{}
	detector := New(conn)
	if detector == nil {
		t.Fatal("expected non-nil detector")
	}
	if detector.conn != conn {
		t.Error("expected connection to be stored")
	}
}
