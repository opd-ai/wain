package render_test

import (
	"testing"

	render "github.com/opd-ai/wain/internal/render"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int32
		expected int32
	}{
		{"positive", 2, 3, 5},
		{"zero", 0, 0, 0},
		{"negative", -4, 4, 0},
		{"both negative", -3, -7, -10},
		{"identity", 42, 0, 42},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := render.Add(tc.a, tc.b)
			if got != tc.expected {
				t.Errorf("Add(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.expected)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	v := render.Version()
	if v == "" {
		t.Fatal("Version() returned empty string")
	}
	t.Logf("render library version: %s", v)
}
