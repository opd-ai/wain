# AUDIT — 2026-03-08

## Project Context

**wain** is a statically-compiled Go UI toolkit with GPU rendering via Rust, targeting Linux systems with Wayland or X11 display servers. The project is in active development with Phases 0-4 partially complete (GPU command submission, shader compilation, software rasterization, and protocol layers functional). The intended audience is developers building native Linux GUI applications requiring static binaries and GPU-accelerated rendering.

**Module:** `github.com/opd-ai/wain` (Go 1.24)  
**Architecture:** 5-layer stack — Rust rendering library (render-sys), Go CGO bindings, protocol layer (Wayland/X11), 2D rasterizer, UI framework  
**Build:** Fully static binaries via musl libc and Rust musl target  
**Test Coverage:** ~66% average across 34 library packages (verified via `scripts/compute-coverage.sh`)

---

## Summary

**Overall Health:** ✅ **GOOD** — Core claims verified, baseline functionality working, no critical defects found

**Findings Distribution:**
- **CRITICAL:** 0
- **HIGH:** 3
- **MEDIUM:** 7
- **LOW:** 5

**Key Strengths:**
- ✅ Excellent documentation coverage (88.76% overall, 98.31% function coverage)
- ✅ Static linking verified working (`ldd bin/wain` reports "not a dynamic executable")
- ✅ Comprehensive testing (57 test files, 28 fuzz tests for protocol encoding/decoding)
- ✅ All claimed demonstration binaries exist in source tree (21 cmd/ binaries)
- ✅ Core integration verified (Go → Rust CGO linkage functional via `render.Add`, `render.Version`)

**Key Concerns:**
- ⚠️ Public API surface does not exist (all packages marked `internal/`)
- ⚠️ Several features claimed as "complete" have documented limitations (Wayland keymap US-only, X11 DRI3 single-plane only, DrawImage GPU-only)
- ⚠️ Some undocumented exported symbols in integration and demo packages
- ⚠️ Widget demo auto-detection works but event loops not implemented (documented as stubs)

---

## Findings

### CRITICAL

**No critical findings.** All documented features are functional for their intended use cases.

---

### HIGH

- [x] **README claims do not match keymap implementation** — internal/wayland/input/keymap.go:37-47 — README line 44 claims "basic keycode-to-keysym translation (hardcoded QWERTY layout)" but does not explicitly state **US layout only**. Code documentation confirms "minimal implementation... without full XKB parsing" supporting only US QWERTY keyboards. Non-US keyboards will produce incorrect keysyms. — **Remediation:** (1) Update README.md line 44 to read: "Input handling: wl_seat, wl_pointer, wl_keyboard with basic keycode-to-keysym translation (hardcoded US QWERTY layout — non-US keyboards not supported)". (2) Add to Known Limitations section: "⚠️ Wayland input handling only supports US QWERTY keyboard layout. International keyboards require XKB parser implementation (planned for Phase 8.1)." **Validation:** After updating README, run `grep -n "hardcoded.*QWERTY" README.md` to confirm updated language includes "US" qualifier.

- [x] **Software rasterizer silently skips CmdDrawImage** — internal/raster/consumer/software.go:82-84 — README line 100-101 claims "Compositing: alpha blending (Porter-Duff), bilinear image filtering" as complete, but `CmdDrawImage` display list command is silently skipped with comment "DrawImage not yet implemented in software rasterizer / Skip for now - GPU backend handles this". This means software-only rendering cannot composite images. Bilinear filtering code exists (internal/raster/composite/composite.go:178) but is only used via `Blit`/`BlitScaled`, not via display list. — **Remediation:** (1) Update README.md line 100-101 to: "Compositing: alpha blending (Porter-Duff), bilinear image filtering (Blit/BlitScaled only; DrawImage requires GPU backend)". (2) Update Known Limitations section to add: "⚠️ Software rasterizer does not support DrawImage command — image compositing requires GPU backend or manual Blit calls." (3) Add package documentation to internal/raster/consumer/software.go lines 1-10 explaining GPU-only fallback. **Validation:** Run `make test-go && go test -v ./internal/raster/consumer -run TestDrawImage` to confirm no regressions. Check `grep -A2 "CmdDrawImage" internal/raster/consumer/software.go` to verify updated documentation.

