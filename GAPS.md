# Implementation Gaps — 2026-05-30

Gaps between what the Wain README/docs claim and what the code actually does.
Each gap references the confirmed findings in `AUDIT.md`.

## Widget system renders nothing through the public API

- **Stated Goal**: README "Features" → "Widget System — Button, Label,
  TextInput, ScrollView, ImageWidget, Spacer" and the Usage examples build a
  layout with `col.Add(label)`, `col.Add(btn)`, then `win.SetLayout(col)`.
- **Current State**: `RenderBridge.walkWidget` (`render.go:149`) only emits draw
  commands for widgets implementing `DisplayListEmitter`. No production widget
  implements `EmitDisplayList` — it exists only on a test mock
  (`render_test.go:65`). The `displayListCanvas` bridge that would translate the
  public `Draw(Canvas)` API into the display list (`publicwidget.go:262`) is
  never invoked. Additionally `Panel.Add` (`layout.go:137`) silently discards
  any non-container child such as a `Button` or `Label`.
- **Impact**: The headline "minimal application" from the README produces a
  blank window: widgets are added and laid out but never drawn. This is the
  toolkit's central promise and it is non-functional through the documented API.
- **Closing the Gap**: Implement `EmitDisplayList` on the layout/widget adapters
  (or have `walkWidget` call `Draw` via `newDisplayListCanvas`), and make
  `Panel.Add` accept all `PublicWidget` types. Add a display-list regression
  test asserting `SetLayout` of a button/label tree emits non-empty commands.
  (See AUDIT CRIT-2, HIGH-13.)

## Wayland protocol implementation cannot decode events or receive fds

- **Stated Goal**: README → "Wain implements Wayland and X11 display protocols
  directly" and "Wayland Protocol — compositor connection, `wl_shm`,
  `xdg_shell`, input, clipboard, DMA-BUF, and output handling".
- **Current State**: `Connection.ReadMessage` reads the event payload but
  returns a message with `Args == nil` (`internal/wayland/client/connection.go:288-292`),
  so every argument-bearing event reaches its handler empty. Incoming file
  descriptors are received with `RecvMsg(buf, 0)` and immediately closed by the
  socket layer (`socket/connection.go:149-153`), so keymap/DMA-BUF/data-offer
  fds never arrive. `Display`, `Registry`, and `Callback` implement no
  `HandleEvent`, so global discovery and roundtrips are dropped, and fd request
  args are encoded with the wrong dynamic type (`int32` vs the required `int`),
  breaking `wl_shm.create_pool`, `wl_data_offer.receive`, and dmabuf plane adds.
- **Impact**: The documented Wayland backend cannot process input, configure
  events, buffer releases, clipboard, or keymaps. Because there is no live
  compositor in CI, tests do not exercise these paths, so the gap is invisible
  to the test suite.
- **Closing the Gap**: Decode payloads per event signature into `msg.Args`;
  receive and retain ancillary fds; implement `HandleEvent` on display/registry/
  callback; pass `int` fd args. Add integration tests that replay captured
  compositor byte streams. (See AUDIT CRIT-3, CRIT-4, HIGH-1, HIGH-2, HIGH-3.)

## Wayland window creation deadlocks

- **Stated Goal**: README → "Display Server Auto-Detection — connects to Wayland
  when available" and the Usage example calls `app.NewWindow(...)`.
- **Current State**: `App.NewWindow` holds `a.mu` (`app.go:322`) across
  `win.initialize()`, and `createWaylandSurface` re-locks the same non-reentrant
  `sync.Mutex` (`app.go:456`), self-deadlocking on every Wayland window
  creation.
- **Impact**: On a Wayland session the first `NewWindow` hangs forever; no window
  ever appears.
- **Closing the Gap**: Do not hold `a.mu` during platform initialization; lock
  only to mutate shared app maps/slices. (See AUDIT CRIT-1.)

## Keyboard input and focus navigation do not work

- **Stated Goal**: README → "Widget System … TextInput" and "Pointer, keyboard,
  touch input"; `TextInput` is documented as an editable text field and Tab is a
  recognized navigation key.
- **Current State**: `TextInput.HandleEvent` inserts `string(e.Rune())`
  (`concretewidgets.go:463`) but the key translators never populate a rune. On
  X11, raw hardware keycodes are stored as `KeyEvent.Key` (`event.go:333`) while
  the dispatcher compares against logical constants like `KeyTab`, so Tab focus
  navigation and key-constant handling fail. X11 middle/right mouse buttons are
  also swapped (`event.go:357`).
