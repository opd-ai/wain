# UNIVERSAL BUG AUDIT (END-TO-END) — 2026-05-30

## Project Profile

- **Purpose:** Wain is a statically-compiled Go UI toolkit for Linux that
  renders via a Rust GPU backend with software fallback. It implements the
  Wayland and X11 display protocols directly (no CGO-linked system graphics
  libraries) to produce fully static, zero-dependency binaries.
- **Target users:** Go developers building Linux desktop apps that need static,
  dependency-free binaries (the README explicitly contrasts with Fyne/Gio/GTK
  which require CGO + system libs).
- **Deployment model:** Single static binary run on an end-user Linux machine
  under either a Wayland compositor or an X11 server. The display server is a
  **trust boundary**: untrusted/buggy bytes arrive over a unix socket and are
  decoded by hand-written wire parsers in `internal/wayland/*` and
  `internal/x11/*`. A second trust boundary is the AT-SPI2 D-Bus interface
  (`internal/a11y`), where a screen-reader client can invoke methods with
  arbitrary offsets.
- **Critical paths (primary stated goals):**
  1. Window creation + event loop (`app.go`).
  2. Widget tree → display list → rasterizer/GPU → present
     (`render.go`, `internal/raster/*`, `internal/render/*`).
  3. Wayland protocol client (`internal/wayland/*`).
  4. X11 protocol client (`internal/x11/*`).
  5. Double/triple buffering frame sync (`internal/buffer`).

## Audit Scope

- **Packages audited:** all 40 Go packages reported by `go list ./...`
  (75 import paths including `cmd/*` demos and `example/*`). Non-Go assets
  (`render-sys/` Rust, shell scripts) were read for context but the Rust
  backend was out of scope for Go bug-hunting.
- **Source inspected:** 201 Go source files (tests skipped for metrics),
  662 functions + 1180 methods.
- **Tooling:** `go-stats-generator v1.0.0`, `go vet ./...`,
  `go test -race ./...`. Manual line-by-line inspection of the highest-risk
  functions and of every confirmed finding's data flow.
- **Environmental note:** 16 packages fail to **build/test** in this sandbox
  because the Rust static library `librender_sys.a` is absent (linker:
  `undefined reference to buffer_allocator_create`, `render_compile_shader`,
  etc.). This is an environmental limitation, **not** a code defect, and is
  excluded from findings. Those packages' Go logic was audited by reading.
  All 35 pure-Go test packages pass under `-race`.

## Coverage Log

Legend: ✅ category audited · — category not applicable to package
(e.g. no concurrency primitives, no external input).

| Package | 3b Logic | 3c Nil | 3d Errors | 3e Resources | 3f Concurrency | 3g Security | 3h Aliasing | 3i Init | 3j API |
|---------|----------|--------|-----------|--------------|----------------|-------------|-------------|---------|--------|
| `wain` (root) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `internal/wayland/client` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `internal/wayland/wire` | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | ✅ | ✅ |
| `internal/wayland/socket` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| `internal/wayland/shm` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `internal/wayland/datadevice` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| `internal/wayland/dmabuf` | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | — | ✅ |
| `internal/wayland/input` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `internal/wayland/output` | ✅ | ✅ | ✅ | — | — | ✅ | ✅ | — | ✅ |
| `internal/wayland/xdg` | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | — | ✅ |
| `internal/x11/client` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `internal/x11/wire` | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | ✅ | ✅ |
| `internal/x11/selection` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| `internal/x11/shm` | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | — | ✅ |
| `internal/x11/dri3` | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | — | ✅ |
| `internal/x11/present` | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | — | ✅ |
| `internal/x11/gc` | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | — | ✅ |
| `internal/x11/dnd` | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | — | ✅ |
| `internal/x11/dpi` | ✅ | ✅ | ✅ | — | — | ✅ | ✅ | — | ✅ |
| `internal/x11/events` | ✅ | ✅ | ✅ | — | — | ✅ | ✅ | — | ✅ |
| `internal/raster` (+ primitives/curves/text/effects/composite/displaylist/consumer) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `internal/render` (+ atlas/backend/display/present) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `internal/buffer` | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | ✅ |
| `internal/a11y` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `internal/ui/layout` | ✅ | ✅ | ✅ | — | — | ✅ | ✅ | — | ✅ |
| `internal/ui/widgets` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `internal/ui/animation` | ✅ | ✅ | ✅ | — | ✅ | — | ✅ | — | ✅ |
| `internal/ui/decorations` | ✅ | ✅ | ✅ | — | — | ✅ | ✅ | — | ✅ |
| `internal/ui/pctwidget` | ✅ | ✅ | ✅ | — | — | ✅ | ✅ | — | ✅ |
| `internal/ui/scale` | ✅ | ✅ | ✅ | — | ✅ | — | ✅ | — | ✅ |
| `internal/integration` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| `internal/demo` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| `cmd/*` (29 demo/tool mains) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | — | ✅ |
| `example/*` (hello, multi-window) | ✅ | ✅ | ✅ | — | — | ✅ | ✅ | — | ✅ |
| `scripts` (analyze_godoc.go dev tool) | ✅ | ✅ | ✅ | ✅ | — | ✅ | ✅ | — | ✅ |

