# Supported Hardware Matrix

This document details the GPU hardware, operating systems, and display servers supported by wain.

## GPU Support

### Intel GPUs

#### Supported Generations

| Generation | Architecture | Example Products | Driver | Status |
|-----------|--------------|------------------|--------|--------|
| **Gen9** | Skylake/Kaby Lake/Coffee Lake | HD Graphics 5xx, 6xx, UHD Graphics 6xx | i915 | ✅ Fully Supported |
| **Gen11** | Ice Lake | Iris Plus Graphics | i915 | ✅ Fully Supported |
| **Gen12** | Tiger Lake/Rocket Lake/Alder Lake | Iris Xe Graphics | i915 | ✅ Fully Supported |
| **Xe** | Alder Lake+ | Arc Graphics, Iris Xe (Xe driver) | xe | ✅ Fully Supported |

#### Chipset Detection

wain detects Intel GPU generation via `I915_GETPARAM` (i915) or `DRM_IOCTL_XE_DEVICE_QUERY` (Xe) and maps chipset IDs:

**Gen9 (Skylake/Kaby Lake/Coffee Lake):**
- 0x1916, 0x1926, 0x192B (Skylake GT2)
- 0x5916, 0x5926 (Kaby Lake GT2)
- 0x3E90, 0x3E92, 0x3E9B (Coffee Lake GT2)

**Gen11 (Ice Lake):**
- 0x8A52, 0x8A5A, 0x8A5C (Ice Lake GT2)

**Gen12 (Tiger Lake/Rocket Lake/Alder Lake):**
- 0x9A49, 0x9A40, 0x9A59 (Tiger Lake GT2)
- 0x4C8A, 0x4C90 (Rocket Lake)
- 0x4680, 0x4682, 0x4690, 0x4692, 0x46A0 (Alder Lake)

**Xe (Xe Driver):**
- Any chipset ID queried via Xe kernel driver

*For a complete chipset ID mapping, see `render-sys/src/detect.rs`.*

#### Required Kernel Features

- **i915 driver:** Linux kernel 4.17+ (for execbuffer2, GEM buffer management)
- **Xe driver:** Linux kernel 6.8+ (for VM binding, exec queues)
- **DRI3 + Present extensions** (X11)
- **zwp_linux_dmabuf_v1 protocol** (Wayland)

### AMD GPUs

#### Supported Generations

| Generation | Architecture | Example Products | Driver | Status |
|-----------|--------------|------------------|--------|--------|
| **RDNA1** | Navi 1x | RX 5000 Series | amdgpu | ✅ Fully Supported |
| **RDNA2** | Navi 2x | RX 6000 Series, Steam Deck APU | amdgpu | ✅ Fully Supported |
| **RDNA3** | Navi 3x | RX 7000 Series | amdgpu | ✅ Fully Supported |

#### Chipset Detection

wain detects AMD GPU generation via `AMDGPU_INFO_DEV_INFO` query and maps device families:

**RDNA1:**
- CHIP_NAVI10, CHIP_NAVI12, CHIP_NAVI14

**RDNA2:**
- CHIP_NAVI21, CHIP_NAVI22, CHIP_NAVI23, CHIP_NAVI24, CHIP_VANGOGH

**RDNA3:**
- CHIP_NAVI31, CHIP_NAVI32, CHIP_NAVI33

*For detailed family mapping, see `render-sys/src/detect.rs` and `render-sys/src/amd.rs`.*

#### Required Kernel Features

- **amdgpu driver:** Linux kernel 5.4+ (for GEM buffer management, CS submission)
- **DRI3 + Present extensions** (X11)
- **zwp_linux_dmabuf_v1 protocol** (Wayland)

### GCN (Pre-RDNA) AMD GPUs

**Status:** ❌ Not Supported

GCN ISA differs significantly from RDNA. Support could be added in the future but is not currently planned.

### NVIDIA GPUs

**Status:** ❌ Not Supported

