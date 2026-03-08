# Phase 9.3: UNIFIED EVENT SYSTEM - Complete Architecture Analysis

## Summary

This document provides a comprehensive analysis of the current event handling architecture in the wain project and identifies what needs to be built for Phase 9.3 (UNIFIED EVENT SYSTEM).

**Key Finding**: The foundation exists (platform-specific event parsers, widget handler stubs, window structure) but needs to be connected through a unified public event API and event dispatcher.

---

## 1. EXISTING INTERNAL EVENT HANDLING CODE

### 1.1 Wayland Input Layer (`/internal/wayland/input/`)

**Files**: input.go (212L), keyboard.go (113L), pointer.go (144L), keymap.go (184L), touch.go (77L)

#### Seat Management (input.go)
- `Seat` - Input device aggregator
- Methods: `GetPointer()`, `GetKeyboard()`, `GetTouch()`
- Capabilities tracking: Pointer, Keyboard, Touch

#### Keyboard (keyboard.go)
- **Structures**:
  - `Keyboard` - Wayland keyboard object
  - `KeyState` enum: Released(0), Pressed(1)
  - `ModifierState` struct: Shift, CapsLock, Ctrl, Alt, NumLock, Meta

- **Handler Stubs** (empty implementations):
  - `HandleKeymap(format, fd, size)` - Loads XKB format from fd
  - `HandleEnter(serial, surfaceID, keys)` - Keyboard focus enter
  - `HandleLeave(serial, surfaceID)` - Keyboard focus leave
  - `HandleKey(serial, time, key, state)` - Key press/release
  - `HandleModifiers(...)` - Modifier state updates (depressed, latched, locked)
  - `HandleRepeatInfo(rate, delay)` - Key repeat settings

- **Key Symbol Resolution**:
  - `ModifierState.decodeModifiers()` converts XKB bitmasks
  - Pattern: Depressed|Latched|Locked → individual modifier flags

#### Pointer/Mouse (pointer.go)
- **Structures**:
  - `ButtonState` enum: Released(0), Pressed(1)
  - `Axis` enum: VerticalScroll(0), HorizontalScroll(1)

- **Handler Stubs**:
  - `HandleEnter(serial, surfaceID, surfaceX, surfaceY)` - Pointer enters surface (fixed-point coords)
  - `HandleLeave(serial, surfaceID)` - Pointer leaves
  - `HandleMotion(time, surfaceX, surfaceY)` - Movement
  - `HandleButton(serial, time, button, state)` - Button press/release
  - `HandleAxis(time, axis, value)` - Scroll events
  - `HandleFrame()` - Groups related events
  - `HandleAxisSource()`, `HandleAxisStop()`, `HandleAxisDiscrete()` - Scroll metadata

**Key Point**: Coordinates in fixed-point format (multiply by 1/256)

#### Keyboard Mapping (keymap.go)
- `Keysym` type (uint32)
- Common keysyms: BackSpace, Tab, Return, Escape, arrows, etc.
- **Function**: `KeycodeToKeysym(keycode, modifiers) Keysym`
  - Linux evdev keycode (1-126) → Keysym
  - Shift-aware lookup for digits, letters
  - Hardcoded QWERTY layout (sufficient for MVP)

#### Touch (touch.go)
- `Touch` struct (minimal)
- Handler stubs:
  - `HandleDown(serial, time, surfaceID, id, x, y)` - Touch contact
  - `HandleUp(serial, time, id)` - Touch release
  - `HandleMotion(time, id, x, y)` - Movement
  - `HandleFrame()` - Groups touch events
  - `HandleCancel()` - Session cancelled
  - `HandleShape(id, major, minor)`, `HandleOrientation(id, orientation)` - Touch metadata

### 1.2 X11 Event Layer (`/internal/x11/events/events.go` - 347 lines)

**Status**: Fully implemented with parser functions (not stubs)

#### Event Type Constants
```
KeyPress, KeyRelease, ButtonPress, ButtonRelease, MotionNotify,
EnterNotify, LeaveNotify, FocusIn, FocusOut, Expose, ConfigureNotify, ...
```

#### Implemented Event Structures with Parsers

