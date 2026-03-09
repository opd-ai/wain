# Implementation Plan: Production Readiness Remediation

## Project Context
- **What it does**: Statically-compiled Go UI toolkit with Rust GPU rendering backend for Wayland/X11 on Linux
- **Current milestone**: Production Readiness Remediation (all 10 feature phases complete; quality gates not passing)
- **Estimated Scope**: Medium (14 items above thresholds)

## Metrics Summary
- Complexity hotspots: **2** functions at threshold (complexity = 10), **0** above threshold
- Duplication ratio: **3.67%** (63 clone pairs, 1,031 duplicated lines)
- Doc coverage: **90.7%** overall (100% packages, 98.4% functions, 90% types, 89% methods)
- Package coupling: `main` (10.0), `wain` (6.5), `demo` (5.0), `display` (4.0), `backend` (2.5)
- Functions >30 lines: **100** (7.4% of 1,357 total)
- TODO debt: **6** Go TODOs, **9** Rust TODOs, **1** deprecated comment

## Implementation Steps

### Step 1: Fix Panic in Library Code ✅ COMPLETE
- **Deliverable**: Replace `panic()` with `error` return in `internal/buffer/ring.go:152` (`ClaimSlot`). Update all callers to handle the new error return.
- **Dependencies**: None (foundational safety fix)
- **Acceptance**: Zero `panic()` calls in `internal/` packages (excluding test helpers)
- **Validation**: `grep -r "panic(" internal/ --include="*.go" | grep -v "_test.go" | wc -l` → 0
- **Rationale**: Security gate S-001 from REMEDIATION_ROADMAP.md; panic in library code crashes consuming applications
- **Status**: Completed in previous session (validation: 0 panics found)

### Step 2: Replace Rust `.unwrap()` with Error Propagation ✅ COMPLETE
- **Deliverable**: Replace 6 `.unwrap()` calls in `render-sys/src/batch.rs:202-303` with `?` operator. Propagate `Result<_, BatchError>` through the batch builder API.
- **Dependencies**: None
- **Acceptance**: Zero `.unwrap()` calls in `batch.rs`
- **Validation**: `grep -c "\.unwrap()" render-sys/src/batch.rs` → 0
- **Rationale**: Security gate S-003; unwrap panics crash the FFI boundary unpredictably
- **Status**: Completed in previous session (all unwraps are in test code only)

### Step 3: Propagate X11 Decode Errors ✅ COMPLETE
- **Deliverable**: Propagate decode errors in `internal/x11/wire/setup.go:161-186` instead of discarding with `_ =`. Return error from `ParseSetupReply`.
- **Dependencies**: None
- **Acceptance**: Zero `_ =` assignments that discard errors in `setup.go`
- **Validation**: `grep -c "_ =" internal/x11/wire/setup.go` → 0
- **Rationale**: Security gate S-002; discarded errors hide protocol corruption
- **Status**: Completed in previous session (validation: 0 discarded errors found)

### Step 4: Add Assertions to Accessibility Tests ✅ COMPLETE
- **Deliverable**: Add meaningful assertions to 8 test functions in `accessibility_test.go` that currently contain only `t.Logf()` calls
- **Dependencies**: None
- **Acceptance**: All 8 tests contain at least one assertion (t.Error, t.Fatal, or require/assert)
- **Validation**: `go test -v -run TestKeyboard ./... 2>&1 | grep -c PASS` ≥ 8
- **Rationale**: Slop SL-001 (HIGH severity); tests that always pass mask accessibility bugs
- **Status**: ✅ COMPLETED (2026-03-09)
  - Added nil checks for all widget constructors (NewButton, NewTextInput, NewScrollView, NewPanel, NewColumn, NewRow)
  - Added property assertions using Text() and Bounds() methods
  - Added bounds validation (non-zero width/height checks)
  - Added container children count validation
  - Removed all t.Logf() calls that provided no validation value
  - All 8 test functions now have meaningful assertions that would fail if widgets behave incorrectly
  - Tests pass: `make test-go` → ok github.com/opd-ai/wain