- **Impact**: Users cannot type into text inputs, cannot Tab between widgets on
  X11, and middle/right clicks are misreported.
- **Closing the Gap**: Translate keycodes→keysyms/runes/modifiers in the
  Wayland/X11 translators; insert real text in `TextInput`; fix the X11 button
  map to `1→Left, 2→Middle, 3→Right`. (See AUDIT HIGH-11, HIGH-12, MED-1.)

## Frame synchronization can stall (lost wakeups)

- **Stated Goal**: README → "Double/Triple Buffering — frame synchronization
  with compositor (`internal/buffer/`)".
- **Current State**: In `internal/buffer/ring.go`, the acquire predicate (slot
  state) is mutated under `slot.mu`, but `MarkReleased` and the context-cancel
  callback broadcast `r.cond` without holding `r.mu` (`ring.go:169,237-241`).
  This violates `sync.Cond` discipline and allows a release/cancel signal to be
  lost if it races with a waiter about to call `Wait()`.
- **Impact**: A renderer waiting for a free buffer can block until an unrelated
  later release, causing intermittent frame stalls; a cancelled acquire can hang
  despite the cancelled context.
- **Closing the Gap**: Mutate slot state and broadcast under the same `r.mu` the
  waiters hold. Add high-iteration `-race` concurrency tests. (See AUDIT HIGH-7,
  HIGH-8.)

## Accessibility focus events are never delivered

- **Stated Goal**: README → "AT-SPI2 Accessibility — D-Bus screen reader
  integration with Accessible, Component, Action, and Text interfaces".
- **Current State**: `emitFocusEvent` emits a signal named
  `org.a11y.atspi.Event.Focus:Focus` (`internal/a11y/manager.go:197`); the `:`
  makes the D-Bus member name invalid, so `Emit` fails and the error is
  discarded. Separately, `Text.GetText` panics on out-of-range start offsets
  (`text_iface.go:15-26`).
- **Impact**: Screen readers are not notified of focus changes, and a malicious
  or buggy AT-SPI client can crash the application by requesting a bad text
  range — undermining the accessibility guarantee.
- **Closing the Gap**: Use a valid signal interface/member and propagate the
  error; clamp `GetText` offsets independently. (See AUDIT MED-9, HIGH-9.)

## Clipboard cannot transfer large content

- **Stated Goal**: README → "Clipboard — read/write clipboard on both Wayland
  and X11".
- **Current State**: The X11 `getSelection` reads at most ~256 KB and ignores
  `bytesAfter`, with no ICCCM `INCR` support
  (`internal/x11/selection/manager.go:193`); larger pastes are silently
  truncated. The Wayland clipboard source goroutine can also leak if
  `SetSelection` fails (`clipboard.go:80`).
- **Impact**: Pasting large text/data from other applications loses content
  beyond 256 KB and provides no error indication.
- **Closing the Gap**: Loop `GetProperty` until `bytesAfter == 0` and implement
  INCR; start the Wayland source only after a successful `SetSelection`.
  (See AUDIT LOW-1, MED-15.)

## Layout sizing does not exactly fill containers

- **Stated Goal**: README → "Layout Containers — Row, Column, Stack, Grid, and
  Panel with flexbox-style alignment, padding, and gap".
- **Current State**: `distributeFlex` (`internal/ui/layout/flex.go:253-258`)
  truncates each flex share independently, dropping leftover pixels on grow and
  rounding small shrink deficits to zero — so children can overflow or
  underfill the container by a few pixels.
- **Impact**: Pixel-imperfect layouts; flex children can overflow their parent.
- **Closing the Gap**: Use largest-remainder distribution so child sizes sum
  exactly to the available axis length. (See AUDIT MED-5.)

## "Fully static, zero-dependency binary" is unverified in this environment

- **Stated Goal**: README → "Fully Static Binaries … output binaries have zero
  runtime dependencies" and "run on any Linux distribution".
- **Current State**: Building the toolkit requires the Rust `librender_sys.a`
  plus musl toolchain (`make build`); without it, 16 Go packages (including the
  root `wain` package) do not link. The pure-Go test suite passes, but no
  end-to-end static binary could be produced or run in this sandbox, and CI
  (per the README badge/workflow) exercises only the pure-Go portions.
- **Impact**: The flagship static-linking claim is plausible but not validated
  by the automated tests available here; regressions in the CGO/static-link path
  would not be caught by `go test ./...` alone.
- **Closing the Gap**: This is a verification gap rather than a code defect — add
  a CI job that runs `make build` + `make check-static` on a runner with the
  Rust/musl toolchain so the static-linkage guarantee is continuously asserted.
