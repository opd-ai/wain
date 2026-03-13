# API Documentation

This document provides an overview of wain's public-facing APIs, organized by layer and functionality.

> **Note:** All packages are currently marked `internal/` and are not guaranteed stable. A public API surface is planned for Phase 9 (stabilization). This documentation describes the current internal architecture.

---

## Table of Contents

- [Rendering Layer](#rendering-layer)
  - [GPU Abstraction](#gpu-abstraction)
  - [Buffer Management](#buffer-management)
  - [Shader Compilation](#shader-compilation)
  - [Backend Selection](#backend-selection)
- [Protocol Layer](#protocol-layer)
  - [Wayland Client](#wayland-client)
  - [X11 Client](#x11-client)
- [Rasterization Layer](#rasterization-layer)
  - [Software Rasterizer](#software-rasterizer)
  - [Display List](#display-list)
- [UI Layer](#ui-layer)
  - [Widget System](#widget-system)
  - [Layout System](#layout-system)
  - [Theming](#theming)
- [Integration Layer](#integration-layer)
  - [Buffer Synchronization](#buffer-synchronization)
  - [Clipboard](#clipboard)
  - [HiDPI Support](#hidpi-support)

---

## Rendering Layer

The rendering layer provides GPU abstraction, buffer management, and shader compilation.

### GPU Abstraction

**Package:** `internal/render`

#### GPU Detection

```go
func DetectGPU() (GpuGeneration, error)
```

Detects the GPU generation by querying `/dev/dri/renderD128`.

**Returns:**
- `GpuIntelGen9`, `GpuIntelGen11`, `GpuIntelGen12`, `GpuIntelXe` (Intel GPUs)
- `GpuAmdRdna1`, `GpuAmdRdna2`, `GpuAmdRdna3` (AMD GPUs)
- `error` if no compatible GPU found

**Example:**
```go
gen, err := render.DetectGPU()
if err != nil {
    log.Fatalf("GPU detection failed: %v", err)
}
fmt.Printf("Detected GPU: %d\n", gen)
```

#### GPU Context Management

```go
func CreateContext(alloc *Allocator) (uint32, error)
```

Creates a GPU context for command submission.

**Parameters:**
- `alloc`: Allocator instance (must be created first)

**Returns:**
- `contextID`: Opaque handle for the GPU context
- `error` if context creation fails

**Example:**
```go
alloc, _ := render.NewAllocator()
defer alloc.Destroy()

ctxID, err := render.CreateContext(alloc)
if err != nil {
    log.Fatalf("Context creation failed: %v", err)
}
```

```go
func DestroyContext(alloc *Allocator, contextID uint32) error
```

Destroys a GPU context. Should be called before destroying the allocator.

**Parameters:**
- `alloc`: Allocator instance
- `contextID`: Context ID from `CreateContext`

**Returns:**
- `error` if context destruction fails

---

### Buffer Management

**Package:** `internal/render`

#### Allocator

```go
type Allocator struct { /* ... */ }

func NewAllocator() (*Allocator, error)
```

Creates a new GPU buffer allocator. Opens `/dev/dri/renderD128` and initializes GPU state.

**Returns:**
- `*Allocator`: Allocator instance
- `error` if GPU initialization fails

**Example:**
```go
alloc, err := render.NewAllocator()
if err != nil {
    // Fall back to software rendering
    return nil, err
}
defer alloc.Destroy()
```

```go
func (a *Allocator) Allocate(width, height int, format PixelFormat, tiling TilingMode) (*BufferHandle, error)
```

Allocates a GPU buffer with the specified dimensions, format, and tiling.

**Parameters:**
- `width`, `height`: Buffer dimensions in pixels
- `format`: Pixel format (e.g., `PixelFormatARGB8888`, `PixelFormatXRGB8888`)
- `tiling`: Tiling mode (e.g., `TilingNone`, `TilingX`, `TilingY`)

**Returns:**
- `*BufferHandle`: Handle to the allocated buffer
- `error` if allocation fails

**Example:**
```go
buf, err := alloc.Allocate(1920, 1080, render.PixelFormatARGB8888, render.TilingY)
if err != nil {
    log.Fatalf("Buffer allocation failed: %v", err)
}
defer buf.Destroy()
```

```go
func (a *Allocator) Destroy() error
```

Destroys the allocator and frees GPU resources. All buffers must be destroyed before calling this.

#### BufferHandle

```go
type BufferHandle struct { /* ... */ }

func (b *BufferHandle) ExportDMABuf() (int, error)
```

Exports the buffer as a DMA-BUF file descriptor for sharing with X11/Wayland.

**Returns:**
- `fd`: File descriptor (must be closed by caller)
- `error` if export fails

**Example:**
```go
fd, err := buf.ExportDMABuf()
if err != nil {
    log.Fatalf("DMA-BUF export failed: %v", err)
}
defer syscall.Close(fd)

// Pass fd to Wayland/X11 for window presentation
```

```go
func (b *BufferHandle) GetInfo() BufferInfo
```

Returns buffer metadata (width, height, stride, format).

**Returns:**
- `BufferInfo`: Struct containing buffer dimensions and format

```go
func (b *BufferHandle) Mmap() ([]byte, error)
```

Memory-maps the buffer for CPU access.

**Returns:**
- `[]byte`: Slice pointing to mmap'd GPU memory
- `error` if mmap fails

**Example:**
```go
pixels, err := buf.Mmap()
if err != nil {
    log.Fatalf("mmap failed: %v", err)
}
defer buf.Munmap()

// Write pixels (e.g., upload texture data)
copy(pixels, imageData)
```

```go
func (b *BufferHandle) Munmap() error
```

Unmaps the buffer. Must be called after `Mmap()`.

```go
func (b *BufferHandle) Destroy() error
```

Destroys the buffer and frees GPU memory.

---

### Shader Compilation

**Package:** `internal/render`

#### ShaderModule

```go
type ShaderModule struct {
    Data []byte  // Compiled shader binary (EU or RDNA ISA)
}

func CompileShaderWGSL(source string, generation GpuGeneration) (*ShaderModule, error)
```

Compiles a WGSL shader to GPU-specific binary (Intel EU or AMD RDNA).

**Parameters:**
- `source`: WGSL shader source code
- `generation`: Target GPU generation (from `DetectGPU`)

**Returns:**
- `*ShaderModule`: Compiled shader binary
- `error` if compilation or validation fails

**Example:**
```go
const vertexShader = `
@vertex
fn vs_main(@location(0) pos: vec2<f32>) -> @builtin(position) vec4<f32> {
    return vec4<f32>(pos, 0.0, 1.0);
}
`

gen, _ := render.DetectGPU()
shader, err := render.CompileShaderWGSL(vertexShader, gen)
if err != nil {
    log.Fatalf("Shader compilation failed: %v", err)
}
```

```go
func CompileShaderGLSL(source string, stage ShaderStage, generation GpuGeneration) (*ShaderModule, error)
```

Compiles a GLSL shader to GPU-specific binary.

**Parameters:**
- `source`: GLSL shader source code
- `stage`: Shader stage (`ShaderStageVertex` or `ShaderStageFragment`)
- `generation`: Target GPU generation

**Returns:**
- `*ShaderModule`: Compiled shader binary
- `error` if compilation or validation fails

---

### Backend Selection

**Package:** `internal/render/backend`

#### Unified Renderer Interface

```go
type Renderer interface {
    Render(displayList *raster.DisplayList) error
    Present() (int, error)  // Returns DMA-BUF fd
    Destroy() error
}

func NewRenderer() (Renderer, error)
```

Creates a renderer with automatic GPU detection and fallback.

**Fallback order:**
1. Intel GPU (if Gen9+ detected)
2. AMD GPU (if RDNA1+ detected)
3. Software rasterizer (always available)

**Returns:**
- `Renderer`: Backend instance (GPU or software)
- `error` only if software fallback also fails (rare)

**Example:**
```go
renderer, err := backend.NewRenderer()
if err != nil {
    log.Fatalf("No rendering backend available: %v", err)
}
defer renderer.Destroy()

// Render a display list
displayList := raster.NewDisplayList()
// ... add draw commands ...

if err := renderer.Render(displayList); err != nil {
    log.Fatalf("Render failed: %v", err)
}

// Present to window
fd, _ := renderer.Present()
// ... attach fd to Wayland/X11 buffer ...
```

---

## Protocol Layer

The protocol layer implements Wayland and X11 client protocols.

### Wayland Client

**Package:** `internal/wayland/client`

#### Connection Setup

```go
type Client struct { /* ... */ }

func NewClient() (*Client, error)
```

Connects to the Wayland compositor via `$WAYLAND_DISPLAY` socket.

**Returns:**
- `*Client`: Wayland client instance
- `error` if connection fails

**Example:**
```go
client, err := client.NewClient()
if err != nil {
    log.Fatalf("Wayland connection failed: %v", err)
}
defer client.Close()
```

```go
func (c *Client) Roundtrip() error
```

Synchronizes with the compositor (waits for all pending events).

#### Registry

```go
func (c *Client) BindInterface(name uint32, interfaceName string, version uint32) (uint32, error)
```

Binds a global Wayland object.

**Parameters:**
- `name`: Global name from `wl_registry.global` event
- `interfaceName`: Interface name (e.g., "wl_compositor", "xdg_wm_base")
- `version`: Protocol version

**Returns:**
- `objectID`: New object ID for the bound interface
- `error` if binding fails

**Example:**
```go
// After receiving wl_registry.global event with name=5, interface="wl_compositor", version=4
compositorID, _ := client.BindInterface(5, "wl_compositor", 4)
```

#### Surface Creation

**Package:** `internal/wayland/compositor`

```go
func CreateSurface(client *client.Client, compositorID uint32) (uint32, error)
```

Creates a `wl_surface`.

**Parameters:**
- `client`: Wayland client instance
- `compositorID`: Compositor object ID (from registry binding)

**Returns:**
- `surfaceID`: New surface object ID
- `error` if creation fails

#### Window Management

**Package:** `internal/wayland/xdg`

```go
func CreateToplevel(client *client.Client, wmBaseID, surfaceID uint32) (uint32, uint32, error)
```

Creates an XDG toplevel window.

**Parameters:**
- `client`: Wayland client instance
- `wmBaseID`: `xdg_wm_base` object ID
- `surfaceID`: `wl_surface` object ID

**Returns:**
- `xdgSurfaceID`: XDG surface object ID
- `xdgToplevelID`: XDG toplevel object ID
- `error` if creation fails

**Example:**
```go
xdgSurfaceID, xdgToplevelID, err := xdg.CreateToplevel(client, wmBaseID, surfaceID)
if err != nil {
    log.Fatalf("Window creation failed: %v", err)
}

// Set window title
xdg.SetTitle(client, xdgToplevelID, "My Application")

// Map the surface
client.Roundtrip()
```

#### Buffer Attachment

**Package:** `internal/wayland/dmabuf`

```go
func CreateBuffer(client *client.Client, dmabufID uint32, fd int, width, height, stride int, format uint32) (uint32, error)
```

Creates a `wl_buffer` from a DMA-BUF file descriptor.

**Parameters:**
- `client`: Wayland client instance
- `dmabufID`: `zwp_linux_dmabuf_v1` object ID
- `fd`: DMA-BUF file descriptor (from `BufferHandle.ExportDMABuf()`)
- `width`, `height`, `stride`: Buffer dimensions
- `format`: DRM fourcc format (e.g., `DRM_FORMAT_ARGB8888`)

**Returns:**
- `bufferID`: `wl_buffer` object ID
- `error` if creation fails

**Example:**
```go
buf, _ := alloc.Allocate(1920, 1080, render.PixelFormatARGB8888, render.TilingY)
fd, _ := buf.ExportDMABuf()
defer syscall.Close(fd)

info := buf.GetInfo()
bufferID, _ := dmabuf.CreateBuffer(client, dmabufID, fd, info.Width, info.Height, info.Stride, DRM_FORMAT_ARGB8888)

// Attach to surface
compositor.Attach(client, surfaceID, bufferID, 0, 0)
compositor.Commit(client, surfaceID)
```

---

### X11 Client

**Package:** `internal/x11/client`

#### Connection Setup

```go
type Client struct { /* ... */ }

func NewClient(display string) (*Client, error)
```

Connects to the X11 server via `$DISPLAY` or the provided display string.

**Parameters:**
- `display`: Display string (e.g., ":0"), or empty for `$DISPLAY`

**Returns:**
- `*Client`: X11 client instance
- `error` if connection fails

**Example:**
```go
client, err := client.NewClient("")  // Use $DISPLAY
if err != nil {
    log.Fatalf("X11 connection failed: %v", err)
}
defer client.Close()
```

#### Window Creation

```go
func (c *Client) CreateWindow(x, y, width, height int) (uint32, error)
```

Creates a window on the root screen.

**Parameters:**
- `x`, `y`: Window position
- `width`, `height`: Window dimensions

**Returns:**
- `windowID`: X11 window ID
- `error` if creation fails

**Example:**
```go
winID, _ := client.CreateWindow(0, 0, 1920, 1080)
client.MapWindow(winID)  // Make window visible
```

#### Buffer Sharing

**Package:** `internal/x11/dri3`

```go
func PixmapFromBuffers(client *client.Client, drawableID uint32, width, height, stride int, depth, bpp uint8, fd int) (uint32, error)
```

Creates a pixmap from a DMA-BUF file descriptor.

**Parameters:**
- `client`: X11 client instance
- `drawableID`: Drawable ID (window or pixmap)
- `width`, `height`, `stride`: Buffer dimensions
- `depth`, `bpp`: Color depth and bits per pixel
- `fd`: DMA-BUF file descriptor

**Returns:**
- `pixmapID`: X11 pixmap ID
- `error` if creation fails

**Example:**
```go
buf, _ := alloc.Allocate(1920, 1080, render.PixelFormatARGB8888, render.TilingY)
fd, _ := buf.ExportDMABuf()
defer syscall.Close(fd)

info := buf.GetInfo()
pixmapID, _ := dri3.PixmapFromBuffers(client, winID, info.Width, info.Height, info.Stride, 24, 32, fd)

// Present pixmap to window
present.Pixmap(client, winID, pixmapID, 0, 0, 0)
```

---

## Rasterization Layer

The rasterization layer provides CPU-based software rendering and display list abstraction.

### Software Rasterizer

**Package:** `internal/raster/core`

#### Canvas

```go
type Canvas struct { /* ... */ }

func NewCanvas(width, height int) *Canvas
```

Creates a new ARGB8888 framebuffer.

**Parameters:**
- `width`, `height`: Canvas dimensions

**Returns:**
- `*Canvas`: Canvas instance

```go
func (c *Canvas) Pixels() []byte
```

Returns the underlying pixel buffer (ARGB8888, 4 bytes per pixel).

#### Drawing Primitives

**Package:** `internal/raster/core`

```go
func FillRect(canvas *Canvas, x, y, width, height int, color Color)
```

Draws a filled rectangle with solid color.

```go
func FillRoundedRect(canvas *Canvas, x, y, width, height int, radius float32, color Color)
```

Draws a filled rectangle with rounded corners.

```go
func DrawLine(canvas *Canvas, x0, y0, x1, y1 int, thickness float32, color Color)
```

Draws an anti-aliased line.

**Package:** `internal/raster/text`

```go
func DrawText(canvas *Canvas, text string, x, y int, fontSize float32, color Color) error
```

Draws text using embedded SDF font atlas.

**Parameters:**
- `canvas`: Target canvas
- `text`: UTF-8 text string
- `x`, `y`: Baseline position
- `fontSize`: Font size in pixels
- `color`: Text color

**Package:** `internal/raster/effects`

```go
func LinearGradient(canvas *Canvas, x, y, width, height int, startColor, endColor Color, angle float32)
```

Draws a linear gradient.

```go
func BoxShadow(canvas *Canvas, x, y, width, height int, blur, spread float32, color Color)
```

Draws a box shadow (Gaussian blur on rect mask).

---

### Display List

**Package:** `internal/raster/displaylist`

#### DisplayList

```go
type DisplayList struct { /* ... */ }

func NewDisplayList() *DisplayList
```

Creates a new display list (vector of draw commands).

```go
func (dl *DisplayList) AddRect(x, y, width, height int, color Color)
```

Adds a solid rectangle to the display list.

```go
func (dl *DisplayList) AddRoundedRect(x, y, width, height int, radius float32, color Color)
```

Adds a rounded rectangle.

```go
func (dl *DisplayList) AddText(text string, x, y int, fontSize float32, color Color)
```

Adds a text draw command.

```go
func (dl *DisplayList) AddGradient(x, y, width, height int, startColor, endColor Color, angle float32)
```

Adds a linear gradient.

#### Consumer

**Package:** `internal/raster/consumer`

```go
type SoftwareConsumer struct { /* ... */ }

func NewSoftwareConsumer(width, height int) *SoftwareConsumer
```

Creates a display list consumer that renders to a CPU framebuffer.

```go
func (sc *SoftwareConsumer) Consume(displayList *DisplayList) error
```

Renders a display list to the internal canvas.

```go
func (sc *SoftwareConsumer) Pixels() []byte
```

Returns the rendered pixel buffer.

---

## UI Layer

The UI layer provides widgets, layout, and theming.

### Widget System

**Package:** `internal/ui/widget`

#### Button

```go
type Button struct {
    Text     string
    OnClick  func()
    Bounds   Rect
    Hovered  bool
    Pressed  bool
}

func NewButton(text string, onClick func()) *Button
```

Creates a button widget.

**Example:**
```go
btn := widget.NewButton("Click Me", func() {
    fmt.Println("Button clicked!")
})

btn.SetBounds(Rect{X: 10, Y: 10, Width: 100, Height: 40})
btn.Draw(canvas, theme)
```

```go
func (b *Button) HandleEvent(event Event) bool
```

Processes input events (mouse clicks, hover).

**Returns:**
- `true` if event was consumed, `false` otherwise

#### TextInput

```go
type TextInput struct {
    Text        string
    Placeholder string
    Bounds      Rect
    Focused     bool
}

func NewTextInput(placeholder string) *TextInput
```

Creates a text input widget.

**Example:**
```go
input := widget.NewTextInput("Enter text...")
input.SetBounds(Rect{X: 10, Y: 60, Width: 200, Height: 30})
input.Draw(canvas, theme)
```

```go
func (ti *TextInput) HandleEvent(event Event) bool
```

Processes keyboard and mouse events.

#### ScrollContainer

```go
type ScrollContainer struct {
    Child         Widget
    Bounds        Rect
    ScrollOffset  int
    ContentHeight int
}

func NewScrollContainer(child Widget) *ScrollContainer
```

Creates a scrollable container for another widget.

**Example:**
```go
// Wrap a tall widget in a scroll container
tallWidget := /* ... */
scroll := widget.NewScrollContainer(tallWidget)
scroll.SetBounds(Rect{X: 10, Y: 100, Width: 300, Height: 200})
scroll.Draw(canvas, theme)
```

---

### Layout System

**Package:** `internal/ui/layout`

#### Row / Column

```go
type Row struct {
    Children []Widget
    Gap      int
    Padding  Padding
}

func NewRow(gap int) *Row
```

Creates a horizontal layout container (flexbox-like).

**Example:**
```go
row := layout.NewRow(10)  // 10px gap between children
row.AddChild(btn1)
row.AddChild(btn2)
row.AddChild(btn3)
row.Layout(Rect{X: 0, Y: 0, Width: 400, Height: 50})
```

```go
type Column struct {
    Children []Widget
    Gap      int
    Padding  Padding
}

func NewColumn(gap int) *Column
```

Creates a vertical layout container.

#### Auto-Layout

**Package:** `internal/ui/pctwidget`

```go
func AutoLayout(widget Widget, parentWidth, parentHeight int) Rect
```

Computes widget bounds based on percentage-based constraints.

**Example:**
```go
// Widget fills 80% of parent width, 50% of parent height
bounds := pctwidget.AutoLayout(widget, 1920, 1080)
widget.SetBounds(bounds)
```

---

### Theming

**Package:** `internal/ui/widget`

#### Theme

```go
type Theme struct {
    BackgroundColor Color
    ForegroundColor Color
    AccentColor     Color
    TextColor       Color
    FontSize        float32
    Scale           float32  // HiDPI scale factor
}

func DefaultTheme() Theme
```

Returns the default theme.

**Example:**
```go
theme := widget.DefaultTheme()
theme.Scale = 2.0  // 2x scale for HiDPI displays
theme.AccentColor = Color{R: 255, G: 0, B: 0, A: 255}  // Red accent

btn.Draw(canvas, theme)
```

---

## Integration Layer

### Buffer Synchronization

**Package:** `internal/buffer`

#### Ring Buffer

```go
type Ring struct { /* ... */ }

func NewRing(count int) *Ring
```

Creates a buffer ring for double/triple buffering.

**Parameters:**
- `count`: Number of buffers (2 for double buffering, 3 for triple)

```go
func (r *Ring) Acquire(ctx context.Context) (int, error)
```

Acquires a free buffer slot (blocks until one is available).

**Returns:**
- `index`: Buffer index (0 to count-1)
- `error` if context canceled

```go
func (r *Ring) Release(index int)
```

Releases a buffer slot (marks as free).

**Example:**
```go
ring := buffer.NewRing(2)  // Double buffering

for {
    idx, _ := ring.Acquire(context.Background())
    
    // Render to buffers[idx]
    renderer.Render(displayList)
    fd, _ := renderer.Present()
    
    // Attach to window
    compositor.Attach(client, surfaceID, bufferIDs[idx], 0, 0)
    compositor.Commit(client, surfaceID)
    
    // Buffer will be released when compositor sends wl_buffer.release
}
```

#### Synchronizer

**Package:** `internal/buffer`

```go
type Synchronizer struct { /* ... */ }

func NewSynchronizer(ring *Ring) *Synchronizer
```

Coordinates buffer ring with compositor release events.

**Package:** `internal/integration`

```go
type WaylandBufferHandler struct { /* ... */ }

func NewWaylandBufferHandler(sync *buffer.Synchronizer) *WaylandBufferHandler
```

Handles `wl_buffer.release` events and releases buffer slots.

**Example:**
```go
sync := buffer.NewSynchronizer(ring)
handler := integration.NewWaylandBufferHandler(sync)

// Register handler for wl_buffer events
client.SetBufferListener(bufferID, handler)
```

---

### Clipboard

**Package:** `internal/wayland/datadevice` (Wayland)  
**Package:** `internal/x11/selection` (X11)

#### Wayland Clipboard

```go
func OfferClipboard(client *client.Client, deviceID uint32, mimeType string, data []byte) error
```

Copies data to the clipboard.

**Parameters:**
- `client`: Wayland client
- `deviceID`: `wl_data_device` object ID
- `mimeType`: MIME type (e.g., "text/plain")
- `data`: Clipboard data

```go
func ReceiveClipboard(client *client.Client, deviceID uint32, mimeType string) ([]byte, error)
```

Pastes data from the clipboard.

#### X11 Clipboard

```go
func SetSelection(client *client.Client, windowID uint32, atom uint32, data []byte) error
```

Copies data to the clipboard.

**Parameters:**
- `client`: X11 client
- `windowID`: Owner window
- `atom`: Selection atom (e.g., `CLIPBOARD` or `PRIMARY`)
- `data`: Clipboard data

```go
func GetSelection(client *client.Client, windowID uint32, atom uint32) ([]byte, error)
```

Pastes data from the clipboard.

---

### HiDPI Support

**Package:** `internal/ui/scale`

#### Scale Manager

```go
type ScaleManager struct { /* ... */ }

func NewScaleManager() *ScaleManager
```

Creates a scale factor manager.

```go
func (sm *ScaleManager) GetScale() float32
```

Returns the current scale factor (defaults to 1.0).

```go
func (sm *ScaleManager) SetScale(scale float32)
```

Sets the scale factor (e.g., 2.0 for 2× HiDPI displays).

**Package:** `internal/wayland/output` (Wayland)  
**Package:** `internal/x11/dpi` (X11)

#### Wayland HiDPI Detection

```go
func GetOutputScale(client *client.Client, outputID uint32) (int, error)
```

Retrieves the scale factor from `wl_output.scale` event.

**Returns:**
- `scale`: Integer scale factor (1, 2, 3, etc.)
- `error` if scale not yet received

#### X11 HiDPI Detection

```go
func GetDPI(client *client.Client) (float32, error)
```

Calculates DPI from screen physical dimensions.

**Returns:**
- `dpi`: Dots per inch (e.g., 96.0, 192.0)
- `error` if DPI calculation fails

**Example:**
```go
// Wayland
scale, _ := output.GetOutputScale(client, outputID)
theme.Scale = float32(scale)

// X11
dpi, _ := dpi.GetDPI(client)
theme.Scale = dpi / 96.0  // Normalize to 1.0 at 96 DPI
```

---

## Error Handling Conventions

### Error Types

Most functions return `error` as the last return value. Errors are typically:

1. **Transient errors** (retryable):
   - Network timeout
   - GPU busy
   - Out of memory (buffer allocation)

2. **Permanent errors** (not retryable):
   - Invalid parameters
   - GPU context lost
   - Unsupported operation

### Error Recovery

**GPU errors:**
```go
err := render.SubmitBatch(alloc, ctxID, batch)
if err != nil {
    if render.IsTimeout(err) {
        // GPU hang, destroy context and fall back to software
        render.DestroyContext(alloc, ctxID)
        return useSoftwareRenderer()
    }
    return err
}
```

**Protocol errors:**
```go
err := client.Roundtrip()
if err != nil {
    // Compositor disconnected, cannot recover
    log.Fatalf("Compositor connection lost: %v", err)
}
```

---

## Thread Safety

### Thread-Safe Packages

- `internal/buffer.Ring`: Safe for concurrent `Acquire`/`Release`
- `internal/buffer.Synchronizer`: Safe for concurrent event handlers
- `internal/render.MemoryStats`: Atomic counters, safe for concurrent updates

### Thread-Unsafe Packages

Most other packages are **not thread-safe**. In particular:

- **Wayland/X11 clients:** Must be used from a single goroutine (protocol serialization)
- **Canvas/Rasterizer:** No internal locking
- **Widget tree:** Not thread-safe

**Recommended pattern:**
- Use a single "UI thread" goroutine for all windowing and rendering
- Use channels to communicate with background workers

**Example:**
```go
func main() {
    uiChan := make(chan func())
    
    // Background worker
    go func() {
        result := expensiveComputation()
        uiChan <- func() {
            updateWidget(result)
        }
    }()
    
    // UI thread
    for {
        select {
        case uiFunc := <-uiChan:
            uiFunc()  // Execute on UI thread
        case event := <-eventChan:
            handleEvent(event)
        }
        
        render()
    }
}
```

---

## Known Limitations

### Widget System

- **Cross-axis alignment (Panel.SetAlign)**: Currently only `AlignStart` (top for Row, left for Column) is supported. `AlignCenter`, `AlignEnd`, and `AlignStretch` are deferred to a future phase.
- **TextInput placeholder**: The `TextInput.SetPlaceholder` method accepts a placeholder string but does not display it. Placeholder rendering support is planned for a future phase.
- **ScrollView child management**: The `ScrollView.Add` method is not yet implemented. ScrollView currently does not support adding PublicWidget children.
- **Wayland event dispatch**: Wayland event reading and dispatch is not fully implemented. The current implementation only flushes outbound requests to prevent deadlock.

---

## Versioning and Stability

### Current Status

**All APIs are considered unstable** (subject to change without notice) until Phase 9 stabilization.

### Planned Stable API Surface

The following will be promoted to a public `github.com/opd-ai/wain` package in Phase 9:

- Renderer interface (`backend.Renderer`)
- Display list abstraction (`displaylist.DisplayList`)
- Widget interfaces (`widget.Widget`, `Button`, `TextInput`, etc.)
- Theme struct (`widget.Theme`)

Internal protocol, rasterization, and GPU APIs will remain in `internal/` packages.

---

## Code Examples

See `cmd/` directory for complete examples:

| Binary | Description |
|--------|-------------|
| `demo` | Basic Wayland window with software rendering |
| `wayland-demo` | Wayland protocol demonstration |
| `x11-demo` | X11 protocol demonstration |
| `widget-demo` | Widget and layout demonstration |
| `x11-dmabuf-demo` | X11 DRI3 buffer sharing |
| `dmabuf-demo` | Wayland dmabuf buffer sharing |
| `gpu-triangle-demo` | GPU command submission (triangle) |
| `gpu-shader-demo` | WGSL shader compilation → EU/RDNA → batch submission |
| `double-buffer-demo` | Double buffering with ring buffer |
| `auto-render-demo` | Auto-detection with GPU/software fallback |
| `clipboard-demo` | Clipboard operations (Wayland + X11) |
| `decorations-demo` | Client-side window decorations |

---

## GPU Rendering Pipeline

Wain includes a shader-driven GPU rendering pipeline that compiles WGSL shaders
to native Intel EU or AMD RDNA machine code at runtime.

### Data Flow

```
WGSL source (text)
    │
    ▼
render_compile_shader()          ← Rust / CGO boundary
    │  (naga IR → EU/RDNA binary)
    ▼
EU kernel binary  (Vec<u8>)      ← Intel Gen9/11/12/Xe
RDNA kernel binary (Vec<u8>)     ← AMD RDNA1/2/3
    │
    ▼
bind_eu_shader_to_batch()        ← render-sys/src/submit.rs
    │  allocates GEM buffer, emits 3DSTATE_PS pipeline state,
    │  registers relocation for shader VA
    ▼
BatchBuilder → SubmittableBatch
    │
    ▼
render_submit_shader_batch()     ← lib.rs #[no_mangle] FFI
    │  i915_submit_batch / xe_submit_batch_simple / amdgpu_submit_with_va
    ▼
GPU executes shader kernel
```

### Go API

The following functions in `internal/render` expose the shader pipeline to Go:

```go
// Compile a WGSL shader to native machine code for the detected GPU.
// gpuGen: 9=Gen9, 11=Gen11, 12=Gen12, 13=Xe, -1=RDNA1, -2=RDNA2, -3=RDNA3.
binary, err := render.CompileShader(wgslSource, gpuGen, isFragment)

// Compile and submit a shader batch in one call.
// Detects GPU automatically from the DRM device.
err = render.SubmitShaderBatch(drmPath, wgslSource, isFragment, contextID)
```

### GPUBackend Integration

`internal/render/backend.GPUBackend` compiles `solid_fill.wgsl` at startup via
`go:embed` and calls `render.SubmitShaderBatch` on every frame.  If the GPU is
unavailable or compilation fails, the backend silently falls back to the
fixed-function batch construction path.

```go
cfg := backend.DefaultConfig()
b, err := backend.New(cfg)
// solid_fill.wgsl is compiled to EU/RDNA binary during New()
```

### Supported GPU Generations

| GPU Family | Generation IDs | Notes |
|-----------|----------------|-------|
| Intel Gen9 | 9 | Skylake, Kaby Lake, Coffee Lake |
| Intel Gen11 | 11 | Ice Lake |
| Intel Gen12 | 12 | Tiger Lake, Rocket Lake, Alder Lake |
| Intel Xe | 13 | Meteor Lake+ |
| AMD RDNA1 | -1 | RX 5000 series |
| AMD RDNA2 | -2 | RX 6000 series, Steam Deck |
| AMD RDNA3 | -3 | RX 7000 series |

---

**Last Updated:** 2026-03-13  
**wain Version:** Phase 4.3 (GPU shader pipeline integrated)
