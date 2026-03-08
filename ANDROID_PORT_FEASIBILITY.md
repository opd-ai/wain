# Android Port Feasibility Report — opd-ai/wain

## Executive Summary

**Verdict: Partially feasible, with significant effort.** The `opd-ai/wain` codebase is deeply coupled to Linux kernel APIs at every layer—DRM ioctls for GPU buffer management, Wayland/X11 for display protocols, `memfd_create` for shared memory, and `SCM_RIGHTS` for fd-passing. The software rasterizer (`internal/raster/`) and shader IR pipeline (naga) are fully portable as-is. However, the GPU rendering path requires a near-complete rewrite to target Android's `AHardwareBuffer`/Vulkan APIs, and the entire display protocol layer must be replaced with Android `NativeWindow`/`SurfaceFlinger` integration. **Estimated total effort: 20–28 person-weeks**, with the highest risk being the GPU command submission replacement (currently raw Intel EU/AMD RDNA machine code generation bypassing any abstraction layer).

## Layer-by-Layer Analysis

### 1. Rust Rendering Library (`render-sys/`)

The Rust library (~9,926 LOC code, ~14,433 LOC total across 32 files) is the deepest platform-coupling layer. Every file except the shader frontend touches Linux-specific APIs.

| Component | File(s) | Linux Dependency | Android Replacement | Effort | Risk |
|-----------|---------|-----------------|---------------------|--------|------|
| DRM device access | `drm.rs` | `/dev/dri/renderD*` open, `DRM_IOCTL_MODE_CREATE_DUMB`, `DRM_IOCTL_GEM_CLOSE`, `DRM_IOCTL_PRIME_HANDLE_TO_FD` via `nix::libc::ioctl` | `AHardwareBuffer_allocate()` / Vulkan `VkDeviceMemory` — no DRM device nodes on Android (blocked by SELinux) | **High** (full rewrite) | 🔴 High — foundational dependency |
| i915 GPU driver | `i915.rs` (~616 LOC) | `I915_GEM_CREATE`, `I915_GEM_MMAP_OFFSET`, `I915_GEM_SET_TILING`, `I915_GEM_WAIT`, `I915_GEM_CONTEXT_CREATE`, `I915_GEM_CONTEXT_DESTROY`, `I915_GEM_EXECBUFFER2` — 7 ioctl wrappers | **No direct replacement.** Intel i915 is not present on Android devices. Must replace with Vulkan command submission (`vkQueueSubmit`) or Android GPU-vendor HAL. | **Very High** (full rewrite) | 🔴 Critical — entire Intel path is inapplicable |
| Xe GPU driver | `xe.rs` (~733 LOC) | `DRM_XE_DEVICE_QUERY`, `DRM_XE_GEM_CREATE`, `DRM_XE_VM_CREATE/DESTROY/BIND`, `DRM_XE_EXEC`, `DRM_XE_EXEC_QUEUE_CREATE/DESTROY` — 8 ioctl wrappers | **No direct replacement.** Intel Xe not present on Android. Same as i915: Vulkan abstraction required. | **Very High** (full rewrite) | 🔴 Critical |
| AMD GPU driver | `amd.rs` (~850 LOC) | `AMDGPU_GEM_CREATE`, `AMDGPU_GEM_MMAP`, `AMDGPU_GEM_WAIT_IDLE`, `AMDGPU_CTX`, `AMDGPU_CS`, `AMDGPU_WAIT_CS`, `AMDGPU_GEM_VA`, `AMDGPU_INFO` — 8 ioctl wrappers | **No direct replacement.** AMD GPUs on Android are rare (some Samsung Exynos). Vulkan abstraction required. | **Very High** (full rewrite) | 🔴 Critical |
| AMD PM4 command encoding | `pm4.rs` (~587 LOC) | AMD PM4 command packet encoding (SET_CONTEXT_REG, DRAW_INDEX_AUTO, EVENT_WRITE, SURFACE_SYNC, ACQUIRE_MEM) — targets AMDGPU kernel driver CS submission | Vulkan render passes + command buffers replace raw PM4 encoding entirely | **Very High** (full rewrite) | 🔴 Critical — ISA-level GPU commands |
| Intel 3D pipeline commands | `cmd/` (state.rs, pipeline.rs, mi.rs, primitive.rs) | Intel MI_NOOP, PIPE_CONTROL, 3DSTATE_VS/PS, 3DPRIMITIVE — targets i915/Xe execbuffer2 | Vulkan pipeline state objects (`VkGraphicsPipeline`, `VkRenderPass`) | **Very High** (full rewrite) | 🔴 Critical — ISA-level GPU commands |
| Intel EU shader compiler | `eu/` (mod.rs, instruction.rs, encoding.rs, regalloc.rs, lower.rs, types.rs) | Pure Rust, but outputs Intel Gen9/11/12 EU machine code binary | **Unusable on Android.** Android GPUs (Adreno, Mali, PowerVR) have different ISAs. Must use Vulkan SPIR-V pipeline (naga can output SPIR-V). | **Very High** (replace with SPIR-V output) | 🔴 Critical — wrong target ISA |
| AMD RDNA shader compiler | `rdna/` (mod.rs, instruction.rs, regalloc.rs, lower.rs, types.rs, encoding.rs) | Pure Rust, but outputs AMD RDNA VOP1/VOP2/VOP3/FLAT instructions | **Unusable on Android** (same as EU backend). Replace with SPIR-V. | **Very High** (replace) | 🔴 Critical |
| Buffer allocator | `allocator.rs` (~313 LOC) | Uses `DrmDevice` + driver-specific GEM ioctls + `nix::libc::mmap` for CPU mapping | `AHardwareBuffer` + `AHardwareBuffer_lock()`/`unlock()` for CPU mapping, or Vulkan `vkMapMemory` | **High** (rewrite allocator abstraction) | 🟡 Medium |
| Slab sub-allocator | `slab.rs` | Pure Rust allocation logic on top of `allocator.rs` | **Portable** — only needs the underlying allocator to be replaced | **Low** (adaptation) | 🟢 Low |
| GPU detection | `detect.rs` (~127 LOC) | Reads `/dev/dri/renderD128`, queries chipset via i915/Xe/AMDGPU ioctls | `vkEnumeratePhysicalDevices` + `vkGetPhysicalDeviceProperties` | **Medium** (rewrite) | 🟡 Medium |
| Batch buffer builder | `batch.rs` (~276 LOC) | Builds command streams for i915 execbuffer2 / Xe exec | Vulkan command buffer recording (`vkBeginCommandBuffer`, etc.) | **High** (rewrite) | 🟡 Medium |
| Pipeline state | `pipeline.rs` (~561 LOC) | Pre-baked Intel 3D pipeline state configurations | `VkGraphicsPipelineCreateInfo` + Vulkan pipeline objects | **High** (rewrite) | 🟡 Medium |
| Surface state | `surface.rs` (~608 LOC) | Intel RENDER_SURFACE_STATE / SAMPLER_STATE encoding | Vulkan descriptor sets + `VkImageView` + `VkSampler` | **High** (rewrite) | 🟡 Medium |
| Shader frontend (naga) | `shader.rs` (~398 LOC) | Pure Rust, uses `naga` crate for WGSL/GLSL → IR parsing & validation | **Portable as-is.** naga is pure Rust with no platform deps. Add `spv-out` feature for SPIR-V output on Android. | **None** | 🟢 None |
| Compiled shaders | `shaders.rs`, `shaders/*.wgsl` | WGSL sources (7 UI shaders) — platform-independent | **Portable as-is.** WGSL → SPIR-V via naga for Vulkan consumption. | **None** | 🟢 None |
| `nix` crate usage | `Cargo.toml` | `nix = "0.27"` with `ioctl` feature — used for `nix::libc::ioctl()`, `nix::libc::mmap()`, `nix::libc::munmap()`, `nix::request_code_readwrite!` macros | Android NDK libc provides `ioctl`, `mmap`, `munmap` natively. However, the DRM ioctls themselves don't exist on Android. Replace with Android NDK equivalents (`AHardwareBuffer_*`, Vulkan). | **High** (remove nix, use Android NDK FFI) | 🟡 Medium |
| C ABI exports | `lib.rs` (~574 LOC) | Exports like `render_detect_gpu`, `buffer_allocator_create`, `buffer_mmap`, `render_submit_batch`, `render_create_context` — all call DRM internals | Must be rewritten to wrap Vulkan/AHardwareBuffer backends | **High** (rewrite exports) | 🟡 Medium |