NVIDIA's proprietary driver does not expose the necessary DRM kernel APIs for direct GPU command submission. nouveau (open-source driver) could theoretically be supported but is not currently planned.

### Software Fallback

**Status:** ✅ Fully Supported

If no compatible GPU is detected or GPU initialization fails, wain automatically falls back to a software rasterizer that produces pixel-identical output to the GPU backends.

**CPU Requirements:**
- x86-64 or ARM64 architecture
- No special CPU features required (SIMD optimizations not yet implemented)

---

## Operating System Support

### Linux

**Status:** ✅ Fully Supported

#### Minimum Kernel Version

- **Intel (i915):** Linux 4.17+
- **Intel (Xe):** Linux 6.8+
- **AMD (amdgpu):** Linux 5.4+
- **Software fallback:** Linux 3.10+ (any kernel with memfd_create syscall)

#### Required Kernel Features

- `/dev/dri/renderD*` nodes (DRM rendernode support)
- `memfd_create` syscall for shared memory allocation
- DMA-BUF infrastructure (CONFIG_DMA_SHARED_BUFFER)
- DRM GEM buffer management
- File descriptor passing via `SCM_RIGHTS` (Unix domain sockets)

#### Tested Distributions

| Distribution | Version | Status | Notes |
|-------------|---------|--------|-------|
| **Ubuntu** | 22.04 LTS | ✅ Tested | Default kernel 5.15+ |
| **Ubuntu** | 24.04 LTS | ✅ Tested | Default kernel 6.8+ |
| **Fedora** | 38+ | ✅ Tested | Default kernel 6.2+ |
| **Arch Linux** | Rolling | ✅ Tested | Latest kernel |
| **Debian** | 12 (Bookworm) | ✅ Expected | Kernel 6.1+ |
| **Alpine Linux** | 3.18+ | ✅ Expected | musl libc native |

*"Expected" means untested but should work based on kernel version.*

### FreeBSD / OpenBSD

**Status:** ❌ Not Supported

DRM ioctl interfaces differ from Linux. Porting would require significant effort.

### macOS / Windows

**Status:** ❌ Not Supported

No DRM/KMS infrastructure. Cross-platform windowing (e.g., via MoltenVK or DirectX) is not planned.

---

## Display Server Support

### Wayland

**Status:** ✅ Fully Supported

#### Required Protocols

**Core (mandatory):**
- `wl_display`, `wl_registry`, `wl_compositor`, `wl_surface`
- `wl_shm`, `wl_shm_pool` (software fallback)
- `xdg_wm_base`, `xdg_surface`, `xdg_toplevel` (window management)
- `wl_seat`, `wl_pointer`, `wl_keyboard` (input)

**GPU buffer sharing (required for GPU backends):**
- `zwp_linux_dmabuf_v1` (version 3+)

**Optional (gracefully degraded if unavailable):**
- `zxdg_decoration_manager_v1` (server-side decorations, falls back to client-side)
- `wl_output` (HiDPI scale detection, defaults to 1.0 if unavailable)
- `wl_data_device_manager` (clipboard/drag-and-drop, degraded without)

#### Tested Compositors

| Compositor | Version | Status | Notes |
|-----------|---------|--------|-------|
| **Sway** | 1.8+ | ✅ Tested | wlroots-based, most standards-compliant |
| **Weston** | 12.0+ | ✅ Tested | Reference compositor |
| **Mutter (GNOME)** | 45+ | ✅ Expected | GNOME Shell compositor |
| **KWin (KDE)** | 5.27+ | ✅ Expected | Plasma compositor |
| **Hyprland** | Latest | ✅ Expected | wlroots-based |

*"Expected" means untested but should work (follows protocol standards).*

### X11

**Status:** ✅ Fully Supported

#### Required Extensions

**Core (mandatory):**
- Core protocol v11+ (CreateWindow, MapWindow, CreateGC, PutImage)
- MIT-SHM extension (software rendering acceleration)

**GPU buffer sharing (required for GPU backends):**
- DRI3 extension (buffer import via DMA-BUF)
- Present extension (frame timing, swap control)

