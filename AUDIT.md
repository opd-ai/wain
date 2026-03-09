# Technical Debt Tracking

This file tracks TODO items in the codebase. Each TODO comment references an item here using the format `// TODO(TD-N): Description` where N is the item number below.

## Active Items

_(No active items)_

## Completed Items

### TD-5: Implement text rendering in bufferCanvas ✅
**Completed:** 2026-03-09  
**File:** `concretewidgets.go:108` (removed TODO)  
**Solution:** Implemented `DrawText()` method in bufferCanvas to enable text rendering for adapted widgets:
1. Access `font.atlas` directly from Font parameter (private field, same package)
2. Call `text.DrawText()` with proper parameters: buffer, text, offset coordinates, font size, color, atlas
3. Added import: `textpkg "github.com/opd-ai/wain/internal/raster/text"`
4. Added nil guard for font and font.atlas
**Files Modified:**
- `concretewidgets.go`: Implemented DrawText method (8 LOC changed)
**Tests:** Code compiles successfully; go build passes
**Impact:** PublicWidget children in ScrollView can now render text. This completes the ScrollView adapter functionality started in TD-2.
**Complexity:** DrawText complexity cc=2 (one if guard), well under threshold of 10

### TD-3: Theme system integration for Panel widget ✅
**Completed:** 2026-03-09  
**File:** `layout.go:421` (syncStyleToInternal), `layout.go:265` (SetTheme)  
**Solution:** Implemented theme propagation system for Panel widgets:
1. Added `theme *Theme` field to Panel struct to cache the current theme
2. Added `SetTheme(theme Theme)` method that sets the theme and recursively propagates to all children
3. Updated `syncStyleToInternal()` to use cached theme if set, otherwise fall back to DefaultDark()
4. Added `extractPanel()` helper to extract the underlying Panel from composite widget types (Row, Column, Stack, Grid)
5. Added `Theme()` method to App as convenience alias for GetTheme()
**Files Modified:**
- `layout.go`: Added theme field, SetTheme method, extractPanel helper, updated syncStyleToInternal (51 LOC added)
- `app.go`: Added Theme() convenience method (6 LOC added)
**Tests:** All tests passing (59 packages), zero regressions
**Impact:** Panels now support theme propagation. Applications can call `panel.SetTheme(app.Theme())` to apply the app's theme to a widget tree. Panels without an explicit theme set continue to use DefaultDark() for backward compatibility.
**Complexity:** Added extractPanel (cc=3.1), Theme (cc=1.3); syncStyleToInternal increased from cc=2 to cc=3 (well under threshold of 10)



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
**Total Active Items:** 0  
**Total Completed Items:** 5
