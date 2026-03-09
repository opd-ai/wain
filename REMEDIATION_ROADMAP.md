# PRODUCTION READINESS ASSESSMENT

## Project Context
- **Language:** Go 1.24 + Rust (stable, edition 2021) — hybrid via CGO/musl static linking
- **Type:** Library / Framework (UI toolkit with public API + demo binaries)
- **Deployment model:** Fully static Linux binary (musl-gcc, zero runtime dependencies)
- **Module path:** `github.com/opd-ai/wain`
- **Go packages:** 65 (25 cmd/, 40 internal/ + root)
- **Rust crate:** `render-sys` (staticlib, GPU rendering backend)
- **Go dependencies:** `golang.org/x/sys` v0.42.0 (sole external dependency)
- **Rust dependencies:** `nix` v0.27 (ioctl), `naga` v0.14 (shader compilation)
- **Existing CI checks:**
  - Rust tests (`cargo test --target x86_64-unknown-linux-musl`)
  - Go tests (`go test ./...`)
  - Integration tests (TestPublicAPI, TestAccessibility, TestExampleApp)
  - Build verification (`scripts/verify-build.sh`)
  - Static binary assertion (`ldd` check)
- **Codebase size:** 12,215 Go LOC (165 files), ~14,400 Rust LOC (32 files)
- **Total functions:** 1,357 Go functions/methods, ~200 Rust functions

---

## Readiness Summary

| Gate | Score | Threshold | Status | Weight for Library/Framework |
|------|-------|-----------|--------|------------------------------|
| Complexity | All functions ≤10 cyclomatic | All ≤10 | ✅ **PASS** | High |
| Function Length | 100 functions >30 lines (7.4%) | All ≤30 lines | ❌ **FAIL** | Medium |
| Documentation | 90.7% overall coverage | ≥80% | ✅ **PASS** | Critical |
| Duplication | 3.71% ratio (64 clone pairs) | <5% | ✅ **PASS** | Medium |
| Circular Dependencies | Zero detected | Zero | ✅ **PASS** | High |
| Naming | 78 violations (score 0.99) | Zero violations | ❌ **FAIL** | Critical |
| Concurrency Safety | 3 goroutine leaks (demo code) | No high-risk patterns | ⚠️ **CONDITIONAL PASS** | Medium |

**Overall: 4.5/7 gates passing — CONDITIONALLY READY**

### Gate Details

#### ✅ Complexity — PASS
- Average cyclomatic complexity: 3.3
- Maximum cyclomatic complexity: 10 (`cmd/window-demo/main.go:main`)
- Functions with complexity >10: **0**
- All functions within threshold.

#### ❌ Function Length — FAIL
- Average function length: 11.0 lines
- Functions >30 lines: 100 (7.4% of 1,357)
- Functions >50 lines: 30 (2.2%)
- Functions >100 lines: 1 (0.1%)

**Top 5 longest functions:**
| Function | File | Lines | Complexity |
|----------|------|-------|------------|
| `main` | `cmd/window-demo/main.go:21` | 110 | 10 |
| `main` | `cmd/auto-render-demo/main.go:24` | 100 | 9 |
| `setupX11Context` | `cmd/x11-dmabuf-demo/main.go:165` | 87 | 8 |
| `main` | `cmd/theme-demo/main.go:10` | 82 | 1 |
| `main` | `cmd/callback-demo/main.go:18` | 81 | 4 |

**Calibration note:** 78 of 100 long functions are in `cmd/` (demo/tool code). The 22 in `internal/` are primarily protocol setup and rendering pipelines where length is expected.

#### ✅ Documentation — PASS
| Scope | Coverage |
|-------|----------|
| Package | 100.0% |
| Function | 98.4% |
| Type | 90.0% |
| Method | 89.0% |
| **Overall** | **90.7%** |

Quality score: 100/100, with 28 code examples and 5,857 inline comments.

