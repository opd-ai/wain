# UNIVERSAL BUG AUDIT (END-TO-END) — 2026-05-21

## Project Profile

**Purpose**: Wain is a statically-compiled Go UI toolkit for Linux that renders through a Rust GPU backend with automatic software fallback. It implements Wayland and X11 display protocols directly and submits GPU commands to kernel DRM interfaces — no OpenGL, Vulkan, or system graphics libraries required.

**Target users**: Linux desktop application developers wanting zero-dependency static binaries.

**Deployment model**: Single-user desktop Linux apps; runs as an unprivileged user process. The primary attack surface is:
- Reading malformed Wayland/X11 events from the compositor
- Clipboard data from other applications (inter-process data)
- User-supplied image/font files

**Critical paths**:
1. `App.Run()` → event loop → render frames (correctness of entire display pipeline)
2. `AppConfig.DRMPath / ForceSoftware` → `initRenderer()` (renderer selection)
3. Clipboard read/write on both backends (`selection.Manager`, `datadevice.Offer`)
4. Damage-tracked rendering (`SoftwareBackend.RenderWithDamage`)

---

## Audit Scope

**Packages audited** (all packages):
- `github.com/opd-ai/wain` (root: app.go, animate.go, dispatcher.go, render.go, resource.go, event.go, accessibility.go)
- `internal/raster/primitives`, `composite`, `curves`, `displaylist`, `effects`, `text`, `consumer`
- `internal/render`, `render/atlas`, `render/backend`, `render/display`, `render/present`
- `internal/ui/animation`, `decorations`, `layout`, `pctwidget`, `scale`, `widgets`
- `internal/wayland/client`, `datadevice`, `dmabuf`, `input`, `output`, `shm`, `socket`, `wire`, `xdg`
- `internal/x11/client`, `dnd`, `dpi`, `dri3`, `events`, `gc`, `present`, `selection`, `shm`, `wire`
- `internal/a11y`, `internal/buffer`, `internal/integration`, `internal/demo`

**Total packages**: 46

**Build status**: Packages that depend on CGO (render-sys Rust shared library) fail to link in this environment due to missing native symbols (`render_add`, `buffer_allocate`, etc.). All pure-Go packages build and test cleanly.

**Test results** (pure-Go packages): All 22 pure-Go internal packages pass `go test -race`.

**go vet**: Clean — no warnings.

---

## Coverage Log

| Package | 3b Logic | 3c Nil | 3d Errors | 3e Resources | 3f Concurrency | 3g Security | 3h Aliasing | 3i Init | 3j API |
|---------|----------|--------|-----------|--------------|----------------|-------------|-------------|---------|--------|
| wain (root) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| raster/primitives | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| raster/composite | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| raster/curves | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| raster/displaylist | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| raster/effects | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| raster/text | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| raster/consumer | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| render (stats) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| render/backend | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| render/display | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| ui/animation | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| ui/pctwidget | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| ui/widgets | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| wayland/datadevice | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| wayland/shm | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| x11/client | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| x11/dnd | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| x11/selection | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| (remaining pkgs) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## Goal-Achievement Summary

| Stated Goal | Status | Blocking Findings |
|-------------|--------|-------------------|
| Display server auto-detection (Wayland → X11 fallback) | ✅ | — |
| GPU renderer auto-detection (Intel → AMD → software) | ⚠️ | H1 (DRMPath/ForceSoftware ignored) |
| Damage-tracked incremental rendering | ⚠️ | H2 (full re-render for text/gradient/image) |
| Clipboard read/write on both Wayland and X11 | ⚠️ | H3, M1, M2 |
| Custom font loading via `App.LoadFont` | ❌ | M3 (path ignored, embedded font always returned) |
| Animation with configurable duration and easing | ⚠️ | M4 (hardcoded 16 ms frame budget) |
| X11 drag-and-drop | ⚠️ | M5 (source-side DnD silently unimplemented) |
| `AppConfig.ForceSoftware` forces software rendering | ❌ | H1 |
| `AppConfig.DRMPath` selects a custom DRM device | ❌ | H1 |
| AT-SPI2 accessibility | ✅ | — |
| Theming (DefaultDark / DefaultLight / HighContrast) | ✅ | — |
| Widget system (Button, Label, TextInput, ScrollView, …) | ✅ | — |