`cmd/*`, `example/*`, and `scripts` are thin demo/tool drivers (each a small
`main`); they were reviewed for resource leaks and injection but carry no
production-critical logic. `cmd/wain-build/main.go` builds the Rust library via
`exec.Command` with fixed argument vectors and no user-controlled command
strings — clean.

## Goal-Achievement Summary

| Stated Goal (README) | Status | Blocking Findings |
|----------------------|--------|-------------------|
| Widget system (Button/Label/TextInput/…) with `SetLayout` | ❌ | CRIT-2 (display list emits nothing), HIGH-13 (`Panel.Add` drops widgets) |
| Wayland protocol implementation | ❌ | CRIT-3 (event args never decoded), CRIT-4 (FDs discarded), HIGH-1/2/3/4 |
| X11 protocol implementation | ⚠️ | HIGH-5 (partial reads), HIGH-6 (unbounded setup), MED-1 (button swap) |
| Display-server / renderer auto-detection | ⚠️ | CRIT-1 (Wayland window creation self-deadlocks) |
| Input handling (pointer, keyboard, touch) | ❌ | HIGH-11 (`TextInput` rune never set), HIGH-12 (X11 keycodes not translated → Tab nav dead) |
| Clipboard read/write (Wayland + X11) | ⚠️ | MED-15 (Wayland source goroutine leak), LOW-1 (X11 INCR/256 KB truncation) |
| Double/triple buffering frame sync | ⚠️ | HIGH-7/8 (lost-wakeup races stall acquisition) |
| AT-SPI2 accessibility | ⚠️ | HIGH-9 (`GetText` panic), MED-9 (focus signal name invalid) |
| Fully static zero-dependency binary | ➖ | Not verifiable in this sandbox (Rust lib absent); CI builds pure-Go only |

## Findings

Severities follow the prompt's classification. Every CRITICAL/HIGH item lists a
concrete data-flow/code path. Line numbers are from the audited tree.

### CRITICAL

- [ ] **CRIT-1 — Wayland window creation self-deadlocks** — `app.go:322` → `app.go:456` — concurrency/deadlock — `App.NewWindow` takes `a.mu.Lock()` (line 322, `defer Unlock`) and, still holding it, calls `win.initialize()` → `initWaylandWindow()` → `createWaylandSurface()` which executes `w.app.mu.Lock()` at `app.go:456`. `a.mu` is a non-reentrant `sync.Mutex` (`app.go:103`), so the same goroutine deadlocks; **every** Wayland `NewWindow` call hangs forever. (Not caught by tests: requires a live compositor.) — **Remediation:** build and platform-initialize the window without holding `a.mu`; take the lock only to append to `a.windows` and to write `surfaceToWindow` (drop the inner lock in `createWaylandSurface`). Validate with a fake-compositor `NewWindow` test under `go test -race ./...`.

- [ ] **CRIT-2 — Public widgets render nothing through `SetLayout`** — `render.go:149` (+ `app.go:1000`, `publicwidget.go:262`) — API/behavioral contract — `RenderBridge.walkWidget` only emits for widgets implementing `DisplayListEmitter`. No production widget implements `EmitDisplayList`; a repository-wide search finds it only on the test mock `mockWidget` (`render_test.go:65`). `Button`/`Label`/etc. expose `Draw(Canvas)`, and a `displayListCanvas` bridge exists (`publicwidget.go:262`) but `newDisplayListCanvas` is **never called** anywhere. Data flow: `Window.SetLayout(col)` → `newLayoutAdapter` → `RenderFrame` → `renderBridge.Render` → `walkWidget` finds no emitter → empty display list → blank frame. The README's headline "Widget System" examples therefore produce blank windows. — **Remediation:** make `layoutAdapter`/concrete widgets implement `EmitDisplayList` (or have `walkWidget` invoke `Draw` via `newDisplayListCanvas`). Validate with a display-list assertion test that `SetLayout(NewButton(...))` emits non-zero commands; `go test -race ./...`.