### 2. Display Protocol Layer

| Component | Package | Linux Dependency | Android Replacement | Effort | Risk |
|-----------|---------|-----------------|---------------------|--------|------|
| Wayland wire format | `internal/wayland/wire/` | Pure Go binary marshaling — no platform deps beyond byte encoding | **Bypass entirely** — not needed on Android | N/A (excluded) | 🟢 None |
| Wayland socket + SCM_RIGHTS | `internal/wayland/socket/` | Unix domain socket with `syscall.SendmsgN`, `syscall.Recvmsg`, `syscall.SCM_RIGHTS` fd passing | **Bypass entirely** — Android uses Binder IPC, not Wayland sockets | N/A (excluded) | 🟢 None |
| Wayland client (Display, Registry, Compositor, Surface) | `internal/wayland/client/` | Connects to `$WAYLAND_DISPLAY` Unix socket | **Replace** with Android `ANativeWindow` via NDK (`ANativeWindow_fromSurface`) | **High** (new backend) | 🟡 Medium |
| Wayland SHM (memfd) | `internal/wayland/shm/` | `memfd_create` syscall (hardcoded SYS number 319 for x86_64), `syscall.Mmap` | Android supports `memfd_create` from API 30+ (Android 11+). `ASharedMemory_create()` is the NDK-preferred alternative (API 26+). Syscall number differs on ARM64 (279). | **Medium** (use `ASharedMemory_create` or fix syscall number) | 🟡 Medium |
| Wayland XDG shell | `internal/wayland/xdg/` | XDG shell protocol (window management) | **Bypass entirely** — Android window management via `Activity`/`NativeActivity` | N/A (excluded) | 🟢 None |
| Wayland input (Seat, Pointer, Keyboard) | `internal/wayland/input/` | wl_seat protocol events | **Replace** with Android `AInputQueue`, `AInputEvent` via NDK | **High** (new backend) | 🟡 Medium |
| Wayland DMA-BUF | `internal/wayland/dmabuf/` | `zwp_linux_dmabuf_v1` protocol for GPU buffer sharing with compositor | **Replace** with `AHardwareBuffer` sharing via `ANativeWindow_setBuffersGeometry` + Vulkan swapchain | **High** (rewrite) | 🟡 Medium |
| Wayland data device | `internal/wayland/datadevice/` | Clipboard/DnD protocol | **Replace** with Android `ClipboardManager` via JNI | **Medium** | 🟢 Low |
| Wayland output | `internal/wayland/output/` | wl_output display properties | **Replace** with Android `AConfiguration` / `DisplayMetrics` | **Low** | 🟢 Low |
| X11 wire format | `internal/x11/wire/` | Pure Go X11 binary protocol | **Bypass entirely** — not needed on Android | N/A (excluded) | 🟢 None |
| X11 client | `internal/x11/client/` | Connects to `/tmp/.X11-unix/X0` Unix socket, X11 auth | **Bypass entirely** | N/A (excluded) | 🟢 None |
| X11 events | `internal/x11/events/` | X11 event types (KeyPress, ButtonPress, etc.) | **Bypass entirely** | N/A (excluded) | 🟢 None |
| X11 GC/PutImage | `internal/x11/gc/` | X11 graphics context operations | **Bypass entirely** | N/A (excluded) | 🟢 None |
| X11 MIT-SHM | `internal/x11/shm/` | System V shared memory (`shmget`, `shmat`, `shmdt`, `shmctl`) — Linux-specific IPC constants | **Bypass entirely** | N/A (excluded) | 🟢 None |
| X11 DRI3 | `internal/x11/dri3/` | DRI3 extension (DMA-BUF fd passing to X server) | **Bypass entirely** | N/A (excluded) | 🟢 None |
| X11 Present | `internal/x11/present/` | Present extension (frame sync) | **Bypass entirely** | N/A (excluded) | 🟢 None |
| X11 DPI | `internal/x11/dpi/` | DPI detection via X11 screen dimensions | **Replace** with Android density APIs | **Low** | 🟢 Low |
| X11 Selection | `internal/x11/selection/` | X11 selection/clipboard protocol | **Bypass entirely** | N/A (excluded) | 🟢 None |