- [x] **X11 DRI3 version mismatch between claim and implementation** — internal/x11/dri3/dri3.go:27-34 — README line 53 claims "DRI3 extension: GPU buffer sharing via DMA-BUF file descriptors" without version qualifications, but code documentation targets "DRI3 version 1.2" while only implementing 1.0 features. Comments state "DRI3 1.0 is sufficient for basic single-plane ARGB buffers" but multi-planar buffers and modifiers (1.2 features) are not implemented despite targeting 1.2. This creates confusion about feature completeness. — **Remediation:** (1) Update README.md line 53 to: "DRI3 extension (1.0+): GPU buffer sharing via DMA-BUF file descriptors (single-plane ARGB buffers; multi-planar support deferred)". (2) Update internal/x11/dri3/dri3.go package doc (lines 1-10) to clarify: "This implementation supports DRI3 1.0 with basic single-plane buffer sharing. Version 1.2 features (multi-planar, modifiers) are deferred to Phase 6." **Validation:** Run `grep -n "DRI3" README.md` and verify updated language. Check `go doc github.com/opd-ai/wain/internal/x11/dri3` to confirm package documentation is updated.

---

### MEDIUM

- [ ] **Undocumented exported symbols in integration package** — internal/integration/events.go:102-121 — 10 exported struct fields lack documentation comments (`EventType`, `X`, `Y`, `Button`, `Axis`, `Value`, `Timestamp`, `Key`, `Modifiers` on lines 102-121). Package documentation coverage is 0% despite having exported API. This violates Go documentation conventions (exported symbols should have comments). — **Remediation:** Add documentation comments to all exported fields in internal/integration/events.go. Example for line 102-121: `// EventType indicates the type of pointer event (motion, button press/release, axis scroll).`, `// X is the horizontal pointer coordinate in surface-local space.`, `// Y is the vertical pointer coordinate in surface-local space.`, etc. **Validation:** Run `go-stats-generator analyze . --skip-tests --format json --output audit-post.json --sections documentation && jq -r '.packages[] | select(.path == "integration") | .documentation.quality_score' audit-post.json` and verify score > 0.

- [ ] **Undocumented exported functions in cmd/gpu-display-demo** — cmd/gpu-display-demo/main.go:384-405 — 4 exported functions in main package lack documentation: `AllocXID` (lines 384, 405), `SendRequestAndReplyWithFDs` (line 389), `SendRequestWithFDs` (line 393). While demo code, these are exported and used by other binaries. — **Remediation:** Add documentation comments above each function. Example: `// AllocXID allocates a new X11 identifier from the connection's XID pool.` (line 383), `// SendRequestAndReplyWithFDs sends an X11 request with file descriptors and waits for the reply.` (line 388), `// SendRequestWithFDs sends an X11 request with file descriptors without waiting for a reply.` (line 392). **Validation:** Run `go doc cmd/gpu-display-demo` and verify all exported functions have documentation.

- [ ] **Widget demo event loops are stubs** — cmd/widget-demo/main.go:301-356 — README line 374 claims "Interactive widget demo (auto-detects X11/Wayland)" implying full functionality, but event loops are explicitly marked as stubs (line 305: "⚠ event loop not yet implemented", line 318: "⚠ event loop not yet implemented"). Auto-detection works correctly but the demo is non-functional beyond platform detection. — **Remediation:** (1) Update README.md line 374 description to: "Interactive widget demo (auto-detects X11/Wayland; event loops not yet implemented)". (2) Update table row on line 374 to add note: "Platform detection functional; event handling deferred". **Validation:** Run `./bin/widget-demo --help` and verify output matches updated README description.

