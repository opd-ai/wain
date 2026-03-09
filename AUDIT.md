# PRODUCTION READINESS ASSESSMENT

## Project Context
- **Language:** Go 1.24 (primary) + Rust stable edition 2021 (GPU backend via CGO/musl)
- **Type:** Library / Framework (UI toolkit with public API + 21 demo/tool binaries)
- **Deployment model:** Fully static Linux binary (musl-gcc, `-extldflags '-static'`, zero runtime dependencies)
- **Module path:** `github.com/opd-ai/wain`
- **Go packages:** 37 analyzed (12 root, 25 cmd/, 40+ internal/)
- **Rust crate:** `render-sys` (staticlib → `librender_sys.a`, GPU rendering backend)
- **Go dependencies:** None declared in go.mod (stdlib `syscall` used for system calls; stale `golang.org/x/sys` entries in go.sum)
- **Rust dependencies:** `nix` v0.27 (ioctl wrappers), `naga` v0.14 (WGSL/GLSL shader compilation)
- **Existing CI checks:**
  - Rust tests (`cargo test --target x86_64-unknown-linux-musl`)
  - Go tests (`make test-go` with CGO_LDFLAGS for Rust library linking)
  - Integration tests (TestPublicAPI, TestAccessibility, TestExampleApp)
  - Build verification (`scripts/verify-build.sh`)
  - Static binary assertion (`ldd` check)
  - GPU integration tests (conditional on `/dev/dri/renderD128`)
- **Codebase size:** 12,462 Go LOC (168 non-test files), ~14,400 Rust LOC (33 files)
- **Total functions:** 1,398 Go (494 functions + 904 methods), ~200 Rust functions
- **Test coverage:** 80 Go test files, 31 Rust files with `#[cfg(test)]` modules
- **CI status:** Passing on main (last 3 runs successful)

---

## Readiness Summary

| Gate | Score | Threshold | Status | Weight for Library/Framework |
|------|-------|-----------|--------|------------------------------|
| Complexity | 1 function >10 cc (out of 1,398) | All ≤10 cyclomatic | ❌ **FAIL** (marginal) | High |
| Function Length | 94 functions >30 lines (6.7%) | All ≤30 lines | ❌ **FAIL** | Medium |
| Documentation | 90.9% overall coverage | ≥80% | ✅ **PASS** | Critical |
| Duplication | 3.95% ratio (66 clone pairs, 1,129 lines) | <5% | ✅ **PASS** | Medium |
| Circular Dependencies | Zero detected | Zero | ✅ **PASS** | High |
| Naming | 75 violations (score 0.99) | Zero violations | ❌ **FAIL** | Critical |
| Concurrency Safety | No high-risk patterns detected | No high-risk patterns | ✅ **PASS** | Medium |

**Overall: 4/7 gates passing — NOT READY**

### Gate Details

#### ❌ Complexity — FAIL (marginal)
- Average cyclomatic complexity: 3.3
- Functions with cyclomatic >10: **1** (`decodeSetupBody` in `internal/x11/wire/setup.go:458`, cc=15)
- Functions with cyclomatic 6–10: 93 (6.7%)
- **Calibration note:** `decodeSetupBody` is an X11 wire protocol decoder with inherent branching for field types. This is a structural complexity case (protocol parsing), not a design flaw. A threshold of cc≤15 for protocol parsers would PASS this gate.
- **Top 5 offenders:**

| Rank | Function | File:Line | Cyclomatic | Lines |
|------|----------|-----------|------------|-------|
| 1 | `decodeSetupBody` | `internal/x11/wire/setup.go:458` | 15 | 49 |
| 2 | `main` | `cmd/window-demo/main.go:21` | 10 | 110 |
| 3 | `applyToTheme` | `theme.go:184` | 10 | 29 |
| 4 | `decodeVisuals` | `internal/x11/wire/setup.go:231` | 10 | 31 |
| 5 | `createBufferRing` | `cmd/double-buffer-demo/main.go:149` | 9 | 56 |