### 3. Build System & Toolchain

| Concern | Current Approach | Android Approach | Effort |
|---------|-----------------|------------------|--------|
| C compiler | `musl-gcc` for static linking with musl libc | Android NDK Clang (`aarch64-linux-android33-clang`). Bionic libc is the only option. No musl on Android. | **Medium** — must switch entire toolchain |
| Static linking | `-extldflags '-static'` produces zero-dependency binary via musl | **Not feasible** on Android. Apps must dynamically link against Bionic libc, `libandroid.so`, `libvulkan.so`, etc. Shared library (`.so`) output is required for `NativeActivity`. | **High** — fundamental architecture change from static binary to shared library |
| Rust target | `x86_64-unknown-linux-musl` / `aarch64-unknown-linux-musl` | `aarch64-linux-android` (or `armv7-linux-androideabi` for 32-bit). Rust supports Android targets via `rustup target add`. | **Low** — standard Rust cross-compilation |
| CGO cross-compilation | `CGO_ENABLED=1 CC=musl-gcc` | `CGO_ENABLED=1 CC=aarch64-linux-android33-clang GOOS=android GOARCH=arm64`. Go has first-class `GOOS=android` support. | **Medium** — requires NDK setup and Go mobile tooling |
| `dl_find_object_stub.c` | GCC 14 + musl compatibility stub (provides weak `_dl_find_object` symbol) | **Not needed.** Android uses Clang/LLVM toolchain (no GCC 14 linkage issue). Bionic libc provides its own unwinding support. | **None** — remove the stub |
| Cargo dependencies | `nix = "0.27"` (ioctl), `naga = "0.14"` (shader parsing) | `naga` works as-is. `nix` can compile for Android but DRM ioctls won't work — replace with `android-ndk-sys` or raw NDK FFI bindings. | **Medium** |
| Go `GOOS` build tags | No explicit build tags — all code assumes Linux | Must introduce `//go:build android` and `//go:build linux` tags to separate Android vs Linux display/input backends | **Medium** — architectural change across ~14 packages |

