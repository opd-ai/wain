# Technical Debt Tracking

This file tracks TODO items in the codebase. Each TODO comment references an item here using the format `// TODO(TD-N): Description` where N is the item number below.

## Active Items

### TD-3: Theme system integration for Panel widget
**File:** `layout.go:388`  
**Priority:** Low  
**Description:** Panel widget currently hardcodes `DefaultDark()` theme instead of reading from `App.theme`. Once the App-level theme system is implemented, Panel should respect the global theme setting while allowing per-widget style overrides.  
**Impact:** Panels ignore app-wide theme, always use dark theme  
**Effort:** ~30 minutes (change to `app.theme.Panel()` once theme system exists)  
**Related:** Blocked by App.theme field implementation (not yet designed)  

## Completed Items

### TD-2: Implement proper child management for ScrollView ✅
**Completed:** 2026-03-09  
**File:** `concretewidgets.go:361-438` (adapter implementation), `concretewidgets.go:431-438` (ScrollView.Add)  
**Solution:** Created a widgetAdapter type that bridges PublicWidget to the internal Widget interface:
1. Added widgetAdapter struct that wraps any PublicWidget instance (15 LOC)
2. Implemented all internal Widget interface methods (Bounds, Handle*, Draw) (30 LOC)
3. Created bufferCanvas that implements Canvas by drawing to primitives.Buffer with offset (75 LOC)
4. Updated ScrollView.Add() to wrap PublicWidget children in adapter before adding to ScrollContainer (8 LOC)
5. The adapter translates pointer events from internal format to public PointerEvent format
6. The bufferCanvas translates Canvas drawing commands to Buffer drawing with coordinate offset
**Files Modified:**
- `concretewidgets.go`: Added widgetAdapter, bufferCanvas types and updated ScrollView.Add (128 LOC added)
**Tests:** Code compiles and builds successfully; go vet passes with no new warnings
**Impact:** ScrollView can now accept and render PublicWidget children. Applications can add buttons, labels, and other public widgets to scroll containers as intended in the API design.
**Limitations:** 
- Text rendering in adapted widgets requires atlas access (not yet implemented)
- DrawImage, gradients, and box shadows not yet supported in bufferCanvas (can be added as needed)

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
**Total Active Items:** 1  
**Total Completed Items:** 3
