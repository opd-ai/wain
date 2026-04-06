# Rust → Pure Go Migration Plan

## Executive Summary

The wain codebase contains a single Rust crate — `render-sys/` (~7,400 LOC) — that implements GPU-accelerated rendering via direct DRM/KMS ioctl calls, GPU buffer management, and shader compilation (WGSL → Intel EU / AMD RDNA machine code). This library is compiled as a static archive (`librender_sys.a`) and linked into the Go binary through CGO.

**Migration feasibility: HIGH, with targeted GC mitigations.**

The Rust code falls into three categories:

| Category | LOC (approx.) | GC Risk | Feasibility |
|---|---|---|---|
| DRM ioctl wrappers + GPU detection | ~1,800 | **Low** — syscall-only, no heap allocation on hot paths | Straightforward; Go's `syscall` / `unix.IoctlSetPointerInt` already handle this |
| Buffer allocator + command submission | ~2,200 | **Medium** — pre-allocated slabs can be replicated with `sync.Pool` and `[]byte` pools | Moderate; main risk is mmap lifecycle management |
| Shader compiler (naga frontend + EU/RDNA backends) | ~3,400 | **High** — IR graph traversal allocates many small nodes; register allocation is allocation-intensive | Hardest; requires arena allocation (`arena` package or manual free-list), pre-sized slices, and careful escape analysis |

The existing Go codebase already contains a pure-Go software rasterizer (`internal/raster/`), Wayland/X11 display layers, and a UI framework — all GC-safe. The migration therefore only touches `render-sys/` and its three Go FFI files in `internal/render/`.

Static-binary and zero-dependency goals are **preserved** — eliminating Rust actually removes the musl-gcc, Cargo, and rustup toolchain requirements.

---

## Component Inventory

