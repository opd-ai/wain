# Implementation Gaps — 2026-05-21

## AppConfig.DRMPath and AppConfig.ForceSoftware Have No Effect

- **Stated Goal**: The README and `AppConfig` GoDoc promise that `DRMPath` selects the DRM device for GPU detection and `ForceSoftware` forces software rendering.
- **Current State**: `initRenderer()` (`app.go:1909,1914`) hard-codes `DRMPath: "/dev/dri/renderD128"` and `ForceSoftware: false` in the `backend.AutoConfig` struct. The fields `a.drmPath` and `a.forceSW`, which are correctly populated from `AppConfig` in `NewAppWithConfig`, are never referenced inside `initRenderer`. Any application passing a custom DRM device path or requesting software rendering via `AppConfig` silently receives the wrong backend.
- **Impact**: The documented GPU selection and software-override API is broken. Applications on systems with the DRM device at a non-default path (`/dev/dri/card0`, `/dev/dri/renderD129`) will fail to detect the GPU. `ForceSoftware: true`, a critical option for testing and headless environments, has no effect.
- **Closing the Gap**: In `initRenderer()`, replace the two hard-coded literals with `a.drmPath` and `a.forceSW`:
  ```go
  cfg := backend.AutoConfig{
      DRMPath:       a.drmPath,   // was: "/dev/dri/renderD128"
      ForceSoftware: a.forceSW,   // was: false
      ...
  }
  ```

---

## App.LoadFont Does Not Load Fonts

- **Stated Goal**: The README feature list states "Widget System — Button, Label, TextInput, ScrollView, ImageWidget, Spacer with percentage-based sizing". `App.LoadFont` and `ResourceManager.LoadFont` are documented as: "loads a font from the specified path at the given size."
- **Current State**: `ResourceManager.LoadFont` (`resource.go`) ignores the `path` argument entirely. It returns a `*Font` struct that shares the embedded default atlas regardless of what path was provided. The method returns no error and gives no indication that the path was not used. The GoDoc comment even adds: "Custom font loading from TTF files will be implemented in a future phase." — but this is buried in the implementation file, not the public API comment.
- **Impact**: Applications that load custom fonts for brand typography or accessibility reasons (e.g., larger default sizes, dyslexia-friendly fonts) silently receive the embedded 14pt monospace fallback. There is no way to tell that the font wasn't loaded.
- **Closing the Gap**: Either (a) return a descriptive `fmt.Errorf("LoadFont: custom TTF loading not yet implemented")` error so callers know they must use `DefaultFont()`, or (b) implement TTF parsing using `golang.org/x/image/font/sfnt` and SDF atlas generation. At minimum the GoDoc of the public `App.LoadFont` method must state that the path is currently ignored.

---

## Damage-Tracked Rendering Falls Back to Full Re-Render for All Non-Primitive Commands

- **Stated Goal**: The `RenderBridge` GoDoc describes "Emit DisplayList commands for dirty regions" and "Submit to renderer with damage rects". The README describes "Software Rasterizer — rectangles, rounded rectangles, anti-aliased lines, Bézier curves, gradients, shadows, and SDF text."
- **Current State**: `SoftwareBackend.RenderWithDamage` (`internal/render/backend/software.go:100`) attempts to render only commands that intersect the damage rectangle. However, the `default:` branch of its command-type switch calls `sb.consumer.Render(dl, sb.buffer)`, which renders the **full original display list**. Since text (`CmdDrawText`), gradients (`CmdLinearGradient`, `CmdRadialGradient`), shadows (`CmdBoxShadow`), and images (`CmdDrawImage`) all fall into `default:`, every real UI frame triggers a full re-render. The damage optimization only applies to pure-rectangle UIs.
- **Impact**: The CPU saving promised by incremental rendering is never realized. On complex UIs the software renderer may consume more CPU than necessary. The damage rect mechanism in `RenderBridge.MarkRegionDirty` is built but ineffective.
- **Closing the Gap**: Implement per-command rendering in the `default:` branch (or extend the switch to cover all `displaylist.Cmd*` types) rather than falling back to full list rendering. The `consumer` package already handles each command type individually — the fallback path should dispatch through it for the filtered command list only.

---

## X11 Drag-and-Drop Source Is Silently Unimplemented

- **Stated Goal**: The README states "X11 Protocol — server connection, windows, DRI3, Present, MIT-SHM, clipboard, drag-and-drop, and HiDPI detection". The `Window.StartDrag` API is a public exported method.
- **Current State**: For Wayland, `startDragForWindow` correctly creates a `wl_data_source` and calls `wl_data_device.start_drag`. For X11, the function does nothing beyond storing `w.dragDataProvider`. A comment says "Registration of w.dragDataProvider above is sufficient for the event loop to respond to XdndStatus/XdndDrop messages" — but the X11 event loop in `dispatchX11Event` has no code to handle `XdndStatus`, `XdndDrop`, or `XdndFinished` ClientMessages, and no XdndEnter/XdndPosition messages are ever sent. The `internal/x11/dnd` package is fully implemented with all required message builders but is unused by the main event loop.
- **Impact**: Any application calling `window.StartDrag(...)` on an X11 display gets a no-op. No drag cursor appears, no drop is possible, no error is returned.
- **Closing the Gap**: Wire `dnd.Manager` into the X11 path in `startDragForWindow`: call `SendEnter`, then drive `SendPosition` from the pointer motion handler, and handle `XdndStatus`/`XdndDrop`/`XdndFinished` in `dispatchX11Event`.

---

## Animation Speed Is Decoupled from Actual Elapsed Time

- **Stated Goal**: `App.Animate` GoDoc states: "Animate schedules a property animation driven by the app's frame loop." The `animation.Animator` is designed to accept a `time.Duration` delta and the code comment at `app.go:2187` acknowledges that measuring real elapsed time is "Future work".
- **Current State**: `renderFrames()` calls `a.animator.Tick(16 * 1e6)` — always 16ms — independent of when the previous frame actually completed. This means animations play at 1× speed only on a display running at exactly 62.5 Hz. At 60 Hz they play at 94% speed; at 30 Hz they play at 47% speed; at 120 Hz they play at 200% speed and complete in half the specified duration.
- **Impact**: Animation durations passed to `App.Animate` are meaningless on any real display. A 300ms fade-in completes in 150ms on a 120 Hz screen and in 600ms on a 30 Hz screen.
- **Closing the Gap**: Track the real frame start time (`lastFrame time.Time`) and pass the actual delta to `Tick`. Add a delta clamp (e.g., 100ms) to avoid runaway catch-up after tab switches or long pauses.

---

## X11 Clipboard Read Is Unreliable (time.Sleep Polling)

- **Stated Goal**: The README states "Clipboard — read/write clipboard on both Wayland and X11". `GetClipboard()` is a public API.
- **Current State**: `selection.Manager.getSelection()` sends `ConvertSelection`, sleeps 50ms, then reads the property. The X11 selection protocol requires waiting for a `SelectionNotify` event before reading the property. On a loaded system, 50ms is insufficient. On a fast system, `GetProperty` may execute before the selection owner has written the data.
- **Impact**: `GetClipboard()` on X11 returns an empty string intermittently. It is unreliable for any production use. The behavior worsens with remote X11 sessions where latency is higher.
- **Closing the Gap**: Register a pending-read waiter in `Manager` when `ConvertSelection` is issued. Signal the waiter from the main event loop when `SelectionNotify` arrives for the matching window and property. Block `getSelection` on that signal rather than sleeping.