#### ✅ Duplication — PASS
- Clone pairs: 64
- Duplicated lines: 1,042 of 12,215 (3.71%)
- Largest clone: 26 lines
- Majority of clones are in `cmd/` demo boilerplate (window setup patterns).

#### ✅ Circular Dependencies — PASS
- Zero circular dependencies detected across 65 packages.

#### ❌ Naming — FAIL
- File name violations: 30 (mostly "stuttering" — e.g., `curves/curves.go`)
- Identifier violations: 46 (single-letter variables: `x`, `y` in demo/graphics code)
- Package name violations: 2 (`wain` directory mismatch, `core` too generic)
- Overall naming score: 0.99

**Top 5 violations:**
| Name | Type | File |
|------|------|------|
| `x` | single letter | `cmd/amd-triangle-demo/main.go:253` |
| `y` | single letter | `cmd/amd-triangle-demo/main.go:254` |
| `wain` | directory mismatch | Root package |
| `core` | generic name | `internal/raster/core` |
| `constants.go` | generic file | `internal/demo/constants.go` |

**Calibration note:** Single-letter `x`/`y` variables in graphics coordinate code are idiomatic. File stuttering (e.g., `atlas/atlas.go`) is a Go convention debate, not a correctness issue. The 2 package violations are substantive.

#### ⚠️ Concurrency Safety — CONDITIONAL PASS
- **Goroutine leaks (3):** All in `cmd/` demo binaries, not in library code
  - `cmd/callback-demo/main.go:48`
  - `cmd/event-demo/main.go:28`
  - `cmd/window-demo/main.go:41`
- **Synchronization:** Proper `sync.Mutex`, `sync.RWMutex`, `atomic.Uint32` throughout
- **No data races detected** in library code
- **No channel misuse** in library code

---

## Security Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH | 1 |
| MEDIUM | 5 |
| LOW | 4 |

**Security Verdict: CONDITIONAL PASS**
- Zero CRITICAL findings
- 1 HIGH finding with documented mitigation path

### Security Findings Table

| ID | Severity | Category | Location | Description | Recommendation |
|----|----------|----------|----------|-------------|----------------|
| S-001 | HIGH | Error Handling | `internal/buffer/ring.go:152` | `ClaimSlot()` calls `panic()` on out-of-bounds slot index. Library code should never panic — this will crash the calling application. | Replace `panic()` with an `error` return. Callers already handle errors in the buffer management path. |
| S-002 | MEDIUM | Error Handling | `internal/x11/wire/setup.go:161-186` | X11 protocol setup parsing discards 15+ decode errors via `_ = DecodeUint16(r)`. Malformed X11 server responses will silently produce corrupt state. | Propagate decode errors and return an error from the setup parsing function. |
| S-003 | MEDIUM | Error Handling (Rust) | `render-sys/src/batch.rs:202-303` | 6 `.unwrap()` calls in production `BatchBuilder` code. Failures during batch construction will panic and crash the process via FFI. | Replace `.unwrap()` with `?` operator and propagate `Result` through the batch API. |
| S-004 | MEDIUM | Error Handling (Rust) | `render-sys/src/allocator.rs:345-350` | `munmap` failure silently ignored (only logged in debug builds). Memory leak on failure. | Log warning in all builds; consider returning `Result` from `Drop` or using a cleanup method. |
| S-005 | MEDIUM | Error Handling (Rust) | `render-sys/src/amd.rs:693` | `unmap_buffer_from_va` result silently discarded with `let _ =`. GPU virtual address leak on failure. | Log the error and propagate where possible. |
| S-006 | MEDIUM | Unsafe Code (Rust) | `render-sys/src/allocator.rs:197-312` | 7 `unsafe` blocks for mmap/munmap/slice operations. Well-documented but any bug here causes undefined behavior across the FFI boundary. | Add `// SAFETY:` comments per Rust convention documenting the invariants each unsafe block relies on. |
| S-007 | LOW | Concurrency | `cmd/callback-demo/main.go:48` | Goroutine spawned without context or done channel. | Add `context.Context` cancellation for graceful shutdown. |
| S-008 | LOW | Concurrency | `cmd/event-demo/main.go:28` | Goroutine spawned without context or done channel. | Add `context.Context` cancellation for graceful shutdown. |
| S-009 | LOW | Concurrency | `cmd/window-demo/main.go:41` | Goroutine spawned without context or done channel. | Add `context.Context` cancellation for graceful shutdown. |
| S-010 | LOW | Dependency Hygiene | `go.sum` | `golang.org/x/sys` v0.20.0 pinned alongside v0.42.0 (duplicate checksum entries). Minimal dependency footprint is excellent — only 1 Go and 2 Rust external dependencies. | Remove stale v0.20.0 entry from go.sum via `go mod tidy`. |