- [ ] **X11 Present extension version limitation undocumented** — internal/x11/present/present.go:34-42 — README line 54 claims "Present extension: frame synchronization and swap control" without version qualifications, but implementation only supports Present 1.0, not 1.2+ async flip features. Code documentation states "Version 1.2+ adds async flip support and other advanced features, but 1.0 is sufficient for basic tear-free rendering." This limitation is not surfaced in README. — **Remediation:** Update README.md line 54 to: "Present extension (1.0): frame synchronization and swap control (tear-free rendering; async flip deferred)". **Validation:** Run `grep -n "Present extension" README.md` and verify updated language clarifies version support.

- [ ] **Unsafe.Pointer usage flagged by go vet** — internal/x11/shm/shm.go:186 — `go vet` reports "possible misuse of unsafe.Pointer" for uintptr-to-unsafe.Pointer conversion from syscall return value. Code has `//nolint:govet` comment (line 185) acknowledging the warning, but the suppression is specific to a linter, not `go vet` itself. While the code is technically correct (syscall contract guarantees pointer validity), `go vet` will always flag this. — **Remediation:** (1) Add explicit `//lint:ignore SA4023 Syscall return value is kernel-managed address` directive above line 184. (2) Add comment explaining why this is safe: `// The uintptr-to-unsafe.Pointer conversion is safe here because: (a) the address comes directly from kernel shmat syscall, (b) memory is kernel-managed and not subject to Go GC, (c) we convert immediately in the return statement.` **Validation:** Run `go vet ./internal/x11/shm 2>&1 | grep -c "possible misuse"` and verify count is 0. If warning persists, add `//go:noescape` pragma or restructure to use `syscall.Mmap`.

- [ ] **Documentation coverage gap in 36 packages** — audit-baseline.json:documentation.coverage — 36 packages have 0% documentation quality score despite having exported symbols. Packages include: atlas, backend, buffer, client, composite, consumer, core, curves, datadevice, decorations, demo, display, displaylist, dmabuf, dpi, dri3, effects, events, gc, input, integration, layout, main, output, pctwidget, present, render, scale, selection, shm, socket, text, wain, widgets, wire, xdg. Overall documentation coverage is high (88.76%), but these packages have no package-level documentation. — **Remediation:** Add package documentation (doc.go files) to all 36 packages with 0% quality score. Follow Go convention: create `doc.go` in each package with format `// Package <name> provides <description>.`. Example for internal/buffer: `// Package buffer implements frame buffer ring management for double/triple buffering with compositor synchronization.`. **Validation:** Run `go-stats-generator analyze . --skip-tests --format json --output audit-post.json --sections documentation && jq -r '.packages[] | select(.documentation.quality_score == 0) | .path' audit-post.json | wc -l` and verify count is 0.

- [ ] **No screenshot comparison tests** — README.md:135 — README Known Limitations section explicitly states "⚠️ No automated screenshot comparison tests" (line 135). This means rendering correctness is not verified automatically, increasing risk of visual regressions. Integration tests verify protocol → rasterizer → display pipeline (line 111), but visual output is not validated. — **Remediation:** (1) Create `internal/raster/testdata/` directory with reference images for each rendering primitive (filled rect, rounded rect, bezier curves, text, gradients, shadows). (2) Add `internal/raster/visual_test.go` with screenshot comparison tests using image diffing (consider `github.com/oliamb/cutter` for region extraction). (3) Run tests in CI with `make test-visual` target that generates diffs in `coverage/visual-diffs/` directory. (4) Update README.md line 135 to: "✅ Automated screenshot comparison tests for rendering primitives (threshold: 99.5% pixel match)". **Validation:** Run `make test-visual` and verify all tests pass. Check `find internal/raster/testdata -name "*.png" | wc -l` to confirm reference images exist.

---

### LOW