| # | Component | Purpose | LOC | GC Risk | Mitigation | Priority |
|---|---|---|---|---|---|---|
| 1 | `render-sys/src/detect.rs` | GPU generation detection (Intel Gen9–Xe, AMD RDNA1–3) via DRM ioctls | 330 | **Low** — single ioctl per call, no heap allocation | Use `golang.org/x/sys/unix.IoctlSetPointerInt` with stack-allocated structs. All param structs (`drm_i915_getparam`, `amdgpu_info`) fit on the stack (&lt;64 bytes). | P0 |
| 2 | `render-sys/src/drm.rs` | Low-level `ioctl()` wrapper + `DrmDevice` (fd owner) | 156 | **Low** — thin syscall wrapper, no heap | Port to `unix.Ioctl*`. `DrmDevice` becomes `type DrmDevice struct { fd int }` — value type, no GC. | P0 |
| 3 | `render-sys/src/i915.rs` | Intel i915 GEM create, mmap, execbuffer2, context management | 593 | **Low** — ioctl structs are fixed-size; relocation arrays are pre-allocated | Declare ioctl structs as `[unsafe.Sizeof(...)]byte` or typed structs with `encoding/binary`. Relocation slices: pre-allocate to expected capacity (typ. &lt;64 entries) and reuse via `sync.Pool`. | P1 |
| 4 | `render-sys/src/xe.rs` | Intel Xe VM, exec-queue, and exec ioctls | 714 | **Low** — similar profile to i915, slightly more structs | Same strategy as i915. Xe query results: read into stack buffer (typ. &lt;4 KB), parse without intermediate allocations. | P1 |
| 5 | `render-sys/src/amd.rs` | AMD AMDGPU GEM, VA management, context, chip info queries | 827 | **Low–Medium** — VA range tracking may accumulate heap state | Use a pre-allocated free-list for VA ranges (`[]VARange` with cap set at init). Chip info queries use fixed structs on stack. | P1 |
| 6 | `render-sys/src/allocator.rs` | Slab-based GPU buffer allocator with tiling support | 444 | **Medium** — `Buffer` objects are long-lived, but slab metadata is small | `Buffer` as value type stored in a pre-allocated `[]Buffer` slab (fixed cap, index-based). `sync.Pool` for temporary ioctl param structs. Tiling enum as `uint32` constant. | P2 |
| 7 | `render-sys/src/slab.rs` | Generic memory slab allocator | 217 | **Medium** — tracks free/used indices, potential slice growth | Pre-allocate slab backing `[]T` to max capacity at creation. Use bitset (`[]uint64`) for free-list instead of linked-list pointers — eliminates per-node allocation entirely. | P2 |
| 8 | `render-sys/src/batch.rs` | Batch buffer builder (GPU command dword stream + relocations) | 335 | **Medium** — appends to `Vec<u32>` and `Vec<Relocation>` on every draw call | Use `*bytes.Buffer` from `sync.Pool` with `Reset()`. Pre-allocate to 4 KB (typical batch). Relocation slice: pre-allocate `[]Relocation` with cap 64 and reset `len` to 0 between frames. | P2 |
| 9 | `render-sys/src/surface.rs` | Surface management (framebuffer association, scanout) | 608 | **Low–Medium** — surfaces are long-lived, created once per window | `Surface` as a struct with value-type fields. Created infrequently (window open); not GC-sensitive. | P2 |
| 10 | `render-sys/src/submit.rs` | Command submission dispatch (auto-detect i915/Xe/AMD) | 252 | **Low** — thin dispatch layer over driver ioctls | Port as a Go `switch` on `GpuGeneration`. No allocations. | P1 |
| 11 | `render-sys/src/cmd/mod.rs` | GPU command type definitions and builder entry point | 101 | **Low** — type defs only | Go constants + types. Zero allocation. | P2 |
| 12 | `render-sys/src/cmd/state.rs` | 3D pipeline state commands (viewport, blend, depth, stencil) | 669 | **Low–Medium** — emits dwords into batch buffer | Write helpers that append `uint32` to a `[]uint32` from the pooled batch. All state structs are value types. Use `binary.LittleEndian.PutUint32` into pre-allocated regions. | P2 |
| 13 | `render-sys/src/cmd/mi.rs` | Memory/instruction commands (MI_NOOP, MI_BATCH_BUFFER_START) | 259 | **Low** — small fixed-size writes | Direct `uint32` writes into batch slice. No heap allocation. | P2 |
| 14 | `render-sys/src/cmd/pipeline.rs` | Pipeline configuration command builders | 210 | **Low** — emits fixed command sequences | Same as `mi.rs`. | P2 |
| 15 | `render-sys/src/cmd/primitive.rs` | Primitive topology setup (3DPRIMITIVE command) | 206 | **Low** — fixed 7-dword command | Single `copy()` into batch slice. | P2 |
| 16 | `render-sys/src/pm4.rs` | AMD PM4 packet format encoder | 599 | **Medium** — builds variable-length packets | Use pre-allocated `[]uint32` packet buffer (cap 256 dwords, covers largest PM4 packet). Reset and reuse per packet via `sync.Pool`. | P2 |
| 17 | `render-sys/src/pipeline.rs` | Pre-baked pipeline state objects (fill, textured, SDF text, shadow) | 562 | **Low** — computed once at startup, read-only thereafter | Singleton `var` at package level, initialized in `init()`. Immutable after creation → zero GC pressure at runtime. | P3 |
| 18 | `render-sys/src/shader.rs` | Shader frontend — WGSL/GLSL parsing via naga | 538 | **High** — naga builds a full IR graph (`Module`, `Type`, `Expression` trees) | Replace naga with a purpose-built WGSL parser in Go: tokenizer → AST → typed IR. Use `arena`-style allocation: single `[]Node` backing slice with index references (not pointers). Pre-size to 4096 nodes. Validate inline during parsing (no second pass). | P4 |
| 19 | `render-sys/src/eu/mod.rs` | Intel EU shader compiler entry point | 532 | **High** — orchestrates lowering, regalloc, encoding | `EUCompiler` as stateful struct with pre-allocated workspace buffers: `[]Instruction` (cap 2048), `[]byte` (cap 64 KB for output binary). Reuse across compilations via `Reset()` method. | P4 |
| 20 | `render-sys/src/eu/lower.rs` | naga IR → Intel EU instruction lowering | 2,845 | **High** — largest file; many temporary IR nodes, expression visitors | **Arena allocation pattern**: define `type Arena[T any] struct { data []T; len int }` with `Alloc() *T` that bumps `len`. All lowered instructions live in a single arena, freed in bulk. Avoid `append()` on the hot path — pre-size all slices. Use `//go:nosplit` on leaf helpers that must not grow the goroutine stack. | P4 |
| 21 | `render-sys/src/eu/encoding.rs` | EU instructions → 128-bit binary encoding | 425 | **Medium** — bitfield packing into `[16]byte` | Use fixed `[16]byte` arrays (stack-allocated). Write fields with bit-shift arithmetic. Output into pre-allocated `[]byte` kernel buffer. Zero heap allocation per instruction. | P4 |
| 22 | `render-sys/src/eu/instruction.rs` | Intel EU ISA instruction definitions | 473 | **Low** — type/constant definitions | Go `const` blocks and typed enums (`type Opcode uint16`). No allocation. | P4 |
| 23 | `render-sys/src/eu/regalloc.rs` | Intel EU register allocator (linear scan) | 151 | **Medium** — interval tracking, spill lists | Pre-allocate `[]Interval` to function size. Use bitset (`[4]uint64` = 256 GRF registers) for live-set instead of `map`. Spill list as `[]uint16` with pre-set cap. | P4 |
| 24 | `render-sys/src/eu/types.rs` | Compiler type definitions (registers, regions, etc.) | 381 | **Low** — pure type defs | Value types in Go. No allocation. | P4 |
| 25 | `render-sys/src/rdna/mod.rs` | AMD RDNA shader compiler entry point | 222 | **High** — similar profile to EU compiler | Same workspace-reuse strategy as EU compiler. `RDNACompiler` struct with pre-allocated `[]Instruction` and `[]byte` output buffers. | P5 |
| 26 | `render-sys/src/rdna/lower.rs` | naga IR → RDNA instruction lowering | 340 | **High** — expression tree traversal, temporary allocations | Same arena pattern as `eu/lower.rs`. Smaller scale (~340 LOC vs ~2,845) reduces risk. | P5 |
| 27 | `render-sys/src/rdna/encoding.rs` | RDNA instructions → binary encoding | 320 | **Medium** — 32-bit/64-bit instruction packing | Fixed `[8]byte` arrays for 64-bit instructions. Append into pre-allocated output buffer. | P5 |
| 28 | `render-sys/src/rdna/instruction.rs` | AMD RDNA ISA instruction definitions | 233 | **Low** — type/constant defs | Go `const` blocks. No allocation. | P5 |
| 29 | `render-sys/src/rdna/regalloc.rs` | RDNA register allocator | 148 | **Medium** — VGPR/SGPR tracking | Bitset-based live-set (RDNA3: 256 VGPRs = `[4]uint64`). Pre-allocated interval slice. | P5 |
| 30 | `render-sys/src/rdna/types.rs` | RDNA type definitions | 162 | **Low** — pure type defs | Value types. No allocation. | P5 |
| 31 | `render-sys/src/shaders.rs` | Embedded WGSL shader source strings | 76 | **Low** — compile-time constants | Go `const` block or `//go:embed` directives. | P0 |
| 32 | `render-sys/build.rs` | Cargo build script (shader pre-compilation) | 75 | N/A | Replace with `go:generate` directive or `init()` compilation. | P6 |
| 33 | `render-sys/Cargo.toml` | Rust package manifest | — | N/A | Delete after migration complete. | P6 |
| 34 | `render-sys/src/gpu_test.rs` | Rust-side GPU integration tests | 279 | N/A | Re-implement as Go `_test.go` files in corresponding packages. | P3 |
| 35 | `render-sys/tests/shader_compile.rs` | Shader compilation integration test | 142 | N/A | Port to Go test in `internal/render/shader_test.go`. | P5 |