### 4. Portable Components (No Changes Needed)

The following components work as-is on Android with zero or minimal changes:

1. **naga shader frontend** (`render-sys/src/shader.rs`, ~398 LOC) — Pure Rust WGSL/GLSL parser and validator. Platform-independent.
2. **WGSL shader sources** (`render-sys/shaders/*.wgsl`, 7 files) — Text files compiled at build time. Fully portable.
3. **Software 2D rasterizer** (`internal/raster/`, ~1,877 LOC across 5 packages):
   - `core/` — ARGB8888 buffer operations, rectangles, rounded rects, lines
   - `curves/` — Bezier curves, arc fills
   - `composite/` — Alpha blending, image blitting, bilinear filtering
   - `effects/` — Box shadow, gradients
   - `text/` — SDF text rendering with embedded font atlas
   - All pure Go, no platform dependencies whatsoever.
4. **Display list abstraction** (`internal/raster/displaylist/`) — Renderer-agnostic command buffer. Portable.
5. **Software rendering consumer** (`internal/raster/consumer/`) — Executes display lists via software rasterizer. Portable.
6. **UI widget layer** (`internal/ui/`, ~1,503 LOC) — Retained/immediate mode UI widgets, layout engine. Pure Go, portable.
7. **Slab sub-allocator logic** (`render-sys/src/slab.rs`) — Pure Rust allocation math. Portable once base allocator is replaced.
8. **EU/RDNA register allocator logic** (`eu/regalloc.rs`, `rdna/regalloc.rs`) — Pure Rust data structures (though the output targets are unusable on Android, the algorithmic patterns could be reused).