1. **KeyPressEvent** / **KeyReleaseEvent**
   - Fields: Type, Detail (keycode), Sequence, Time (ms), Root, Event, Child
   - Coordinates: RootX/Y, EventX/Y (int16)
   - State: uint16 (modifier mask)
   - SameScreen: bool
   - Parser: `ParseKeyPressEvent(header wire.EventHeader, data []byte)`

2. **ButtonPressEvent** / **ButtonReleaseEvent**
   - Detail: 1=left, 2=middle, 3=right, 4=scroll up, 5=scroll down
   - Same coordinate/state structure

3. **MotionNotifyEvent**
   - Detail: 0=normal, 1=hint
   - Coordinates, modifier state

4. **ExposeEvent**
   - Window, exposure region (X, Y, Width, Height), Count

5. **ConfigureNotifyEvent**
   - Window configuration: position (X, Y), size (Width, Height)
   - AboveSibling, BorderWidth, OverrideRedirect

#### Modifier Masks
```
ModifierShift, ModifierLock, ModifierControl, ModifierMod1 (Alt),
ModifierMod2 (NumLock), ModifierMod3, ModifierMod4 (Super), ModifierMod5,
ModifierButton1-5 (mouse buttons)
```

Helper: `HasModifier(state uint16, modifier ModifierMask) bool`

---

## 2. EXISTING WIDGET/UI EVENT HANDLING

### 2.1 Widget Base Interface (`/internal/ui/widgets/widgets.go`)

```go
type Widget interface {
    Bounds() (width, height int)
    HandlePointerEnter()
    HandlePointerLeave()
    HandlePointerDown(button uint32)
    HandlePointerUp(button uint32)
    Draw(buf *core.Buffer, x, y int) error
}
```

**Status**: Pointer-only; no keyboard handling

### 2.2 Implemented Widgets

#### Button
- State tracking: PointerStateNormal, PointerStateHover, PointerStatePressed
- Callback: `SetOnClick(func())`
- Methods: `HandlePointerEnter()`, `HandlePointerLeave()`, `HandlePointerDown()`, `HandlePointerUp()`

#### TextInput
- Method: `HandleKeyPress(key rune)` - Character input (NOT in Widget interface)
- Supports backspace, character limits

#### ScrollContainer
- Item-based scrolling

#### Theme System
- BackgroundNormal/Hover/Pressed/Disabled
- TextNormal/Hover/Pressed/Disabled
- BorderNormal/Hover/Pressed/Focus
- Shadow configuration, FontSize, Scale

### 2.3 Window Event Callbacks (`/app.go`, lines 237-242)

```go
type Window struct {
    // ... fields ...
    
    // Event handlers (LIFECYCLE ONLY)
    onResize      func(width, height int)
    onClose       func()
    onFocus       func(focused bool)
    onScaleChange func(scale float64)
}
```

**Status**: Only lifecycle callbacks; no input callbacks

---

## 3. PLATFORM COMPARISON TABLE

### Keyboard Events

| Feature | Wayland | X11 |
|---------|---------|-----|
| Event Handler | `Keyboard.HandleKey()` (stub) | `ParseKeyPressEvent()` (implemented) |
| Keycode Format | Linux evdev (uint32) | X11 keycode (uint8) |
| Keysym Source | `Keymap.KeycodeToKeysym()` (hardcoded QWERTY) | Requires server query |
| Modifiers | Depressed/latched/locked bitmasks | Single state mask |
| Focus Events | Enter/leave separate | `FocusIn`/`FocusOut` events |
| Repeat | `HandleRepeatInfo(rate, delay)` | Not built-in |

### Pointer Events

| Feature | Wayland | X11 |
|---------|---------|-----|
| Handlers | `HandleMotion()`, `HandleButton()`, `HandleAxis()` (stubs) | `ParseMotionNotifyEvent()`, `ParseButtonPressEvent()` (implemented) |
| Coordinates | Fixed-point (×1/256) | int16 |
| Button Codes | 272=left, 273=right, 274=middle | 1=left, 2=middle, 3=right |
| Scroll | Separate axis event | Buttons 4 (up), 5 (down) |
| Focus | Enter/leave events | Implicit via window focus |
| Metadata | Source, stop, discrete events | Single value |

---

## 4. EXISTING INPUT DEMOS

### widget-demo (`/cmd/widget-demo/main.go` - 358 lines)

