# Technical Debt Tracking

This file tracks TODO items in the codebase. Each TODO comment references an item here using the format `// TODO(TD-N): Description` where N is the item number below.

## Active Items

### TD-2: Implement proper child management for ScrollView
**File:** `concretewidgets.go:361`  
**Priority:** Medium  
**Description:** The public `ScrollView.Add(child PublicWidget)` method is a stub. Need to extend `internal/ui/widgets.ScrollContainer` to accept PublicWidget children (currently only works with internal widgets). Requires bridging PublicWidget interface to internal widget representation.  
**Impact:** ScrollView widget cannot hold child widgets, limiting its utility  
**Effort:** ~2-3 hours (add child container to ScrollContainer, implement layout pass for children)  
**Related:** None  

### TD-3: Theme system integration for Panel widget
**File:** `layout.go:388`  
**Priority:** Low  
**Description:** Panel widget currently hardcodes `DefaultDark()` theme instead of reading from `App.theme`. Once the App-level theme system is implemented, Panel should respect the global theme setting while allowing per-widget style overrides.  
**Impact:** Panels ignore app-wide theme, always use dark theme  
**Effort:** ~30 minutes (change to `app.theme.Panel()` once theme system exists)  
**Related:** Blocked by App.theme field implementation (not yet designed)  

## Completed Items

### TD-4: Full Wayland event reading and dispatch ✅
**Completed:** 2026-03-09  
**File:** `app.go:1471` (removed TODO), `internal/wayland/client/connection.go:222-298` (added), `internal/wayland/input/*.go` (event handlers added)  
**Solution:** Implemented full Wayland event reading and dispatch infrastructure:
1. Added `ReadMessage()` method to Connection for reading event messages from compositor socket (27 LOC)
2. Added `DispatchMessage()` method to Connection for routing events to object handlers (15 LOC)
3. Implemented `EventHandler` interface for objects that can process events
4. Updated `processWaylandEvents()` in app.go to read and dispatch events (21 LOC)
5. Added `HandleEvent()` implementations for all input objects:
   - Keyboard: Handles keymap, enter, leave, key, modifiers, repeat_info events (142 LOC)
   - Pointer: Handles enter, leave, motion, button, axis, frame, axis_source, axis_stop, axis_discrete events (173 LOC)
   - Touch: Handles down, up, motion, frame, cancel, shape, orientation events (134 LOC)
   - Seat: Handles capabilities, name events (36 LOC)
6. All event handlers properly decode wire protocol arguments and call existing handler methods
**Files Modified:**
- `internal/wayland/client/connection.go`: Added ReadMessage, DispatchMessage, EventHandler interface (75 LOC)
- `app.go`: Updated processWaylandEvents and added dispatchWaylandEvent (21 LOC)
- `internal/wayland/input/keyboard.go`: Added HandleEvent implementation (142 LOC)
- `internal/wayland/input/pointer.go`: Added HandleEvent implementation (173 LOC)
- `internal/wayland/input/touch.go`: Added HandleEvent implementation (134 LOC)
- `internal/wayland/input/seat.go`: Added HandleEvent implementation (36 LOC)
**Tests:** All 42 input tests passing (100% pass rate)
**Impact:** Wayland apps can now receive and process user input (keyboard, mouse, touch), window events, and compositor notifications. This enables full interactive Wayland applications.

### TD-1: Add placeholder support to TextInput widget ✅
**Completed:** 2026-03-09  
**File:** `concretewidgets.go:248` (fixed), `internal/ui/widgets/widgets.go:448` (added)  
**Solution:** Added `SetPlaceholder(string)` method to internal TextInput widget that updates the placeholder field. Updated public TextInput.SetPlaceholder() to call through to internal widget. Placeholder rendering was already implemented via getDisplayText() helper (grayed-out text when input is empty).  
**Files Modified:**
- `internal/ui/widgets/widgets.go`: Added SetPlaceholder method (3 LOC)
- `concretewidgets.go`: Updated SetPlaceholder to call internal method, removed TODO  
**Tests:** All tests passing (59 packages)

---

**Last Updated:** 2026-03-09  
**Total Active Items:** 2  
**Total Completed Items:** 2
