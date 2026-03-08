# Accessibility in Wain

## Current Status

**Wain does not currently implement AT-SPI2 accessibility support.**

This document explains the technical constraints, future possibilities, and workarounds for assistive technology integration.

---

## Background: AT-SPI2 on Linux

AT-SPI2 (Assistive Technology Service Provider Interface 2) is the standard Linux accessibility protocol that enables screen readers (like Orca), magnifiers, and other assistive tools to interact with applications. It works through:

- **D-Bus IPC**: Applications expose their widget tree via D-Bus interfaces
- **Hierarchical Object Model**: Each UI element becomes a D-Bus object with properties (name, role, state) and methods (click, focus, etc.)
- **Event Notifications**: Applications emit signals when the UI changes (focus, text edits, window creation)

---

## Why AT-SPI2 Is Not Currently Implemented

### 1. Runtime D-Bus Dependency

AT-SPI2 requires:
- A running `dbus-daemon` system service (present on all desktop Linux distributions)
- Access to session bus via Unix socket (`$DBUS_SESSION_BUS_ADDRESS`)
- Registration with `org.a11y.atspi.Registry`

**Implication**: While Wain can be built as a fully static binary, accessibility requires a running D-Bus session, making it **environment-dependent**.

### 2. Implementation Complexity

A functional AT-SPI2 implementation requires:

| Component | Scope |
|-----------|-------|
| D-Bus service export | Expose widget hierarchy over D-Bus |
| `Accessible` interface | Name, description, role, parent/child navigation |
| `Component` interface | Position, size, visibility, hit testing |
| `Action` interface | Click, press, focus actions |
| `Text` interface | Text content, caret position, selections |
| `Value` interface | Slider/spinner values and ranges |
| Event emission | Focus changes, text edits, tree updates |

**Estimated effort**: 2-3 weeks for MVP (buttons, text, basic navigation), 1-2 months for comprehensive support.

### 3. Platform Limitations

AT-SPI2 is **Linux-only**. Other platforms require separate implementations:
- **Windows**: UI Automation (UIA)
- **macOS**: Accessibility API
- **Wayland/X11**: Both protocols are silent on accessibility; AT-SPI2 is the de facto standard

---

## Workarounds for Accessibility

### Option 1: High-Contrast Themes
Wain supports custom themes (see `internal/ui/widgets/theme.go`). Users can configure high-contrast color schemes for visual accessibility:

```go
theme := &widgets.Theme{
    Background:  color.RGBA{0, 0, 0, 255},      // Black
    Foreground:  color.RGBA{255, 255, 0, 255},  // Yellow
    Accent:      color.RGBA{255, 255, 255, 255}, // White
    Scale:       2.0, // HiDPI scaling for larger text
}
```

### Option 2: Keyboard Navigation
All Wain widgets support keyboard navigation:
- Tab/Shift-Tab: focus traversal
- Enter/Space: activate buttons
- Arrow keys: navigate lists/scrollable containers
- Text fields: standard editing shortcuts

Applications should ensure all functionality is accessible via keyboard.

### Option 3: Export Widget Tree to External Tool
Applications can introspect their own widget tree and export it to a file or socket:

```go
// Example: Export widget tree as JSON for external accessibility tools
func ExportWidgetTree(root *widgets.Container) []byte {
    tree := buildAccessibilityTree(root)
    json, _ := json.MarshalIndent(tree, "", "  ")
    return json
}
```

This allows custom assistive tools to parse the UI structure without AT-SPI2.

---

## Future Implementation Path

If AT-SPI2 support is prioritized, the implementation would involve:

### Step 1: D-Bus Integration (Week 1)
- Add `github.com/godbus/dbus/v5` dependency
- Implement D-Bus session bus connection at startup
- Register application with `org.a11y.atspi.Registry`

### Step 2: Core Accessible Interface (Week 2)
- Expose root widget container at `/org/a11y/atspi/accessible/root`
- Implement properties: Name, Description, Role, Parent, ChildCount
- Implement methods: `GetChildAtIndex`, `GetChildren`
- Add object path generation for all widgets

### Step 3: Action & Component Interfaces (Week 3)
- `Action` interface: expose clickable widgets
- `Component` interface: widget bounds and visibility
- Event emission: focus changes, button clicks

### Step 4: Text Interface (Week 4-5)
- Expose text content from `TextInput` widgets
- Caret position and text selections
- Text change events

### Step 5: Testing (Week 6)
- Validate with Orca screen reader
- Test with Accerciser accessibility inspector
- Automated tests using AT-SPI2 client library

**Total estimated effort**: 6 weeks for production-ready implementation

---

## Design Constraints for Future Work

If AT-SPI2 is added, these constraints must be respected:

1. **Lazy Object Creation**: Don't expose invisible widgets to D-Bus (reduces overhead)
2. **Caching**: Cache D-Bus object paths to avoid repeated allocations
3. **Batch Events**: Coalesce rapid widget tree changes into single event bursts
4. **Thread Safety**: D-Bus calls may come from any thread; protect widget tree access
5. **Graceful Degradation**: If D-Bus registration fails, application should continue without accessibility (log warning)

---

## References

- **AT-SPI2 Specification**: [Freedesktop AT-SPI Wiki](https://www.freedesktop.org/wiki/Accessibility/AT-SPI2/)
- **D-Bus Go Binding**: [godbus/dbus](https://github.com/godbus/dbus)
- **Orca Screen Reader**: [GNOME Orca](https://help.gnome.org/users/orca/stable/)
- **Accerciser Accessibility Inspector**: [Accerciser](https://help.gnome.org/users/accerciser/stable/)

---

## Commitment to Accessibility

While full AT-SPI2 support is not currently implemented, we recognize the importance of accessible software. Contributions to implement AT-SPI2 are welcome. See `CONTRIBUTING.md` for guidelines.

For questions or proposals, open an issue on the project tracker.
