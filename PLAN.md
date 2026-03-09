# Implementation Plan: AUDIT Remediation v1

## Project Context
- **What it does**: Wain is a statically-compiled Go UI toolkit with Rust-based GPU rendering that implements Wayland and X11 display protocols from scratch, producing single fully-static binaries with zero runtime dependencies.
- **Current milestone**: AUDIT.md remediation — 8 open findings (1 CRITICAL, 2 HIGH, 3 MEDIUM, 2 LOW)
- **Estimated Scope**: Medium (10 items above various thresholds; manageable in ~5 focused steps)

## Metrics Summary (2026-03-09)

| Metric | Value | Assessment |
|--------|-------|------------|
| Lines of Code | 13,078 | — |
| Total Packages | 37 | — |
| Functions with cc≥9 | 10 total, 6 production | Medium |
| Functions with cc≥10 | 4 total, 4 production | Action needed |
| Production functions >50 lines | 3 | Action needed |
| Duplication ratio | 3.35% | Medium (within tolerance) |
| Packages with null doc coverage | 37 | HIGH — complete gap |
| go vet warnings | 1 | Action needed |

**Complexity hotspots (production code, cc≥10):**
- `bindWaylandGlobals` (app.go:1204, cc=11) — critical display initialization path
- `applyToTheme` (theme.go:195, cc=10) — theme configuration
- `decodeVisuals` (internal/x11/wire/setup.go:231, cc=10) — X11 protocol decoding

**Long production functions (>50 lines):**
- `KeyToString` (internal/demo/logging.go:113, 51 lines)
- `BlitScaled` (internal/raster/composite/ops.go:107, 51 lines)
- `bindWaylandGlobals` (app.go:1204, 58 lines)

## Implementation Steps

### Step 1: Fix GPU Integration Tests (CRITICAL)

**Deliverable**: GPU tests skip gracefully on systems without compatible hardware instead of failing the entire test suite.

**Dependencies**: None (blocking issue)

**Changes**:
1. Add build tag `//go:build gpu_required` to `internal/integration/gpu_test.go`
2. Add hardware capability check at test start with `t.Skip()` when DRM unavailable
3. Update `render-sys/src/allocator.rs` to return `ENOTSUP` error code for mmap failures on unsupported hardware
4. Update README.md Requirements section to document GPU hardware requirements

**Acceptance**: `make test-go` passes with exit code 0 on systems without GPU hardware

**Validation**:
```bash
make test-go 2>&1 | grep -E "^(ok|PASS|---)" | head -5
# Expected: "ok github.com/opd-ai/wain/internal/integration" or SKIP messages
```

---

### Step 2: Fix go vet Warning (HIGH)

**Deliverable**: Zero `go vet` warnings in the codebase.

**Dependencies**: None

**Changes**:
1. Modify `internal/x11/shm/extension.go:190-192` to eliminate intermediate variable in syscall-to-unsafe.Pointer conversion
2. Replace current pattern with immediate conversion in return statement
3. Add test verifying shmat memory validity after GC cycles

**Acceptance**: `go vet ./...` exits with code 0 and produces no output

**Validation**:
```bash
go vet ./... 2>&1 | wc -l
# Expected: 0
```

---

### Step 3: Reduce Complexity of bindWaylandGlobals (MEDIUM)

**Deliverable**: Split `bindWaylandGlobals` (app.go:1204) into focused helper functions, reducing cyclomatic complexity from 11 to <9.

**Dependencies**: None

**Changes**:
1. Extract `bindCompositor(registry *Registry, globals map[string]Global) error` — handles compositor and SHM binding
2. Extract `bindShellProtocols(registry *Registry, globals map[string]Global) error` — handles xdg-shell and xdg-decoration
3. Extract `bindInputDevices(registry *Registry, globals map[string]Global) error` — handles seat, pointer, keyboard
4. Reduce main `bindWaylandGlobals` to ~20 lines calling these three helpers

**Acceptance**: Function cyclomatic complexity below 9

**Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json | \
  jq '.functions[] | select(.name == "bindWaylandGlobals") | .complexity.cyclomatic'