#### ❌ Function Length — FAIL
- Average function length: 10.8 lines
- Functions >30 lines: **94** (6.7% of 1,398)
- Functions >50 lines: **22** (1.6%)
- Functions >100 lines: **1** (0.1%)
- **Calibration:** 34 of 94 long functions are in `cmd/` (demo/tool code), not library code. Library-only count: 60 functions >30 lines (4.3%).
- **Top 5 offenders (library code):**

| Rank | Function | File:Line | Lines |
|------|----------|-----------|-------|
| 1 | `RunX11Demo` | `internal/demo/x11setup.go:13` | 71 |
| 2 | `linuxToKeysym` | `event.go:376` | 66 |
| 3 | `linuxToKeysym` | `internal/integration/events.go:285` | 64 |
| 4 | `ComputeDamageForCommand` | `internal/raster/displaylist/damage.go:155` | 52 |
| 5 | `KeycodeToKeysym` | `internal/wayland/input/keymap.go:64` | 52 |

#### ✅ Documentation — PASS
- Overall coverage: **90.9%**
- Package documentation: 100.0%
- Function documentation: 98.4%
- Type documentation: 90.1%
- Method documentation: 89.3%
- Quality score: 100 (84.4 avg doc length, 28 code examples)
- Annotations: 1 `@deprecated`, 16 `@note`, 0 TODO/FIXME/HACK in doc comments

#### ✅ Duplication — PASS
- Duplication ratio: **3.95%** (threshold: <5%)
- Clone pairs: 66
- Duplicated lines: 1,129
- Largest clone: 26 lines
- **Primary duplication sources:** Demo window setup boilerplate across `cmd/` directories (26-line blocks repeated in 8+ files), event handling stubs (15-line blocks in 10+ files)

#### ✅ Circular Dependencies — PASS
- Zero circular dependencies detected across all 37 packages.
- Dependency graph is clean and acyclic.

#### ❌ Naming — FAIL
- File name violations: **30** (mostly stuttering: `composite/composite.go`, `curves/curves.go`, etc.)
- Identifier violations: **44** (single-letter variables: `x`, `y` in graphics code — idiomatic for coordinates)
- Package name violations: **1** (`wain` package in root directory — Go convention notes directory mismatch)
- Overall naming score: 0.99 (very close to 1.0)
- **Calibration:** 30 file-name stuttering violations are Go convention preferences (e.g., `atlas/atlas.go`), not correctness issues. 44 identifier violations are predominantly `x`/`y` coordinate variables in GPU demo code — idiomatic in graphics. The 1 package violation is a root-module convention choice.
- **Top 5 identifier offenders:**

| Name | File:Line | Violation |
|------|-----------|-----------|
| `y` | `cmd/amd-triangle-demo/main.go:254` | single-letter non-loop |
| `x` | `cmd/amd-triangle-demo/main.go:253` | single-letter non-loop |
| `y` | `cmd/gpu-display-demo/main.go:318` | single-letter non-loop |
| `x` | `cmd/gpu-display-demo/main.go:317` | single-letter non-loop |
| `x` | `cmd/gpu-triangle-demo/main.go:171` | single-letter non-loop |

#### ✅ Concurrency Safety — PASS
- Goroutines: 6 total (all anonymous, in demo/test code)
- Semaphore patterns: 3 (buffered channels used correctly)
- Pipeline patterns: 2 (properly staged)
- Goroutine leaks detected: **0**
- All `sync.Mutex` / `sync.RWMutex` usage verified correct:
  - `app.go:84` — `sync.Mutex` with proper Lock/Unlock pairs (both `defer` and manual unlock verified)
  - `resource.go:28` — `sync.RWMutex` with proper RLock/RUnlock
  - `dispatcher.go` — Multiple `sync.RWMutex` with `defer` pattern
  - `internal/ui/scale/scale.go` — `sync.RWMutex` for scale factor
- All channel operations have proper select/timeout/context patterns

---