---

## Findings

### HIGH

- [ ] **AppConfig.DRMPath and ForceSoftware are silently ignored** — `app.go:1909,1914` — Logic bug — `initRenderer()` hard-codes `DRMPath: "/dev/dri/renderD128"` and `ForceSoftware: false` instead of using `a.drmPath` and `a.forceSW` that were correctly stored from the user's `AppConfig`. Any application calling `NewAppWithConfig(AppConfig{DRMPath: "/dev/dri/card1", ForceSoftware: true})` silently gets the default values; the GPU selection and software-fallback override are completely non-functional. — **Remediation**: In `initRenderer()`, replace the two hard-coded literals:
  ```go
  DRMPath:       a.drmPath,
  ForceSoftware: a.forceSW,
  ```
  Validate with `go test -race ./...` and manual test with `ForceSoftware: true`.

- [ ] **SoftwareBackend.RenderWithDamage falls through to full re-render for every non-trivial command type** — `internal/render/backend/software.go:100` — Logic bug — When `damage` regions are provided, the method iterates the filtered command slice but, on encountering `CmdDrawText`, `CmdLinearGradient`, `CmdRadialGradient`, `CmdBoxShadow`, or `CmdDrawImage`, falls into the `default:` branch and calls `sb.consumer.Render(dl, sb.buffer)`, which renders the **entire original (unfiltered) display list**. Because real UIs always contain text, damage-tracking is effectively bypassed on every frame that contains a label or button. Concrete path: `App.eventLoop → renderFrames → Window.RenderFrame → RenderBridge.Render (fullRedraw=false) → SoftwareBackend.RenderWithDamage(damage=[rect]) → hit default: → Render(full dl)`. — **Remediation**: Replace the `default:` branch with a delegation to `sc.SoftwareConsumer.renderCommand(cmd, sb.buffer)` (or inline equivalent), removing the early-return fallback. Validate with a test that checks only damaged regions are re-rendered for a display list containing a `CmdDrawText` command.

- [ ] **X11 TARGETS selection response uses wrong `format` field (8 instead of 32)** — `internal/x11/selection/manager.go:185` — Protocol correctness bug — When `HandleSelectionRequest` is called with `target == TARGETS`, `resolveSelectionData` returns `actualType = 4` (XA_ATOM) and `format=8` is passed to `ChangeProperty`. The X11 protocol requires `format=32` for ATOM-typed property data. Passing `format=8` tells the X server to interpret the 32-bit atom IDs as individual bytes, causing all client applications requesting TARGETS to receive garbled data and failing to negotiate a paste MIME type. Every X11 clipboard paste from a compliant receiver is broken. Concrete path: X11 paste request → `HandleSelectionRequest` → `resolveSelectionData` returns `(buf, 4, true)` → `ChangeProperty(…, 4, 8, 0, buf)` → X server stores 8-bit format. — **Remediation**: Change the `ChangeProperty` call to use `format=32` when `actualType` is an atom type:
  ```go
  format := uint8(8)
  if actualType == 4 { // XA_ATOM
      format = 32
  }
  m.conn.ChangeProperty(requestor, property, actualType, format, 0, data)
  ```
  Validate by verifying TARGETS negotiation works with `xclip -o` on an X11 display.

---

### MEDIUM