---

## FFI Boundaries and Elimination Plan

### Current FFI Surface

Three Go files form the CGO bridge:

| Go File | C Functions Called | Elimination Strategy |
|---|---|---|
| `internal/render/binding.go` (448 LOC) | `render_add`, `render_version`, `render_detect_gpu`, `render_submit_batch`, `render_create_context`, `render_destroy_context`, `render_submit_shader_batch` | Replace each `C.render_*()` call with a direct Go function call to the new pure-Go `internal/gpu/` packages. Remove all `// #cgo` directives and `import "C"`. |
| `internal/render/dmabuf.go` (258 LOC) | `buffer_allocator_create`, `buffer_allocator_destroy`, `buffer_allocate`, `buffer_export_dmabuf`, `buffer_get_info`, `buffer_get_handle`, `buffer_destroy`, `buffer_mmap`, `buffer_munmap` | Replace `C.BufferAllocator` / `C.Buffer` opaque handles with Go struct pointers. `mmap`/`munmap` via `unix.Mmap()`/`unix.Munmap()`. |
| `internal/render/shader.go` (96 LOC) | `render_compile_shader`, `render_shader_free` | Replace with direct call to Go shader compiler. `ShaderBinary.Data` becomes a `[]byte` — no manual free needed (GC handles it). |

