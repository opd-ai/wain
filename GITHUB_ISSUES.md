# GitHub Issues for Technical Debt Items

This file contains issue descriptions for all TODO comments in the codebase. These can be converted to GitHub Issues for better tracking visibility and collaboration.

**How to create issues:**
```bash
# Using GitHub CLI (if available):
gh issue create --title "[TD-2] Implement proper child management for ScrollView" \
  --body-file <(sed -n '/^## Issue 1:/,/^---$/p' GITHUB_ISSUES.md) \
  --label "technical-debt"

# Or manually: Copy each issue section below and paste into GitHub Issues web UI
```

---

## Issue 1: [TD-2] Implement proper child management for ScrollView

**Labels:** `technical-debt`, `enhancement`, `priority:medium`

### Description
The public `ScrollView.Add(child PublicWidget)` method is currently a stub and doesn't actually add children to the scroll container.

### Location
**File:** `concretewidgets.go:361`

### Priority
Medium

### Impact
ScrollView widget cannot hold child widgets, limiting its utility for building scrollable layouts with multiple children.

### Effort Estimate
~2-3 hours
- Add child container to `internal/ui/widgets.ScrollContainer`
- Implement layout pass for children
- Bridge PublicWidget interface to internal widget representation

### Related Items
- TECHNICAL_DEBT.md TD-2

### Implementation Notes
Need to extend `internal/ui/widgets.ScrollContainer` to accept PublicWidget children (currently only works with internal widgets). Requires bridging PublicWidget interface to internal widget representation.

### Code Reference
```go
// TODO(TD-2): Implement proper child management for ScrollView
func (s *ScrollView) Add(child PublicWidget) {
    // stub
}
```

---

## Issue 2: [TD-3] Theme system integration for Panel widget

**Labels:** `technical-debt`, `enhancement`, `priority:low`

### Description
Panel widget currently hardcodes `DefaultDark()` theme instead of reading from `App.theme`.

### Location
**File:** `layout.go:388`

### Priority
Low

### Impact
Panels ignore app-wide theme settings and always use dark theme, breaking theme consistency when users want light themes or custom themes.

### Effort Estimate
~30 minutes (change to `app.theme.Panel()` once theme system exists)

### Blocked By
App-level theme system implementation (not yet designed)

### Related Items
- TECHNICAL_DEBT.md TD-3
- Future App.theme field implementation

### Implementation Notes
Once the App-level theme system is implemented, Panel should respect the global theme setting while allowing per-widget style overrides. Simple fix once the blocking work is complete.

### Code Reference
```go
base:     DefaultDark(), // TODO(TD-3): Get from App.theme when available
```

---

## Issue 3: [TD-4] Full Wayland event reading and dispatch

**Labels:** `technical-debt`, `bug`, `priority:high`, `wayland`

### Description
The Wayland display server currently only sends requests but doesn't read events from the compositor. This prevents handling user input, window state changes, and other compositor events.

### Location
**File:** `app.go:1471`

### Priority
High

### Impact
Wayland applications cannot receive user input (keyboard, mouse) or respond to compositor events, making the Wayland backend non-functional for interactive applications.

### Effort Estimate
~8-12 hours
- Wire protocol event parser: 2h
- Event dispatch loop: 3h
- Event mapping to wain Event types: 3h
- Testing with real compositor: 4h

### Related Items
- TECHNICAL_DEBT.md TD-4
- ROADMAP.md Phase 9.2 "Wayland Event Reading" (currently incomplete)

### Implementation Steps
1. Implement wire protocol event parser in `internal/wayland/wire`
2. Add event dispatch loop to `App.pollEvents()` for Wayland path
3. Map Wayland events (wl_keyboard, wl_pointer, etc.) to wain Event types
4. Test with real Wayland compositor (Weston, Sway)

### Code Reference
```go
// TODO(TD-4): Implement full Wayland event reading and dispatch
```

---

## Issue 4: [TD-5] Implement immediate loading in Intel EU instruction encoder

**Labels:** `technical-debt`, `enhancement`, `priority:medium`, `rust`, `gpu`

### Description
Intel EU instruction encoder needs proper implementation of immediate value loading.

### Location
**File:** `render-sys/src/eu/lower.rs:577`

### Priority
Medium

### Impact
Limited instruction encoding capabilities for Intel GPU backend. May cause incorrect shader compilation or runtime errors when immediate values are needed.

### Effort Estimate
~3-4 hours

### Related Items
- Intel EU backend implementation (Phase 3-4)
- Shader compilation pipeline

### Implementation Notes
Current code has a placeholder comment. Need to implement proper immediate value encoding according to Intel GPU instruction format specifications.

### Code Reference
```rust
// TODO(TD-5): Implement immediate loading properly
```

---

## Issue 5: [TD-6] Implement swizzle control bits in EU instruction encoding

**Labels:** `technical-debt`, `enhancement`, `priority:medium`, `rust`, `gpu`

