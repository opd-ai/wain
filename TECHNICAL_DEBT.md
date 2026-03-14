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

## TD-3 — GPU shader-to-ISA compilation CI gate (IN PROGRESS)

**Status:** In Progress  
**File:** `render-sys/src/eu/`, `render-sys/src/rdna/`, `.github/workflows/ci.yml`  
**Priority:** P2 / Medium  
**Effort:** Medium

Goals #9 (Intel EU backend) and #10 (AMD RDNA backend) are partially achieved:
code compiles and unit tests pass, but no automated test verifies that WGSL
shaders produce non-empty ISA byte sequences without physical GPU hardware.

**Planned action:** Add a Rust integration test (`tests/shader_compile.rs`) that
loads each of the 7 WGSL shaders, runs them through the EU and RDNA lowering
passes, and asserts the emitted byte blobs are non-empty. Wire this into CI.

---

## TD-4 — AT-SPI2 build tag undocumented in README / STABILITY.md

**Status:** Open  
**File:** `README.md`, `STABILITY.md`, `accessibility.go`  
**Priority:** P3 / Low  
**Effort:** Trivial

`EnableAccessibility` is a no-op returning `nil` without `-tags=atspi`. The
`ACCESSIBILITY.md` documents this, but the README feature bullet and the
`STABILITY.md` stability guarantees section do not.

**Planned action:** Add a parenthetical note to both files pointing at
`ACCESSIBILITY.md`.

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

## TD-6 — `internal/a11y` has zero test coverage

**Status:** Open  
**File:** `internal/a11y/` (10 source files, 75 functions)  
**Priority:** P3 / Medium  
**Effort:** Medium

The AT-SPI2 implementation has no test files. Any refactor silently breaks
screen-reader support without CI feedback.

**Planned action:** Add `internal/a11y/manager_test.go` with stub D-Bus
connection tests covering registration, focus events, and action invocation.
Target: ≥ 70% statement coverage with `-tags=atspi`.

---

*Last updated: 2026-03-14*