**Optional (gracefully degraded if unavailable):**
- SYNC extension (frame synchronization, degrades to busy-wait if unavailable)
- XFIXES extension (clipboard, degrades if unavailable)

#### Tested X Servers

| X Server | Version | Status | Notes |
|----------|---------|--------|-------|
| **Xorg** | 1.20+ | ✅ Tested | Standard X11 server |
| **Xorg** | 21.1+ | ✅ Tested | Latest stable |
| **Xwayland** | 22.1+ | ✅ Expected | X11 compatibility on Wayland |

### Mir / Other Display Servers

**Status:** ❌ Not Supported

Only Wayland and X11 protocols are implemented.

---

## Build Environment Support

### Compiler Toolchains

#### Go Compiler

- **Minimum version:** Go 1.24
- **CGO required:** Yes (links to Rust static library)
- **Architectures:** x86-64, ARM64

#### Rust Compiler

- **Minimum version:** Rust 1.75+ (stable channel)
- **Target triple:** `x86_64-unknown-linux-musl` (primary), `aarch64-unknown-linux-musl` (ARM64)
- **Required cargo features:** None (default build)

#### C Compiler (for CGO)

- **musl-gcc** (primary): Static linking with musl libc
- **gcc** (fallback): Requires `-static` and musl headers
- **clang** (experimental): Untested but should work with musl

### Static Linking

wain produces **fully static binaries** with zero dynamic dependencies:

```bash
$ ldd bin/wain
        not a dynamic executable
```

**Required for static linking:**
- Rust compiled with `target = "x86_64-unknown-linux-musl"` (produces `.a` staticlib)
- Go compiled with `CGO_ENABLED=1 CC=musl-gcc` and `-extldflags '-static'`
- GCC 14+ users: requires `dl_find_object` stub (included in Makefile)

---

## Performance Characteristics

### GPU Backend Performance

**Target:** <2ms GPU frame time for typical UI workload

**Typical workload:**
- 200-500 rectangles (buttons, panels, backgrounds)
- 50-100 text runs (labels, buttons, input fields)
- 5-10 shadows (box-shadow blur)
- 1-2 gradients (backgrounds)

**Measured performance (on i7-10710U, Intel UHD Graphics):**
- Simple UI (50 rects, 10 text runs): ~0.3ms GPU time
- Complex UI (500 rects, 100 text runs, 10 shadows): ~1.5ms GPU time

**Bottlenecks:**
- Vertex throughput: Limited by vertex buffer upload (mmap write)
- Texture atlas: SDF glyph cache miss rate
- Overdraw: Multiple overlapping translucent layers

### Software Backend Performance

**Target:** 16ms frame time @ 1080p (60 FPS)

**Measured performance (on i7-10710U, software rendering):**
- Simple UI (50 rects, 10 text runs): ~2ms CPU time
- Complex UI (500 rects, 100 text runs, 10 shadows): ~12ms CPU time

**Bottlenecks:**
- Box shadow blur (separable Gaussian on CPU)
- Anti-aliased curve rasterization
- Memory bandwidth (framebuffer writes)

**Note:** SIMD optimizations (AVX2/NEON) are not yet implemented. Performance could improve 2-4× with SIMD.

---

## Unsupported Configurations

The following configurations are explicitly **not supported**:

1. **32-bit architectures** (x86, ARM32, etc.)
   - Reason: Not tested, may have pointer size issues in C ABI

2. **Big-endian architectures** (POWER, SPARC, etc.)
   - Reason: Wire protocols assume little-endian byte order

3. **Non-Linux kernels** (FreeBSD, OpenBSD, macOS, Windows)
   - Reason: DRM/KMS APIs are Linux-specific

4. **Dynamic linking** (glibc, system libraries)
   - Reason: Project design mandates fully static binaries

5. **NVIDIA proprietary driver**
   - Reason: No DRM kernel API access (CUDA/Vulkan API only)

6. **Pre-RDNA AMD GPUs** (GCN architecture)
   - Reason: Different ISA, not implemented

