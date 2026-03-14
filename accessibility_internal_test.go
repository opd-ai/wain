package wain

import "testing"

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