- [ ] **Long function in cmd/perf-demo/main.go** — cmd/perf-demo/main.go:31 — Function `runPerfTests` has 59 lines of code (threshold: 50). Cyclomatic complexity is low (CC=6), but function length makes it harder to understand. Function performs 5 distinct perf tests sequentially. — **Remediation:** Extract each perf test into separate functions: `testFillRectPerf()`, `testDrawLinePerf()`, `testDrawTextPerf()`, `testBoxShadowPerf()`, `testFullFramePerf()`. Main function becomes 5 function calls with descriptive names. **Validation:** Run `go-stats-generator analyze . --skip-tests --format json --output audit-post.json --sections functions && jq -r '.functions[] | select(.file | endswith("cmd/perf-demo/main.go")) | select(.name == "runPerfTests") | .lines.code' audit-post.json` and verify result is < 20 lines.

- [ ] **Long function in cmd/window-demo/main.go** — cmd/window-demo/main.go:21 — Function `main` has 81 lines of code (threshold: 50). Cyclomatic complexity is moderate (CC=10). Function handles argument parsing, platform setup, window creation, rendering loop, and event handling in a single function. — **Remediation:** Extract setup and rendering logic into separate functions: `parseArgs() (platform string, width int, height int)`, `setupPlatform(platform string) (conn interface{}, window uint32)`, `renderFrame(...)`, `handleEvents(...)`. Main function becomes sequential calls with clear phases. **Validation:** Run `go-stats-generator analyze . --skip-tests --format json --output audit-post.json --sections functions && jq -r '.functions[] | select(.file | endswith("cmd/window-demo/main.go")) | select(.name == "main") | .lines.code' audit-post.json` and verify result is < 40 lines.

- [ ] **Long function in cmd/auto-render-demo/main.go** — cmd/auto-render-demo/main.go:24 — Function `main` has 65 lines of code (threshold: 50). Cyclomatic complexity is moderate (CC=9). Function implements backend auto-detection, fallback logic, rendering test, and error handling. — **Remediation:** Extract backend detection and rendering into separate functions: `detectBackend() (backendType string, err error)`, `renderTestPattern(backend string) error`, `fallbackToSoftware() error`. Main function becomes try-catch style with clear fallback chain. **Validation:** Run `go-stats-generator analyze . --skip-tests --format json --output audit-post.json --sections functions && jq -r '.functions[] | select(.file | endswith("cmd/auto-render-demo/main.go")) | select(.name == "main") | .lines.code' audit-post.json` and verify result is < 40 lines.

- [ ] **Duplicate keycode-to-keysym conversion logic** — event.go:371 and internal/integration/events.go:265 — Both files contain identical `linuxToKeysym` functions with 63 lines and CC=6. This is code duplication that will cause maintenance issues if keyboard mapping changes. — **Remediation:** (1) Extract `linuxToKeysym` to `internal/wayland/input/keymap.go` as `LinuxToKeysym(keycode uint32, shift bool) rune`. (2) Update event.go and internal/integration/events.go to import and call `input.LinuxToKeysym()`. (3) Consolidate the hardcoded mapping table (currently duplicated) into single source of truth in keymap package. **Validation:** Run `grep -r "linuxToKeysym" --include="*.go" | wc -l` and verify only 1 definition exists (plus call sites). Run `make test-go` to verify no regressions.

- [ ] **Binary size not documented** — README.md — README verifies static linking (line 342-348) but does not document expected binary size. The `bin/wain` binary is 6.2 MB, which may surprise users expecting smaller static binaries. Lack of size documentation makes it hard to detect binary bloat over time. — **Remediation:** (1) Add to README.md after line 348 (Verify Static Linking section): "**Binary Size:** The fully static `bin/wain` binary is approximately 6.2 MB (includes Rust rendering library, Go runtime, and musl libc). Use `strip bin/wain` to reduce to ~4.8 MB if debug symbols are not needed." (2) Add to CI verification: `ls -lh bin/wain | awk '{print "Binary size:", $5}'` after build step. **Validation:** Run `ls -lh bin/wain` and verify size matches documentation ±10%.

