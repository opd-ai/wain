# Implementation Plan: Code Quality & Maintenance Enhancement

## Project Context
- **What it does**: A statically-compiled Go UI toolkit with GPU rendering via Rust (Intel + AMD backends)
- **Current milestone**: All 8 roadmap phases complete — project ready for quality refinement
- **Estimated Scope**: Medium (15 complexity hotspots, 4% duplication, 90% doc coverage)

## Metrics Summary
| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Functions > complexity 9.0 | **41** | <15 | ⚠️ Medium |
| Functions > complexity 15.0 | **0** | <5 | ✅ Good |
| Duplication ratio | **4.03%** | <3% | ⚠️ Medium |
| Duplicate lines | 861 | — | — |
| Clone pairs | 45 | — | — |
| Doc coverage (overall) | **90.15%** | >90% | ✅ Good |
| Doc coverage (methods) | **87.24%** | >90% | ⚠️ Near threshold |
| High coupling packages | 7/35 | — | — |

### Complexity Hotspots (>10.0 overall)
| Function | Package | File | Complexity |
|----------|---------|------|------------|
| FillRoundedRect | core | internal/raster/core/rect.go:47 | 14.5 |
| lineCoverage | core | internal/raster/core/line.go:60 | 14.0 |
| handleGeometry | output | internal/wayland/output/output.go:149 | 13.5 |
| main | main | cmd/auto-render-demo/main.go:23 | 13.2 |
| drawGlyph | text | internal/raster/text/text.go:47 | 13.2 |
| createBufferRing | main | cmd/double-buffer-demo/main.go:144 | 12.7 |
| encodeArgument | wire | internal/wayland/wire/wire.go:381 | 12.7 |
| Coalesce | displaylist | internal/raster/displaylist/damage.go:53 | 12.4 |
| AttachAndDisplayBuffer | demo | internal/demo/shm.go:19 | 12.2 |
| FillRect | core | internal/raster/core/rect.go:7 | 11.9 |
| Blit | composite | internal/raster/composite/composite.go:35 | 11.9 |
| SetupWaylandGlobals | demo | internal/demo/wayland.go:41 | 10.9 |
| setupX11Context | main | cmd/x11-dmabuf-demo/main.go:159 | 10.9 |
| MakePair | socket | internal/wayland/socket/socket.go:236 | 10.9 |
| AcquireForWriting | buffer | internal/buffer/ring.go:160 | 10.6 |

### Top Duplication Violations
| Lines | Type | Locations |
|-------|------|-----------|
| 34 | renamed | cmd/gpu-triangle-demo/main.go (293-326, 421-454) |
| 29 | renamed | internal/render/display/wayland.go:82-110, x11.go:142-170 |
| 26 | renamed | cmd/dmabuf-demo/main.go (151-176, 160-185) |
| 25 | exact | cmd/gpu-triangle-demo/main.go (3 locations) |
| 25 | renamed | cmd/perf-demo/main.go (4 locations) |