---

## Slop Summary

| Severity | Count |
|----------|-------|
| HIGH | 4 |
| MEDIUM | 10 |
| LOW | 9 |

**Slop Debt Score: HIGH** (23 total findings, 4 HIGH)

### Slop Findings Table

| ID | Category | Severity | Location | Description | Suggested Fix |
|----|----------|----------|----------|-------------|---------------|
| SL-001 | Test Slop | HIGH | `accessibility_test.go` (8 tests) | 8 test functions contain only `t.Logf()` — no assertions. These tests always pass regardless of behavior. | Add assertions validating actual accessibility behavior. Example: `TestKeyboardNavigation` should assert that focus moves between widgets after simulated Tab key. |
| SL-002 | Test Slop | HIGH | `concretewidgets_test.go` (8 tests, lines 169-306) | 8 test functions have no assertions — `TestSpacerDraw`, `TestButtonImplementsPublicWidget`, etc. only instantiate objects. | Add type assertions: `var _ PublicWidget = (*Button)(nil)` for interface compliance tests. Add draw output validation for `TestSpacerDraw`. |
| SL-003 | Magic Numbers (Rust) | HIGH | `render-sys/src/eu/encoding.rs:24-68` | 50+ bare hex opcodes without named constants (e.g., `0x40` for Add, `0x01` for Mov). Any encoding bug is invisible. | Create `mod opcodes { pub const ADD: u8 = 0x40; pub const MUL: u8 = 0x41; ... }` |
| SL-004 | Magic Numbers (Rust) | HIGH | `render-sys/src/rdna/encoding.rs:29-125` | 50+ bare hex opcodes and bitmasks for RDNA instruction encoding without named constants. | Create `mod rdna_opcodes { pub const V_MOV_B32: u8 = 0x01; ... }` and `mod bitfields { pub const VOP3_PREFIX: u32 = 0xD1000000; ... }` |
| SL-005 | Dead Code | MEDIUM | `event.go:299-371` | 7 unexported functions (`translateX11KeyPressEvent`, `translateX11KeyReleaseEvent`, `translateX11ButtonPressEvent`, `translateX11ButtonReleaseEvent`, `translateX11MotionNotifyEvent`, `translateX11ConfigureNotifyEvent`, `linuxToKeysym`) are defined but unreferenced. | These appear to be Phase 10 planned features. Either wire them into the event dispatch path or gate behind a build tag until needed. |
| SL-006 | Dead Code | MEDIUM | `color.go:29-34` | `toInternal()` and `toU32()` — unexported helper functions with zero callers. | Remove or connect to the public Color API. |
| SL-007 | Dead Code | MEDIUM | `internal/render/backend/batch.go:37`, `scissor.go:39` | `batchCommands()` and `buildScissorStateBuffer()` — GPU rendering helpers with zero callers. | Connect to the render backend pipeline or remove. |
| SL-008 | Naming | MEDIUM | 10 files with stuttering names | `internal/raster/curves/curves.go`, `internal/render/atlas/atlas.go`, `internal/render/backend/backend.go`, etc. | Rename to non-stuttering: `curves/doc.go`, `atlas/doc.go`, etc. — or accept as Go convention. |
| SL-009 | TODO Debt | MEDIUM | `render-sys/src/shader.rs:175-215` | 7 TODO comments marking stub shader rendering functions (`render_solid_color`, `render_gradient`, `render_textured_quad`, etc.) — incomplete GPU pipeline. | Convert to tracked issues or complete the implementations. |
| SL-010 | Duplication | MEDIUM | `cmd/` demo boilerplate | 64 clone pairs across demo binaries (window setup, event loop, buffer creation). Largest clone: 26 lines. | Extract shared demo scaffolding into `internal/demo/` helpers. E.g., `demo.SetupWaylandWindow()` replacing the 7-instance 7-line clone. |
| SL-011 | Missing Docs (Rust) | MEDIUM | `render-sys/src/amd.rs`, `i915.rs`, `xe.rs`, `allocator.rs` | 52+ public structs/enums without doc comments. DRM ioctl structures, GPU constants, and driver types are undocumented. | Add `///` doc comments explaining the DRM/GPU semantics for each public type. Priority: types exposed via FFI to Go. |
| SL-012 | Dead Code (Rust) | MEDIUM | `render-sys/src/eu/instruction.rs:13-109` | 5 `#[allow(dead_code)]` annotations suppressing warnings on instruction types (`EUOpcode`, `RegFile`, `Register`, `SharedFunctionID`, `SendDescriptor`). | Document which variants are used and remove `#[allow(dead_code)]` — or prefix unused variants with `_`. |
| SL-013 | Duplicate Code (Rust) | MEDIUM | `render-sys/src/eu/` vs `render-sys/src/rdna/` | Near-duplicate `lower_math()`, `lower_image_sample()`, `lower_expression()` functions across GPU backends. | Extract shared lowering logic into a trait or shared utility module. Backend-specific parts remain in `eu/` and `rdna/`. |
| SL-014 | TODO Debt | MEDIUM | `layout.go:178,189,200` | 3 TODOs for unimplemented layout features (style customization, cross-axis alignment) referencing Phases 10.4–10.5. | Convert to GitHub Issues with phase labels. |
| SL-015 | Test Slop | LOW | `internal/x11/shm/shm_test.go` (7 functions) | Test functions call `t.Fatal()` without `t.Helper()`, causing incorrect line numbers in failure output. | Add `t.Helper()` as the first line in each helper function. |
| SL-016 | Naming | LOW | 46 identifier violations | Single-letter variables (`x`, `y`) in graphics code. Flagged by naming analysis but idiomatic for coordinate math. | Accept as intentional — `x`/`y` in 2D graphics coordinate code is standard practice. No action needed. |
| SL-017 | Test Slop | LOW | `internal/ui/scale/scale_test.go:99` | `TestConcurrentAccess` spawns goroutines but has no assertions — only verifies no panics. | Add assertions verifying the scale factor value is correct after concurrent access. |
| SL-018 | Unused Receivers | LOW | 80 instances across codebase | Methods with unused receivers (interface compliance stubs). | Use `_` as receiver name for interface compliance methods: `func (_ MyType) Method()`. |
| SL-019 | Anti-pattern | LOW | 177 instances | `bare_error_return` pattern — functions returning `error` without wrapping context. | Wrap errors with `fmt.Errorf("context: %w", err)` in critical paths. Low priority for internal code. |
| SL-020 | Naming | LOW | `internal/raster/core` | Package name `core` is generic. | Rename to `rastercore` or `pixel` to be more descriptive. |
| SL-021 | Test Slop | LOW | `internal/render/backend/backend_test.go:162` | `TestPackVertices` has no assertions. | Add assertions on packed vertex buffer contents. |
| SL-022 | Dead Code (Rust) | LOW | `render-sys/src/rdna/lower.rs:221,227` | Expression lowering results assigned to `_` variables — incomplete implementation. | Complete the RDNA lowering implementation or add explicit TODO comments. |
| SL-023 | TODO Debt | LOW | `render-sys/src/eu/lower.rs:168,273` | 2 TODOs for Intel EU lowering (`immediate loading`, `swizzle control bits`). | Convert to tracked issues. |