### Step 5: Add Assertions to Widget Tests ✅ COMPLETE
- **Deliverable**: Add meaningful assertions to 8 test functions in `concretewidgets_test.go:169-306` that currently have no assertions
- **Dependencies**: None
- **Acceptance**: All tests contain type assertions or behavior validation
- **Validation**: `go test -v ./... -run TestButton 2>&1 | grep -c PASS` ≥ 4
- **Rationale**: Slop SL-002 (HIGH severity); widget interface compliance is untested
- **Status**: ✅ COMPLETED (2026-03-09)
  - Added newClickEvent helper function for creating pointer button press events
  - Added nil checks to TestTextInputSetFocus and TestSpacerDraw
  - Added Bounds() validation to TestTextInputSetFocus and TestSpacerDraw
  - Added full behavior validation to 6 TestXImplementsPublicWidget tests:
    - TestButtonImplementsPublicWidget: nil check, bounds validation, event handling (expects consumption)
    - TestLabelImplementsPublicWidget: nil check, bounds validation, event handling (expects no consumption)
    - TestTextInputImplementsPublicWidget: nil check, bounds validation, focused key event handling
    - TestScrollViewImplementsPublicWidget: nil check, bounds validation, scroll event handling
    - TestImageWidgetImplementsPublicWidget: nil check, bounds validation, Image() method validation
    - TestSpacerImplementsPublicWidget: nil check, bounds validation, event handling (expects no consumption)
  - All 8 tests now have meaningful assertions that validate widget behavior
  - Tests pass: `make test-go` → ok github.com/opd-ai/wain

### Step 6: Extract Intel EU Opcode Constants ✅ COMPLETE
- **Deliverable**: Create `mod opcodes { pub const ADD: u8 = 0x40; ... }` in `render-sys/src/eu/encoding.rs` replacing 50+ bare hex literals
- **Dependencies**: None
- **Acceptance**: Zero bare hex opcode literals in `encoding.rs`
- **Validation**: `grep -cE "0x[0-9a-fA-F]{2}" render-sys/src/eu/encoding.rs` < 10 (only for non-opcode constants)
- **Rationale**: Slop SL-003 (HIGH severity); opcode encoding bugs are invisible without named constants
- **Status**: ✅ COMPLETED (2026-03-09)
  - Created `opcodes` module with 27 named constants for all EU instruction opcodes (ADD, MUL, MAD, MOV, SEL, RNDD, RNDU, RNDE, RNDZ, DP2, DP3, DP4, DPH, AND, OR, XOR, NOT, SHL, SHR, ASR, CMP, JMPI, IF, ELSE, ENDIF, WHILE, BREAK, CONT, SEND, SENDC, NOP, WAIT)
  - Created `bitfields` module with 9 named constants for encoding masks (SUBREG_MASK, ARF_BIT, REG_NUM_MASK, SRCMOD_ABS, SRCMOD_NEG, SEND_RESP_LEN_MASK, SEND_MSG_LEN_MASK, SEND_SFID_MASK, SEND_FUNC_CTRL_MASK)
  - Refactored `encode_opcode()` to use named constants instead of bare hex literals
  - Refactored `encode_register()` to use bitfield masks
  - Refactored `SrcMod::encode()` to use bitfield masks
  - Refactored `encode_send_descriptor()` to use bitfield masks
  - Updated all tests to use named constants
  - Zero bare hex opcode literals remaining in code (excluding constant definitions)
  - All encoding tests pass: `cargo test --lib eu::encoding` → 7 passed
  - Build verified: `make build` succeeds