- [ ] **`getSelection()` uses `time.Sleep(50ms)` instead of waiting for SelectionNotify** — `internal/x11/selection/manager.go:167` — Race condition / reliability bug — `GetClipboard()` and `GetPrimary()` send a `ConvertSelection` request, sleep 50ms, then unconditionally read the property. The X11 selection protocol requires waiting for the `SelectionNotify` event before reading. If the selection owner is slow (another app under load, virtualized desktop, remote X11) the property will not be available after 50ms and an empty string is silently returned. Conversely, if `GetProperty` is called before the SelectionNotify, the read may race with the owner writing the property. — **Remediation**: Implement proper SelectionNotify event waiting in the `eventLoop` by storing a pending-read channel in `Manager`, signaling it from `dispatchX11Event` when `SelectionNotify` is received for the expected property, and blocking `getSelection` on that channel instead of sleeping.

- [ ] **`App.LoadFont` documents custom font loading but always returns the embedded font** — `resource.go:LoadFont` — API behavioral contract violation — `LoadFont(path, size)` GoDoc states: "loads a font from the specified path at the given size." The implementation ignores `path` entirely and returns a `*Font` backed by the embedded default atlas regardless of the path argument. There is no error indicating the path was not used. Applications that call `app.LoadFont("/opt/fonts/Roboto.ttf", 16)` silently get the embedded fallback font. — **Remediation**: Either implement TTF loading (future work), or return `ErrInvalidFontData` with a clear message like `"custom font loading from TTF not yet implemented; path ignored"`, or document the stub behavior explicitly in GoDoc. Validate with a test that expects a meaningful error or note when `path != ""`.

- [ ] **X11 drag-and-drop source is silently unimplemented** — `app.go:2264–2270` — Feature gap / silent failure — `startDragForWindow` includes the comment "Registration of w.dragDataProvider above is sufficient for the event loop to respond to XdndStatus/XdndDrop messages" but the X11 event loop does not handle `XdndStatus`, `XdndDrop`, or `XdndFinished` ClientMessages, and no XdndEnter/XdndPosition messages are sent from the source. Calling `Window.StartDrag` on X11 does nothing observable. — **Remediation**: Implement the XDND source protocol in `startDragForWindow` for X11: call `dnd.Manager.AdvertiseAware()`, `SendEnter(target, mimeTypeAtoms)`, then drive `SendPosition` messages in the pointer motion handler. Wire `XdndStatus`/`XdndDrop`/`XdndFinished` handling in `dispatchX11Event`.

- [ ] **`resolveSelectionData` returns TARGETS regardless of selection ownership** — `internal/x11/selection/manager.go:202` — Logic bug — The `if target == m.targetsAtom` branch fires even when the caller does not own either selection. After `HandleSelectionClear` has cleared both `ownsClipboard` and `ownsPrimary`, any application can still send a SelectionRequest for TARGETS and receive a valid response, causing it to believe this client can fulfill a paste. — **Remediation**: Add an ownership guard before the TARGETS branch:
  ```go
  if target == m.targetsAtom && (m.ownsClipboard || m.ownsPrimary) {
  ```
  Validate with a test that verifies TARGETS returns `ok=false` when neither selection is owned.

- [ ] **`HandleSelectionRequest` does not guard against `property == 0`** — `internal/x11/selection/manager.go:185` — Protocol correctness bug — When a SelectionRequest event arrives with `property == None (0)`, the correct response is to send `SelectionNotify` with `property=None` (indicating refusal). Instead, `ChangeProperty(requestor, 0, ...)` is called even though atom 0 is `None`, not a valid property atom. This violates the X11 selection protocol and can trigger a `BadAtom` error or failed transfer for the requesting client. — **Remediation**: Add at the top of `HandleSelectionRequest`:
  ```go
  if property == 0 {
      return m.sendSelectionNotify(requestor, selection, target, 0, timestamp)
  }
  ```