---

## Remediation Plan

### Phase 1: Critical Security Fixes
*Priority: Eliminate crash-inducing code in library paths*

- [ ] **S-001:** Replace `panic()` with `error` return in `internal/buffer/ring.go:152` (`ClaimSlot`). Update callers to handle the new error return.
- [ ] **S-003:** Replace 6 `.unwrap()` calls in `render-sys/src/batch.rs:202-303` with `?` operator. Propagate `Result<_, BatchError>` through the batch builder API.
- [ ] **S-002:** Propagate decode errors in `internal/x11/wire/setup.go:161-186` instead of discarding with `_ =`. Return error from `ParseSetupReply`.

### Phase 2: Documentation & Naming (Highest-weight gates for Library/Framework)
*Priority: Library consumers need clear API docs and idiomatic naming*

- [ ] **Gate: Naming** — Fix 2 package-level violations:
  - Rename `internal/raster/core` to `internal/raster/rastercore` or similar
  - Address root package `wain` directory mismatch (document rationale if intentional)
- [ ] **SL-011:** Add `///` doc comments to 52+ undocumented public Rust types in `amd.rs`, `i915.rs`, `xe.rs`, `allocator.rs`, `pipeline.rs`, `surface.rs`
- [ ] **S-006:** Add `// SAFETY:` comments to all 10 `unsafe` blocks in Rust code per convention

