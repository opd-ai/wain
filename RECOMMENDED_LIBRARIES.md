# Recommended Third-Party Libraries for opd-ai/wain

## Summary

Analysis of the 14 subsystems in `opd-ai/wain` reveals that the majority of the codebase—Wayland/X11 protocol clients, the software rasterizer, input handling, and the UI layout engine—is well-served by custom, purpose-built implementations and should remain as-is. Five subsystems benefit from third-party library adoption: **text shaping** (`go-text/typesetting`), **font rasterization** (`golang.org/x/image/font`), **image decoding** (Go stdlib `image` + `golang.org/x/image`), **shader compilation** (`naga`), and **accessibility** (`godbus/dbus`). All five are pure Go or pure Rust with confirmed static compilation support—no dynamic linking required. Additionally, the existing `nix` dependency should be upgraded from v0.27 to v0.29+ for security and feature improvements. Adoption should be phased: `naga` first (Phase 4 critical path), then `golang.org/x/image` and `go-text/typesetting` (Phase 5–8 enhancements), and `godbus/dbus` last (Phase 8 accessibility).

## Per-Subsystem Recommendations

### Wayland Client Protocol
- **Current implementation:** Complete pure Go implementation in `internal/wayland/` (~4,000 LOC, 8 packages). Covers wire format marshaling/unmarshaling, fd passing via SCM_RIGHTS, wl_display, wl_registry, wl_compositor, wl_surface, wl_shm, xdg_wm_base, xdg_toplevel, wl_seat input devices, and zwp_linux_dmabuf_v1.
- **Recommended library:** Keep custom implementation
- **Static compilation:** N/A
- **Rationale:** No mature pure Go Wayland client library exists. The custom implementation is tightly integrated with the project's wire format, fd-passing, and display-list architecture. It avoids CGO for the protocol layer, which is a core design goal.
- **Risk / trade-offs:** Maintaining a custom Wayland client requires tracking protocol updates (e.g., new xdg-shell versions, fractional-scale). However, this project only implements the subset needed for a GUI toolkit, not the full protocol, so maintenance burden is bounded.

### X11 Client Protocol
- **Current implementation:** Complete pure Go implementation in `internal/x11/` (~2,500 LOC, 7 packages). Covers wire format, connection setup, authentication, CreateWindow, CreateGC, PutImage, MIT-SHM, DRI3, and Present extensions.
- **Recommended library:** Keep custom implementation
- **Static compilation:** N/A
- **Rationale:** `jezek/xgb` (v1.3.0, pure Go) is a viable alternative with broader X11 extension coverage, but the custom implementation is purpose-built for the project's specific needs (DRI3/Present for GPU buffer sharing, MIT-SHM for software path). Switching would require rearchitecting the protocol integration layer with no clear benefit given the current feature set.
- **Risk / trade-offs:** If additional X11 extensions are needed beyond the current set (e.g., XInput2 for advanced touch, Xrandr for multi-monitor), the custom implementation must be extended manually. At that point, `jezek/xgb` should be reconsidered.

### Input Handling (Keyboard/Mouse/Touch)
- **Current implementation:** Custom Go in `internal/wayland/input/` (keyboard, pointer, touch, keymap parsing) and `internal/x11/events/` (KeyPress, ButtonPress, MotionNotify). Includes a minimal xkb keymap parser for Wayland.
- **Recommended library:** Keep custom implementation
- **Static compilation:** N/A
- **Rationale:** The input handling is tightly coupled to the protocol layers. The Wayland input code implements wl_seat, wl_pointer, wl_keyboard, and wl_touch directly on the wire format. The xkb keymap parser handles common US/European layouts. A full `libxkbcommon` integration via CGO would enable all keyboard layouts globally but adds a C dependency that must be statically compiled with musl—possible but adds build complexity for a marginal gain at this stage.
- **Risk / trade-offs:** The minimal xkb parser may not support all keyboard layouts (CJK, Dvorak, custom Compose sequences). If internationalization becomes a priority, integrating `libxkbcommon` as a static C library (musl-compatible) should be revisited.