### Description
Intel EU instruction encoder needs swizzle control bit implementation for vector operations.

### Location
**File:** `render-sys/src/eu/lower.rs:1554`

### Priority
Medium

### Impact
Missing swizzle control prevents proper vector component manipulation in shaders, limiting shader capabilities for the Intel GPU backend.

### Effort Estimate
~2-3 hours

### Related Items
- Issue #4 (Immediate loading)
- Intel EU backend implementation

### Implementation Notes
Swizzle control bits determine how vector components are rearranged during operations. Need to encode these according to Intel GPU Gen9+ instruction format.

### Code Reference
```rust
// TODO(TD-6): Implement swizzle control bits in instruction encoding
```

---

## Issue 6: [TD-7] Complete GPU shader compilation for solid color rendering

**Labels:** `technical-debt`, `enhancement`, `priority:high`, `rust`, `gpu`, `shader`

### Description
GPU backend shader compilation and batch submission for solid color rendering is incomplete.

### Location
**File:** `render-sys/src/shader.rs:413`

### Priority
High

### Impact
Core rendering primitive (solid color fills) doesn't work on GPU backend, blocking all GPU-accelerated rendering.

### Effort Estimate
~4-6 hours

### Related Items
- ROADMAP.md Phase 4 (GPU Rendering Pipeline)
- Issues #7-11 (other shader TODOs)

### Implementation Steps
1. Compile shader to EU binary or RDNA ISA
2. Submit batch to GPU command queue
3. Execute rendering operation
4. Validate output with test cases

### Code Reference
```rust
// TODO(TD-7): Compile shader to EU binary, submit batch, render
```

---

## Issue 7: [TD-8] Complete GPU shader compilation for gradient rendering

**Labels:** `technical-debt`, `enhancement`, `priority:high`, `rust`, `gpu`, `shader`

### Description
GPU backend shader compilation for gradient rendering is incomplete.

### Location
**File:** `render-sys/src/shader.rs:446`

### Priority
High

### Impact
Gradient rendering unavailable on GPU backend, limiting visual quality and design capabilities.

### Effort Estimate
~3-5 hours

### Related Items
- Issue #6 (solid color - prerequisite)
- ROADMAP.md Phase 4

### Implementation Steps
1. Compile gradient shader
2. Submit batch with gradient parameters
3. Render gradient geometry

### Code Reference
```rust
// TODO(TD-8): Compile shader, submit batch, render gradient
```

---

## Issue 8: [TD-9] Complete GPU shader compilation for textured quad rendering

**Labels:** `technical-debt`, `enhancement`, `priority:high`, `rust`, `gpu`, `shader`

### Description
GPU backend shader compilation for textured quad rendering is incomplete.

### Location
**File:** `render-sys/src/shader.rs:471`

### Priority
High

### Impact
Image rendering unavailable on GPU backend. Cannot display images, icons, or UI textures.

### Effort Estimate
~4-6 hours

### Related Items
- Issue #6 (solid color - prerequisite)
- Texture upload and management system

### Implementation Steps
1. Compile texture sampling shader
2. Upload texture data to GPU
3. Render textured quad with proper UV mapping

### Code Reference
```rust
// TODO(TD-9): Compile shader, upload texture, render textured quad
```

---

## Issue 9: [TD-10] Complete GPU shader compilation for SDF text rendering

**Labels:** `technical-debt`, `enhancement`, `priority:high`, `rust`, `gpu`, `shader`, `text`

### Description
GPU backend shader compilation for SDF (Signed Distance Field) text rendering is incomplete.

### Location
**File:** `render-sys/src/shader.rs:488`

### Priority
High

### Impact
Text rendering unavailable on GPU backend. UI cannot display text, making GPU backend unusable for most applications.

### Effort Estimate
~5-7 hours

### Related Items
- Issue #8 (textured quad - prerequisite)
- Font atlas generation system
- SDF text rasterizer

### Implementation Steps
1. Compile SDF text shader
2. Upload SDF atlas texture
3. Render text glyphs with proper anti-aliasing

### Code Reference
```rust
// TODO(TD-10): Compile shader, upload SDF atlas, render text
```

---

## Issue 10: [TD-11] Complete GPU shader compilation for rounded rectangle rendering

**Labels:** `technical-debt`, `enhancement`, `priority:medium`, `rust`, `gpu`, `shader`

### Description
GPU backend shader compilation for rounded rectangle rendering with clipping is incomplete.

### Location
**File:** `render-sys/src/shader.rs:504`

### Priority
Medium

### Impact
UI widgets with rounded corners won't render correctly on GPU backend, affecting visual design and polish.

### Effort Estimate
~3-4 hours

### Related Items
- Issue #6 (solid color - prerequisite)
- UI widget styling system

### Implementation Steps
1. Compile rounded rectangle shader with SDF-based corner rounding
2. Implement clipping logic
3. Render with anti-aliasing

### Code Reference
```rust
// TODO(TD-11): Compile shader, render rounded rectangle with clipping
```