### Supporting Files to Remove

| File | Reason |
|---|---|
| `internal/render/dl_find_object_stub.c` | GCC 14 / musl workaround — unnecessary without Rust static linking |
| `internal/render/dl_find_object_stub.o` | Compiled stub object |

### New Package Structure (Proposed)

```
internal/
  gpu/
    detect/        ← detect.rs (GPU probing)
    drm/           ← drm.rs (ioctl wrapper)
    i915/          ← i915.rs (Intel i915 driver)
    xe/            ← xe.rs (Intel Xe driver)
    amd/           ← amd.rs (AMDGPU driver)
    alloc/         ← allocator.rs, slab.rs (buffer allocator)
    batch/         ← batch.rs, cmd/* (command building)
    surface/       ← surface.rs (surface management)
    submit/        ← submit.rs (command dispatch)
    pm4/           ← pm4.rs (AMD packet format)
    pipeline/      ← pipeline.rs (pre-baked states)
    shader/
      wgsl/        ← shader.rs (WGSL parser, replaces naga)
      eu/          ← eu/* (Intel EU backend)
      rdna/        ← rdna/* (AMD RDNA backend)
  render/
    binding.go     ← rewritten: calls internal/gpu/* directly, no CGO
    dmabuf.go      ← rewritten: uses internal/gpu/alloc + unix.Mmap
    shader.go      ← rewritten: uses internal/gpu/shader/*
```

---

## Key Risks and Blockers

### 1. Shader Compiler GC Pressure (HIGH RISK)
**Risk:** The shader compiler (eu/lower.rs alone is 2,845 LOC) performs graph-intensive IR transformations. A naive Go port would create thousands of small heap objects per compilation, causing GC pauses.

**Mitigation:**
- Arena-based allocation: `type Arena[T any] struct { data []T; len int }` — bulk-allocate, index-reference, bulk-free.
- Pre-size all IR slices to typical shader size (512 expressions, 2048 instructions).
- Compiler struct carries workspace buffers, reused across compilations via `Reset()`.
- **Benchmark gate:** Must demonstrate ≤10% regression vs Rust on reference shaders before merging.

### 2. naga Replacement (HIGH RISK)
**Risk:** Rust uses the `naga` crate (a mature WGSL/GLSL parser and validator). No equivalent Go library exists.

**Mitigation:**
- Write a purpose-built WGSL parser targeting only the subset used by wain's shaders (vertex + fragment stages, no compute).
- The wain shader set is small and known (`render-sys/shaders/`), so the parser can be scoped narrowly.
- Validate against all existing WGSL sources in the repo as acceptance test.
- **Fallback:** If full WGSL parsing proves too costly, pre-compile shaders at build time via `go:generate` and ship binary blobs (trades flexibility for simplicity).

### 3. ioctl Struct Layout Correctness (MEDIUM RISK)
**Risk:** DRM ioctl structs must match kernel ABI exactly (field sizes, alignment, padding). Rust's `#[repr(C)]` guarantees layout; Go struct layout is not guaranteed to match.

**Mitigation:**
- Use `encoding/binary.Read/Write` with explicit `binary.LittleEndian` for all ioctl structs.
- Alternatively, define structs as `[N]byte` and use `unsafe.Pointer` casting with `//go:nosplit` to avoid overhead.
- Add `_test.go` files that assert `unsafe.Sizeof(IoctlStruct{})` matches expected kernel sizes.
- Reference `golang.org/x/sys/unix` for existing DRM ioctl patterns.

### 4. mmap Lifecycle Management (MEDIUM RISK)
**Risk:** GPU buffers mapped via `mmap()` must not be collected or moved by the GC. Currently Rust manages this with raw pointers.

**Mitigation:**
- Use `unix.Mmap()` which returns `[]byte` backed by mmap'd memory — GC-safe (runtime knows about mmap regions).
- Wrap in `BufferHandle` with a finalizer (`runtime.SetFinalizer`) as a safety net for unmapping.
- Explicit `Munmap()` on the happy path; finalizer only for leak prevention.