**Purpose**: Interactive Phase 1 widgets demo

**Input Handling Patterns**:
```go
// Mouse interaction (manual hit-testing)
app.clickButton.HandlePointerEnter()
app.clickButton.HandlePointerLeave()
app.clickButton.HandlePointerDown(button)
app.clickButton.HandlePointerUp(button)

// Keyboard (string-based key names)
if key == "BackSpace" && len(app.inputText) > 0 { ... }
if key == "Escape" { app.running = false }

// Hit-testing
if pointInRect(x, y, 50, 60, 150, 40) {
    app.clickButton.HandlePointerDown(button)
}
```

**Status**: 
- Wayland event loop: stub ("⚠ Wayland event loop not yet implemented")
- X11 event loop: simulated (manual calls to handlers)

### wain-demo (`/cmd/wain-demo/main.go` - 52 lines)

**Purpose**: Validates Phase 9.1 (APPLICATION LIFECYCLE)

**Status**: Working - demonstrates `app.Run()` lifecycle with signal handling

---

## 5. GAPS & WHAT NEEDS TO BE BUILT

### 5.1 Missing Components

| Component | Current State | Needed For |
|-----------|---------------|-----------|
| **Public Event Types** | None | User-facing API |
| **Event Dispatcher** | None | Central routing |
| **Widget Tree Structure** | None | Hit-testing |
| **Focus Manager** | None | Tab order, focus chain |
| **Platform Translators** | None | Platform event → public event |
| **Window Input Callbacks** | None | `win.OnKeyPress()` API |
| **Event Consumption** | None | Propagation control |

### 5.2 Required Public API (from ROADMAP)

1. **KeyEvent**
   - Key: string ("Tab", "Escape", "a")
   - Action: Press, Release, Repeat
   - Modifiers: {Shift, Control, Alt, Super, ...}
   - Time, Consumed

2. **PointerEvent**
   - X, Y: int (pixel coordinates)
   - Button: 1=left, 2=middle, 3=right
   - ScrollDelta: float64
   - Action: Move, Press, Release, Scroll
   - Modifiers, Time, Consumed

3. **TouchEvent**
   - ID, X, Y, Action, Time, Consumed

4. **WindowEvent**
   - Type: Resize, Close, Focus, ScaleChange
   - Data varies by type

5. **CustomEvent**
   - App-defined events via channel

### 5.3 Required Event Dispatch Flow

1. Platform event → Parse (already done for X11, stubs for Wayland)
2. Parse → Public event type (MISSING)
3. Public → Hit-test widget tree (MISSING)
4. Hit-test → Widget callback (Partially done)
5. Widget → Window → App (MISSING)
6. Consumed flag stops propagation (MISSING)

---

## 6. KEYBOARD FOCUS MANAGEMENT

**Current State**:
```go
type Window struct {
    focused bool    // Only this
}
```

**Needed**:
- Focus chain (list of focusable widgets in tab order)
- Current focused widget tracking
- Tab/Shift+Tab navigation
- Explicit focus API: `win.SetFocus(widget)`
- Focus gain/loss events

---

## 7. IMPLEMENTATION ROADMAP (PHASES)

### 7a: Public Event Types (1-2 days)
- Create `/event.go` with public event types
- Tests in `/event_test.go`

### 7b: Platform Translators (2-3 days)
- `/internal/wayland/input/translate.go`
- `/internal/x11/events/translate.go`
- Convert platform events to public types
- Test with existing test fixtures

### 7c: Event Dispatcher (2-3 days)
- `/internal/events/dispatcher.go` - Central dispatcher
- `/internal/events/focus_manager.go` - Focus chain
- `/internal/events/widget_tree.go` - Widget traversal

### 7d: Widget & Window Extensions (1-2 days)
- Extend Widget interface with `Handle(event Event) bool`
- Add input callbacks to Window struct
- Implement callback registration

### 7e: Platform Integration (2-3 days)
- Wire Wayland handlers to translator
- Wire X11 event loop to translator
- Initialize dispatcher in app.Run()

### 7f: Validation (1-2 days)
- Update widget-demo to use new API
- Verify all input scenarios

**Total**: 9-15 days

---

## 8. KEY FILES REFERENCE