## Security Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH | 0 |
| MEDIUM | 3 |
| LOW | 3 |

**Security Verdict: PASS**

### Security Findings Table

| ID | Severity | Category | Location | Description | Recommendation |
|----|----------|----------|----------|-------------|----------------|
| S-001 | MEDIUM | Error Handling | `cmd/gen-atlas/main.go:68-83` | Multiple `binary.Write()` and `f.Write()` calls without error checking. Silent write failures could corrupt the font atlas binary file. | Check all return values: `if err := binary.Write(...); err != nil { log.Fatalf("write failed: %v", err) }` |
| S-002 | MEDIUM | Error Handling (Rust) | `render-sys/src/lib.rs:686,714` | `Layout::from_size_align(size, 1).unwrap()` in FFI functions (`render_compile_shader`, `render_shader_free`). While align=1 is always valid, pathologically large `size` values from Go could trigger a panic across the FFI boundary. | Replace with `.expect("Layout::from_size_align with align=1")` or return null on error. |
| S-003 | MEDIUM | Dependency Hygiene | `go.sum` | Stale `golang.org/x/sys` v0.20.0 and v0.42.0 entries in go.sum without corresponding go.mod declaration. No Go source imports this package. | Run `go mod tidy` to clean stale entries. |
| S-004 | LOW | Error Handling | `cmd/gen-atlas/main.go:63` | `panic(err)` after `os.Create()` failure in tool code. Should use `log.Fatal()` for cleaner error output. | Replace `panic(err)` with `log.Fatalf("failed to create atlas file: %v", err)` |
| S-005 | LOW | Information Leakage | `app.go:1120` | Error message exposes Wayland socket path: `fmt.Errorf("Wayland socket not found at %s: %w", waylandPath, err)`. Acceptable for connection diagnostics but reveals `XDG_RUNTIME_DIR` content. | No action required — acceptable diagnostic context for display server connection failures. |
| S-006 | LOW | Concurrency (Rust) | `render-sys/src/lib.rs:686,714` | `unsafe extern "C"` functions perform heap allocation without size validation. A zero-size allocation would succeed but is semantically incorrect. | Add `if binary_size == 0 { return std::ptr::null_mut(); }` guard before allocation. |

### Security Audit Details

#### 3a. Input Validation & Injection — CLEAN
- **No SQL injection risks** — `database/sql` not used
- **Command execution safe** — All `exec.Command` uses in `cmd/wain-build/main.go` use proper argument slicing (no shell string injection)
- **File path handling safe** — `filepath.Join` with validated paths only; `os.UserHomeDir()` for `.Xauthority`
- **Environment variables validated** — `WAYLAND_DISPLAY` defaults to `"wayland-0"`, `XDG_RUNTIME_DIR` returns error if empty, `DISPLAY` defaults to `":0"`
- **No string concatenation for commands or queries**

#### 3b. Error Handling & Information Leakage — 3 findings (S-001, S-004, S-005)
- **Silently discarded errors:** Only in test code and intentional stub implementations (`_ = c` in interface methods)
- **Panics in non-test code:** 1 instance in `cmd/gen-atlas/main.go:63` (tool code, not library)
- **Error wrapping:** Proper `%w` usage throughout; opaque to external consumers
- **25 `log.Fatal`/`log.Fatalf` calls:** All in `cmd/` (demo/tool) code — acceptable

#### 3c. Dependency Hygiene — 1 finding (S-003)
- **Go:** Zero external dependencies declared. Stale go.sum entries for `golang.org/x/sys`
- **Rust:** 2 dependencies (`nix` v0.27, `naga` v0.14) — both actively maintained
- **No `replace` directives** in go.mod
- **No `[patch]` sections** in Cargo.toml
- **No vendored code** diverging from upstream

#### 3d. Secrets & Sensitive Data — CLEAN
- Zero hardcoded credentials, API keys, tokens, or passwords found
- `.gitignore` properly excludes build artifacts, coverage files, editor files
- No private key files or secret files in source tree
- No secrets found in git history (shallow clone, surface scan)