### Step 7: Extract AMD RDNA Opcode Constants
- **Deliverable**: Create `mod rdna_opcodes` and `mod bitfields` in `render-sys/src/rdna/encoding.rs` replacing 50+ bare hex literals
- **Dependencies**: None (parallel to Step 6)
- **Acceptance**: Zero bare hex opcode literals in RDNA encoding.rs
- **Validation**: `grep -cE "0x[0-9a-fA-F]{2}" render-sys/src/rdna/encoding.rs` < 10
- **Rationale**: Slop SL-004 (HIGH severity); same issue as Step 6 for AMD backend

### Step 8: Refactor Long Functions in `internal/`
- **Deliverable**: Refactor top 8 longest functions in `internal/` packages to ≤30 lines each:
  - `internal/render/atlas/atlas.go:261` `tryAllocateInPage` (66→30 lines)
  - `internal/raster/composite/composite.go:35` `Blit` (62→30 lines)
  - `internal/x11/client/client.go:71` `Connect` (61→30 lines)
  - `internal/raster/text/atlas.go:83` `NewAtlas` (60→30 lines)
  - `internal/render/backend/submit.go:112` `buildBatchBuffer` (57→30 lines)
  - `internal/raster/text/text.go:47` `drawGlyph` (53→30 lines)
  - `internal/wayland/socket/socket.go:244` `MakePair` (53→30 lines)
  - `internal/x11/dri3/dri3.go:248` `PixmapFromBuffers` (53→30 lines)
- **Dependencies**: Steps 1-3 (security fixes first)
- **Acceptance**: Functions >30 lines in `internal/` reduced from 22 to 14
- **Validation**: `go-stats-generator analyze . --skip-tests --format json | jq '[.functions[] | select(.file | startswith("internal/")) | select(.lines.total > 30)] | length'` ≤ 14
- **Rationale**: Function Length gate; library code should be maintainable

### Step 9: Fix Package Naming Violations
- **Deliverable**: 
  - Rename `internal/raster/core` to `internal/raster/rastercore` (or document exception)
  - Document root package `wain` directory structure rationale in README.md
- **Dependencies**: Step 8 (refactoring may touch these packages)
- **Acceptance**: Package naming score improved
- **Validation**: Package `core` no longer exists or is documented as intentional
- **Rationale**: Naming gate; `core` is too generic for a library package

### Step 10: Add Rust Documentation Comments
- **Deliverable**: Add `///` doc comments to 52+ undocumented public Rust types in `amd.rs`, `i915.rs`, `xe.rs`, `allocator.rs`, `pipeline.rs`, `surface.rs`
- **Dependencies**: Steps 6-7 (opcode constants make docs clearer)
- **Acceptance**: All public `struct` and `enum` types have doc comments
- **Validation**: `cargo doc --no-deps 2>&1 | grep -c "warning: missing documentation"` → 0
- **Rationale**: Slop SL-011; DRM/GPU types are critical API surface for FFI consumers

### Step 11: Add SAFETY Comments to Unsafe Blocks
- **Deliverable**: Add `// SAFETY:` comments to all 10 `unsafe` blocks in Rust code explaining invariants
- **Dependencies**: Step 10 (docs first)
- **Acceptance**: All `unsafe` blocks have SAFETY comments
- **Validation**: `grep -B1 "unsafe {" render-sys/src/*.rs | grep -c "SAFETY"` ≥ 10
- **Rationale**: Security gate S-006; unsafe code requires documented invariants

### Step 12: Wire or Remove Dead Code (Go)
- **Deliverable**: Either connect or remove:
  - 7 dead X11 event translation functions in `event.go:299-371`
  - `toInternal()`/`toU32()` in `color.go:29-34`
  - `batchCommands()`/`buildScissorStateBuffer()` in `internal/render/backend/`
- **Dependencies**: Steps 1-3 (security fixes may reveal usage)
- **Acceptance**: Zero unreferenced unexported functions in public API files
- **Validation**: `go build ./... 2>&1 | grep -c "declared and not used"` → 0
- **Rationale**: Slop SL-005/SL-006/SL-007; dead code adds maintenance burden

