# Widget Reference

This document provides comprehensive documentation for all widget types in wain, including usage examples and visual descriptions.

## Table of Contents

- [Widget Basics](#widget-basics)
- [Layout Containers](#layout-containers)
  - [Panel](#panel)
  - [Row](#row)
  - [Column](#column)
  - [Stack](#stack)
  - [Grid](#grid)
  - [ScrollView](#scrollview)
- [Interactive Widgets](#interactive-widgets)
  - [Button](#button)
  - [TextInput](#textinput)
- [Display Widgets](#display-widgets)
  - [Label](#label)
  - [ImageWidget](#imagewidget)
  - [Spacer](#spacer)
- [Sizing and Layout](#sizing-and-layout)
- [Styling](#styling)
- [Event Handling](#event-handling)

## Widget Basics

### Widget Interface

All widgets implement the `PublicWidget` interface:

```go
type PublicWidget interface {
	Bounds() (width, height int)
	HandleEvent(Event) bool
	Draw(Canvas)
}
```

### Container Interface

Containers can hold child widgets:

```go
type Container interface {
	PublicWidget
	Add(child PublicWidget)
	Children() []PublicWidget
}
```

### Percentage-Based Sizing

All widgets use percentage-based sizing relative to their parent container:

```go
wain.Size{
	Width:  50,   // 50% of parent width
	Height: 100,  // 100% of parent height
}
```

This approach:
- Automatically adapts to window resize
- Handles HiDPI scaling transparently
- Eliminates manual pixel calculations
- Makes responsive layouts simple

## Layout Containers

### Panel

A styled rectangular container that can hold child widgets.

**Type:** `*wain.Panel`

**Constructor:**
```go
func NewPanel(size Size) *Panel
```

**Methods:**
```go
SetFlowDirection(dir FlowDirection)  // FlowRow or FlowColumn
SetPadding(padding int)               // Internal padding in pixels
SetGap(gap int)                       // Space between children in pixels
SetAlign(align Align)                 // AlignStart, AlignCenter, AlignEnd, AlignStretch
SetPosition(x, y int)                 // Absolute positioning (overrides layout)
ClearPosition()                       // Return to automatic layout
SetVisible(visible bool)              // Show/hide the panel
SetStyle(override StyleOverride)      // Apply custom styling
```

**Example:**
```go
// Create a panel taking 50% width, 100% height
panel := wain.NewPanel(wain.Size{Width: 50, Height: 100})
panel.SetPadding(10)
panel.SetGap(5)
panel.SetFlowDirection(wain.FlowColumn)

// Add child widgets
panel.Add(wain.NewLabel("Title", wain.Size{Width: 100, Height: 10}))
panel.Add(wain.NewButton("Click Me", wain.Size{Width: 50, Height: 8}))
```

**Visual Description:**
- Rectangular container with optional background color
- Supports rounded corners via theme
- Can have border and shadow
- Children are laid out according to flow direction

### Row

A horizontal layout container that arranges children left-to-right.

**Type:** `*wain.Row` (extends `*wain.Panel`)

**Constructor:**
```go
func NewRow() *Row
```

**Behavior:**
- Automatically sets `FlowDirection` to `FlowRow`
- Distributes horizontal space based on children's Width percentages
- All children share the container's full height

**Example:**
```go
// Create a row for header layout
header := wain.NewRow()
header.SetPadding(10)
header.SetGap(10)

// Left: title (70% width)
header.Add(wain.NewLabel("My App", wain.Size{Width: 70, Height: 100}))

// Right: button (30% width)
header.Add(wain.NewButton("Settings", wain.Size{Width: 30, Height: 100}))
```

**Use Cases:**
- Navigation bars
- Toolbars
- Button groups
- Multi-column layouts

### Column

A vertical layout container that arranges children top-to-bottom.

**Type:** `*wain.Column` (extends `*wain.Panel`)

**Constructor:**
```go
func NewColumn() *Column
```

**Behavior:**
- Automatically sets `FlowDirection` to `FlowColumn`
- Distributes vertical space based on children's Height percentages
- All children share the container's full width

**Example:**
```go
// Create a column for sidebar
sidebar := wain.NewColumn()
sidebar.SetPadding(10)
sidebar.SetGap(5)

// Stack buttons vertically
sidebar.Add(wain.NewButton("Home", wain.Size{Width: 100, Height: 15}))
sidebar.Add(wain.NewButton("Settings", wain.Size{Width: 100, Height: 15}))
sidebar.Add(wain.NewButton("About", wain.Size{Width: 100, Height: 15}))
```

**Use Cases:**
- Sidebars
- Vertical menus
- Form layouts
- Stacked content

### Stack

A layering container that stacks children on top of each other (Z-axis).

**Type:** `*wain.Stack`

**Constructor:**
```go
func NewStack() *Stack
```

**Behavior:**
- Children are layered in add order (first = bottom, last = top)
- All children occupy the full container bounds
- Later children render on top of earlier ones

**Example:**
```go
// Create overlay UI
stack := wain.NewStack()

// Background panel (bottom layer)
background := wain.NewPanel(wain.Size{Width: 100, Height: 100})
background.SetStyle(wain.StyleOverride{Background: &wain.Black})
stack.Add(background)

// Modal dialog (top layer)
modal := wain.NewPanel(wain.Size{Width: 60, Height: 40})
modal.SetPosition(20, 30)  // Center it
stack.Add(modal)
```

**Use Cases:**
- Modal dialogs
- Tooltips
- Overlays
- Background images with UI on top

### Grid

A fixed-column grid container that arranges children in rows and columns.

**Type:** `*wain.Grid`

**Constructor:**
```go
func NewGrid(columns int) *Grid
```

**Methods:**
```go
SetColumns(columns int)  // Change grid column count
Columns() int            // Get current column count
```

**Behavior:**
- Arranges children in a grid with fixed number of columns
- Each cell is evenly divided
- Children's percentage sizes are relative to their cell
- Rows are added automatically as needed

**Example:**
```go
// Create a 3-column grid for icon buttons
grid := wain.NewGrid(3)
grid.SetPadding(10)
grid.SetGap(5)

// Add 9 buttons (creates 3 rows automatically)
for i := 1; i <= 9; i++ {
	btn := wain.NewButton(fmt.Sprintf("Btn %d", i), 
		wain.Size{Width: 100, Height: 100})
	grid.Add(btn)
}
```

**Use Cases:**
- Icon grids
- Image galleries
- Calculator layouts
- Uniform button arrays

### ScrollView

A scrollable container for overflow content.

**Type:** `*wain.ScrollView`

**Constructor:**
```go
func NewScrollView(size Size) *ScrollView
```

**Methods:**
```go
SetContent(content PublicWidget)    // Set the scrollable content
OnScroll(handler func(offset int))  // Register scroll callback
SetStyle(override StyleOverride)    // Apply custom styling
```

**Example:**
```go
// Create scrollable content area
scroll := wain.NewScrollView(wain.Size{Width: 100, Height: 80})

// Create large content (taller than viewport)
content := wain.NewColumn()
for i := 0; i < 20; i++ {
	content.Add(wain.NewLabel(
		fmt.Sprintf("Item %d", i),
		wain.Size{Width: 100, Height: 5},
	))
}

scroll.SetContent(content)

// Handle scroll events
scroll.OnScroll(func(offset int) {
	log.Printf("Scrolled to offset: %d", offset)
})
```

**Visual Description:**
- Viewport shows a portion of the content
- Scrollbar appears when content exceeds viewport height
- Mouse wheel scrolls content
- Touch/drag scrolling supported

**Use Cases:**
- Long lists
- Text documents
- Large forms
- Content feeds

## Interactive Widgets

### Button

A clickable button with text and visual feedback.

**Type:** `*wain.Button`

**Constructor:**
```go
func NewButton(text string, size Size) *Button
```

**Methods:**
```go
OnClick(handler func())        // Register click callback
SetText(text string)           // Change button text
Text() string                  // Get current text
SetEnabled(enabled bool)       // Enable/disable button
SetStyle(override StyleOverride)  // Apply custom styling
```

**Example:**
```go
// Create a submit button
btn := wain.NewButton("Submit", wain.Size{Width: 30, Height: 8})
btn.OnClick(func() {
	fmt.Println("Form submitted!")
	// Perform submit action
})

// Disable button during processing
btn.SetEnabled(false)
// ... do work ...
btn.SetEnabled(true)
```

**Visual States:**
- **Normal:** Default appearance
- **Hover:** Highlighted when mouse is over button
- **Press:** Pressed appearance when mouse button is down
- **Disabled:** Grayed out, no interaction

**Use Cases:**
- Form submission
- Navigation
- Actions and commands
- Confirmations

### TextInput

A single-line editable text field with cursor.

**Type:** `*wain.TextInput`

**Constructor:**
```go
func NewTextInput(placeholder string, size Size) *TextInput
```

**Methods:**
```go
OnChange(handler func(text string))  // Register text change callback
SetText(text string)                 // Set the text content
Text() string                        // Get current text
SetStyle(override StyleOverride)     // Apply custom styling
```

**Example:**
```go
// Create a search input
input := wain.NewTextInput("Search...", wain.Size{Width: 60, Height: 8})
input.OnChange(func(text string) {
	fmt.Printf("Search query: %s\n", text)
	// Perform search
})

// Pre-populate text
input.SetText("initial value")
```

**Features:**
- Cursor positioning and blinking
- Text selection (future)
- Keyboard input
- Copy/paste support (via system clipboard)
- Placeholder text when empty

**Visual States:**
- **Unfocused:** Border, placeholder text shown
- **Focused:** Highlighted border, cursor visible
- **Text entry:** Characters appear at cursor position

**Use Cases:**
- Search fields
- Form inputs
- Name/email entry
- Configuration values

## Display Widgets

### Label

Static text display widget.

**Type:** `*wain.Label`

**Constructor:**
```go
func NewLabel(text string, size Size) *Label
```

**Methods:**
```go
SetText(text string)              // Change the displayed text
Text() string                     // Get current text
SetStyle(override StyleOverride)  // Apply custom styling
```

**Example:**
```go
// Create a title label
title := wain.NewLabel("Welcome to My App", wain.Size{Width: 100, Height: 10})

// Create a status label that updates
status := wain.NewLabel("Ready", wain.Size{Width: 100, Height: 5})

// Update text dynamically
status.SetText("Processing...")
// ... do work ...
status.SetText("Complete!")
```

**Visual Description:**
- Text rendered at specified font size
- Color from theme or style override
- Left-aligned by default
- Supports theme font scaling

**Use Cases:**
- Titles and headings
- Status messages
- Descriptions
- Static information

### ImageWidget

Displays an image resource.

**Type:** `*wain.ImageWidget`

**Constructor:**
```go
func NewImageWidget(img *Image, size Size) *ImageWidget
```

**Methods:**
```go
SetImage(img *Image)             // Change the displayed image
SetStyle(override StyleOverride) // Apply custom styling
```

**Example:**
```go
// Load an image
img, err := app.LoadImage("icon.png")
if err != nil {
	log.Fatal(err)
}

// Create image widget (image scaled to fit bounds)
imgWidget := wain.NewImageWidget(img, wain.Size{Width: 20, Height: 20})

// Change image dynamically
newImg, _ := app.LoadImage("logo.png")
imgWidget.SetImage(newImg)
```

**Behavior:**
- Image is scaled to fit widget bounds
- Maintains aspect ratio (future: configurable)
- Supports PNG and JPEG formats
- GPU texture uploading for performance

**Use Cases:**
- Icons
- Logos
- Thumbnails
- Background images

### Spacer

An invisible widget that consumes percentage space for layout alignment.

**Type:** `*wain.Spacer`

**Constructor:**
```go
func NewSpacer(size Size) *Spacer
```

**Example:**
```go
// Create a row with spacer for right-alignment
row := wain.NewRow()

// Left spacer takes 70% width (pushes content right)
row.Add(wain.NewSpacer(wain.Size{Width: 70, Height: 100}))

// Button on the right (30% width)
row.Add(wain.NewButton("Close", wain.Size{Width: 30, Height: 100}))
```

**Use Cases:**
- Pushing widgets to edges
- Creating gaps between widgets
- Centering content
- Flexible spacing in layouts

## Sizing and Layout

### Size Specification

All widgets accept a `Size` struct:

```go
type Size struct {
	Width  float64  // 0-100, percentage of parent
	Height float64  // 0-100, percentage of parent
}
```

### Layout Flow

Containers support two flow directions:

```go
panel.SetFlowDirection(wain.FlowRow)     // Horizontal (left-to-right)
panel.SetFlowDirection(wain.FlowColumn)  // Vertical (top-to-bottom)
```

### Alignment

Control how children are aligned in the cross-axis:

```go
const (
	AlignStart   Align = 0  // Top (column) or Left (row)
	AlignCenter  Align = 1  // Center
	AlignEnd     Align = 2  // Bottom (column) or Right (row)
	AlignStretch Align = 3  // Stretch to fill
)

panel.SetAlign(wain.AlignCenter)
```

### Padding and Gap

```go
panel.SetPadding(10)  // Internal padding (pixels)
panel.SetGap(5)       // Space between children (pixels)
```

### Absolute Positioning

Override automatic layout with absolute positioning:

```go
widget.SetPosition(100, 50)  // Position at (100, 50) pixels
widget.ClearPosition()       // Return to automatic layout
```

### Visibility

Control widget visibility:

```go
widget.SetVisible(false)  // Hide widget
widget.SetVisible(true)   // Show widget
```

## Styling

### Theme System

Apply application-wide themes:

```go
app.SetTheme(wain.DefaultDark())       // Dark theme
app.SetTheme(wain.DefaultLight())      // Light theme
app.SetTheme(wain.HighContrast())      // High contrast theme
```

### Per-Widget Style Overrides

Customize individual widgets:

```go
override := wain.StyleOverride{
	Background:   &wain.RGB(40, 40, 60),
	Foreground:   &wain.White,
	BorderWidth:  wain.IntPtr(2),
	BorderRadius: wain.IntPtr(8),
}
widget.SetStyle(override)
```

### Theme Structure

```go
type Theme struct {
	Background   Color    // Background color
	Foreground   Color    // Text color
	Accent       Color    // Accent/highlight color
	Border       Color    // Border color
	FontSize     float64  // Base font size in points
	Padding      int      // Default padding in pixels
	Gap          int      // Default gap in pixels
	BorderWidth  int      // Default border width in pixels
	BorderRadius int      // Default border radius in pixels
	Scale        float64  // HiDPI scale factor (auto-detected)
}
```

### Built-in Themes

**DefaultDark:**
- Dark background (28, 28, 28)
- Light text (220, 220, 220)
- Blue accent (70, 130, 180)
- Suitable for general use

**DefaultLight:**
- Light background (245, 245, 245)
- Dark text (40, 40, 40)
- Blue accent (70, 130, 180)
- Bright, clean appearance

**HighContrast:**
- Pure black background (0, 0, 0)
- Pure white text (255, 255, 255)
- Yellow accent (255, 255, 0)
- Maximum contrast for accessibility

## Event Handling

### Event Types

```go
type Event interface {
	EventType() EventType
	Consumed() bool
	Consume()
}
```

**Event Types:**
- `PointerMove` - Mouse movement
- `PointerEnter` - Mouse enters widget
- `PointerLeave` - Mouse leaves widget
- `PointerButtonPress` - Mouse button pressed
- `PointerButtonRelease` - Mouse button released
- `PointerScroll` - Mouse wheel scroll
- `KeyPress` - Keyboard key pressed
- `KeyRelease` - Keyboard key released

### Widget Callbacks

**Button:**
```go
button.OnClick(func() {
	// Handle click
})
```

**TextInput:**
```go
input.OnChange(func(text string) {
	// Handle text change
})
```

**ScrollView:**
```go
scroll.OnScroll(func(offset int) {
	// Handle scroll
})
```

### Custom Event Handling

Implement `HandleEvent` for custom widgets:

```go
func (w *MyWidget) HandleEvent(evt Event) bool {
	switch e := evt.(type) {
	case *PointerEvent:
		if e.EventType() == PointerButtonPress {
			// Handle click
			return true  // Event consumed
		}
	case *KeyEvent:
		if e.EventType() == KeyPress {
			// Handle key press
			return true
		}
	}
	return false  // Event not handled
}
```

### Cross-Goroutine Updates

Use `App.Notify()` to update UI from background goroutines:

```go
go func() {
	result := doBackgroundWork()
	
	app.Notify(func() {
		// This runs on the UI goroutine
		label.SetText(result)
		window.Redraw()
	})
}()
```

## Best Practices

### Layout Design

1. **Use percentage sizing** - Avoid hardcoded pixel sizes
2. **Nest containers** - Build complex layouts from simple containers
3. **Use Spacer wisely** - For flexible spacing and alignment
4. **Test window resize** - Ensure layout adapts properly

### Performance

1. **Minimize redraws** - Only call `Redraw()` when necessary
2. **Use damage tracking** - Automatic for internal widgets
3. **Batch updates** - Group multiple widget changes before redraw
4. **Use ScrollView** - For large lists to limit visible widgets

### Accessibility

1. **Use HighContrast theme** - For visually impaired users
2. **Keyboard navigation** - Ensure all actions are keyboard-accessible
3. **Sufficient sizes** - Make clickable widgets large enough (>8% height)
4. **Clear labels** - Use descriptive text

### Code Organization

1. **Separate UI and logic** - Keep business logic in separate functions
2. **Use callbacks** - For event-driven architecture
3. **Reusable components** - Create custom widget types for repeated patterns
4. **Theme consistently** - Use app-wide themes, override sparingly

## Complete Example

Putting it all together - a simple note-taking app:

```go
package main

import (
	"log"
	"github.com/opd-ai/wain"
)

func main() {
	app := wain.NewApp()
	defer app.Close()
	
	app.SetTheme(wain.DefaultLight())
	
	win, _ := app.NewWindow(wain.WindowConfig{
		Title:  "Notes",
		Width:  800,
		Height: 600,
	})
	
	// Layout: Column with header, input, and list
	root := wain.NewColumn()
	root.SetPadding(10)
	root.SetGap(10)
	
	// Header
	header := wain.NewLabel("My Notes", wain.Size{Width: 100, Height: 10})
	root.Add(header)
	
	// Input row
	inputRow := wain.NewRow()
	inputRow.SetGap(10)
	
	input := wain.NewTextInput("Enter note...", wain.Size{Width: 80, Height: 8})
	inputRow.Add(input)
	
	addBtn := wain.NewButton("Add", wain.Size{Width: 20, Height: 8})
	inputRow.Add(addBtn)
	
	root.Add(inputRow)
	
	// Notes list (scrollable)
	scroll := wain.NewScrollView(wain.Size{Width: 100, Height: 82})
	notesList := wain.NewColumn()
	notesList.SetGap(5)
	scroll.SetContent(notesList)
	root.Add(scroll)
	
	// Add note handler
	addBtn.OnClick(func() {
		text := input.Text()
		if text != "" {
			note := wain.NewLabel(text, wain.Size{Width: 100, Height: 5})
			notesList.Add(note)
			input.SetText("")
			win.Redraw()
		}
	})
	
	win.SetRoot(root)
	win.Show()
	app.Run()
}
```

This example demonstrates:
- Multi-level layout (Column → Row → widgets)
- Percentage-based sizing that adapts to window resize
- ScrollView for dynamic content
- Event callbacks for interactivity
- Proper resource management with defer

For more examples, see the `cmd/` directory in the wain repository.