### 5. Performance Regression in Batch Building (LOW–MEDIUM RISK)
**Risk:** Batch building appends dwords on every draw call — a hot path during frame rendering. `append()` may trigger GC if slices grow.

**Mitigation:**
- Pre-allocate batch buffer to 4 KB (1024 dwords), covering typical UI frames.
- Use `sync.Pool` to recycle `[]uint32` slices between frames.
- Profile with `GODEBUG=gctrace=1` to verify zero mid-frame GC pauses.

### 6. No Windows/macOS Support Needed
**Non-risk:** wain is Linux-only. All ioctl, DRM, mmap, Wayland/X11 code is Linux-specific. No cross-platform portability concerns for the GPU layer.

---

## Migration Checklist

### Phase 0 — Foundation (P0)
- [ ] Port `render-sys/src/drm.rs` → `internal/gpu/drm/` — DRM device open and ioctl wrapper using `unix.IoctlSetPointerInt`. Acceptance: `go vet` passes; unit test opens `/dev/dri/renderD128` (or mock fd) and calls `DRM_IOCTL_VERSION` successfully.
- [ ] Port `render-sys/src/detect.rs` → `internal/gpu/detect/` — GPU generation detection via i915/Xe/AMDGPU param ioctls. Acceptance: `DetectGPU("/dev/dri/renderD128")` returns correct `GpuGeneration` on Intel and AMD test machines; matches Rust output.
- [ ] Port `render-sys/src/shaders.rs` → `internal/gpu/shader/shaders.go` — embed WGSL source strings using `//go:embed`. Acceptance: all shader strings accessible; byte-for-byte identical to Rust constants.
- [ ] Add benchmark harness (`internal/gpu/detect/detect_bench_test.go`) — `BenchmarkDetectGPU` baseline for comparison with Rust FFI path. Acceptance: benchmark runs; results recorded in `testdata/bench-baseline.txt`.

### Phase 1 — Driver ioctls (P1)
- [ ] Port `render-sys/src/i915.rs` → `internal/gpu/i915/` — GEM create, mmap offset, execbuffer2, context create/destroy. Acceptance: struct sizes match kernel ABI (`unsafe.Sizeof` tests); `go vet` clean; integration test creates and destroys a GEM buffer on i915 hardware.
- [ ] Port `render-sys/src/xe.rs` → `internal/gpu/xe/` — VM create, exec-queue create/destroy, exec. Acceptance: struct size tests pass; integration test on Xe hardware creates exec queue and submits no-op batch.
- [ ] Port `render-sys/src/amd.rs` → `internal/gpu/amd/` — GEM create, VA management, context, chip info. Acceptance: struct size tests pass; integration test on AMD hardware allocates and frees a GEM buffer.
- [ ] Port `render-sys/src/submit.rs` → `internal/gpu/submit/` — driver-agnostic command submission dispatch. Acceptance: calls correct driver backend based on `GpuGeneration`; unit test with mock fd verifies dispatch logic.
- [ ] Add benchmarks for i915/Xe/AMD buffer create+destroy cycles. Acceptance: benchmarks run; ≤10% overhead vs Rust FFI path on each driver.

### Phase 2 — Buffer and Command Management (P2)
- [ ] Port `render-sys/src/allocator.rs` + `render-sys/src/slab.rs` → `internal/gpu/alloc/` — slab-based buffer allocator with `sync.Pool` for ioctl param structs and bitset free-list. Acceptance: allocate/free 1000 buffers without OOM; `go test -benchmem` shows ≤2 allocs/op on steady-state allocate+free.
- [ ] Port `render-sys/src/batch.rs` → `internal/gpu/batch/` — batch buffer builder with pre-allocated `[]uint32` from `sync.Pool`. Acceptance: build a 100-command batch; `go test -benchmem` shows 0 allocs/op after warmup.
- [ ] Port `render-sys/src/cmd/` (mod.rs, state.rs, mi.rs, pipeline.rs, primitive.rs) → `internal/gpu/batch/cmd/` — GPU command helpers. Acceptance: emit MI_NOOP, 3DSTATE_*, 3DPRIMITIVE into batch; binary output matches Rust reference output byte-for-byte.
- [ ] Port `render-sys/src/pm4.rs` → `internal/gpu/pm4/` — AMD PM4 packet encoder with pre-allocated packet buffer. Acceptance: encode DISPATCH_INDIRECT, WRITE_DATA packets; output matches Rust reference.
- [ ] Port `render-sys/src/surface.rs` → `internal/gpu/surface/` — surface management and framebuffer association. Acceptance: create surface, associate with DRM framebuffer; integration test on real hardware.
- [ ] Port `render-sys/src/pipeline.rs` → `internal/gpu/pipeline/` — pre-baked pipeline state objects (fill, textured, SDF, shadow). Acceptance: pipeline states are initialized once in `init()`; `go test -benchmem` shows 0 allocs after init.
- [ ] Add benchmarks for batch build (100 draw calls) and surface create. Acceptance: ≤10% regression vs Rust path.