- [ ] **CRIT-3 — Wayland event payloads are read but never decoded (Args always nil)** — `internal/wayland/client/connection.go:288-292` — protocol/logic — `ReadMessage` reads the payload into `c.eventBuffer` then returns `&wire.Message{Header: header}` with `Args == nil`. `DispatchMessage` (`:312`) hands that nil slice to every object's `HandleEvent`. All argument-bearing compositor events (registry `global`, `xdg_surface.configure`, `wl_pointer.motion`, `wl_keyboard.key`, `wl_output.geometry`, `wl_buffer.release`, …) arrive with no data, so the documented Wayland event handling cannot function. (Not caught by tests: no live compositor in CI.) — **Remediation:** decode the payload per the target object/interface event signature and populate `msg.Args` before dispatch; reject short/leftover payload. Validate with an integration test feeding a captured `wl_registry.global` byte stream and asserting populated `Args`.

- [ ] **CRIT-4 — Incoming Wayland file descriptors are silently closed** — `internal/wayland/client/connection.go:240,264` → `internal/wayland/socket/connection.go:149-153` — resource/protocol — header and payload are read with `RecvMsg(buf, 0)` (maxFDs = 0). `socket.RecvMsg` closes every received SCM_RIGHTS fd when `len(fds) > maxFDs` (lines 149-153) and returns none. Thus `wl_keyboard.keymap`, `wl_data_offer` and `zwp_linux_dmabuf` fds are closed before any handler can `mmap`/consume them — keymap loading and clipboard/DMA-BUF receive are impossible. — **Remediation:** receive each full message (header+body+ancillary) once with an appropriate `maxFDs`, retain the fds, and bind them to fd-typed event args. Validate by sending a keymap fd over a socketpair and asserting it is mappable.

### HIGH