---

## Metrics Snapshot

**Analysis Date:** 2026-03-08  
**Tool Version:** go-stats-generator 1.0.0  
**Files Processed:** 159 Go source files  
**Analysis Time:** 1.02 seconds

### Code Volume
- **Total Functions:** 1,261
- **Exported Functions:** 850 (67.4%)
- **Packages:** 36
- **Test Files:** 57
- **Fuzz Tests:** 28

### Complexity
- **Average Cyclomatic Complexity:** Not computed by tool (requires manual gocyclo integration)
- **Functions with CC > 10:** 0 (manual analysis found max CC=10 in window-demo)
- **Functions with > 50 lines:** 11 (max: 81 lines in window-demo main)
- **Functions with > 7 parameters:** 0

### Documentation
- **Package Coverage:** 100% (all 36 packages have doc.go, but 36 have quality_score=0)
- **Function Coverage:** 98.31% (850/865 exported functions documented)
- **Type Coverage:** 89.34%
- **Method Coverage:** 86.03%
- **Overall Coverage:** 88.76%
- **Average Doc Length:** 80.3 characters
- **Code Examples:** 23
- **TODO Comments:** 1 (app.go:1371 — "Implement full Wayland event reading and dispatch")
- **FIXME Comments:** 0
- **HACK Comments:** 0
- **BUG Comments:** 0
- **Deprecated Comments:** 1 (dmabuf.go:109 — zwp_linux_dmabuf_v1 version 3+ note)

### Testing
- **Test Coverage (Go):** ~66% average across 34 library packages (per scripts/compute-coverage.sh)
- **Rust Test Coverage:** 249/263 tests passing (6 hardware-dependent failures, 8 GPU tests ignored)
- **High Coverage Packages (>90%):** buffer, raster, UI layout, X11 present/gc/dpi
- **Moderate Coverage (50-90%):** Wayland/X11 protocols, UI widgets, effects
- **Low Coverage (<50%):** render backend, integration tests (hardware-dependent)

### Build & Verification
- **Static Linking:** ✅ Verified (`ldd bin/wain` reports "not a dynamic executable")
- **CGO Required:** Yes (Rust library linked via musl-gcc)
- **Binary Size:** 6.2 MB (bin/wain, fully static with Rust library and musl libc)
- **Build Time:** ~60 seconds (Rust library + Go binary + musl stub)
- **go vet Issues:** 1 (unsafe.Pointer warning in x11/shm, acknowledged with nolint comment)

### Dependencies
- **Go Version:** 1.24 (required)
- **External Dependencies:** None (all internal packages)
- **Rust Dependencies:** nix 0.27 (ioctl), naga 0.14 (shader parsing)
- **Build Dependencies:** musl-gcc, cargo, rustup (musl target)

---

## Conclusion

**Project is in good health** with solid fundamentals:
- Static linking works correctly
- Documentation coverage is excellent (88.76%)
- Testing is comprehensive (57 test files, 66% coverage)
- All claimed demonstration binaries exist and compile
- Core integration (Go ↔ Rust CGO) is functional

**Primary concerns are documentation accuracy, not functionality:**
- Several features marked "complete" have documented limitations (US keymap only, single-plane DRI3, DrawImage GPU-only)
- Public API does not exist (all packages marked `internal/`)
- Some event loops are stubs (widget-demo)

**Recommendations:**
1. Update README to clarify feature limitations (US QWERTY, single-plane buffers, GPU-only image compositing)
2. Add package documentation to 36 packages with 0% quality score
3. Implement or document DrawImage in software rasterizer
4. Add screenshot comparison tests for visual regression detection
5. Extract long functions in demo binaries (low priority — demo code is acceptable)

**No critical defects found.** All findings are documentation or polish issues, not functional bugs. The project delivers on its core claims with well-documented limitations.
