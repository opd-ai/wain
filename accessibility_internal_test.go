package wain

import (
	"testing"

	"github.com/opd-ai/wain/internal/a11y"
)

// TestAccessibilityManagerAssociateWidget verifies AssociateWidget stores the mapping.
func TestAccessibilityManagerAssociateWidget(t *testing.T) {
	am := &AccessibilityManager{
		widgetIDs: make(map[Widget]uint64),
	}

	w := &BaseWidget{}
	am.AssociateWidget(w, 42)

	am.mu.RLock()
	id, ok := am.widgetIDs[w]
	am.mu.RUnlock()

	if !ok {
		t.Error("AssociateWidget: widget not stored")
	}
	if id != 42 {
		t.Errorf("AssociateWidget: id = %d, want 42", id)
	}
}

// TestAccessibilityManagerWireFocusManager verifies the focus hook is installed.
func TestAccessibilityManagerWireFocusManager(t *testing.T) {
	am := &AccessibilityManager{
		widgetIDs: make(map[Widget]uint64),
	}

	fm := NewFocusManager()
	am.WireFocusManager(fm)

	// Calling Focus with an unregistered widget must not panic.
	w := &BaseWidget{}
	fm.Focus(w)
}

// TestAccessibilityManagerWireFocusManagerWithRegistered triggers the focus hook
// for a widget that was registered with AssociateWidget.
func TestAccessibilityManagerWireFocusManagerWithRegistered(t *testing.T) {
	am := &AccessibilityManager{
		widgetIDs: make(map[Widget]uint64),
	}

	w := &BaseWidget{}
	am.AssociateWidget(w, 99)

	fm := NewFocusManager()
	fm.SetChain([]Widget{w})
	am.WireFocusManager(fm)

	// Trigger focus change; hook should be called for the registered widget.
	fm.Focus(w)
}

// newTestAccessibilityManager creates an AccessibilityManager with a stub mgr for
// headless testing (no AT-SPI2 required).
func newTestAccessibilityManager() *AccessibilityManager {
	return &AccessibilityManager{
		mgr:       &a11y.Manager{},
		widgetIDs: make(map[Widget]uint64),
	}
}

// TestAccessibilityManagerClose_NonNil covers the am.mgr.Close() path.
func TestAccessibilityManagerClose_NonNil(t *testing.T) {
	am := newTestAccessibilityManager()
	am.Close() // must not panic
}

// TestAccessibilityManagerRegisterMethods covers RegisterPanel/Button/Label/Entry/ScrollPane.
func TestAccessibilityManagerRegisterMethods(t *testing.T) {
	am := newTestAccessibilityManager()
	am.RegisterPanel("panel", 0)
	am.RegisterButton("btn", 0, nil)
	am.RegisterLabel("lbl", 0)
	am.RegisterEntry("entry", 0)
	am.RegisterScrollPane("scroll", 0)
}

// TestAccessibilityManagerSetMethods covers SetBounds/SetFocused/SetText/SetName.
func TestAccessibilityManagerSetMethods(t *testing.T) {
	am := newTestAccessibilityManager()
	am.SetBounds(1, 0, 0, 100, 50)
	am.SetFocused(1, true)
	am.SetText(1, "hello")
	am.SetName(1, "mywidget")
}
