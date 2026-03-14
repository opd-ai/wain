# Wain Tutorial: Building a Simple Form Application

This tutorial walks through building a contact-form application with wain. By the end
you will have a window with a labelled text field, a submit button, themed colours,
and clipboard read/write.

---

## Prerequisites

```bash
go get github.com/opd-ai/wain
```

A C compiler for musl (`musl-gcc`) is required for the fully-static build; the pure-Go
development path (software renderer, no GPU) works with the standard `gcc` toolchain.

---

## 1. Hello, App

Every wain program starts with an `App` and ends with `app.Run()`.

```go
package main

import (
    "fmt"
    "log"

    "github.com/opd-ai/wain"
)

func main() {
    app := wain.NewApp()

    app.Notify(func() {
        win, err := app.NewWindow(wain.WindowConfig{
            Title:       "Contact Form",
            Width:       480,
            Height:      320,
            Decorations: true,
        })
        if err != nil {
            log.Fatalf("create window: %v", err)
        }
        win.OnClose(func() { app.Quit() })
        _ = win
    })

    if err := app.Run(); err != nil {
        fmt.Println("No display available:", err)
    }
}
```

`app.Notify` schedules a callback to run once the event loop is ready.  All window
creation must happen inside this callback (or from subsequent `Notify` calls).

---

## 2. Layout: Column and Row

Use `NewColumn` for vertical stacking and `NewRow` for horizontal alignment.

```go
func buildForm() *wain.Column {
    form := wain.NewColumn()
    form.SetPadding(24)
    form.SetGap(12)
    return form
}
```

`SetPadding` adds space around the container edges.  `SetGap` controls the space
between children.

---

## 3. Widgets: Label and TextInput

```go
func addNameField(form *wain.Column) *wain.TextInput {
    label := wain.NewLabel("Your name:", wain.Size{Width: 100, Height: 20})
    form.Add(label)

    input := wain.NewTextInput("", wain.Size{Width: 100, Height: 28})
    input.SetPlaceholder("Jane Smith")
    form.Add(input)

    return input
}
```

`NewTextInput` accepts an initial text value and a size.  Both width and height values
are percentage-based when the parent has a fixed pixel size, or absolute pixels when
used at the root.

---

## 4. Events: Button OnClick

```go
func addSubmitButton(form *wain.Column, nameInput *wain.TextInput) {
    row := wain.NewRow()
    row.SetGap(8)

    btn := wain.NewButton("Submit", wain.Size{Width: 40, Height: 32})
    btn.OnClick(func() {
        name := nameInput.Text()
        if name == "" {
            name = "(empty)"
        }
        fmt.Printf("Submitted: %s\n", name)
    })
    row.Add(btn)

    form.Add(row)
}
```

`OnClick` replaces any previously registered handler; call it once per button.

---

## 5. Theming

Wain ships three built-in themes and supports custom overrides.

```go
app := wain.NewAppWithConfig(wain.AppConfig{
    Theme: wain.DefaultDark(),
})
```

Available themes:

| Function             | Description                                |
|----------------------|--------------------------------------------|
| `wain.DefaultLight()` | Light background, high contrast text      |
| `wain.DefaultDark()`  | Dark background, muted palette            |
| `wain.HighContrast()` | Maximum contrast for accessibility        |

You can also tweak individual fields:

```go
theme := wain.DefaultLight()
theme.Accent = wain.RGB(0x00, 0x77, 0xff)  // custom accent colour
```

---

## 6. Clipboard

Clipboard access is tied to a `Window` and dispatches to the active display server
(Wayland `data_device` or X11 `CLIPBOARD` selection) automatically.

```go
// Write to clipboard
if err := win.SetClipboard("Hello from wain!"); err != nil {
    log.Printf("clipboard write: %v", err)
}

// Read from clipboard
text, err := win.GetClipboard()
if err != nil {
    log.Printf("clipboard read: %v", err)
} else {
    fmt.Println("clipboard:", text)
}
```

---

## 7. Window Lifecycle

```go
win.OnClose(func() {
    fmt.Println("window closed")
    app.Quit()
})
```

`app.Quit()` signals the event loop to exit cleanly.  It is safe to call from any
goroutine.

---

## 8. Complete Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/opd-ai/wain"
)

func main() {
    app := wain.NewAppWithConfig(wain.AppConfig{Theme: wain.DefaultDark()})

    app.Notify(func() {
        win, err := app.NewWindow(wain.WindowConfig{
            Title:       "Contact Form",
            Width:       480,
            Height:      320,
            Decorations: true,
        })
        if err != nil {
            log.Fatalf("create window: %v", err)
        }
        win.OnClose(func() { app.Quit() })
        win.SetLayout(buildUI(win))
    })

    if err := app.Run(); err != nil {
        fmt.Println("No display server:", err)
    }
}

func buildUI(win *wain.Window) *wain.Column {
    form := wain.NewColumn()
    form.SetPadding(24)
    form.SetGap(12)

    form.Add(wain.NewLabel("Your name:", wain.Size{Width: 100, Height: 20}))

    nameInput := wain.NewTextInput("", wain.Size{Width: 100, Height: 28})
    nameInput.SetPlaceholder("Jane Smith")
    form.Add(nameInput)

    row := wain.NewRow()
    row.SetGap(8)

    submitBtn := wain.NewButton("Submit", wain.Size{Width: 40, Height: 32})
    submitBtn.OnClick(func() {
        name := nameInput.Text()
        fmt.Printf("Submitted: %s\n", name)
        _ = win.SetClipboard(name)
    })
    row.Add(submitBtn)

    clearBtn := wain.NewButton("Clear", wain.Size{Width: 40, Height: 32})
    clearBtn.OnClick(func() { nameInput.SetText("") })
    row.Add(clearBtn)

    form.Add(row)
    return form
}
```

---

## 9. Building and Running

```bash
# Development build (software renderer, standard gcc)
go build -o contact-form .
./contact-form

# Fully-static production build (requires musl-gcc and the Rust library)
make wain   # builds render-sys/librender_sys.a then the Go binary
ldd bin/wain  # should print "not a dynamic executable"
```

---

## Next Steps

- **Accessibility**: call `wain.EnableAccessibility("my-app")` to register with AT-SPI2
  and enable Orca/screen-reader support (requires `-tags=atspi` build).
- **Scrollable lists**: wrap a `Column` in a `ScrollView` for long content.
- **Images**: use `NewImageWidget` with a decoded `image.RGBA` resource.
- **DPI scaling**: set `AppConfig.Scale` or use `wain.SystemDPI()` to obtain the
  display's pixel ratio.
- **API reference**: run `go doc github.com/opd-ai/wain` for the full public API.
