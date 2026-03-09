# Getting Started with Wain

This guide provides step-by-step instructions for building your first GUI application with wain.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Your First Application](#your-first-application)
- [Building and Running](#building-and-running)
- [Next Steps](#next-steps)

## Prerequisites

### Required

- **Go 1.24** or later ([download](https://go.dev/dl/))
- **Linux** operating system with X11 or Wayland display server

### Optional (for rebuilding from source)

If you want to rebuild the Rust rendering backend from source:

- **Rust** (stable) with musl target
- **musl-gcc** compiler

To install optional prerequisites:

```bash
# Install Rust (if not already installed)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Add musl target for your architecture
rustup target add x86_64-unknown-linux-musl     # For x86_64
# OR
rustup target add aarch64-unknown-linux-musl    # For ARM64

# Install musl-gcc
sudo apt-get install musl-tools    # Ubuntu / Debian
# OR
sudo dnf install musl-gcc          # Fedora
# OR
sudo pacman -S musl                # Arch Linux
```

**Note:** For tagged releases, pre-built static libraries are provided, so the Rust toolchain is **not required** for normal development.

## Installation

### Standard Installation (Recommended)

For tagged releases, simply use `go get`:

```bash
go get github.com/opd-ai/wain
```

This downloads the module with pre-built static libraries for common platforms (x86_64, aarch64 Linux). No Rust toolchain required.

### Installing from Source

To install from the repository and rebuild the Rust backend:

```bash
# Clone the repository
git clone https://github.com/opd-ai/wain.git
cd wain

# Build the Rust rendering library
go generate ./...

# The library is now ready for use
go build ./cmd/example-app
```

## Your First Application

### Minimal "Hello World"

Create a new directory for your project:

```bash
mkdir my-wain-app
cd my-wain-app
go mod init my-wain-app
```

Create `main.go`:

```go
package main

import (
	"log"
	"github.com/opd-ai/wain"
)

func main() {
	// Create the application
	app := wain.NewApp()
	defer app.Close()

	// Create a window
	win, err := app.NewWindow(wain.WindowConfig{
		Title:  "Hello Wain",
		Width:  800,
		Height: 600,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Show the window
	win.Show()

	// Run the event loop (blocks until window is closed)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

### Adding Widgets

Let's create a more interesting application with buttons and text:

```go
package main

import (
	"log"
	"github.com/opd-ai/wain"
)

func main() {
	// Create application
	app := wain.NewApp()
	defer app.Close()

	// Create window
	win, err := app.NewWindow(wain.WindowConfig{
		Title:  "My First App",
		Width:  800,
		Height: 600,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create UI layout
	root := wain.NewColumn()
	root.SetPadding(20)
	root.SetGap(10)

	// Add a title label
	title := wain.NewLabel("Welcome to Wain!", wain.Size{Width: 100, Height: 10})
	root.Add(title)

	// Add a button
	button := wain.NewButton("Click Me", wain.Size{Width: 30, Height: 8})
	button.OnClick(func() {
		log.Println("Button clicked!")
		title.SetText("Button was clicked!")
		win.Redraw()
	})
	root.Add(button)

	// Add a text input
	input := wain.NewTextInput("Type here...", wain.Size{Width: 50, Height: 8})
	input.OnChange(func(text string) {
		log.Printf("Input changed: %s\n", text)
	})
	root.Add(input)

	// Set the root widget for the window
	win.SetRoot(root)
	win.Show()

	// Run the event loop
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

### Understanding Percentage-Based Sizing

Wain uses percentage-based sizing for automatic layout:

```go
// Size is specified as percentage of parent container
wain.Size{
	Width:  50,   // 50% of parent width
	Height: 100,  // 100% of parent height
}
```

This approach:
- Eliminates manual pixel calculations
- Automatically adapts to window resize
- Handles HiDPI scaling transparently
- Makes responsive layouts simple

### Layout Containers

Wain provides several container types for organizing widgets:

```go
// Row: arranges children horizontally
row := wain.NewRow()
row.Add(widget1)  // Left
row.Add(widget2)  // Center
row.Add(widget3)  // Right

// Column: arranges children vertically
column := wain.NewColumn()
column.Add(widget1)  // Top
column.Add(widget2)  // Middle
column.Add(widget3)  // Bottom

// ScrollView: scrollable container for overflow content
scroll := wain.NewScrollView(wain.Size{Width: 100, Height: 50})
scroll.SetContent(largePanel)

// Stack: layers widgets on top of each other
stack := wain.NewStack()
stack.Add(background)  // Bottom layer
stack.Add(overlay)     // Top layer

// Grid: arranges children in a fixed-column grid
grid := wain.NewGrid(3)  // 3-column grid
for i := 0; i < 9; i++ {
	grid.Add(wain.NewPanel(wain.Size{Width: 100, Height: 100}))
}
```

## Building and Running

### Standard Build

With pre-built libraries (default for tagged releases):

```bash
go build .
./my-wain-app
```

The resulting binary is **fully static** with zero runtime dependencies.

### Rebuilding from Source

If you want to rebuild the Rust backend:

```bash
# Method 1: Use go generate (in the wain module directory)
cd $GOPATH/pkg/mod/github.com/opd-ai/wain@version
go generate ./...
cd /path/to/your/project
go build .

# Method 2: Use the wain-build helper tool (recommended)
go install github.com/opd-ai/wain/cmd/wain-build@latest
wain-build  # Builds Rust library in current directory
go build .
```

### Verifying Static Linkage

To confirm your binary is fully static:

```bash
ldd ./my-wain-app
# Expected output: "not a dynamic executable"
```

## Next Steps

Now that you have a working wain application:

1. **Explore Widgets** — See [WIDGETS.md](WIDGETS.md) for detailed documentation of all widget types with examples and screenshots

2. **Learn Theming** — Customize your application's appearance:
   ```go
   app.SetTheme(wain.DefaultLight())  // Switch to light theme
   app.SetTheme(wain.HighContrast())  // High contrast for accessibility
   ```

3. **Study Examples** — The `cmd/` directory contains demonstration applications:
   - `cmd/example-app/` — Complete multi-panel application
   - `cmd/theme-demo/` — Theme switching demonstration
   - `cmd/callback-demo/` — Event callback examples
   - `cmd/window-demo/` — Window configuration examples

4. **Read API Documentation** — See [API.md](API.md) for comprehensive API reference

5. **Understand Architecture** — See [README.md](README.md#architecture) for system architecture

6. **Check Hardware Support** — See [HARDWARE.md](HARDWARE.md) for GPU and display server compatibility

## Common Issues

### "no display server available"

Wain requires either X11 or Wayland. Ensure you're running in a graphical environment:

```bash
# Check for X11
echo $DISPLAY

# Check for Wayland
echo $WAYLAND_DISPLAY
```

### Build errors with CGO

Wain uses CGO to link the Rust library. Ensure `CGO_ENABLED=1`:

```bash
export CGO_ENABLED=1
go build .
```

### "cannot find librender_sys.a"

The Rust static library is missing. Either:
1. Use a tagged release with pre-built libraries
2. Run `go generate ./...` in the wain module directory
3. Use the `wain-build` helper tool

### High memory usage

For applications with many widgets, consider:
- Using damage tracking (widgets only redraw when changed)
- Limiting the number of visible widgets
- Using ScrollView for large lists

## Help and Support

- **Documentation**: [API.md](API.md), [WIDGETS.md](WIDGETS.md), [HARDWARE.md](HARDWARE.md)
- **Examples**: See `cmd/` directory for working demos
- **Issues**: [GitHub Issues](https://github.com/opd-ai/wain/issues)
- **Contributing**: See [CONTRIBUTING.md](CONTRIBUTING.md) (if available)

## License

Wain is released under the MIT License. See [LICENSE](LICENSE) for details.