7. **Intel GPUs older than Gen9** (Haswell, Broadwell, etc.)
   - Reason: Command encoding differs (3DSTATE formats changed in Gen9)

---

## Testing Matrix

wain's CI pipeline tests the following configurations:

| Test | GPU | Driver | Display Server | OS | Status |
|------|-----|--------|----------------|----|----|
| Build | N/A | N/A | N/A | Ubuntu 24.04 | ✅ Passing |
| Static Linking | N/A | N/A | N/A | Ubuntu 24.04 | ✅ Passing |
| Unit Tests (Go) | Software | N/A | N/A | Ubuntu 24.04 | ✅ Passing |
| Unit Tests (Rust) | Software | N/A | N/A | Ubuntu 24.04 | ✅ Passing |
| Integration Tests | Software | N/A | N/A | Ubuntu 24.04 | ✅ Passing |
| Screenshot Tests | Software | N/A | N/A | Ubuntu 24.04 | ✅ Passing |
| GPU Tests (Intel) | Intel UHD | i915 | X11 | Manual | ⚠️ Manual |
| GPU Tests (AMD) | RDNA2 | amdgpu | Wayland | Manual | ⚠️ Manual |

**Manual tests** require physical hardware and are run on-demand before releases.

---

## Recommended Hardware Configurations

### Minimum Configuration

- **CPU:** x86-64 or ARM64, 2+ cores
- **RAM:** 256 MB (software rendering only)
- **GPU:** Intel Gen9+ or AMD RDNA1+
- **Display:** X11 or Wayland compositor with DRI3/dmabuf support
- **OS:** Linux kernel 4.17+ (i915) or 5.4+ (amdgpu)

### Recommended Configuration

- **CPU:** x86-64, 4+ cores, 2.0 GHz+
- **RAM:** 512 MB
- **GPU:** Intel Gen12 (Iris Xe) or AMD RDNA2+
- **Display:** Wayland compositor (Sway, Mutter, KWin)
- **OS:** Linux kernel 6.1+ (LTS)

### Development Configuration

- **CPU:** x86-64, 8+ cores (for parallel compilation)
- **RAM:** 4 GB+ (for Rust compilation)
- **GPU:** Intel or AMD with kernel DRM debug enabled
- **Display:** X11 + Wayland (test both protocols)
- **OS:** Linux kernel 6.8+ (latest features)

---

## Validation Checklist

Before deploying on a new system, verify:

- [ ] `/dev/dri/renderD128` exists and is readable
- [ ] `memfd_create` syscall is available (Linux 3.17+)
- [ ] Wayland socket at `$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY` or X11 at `$DISPLAY`
- [ ] GPU driver module loaded (`lsmod | grep i915` or `lsmod | grep amdgpu`)
- [ ] DRI3 extension available (X11): `xdpyinfo | grep DRI3`
- [ ] dmabuf protocol available (Wayland): compositor documentation or trial run

**Quick validation test:**

```bash
# Build wain
make build

# Verify static linking
ldd bin/wain  # should output "not a dynamic executable"

# Run auto-detection demo (falls back to software if GPU unavailable)
./bin/auto-render-demo
```

If the demo runs successfully, your system is supported.

---

## Getting Help

If your hardware configuration is not listed or you encounter issues:

1. Check `/dev/dri/` for render nodes:
   ```bash
   ls -l /dev/dri/
   ```

2. Check GPU driver and kernel version:
   ```bash
   lsmod | grep -E "i915|xe|amdgpu"
   uname -r
   ```

3. Run wain with debug logging (when implemented in future releases)

4. Report hardware details in a GitHub issue with:
   - GPU model and driver version
   - Linux kernel version and distribution
   - Display server (X11 or Wayland, compositor name)
   - Output of `lspci | grep VGA`
   - Output of `dmesg | grep -E "i915|xe|amdgpu" | tail -20`

---

**Last Updated:** 2026-03-08  
**wain Version:** Phase 8 (current development milestone)