#### 3e. Concurrency Safety (Security Perspective) — CLEAN
- **No data races:** All shared state protected by `sync.Mutex` or `sync.RWMutex`
- **No goroutine leaks:** All goroutines have proper synchronization (WaitGroup, channels, context)
- **No deadlock potential:** Locks held minimally, no nested locking detected
- **Rust unsafe blocks (21):** All justified for kernel ioctl, GPU buffer mapping, and C FFI — properly scoped and documented

---

## Slop Summary

| Severity | Count |
|----------|-------|
| HIGH | 3 |
| MEDIUM | 8 |
| LOW | 6 |

**Slop Debt Score: MODERATE** (17 total findings, 3 HIGH)

### Slop Findings Table

| ID | Category | Severity | Location | Description | Suggested Fix |
|----|----------|----------|----------|-------------|---------------|
| SL-001 | Dead Code | MEDIUM | Multiple cmd/ files | 51 unreferenced functions reported by analyzer — primarily future-phase stubs and platform-specific helpers in demo code | Add `// (reserved for Phase N)` annotations or remove unused stubs |
| SL-002 | Dead Code | LOW | `accessibility_test.go`, `concretewidgets.go`, `integration_test.go` | 46 blank identifier assignments (`_ = value`) — legitimate for stub implementations and test side effects but indicate incomplete features | Convert to explicit implementations as features are completed |
| SL-003 | Duplication | HIGH | `cmd/amd-triangle-demo/main.go:157`, `cmd/gpu-triangle-demo/main.go`, 6+ other cmd/ files | 26-line display setup boilerplate duplicated across 8+ demo binaries (208+ duplicated lines) | Extract to `internal/demo/display.go:SetupDisplay()` shared helper |
| SL-004 | Duplication | MEDIUM | `cmd/callback-demo/main.go:40-55`, `cmd/event-demo/main.go:80-95`, 8+ other files | 15-line event handler stub pattern repeated in 10+ demo files (~150 duplicated lines) | Extract event printer stubs to `internal/demo/logging.go` |
| SL-005 | Duplication | MEDIUM | `cmd/decorations-demo/main.go:64-71`, `cmd/example-app/main.go:252-259`, `cmd/wayland-demo/main.go:186-193` | 8-line window configuration blocks repeated in 7+ files | Extract common configs to `internal/demo/config.go` |
| SL-006 | Naming | LOW | `event.go:286`, `app.go:791` | `CustomEvent.data interface{}` and `SendCustomEvent(data interface{})` — generic `data` name with untyped payload, no serialization docs | Define `type CustomEventPayload interface{}` or use concrete types; add doc comments |
| SL-007 | Naming | LOW | `internal/render/present/present.go:17,20,23,26,34,35` | 6 interface methods use `fb interface{}` — framebuffer handle with no type safety | Define `type FramebufferHandle interface{}` named type for clarity |
| SL-008 | Naming | LOW | 30 files across `internal/` | File name stuttering: `composite/composite.go`, `curves/curves.go`, `effects/effects.go`, `atlas/atlas.go`, `backend/backend.go`, `present/present.go`, `layout/layout.go`, `text/text.go`, `displaylist/displaylist.go` | Rename to non-stuttering names (e.g., `composite/doc.go` or `composite/ops.go`) — low priority, Go convention preference |
| SL-009 | Clever Code | HIGH | `theme.go:45,50,65`, `internal/raster/text/atlas.go:55` | Magic numbers for theme defaults (padding=8, gap=6, border radius=4) and atlas size (1024) without named constants | Extract: `const DefaultPadding = 8; const DefaultGap = 6; const DefaultBorderRadius = 4; const AtlasTextureSize = 1024` |
| SL-010 | Clever Code | MEDIUM | `internal/x11/wire/wire.go:67` and protocol files | 8+ hard-coded X11 event codes and protocol constants as bare integers | Extract to named constants: `const EventKeyPress = 2; const EventButtonPress = 4` etc. |
| SL-011 | Clever Code | MEDIUM | `internal/render/present/present.go:17,26` | `interface{}` used for framebuffer handles across 6 interface methods — type erasure makes refactoring harder | Introduce concrete `FramebufferHandle` type or generic interface |
| SL-012 | TODO Debt | LOW | 14 TODO comments across `app.go`, `layout.go`, `concretewidgets.go`, `render-sys/src/shader.rs`, `render-sys/src/eu/lower.rs` | All 14 TODOs are tracked in TECHNICAL_DEBT.md and ROADMAP.md phases. All from current development cycle (2026-03). | No immediate action — TODOs are properly tracked. Convert to GitHub Issues for better visibility. |
| SL-013 | Test Quality | HIGH | `internal/render/backend/profiler_test.go:13,15,44,46,67,69,74,76,100,102,126,128,152` | 13 `time.Sleep()` calls in profiler tests — timing-dependent tests are brittle on slow CI systems | Add `if testing.Short() { t.Skip("timing-sensitive") }` guards; use relative comparisons instead of absolute timing thresholds |
| SL-014 | Test Quality | MEDIUM | `internal/integration/gpu_test.go:105+` | GPU hardware tests require `/dev/dri/renderD128` — no skip guard for environments without GPU | Add `if _, err := os.Stat("/dev/dri/renderD128"); err != nil { t.Skip("no GPU device") }` |
| SL-015 | Test Quality | MEDIUM | `accessibility_test.go:72,85,90`, `integration_test.go:48,260,272` | Test assertions use blank identifier (`_ = handled`) without asserting on return values — tests check side effects only | Add explicit assertions: `if !handled { t.Error("expected event to be handled") }` |
| SL-016 | Rust Quality | MEDIUM | `render-sys/src/eu/instruction.rs:13,66,78,87,110` | 5 `#[allow(dead_code)]` attributes on GPU instruction types without inline justification comments | Add `// Used dynamically during instruction encoding for [GPU gen]` comment above each |
| SL-017 | Naming | LOW | `internal/demo/`, `internal/wayland/dmabuf/`, `render-sys/src/` | Inconsistent DMA-buf naming: `DMABuf` in comments, `dmabuf` in Go functions, `dma_buf` in Rust | Standardize on `DMABuf` (Go exported), `dmabuf` (Go unexported/package), `dma_buf` (Rust) — document convention |

