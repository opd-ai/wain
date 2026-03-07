# Implementation Plan: Phase 2.4 — DRI3/Present X11 Integration

## Project Context
- **What it does**: A statically-compiled Go UI toolkit with GPU rendering via Rust, supporting native X11/Wayland protocols
- **Current milestone**: Phase 2.4 — DRI3 Integration with X11 (completes Phase 2: DRM/KMS Buffer Infrastructure)
- **Estimated Scope**: Medium (8 items requiring implementation)

## Metrics Summary
| Metric | Current | Target |
|--------|---------|--------|
| Complexity hotspots (CC > 9) | 0 | 0 |
| Duplication ratio | 9.68% | < 7% |
| Documentation coverage | 91.9% | > 90% |
| Package coupling (max) | 4.5 (main) | < 4.0 |

**Notable Package Metrics:**
- `widgets` has highest cohesion (10.0) — well-designed widget abstraction
- `main` package (demo apps) has high coupling (4.5) — expected for integration demos
- `core` raster package has zero coupling — good isolation for rendering primitives

**Code Duplication Hotspots (11 violations):**
- Demo apps share ~200+ duplicated lines (rendering loops, buffer management)
- Raster core/curves share 29-line buffer initialization pattern
- Largest clone: 83 lines between `cmd/demo/main.go` and `cmd/x11-demo/main.go`

## Current Phase 2 Progress
| Subtask | Status | Implementation |
|---------|--------|----------------|
| 2.1 Kernel ioctl wrappers | ✅ Done | `render-sys/src/{drm,i915,xe}.rs` (961 LOC) |
| 2.2 Buffer allocator | ✅ Done | `render-sys/src/{allocator,slab}.rs` (441 LOC) |
| 2.3 DMA-BUF + Wayland | ✅ Done | `internal/wayland/dmabuf/` (566 LOC) |
| 2.4 DRI3 + X11 | ❌ Missing | Target of this plan |

## Implementation Steps

### Step 1: DRI3 Extension Query & Negotiation
- **Deliverable**: Add DRI3 extension detection and version negotiation to X11 client
- **Files**: Create `internal/x11/dri3/dri3.go`
- **Dependencies**: Existing `internal/x11/client/` and `internal/x11/wire/`
- **Acceptance**: Extension query returns DRI3 version ≥ 1.2
- **Validation**: 
  ```bash
  go-stats-generator analyze . --skip-tests --format json | jq '.packages[] | select(.name == "dri3")'
  ```

### Step 2: Present Extension Implementation  
- **Deliverable**: Implement Present extension for frame synchronization
- **Files**: Create `internal/x11/present/present.go`
- **Dependencies**: Step 1 (DRI3 queries Present support)
- **Acceptance**: PresentPixmap and PresentCompleteNotify implemented
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json | jq '.packages[] | select(.name == "present")'
  ```

### Step 3: DRI3PixmapFromBuffers Implementation
- **Deliverable**: Share GPU buffers with X server via DMA-BUF fds
- **Files**: Extend `internal/x11/dri3/dri3.go`
- **Dependencies**: Steps 1-2; existing Rust buffer allocator
- **Acceptance**: Successfully create X pixmap from GPU buffer fd
- **Validation**:
  ```bash
  go-stats-generator analyze internal/x11/dri3 --skip-tests --format json | jq '.documentation.coverage.functions'
  ```
  Target: ≥ 95% function documentation

### Step 4: X11 DMA-BUF Demo Binary
- **Deliverable**: Create `cmd/x11-dmabuf-demo/main.go` demonstrating GPU buffer sharing on X11
- **Files**: New `cmd/x11-dmabuf-demo/` directory
- **Dependencies**: Steps 1-3
- **Acceptance**: Demo opens window with GPU-allocated buffer displayed via DRI3
- **Validation**:
  ```bash
  make build && ./bin/x11-dmabuf-demo 2>&1 | grep -q "Demo completed"
  ```

### Step 5: Extract Shared Demo Utilities
- **Deliverable**: Reduce duplication by extracting common demo patterns to shared package
- **Files**: Create `internal/demo/` package with buffer loop, timing, error handling
- **Dependencies**: None (can parallelize with Steps 1-4)
- **Acceptance**: Reduce duplication ratio from 9.68% to <7%
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json | jq '.duplication.duplication_ratio'
  ```