### Package Coupling Analysis
| Package | Coupling Score | Dependencies |
|---------|----------------|--------------|
| main (cmd/*) | 10.0 | 21 |
| display | 3.5 | 7 |
| demo | 3.5 | 7 |
| backend | 2.5 | 5 |
| consumer | 2.0 | 4 |
| decorations | 2.0 | 4 |
| widgets | 2.0 | 4 |

---

## Implementation Steps

### Step 1: Extract Render Presentation Logic
- **Deliverable**: Create shared `internal/render/present/present.go` to deduplicate 29-line render-and-present pattern across Wayland and X11 display modules
- **Dependencies**: None
- **Files**: 
  - `internal/render/display/wayland.go` (lines 82-110)
  - `internal/render/display/x11.go` (lines 142-170)
- **Acceptance**: Duplication ratio reduced by ≥0.3%, no new complexity hotspots
- **Validation**: 
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections duplication | \
    jq '.duplication | {ratio: .duplication_ratio, lines: .duplicated_lines}'
  ```

### Step 2: Refactor FillRoundedRect (Highest Complexity)
- **Deliverable**: Extract corner-drawing and edge-filling into helper functions to reduce cyclomatic complexity
- **Dependencies**: None
- **Files**: `internal/raster/core/rect.go` (line 47, complexity 14.5)
- **Acceptance**: `FillRoundedRect` complexity ≤9.0
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.name == "FillRoundedRect")] | .[0].complexity.overall'
  ```

### Step 3: Refactor lineCoverage Function
- **Deliverable**: Simplify line coverage calculation by extracting slope-handling and pixel-iteration logic
- **Dependencies**: None
- **Files**: `internal/raster/core/line.go` (line 60, complexity 14.0)
- **Acceptance**: `lineCoverage` complexity ≤9.0
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.name == "lineCoverage")] | .[0].complexity.overall'
  ```

### Step 4: Deduplicate Demo Setup Patterns
- **Deliverable**: Create shared demo setup helpers in `internal/demo/` to eliminate GPU triangle/dmabuf/perf demo duplication (25-34 line blocks)
- **Dependencies**: None
- **Files**:
  - `cmd/gpu-triangle-demo/main.go` (3+ duplicated blocks)
  - `cmd/dmabuf-demo/main.go`
  - `cmd/perf-demo/main.go`
- **Acceptance**: Clone pairs reduced by ≥5, duplication ratio <3.5%
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections duplication | \
    jq '.duplication | {pairs: .clone_pairs, ratio: .duplication_ratio}'
  ```

### Step 5: Simplify Wayland Output Event Handler
- **Deliverable**: Refactor `handleGeometry` in output package to extract field assignments into a struct builder
- **Dependencies**: None
- **Files**: `internal/wayland/output/output.go` (line 149, complexity 13.5)
- **Acceptance**: `handleGeometry` complexity ≤9.0
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.name == "handleGeometry")] | .[0].complexity.overall'
  ```

### Step 6: Refactor encodeArgument Switch Statement
- **Deliverable**: Replace type-switch with map-based encoder dispatch or extract type-specific encoders
- **Dependencies**: None
- **Files**: `internal/wayland/wire/wire.go` (line 381, complexity 12.7)
- **Acceptance**: `encodeArgument` complexity ≤9.0
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.name == "encodeArgument")] | .[0].complexity.overall'
  ```

### Step 7: Improve Method Documentation Coverage
- **Deliverable**: Add godoc comments to undocumented exported methods (target: >90% method coverage)
- **Dependencies**: Steps 1-6 (avoid documenting code that will be refactored)
- **Files**: Focus on high-export packages: `backend`, `widgets`, `display`
- **Acceptance**: Method documentation coverage ≥90%
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections documentation | \
    jq '.documentation.coverage.methods'
  ```

### Step 8: Extract Damage Coalescing Logic
- **Deliverable**: Simplify `Coalesce` function by extracting overlap detection and region merging into helpers
- **Dependencies**: None (can run parallel with Steps 2-6)
- **Files**: `internal/raster/displaylist/damage.go` (line 53, complexity 12.4)
- **Acceptance**: `Coalesce` complexity ≤9.0
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions | \
    jq '[.functions[] | select(.name == "Coalesce")] | .[0].complexity.overall'
  ```

---

## Success Criteria

| Metric | Current | Target | Validation |
|--------|---------|--------|------------|
| Functions > complexity 9.0 | 41 | ≤20 | `jq '[.functions[] \| select(.complexity.overall > 9.0)] \| length'` |
| Duplication ratio | 4.03% | <3.0% | `jq '.duplication.duplication_ratio'` |
| Doc coverage (methods) | 87.24% | ≥90% | `jq '.documentation.coverage.methods'` |
| Clone pairs | 45 | ≤35 | `jq '.duplication.clone_pairs'` |

---

## Deferred Work (Out of Scope)

These items from the roadmap are **not addressed** by this plan:

1. **VT Switch Handling** (Phase 7.2 deferred) — Requires kernel/terminal signal integration
2. **DPMS Handling** (Phase 7.2 deferred) — Requires compositor event integration
3. **AT-SPI2 Accessibility** (Phase 8.4 documented) — Documented limitation, 6-week effort
4. **GPU Backend Screenshot Tests** (screenshot_test.go TODO) — Requires Phase 5 GPU rendering maturity

---

## Metrics Baseline Reference

Generated: 2026-03-08  
Tool: go-stats-generator 1.0.0  
Files analyzed: 137 Go files  
Total LOC: 9,409  
Total functions: 362  
Total methods: 626  

```bash
# Regenerate baseline
go-stats-generator analyze . --skip-tests --format json --output metrics.json \
  --sections functions,duplication,documentation,packages,patterns
```
