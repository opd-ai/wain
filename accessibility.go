package wain

import (
	"log"
	"sync"

	"github.com/opd-ai/wain/internal/a11y"
)

// AccessibleWidget is an optional interface that public widgets can implement
// to provide richer accessibility metadata.
//
// Widgets that do not implement AccessibleWidget receive default accessibility
// information inferred from their type (role) and any Text() method.
//
// Accessibility is enabled per-window via EnableAccessibility. The Manager
// calls these methods when registering each widget in the AT-SPI2 tree.
type AccessibleWidget interface {
	// AccessibleName returns the short name announced by screen readers.
	// For buttons this is typically the button label; for inputs, a field label.
	AccessibleName() string

	// AccessibleDescription returns a longer description for screen readers.
	// Return an empty string to omit the description.
	AccessibleDescription() string
}

// AccessibilityManager wraps the internal a11y.Manager and associates widget
// accessible IDs with PublicWidget instances for lifecycle management.
type AccessibilityManager struct {
	mgr       *a11y.Manager
	mu        sync.RWMutex
	widgetIDs map[Widget]uint64
}

// EnableAccessibility attaches AT-SPI2 accessibility to a wain application.
// appName should be a short identifier for the application (e.g. "my-app").
//
// Returns nil if D-Bus is unavailable; accessibility is silently disabled and
// the application continues normally. Check the log for a diagnostic message.
//
//	am := wain.EnableAccessibility("my-app")
//	if am != nil {
//	    id := am.Register(myButton, 0)
//	    am.SetBounds(id, 10, 10, 80, 30)
//	}
func EnableAccessibility(appName string) *AccessibilityManager {
	mgr, err := a11y.NewManager(appName)
	if err != nil {
		log.Printf("wain: accessibility disabled: %v", err)
		return nil
	}
	return &AccessibilityManager{
		mgr:       mgr,
		widgetIDs: make(map[Widget]uint64),
	}
}

// Close releases the D-Bus connection and removes all exported objects.
// Call this when the application exits.
func (am *AccessibilityManager) Close() {
	if am != nil {
		am.mgr.Close()
	}
}

// RegisterPanel registers a generic container widget and returns its ID.
// parentID 0 means this is a root-level object.
func (am *AccessibilityManager) RegisterPanel(name string, parentID uint64) uint64 {
	return am.mgr.RegisterPanel(name, parentID)
}

// RegisterButton registers a button widget.
// onClick is called when an assistive tool activates the button.
func (am *AccessibilityManager) RegisterButton(name string, parentID uint64, onClick func() bool) uint64 {
	return am.mgr.RegisterButton(name, parentID, onClick)
}

// RegisterLabel registers a static label widget.
func (am *AccessibilityManager) RegisterLabel(text string, parentID uint64) uint64 {
	return am.mgr.RegisterLabel(text, parentID)
}

// RegisterEntry registers an editable text field widget.
func (am *AccessibilityManager) RegisterEntry(name string, parentID uint64) uint64 {
	return am.mgr.RegisterEntry(name, parentID)
}

// RegisterScrollPane registers a scrollable container widget.
func (am *AccessibilityManager) RegisterScrollPane(name string, parentID uint64) uint64 {
	return am.mgr.RegisterScrollPane(name, parentID)
}

// SetBounds updates the on-screen rectangle for the accessible object with id.
func (am *AccessibilityManager) SetBounds(id uint64, x, y, width, height int32) {
	am.mgr.SetBounds(id, x, y, width, height)
}

// SetFocused updates the keyboard-focus state of the accessible object with id.
// When focused is true, an AT-SPI2 focus event is emitted to notify screen readers.
func (am *AccessibilityManager) SetFocused(id uint64, focused bool) {
	am.mgr.SetFocused(id, focused)
}

// SetText updates the text content of an Entry object with id.
func (am *AccessibilityManager) SetText(id uint64, text string) {
	am.mgr.SetText(id, text)
}

// SetName updates the accessible name of the object with id.
func (am *AccessibilityManager) SetName(id uint64, name string) {
	am.mgr.SetName(id, name)
}

// AssociateWidget records that widget corresponds to the accessible object id.
// Call this after registering a widget so that WireFocusManager can emit focus
// events when the widget gains or loses keyboard focus.
func (am *AccessibilityManager) AssociateWidget(widget Widget, id uint64) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.widgetIDs[widget] = id
}

// WireFocusManager installs a focus-change hook on fm so that AT-SPI2
// focus signals are emitted automatically when keyboard focus changes.
// Only widgets previously registered with AssociateWidget will trigger signals.
func (am *AccessibilityManager) WireFocusManager(fm *FocusManager) {
	fm.SetFocusChangeHook(func(w Widget) {
		am.mu.RLock()
		id, ok := am.widgetIDs[w]
		am.mu.RUnlock()
		if ok {
			am.mgr.SetFocused(id, w != nil)
		}
	})
}