### Step 6: Integration Test for DRI3 Path
- **Deliverable**: Add integration tests verifying DRI3 buffer sharing works end-to-end
- **Files**: `internal/integration/dri3_test.go`
- **Dependencies**: Steps 1-4
- **Acceptance**: Test passes on systems with DRI3-capable X server
- **Validation**:
  ```bash
  make test-go 2>&1 | grep -E "(PASS|FAIL).*dri3"
  ```

### Step 7: Update Phase 2 Documentation
- **Deliverable**: Update README.md and ROADMAP.md to reflect Phase 2 completion
- **Files**: `README.md`, `ROADMAP.md`
- **Dependencies**: Steps 1-6
- **Acceptance**: Phase 2 marked complete; Phase 3 prerequisites documented
- **Validation**: Manual review

### Step 8: Buffer Double/Triple Buffering Foundation
- **Deliverable**: Implement frame buffer ring management for both X11 and Wayland
- **Files**: Create `internal/buffer/ring.go` with shared buffer management logic
- **Dependencies**: Steps 1-5
- **Acceptance**: Buffer ring handles 2-3 frames with proper synchronization
- **Validation**:
  ```bash
  go-stats-generator analyze internal/buffer --skip-tests --format json | jq '.functions | map(select(.cyclomatic_complexity > 9)) | length'
  ```
  Target: 0 (no complex functions)

## Dependency Graph
```
     ┌──────────────────────────────────────────────────┐
     │                                                  │
     v                                                  │
[Step 1: DRI3 Query] ──> [Step 2: Present] ──> [Step 3: PixmapFromBuffers]
                                                        │
                                                        v
[Step 5: Extract Utils] ─────────────────────> [Step 4: Demo Binary]
                                                        │
                                                        v
                              [Step 6: Integration Test] ──> [Step 7: Docs]
                                                        │
                                                        v
                                            [Step 8: Buffer Ring]
```

## Scope Classification Rationale

| Criterion | Value | Classification |
|-----------|-------|----------------|
| Functions to implement | ~12-15 new | Medium |
| Lines of code estimated | ~600-900 | Medium |
| Packages to create | 3 new | Medium |
| Complexity risk | Moderate (X11 protocol work) | Medium |
| Duplication debt | 11 violations | Requires Step 5 |

## Success Criteria for Phase 2 Completion

1. **Functional**: `x11-dmabuf-demo` displays GPU-allocated buffer via DRI3
2. **Quality**: Duplication ratio < 7% (down from 9.68%)
3. **Documentation**: Function coverage ≥ 95% for new packages
4. **Testing**: Integration test passes on CI
5. **No regression**: All existing tests continue to pass

## Validation Commands Summary

```bash
# Full metrics after implementation
go-stats-generator analyze . --skip-tests --format json --output metrics-post.json

# Compare duplication
jq '.duplication.duplication_ratio' metrics-post.json

# Check new packages
jq '.packages[] | select(.name | test("dri3|present|buffer|demo"))' metrics-post.json

# Verify no new complexity hotspots
jq '[.functions[] | select(.cyclomatic_complexity > 9)] | length' metrics-post.json
```

---

## Gaps Document: Out-of-Scope Findings

The following items were identified during analysis but are outside Phase 2.4 scope:

### Deferred to Phase 3 (GPU Command Submission)
- Hardware detection and GPU generation query
- Batch buffer construction
- Pipeline state objects

### Deferred to Phase 5 (Rendering Backend Integration)  
- Damage tracking for partial redraws
- Texture atlas management for GPU path

### Technical Debt (No Timeline)
- `main` package coupling (4.5) — acceptable for demo apps, would need refactoring if demos become public API
- Missing `test_coverage` metrics — CI does not yet report coverage percentages

### Nice-to-Have Improvements
- Convert demo 83-line clone to shared rendering loop abstraction
- Add property-based tests for wire protocol encoding (complement existing fuzz tests)