### 5. Recommended Porting Strategy

**Phase A: Foundation (Weeks 1–4)**
1. **Introduce build-tag architecture** — Add `//go:build android` tags to all platform-specific Go packages. Create `internal/android/` package tree.
2. **Switch Rust target** — Add `aarch64-linux-android` to Cargo build and validate naga + slab compile cleanly.
3. **Replace static linking with shared library** — Convert output from static binary to `NativeActivity`-compatible `.so`. Modify Makefile/build system.
4. **Remove `dl_find_object_stub.c`** — Not needed with Android's Clang toolchain.

**Phase B: Android Display Backend (Weeks 5–8)**
5. **Implement `ANativeWindow` backend** — New Go package `internal/android/window/` wrapping `ANativeWindow_fromSurface`, `ANativeWindow_setBuffersGeometry`, `ANativeWindow_lock`, `ANativeWindow_unlockAndPost` via CGO.
6. **Implement Android input backend** — New Go package `internal/android/input/` wrapping `AInputQueue_getEvent`, `AInputEvent_getType` via CGO.
7. **Software rendering path on Android** — Wire software rasterizer → `ANativeWindow_lock` (CPU-rendered pixels directly to screen). This provides a working baseline before GPU.

**Phase C: Vulkan GPU Backend (Weeks 9–16)**
8. **Implement Vulkan rendering backend in Rust** — New module `render-sys/src/vulkan/` wrapping `ash` crate (Vulkan Rust bindings):
   - Device enumeration (`vkEnumeratePhysicalDevices`)
   - Swapchain creation (`VK_KHR_android_surface` + `VK_KHR_swapchain`)
   - Command buffer recording (replaces Intel batch.rs / AMD pm4.rs)
   - Pipeline state (replaces Intel cmd/ and pipeline.rs)
   - Descriptor sets (replaces surface.rs)
9. **SPIR-V shader output** — Add `spv-out` feature to naga dependency. Compile WGSL → SPIR-V at build time (or runtime). Replaces Intel EU and AMD RDNA backends entirely.
10. **`AHardwareBuffer` integration** — For zero-copy buffer sharing between Vulkan and Android compositor, use `VK_ANDROID_external_memory_android_hardware_buffer` extension.

**Phase D: Integration & Polish (Weeks 17–20)**
11. **Triple-buffered presentation** — Implement Vulkan swapchain-based frame pacing (replaces Wayland DMA-BUF / X11 Present flow).
12. **DPI/density handling** — Wire Android `AConfiguration_getDensity` to the UI layer.
13. **Testing on real devices** — Validate on Qualcomm Adreno, ARM Mali, and PowerVR GPUs.

### 6. Effort Estimate

| Layer | Scope | LOC Affected | Effort (person-weeks) |
|-------|-------|-------------|----------------------|
| **Build system & toolchain** | Makefile, Cargo.toml, build tags, NDK integration | ~500 LOC | 2 |
| **Android display backend** (ANativeWindow) | New Go packages for window, input, lifecycle | ~1,500 LOC new | 3–4 |
| **Software rendering integration** | Wire rasterizer to ANativeWindow | ~300 LOC | 1 |
| **Vulkan rendering backend** (Rust) | Replace drm.rs, i915.rs, xe.rs, amd.rs, pm4.rs, batch.rs, pipeline.rs, surface.rs, allocator.rs, cmd/ | ~5,000 LOC rewrite + ~3,000 LOC new | 8–12 |
| **SPIR-V shader pipeline** | Replace eu/ and rdna/ backends with naga SPIR-V output | ~2,000 LOC removed, ~500 LOC new | 2–3 |
| **C ABI export layer** | Rewrite lib.rs exports for Vulkan/AHB | ~600 LOC | 1–2 |
| **Go CGO bindings update** | Update binding.go, dmabuf.go for new Rust ABI | ~400 LOC | 1 |
| **Integration testing** | Multi-device testing, frame pacing, input validation | — | 2–4 |
| **Total** | | | **20–28 person-weeks** |

### 7. Blocking Risks