---

## Issue 11: [TD-12] Complete GPU shader compilation for radial gradient rendering

**Labels:** `technical-debt`, `enhancement`, `priority:medium`, `rust`, `gpu`, `shader`

### Description
GPU backend shader compilation for radial gradient rendering is incomplete.

### Location
**File:** `render-sys/src/shader.rs:520`

### Priority
Medium

### Impact
Advanced gradient effects unavailable on GPU backend, limiting visual design capabilities.

### Effort Estimate
~3-4 hours

### Related Items
- Issue #7 (linear gradient)
- ROADMAP.md Phase 4

### Implementation Steps
1. Compile radial gradient shader
2. Submit batch with gradient center/radius parameters
3. Render radial gradient

### Code Reference
```rust
// TODO(TD-12): Compile shader, render radial gradient
```

---

## Issue 12: [TD-13] Complete GPU shader compilation for blur effects

**Labels:** `technical-debt`, `enhancement`, `priority:low`, `rust`, `gpu`, `shader`, `effects`

### Description
GPU backend shader compilation for two-pass blur effects is incomplete.

### Location
**File:** `render-sys/src/shader.rs:536`

### Priority
Low

### Impact
Blur effects (shadows, backgrounds) unavailable on GPU backend. Affects visual polish but not core functionality.

### Effort Estimate
~4-6 hours (two-pass algorithm is more complex)

### Related Items
- Issue #6 (solid color - prerequisite)
- Framebuffer/rendertarget system for multi-pass rendering

### Implementation Steps
1. Compile horizontal blur shader
2. Compile vertical blur shader
3. Implement two-pass rendering with intermediate framebuffer
4. Optimize for performance (separable convolution)

### Code Reference
```rust
// TODO(TD-13): Compile shader, render two-pass blur
```

---

## Issue 13: [TD-14] Implement GPU backend rendering for integration tests

**Labels:** `technical-debt`, `testing`, `priority:medium`, `gpu`

### Description
Integration tests for screenshot functionality need GPU backend rendering implementation.

### Location
**File:** `internal/integration/screenshot_test.go:474`

### Priority
Medium

### Impact
Integration tests cannot validate GPU rendering correctness. Software rasterizer tests only, limiting CI coverage.

### Effort Estimate
~2-3 hours (depends on Phase 5 completion)

### Blocked By
- ROADMAP.md Phase 5 completion
- Issues #6-12 (GPU shader implementations)

### Related Items
- GPU backend integration tests
- CI/CD pipeline GPU testing

### Implementation Notes
This is blocked by Phase 5 (GPU rendering pipeline) completion. Once GPU backend is functional, update tests to run against both software and GPU backends.

### Code Reference
```go
// TODO(TD-14): Implement GPU backend rendering when Phase 5 is complete
```

---

## Summary Statistics

- **Total TODO items:** 13
- **Priority breakdown:**
  - High: 5 (TD-4, Issues #6-9)
  - Medium: 6 (TD-2, Issues #4-5, #10-11, #13)
  - Low: 2 (TD-3, Issue #12)
- **Categories:**
  - GPU/Shader: 8 (Issues #4-12)
  - Wayland: 1 (TD-4)
  - UI Widgets: 2 (TD-2, TD-3)
  - Testing: 1 (Issue #13)

## Automation

To bulk-create these issues using GitHub CLI:

```bash
#!/bin/bash
# Requires: gh CLI authenticated

# Function to create issue from section
create_issue() {
  local title="$1"
  local start_line="$2"
  local end_line="$3"
  
  local body=$(sed -n "${start_line},${end_line}p" GITHUB_ISSUES.md)
  gh issue create --title "$title" --body "$body" --label "technical-debt"
}

# Create all issues (line numbers approximate - adjust as needed)
create_issue "[TD-2] Implement proper child management for ScrollView" 15 52
create_issue "[TD-3] Theme system integration for Panel widget" 56 91
create_issue "[TD-4] Full Wayland event reading and dispatch" 95 145
create_issue "Implement immediate loading in Intel EU instruction encoder" 149 182
create_issue "Implement swizzle control bits in EU instruction encoding" 186 219
create_issue "Complete GPU shader compilation for solid color rendering" 223 263
create_issue "Complete GPU shader compilation for gradient rendering" 267 301
create_issue "Complete GPU shader compilation for textured quad rendering" 305 343
create_issue "Complete GPU shader compilation for SDF text rendering" 347 390
create_issue "Complete GPU shader compilation for rounded rectangle rendering" 394 431
create_issue "Complete GPU shader compilation for radial gradient rendering" 435 472
create_issue "Complete GPU shader compilation for blur effects" 476 520
create_issue "Implement GPU backend rendering for integration tests" 524 566
```

## Notes

- All TODO comments already reference TECHNICAL_DEBT.md entries
- Creating GitHub Issues provides better visibility and collaboration tools
- Labels can be customized per project preferences
- Consider using GitHub Projects for tracking progress across issues