### Software 2D Rasterizer
- **Current implementation:** Custom pure Go in `internal/raster/` (~5 packages). Implements filled/rounded rectangles, anti-aliased lines, quadratic/cubic Bézier curves, arc fills, Porter-Duff alpha compositing, bilinear-filtered image blitting, box shadow (separable Gaussian blur), linear/radial gradients, and SDF text rendering.
- **Recommended library:** Keep custom implementation
- **Static compilation:** N/A
- **Rationale:** The rasterizer is purpose-built for UI primitives and integrates directly with the project's ARGB8888 buffer format, display list architecture, and SDF text pipeline. General-purpose 2D libraries like `tdewolff/canvas` or `fogleman/gg` bring unnecessary dependencies (font parsing, PDF export) and don't match the project's SDF-based text approach. The custom rasterizer also serves as the reference implementation for GPU shader validation (Phase 4.5).
- **Risk / trade-offs:** Performance of the pure Go rasterizer may be lower than optimized C libraries for complex scenes. This is acceptable because the software renderer is a fallback path—GPU rendering (Phase 5) will handle production workloads.

### Text Shaping & Layout
- **Current implementation:** Basic SDF text rendering in `internal/raster/text/` with a pre-baked atlas covering ASCII printable characters (0x20–0x7E). No text shaping (no ligatures, no bidirectional text, no complex script support).
- **Recommended library:** `go-text/typesetting` (Go) — [pkg.go.dev/github.com/go-text/typesetting](https://pkg.go.dev/github.com/go-text/typesetting)
- **Static compilation:** confirmed
  - Pure Go, no CGO dependencies. Compatible with `go build` under musl-gcc static linking.
- **Rationale:** `go-text/typesetting` provides a production-quality, pure Go text shaping engine (HarfBuzz port) with OpenType feature support (ligatures, contextual alternates, kerning), bidirectional text layout, and complex script shaping (Arabic, Devanagari, Thai, etc.). It is actively maintained and used by major Go UI frameworks (Fyne, Gio, Ebitengine). Pre-1.0 versioning (v0.x.y) but API is stabilizing.
- **Risk / trade-offs:** Pre-1.0 API may have breaking changes between minor versions. The library adds ~5–10 MB to binary size due to Unicode tables and shaping data. Integration requires bridging between the library's glyph output and the project's SDF atlas pipeline.

### Font Rasterization / SDF Atlas
- **Current implementation:** Pre-baked 256×256 SDF font atlas embedded as `internal/raster/text/data/atlas.bin` (69 KB), generated by `cmd/gen-atlas/`. Covers ASCII printable range only. No runtime glyph rasterization.
- **Recommended library:** `golang.org/x/image/font/sfnt` + `golang.org/x/image/font/opentype` (Go) — [pkg.go.dev/golang.org/x/image/font](https://pkg.go.dev/golang.org/x/image/font)
- **Static compilation:** confirmed
  - Pure Go, part of the official Go extended library. No CGO dependencies.
- **Rationale:** These packages provide runtime TrueType/OpenType font parsing and glyph rasterization in pure Go. They would enable dynamic SDF atlas generation for characters beyond ASCII—essential for internationalization. The `sfnt` package handles font file parsing; `opentype` provides a higher-level API for glyph metrics and rasterization. Combined with `go-text/typesetting` for shaping, this covers the full text pipeline.
- **Risk / trade-offs:** Runtime glyph rasterization is slower than pre-baked atlases. The recommended approach is to rasterize glyphs on-demand and cache them in the SDF atlas, amortizing the cost. The `golang.org/x/image/font` rasterizer produces bitmap glyphs, not SDF—an SDF conversion step would need to be added.

### Image Decoding
- **Current implementation:** Bilinear-filtered image blitting in `internal/raster/composite/` operates on raw ARGB8888 pixel buffers. No image file format decoding exists in the codebase.
- **Recommended library:** Go stdlib `image/png`, `image/jpeg` + `golang.org/x/image` v0.36.0 (Go) — [pkg.go.dev/golang.org/x/image](https://pkg.go.dev/golang.org/x/image)
- **Static compilation:** confirmed
  - Go stdlib and `golang.org/x/image` are pure Go. No CGO dependencies.
- **Rationale:** The Go standard library provides PNG and JPEG decoding out of the box. `golang.org/x/image` adds WebP, BMP, TIFF, and additional drawing utilities. All are pure Go with no external dependencies, making them ideal for the static binary constraint. These are canonical, well-tested implementations maintained by the Go team.
- **Risk / trade-offs:** Pure Go decoders are slower than C libraries (libpng, libjpeg-turbo) for large images. For a UI toolkit where images are typically small (icons, backgrounds), this is acceptable. If performance becomes an issue, images can be pre-decoded at load time and cached as raw pixel buffers.

### UI Layout Engine (Flexbox-like)
- **Current implementation:** Custom flexbox-like layout engine in `internal/ui/layout/` (~200 LOC). Supports Row/Column direction, Start/Center/End/Stretch alignment, SpaceBetween/SpaceAround justification, and flex-grow/shrink properties. Additional percentage-based sizing in `internal/ui/pctwidget/`.
- **Recommended library:** Keep custom implementation
- **Static compilation:** N/A
- **Rationale:** The layout engine is compact (~200 LOC), correct for current UI needs, and tightly integrated with the widget layer. No mature pure Go flexbox library exists that would justify the integration effort. Facebook's Yoga (C/C++) is the industry standard but adds a C++ dependency and is overkill for the current widget complexity. The custom implementation can be extended incrementally as layout requirements grow.
- **Risk / trade-offs:** If the widget layer grows to support complex nested layouts, CSS Grid, or constraint-based sizing, a more comprehensive layout engine may be needed. At that point, Yoga via CGO (statically linked) would be the primary candidate.

### DRM/KMS Ioctl Wrappers
- **Current implementation:** Custom Rust wrappers in `render-sys/src/drm.rs`, `i915.rs`, `xe.rs` using the `nix` crate (v0.27) for safe ioctl calls. Covers DRM_IOCTL_MODE_CREATE_DUMB, GEM_CLOSE, PRIME_HANDLE_TO_FD, i915 GEM operations, and Xe driver ioctls.
- **Recommended library:** Keep custom implementation (upgrade `nix` to v0.29+)
- **Static compilation:** confirmed
  - `nix` is pure Rust. The custom ioctl wrappers issue syscalls directly, with no libdrm dependency.
- **Rationale:** The Smithay `drm` crate (v0.14.1) provides a higher-level DRM abstraction that bypasses libdrm, but it targets modesetting use cases and doesn't expose the GPU-specific ioctls (i915 GEM, Xe VM bind) that this project needs. The custom approach using `nix` ioctl macros gives precise control over the exact ioctls required. However, `nix` should be upgraded from v0.27 to v0.29+ for bug fixes and improved API safety. Note: v0.27 is still functional but v0.29+ improves `ioctl!` macro ergonomics.
- **Risk / trade-offs:** Custom ioctl wrappers must be manually updated when kernel APIs change (rare for stable ioctls like i915 GEM, more likely for Xe as it matures). The `nix` upgrade from v0.27 to v0.29+ may require minor API adjustments (some function signatures changed).

### GPU Buffer Allocator
- **Current implementation:** Custom Rust slab allocator in `render-sys/src/allocator.rs` and `slab.rs`. Manages GPU buffer allocation with tiling format support (None/X/Y), i915 and Xe driver backends, DMA-BUF export.
- **Recommended library:** Keep custom implementation
- **Static compilation:** N/A
- **Rationale:** The `gpu-allocator` crate (by Traverse Research) targets Vulkan/D3D12 memory allocation, not raw DRM GEM buffer management. No off-the-shelf crate provides direct DRM buffer allocation with Intel tiling format support. The custom allocator is purpose-built for the project's specific needs: GEM buffer allocation → tiling configuration → DMA-BUF export → Wayland/X11 compositor sharing.
- **Risk / trade-offs:** The slab allocator may need to evolve for multi-buffer management, eviction policies, and memory pressure handling as GPU rendering (Phase 5) demands grow. These are incremental extensions to the existing design.

### Shader Compilation (naga → GPU ISA)
- **Current implementation:** Planned for Phase 4. The ROADMAP specifies using `naga` for GLSL/WGSL → IR translation, with custom Rust backends to lower IR to Intel EU and AMD RDNA machine code.
- **Recommended library:** `naga` v28.0.0 (Rust) — [crates.io/crates/naga](https://crates.io/crates/naga)
- **Static compilation:** confirmed
  - Pure Rust, no external C/C++ dependencies. Compiles cleanly with `--target x86_64-unknown-linux-musl`.
  - Add to `Cargo.toml`: `naga = { version = "28", features = ["glsl-in", "wgsl-in"] }`
- **Rationale:** Naga is the shader translation library from the wgpu/gfx-rs ecosystem. It parses GLSL (440+) and WGSL into a well-typed SSA-like IR, which can then be lowered to custom backends. It is actively maintained, battle-tested in production (powers wgpu), and is the only pure Rust shader compiler frontend with both GLSL and WGSL support. The ROADMAP already identifies naga as the preferred choice over C++ alternatives like `glslang`.
- **Risk / trade-offs:** Naga follows wgpu's versioning (major bumps are frequent). Pin to a specific major version in `Cargo.toml` to avoid unexpected breakage. The IR-to-Intel-EU and IR-to-AMD-RDNA backends must be written from scratch—naga provides the frontend and IR but not GPU ISA backends (this is expected and planned in Phases 4 and 6).

### GPU Command Encoding (Intel/AMD)
- **Current implementation:** Planned for Phases 3 (Intel) and 6 (AMD). Will encode Intel 3D pipeline commands (MI_BATCH_BUFFER_START, 3DSTATE_*, 3DPRIMITIVE, PIPE_CONTROL) and AMD PM4 packets.
- **Recommended library:** Keep custom implementation (no suitable library exists)
- **Static compilation:** N/A
- **Rationale:** Raw GPU command encoding at the kernel driver level is inherently hardware-specific. No library provides pre-built Intel EU or AMD PM4 command encoding—this is the domain of GPU driver stacks (Mesa, RADV). The project intentionally builds this from scratch to avoid the massive Mesa dependency tree. Mesa's genxml definitions and Intel PRMs serve as reference documentation, not library dependencies.
- **Risk / trade-offs:** This is the highest-risk, highest-effort subsystem. Correctness depends on precise alignment with Intel PRM and AMD ISA documentation. The software rasterizer (Phase 1) serves as the reference implementation for pixel-level validation (Phase 4.5).

### Accessibility (AT-SPI2)
- **Current implementation:** Planned for Phase 8. No implementation exists yet.
- **Recommended library:** `godbus/dbus` v5.2.2 (Go) — [pkg.go.dev/github.com/godbus/dbus/v5](https://pkg.go.dev/github.com/godbus/dbus/v5)
- **Static compilation:** confirmed
  - Pure Go, no CGO dependencies. Uses Unix domain sockets for D-Bus communication.
  - Install: `go get github.com/godbus/dbus/v5@v5.2.2`
- **Rationale:** AT-SPI2 accessibility on Linux communicates over D-Bus. `godbus/dbus` is the standard pure Go D-Bus implementation, used by Fyne, systemd service managers, and desktop notification libraries. It provides a goroutine-safe API with signal handling, method calls, and property access—all needed for exposing the widget tree via AT-SPI2. BSD-2-Clause license.
- **Risk / trade-offs:** D-Bus communication requires a running D-Bus session bus (standard on desktop Linux). The AT-SPI2 protocol itself is complex—`godbus/dbus` handles transport, but the AT-SPI2 object model (roles, states, relations) must be implemented on top. Consider the `AtspiGo` project as reference implementation, though it is not mature enough to use directly.

### Clipboard / Drag-and-Drop
- **Current implementation:** Planned for Phase 8. No implementation exists yet.
- **Recommended library:** Keep custom implementation (extend existing protocol layers)
- **Static compilation:** N/A
- **Rationale:** Clipboard and drag-and-drop are Wayland protocol extensions (`wl_data_device_manager`, `wl_data_device`, `wl_data_source`, `wl_data_offer`) and X11 protocol operations (selections: CLIPBOARD, PRIMARY; XDND for drag-and-drop). The existing custom protocol implementations in `internal/wayland/` and `internal/x11/` are the natural foundation for this feature—it requires adding new protocol objects following the established wire format patterns, not a separate library.
- **Risk / trade-offs:** MIME type negotiation and format conversion between applications adds complexity. The Wayland `data_device` protocol and X11 selection mechanism have different semantics that must be abstracted at the widget layer.

## Libraries Evaluated but Rejected

| Library | Language | Subsystem | Rejection Reason |
|---------|----------|-----------|-----------------|
| `jezek/xgb` v1.3.0 | Go | X11 protocol | Pure Go and well-maintained, but custom implementation is already complete and tightly integrated. Would require rearchitecting protocol layer. Reconsider if many new X11 extensions are needed. |
| `gotk3`/`gotk4` | Go | Windowing/UI | Requires GTK shared libraries (`libgtk-3.so`/`libgtk-4.so`); fundamentally incompatible with static compilation constraint. |
| `go-gl/gl` | Go | Rendering | Requires OpenGL shared libraries (`libGL.so`); project targets raw kernel GPU interfaces, not OpenGL. |
| `veandco/go-sdl2` | Go | Windowing | SDL2 requires dynamic linking (`libSDL2.so`); adds unnecessary abstraction layer over protocols already implemented. |
| `fyne-io/fyne` | Go | UI framework | Full GUI toolkit with its own renderer; too heavy, brings dynamic dependencies, conflicts with project architecture. |
| `tdewolff/canvas` | Go | Rasterizer | Comprehensive 2D library but brings font parsing, PDF export, and SVG dependencies. Overkill for UI primitive rendering; doesn't match SDF text approach. |
| `fogleman/gg` | Go | Rasterizer | Depends on `golang.org/x/image/font` for text rendering (different approach from SDF); adds freetype-style rasterization that duplicates existing capability. |
| `srwiley/rasterx` | Go | Rasterizer | SVG 2.0 path rasterizer; good for general vector graphics but doesn't provide UI-specific primitives (box shadow, gradient fills, SDF text). No tagged releases. |
| `Smithay/drm-rs` v0.14.1 | Rust | DRM/KMS | Higher-level DRM abstraction, but targets modesetting. Does not expose GPU-specific ioctls (i915 GEM_CREATE, GEM_SET_TILING, Xe VM_BIND) needed for this project. |
| `gpu-allocator` | Rust | Buffer allocator | Designed for Vulkan/D3D12 memory allocation, not raw DRM GEM buffer management. Wrong abstraction level. |
| `wgpu` | Rust | Rendering | Full GPU abstraction layer with Vulkan/Metal/GL backends. Project intentionally targets raw kernel interfaces; wgpu's portability layer adds complexity without benefit. |
| `ash` | Rust | Rendering | Vulkan bindings. Project doesn't use the Vulkan API—it submits batch buffers directly to i915/Xe/AMDGPU kernel drivers. |
| `winit` | Rust | Windowing | Rust windowing library. Go handles all windowing; adding Rust windowing would duplicate the protocol layer. |
| `smithay-client-toolkit` | Rust | Wayland | Rust Wayland client toolkit. Go handles all Wayland protocol; Rust layer only handles GPU-side operations. |
| `glslang` | C++ | Shader compilation | C++ GLSL compiler producing SPIR-V. Adds C++ build dependency; `naga` (pure Rust) is preferred per ROADMAP to avoid C++ in the build chain. |
| `nicholasgasior/goflex` | Go | Layout engine | Unmaintained; insufficient feature set for UI toolkit layout needs. |
| Yoga (Facebook) | C/C++ | Layout engine | Industry-standard flexbox engine but adds C++ dependency via CGO. Overkill for current layout complexity (~200 LOC custom engine). Reconsider if layout requirements grow significantly. |
| `AtspiGo` | Go | Accessibility | AT-SPI2 Go bindings; not mature enough for production use. Use `godbus/dbus` as transport and implement AT-SPI2 protocol on top. |

## Next Steps

Adoption should follow ROADMAP phase ordering and prioritize effort-to-value ratio:

1. **Upgrade `nix` crate to v0.29+** (Immediate, low effort)
   - Phase: Applies to current Phase 2 code
   - Effort: ~1 hour (update `Cargo.toml`, fix minor API changes)
   - Value: Security fixes, improved ioctl macro ergonomics
   - File: `render-sys/Cargo.toml` — change `nix = { version = "0.27", ... }` to `nix = { version = "0.29", ... }`

2. **Integrate `naga` for shader frontend** (Phase 4, high value)
   - Phase: Phase 4 critical path
   - Effort: ~2 hours for crate integration; weeks for custom IR→EU backend
   - Value: Enables the entire GPU shader compilation pipeline
   - File: `render-sys/Cargo.toml` — add `naga = { version = "28", features = ["glsl-in", "wgsl-in"] }`

3. **Add `golang.org/x/image` for image decoding** (Phase 5, medium value)
   - Phase: Phase 5 (texture atlas management needs image loading)
   - Effort: ~2–4 hours for integration with existing pixel buffer pipeline
   - Value: Enables loading PNG/JPEG/WebP images for UI content
   - Command: `go get golang.org/x/image@v0.36.0`

4. **Add `golang.org/x/image/font` for dynamic glyph rasterization** (Phase 5, medium value)
   - Phase: Phase 5 (GPU texture atlas needs dynamic glyph updates)
   - Effort: ~1–2 days to integrate with SDF atlas generation pipeline
   - Value: Extends font support beyond pre-baked ASCII range
   - Depends on: Step 3 (same module)

5. **Add `go-text/typesetting` for text shaping** (Phase 8, medium value)
   - Phase: Phase 8 (internationalization, HiDPI text)
   - Effort: ~2–3 days to bridge shaping output with SDF rendering pipeline
   - Value: Full Unicode text support (ligatures, bidi, complex scripts)
   - Command: `go get github.com/go-text/typesetting@latest`

6. **Add `godbus/dbus` for accessibility** (Phase 8, lower immediate value)
   - Phase: Phase 8 (AT-SPI2 accessibility)
   - Effort: ~1–2 weeks for full AT-SPI2 widget tree exposure
   - Value: Accessibility compliance for desktop Linux
   - Command: `go get github.com/godbus/dbus/v5@v5.2.2`