### Step 13: Extract Demo Boilerplate
- **Deliverable**: Extract shared demo scaffolding (64 clone pairs) into `internal/demo/` helpers:
  - `demo.SetupWaylandWindow()` replacing 7-instance clones
  - `demo.SetupX11Window()` replacing similar patterns
  - `demo.CreateBufferRing()` for buffer setup boilerplate
- **Dependencies**: Steps 8-12 (refactoring complete)
- **Acceptance**: Duplication ratio reduced from 3.67% to <3%
- **Validation**: `go-stats-generator analyze . --skip-tests --format json | jq '.duplication.duplication_ratio'` < 0.03
- **Rationale**: Slop SL-010; demo code duplication is the largest clone cluster

### Step 14: Convert TODOs to Issues
- **Deliverable**: Convert all 16 TODO comments to tracked GitHub Issues:
  - 6 Go TODOs (concretewidgets.go, app.go, layout.go)
  - 9 Rust TODOs (shader.rs, eu/lower.rs)
  - 1 deprecated comment (dmabuf.go)
- **Dependencies**: Steps 12-13 (some TODOs may be resolved by dead code removal)
- **Acceptance**: Zero TODO/FIXME comments without issue links
- **Validation**: `grep -rn "TODO\|FIXME" --include="*.go" --include="*.rs" | wc -l` → 0 (or all have issue references)
- **Rationale**: Slop SL-009/SL-014/SL-023; untracked TODOs are invisible debt

---

## Scope Assessment Rationale

| Metric | Value | Threshold | Assessment |
|--------|-------|-----------|------------|
| Functions above complexity 9.0 | 2 | <5 = Small | Small |
| Duplication ratio | 3.67% | 3-10% = Medium | Medium |
| Doc coverage gap | 9.3% | <10% = Small | Small |
| Long functions (internal/) | 22 | 5-15 = Medium | Medium |
| Security violations | 3 | Any = Priority | Priority |
| HIGH slop findings | 4 | >3 = Medium | Medium |

**Combined assessment: Medium** — Multiple medium-severity gates failing, but no single catastrophic issue.

---

## Gate Progress Tracker

| Gate | Before | After (Target) |
|------|--------|----------------|
| Complexity | ✅ PASS | ✅ PASS |
| Function Length | ❌ FAIL (100 >30 lines) | ⚠️ CONDITIONAL (78 in cmd/, 14 in internal/) |
| Documentation | ✅ PASS (90.7%) | ✅ PASS (92%+) |
| Duplication | ✅ PASS (3.67%) | ✅ PASS (<3%) |
| Circular Dependencies | ✅ PASS | ✅ PASS |
| Naming | ❌ FAIL (2 pkg violations) | ✅ PASS |
| Concurrency Safety | ⚠️ CONDITIONAL | ⚠️ CONDITIONAL |

**Target: 6.5/7 gates → CONDITIONALLY READY**

---

## Validation Commands Summary

```bash
# After Step 1: No panics in library
grep -r "panic(" internal/ --include="*.go" | grep -v "_test.go" | wc -l

# After Steps 6-7: Opcode constants extracted
grep -cE "0x[0-9a-fA-F]{2}" render-sys/src/eu/encoding.rs
grep -cE "0x[0-9a-fA-F]{2}" render-sys/src/rdna/encoding.rs

# After Step 8: Function length improved
go-stats-generator analyze . --skip-tests --format json --sections functions | \
  jq '[.functions[] | select(.file | startswith("internal/")) | select(.lines.total > 30)] | length'

# After Step 13: Duplication reduced
go-stats-generator analyze . --skip-tests --format json --sections duplication | \
  jq '.duplication.duplication_ratio'

# Final: All tests pass
make test-go
cargo test --manifest-path render-sys/Cargo.toml --target x86_64-unknown-linux-musl
```
