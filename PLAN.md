# Implementation Plan: Post-Roadmap Maintenance & Enhancement

## Project Context
- **What it does**: A statically-compiled Go UI toolkit with GPU rendering via Rust, supporting both X11 and Wayland display servers on Intel and AMD GPUs.
- **Current milestone**: All 8 phases complete per ROADMAP.md; this plan targets quality improvements and maintainability.
- **Estimated Scope**: Large (60 functions above complexity 9.0, 4.7% duplication ratio)

## Metrics Summary
- **Complexity hotspots**: 60 functions above threshold (≥9.0 overall complexity)
- **Duplication ratio**: 4.71% (944 duplicated lines across 51 clone pairs)
- **Doc coverage**: 90.87% overall (functions 98%, methods 88.5%)
- **Package coupling**: backend (2.5), widgets (2.0), decorations (2.0), main demos (10.0)

## Implementation Steps

### Step 1: Refactor Demo Application Logic
- **Deliverable**: Extract duplicate rendering/setup code from cmd/*-demo binaries into internal/demo package
- **Dependencies**: None
- **Files**: 
  - `cmd/double-buffer-demo/main.go` (complexity 20.2, 17.4)
  - `cmd/auto-render-demo/main.go` (complexity 13.2)
  - `cmd/dmabuf-demo/main.go` (complexity 12.2)
  - `cmd/wayland-demo/main.go` (complexity 12.2)
- **Acceptance**: Reduce main package high-complexity functions from 9 to ≤4
- **Validation**: 
```bash
go-stats-generator analyze . --skip-tests --format json --sections functions | \
  jq '[.functions[] | select(.package == "main" and .complexity.overall > 9.0)] | length'
# Target: ≤4
```

### Step 2: Deduplicate X11 Client Protocol Code
- **Deliverable**: Extract shared request/reply handling from SendRequestAndReplyWithFDs and reduce X11 duplication
- **Dependencies**: None
- **Files**:
  - `internal/x11/client/client.go` (complexity 18.9)
  - `internal/x11/dri3/dri3.go` (duplication with present.go)
  - `internal/x11/present/present.go` (duplication with dri3.go)
- **Acceptance**: Reduce X11 client complexity ≤12, eliminate dri3/present clone pair
- **Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json --sections functions,duplication | \
  jq '{complexity: [.functions[] | select(.name == "SendRequestAndReplyWithFDs")] | .[0].complexity.overall, dri3_present_dupes: [.duplication.clones[] | select(.instances[0].file | contains("dri3") or contains("present"))] | length}'
# Target: complexity ≤12, dri3_present_dupes == 0
```

### Step 3: Refactor Atlas Region Allocation
- **Deliverable**: Simplify AllocateImageRegion by extracting helper functions for boundary checking and fallback logic
- **Dependencies**: None
- **Files**:
  - `internal/render/atlas/atlas.go` (AllocateImageRegion complexity 16.3, tryAllocateInPage 10.6)
- **Acceptance**: AllocateImageRegion complexity ≤10
- **Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json --sections functions | \
  jq '[.functions[] | select(.name == "AllocateImageRegion")] | .[0].complexity.overall'
# Target: ≤10
```

### Step 4: Simplify Raster Effects Module
- **Deliverable**: Refactor LinearGradient and blur functions to reduce branching and nesting
- **Dependencies**: None
- **Files**:
  - `internal/raster/effects/effects.go` (LinearGradient 15.0, blurHorizontal 11.1, blurVertical 11.1)
- **Acceptance**: LinearGradient complexity ≤10, blur functions ≤8
- **Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json --sections functions | \
  jq '[.functions[] | select(.package == "effects" and .complexity.overall > 9.0)] | length'
# Target: 0
```

### Step 5: Reduce Widget Drawing Complexity
- **Deliverable**: Factor out common widget rendering patterns to reduce Draw/RenderToDisplayList complexity
- **Dependencies**: None  
- **Files**:
  - `internal/ui/widgets/widgets.go` (multiple Draw/RenderToDisplayList at 11.4, 10.9, 9.3)
  - `internal/ui/pctwidget/widget.go` (Draw 11.4)
  - `internal/ui/pctwidget/autolayout.go` (AutoLayout 15.3)
- **Acceptance**: No widget functions above complexity 10
- **Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json --sections functions | \
  jq '[.functions[] | select((.package == "widgets" or .package == "pctwidget") and .complexity.overall > 10.0)] | length'
# Target: 0
```

### Step 6: Consolidate Wire Protocol Encoding
- **Deliverable**: Reduce duplication in encodeArgument, DecodeString, EncodeString via shared codec helpers
- **Dependencies**: None
- **Files**:
  - `internal/wayland/wire/wire.go` (encodeArgument 12.7, DecodeString 10.1, EncodeString 10.1)
  - `internal/x11/wire/setup.go` (DecodeSetupReply 11.4, ReadAuthority 10.6)
- **Acceptance**: Wire encoding complexity ≤9, eliminate setup.go internal duplication
- **Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json --sections functions,duplication | \
  jq '{wire_complexity: [.functions[] | select(.package == "wire" and .complexity.overall > 9.0)] | length, setup_dupes: [.duplication.clones[] | select(.instances[0].file | contains("setup.go"))] | length}'
# Target: wire_complexity == 0, setup_dupes == 0
```

### Step 7: Improve Backend Render Pipeline
- **Deliverable**: Simplify RenderWithDamage, submitBatchesWithScissor, and New() initialization
- **Dependencies**: Step 5 (widget changes may affect backend interface)
- **Files**:
  - `internal/render/backend/backend.go` (RenderWithDamage 11.4, New 10.9)
  - `internal/render/backend/submit.go` (submitBatchesWithScissor 10.1, duplication)
  - `internal/render/commands.go` (18-line duplicate at lines 138, 160)
- **Acceptance**: Backend package has no functions above complexity 9.0, eliminate submit.go duplication
- **Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json --sections functions,duplication | \
  jq '{backend_complexity: [.functions[] | select(.package == "backend" and .complexity.overall > 9.0)] | length, submit_dupes: [.duplication.clones[] | select(.instances[0].file | contains("submit.go"))] | length}'
# Target: backend_complexity == 0, submit_dupes == 0
```

### Step 8: Reduce Package Coupling in Decorations
- **Deliverable**: Decouple decorations from direct raster/widget dependencies, use interfaces
- **Dependencies**: Step 5, Step 7
- **Files**:
  - `internal/ui/decorations/resize.go` (HitTest 12.2)
  - `internal/ui/decorations/titlebar.go` (19-line duplication)
- **Acceptance**: Decorations coupling score ≤1.0, no functions above complexity 10
- **Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json --sections packages,functions | \
  jq '{coupling: [.packages[] | select(.name == "decorations")] | .[0].coupling_score, high_complexity: [.functions[] | select(.package == "decorations" and .complexity.overall > 10.0)] | length}'
# Target: coupling ≤1.0, high_complexity == 0
```

### Step 9: Improve Method Documentation Coverage
- **Deliverable**: Add missing documentation for exported methods (currently 88.5% coverage)
- **Dependencies**: All refactoring steps complete
- **Files**:
  - Focus on packages: backend (54 functions), decorations (50 functions), widgets (49 functions), datadevice (44 functions)
- **Acceptance**: Method documentation coverage ≥95%
- **Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json --sections documentation | \
  jq '.documentation.coverage.methods'
# Target: ≥95
```

### Step 10: Final Metrics Validation
- **Deliverable**: Verify all quality targets met, run full test suite
- **Dependencies**: All previous steps
- **Acceptance**: 
  - Functions above complexity 9.0: ≤20 (from 60)
  - Duplication ratio: ≤3% (from 4.71%)
  - Doc coverage: ≥93% overall (from 90.87%)
- **Validation**:
```bash
go-stats-generator analyze . --skip-tests --format json --sections functions,duplication,documentation | \
  jq '{high_complexity_count: [.functions[] | select(.complexity.overall > 9.0)] | length, duplication_ratio: .duplication.duplication_ratio, doc_coverage: .documentation.coverage.overall}'
# Target: {high_complexity_count: ≤20, duplication_ratio: ≤0.03, doc_coverage: ≥93}
```

---

## Complexity Distribution (Baseline)

| Complexity Range | Count | Target |
|-----------------|-------|--------|
| 9.0 - 12.0 | 39 | ≤15 |
| 12.0 - 15.0 | 14 | ≤5 |
| 15.0 - 18.0 | 4 | 0 |
| 18.0+ | 3 | 0 |
| **Total >9.0** | **60** | **≤20** |

## Duplication Hotspots (Baseline)

| Location | Lines | Clone Count |
|----------|-------|-------------|
| cmd/*/main.go | 34, 26, 25 | 4 clusters |
| internal/render/backend/submit.go | 23 | 2 |
| internal/ui/decorations/titlebar.go | 19 | 2 |
| internal/ui/widgets/widgets.go | 19 | 2 |
| internal/render/commands.go | 18 | 2 |

## Notes

- All phases from ROADMAP.md are complete; this plan addresses technical debt identified by metrics
- Steps are ordered by impact (highest-complexity files first) then dependency
- Each step is independently testable with specific validation commands
- Demo refactoring (Step 1) has highest impact: 9 of 60 high-complexity functions are in main package
- Generated: 2026-03-08 using go-stats-generator v1.0.0
