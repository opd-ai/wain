# API Stability Policy

## Stability Guarantee

Starting with **v1.0.0**, `github.com/opd-ai/wain` follows [Semantic Versioning](https://semver.org/):

| Version bump | When |
|---|---|
| **Patch** (`v1.x.Y`) | Bug fixes only; no API changes |
| **Minor** (`v1.X.0`) | Backward-compatible additions; existing code still compiles |
| **Major** (`v2.0.0`) | Breaking changes after full deprecation cycle |

The stability guarantee covers all exported identifiers in the root `wain` package. Internal
packages (`internal/...`) are exempt and may change at any time.

## Deprecation Policy

1. **Announce**: Add a `// Deprecated: use Foo instead.` GoDoc comment on the old identifier.
2. **Wait one minor release**: The deprecated identifier stays fully functional for at least one
   `v1.X.0` minor release after the announcement.
3. **Remove**: Remove the deprecated identifier in the next major release (`v2.0.0` or later).

Example deprecation comment:

```go
// Deprecated: Use NewButton instead. Will be removed in v2.
func OldButton(label string) *Button { return NewButton(label, Size{}) }
```

## Covered Identifiers (v1.0.0)

The following public identifiers are pinned by `compat_test.go` as compile-time assertions:

### Constructors
- `NewApp() *App`
- `NewAppWithConfig(AppConfig) *App`
- `NewButton(string, Size) *Button`
- `NewLabel(string, Size) *Label`
- `NewTextInput(string, Size) *TextInput`
- `NewPanel(Size) *Panel`
- `NewRow() *Row`
- `NewColumn() *Column`
- `NewStack() *Stack`
- `NewGrid(int) *Grid`
- `NewScrollView(Size) *ScrollView`
- `NewSpacer(Size) *Spacer`
- `EnableAccessibility(string) *AccessibilityManager`

### Methods
- `(*App).NewWindow(WindowConfig) (*Window, error)`
- `(*App).Run() error`
- `(*App).Quit()`
- `(*Window).RenderFrame() error`
- `(*Window).SetTitle(string) error`
- `(*Window).Close() error`
- `(*Window).Dispatcher() *EventDispatcher`

### Interfaces
- `Presenter` — `Present(context.Context) error` + `Close() error`
- `Widget`, `PublicWidget`, `Container`, `Canvas`, `AccessibleWidget`

## Migration Path Template

When a signature must change, provide a migration shim:

```go
// NewButton creates a Button widget.
// Deprecated: Use NewButtonWithOptions instead. Will be removed in v2.
func NewButton(label string, size Size) *Button {
    return NewButtonWithOptions(label, ButtonOptions{Size: size})
}
```

Document the migration in `CHANGELOG.md` and the release notes.

## Build Tag: `atspi`

AT-SPI2 screen-reader support requires the `atspi` build tag:

```bash
go build -tags=atspi ./...
```

Without this tag, `EnableAccessibility` returns `nil` and all accessibility
operations are no-ops. This is intentional — the D-Bus dependency is only
compiled in when explicitly requested.