### Phase 3 — Integration and Test Porting (P3)
- [ ] Port `render-sys/src/gpu_test.rs` → Go integration tests in `internal/gpu/*_test.go`. Acceptance: all ported tests pass on target hardware; test coverage ≥ Rust test coverage.
- [ ] Rewrite `internal/render/binding.go` — remove all `import "C"` and `// #cgo` directives; replace `C.render_*()` calls with direct Go calls to `internal/gpu/` packages. Acceptance: `go build` succeeds with `CGO_ENABLED=0`; all existing render tests pass.
- [ ] Rewrite `internal/render/dmabuf.go` — replace `C.BufferAllocator`/`C.Buffer` with Go structs backed by `internal/gpu/alloc`; use `unix.Mmap()`/`unix.Munmap()`. Acceptance: DMA-BUF export returns valid fd; mmap/munmap lifecycle test passes.
- [ ] Rewrite `internal/render/shader.go` — call Go shader compiler directly; remove `render_shader_free` (GC handles `[]byte`). Acceptance: compile all WGSL shaders in `render-sys/shaders/`; binary output matches Rust reference.

### Phase 4 — Intel EU Shader Compiler (P4)
- [ ] Port `render-sys/src/eu/types.rs` + `render-sys/src/eu/instruction.rs` → `internal/gpu/shader/eu/` — type definitions, opcode constants, register types. Acceptance: all Intel EU opcodes and register types defined; `go vet` clean.
- [ ] Port `render-sys/src/eu/regalloc.rs` → `internal/gpu/shader/eu/regalloc.go` — linear-scan register allocator with bitset live-sets (`[4]uint64`). Acceptance: allocate registers for test IR; output matches Rust reference allocation.
- [ ] Port `render-sys/src/eu/encoding.rs` → `internal/gpu/shader/eu/encoding.go` — 128-bit instruction encoding using `[16]byte` stack arrays. Acceptance: encode 10 reference instructions; binary output matches Rust byte-for-byte.
- [ ] Port `render-sys/src/eu/lower.rs` → `internal/gpu/shader/eu/lower.go` — IR lowering with arena allocation (`Arena[Instruction]`, pre-sized 2048). Acceptance: lower all WGSL shaders in `render-sys/shaders/`; instruction count matches Rust output ±0.
- [ ] Port `render-sys/src/eu/mod.rs` → `internal/gpu/shader/eu/compiler.go` — compiler orchestration with reusable workspace. Acceptance: full compile pipeline produces identical binaries to Rust for all test shaders.
- [ ] Add benchmark `BenchmarkEUCompile` — compile reference vertex + fragment shaders. Acceptance: ≤10% regression vs Rust; `GODEBUG=gctrace=1` shows no GC during compilation. **If regression exceeds 10%, document the gap and proposed further optimizations as a blocking issue.**

### Phase 5 — AMD RDNA Shader Compiler (P5)
- [ ] Port `render-sys/src/rdna/types.rs` + `render-sys/src/rdna/instruction.rs` → `internal/gpu/shader/rdna/` — RDNA type definitions and opcodes. Acceptance: all RDNA opcodes defined; `go vet` clean.
- [ ] Port `render-sys/src/rdna/regalloc.rs` → `internal/gpu/shader/rdna/regalloc.go` — VGPR/SGPR allocator with bitset (`[4]uint64`). Acceptance: matches Rust register assignments for test IR.
- [ ] Port `render-sys/src/rdna/encoding.rs` → `internal/gpu/shader/rdna/encoding.go` — 32/64-bit instruction encoding. Acceptance: binary output matches Rust byte-for-byte.
- [ ] Port `render-sys/src/rdna/lower.rs` → `internal/gpu/shader/rdna/lower.go` — IR lowering with arena allocation. Acceptance: lower all WGSL shaders; instruction count matches Rust.
- [ ] Port `render-sys/src/rdna/mod.rs` → `internal/gpu/shader/rdna/compiler.go` — RDNA compiler orchestration. Acceptance: full compile pipeline produces identical binaries for all test shaders.
- [ ] Port `render-sys/tests/shader_compile.rs` → `internal/gpu/shader/shader_compile_test.go`. Acceptance: all shader compile tests pass.
- [ ] Add benchmark `BenchmarkRDNACompile`. Acceptance: ≤10% regression vs Rust. **If regression exceeds 10%, document explicitly.**