### Phase 3: High-Severity Slop Remediation
*Priority: Tests that always pass mask real bugs*

- [ ] **SL-001:** Add meaningful assertions to 8 tests in `accessibility_test.go`
- [ ] **SL-002:** Add meaningful assertions to 8 tests in `concretewidgets_test.go`
- [ ] **SL-003:** Extract bare hex opcodes in `render-sys/src/eu/encoding.rs` into named constants module
- [ ] **SL-004:** Extract bare hex opcodes in `render-sys/src/rdna/encoding.rs` into named constants module

### Phase 4: Remaining Readiness Gates
*Priority: Function length gate and concurrency cleanup*

- [ ] **Gate: Function Length** — Refactor top 10 longest functions (prioritize `internal/` over `cmd/`):
  - `internal/render/atlas/atlas.go:261` `tryAllocateInPage` (66 lines) — extract sub-steps
  - `internal/raster/composite/composite.go:35` `Blit` (62 lines) — extract bounds checking
  - `internal/x11/client/client.go:71` `Connect` (61 lines) — extract auth and socket setup
  - `internal/raster/text/atlas.go:83` `NewAtlas` (60 lines) — extract glyph rendering loop
  - `internal/render/backend/submit.go:112` `buildBatchBuffer` (57 lines) — extract command encoding
  - `internal/raster/text/text.go:47` `drawGlyph` (53 lines) — extract pixel operations
  - `internal/wayland/socket/socket.go:244` `MakePair` (53 lines) — extract socket configuration
  - `internal/x11/dri3/dri3.go:248` `PixmapFromBuffers` (53 lines) — extract buffer validation
- [ ] **Gate: Concurrency** — Add `context.Context` to goroutines in:
  - `cmd/callback-demo/main.go:48`
  - `cmd/event-demo/main.go:28`
  - `cmd/window-demo/main.go:41`
- [ ] **S-004/S-005:** Log `munmap`/`unmap_buffer_from_va` errors in `render-sys/src/allocator.rs:345` and `amd.rs:693`

### Phase 5: Medium/Low Slop Cleanup
*Priority: Reduce entropy for long-term maintainability*

