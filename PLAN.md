# Implementation Plan: Phase 9.3 — Unified Event System

## Project Context
- **What it does**: A statically-compiled Go UI toolkit with GPU rendering via Rust, targeting both Wayland and X11 display servers with auto-detection
- **Current milestone**: Phase 9.3 (Unified Event System) — first incomplete item in Phase 9 (Public API Infrastructure)
- **Estimated Scope**: **Medium** (11 functions above complexity threshold in event-related code)

## Metrics Summary
- **Complexity hotspots**: 48 functions above threshold (complexity > 9.0)
  - `app.go`: 6 functions (initWaylandWindow: 17.9, NewWindow: 13.5, connectWayland: 12.2)
  - `dispatcher.go`: 3 functions (dispatchKey: 11.9)
  - Protocol layers: X11/Wayland client packages contribute additional complexity
- **Duplication ratio**: 3.24% (782 duplicated lines across 45 clone pairs)
- **Doc coverage**: 89.9% overall (functions: 98.2%, types: 89.4%, methods: 87.8%)
- **Package coupling**: Notable concerns in `main` (10.0), `demo` (5.0), `wain` (4.5), `display` (4.0)

## Codebase Overview
| Metric | Value |
|--------|-------|
| Total LOC | 10,606 |
| Functions | 412 |
| Methods | 744 |
| Packages | 36 |
| Files | 150 |

## Implementation Steps

### Step 1: Define Public Event Types
- **Deliverable**: Create public event type definitions in `event.go` at module root
  - `PointerEvent` (move, button, scroll) with Position, Button, Delta fields
  - `KeyEvent` (press, release, repeat) with Key, Modifiers, Repeat fields
  - `WindowEvent` (resize, close, focus) with Type, Size fields
  - `CustomEvent` for application-injected events via channel
- **Dependencies**: None (foundational types)
- **Acceptance**: Event types compile and are documented in godoc; no new complexity hotspots
- **Validation**: `go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.file | endswith("event.go")) | select(.complexity.overall > 9.0)] | length'` → 0

### Step 2: Implement Event Translation Layer
- **Deliverable**: Create translation functions in `internal/integration/events.go`
  - `translateWaylandPointer(wl_pointer_event) → PointerEvent`
  - `translateWaylandKeyboard(wl_keyboard_event) → KeyEvent`
  - `translateX11Event(x11.Event) → Event`
  - Consolidate existing key translation from `dispatcher.go` (complexity 11.9)
- **Dependencies**: Step 1 (event types must exist)
- **Acceptance**: Reduce `dispatcher.go` dispatchKey complexity from 11.9 to ≤9.0
- **Validation**: `go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.name == "dispatchKey")] | .[0].complexity.overall'` ≤ 9.0

### Step 3: Refactor app.go Event Handling
- **Deliverable**: Extract event dispatch logic from `app.go` into dedicated methods
  - Split `initWaylandWindow` (complexity 17.9) into initialization + event setup
  - Extract callback registration from `connectWayland` (complexity 12.2)
  - Target: No function in app.go exceeds complexity 12.0
- **Dependencies**: Step 2 (translation layer available)
- **Acceptance**: `app.go` max function complexity ≤ 12.0 (from 17.9)
- **Validation**: `go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.file | endswith("app.go"))] | max_by(.complexity.overall) | .complexity.overall'` ≤ 12.0

### Step 4: Add Event Dispatch Infrastructure
- **Deliverable**: Implement event dispatch in `App` type
  - Event queue with buffering (internal channel-based)
  - Hit-testing from root widget (traverse widget tree)
  - Event consumption (mark handled to stop propagation)
  - Focus management (keyboard focus, tab order)
- **Dependencies**: Step 3 (clean event handling in app.go)
- **Acceptance**: Event dispatch testable independently; demo apps receive events via public API
- **Validation**: `go test ./... -run Event` passes; no new functions above complexity 9.0