### Phase 6 — WGSL Parser (replaces naga) (P4–P5 dependency)
- [ ] Implement WGSL tokenizer in `internal/gpu/shader/wgsl/tokenizer.go` — lexes WGSL source into tokens using `[]byte` slicing (zero-copy). Acceptance: tokenize all shaders in `render-sys/shaders/`; token stream matches expected output.
- [ ] Implement WGSL parser in `internal/gpu/shader/wgsl/parser.go` — builds typed IR using arena-allocated nodes (`[]Node` backing slice, index references). Acceptance: parse all shaders; AST node count matches naga IR node count ±5%.
- [ ] Implement WGSL validator in `internal/gpu/shader/wgsl/validate.go` — type-checks IR inline during parsing. Acceptance: reject known-bad shaders; accept all valid shaders.
- [ ] Add benchmark `BenchmarkWGSLParse`. Acceptance: parse + validate reference shader in ≤1ms; `go test -benchmem` shows ≤10 allocs/op (arena bulk allocation).

### Phase 7 — Cleanup and Build System (P6)
- [ ] Remove `render-sys/` directory entirely (all Rust source, Cargo.toml, Cargo.lock, build.rs, tests/). Acceptance: `find . -name '*.rs' -o -name 'Cargo.toml'` returns empty.
- [ ] Remove `internal/render/dl_find_object_stub.c` and `internal/render/dl_find_object_stub.o`. Acceptance: files deleted; no references remain.
- [ ] Update `Makefile` — remove `rust`, `check-musl-rust-target`, `check-cargo` targets; remove `CGO_LDFLAGS` referencing `librender_sys.a`; set `CGO_ENABLED=0` for all Go build targets. Acceptance: `make build` succeeds without Rust toolchain installed; `make check-deps` does not require `rustc`, `cargo`, or `musl-gcc`.
- [ ] Update `go.mod` — run `go mod tidy`; verify no new external dependencies added (all GPU code uses stdlib + `golang.org/x/sys/unix`). Acceptance: `go mod tidy` produces no diff; `go list -m all` shows no new modules.
- [ ] Update `README.md` — remove Rust toolchain requirements; update build instructions; update architecture diagram to reflect pure Go GPU backend. Acceptance: README accurately describes pure-Go build process.
- [ ] Remove `CGO_ENABLED=1` requirement from CI workflows (`.github/workflows/`). Acceptance: CI builds pass with `CGO_ENABLED=0`.
- [ ] Final static linkage verification: `go build -ldflags "-extldflags '-static'" -o bin/wain ./cmd/wain && ldd bin/wain` reports "not a dynamic executable". Acceptance: binary is fully static; file size is within 20% of previous Rust-linked binary.

### Phase 8 — Performance Validation (P6)
- [ ] Run full benchmark suite (`go test -bench=. -benchmem ./internal/gpu/...`) on Intel Gen12, Intel Xe, and AMD RDNA2 hardware. Acceptance: all benchmarks within 10% of Rust FFI baselines recorded in Phase 0–5.
- [ ] Profile GC behavior under load: render 1000 frames with `GODEBUG=gctrace=1`; verify no GC pause exceeds 1ms during frame rendering. Acceptance: GC trace log shows max pause ≤1ms; no mid-frame collections.
- [ ] **If any benchmark exceeds 10% regression:** file a blocking issue with component name, measured regression %, and proposed optimization (e.g., tighter arena sizing, `//go:nosplit` on hot functions, `unsafe.Slice` for zero-copy ioctl buffers). Acceptance: issue filed with reproduction steps and proposed fix.
- [ ] Validate overall application performance: run `widget-demo` and `gpu-ui-demo` end-to-end; measure frame time p99. Acceptance: p99 frame time ≤ previous Rust-backed build ± 10%.
