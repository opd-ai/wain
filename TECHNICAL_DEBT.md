# Technical Debt Register

This file tracks known technical debt items in the Wain codebase.
All `// TODO(TD-N):` comments in source code reference items in this file.

See [CONTRIBUTING.md](./CONTRIBUTING.md) for the process to add new entries.

---

## TD-1 — DragDrop data delivery (RESOLVED)

**Status:** Resolved (v1.1)  
**File:** `app.go:2152`, `event.go`  
**Priority:** P0 / Critical  
**Effort:** Medium

`dispatchDragEvent` called `w.dropHandler("", nil)` unconditionally; the
`DragDropHandler` contract was silently broken. Fixed by:
- Adding `mimeType string` and `data []byte` fields to `DragEvent`.
- Polling `waylandDataDevice` drag channels in `processWaylandDragEvents`.
- Updating `dispatchDragEvent` to pass `evt.mimeType` and `evt.data`.

---

## TD-2 — `bufferCanvas` image / gradient / shadow stubs (RESOLVED)

**Status:** Resolved (v1.1)  
**File:** `concretewidgets.go:122-140`  
**Priority:** P1 / High  
**Effort:** Small

`bufferCanvas.DrawImage`, `LinearGradient`, `RadialGradient`, and `BoxShadow`
were documented no-ops. Fixed by wiring each method to the corresponding
`internal/raster/composite` and `internal/raster/effects` implementations.

---

## TD-3 — GPU shader-to-ISA compilation CI gate (RESOLVED)

**Status:** Resolved (v1.1)  
**File:** `render-sys/tests/shader_compile.rs`, `render-sys/Cargo.toml`, `.github/workflows/ci.yml`, `HARDWARE.md`  
**Priority:** P2 / Medium  
**Effort:** Medium

Goals #9 (Intel EU backend) and #10 (AMD RDNA backend) are partially achieved:
code compiles and unit tests pass, but no automated test verifies that WGSL
shaders produce non-empty ISA byte sequences without physical GPU hardware.

**Resolution:** Added `render-sys/tests/shader_compile.rs` — a Cargo integration
test that loads all 7 WGSL UI shaders, compiles each through naga → Intel EU
(Gen9/Gen11/Gen12) and AMD RDNA (RDNA1/RDNA2/RDNA3) lowering passes, and asserts
the emitted byte blobs are non-empty and correctly aligned. Added `"rlib"` to
`Cargo.toml` crate-type to enable integration tests alongside the staticlib.
Added explicit "Run shader-to-ISA compilation CI gate" step to CI. Updated
`HARDWARE.md` to distinguish "code-verified" from "hardware-validated" GPU support.

---

## TD-4 — AT-SPI2 build tag undocumented in README / STABILITY.md (RESOLVED)

**Status:** Resolved (v1.1)  
**File:** `README.md`, `STABILITY.md`, `accessibility.go`  
**Priority:** P3 / Low  
**Effort:** Trivial

`EnableAccessibility` is a no-op returning `nil` without `-tags=atspi`. The
`ACCESSIBILITY.md` documents this, but the README feature bullet and the
`STABILITY.md` stability guarantees section did not.

**Resolution:** Added parenthetical to `README.md` feature bullet (line 53):
`requires \`-tags=atspi\` — see ACCESSIBILITY.md`. Added a "Build Tag: atspi"
section to `STABILITY.md` with usage instructions.

---

## TD-5 — golangci-lint version mismatch (RESOLVED)

**Status:** Resolved (v1.1)  
**File:** `.golangci.yml`, `.github/workflows/ci.yml`  
**Priority:** P0 / High  
**Effort:** Trivial

`.golangci.yml` listed removed linters (`structcheck`, `varcheck`, `deadcode`)
and the config used v1 format, causing hard failures against Go 1.24. Fixed by:
- Removing the three deprecated linters.
- Migrating to golangci-lint v2 config format.
- Updating CI to use `golangci-lint-action@v7` with `version: v2`.

---

## TD-6 — `internal/a11y` has zero test coverage (RESOLVED)

**Status:** Resolved (v1.1)  
**File:** `internal/a11y/manager_test.go` (new)  
**Priority:** P3 / Medium  
**Effort:** Medium

The AT-SPI2 implementation had no test files. Any refactor silently broke
screen-reader support without CI feedback.

**Resolution:** Added `internal/a11y/manager_test.go` with headless unit tests
covering `AccessibleObject` setters, all four AT-SPI2 interface wrappers
(`accessibleIface`, `componentIface`, `actionIface`, `textIface`), and the
D-Bus-free Manager paths (`lookupObject`, `SetBounds`, `SetText`, `SetName`,
`SetFocused`). Coverage: 74.2% with `-tags=atspi`, exceeding the 70% target.

---

*Last updated: 2026-03-15*