### Creating (New Files)
```
/event.go                                  - Public event types
/internal/events/dispatcher.go            - Event dispatcher
/internal/events/focus_manager.go         - Focus chain management
/internal/wayland/input/translate.go      - Wayland translator
/internal/x11/events/translate.go         - X11 translator
```

### Modifying
```
/app.go                                   - Add input callbacks to Window
/internal/ui/widgets/widgets.go           - Add Handle() method
/internal/wayland/input/keyboard.go       - Call translator in handlers
/internal/wayland/input/pointer.go        - Call translator in handlers
/internal/x11/client/                     - Wire event loop
```

### Testing
```
/internal/wayland/input/input_test.go     - Existing test fixtures
/internal/x11/events/events_test.go       - Existing test fixtures
/internal/ui/widgets/widgets_test.go      - Widget event tests
```

---

## 9. VALIDATION CHECKLIST

### Completion Criteria (from ROADMAP 9.3)
- [ ] Public event types for Pointer, Key, Touch, Window, Custom
- [ ] Event dispatch from platform → public → widget tree
- [ ] Event consumption (propagation control)
- [ ] Keyboard focus management (tab order, explicit focus)
- [ ] API example: `win.OnKeyPress(func(e wain.KeyEvent) { ... })`
- [ ] All existing input demos work through new public API

### Input Demos Must Pass
- [ ] Pointer hover - button color changes
- [ ] Click - counter increments
- [ ] Keyboard input - text appears in field
- [ ] Scroll - container scrolls
- [ ] Tab navigation - focus moves between widgets
- [ ] Escape - app quits

---

## 10. ARCHITECTURE PATTERNS TO REUSE

### Pattern 1: Widget Callback (Current)
```go
// From widgets.go
type Button struct {
    onClick func()
}

func (b *Button) HandlePointerUp(button uint32) {
    if b.onClick != nil {
        b.onClick()
    }
}
```

**Adapt for Phase 9.3**: Callback should return `bool` (consumed)

### Pattern 2: Modifier Masks (X11)
```go
// From events.go
const ModifierShift ModifierMask = 1 << 0

func HasModifier(state uint16, modifier ModifierMask) bool {
    return state&uint16(modifier) != 0
}
```

**Reuse in public API**: ModifierSet struct with Shift, Control, Alt, Super fields

### Pattern 3: Keymap Translation (Wayland)
```go
// From keymap.go
func (km *Keymap) KeycodeToKeysym(keycode uint32, modifiers ModifierState) Keysym {
    shifted := modifiers.Shift
    switch keycode {
    case 14: return KeysymBackSpace
    }
}
```

**Extend**: Create translator outputting human-readable strings ("Tab", "Escape")

### Pattern 4: Hit-Testing (Demo)
```go
// From widget-demo/main.go
func pointInRect(px, py, rx, ry, rw, rh int) bool {
    return px >= rx && px < rx+rw && py >= ry && py < ry+rh
}
```

**Centralize**: In Dispatcher.hitTest(x, y) Widget

---

## 11. GOTCHAS & SOLUTIONS

| Issue | Cause | Solution |
|-------|-------|----------|
| Fixed-point coordinates (Wayland) | Compositor uses 1/256 precision | Multiply by 1/256 in translator |
| X11 button scroll codes | Buttons 4=scroll-up, 5=scroll-down | Map to PointerScroll action |
| Tab order vs. visual order | May differ | Explicit FocusManager.AddWidget(w, order) |
| Modifier state race | State updates asynchronous | Always use current state in handler |
| Focus loss during Alt-Tab | Expected behavior | Handled by window focus callback |
| Key repeat timing | Need delay + rate from compositor | Store in Window, implement timer |

---

## NEXT STEPS

1. **Read this document** to understand current architecture
2. **Review the design document** (EVENT_SYSTEM_DESIGN.md) for flow diagrams
3. **Check quick reference** (QUICK_REFERENCE.md) for implementation details
4. **Start with Phase 7a**: Create public event types in `/event.go`
5. **Build platform translators** in Phase 7b
6. **Implement dispatcher** in Phase 7c
7. **Test with existing widget-demo**

---

**Document Generated**: March 2024
**Scope**: Complete architecture analysis for Phase 9.3 UNIFIED EVENT SYSTEM
**Status**: Ready for implementation

