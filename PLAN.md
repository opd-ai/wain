# Implementation Plan: Code Quality & Completeness Milestone

## Project Context
- **What it does**: Wain is a statically-compiled Go UI toolkit with GPU-accelerated graphics on Linux via a Rust rendering library, supporting Wayland/X11 from scratch with zero runtime dependencies.
- **Current milestone**: All 10 phases complete (0-10); remaining work is code quality hardening and gap closure
- **Estimated Scope**: Small (6 functions above cc=9, 4.6% duplication, 91% doc coverage)

## Metrics Summary
- **Complexity hotspots**: 6 functions above threshold (cc>9)
  - `HandleEvent` in pointer.go (cc=31), keyboard.go (cc=28), touch.go (cc=25)
  - `main` in gen-atlas (cc=20)
  - `applyToTheme` (cc=10), `decodeVisuals` (cc=10)
- **Duplication ratio**: 4.6% (75 clone pairs, 1342 duplicated lines)
- **Doc coverage**: 91.0% overall (100% packages, 98.6% functions, 90.2% types, 89.0% methods)
- **Large functions**: 18 functions >50 lines (3 are Wayland input handlers)
- **Package coupling**: Notable packages by LOC: wain (4405), main (4292), backend (1546), wire (1523)

## Known Gaps (from ROADMAP)
1. **Wayland Event Path** (9.3): X11 event path complete; Wayland event path marked TODO
2. **TODO Comments**: 4 tracked TODOs in source code:
   - `app.go:1492` - Dispatch to window-specific handlers for input events
   - `concretewidgets.go:361` - Proper child management for ScrollView
   - `layout.go:388` - Get theme from App.theme when available
   - `screenshot_test.go:474` - GPU backend rendering (Phase 5 dependency)

## Implementation Steps

### Step 1: Reduce Wayland Input Handler Complexity ✅
- **Completed**: 2026-03-09
- **Deliverable**: Refactor `HandleEvent` in `internal/wayland/input/{pointer,keyboard,touch}.go` from cc>25 to cc≤10 by extracting case-specific handlers
- **Dependencies**: None
- **Acceptance**: 0 functions with cc>15 in input package ✅
- **Validation**: 
  ```bash
  go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.file | contains("internal/wayland/input")) | select(.complexity.cyclomatic > 15)] | length'
  # Expected: 0 ✅
  ```
- **Approach**: Extract each Wayland opcode case into a dedicated `handle<Opcode>` method, matching the X11 event handler pattern already used in `app.go`
- **Result**: 
  - pointer.HandleEvent: cc 31 → 2 (93.5% reduction)
  - keyboard.HandleEvent: cc 28 → 2 (92.9% reduction)
  - touch.HandleEvent: cc 25 → 2 (92.0% reduction)
  - All input tests pass (100% pass rate)
  - Zero functions with cc>15 in input package
- **Files Modified**:
  - `internal/wayland/input/pointer.go`: Extracted 8 helper methods (handleEnterEvent, handleLeaveEvent, handleMotionEvent, etc.)
  - `internal/wayland/input/keyboard.go`: Extracted 6 helper methods (handleKeymapEvent, handleEnterEvent, handleKeyEvent, etc.)
  - `internal/wayland/input/touch.go`: Extracted 5 helper methods (handleDownEvent, handleUpEvent, handleMotionEvent, etc.)

### Step 2: Implement Wayland Event Translation
- **Deliverable**: Complete Wayland event path in `app.go` with translation functions mirroring X11 pattern (`translateWaylandKeyEvent`, `translateWaylandPointerEvent`, etc.)
- **Dependencies**: Step 1 (reduced complexity makes integration easier)
- **Acceptance**: Wayland events dispatch through unified event system; remove "Wayland event path TODO" from ROADMAP
- **Validation**: 
  ```bash
  grep -c "Wayland event path TODO" ROADMAP.md
  # Expected: 0
  go test ./... -run ".*Wayland.*Event"
  # Expected: All pass
  ```
- **Files**: `app.go` (~100 LOC addition), `internal/wayland/input/` (integration)

### Step 3: Deduplicate Demo Code
- **Deliverable**: Extract shared 6-line clone (found in 11 demo files) into a `demo` helper package or shared function
- **Dependencies**: None
- **Acceptance**: Duplication ratio <4%
- **Validation**: 
  ```bash
  go-stats-generator analyze . --skip-tests --format json | jq '.duplication.duplication_ratio'
  # Expected: <0.04
  ```
- **Files**: `internal/demo/`, `cmd/*/main.go`

### Step 4: Refactor gen-atlas main()
- **Deliverable**: Split `cmd/gen-atlas/main.go` main() (cc=20, 103 lines) into focused functions: `parseFlags`, `loadFont`, `generateAtlas`, `writeOutput`
- **Dependencies**: None
- **Acceptance**: main() cc≤5, no functions >50 lines in gen-atlas
- **Validation**: 
  ```bash
  go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.file | contains("gen-atlas")) | select(.complexity.cyclomatic > 5)] | length'
  # Expected: 0
  ```

### Step 5: Resolve Tracked TODOs
- **Deliverable**: Address 3 non-deferred TODOs in library code
  - `app.go:1492`: Add window-specific input event dispatch
  - `concretewidgets.go:361`: Implement ScrollView child management
  - `layout.go:388`: Wire theme from App context
- **Dependencies**: Steps 1-2 (event system completion helps TODO at line 1492)
- **Acceptance**: `grep -c "TODO" *.go | head -1` returns 0 for library files (excluding tests)
- **Validation**: 
  ```bash
  grep -c "TODO" app.go concretewidgets.go layout.go
  # Expected: 0
  ```

### Step 6: Achieve 95% Documentation Coverage
- **Deliverable**: Add doc comments to remaining undocumented methods (89% → 95%)
- **Dependencies**: None
- **Acceptance**: Documentation coverage ≥95% overall
- **Validation**: 
  ```bash
  go-stats-generator analyze . --skip-tests --format json | jq '.documentation.coverage.overall'
  # Expected: ≥95.0
  ```
- **Focus**: `internal/wayland/input/` methods, widget type methods

---

## Metrics Targets (Post-Implementation)

| Metric | Current | Target |
|--------|---------|--------|
| Functions cc>9 | 6 | 0-2 |
| Functions cc>15 | 4 | 0 |
| Duplication ratio | 4.6% | <4% |
| Doc coverage | 91.0% | ≥95% |
| Functions >50 lines | 18 | <10 |
| TODO comments (lib) | 4 | 0 |

## Out of Scope (Future Gaps)

These findings are deferred to future phases:
- **GPU Backend Screenshot Test** (`screenshot_test.go:474`): Depends on Phase 5 GPU rendering; currently uses software backend
- **VT/DPMS Handling** (Phase 7.2): Requires kernel/compositor integration beyond current scope
- **Pure-Go Software Mode**: Long-term goal to eliminate CGO requirement entirely

---

## Validation Commands Summary

```bash
# Full baseline after changes
go-stats-generator analyze . --skip-tests --format json --output metrics-post.json

# Complexity check (target: 0 functions >15)
jq '[.functions[] | select(.complexity.cyclomatic > 15)] | length' metrics-post.json

# Duplication check (target: <0.04)
jq '.duplication.duplication_ratio' metrics-post.json

# Doc coverage check (target: ≥0.95)
jq '.documentation.coverage.overall' metrics-post.json

# TODO count in library code (target: 0)
grep -r "TODO" *.go internal/**/*.go 2>/dev/null | grep -v "_test.go" | wc -l
```