- [ ] **`int32` overflow in `validateBufferParams` and `CreateBuffer`** — `internal/wayland/shm/pool.go:68,107` — Arithmetic overflow — `bufferSize := int32(height) * stride` overflows when `height * stride > 2147483647`. For a 1080p buffer (height=1080, stride=1920*4=7680) `bufferSize = 8,294,400` — fine. But Wayland compositors can create larger pools (e.g., 8K at stride=32768 → height=4320: `4320 * 32768 = 141,557,760` — fine). The real overflow risk appears when `stride` is provided by an untrusted compositor at a very large value; at `height=32768, stride=65536`, `bufferSize = int32(32768 * 65536)` overflows. An overflow would produce a negative `bufferSize` that passes the `> p.size` guard (negative < any positive pool size), allowing a subsequent `p.mapping[offset : offset+bufferSize]` panic or memory read outside the mapping. — **Remediation**: Use `int64` for the intermediate product:
  ```go
  bufferSize := int64(height) * int64(stride)
  if bufferSize > math.MaxInt32 || offset+int32(bufferSize) > p.size {
      return fmt.Errorf(...)
  }
  ```

- [ ] **Animation frame timing hardcoded to 16ms regardless of actual elapsed time** — `app.go:2188` — Logic bug — `a.animator.Tick(16 * 1e6)` always advances animations by exactly 16ms, but the comment acknowledges "Future work: measure actual elapsed time per frame." On a system running at 10 FPS, each frame takes 100ms but animations advance only 16ms — animations play at 16% of the intended speed. On a 120 Hz display, each frame takes about 8.33ms but animations advance 16ms — animations play at about 192% speed and complete early. — **Remediation**: Record the real frame start time and pass the elapsed duration:
  ```go
  now := time.Now()
  dt := now.Sub(lastFrame)
  if dt > 100*time.Millisecond { dt = 100*time.Millisecond } // clamp for tab-switch catchup
  a.animator.Tick(dt)
  lastFrame = now
  ```
  Initialize `lastFrame = time.Now()` before the event loop.

---

### LOW

- [ ] **`damageDrawText` uses `len(data.Text)` (byte count) not rune count for width estimate** — `internal/raster/displaylist/damage.go:~line 220` — Logic bug — `width := len(data.Text) * data.FontSize / 2` overestimates the pixel width of multi-byte UTF-8 strings (a CJK character is 3 bytes but renders as one glyph). The damage region will be wider than necessary (wasted re-renders) but never narrower (no missed redraws). Impact is waste of CPU on incremental rendering, not correctness. — **Remediation**: Use `utf8.RuneCountInString(data.Text)` instead of `len(data.Text)`.

- [ ] **`min`/`max` helper functions are redeclared in three packages** — `internal/raster/composite/ops.go`, `internal/raster/curves/bezier.go`, `internal/raster/displaylist/damage.go` — Code smell — Go 1.21 introduced `min`/`max` as language builtins. The local declarations shadow them without causing errors but are dead weight. They will cause compilation failures if a future Go version removes the ability to shadow builtins. — **Remediation**: Remove the local `min`/`max` definitions and use the builtins. Validate with `go build ./...`.

- [ ] **`resource.go init()` re-registers PNG/JPEG decoders already registered by the standard library** — `resource.go:init()` — Logic bug (low impact) — `image/png` and `image/jpeg` are blank-imported alongside the explicit `image.RegisterFormat` calls in `init()`. The `image` package deduplicates by format name, so the re-registrations are no-ops, but they're confusing and redundant. — **Remediation**: Remove the explicit `image.RegisterFormat` calls from `init()` since the imports already register the decoders.

- [ ] **`sendSelectionNotify` encodes `SendEvent` with the wrong wire layout and request length** — `internal/x11/selection/manager.go:213` — Protocol bug — `SendEvent` expects `propagate` in the request header's data byte plus a fixed 40-byte payload (`destination`, `event-mask`, and a 32-byte event), for 44 bytes total / 11 length units. Instead, `sendSelectionNotify` builds a 44-byte payload, puts `propagate` into the payload itself, shifts the remaining fields, and causes the computed request length to become 12 units. The adapter also always sends the header data byte as 0. This mis-encodes the request on the wire and can cause the X server to reject or misinterpret the SelectionNotify event. — **Remediation**: Rework `sendSelectionNotify` to match the actual `SendEvent` wire format: pass `propagate` via the request header's data byte, keep the payload at the fixed 40-byte layout, and ensure the request length remains 11 units.