- [ ] **SL-005:** Wire or remove 7 dead X11 event translation functions in `event.go:299-371`
- [ ] **SL-006:** Wire or remove `toInternal()`/`toU32()` in `color.go:29-34`
- [ ] **SL-007:** Wire or remove `batchCommands()`/`buildScissorStateBuffer()` in `internal/render/backend/`
- [ ] **SL-010:** Extract demo boilerplate (64 clone pairs) into `internal/demo/` shared helpers
- [ ] **SL-012:** Remove `#[allow(dead_code)]` in `render-sys/src/eu/instruction.rs` — document or prefix unused variants
- [ ] **SL-013:** Extract shared lowering logic from `eu/` and `rdna/` backends into a trait
- [ ] **SL-009/SL-014/SL-023:** Convert all TODO comments (16 total: 6 Go, 9 Rust, 1 deprecated) to GitHub Issues
- [ ] **SL-015:** Add `t.Helper()` to 7 test functions in `internal/x11/shm/shm_test.go`
- [ ] **SL-017:** Add assertions to `TestConcurrentAccess` in `internal/ui/scale/scale_test.go:99`
- [ ] **SL-021:** Add assertions to `TestPackVertices` in `internal/render/backend/backend_test.go:162`
- [ ] **SL-018:** Use `_` receiver for 80 unused receiver methods (interface compliance stubs)
- [ ] **S-010:** Run `go mod tidy` to clean stale `golang.org/x/sys` v0.20.0 from `go.sum`

---

## Readiness Verdicts

| Gates Passing | Security | Slop Debt | Overall Verdict |
|---------------|----------|-----------|-----------------|
| 4.5/7 | CONDITIONAL PASS | HIGH (23 findings, 4 HIGH) | **NOT READY** |

### Path to PRODUCTION READY
1. Fix S-001 (panic in library), S-002 (discarded errors), S-003 (Rust unwraps) → Security becomes **PASS**
2. Address Function Length gate (refactor top 8 internal functions) → Gates become 5.5/7
3. Address Naming gate (2 package violations) → Gates become 6.5/7
4. Fix SL-001 through SL-004 (test assertions + magic numbers) → Slop becomes **MODERATE**
5. Result: 6.5/7 gates, PASS security, MODERATE slop → **CONDITIONALLY READY**

To reach full **PRODUCTION READY**: additionally fix remaining function length violations and achieve all 7 gates passing.

---

## Appendix A: Metrics Source Data

Generated by `go-stats-generator` (Go analysis) and manual Rust audit on 2026-03-09.

| Metric | Value |
|--------|-------|
| Go Lines of Code | 12,215 |
| Go Functions/Methods | 1,357 |
| Go Packages | 65 |
| Go Test Files | 61 |
| Rust Lines of Code | ~14,400 |
| Rust Source Files | 32 |
| Go External Dependencies | 1 (`golang.org/x/sys`) |
| Rust External Dependencies | 2 (`nix`, `naga`) |
| Average Function Length | 11.0 lines |
| Average Cyclomatic Complexity | 3.3 |
| Documentation Coverage | 90.7% |
| Duplication Ratio | 3.71% |
| Clone Pairs | 64 |
| Circular Dependencies | 0 |
| Naming Score | 0.99 |

## Appendix B: Anti-Pattern Distribution

| Anti-Pattern | Count | Severity | Notes |
|--------------|-------|----------|-------|
| Bare error return | 177 | Low | Functions return errors without wrapping context |
| Unused receiver | 80 | Low | Interface compliance stubs |
| Memory allocation | 19 | Info | Standard allocation patterns |
| Resource leak | 15 | Low | Mostly in demo/test code |
| Giant switch | 4 | Low | Keycode mapping tables — acceptable |
| Goroutine leak | 3 | Medium | Demo code only |
| Any/interface{} overuse | 2 | Low | Event data and display list — justified |
| Log fatal in library | 1 | Medium | Investigate and convert to error return |
| Panic in library | 1 | High | S-001 (ring.go:152) |