### Step 5: Expose Public Event Callbacks on Window
- **Deliverable**: Add callback registration methods to `Window` type
  - `OnPointerMove(func(PointerEvent))`, `OnPointerButton(func(PointerEvent))`
  - `OnKeyPress(func(KeyEvent))`, `OnKeyRelease(func(KeyEvent))`
  - `OnResize(func(WindowEvent))`, `OnClose(func(WindowEvent))`, `OnFocus(func(WindowEvent))`
  - `OnCustom(func(CustomEvent))` for application-defined events
- **Dependencies**: Step 4 (dispatch infrastructure)
- **Acceptance**: All existing input demos work through public event API
- **Validation**: `go-stats-generator analyze . --skip-tests --format json | jq '.documentation.coverage.methods'` ≥ 90%

### Step 6: Create Event Demo Application
- **Deliverable**: Create `cmd/event-demo/main.go` demonstrating the unified event system
  - Show all event types being received and logged
  - Demonstrate event propagation and consumption
  - Test on both X11 and Wayland
- **Dependencies**: Step 5 (callbacks exposed)
- **Acceptance**: Demo runs on both display servers; logs all event types
- **Validation**: `go build ./cmd/event-demo && ldd ./event-demo` reports "not a dynamic executable"

### Step 7: Update Documentation
- **Deliverable**: Update `API.md` with public event API documentation
  - Document all event types with field descriptions
  - Provide code examples for common use cases
  - Document event propagation model
- **Dependencies**: Step 6 (API finalized)
- **Acceptance**: API.md includes event system section; doc coverage remains ≥ 89%
- **Validation**: `go-stats-generator analyze . --skip-tests --format json | jq '.documentation.coverage.overall'` ≥ 89.0

## Scope Assessment Rationale

| Metric | Current | Target | Items |
|--------|---------|--------|-------|
| Functions > complexity 9.0 (event-related) | 11 | ≤5 | app.go:6, dispatcher.go:3, demo:2 |
| Duplication ratio | 3.24% | <3% | Focus on cmd/ demo code consolidation |
| Doc coverage gap | 10.1% | <10% | Add event type documentation |

**Scope: Medium** — 11 functions require attention, focused on event handling in app.go and dispatcher.go. Duplication is within acceptable range but demos show patterns that could be consolidated. Documentation is strong overall but event types need explicit coverage.

## Validation Commands Summary

```bash
# Step 1: Verify no complex event types
go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.file | endswith("event.go")) | select(.complexity.overall > 9.0)] | length'

# Step 2: Check dispatchKey complexity reduction
go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.name == "dispatchKey")] | .[0].complexity.overall'

# Step 3: Check app.go max complexity
go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.file | endswith("app.go"))] | max_by(.complexity.overall) | .complexity.overall'

# Step 5: Check method documentation coverage
go-stats-generator analyze . --skip-tests --format json | jq '.documentation.coverage.methods'

# Step 7: Check overall documentation
go-stats-generator analyze . --skip-tests --format json | jq '.documentation.coverage.overall'

# Overall complexity hotspots (should decrease)
go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.complexity.overall > 9.0)] | length'
```

---

## Gaps Document (Out of Scope for Phase 9.3)

The following issues were identified by metrics but are outside the current milestone:

### High-Complexity Functions (Non-Event)
- `internal/raster/text/text.go:drawGlyph` (13.2) — rendering logic
- `internal/render/damage.go:Coalesce` (12.4) — damage tracking
- `internal/raster/composite/composite.go:Blit` (11.9) — compositing
- `internal/raster/core/rect.go:FillRect` (11.9) — primitives

### Package Coupling Issues
- `main` package coupling score: 10.0 (cmd/ binaries import many packages)
- `demo` package coupling score: 5.0 (expected for demo code)

### Duplication Candidates
- `cmd/decorations-demo`, `cmd/wayland-demo`, `internal/demo/x11setup.go` share 7-line initialization blocks
- `internal/render/commands.go` has internal duplication (7 lines, 2 locations)

These items should be addressed in:
- **Phase 9.4** (Render Integration Bridge) — damage tracking complexity
- **Phase 10.7** (Complete Application Example) — demo code consolidation
- **Post-Phase 10** — rasterizer optimization pass