| Risk | Severity | Description | Mitigation |
|------|----------|-------------|------------|
| **No DRM device access on Android** | 🔴 **Showstopper** | Android SELinux policy completely blocks access to `/dev/dri/renderD*`. All 3 GPU driver modules (i915, Xe, AMDGPU) and ~2,200 LOC of ioctl wrappers are unusable. The entire GPU buffer management model must be replaced. | Must implement Vulkan or `AHardwareBuffer` path. No workaround. |
| **Intel/AMD GPU ISAs not present** | 🔴 **Showstopper** | Android devices use Qualcomm Adreno, ARM Mali, or PowerVR GPUs. The Intel EU backend (~2,800 LOC) and AMD RDNA backend (~1,500 LOC) generate machine code for ISAs that don't exist on any Android device. | Must use Vulkan SPIR-V pipeline. The Intel EU and AMD RDNA code cannot be reused. |
| **Static binary model incompatible** | 🟡 **Major** | wain's core design produces a single static binary with zero shared library dependencies. Android requires `.so` shared libraries loaded by `NativeActivity`, dynamically linked against `libandroid.so`, `libvulkan.so`, and Bionic libc. | Fundamental architectural change from static → dynamic linking. The entire build system must change. |
| **`memfd_create` syscall number** | 🟡 **Major** | `internal/wayland/shm/memfd.go` hardcodes x86_64 syscall number 319. ARM64 uses 279. Additionally, `memfd_create` is only available on Android API 30+ (Android 11+). | Use `ASharedMemory_create()` (NDK API 26+) or fix syscall number with build tags. |
| **Vulkan driver quality varies** | 🟡 **Moderate** | Android Vulkan drivers (especially on budget devices) have varying quality and feature support. Some may lack extensions needed for `AHardwareBuffer` import or external memory. | Target Vulkan 1.1 (API level 28+) as minimum. Test on multiple vendor GPUs. |
| **Go mobile toolchain maturity** | 🟢 **Low** | `GOOS=android` is supported but less battle-tested than Linux. CGO cross-compilation to Android NDK requires careful setup. | Well-documented path via `gomobile` project and NDK integration guides. |
| **X11 MIT-SHM System V IPC** | 🟢 **Non-issue** | `internal/x11/shm/` uses `shmget`/`shmat` (System V IPC). Android blocks SysV IPC. | Entire X11 layer can be excluded via build tags. Not needed on Android. |

---

## Appendix: Files Requiring Changes by Category

### Needs Full Rewrite (Linux-only, no code reusable on Android)
- `render-sys/src/drm.rs`
- `render-sys/src/i915.rs`
- `render-sys/src/xe.rs`
- `render-sys/src/amd.rs`
- `render-sys/src/pm4.rs`
- `render-sys/src/cmd/` (all files)
- `render-sys/src/eu/` (all files)
- `render-sys/src/rdna/` (all files)
- `render-sys/src/batch.rs`
- `render-sys/src/allocator.rs`
- `render-sys/src/detect.rs`
- `render-sys/src/pipeline.rs`
- `render-sys/src/surface.rs`
- `render-sys/src/lib.rs` (C ABI exports)

### Needs Adaptation
- `render-sys/Cargo.toml` (add Android targets, `ash` crate, `spv-out` feature)
- `internal/render/binding.go` (update CGO bindings)
- `internal/render/dmabuf.go` (replace DMA-BUF with AHardwareBuffer)
- `internal/wayland/shm/memfd.go` (fix syscall number or use ASharedMemory)
- `Makefile` (add Android build targets)
- `.envrc` (add Android NDK configuration)

### Portable As-Is
- `render-sys/src/shader.rs`
- `render-sys/src/shaders.rs`
- `render-sys/src/slab.rs`
- `render-sys/shaders/*.wgsl` (all 7 shader files)
- `internal/raster/` (all 5 packages, ~1,877 LOC)
- `internal/ui/` (all packages, ~1,503 LOC)
- `internal/raster/displaylist/`
- `internal/raster/consumer/`

### Excluded on Android (via build tags)
- `internal/wayland/` (all 9 packages, ~3,392 LOC)
- `internal/x11/` (all 9 packages, ~2,888 LOC)