# Expected: < 9
```

---

### Step 4: Reduce Complexity of Other High-CC Functions (MEDIUM)

**Deliverable**: Reduce cyclomatic complexity of remaining cc≥10 functions.

**Dependencies**: Step 3 (pattern established)

**Changes**:
1. **applyToTheme** (theme.go:195, cc=10): Extract `applyColorPalette(theme *Theme, colors ColorPalette)` and `applyFontConfig(theme *Theme, fonts FontConfig)`
2. **decodeVisuals** (internal/x11/wire/setup.go:231, cc=10): Extract `parseVisualDepth(buf []byte, offset int) (Visual, int, error)` for per-visual parsing

**Acceptance**: Zero functions with cyclomatic complexity ≥10

**Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json | \
  jq '[.functions[] | select(.complexity.cyclomatic >= 10)] | length'
# Expected: 0
```

---

### Step 5: Split Long Production Functions (MEDIUM)

**Deliverable**: Reduce line count of production functions exceeding 50 lines.

**Dependencies**: Steps 3-4 (overlapping refactoring)

**Changes**:
1. **BlitScaled** (internal/raster/composite/ops.go:107): Extract `blitScaledCore(dst, src *Image, scale float64)` for inner loop
2. **KeyToString** (internal/demo/logging.go:113): Convert to lookup table pattern (map[KeyCode]string)
3. `bindWaylandGlobals` already addressed in Step 3

**Acceptance**: Zero production functions >50 lines (excluding cmd/ demo binaries)

**Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json | \
  jq '[.functions[] | select(.lines.total > 50 and (.file | contains("cmd/") | not))] | length'
# Expected: 0
```

---

## Deferred Items (Future Plan)

The following AUDIT findings are deferred to a future plan as they require larger scope or are lower priority:

### Documentation Coverage (HIGH priority, Large scope)
- **Finding**: 37 packages have null documentation coverage
- **Scope**: Large — requires GoDoc comments for all exported symbols
- **Recommendation**: Create separate documentation sprint with package-by-package approach
- **Priority order**: public API (done) → internal/render (CGO boundary) → protocol packages

### Test Coverage Metrics (LOW)
- **Finding**: Coverage percentage unknown; no baseline established
- **Recommendation**: Run `make coverage`, establish 60%/80% thresholds, add CI gate

### Naming Convention Documentation (LOW)
- **Finding**: File stuttering (wire/wire.go) is idiomatic but undocumented
- **Recommendation**: Document exemption in CONTRIBUTING.md; no code changes needed

---

## Success Criteria

After completing Steps 1-5:

| Metric | Before | Target |
|--------|--------|--------|
| `make test-go` exit code | Non-zero (GPU failures) | 0 |
| `go vet ./...` warnings | 1 | 0 |
| Functions with cc≥10 | 4 | 0 |
| Production functions >50 lines | 3 | 0 |
| AUDIT open findings | 8 | 3 (deferred) |

---

## Validation Suite

Run all validations after implementing all steps:

```bash
# Step 1: GPU tests pass or skip
make test-go 2>&1 | tail -5

# Step 2: No go vet warnings
go vet ./... 2>&1 | wc -l

# Steps 3-4: No high-complexity functions
go-stats-generator analyze . --skip-tests --format json | \
  jq '[.functions[] | select(.complexity.cyclomatic >= 10)] | length'

# Step 5: No long production functions
go-stats-generator analyze . --skip-tests --format json | \
  jq '[.functions[] | select(.lines.total > 50 and (.file | contains("cmd/") | not))] | length'

# Summary
echo "=== Validation Summary ==="
echo "test-go: $(make test-go >/dev/null 2>&1 && echo PASS || echo FAIL)"
echo "go-vet: $(go vet ./... 2>&1 | wc -l) warnings"
echo "cc>=10: $(go-stats-generator analyze . --skip-tests --format json 2>/dev/null | jq '[.functions[] | select(.complexity.cyclomatic >= 10)] | length') functions"
echo "lines>50: $(go-stats-generator analyze . --skip-tests --format json 2>/dev/null | jq '[.functions[] | select(.lines.total > 50 and (.file | contains("cmd/") | not))] | length') functions"
```

---

**Generated**: 2026-03-09T23:08Z  
**Tool**: go-stats-generator 1.0.0  
**Baseline**: metrics.json (171 files, 13,078 LOC)
