package wain_test

// TestAPICompat is a compile-time assertion file. Each variable below is a
// function-value assignment that will fail to compile if the corresponding
// exported signature changes. No runtime assertions are needed — a build failure
// is the signal.
//
// Add a new entry here whenever a new public function or method is added to the
// wain package. Remove entries only when a function is intentionally removed
// after the deprecation policy in STABILITY.md has been followed.
//
// Run with:
//
//	go test -run TestAPICompat ./...
import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/opd-ai/wain"
)

func TestAPICompat(t *testing.T) {
	t.Log("API compatibility assertions: all signatures verified at compile time")
}

// Compile-time signature pins — these assignments will fail to compile if the
// function signatures change. The blank identifier discards the value.

// App constructor and lifecycle.
var (
	_ func() *wain.App               = wain.NewApp
	_ func(wain.AppConfig) *wain.App = wain.NewAppWithConfig
)

// Window constructor.
var _ func(*wain.App, wain.WindowConfig) (*wain.Window, error) = func(a *wain.App, cfg wain.WindowConfig) (*wain.Window, error) {
	return a.NewWindow(cfg)
}

// Widget constructors.
var (
	_ func(string, wain.Size) *wain.Button    = wain.NewButton
	_ func(string, wain.Size) *wain.Label     = wain.NewLabel
	_ func(string, wain.Size) *wain.TextInput = wain.NewTextInput
	_ func(wain.Size) *wain.Panel             = wain.NewPanel
	_ func() *wain.Row                        = wain.NewRow
	_ func() *wain.Column                     = wain.NewColumn
	_ func() *wain.Stack                      = wain.NewStack
	_ func(int) *wain.Grid                    = wain.NewGrid
	_ func(wain.Size) *wain.ScrollView        = wain.NewScrollView
	_ func(wain.Size) *wain.Spacer            = wain.NewSpacer
)

// Accessibility.
var _ func(string) *wain.AccessibilityManager = wain.EnableAccessibility

// Window methods.
var (
	_ func(*wain.Window) error                 = (*wain.Window).RenderFrame
	_ func(*wain.Window, string) error         = (*wain.Window).SetTitle
	_ func(*wain.Window) error                 = (*wain.Window).Close
	_ func(*wain.Window) *wain.EventDispatcher = (*wain.Window).Dispatcher
)

// App methods.
var (
	_ func(*wain.App) error                                                                = (*wain.App).Run
	_ func(*wain.App)                                                                      = (*wain.App).Quit
	_ func(*wain.App, io.Reader, string) (*wain.Image, error)                              = (*wain.App).LoadImageFromReader
	_ func(*wain.App) []*wain.Window                                                       = (*wain.App).Windows
	_ func(*wain.App, float64, float64, time.Duration, wain.EasingFunc, func(float64)) *wain.Animation = (*wain.App).Animate
)

// Window DnD methods.
var (
	_ func(*wain.Window, []string, wain.DragDropHandler)                    = (*wain.Window).SetDropTarget
	_ func(*wain.Window, []string, wain.DragDataProvider, *wain.Image)      = (*wain.Window).StartDrag
)

// Presenter interface — any implementation must satisfy Present + Close.
var _ wain.Presenter = (*presenterCheck)(nil)

type presenterCheck struct{}

func (presenterCheck) Present(_ context.Context) error { return nil }
func (presenterCheck) Close() error                    { return nil }