- [ ] **HIGH-1 — Wayland message header size is unvalidated before slicing** — `internal/wayland/client/connection.go:250-254,264` — nil/bounds + logic — the header is hand-decoded (bypassing `wire.DecodeHeader`'s validation); `readMessagePayload` then slices `c.eventBuffer[wire.HeaderSize:header.Size]` where `eventBuffer` is 4096 bytes (`connection.go:103`) and `header.Size` is a compositor-controlled `uint16` (≤65535). `Size > 4096` panics (slice out of range); `Size < 8` is accepted as empty and desynchronizes the stream. — **Remediation:** validate `MinMessageSize ≤ Size ≤ MaxMessageSize` (or call `wire.DecodeHeader`) before slicing. Validate: `go test ./internal/wayland/client ./internal/wayland/wire`.

- [ ] **HIGH-2 — Core Wayland objects have no event handler** — `internal/wayland/client/connection.go:306-310`, `client/registry.go`, `client/display.go` — logic/protocol — `DispatchMessage` silently returns nil for objects not implementing `EventHandler`. `Display`, `Registry`, and `Callback` do not implement `HandleEvent`, so `wl_registry.global` (global discovery), `wl_callback.done` (roundtrip completion), and `wl_display.error`/`delete_id` are dropped. Combined with CRIT-3 this leaves the registry empty. — **Remediation:** implement `HandleEvent` on display, registry, and callback; do not silently ignore core protocol events. Validate: registry-global dispatch populates `Registry.globals`.

- [ ] **HIGH-3 — fd request arguments use the wrong dynamic type** — `internal/wayland/shm/protocol.go:120`, `internal/wayland/datadevice/manager.go:512`, `internal/wayland/dmabuf/protocol.go:200` — type assertion/logic — `wire.encodeFDArg` requires `Value.(int)` (`internal/wayland/wire/protocol.go:458-463`) but these callers pass `int32`. The type assertion fails, so `wl_shm.create_pool`, `wl_data_offer.receive`, and `zwp_linux_buffer_params.add` cannot send their fds. — **Remediation:** pass `int` (or make `encodeFDArg` accept both). Validate: `go test ./internal/wayland/shm ./internal/wayland/datadevice ./internal/wayland/dmabuf`.

- [ ] **HIGH-4 — SHM buffer pixel slices dangle after pool resize** — `internal/wayland/shm/pool.go:149-155`, `internal/wayland/shm/buffer.go:44-45` — resource/aliasing — `Pool.Resize` `munmap`s and re-`mmap`s the pool, but already-created `Buffer.pixels` slices still reference the unmapped region. A subsequent `Buffer.Pixels()` read/write touches freed memory → SIGSEGV or corruption. — **Remediation:** after remap, re-slice every `p.buffers` entry against the new mapping, or compute `Buffer.Pixels()` lazily from the current pool base. Validate: create buffer → resize → access pixels under `go test ./internal/wayland/shm`.

- [ ] **HIGH-5 — X11 replies use `Read` instead of `io.ReadFull` (partial-read corruption)** — `internal/x11/client/connection.go:417,433,585,624` — logic/protocol — these sites read fixed-size reply headers/bodies with `c.conn.Read`, which may return fewer bytes than requested on a stream socket, leaving zero-filled trailing bytes that corrupt parsed fields. The sibling `internal/x11/wire` package uses `io.ReadFull` everywhere (`wire/protocol.go:312…`, `wire/setup.go:187…`), confirming this is an inconsistency, not intent. `dataLen*4` at `:432` can also overflow `uint32`. — **Remediation:** use `io.ReadFull`; bound `dataLen` and compute byte length via `uint64`. Validate: `go test ./internal/x11/client`.

- [ ] **HIGH-6 — X11 setup reply decoded by count without bounding to announced length** — `internal/x11/wire/setup.go` (success path, ~`:467-506`) — logic/resource — the setup `length` field is read then discarded; server-controlled `numFormats`/`numScreens`/`numDepths`/`numVisuals` drive `make()` and nested reads with no bound against the declared body size. A malicious/buggy server can force large allocations or block the client awaiting records that never arrive. — **Remediation:** read the body into a bounded `length*4` buffer and decode counts only within it. Validate: `go test ./internal/x11/wire`.

- [ ] **HIGH-7 — Buffer ring lost wakeup on release (frame-sync stall)** — `internal/buffer/ring.go:237-241` vs `:172-183` — concurrency — `AcquireForWriting` waits on `r.cond` (guarded by `r.mu`), but the acquire predicate is slot state mutated under `slot.mu`, and `MarkReleased` calls `r.cond.Broadcast()` (`:241`) **without holding `r.mu`**. If a release's broadcast lands between an acquirer's failed scan and its `cond.Wait()`, the wakeup is lost and rendering stalls until the next unrelated release or context cancel. — **Remediation:** perform the slot transition and `Broadcast` under `r.mu` (the same mutex the waiter holds). Validate: `go test ./internal/buffer -run TestRing_ConcurrentAcquire -count=1000 -race`.

- [ ] **HIGH-8 — Buffer ring cancellation wakeup can also be lost** — `internal/buffer/ring.go:169,179-182` — concurrency/context — the `context.AfterFunc(ctx, …)` callback broadcasts `r.cond` without holding `r.mu`. If cancellation fires between the `ctx.Err()` check (`:179`) and `cond.Wait()` (`:182`), the broadcast is missed and the goroutine blocks despite a cancelled context (context leak). — **Remediation:** lock `r.mu` inside the AfterFunc callback before broadcasting. Validate: `go test ./internal/buffer -run TestRing_AcquireTimeout -count=1000 -race`.

- [ ] **HIGH-9 — AT-SPI `GetText` start offset not clamped → panic** — `internal/a11y/text_iface.go:15-26,37-41` — nil/bounds (D-Bus-reachable) — `clampOffsets` clamps `end` but not `start`. For `GetText(999, -1)` on `"hello"`, `end` becomes 5 then `end = start = 999`, returning `(999, 999)`; `runes[from:to]` panics. A screen-reader (or any D-Bus client) can crash the process/method goroutine. — **Remediation:** clamp `start` and `end` independently to `[0, len(runes)]` then enforce `start ≤ end`. Validate: `go test ./internal/a11y -tags=atspi` with `GetText(999,-1)` / `GetText(999,1000)` cases.

- [ ] **HIGH-10 — DMA-BUF fd leak on framebuffer reuse** — `internal/render/display/present_helper.go:26-38` (+ `wayland.go:143-154`, `x11.go:188-199`) — resource — `renderer.Present()` exports a fresh fd each call; when `fb.Fd >= 0` (reused framebuffer) the newly returned fd is neither stored nor closed. One fd leaks per reused-framebuffer present until fd exhaustion. — **Remediation:** only call `Present()` when `fb.Fd < 0`, else close the returned fd immediately. Validate: watch `/proc/$PID/fd` across repeated presents (display tests require the Rust lib).

- [ ] **HIGH-11 — `TextInput` inserts characters from an always-empty rune** — `concretewidgets.go:463` (+ `event.go:329`) — input logic — `TextInput.HandleEvent` inserts `string(e.Rune())`, but neither the X11 nor Wayland key translators populate the event rune/text. Focused text inputs receive NUL bytes / nothing, and Backspace/Delete are not handled separately. — **Remediation:** populate rune/text from the keymap (and handle editing keys) in the translators; insert real text in `TextInput`. Validate with X11/Wayland key-event tests asserting typed text.

- [ ] **HIGH-12 — X11 key events expose raw keycodes, breaking key constants & Tab navigation** — `event.go:333` — API/logic — X11 `KeyPress` detail (a hardware keycode) is stored directly as `KeyEvent.Key`, but the dispatcher compares against logical constants (`KeyTab`, etc.). On X11, Tab focus navigation and all key-constant checks fail. — **Remediation:** translate X11 keycodes to keysyms/modifiers/runes before populating `KeyEvent`. Validate: an X11 Tab event advances focus in a dispatcher test.

- [ ] **HIGH-13 — `Panel.Add` silently drops leaf widgets** — `layout.go:137` — logic/API contract — `Panel.Add(child PublicWidget)` switches only on `*Panel`/`*Row`/`*Column`/`*Stack`/`*Grid`; any other `PublicWidget` (Button, Label, TextInput — exactly what the README examples add) hits no case and is discarded with no error. — **Remediation:** handle the general `PublicWidget` case (wrap via adapter) or return/panic on unsupported types. Validate: a test asserting `panel.Add(NewButton(...))` increases child count and renders.

- [ ] **HIGH-14 — Window callbacks invoked while holding `w.mu` (reentrancy deadlock)** — `app.go:1216-1226` (`onResize`) and `app.go:848-869` (`onClose`) — concurrency — both call user callbacks while holding `w.mu`. A callback that calls `Size()`, `Close()`, or any locked setter self-deadlocks. — **Remediation:** snapshot the callback + needed state under the lock, unlock, then invoke. Validate with a reentrant-callback test.

- [ ] **HIGH-15 — Dispatcher invokes handlers under `RLock` (reentrancy deadlock)** — `dispatcher.go:96` — concurrency — `Dispatch` holds `d.mu.RLock()` while calling widget and registered handlers; a handler that calls `OnPointer`/`OnKey`/`SetWidgetRoot` (which take `d.mu.Lock()`) blocks forever. — **Remediation:** copy handlers/root under the read lock, release it, then call. Validate with a reentrant-handler test.

- [ ] **HIGH-16 — `atlas.UploadImageData` can panic on short pixel slices** — `internal/render/atlas/texture.go:360-390` — nil/bounds — the function validates dimensions but not `len(pixels)`; `pixels[srcOffset:srcOffset+srcStride]` panics if the caller passes fewer than `width*height*4` bytes. Malformed image input crashes the renderer. — **Remediation:** require `len(pixels) >= width*height*4` (overflow-checked) and bounds-check the destination. Validate: short-slice test under `go test ./internal/render/atlas` (needs Rust lib).

- [ ] **HIGH-17 — `composite.Blit` ignores clipped destination origin → negative-index panic** — `internal/raster/composite/ops.go:47` — nil/bounds + doc mismatch — `calculateClippedRegion` computes clipped `dstX1/dstY1` (lines 61-64) but `Blit` passes the **original** `dstX/dstY` to `blitRows` (line 47). For `Blit(dst, -1, 0, src, 0, 0, 10, 10)`, `blitRows` computes `dstOffset = dstY*stride + (-1)*4` → negative `dst.Pixels[dstOffset:]` index → panic, despite the doc claiming "Coordinates are automatically clipped to buffer bounds." — **Remediation:** return and use the clipped `dstX1/dstY1`. Validate: add a negative-destination blit test; `go test ./internal/raster/composite`.

### MEDIUM

- [ ] **MED-1 — X11 middle/right mouse buttons are swapped** — `event.go:357,381` — logic — the map `PointerButton(0x110 + detail - 1)` assumes sequential Linux codes, but the constants are `Left=0x110`, `Right=0x111`, `Middle=0x112` (`event.go:76-80`). X11 detail 2 (middle) → `0x111` (Right) and detail 3 (right) → `0x112` (Middle), so middle and right clicks are reported swapped (both press and release). — **Remediation:** map explicitly `1→Left, 2→Middle, 3→Right`. Validate: a button-translation unit test.

- [ ] **MED-2 — X11 received fds leaked on reply-data error** — `internal/x11/client/connection.go:538,543` — resource — fds installed via `ReadMsgUnix` are not closed if `readAdditionalReplyData` later fails, letting a server exhaust the client fd table with fds + malformed data. — **Remediation:** extract fds immediately and close them on every later error path. Validate: `go test ./internal/x11/client`.

- [ ] **MED-3 — X11 setup-failure decode over-reads** — `internal/x11/wire/setup.go:220-223` — logic — `failDataLen` already covers the reason string + padding (in 4-byte units); the code reads `reasonLen` bytes then skips `failDataLen*4` more, consuming bytes from the next message and desynchronizing. — **Remediation:** skip `int(failDataLen)*4 - int(reasonLen)` (validated ≥0). Validate: `go test ./internal/x11/wire`.

- [ ] **MED-4 — X11 `getSelection` accepts unrelated SelectionNotify** — `internal/x11/selection/manager.go:181,186,193` — logic/security — only the property atom from `HandleSelectionNotify` is used; requestor/selection/target/timestamp are not validated against the pending request, so a spurious `SelectionNotify` can steer `GetProperty(..., delete=true)` to read/delete an arbitrary property. — **Remediation:** validate the full SelectionNotify fields against the in-flight request. Validate: `go test ./internal/x11/selection`.

- [ ] **MED-5 — Layout flex distribution loses/overflows pixels** — `internal/ui/layout/flex.go:253-258` — logic/rounding — each grow/shrink share is truncated with `int(...)` independently, so leftover pixels vanish and small shrink deficits round to 0. Example: three 34px children shrinking into 100px each shrink by `int(2/3)=0`, leaving 102px that overflows the container. — **Remediation:** use largest-remainder distribution so allocated sizes sum exactly to the available space. Validate: `go test ./internal/ui/layout` with `3×34px→100px` and `3 grow→100px` cases.

- [ ] **MED-6 — `composite.BlitScaled` samples outside the requested source rect** — `internal/raster/composite/ops.go:141-158` — logic — bounds are checked against `src.Width/Height`, not the requested sub-rectangle, and `scaledSourceCoord` always uses `s1=s0+1`, so 1px-wide/high source rects are skipped and sub-rects sample neighbouring pixels. — **Remediation:** clamp samples to `[srcX, srcX+srcWidth-1] × [srcY, srcY+srcHeight-1]`. Validate: 1×1 and sub-rect scaling tests; `go test ./internal/raster/composite`.

- [ ] **MED-7 — 1px horizontal/vertical lines produce zero-area damage** — `internal/raster/displaylist/damage.go:174-180,225` — logic — `w := data.Width / 2` is 0 for width 1, so a vertical line gets `Width:0` (horizontal gets `Height:0`); `rectsIntersect` rejects it and `FilterCommandsByDamage` omits the line on damaged renders. — **Remediation:** use a ceil half-width and `max(1, …)` dimensions including endpoints. Validate: a damage-filter test for 1px lines; `go test ./internal/raster/displaylist`.

- [ ] **MED-8 — Damaged software renders do not clear the damaged region** — `internal/render/backend/software.go:66-87` — logic/aliasing — the comment claims damaged regions are cleared then re-rendered, but the code only filters commands and draws over the existing buffer, so removed/moved content persists and translucent draws accumulate alpha across frames. — **Remediation:** clear each damage rect before `RenderCommands`, or fall back to a full redraw. Validate: a two-frame damage test (needs Rust lib).

- [ ] **MED-9 — AT-SPI focus signal name is invalid (events never emit)** — `internal/a11y/manager.go:197` — error handling/D-Bus — `Emit(path, "org.a11y.atspi.Event.Focus:Focus", …)` yields member `Focus:Focus`; `:` is an illegal D-Bus member char, so `Emit` returns an error that is discarded. Screen readers are never told about focus changes. — **Remediation:** use a valid interface/member (e.g. `org.a11y.atspi.Event.Focus`, member `Focus`) and propagate the error. Validate: `go test ./internal/a11y -tags=atspi` asserting a valid signal name.

- [ ] **MED-10 — Atlas accepts zero/negative regions** — `internal/render/atlas/texture.go:176-183,274-304` — nil/bounds — `AllocateImageRegion` rejects only regions larger than the page; `width/height ≤ 0` create degenerate/negative shelf positions and UVs, enabling overlapping allocations and corrupt sampling. — **Remediation:** error on `width ≤ 0 || height ≤ 0` and validate `imagePageSize > 0` in `New`. Validate: invalid-dimension tests (needs Rust lib).

- [ ] **MED-11 — Race on `releasedAt` in framebuffer selection** — `internal/render/display/framebuffer.go:87-93,186-189` (+ `wayland.go:132-134`, `x11.go:177-179`) — concurrency — `findOldestAvailable` reads `fb.releasedAt` after releasing `fb.mu`, while pipeline release paths write `releasedAt` via `setState` without the pool lock. — **Remediation:** read state + `releasedAt` under `fb.mu`, or route all releases through pool-locked logic. Validate: `go test -race ./internal/render/display` (needs Rust lib).

- [ ] **MED-12 — Stale initial release signal in framebuffer pool** — `internal/render/display/framebuffer.go:97-103,137-138,174-177` — concurrency — framebuffers are pre-signalled available; `Acquire` does not drain `releaseChan`, so `WaitRelease` can return immediately for a framebuffer that was never actually released, risking premature reuse/out-of-order display. — **Remediation:** drain `releaseChan` on acquire, or replace the channel with a state-tied condition. Validate: a test asserting `WaitRelease` blocks right after `Acquire`.

- [ ] **MED-13 — `Window.Close` marks closed before platform cleanup succeeds** — `app.go:855` — resource/error handling — `w.closed = true` is set before `DestroyWindow`; if destroy fails, a later `Close` returns nil early and the window/resources can never be cleaned up. — **Remediation:** set `closed` only after successful platform teardown (or keep retry state). Validate with a failing-mock destroy test.

- [ ] **MED-14 — `FocusManager.Focus` never clears the previous focus** — `dispatcher.go:290` — logic — `Focus(w)` marks the new widget focused but does not unfocus the previously focused widget, so multiple widgets simultaneously report focused. — **Remediation:** clear the old focused index before setting the new one. Validate: a two-widget focus test.

- [ ] **MED-15 — Wayland clipboard source goroutine can leak** — `clipboard.go:80` — resource/concurrency — the `serveClipboardSource` goroutine is started before `SetSelection`; if `SetSelection` fails, the goroutine can block forever on send/cancel. — **Remediation:** call `SetSelection` first, or cancel/close the source on error. Validate: a failing-`SetSelection` test with a goroutine-leak check.

- [ ] **MED-16 — Wayland drop read error swallowed** — `app.go:2195` — error handling — the drag-and-drop `ReadData` error is ignored, so the drop handler is invoked with nil/truncated data and no indication of failure. — **Remediation:** surface the read failure (skip the drop or report via an error callback). Validate: a mock-offer read-failure test.

- [ ] **MED-17 — Keymap fd/mmap lifecycle leaks** — `internal/wayland/input/keymap.go:99-104`, `internal/wayland/input/keyboard.go:87-90` — resource — `NewKeymap` does not close `fd` when `syscall.Mmap` fails; `Keyboard.HandleKeymap` replaces `k.keymap` without unmapping the previous one. Repeated keymap events leak fds/mmaps. — **Remediation:** always close the fd after the mmap attempt and unmap the old keymap before replacing. Validate: an fd/mmap-count test around repeated keymap events.

### LOW

- [ ] **LOW-1 — X11 clipboard truncates large data and mishandles INCR** — `internal/x11/selection/manager.go:193,198` — logic — `getSelection` requests `length=65536` 4-byte units (≈256 KB cap) and ignores both `actualType` and `bytesAfter`, so larger selections are silently truncated and ICCCM `INCR` transfers return the marker rather than the data. — **Remediation:** inspect `actualType`, loop `GetProperty` until `bytesAfter==0`, and implement INCR. Validate: `go test ./internal/x11/selection`.

- [ ] **LOW-2 — Clipped box-shadow mask is mis-offset at edges** — `internal/raster/effects/visual.go:47,54,78-81` — logic — when `shadowX/shadowY` are clipped to bounds, `createShadowMask` still uses the unclipped origin, so partially off-screen shadows shift/disappear near edges. — **Remediation:** compute mask core coordinates relative to the clipped origin. Validate: an edge-clipped shadow golden test; `go test ./internal/raster/effects`.

- [ ] **LOW-3 — Framebuffer pool `Close` swallows `syscall.Close` errors** — `internal/render/display/framebuffer.go:258-261` — error handling — close errors are ignored while fds are still set to `-1`, hiding leaks. — **Remediation:** collect and return close errors; clear fd only on success. Validate: an invalid-fd injection test (needs Rust lib).

- [ ] **LOW-4 — Atlas `Destroy` stops at the first page error** — `internal/render/atlas/texture.go:502-505` — resource — returning on the first failed page leaks the remaining pages' GPU buffers. — **Remediation:** attempt all destroys and return a joined error. Validate: a mock failing-page test (needs Rust lib).

- [ ] **LOW-5 — `BasePublicWidget.Children` returns the internal slice** — `publicwidget.go:205` — aliasing/contract — the `Container.Children` doc says mutating the returned slice does not affect state, but `BasePublicWidget.Children` returns `w.children` directly (note: `Panel.Children()` correctly copies). — **Remediation:** return a copy. Validate: a mutation-isolation test.

- [ ] **LOW-6 — X11 extended buttons mis-default to PointerMove** — `internal/integration/events.go:215-239,245-257` — logic — for `ButtonPress`, details outside 1..5 leave `pe.eventType` at its zero value `PointerMove`, so button 6+ presses are reported as motion. — **Remediation:** have `applyX11ButtonPress` report unsupported details and return nil/explicit mapping. Validate: `go test ./internal/integration -run TestTranslateX11Pointer`.

- [ ] **LOW-7 — Event timestamps use `time.Now()` instead of server time** — `internal/integration/events.go:168,198,216,276` (and `event.go` translators) — logic — translators stamp events with wall-clock receipt time, ignoring the protocol `Time` field, which breaks latency/ordering/double-click timing. — **Remediation:** derive `Timestamp()` from the protocol timestamp (or correct the documented contract). Validate: a test setting event `Time` and asserting the translated timestamp.

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Total functions | 662 (+1180 methods) |
| Functions above complexity 15 | 1 (`getSelection`, overall 15.3 / cyclomatic 11) |
| Functions > 50 lines | 7 (0.4%) |
| Avg cyclomatic complexity | 3.2 |
| Doc coverage | 91.4% (pkg 100%, func 98.3%, type 91.3%, method 89.6%) |
| Duplication ratio | 0.68% (238 lines) |
| Test pass rate | 35/35 pure-Go test packages pass `-race`; 16 packages un-buildable here (Rust `librender_sys.a` absent — environmental); 24 packages have no tests |
| go vet warnings | 0 |
| Dead code (unreferenced funcs) | 19 (per go-stats-generator) |
| TODO/FIXME/HACK/BUG markers | 0 / 0 / 0 / 0 |

## False Positives Considered and Rejected

| Candidate | Reason Rejected |
|-----------|-----------------|
| Rust cgo link failures in 16 packages | Environmental (missing `librender_sys.a`), not a code defect; excluded per Phase 3l. |
| `wire.DecodeString`/`DecodeArray` (wayland) OOB | Length capped at `MaxMessageSize-HeaderSize` before `make`/index. |
| `wire.readPadding`/`writePadding` | Padding mathematically bounded to 0..3. |
| `xdg/toplevel.go` state parse OOB | Loop guard `i+3 < len(statesData)` prevents overrun. |
| `internal/wayland/shm/pool.go` offset math | Uses positive checks + `int64` before slicing. |
| `internal/x11/present/extension.go:251` request length | Length 18 is correct including the 4-byte header. |
| `internal/x11/events` fixed-event parsers | Check `len(data) < 28` before slicing 32-byte events. |
| `internal/x11/shm` `GetBuffer` `unsafe.Slice` | Validates nil addr, negative size, and caps size first. |
| `internal/x11/dri3` `Open` unexpected fds | Explicitly closed on the error path. |
| `raster.NewBuffer` allocation overflow | Max 16384×16384 keeps `w*4*h` within signed 32-bit range. |
| `SoftwareBackend.Pixels` returns backing slice | Documented intentional direct-access API. |
| GPU scissor negative coords | Production damage path clamps via `GPUBackend.processDamage`. |
| `internal/ui/scale` DPI division | Constant `96.0` denominator; zero DPI normalized by `Set`. |
| `internal/ui/animation` duration division | Guarded by `Duration <= 0`. |
| `internal/ui/widgets` scrollbar divisions | Guarded by `contentHeight > height` (positive denominator). |
| `a11y` `objectPath()`/`Export` ignored error | Path built from a constant valid prefix + numeric ID. |
| `resource.go:168` ignored image `Close` | Read-only load; decode errors already propagated, close error non-actionable. |
| `AccessibilityManager.Close()` nil deref | Method guards `am != nil`. |
| `imageToRGBA` bounds | Loop bounds derive from decoded dimensions / allocated buffer. |
| Unchecked `EventHandler` type assertions (wayland) | Not currently compositor-reachable because args are not yet decoded (see CRIT-3); becomes relevant once decoding is added. |

## Remaining Scope

A complete pass was performed across all 40 Go packages; the final pass produced
no new confirmed findings above LOW. Items below are explicitly **out of scope**
for this Go audit (not "unaudited"):

| Area | Status | Notes |
|------|--------|-------|
| `render-sys/` (Rust) | Out of scope | C-ABI/Rust GPU backend; this audit targets Go. Its absence blocks building 16 Go packages here, which were instead audited by reading. |
| Live end-to-end runtime (real compositor / X server / GPU) | Not exercisable | Sandbox has no display server or GPU; CRIT-1/3/4 and several HIGH items are confirmed by code reading and cannot be reproduced via `go test` in this environment. |