- [ ] **`FocusManager` notifies the focus-change hook while holding `fm.mu` (lock inversion risk)** — `dispatcher.go:FocusManager.FocusPrev/FocusNext/Focus` — Concurrency / deadlock risk — `notifyFocusChange(w)` is called inside `fm.mu.Lock()`. If the hook itself tries to acquire `fm.mu` (or any other lock held by a caller of `Focus()`), a deadlock results. The AT-SPI2 hook route (`dispatcher.go:SetFocusChangeHook`) passes arbitrary user code through this path. — **Remediation**: Capture the hook reference inside the lock, then release the lock before invoking the hook:
  ```go
  fm.mu.Unlock()
  hook(w)
  ```

- [ ] **`dispatchX11Event` sends the event to ALL windows instead of the targeted window** — `app.go:2029` — Logic bug — `dispatchX11Event` calls `win.handleX11Event(eventType, eventBuf)` for every window in `a.windows`, even though the X11 event is directed at one specific window (identified by `eventBuf`'s window field). Windows that do not own the event will attempt to parse and route it, resulting in spurious callbacks and UI state mutations. — **Remediation**: Parse the window XID from `eventBuf` (typically at byte offset 4 for most event types) and look up the target window before dispatching.

---

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Go version | 1.24 |
| Total packages | 46 |
| Test pass rate (pure-Go) | 22/22 (100%) |
| Test pass rate (CGO-linked) | N/A — Rust shared library not present in build environment |
| go vet warnings | 0 |
| HIGH findings | 3 |
| MEDIUM findings | 7 |
| LOW findings | 6 |

---

## False Positives Considered and Rejected

| Candidate | Reason Rejected |
|-----------|----------------|
| `datadevice.Offer.ReadData` closes `writeFD` before `io.ReadAll` | This is the correct UNIX pipe pattern: the compositor receives a copy of `writeFD` via SCM_RIGHTS through the Wayland socket. After the copy is sent, the local copy is correctly closed. `io.ReadAll` blocks until the compositor closes its copy, which is proper EOF signalling. Not a bug. |
| `resolveSelectionData` returns `4` for XA_ATOM | XA_ATOM is defined in the X11 protocol as pre-defined atom 4. The hardcoded literal is functionally correct, though fragile style. Kept as LOW style note but not a correctness bug. |
| `BlendPixel` uses integer arithmetic instead of floating-point alpha | The fixed-point 8-bit multiply-and-divide is standard Porter-Duff for 8-bit channels; no precision issue at this bit depth. Not a bug. |
| `Animator.Tick` does not protect `animations` slice with a mutex | The GoDoc explicitly states the Animator is designed for single-threaded use from the render loop. No concurrency contract is violated. Not a bug. |
| `Coalesce` in `DamageTracker` is O(n²) in the number of regions | For typical UI update counts (under 50 regions), n² is negligible. Only flagged if user-controlled unbounded input could grow `regions` — no such path exists. Not a hot-path finding. |
| `x11/dnd.Manager.SendPosition` packs x,y into a single uint32 using `(x<<16)\|y` | This is the XDND wire format as specified in the freedesktop XDND spec. Not a bug. |
| `Buffer.NewBuffer` limits dimensions to 16384 — no overflow in stride*height | stride=16384*4=65536, height=16384 → size=16384*65536=1,073,741,824 < `int` max on 64-bit. The 16384 guard prevents the int overflow for the Go `primitives` package. Not a bug at that layer. |

---

## Remaining Scope

All 46 packages have been audited. No remaining scope.