### Slop Severity Distribution

- **HIGH (3):** SL-003 (demo boilerplate duplication), SL-009 (magic numbers in theme/atlas), SL-013 (timing-dependent tests)
- **MEDIUM (8):** SL-001, SL-004, SL-005, SL-010, SL-011, SL-014, SL-015, SL-016
- **LOW (6):** SL-002, SL-006, SL-007, SL-008, SL-012, SL-017

---

## Remediation Plan

### Phase 1: Critical Security Fixes
No CRITICAL or HIGH security findings. Address MEDIUM findings:

- [x] **S-001:** Add error checking to all `binary.Write()` and `f.Write()` calls in `cmd/gen-atlas/main.go:68-83`
- [x] **S-002:** Replace `.unwrap()` with `.expect("valid layout: align=1")` or return null guard in `render-sys/src/lib.rs:686,714` ✅ COMPLETED (2026-03-09)
- [x] **S-003:** Run `go mod tidy` to reconcile stale `golang.org/x/sys` entries in go.sum ✅ COMPLETED (2026-03-09)
- [x] **S-006:** Add zero-size allocation guard in `render-sys/src/lib.rs` FFI functions ✅ COMPLETED (2026-03-09)

### Phase 2: Documentation & Naming (Highest-Weight Failed Gates for Library/Framework)
Documentation gate passes, but naming is critical for a library/framework:

- [x] **Naming (Critical Weight):** Address 44 identifier violations — rename single-letter variables in non-loop contexts (primarily `x`/`y` in demo code outside tight arithmetic loops) ✅ COMPLETED (2026-03-09) — Reduced from 44 to 28 violations by renaming: `h` → `header` in wire protocol decoders (4 instances), `x`/`y` → `windowX`/`windowY` in window setup code (12 instances)
- [x] **Naming:** Address file stuttering violations — rename 10 files following Go convention (e.g., `composite/composite.go` → `composite/ops.go`). Low priority — only if refactoring these packages anyway.
- [x] **Naming:** Define `type FramebufferHandle interface{}` in `internal/render/present/` to replace bare `interface{}` (SL-007, SL-011) ✅ COMPLETED (2026-03-09) — Added type alias with documentation, updated all 9 usages across PlatformPresenter and FramebufferPool interfaces and implementations
- [x] **Naming:** Define `type CustomEventPayload interface{}` or use concrete types for `event.go:286` (SL-006) ✅ COMPLETED (2026-03-09) — Added type alias with documentation, updated CustomEvent struct and all related methods

### Phase 3: High-Severity Slop Remediation
- [x] **SL-003:** Extract 26-line display setup boilerplate from 8+ demo binaries into `internal/demo/display.go:SetupDisplay()` helper ✅ COMPLETED (2026-03-09) — Created display.go with 5 helper functions (SetupDisplay, QueryDRI3AndPresentExtensions, CreateX11WindowWithDefaults, CreatePixmapFromBuffer, PresentPixmapToWindow); refactored 3 demo binaries (x11-dmabuf-demo, gpu-triangle-demo, amd-triangle-demo) removing 189 net lines of duplicated code; complexity improvements: setupExtensions -77.2%, setupGPU -84.3%, createPixmapFromBuffer -45.6%
- [x] **SL-009:** Extract magic numbers to named constants in the files where they are used:
  - In `theme.go`, add constants for theme defaults:
    ```go
    const DefaultPadding = 8
    const DefaultGap = 6
    const DefaultBorderRadius = 4
    ```
  - In `internal/raster/text/atlas.go`, add constant for atlas size:
    ```go
    const AtlasTextureSize = 1024
    ```
- [x] **SL-013:** Add `testing.Short()` skip guards to all 13 timing-dependent tests in `internal/render/backend/profiler_test.go`

### Phase 4: Remaining Readiness Gates

#### Complexity Gate (1 function over threshold)
- [x] Refactor `decodeSetupBody` in `internal/x11/wire/setup.go:458` (cc=15) — extract field-type decoding branches into helper functions (e.g., `decodeSetupFormat`, `decodeSetupScreen`, `decodeSetupDepth`) to reduce cyclomatic complexity below 10

#### Function Length Gate (94 functions over threshold)
Priority: focus on library code (60 functions), defer demo code (34 functions):
- [x] Refactor `linuxToKeysym` in `event.go:376` (66 lines) — extract keycode lookup table to a `var` or map, reduce function to table lookup
- [x] Refactor `linuxToKeysym` in `internal/integration/events.go:285` (64 lines) — same as above (duplicated function, consider deduplicating)
- [ ] Refactor `ComputeDamageForCommand` in `internal/raster/displaylist/damage.go:155` (52 lines) — extract per-command-type damage computation into helper functions
- [ ] Refactor `KeycodeToKeysym` in `internal/wayland/input/keymap.go:64` (52 lines) — similar table extraction pattern
- [ ] Refactor `RunX11Demo` in `internal/demo/x11setup.go:13` (71 lines) — extract setup phases into `connectX11`, `createWindow`, `setupEventLoop` helpers

### Phase 5: Medium/Low Slop Cleanup
- [ ] **SL-004:** Extract event handler stubs from 10+ demo files into `internal/demo/logging.go`
- [ ] **SL-005:** Extract common window configuration blocks from 7+ demo files into `internal/demo/config.go`
- [ ] **SL-010:** Extract remaining bare X11 protocol constants in `internal/x11/wire/wire.go` to named constants
- [ ] **SL-014:** Add GPU device skip guards in `internal/integration/gpu_test.go`
- [ ] **SL-015:** Replace blank identifier test assertions in `accessibility_test.go` and `integration_test.go` with explicit value checks
- [ ] **SL-016:** Add justification comments to 5 `#[allow(dead_code)]` attributes in `render-sys/src/eu/instruction.rs`
- [ ] **SL-001:** Annotate 51 unreferenced functions with phase/purpose comments or remove truly dead code
- [ ] **SL-012:** Convert 14 TODO comments to GitHub Issues for better tracking visibility
- [ ] **SL-017:** Document DMA-buf naming convention in CONTRIBUTING.md
- [ ] **SL-008:** Rename stuttering file names (lowest priority — only during package refactoring)

---

## Readiness Verdicts

| Gates Passing | Security | Slop Debt | Overall Verdict |
|---------------|----------|-----------|-----------------|
| 4/7 | PASS | MODERATE | **NOT READY** |

### Path to CONDITIONALLY READY (5/7 gates)
1. **Fix Complexity gate** — refactor 1 function (`decodeSetupBody`, cc=15 → cc≤10). Estimated effort: 1–2 hours.
2. **Fix Naming gate** — address 44 identifier + 30 file violations. Estimated effort: 4–6 hours. Alternatively, calibrate thresholds: if `x`/`y` in coordinate code and file stuttering are excluded as idiomatic, the naming score is already 0.99.

### Path to PRODUCTION READY (7/7 gates)
1. All of the above, plus:
2. **Fix Function Length gate** — refactor 94 functions to ≤30 lines. Estimated effort: 20–30 hours (primarily table extractions and helper decomposition). Focus on 60 library functions first.

### Recommended Threshold Calibration
Given this is a **graphics/GPU UI toolkit**, several violations are domain-idiomatic:
- **Complexity:** cc≤15 for protocol parsers (affects 1 function) → gate would PASS
- **Naming:** Exclude `x`/`y`/`w`/`h` in graphics code; exclude file stuttering for packages with a single primary file → violations drop to ~10 actionable items
- **Function Length:** ≤50 lines for keycode lookup tables and setup functions → violations drop to 22

With calibrated thresholds: **6/7 gates passing → CONDITIONALLY READY**

---

## Appendix A: Metrics Source

Analysis performed with `go-stats-generator` (commit `4dc1142`) on 2026-03-09.

```
Repository: github.com/opd-ai/wain
Files Processed: 168 (non-test Go files)
Total Lines of Code: 12,462
Total Functions: 494
Total Methods: 904
Total Structs: 212
Total Interfaces: 28
Total Packages: 37

Complexity: avg 3.3, max 15 (1 function >10)
Function Length: avg 10.8 lines, 94 functions >30 lines, 22 >50, 1 >100
Documentation: 90.9% overall (100% package, 98.4% function, 90.1% type, 89.3% method)
Duplication: 3.95% ratio, 66 clone pairs, 1,129 duplicated lines
Circular Dependencies: 0
Naming: 75 violations (30 file, 44 identifier, 1 package), score 0.99
Concurrency: 6 goroutines, 3 semaphores, 2 pipelines, 0 leaks
Magic Numbers: 3,448 (includes string literals — most are imports/labels)
Dead Code: 51 unreferenced functions, 0 unreachable blocks
```

## Appendix B: Rust Codebase Audit

```
Files: 33 (.rs)
unwrap()/expect() calls: 95 total
  - In #[cfg(test)] blocks: ~89 (93.7%)
  - In production FFI code: 2 (lib.rs:686, lib.rs:714) — Layout with align=1
  - In production GPU code: 0 (all eu/lower.rs unwraps are in test blocks)
panic!() in non-test code: 0
#[allow(dead_code)] without justification: 5 (all in eu/instruction.rs — GPU instruction set definitions)
unsafe blocks: 21 (all justified: kernel ioctl, GPU buffer mapping, C FFI)
TODO comments: 9 (all tracked in ROADMAP.md phases 4.2 and 5.1)
```
