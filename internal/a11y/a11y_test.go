package a11y

import (
	"testing"
)

// TestStateSet verifies Set, Clear, Has, and Uint32s operations.
func TestStateSet(t *testing.T) {
	var ss StateSet
	ss.Set(stateEnabled)
	ss.Set(stateVisible)

	if !ss.Has(stateEnabled) {
		t.Error("expected stateEnabled to be set")
	}
	if !ss.Has(stateVisible) {
		t.Error("expected stateVisible to be set")
	}
	if ss.Has(stateFocused) {
		t.Error("expected stateFocused to be unset")
	}

	ss.Clear(stateEnabled)
	if ss.Has(stateEnabled) {
		t.Error("expected stateEnabled to be cleared")
	}

	words := ss.Uint32s()
	if len(words) != 2 {
		t.Fatalf("expected 2 words, got %d", len(words))
	}
}

// TestStateSetHighBit verifies states above index 31 use the second word.
func TestStateSetHighBit(t *testing.T) {
	var ss StateSet
	ss.Set(stateVisible)   // index 27
	ss.Set(stateSensitive) // index 28
	words := ss.Uint32s()
	// Both are in the first word (indices < 32).
	if words[0] == 0 {
		t.Error("expected first word to be non-zero for stateVisible/stateSensitive")
	}
	if words[1] != 0 {
		t.Error("expected second word to be zero")
	}
}

// TestDefaultStates checks that defaultStates populates the expected bits.
func TestDefaultStates(t *testing.T) {
	enabled := defaultStates(false, true)
	if !enabled.Has(stateEnabled) {
		t.Error("enabled widget should have stateEnabled")
	}
	if !enabled.Has(stateVisible) {
		t.Error("enabled widget should have stateVisible")
	}
	if enabled.Has(stateFocused) {
		t.Error("unfocused widget should not have stateFocused")
	}

	focused := defaultStates(true, true)
	if !focused.Has(stateFocused) {
		t.Error("focused widget should have stateFocused")
	}
	if !focused.Has(stateFocusable) {
		t.Error("focused widget should have stateFocusable")
	}
}

// TestAccessibleObjectSnap verifies that snap captures a consistent copy.
func TestAccessibleObjectSnap(t *testing.T) {
	obj := &AccessibleObject{
		id:       1,
		parentID: 0,
		role:     RolePushButton,
		name:     "OK",
		enabled:  true,
		width:    80,
		height:   30,
	}
	s := obj.snap()
	if s.name != "OK" {
		t.Errorf("name: want OK, got %q", s.name)
	}
	if s.role != RolePushButton {
		t.Errorf("role: want RolePushButton, got %v", s.role)
	}
	if s.width != 80 || s.height != 30 {
		t.Errorf("bounds: want 80×30, got %d×%d", s.width, s.height)
	}
}

// TestAccessibleObjectAddChild verifies child registration is thread-safe.
func TestAccessibleObjectAddChild(t *testing.T) {
	obj := &AccessibleObject{id: 1}
	obj.addChild(2)
	obj.addChild(3)
	s := obj.snap()
	if len(s.childIDs) != 2 {
		t.Fatalf("expected 2 children, got %d", len(s.childIDs))
	}
	if s.childIDs[0] != 2 || s.childIDs[1] != 3 {
		t.Errorf("child IDs: want [2 3], got %v", s.childIDs)
	}
}

// TestRoleString verifies human-readable role names.
func TestRoleString(t *testing.T) {
	cases := []struct {
		role Role
		want string
	}{
		{RolePushButton, "push button"},
		{RoleEntry, "entry"},
		{RoleLabel, "label"},
		{RoleUnknown, "unknown"},
		{Role(999), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.role.String(); got != tc.want {
			t.Errorf("Role(%d).String() = %q, want %q", tc.role, got, tc.want)
		}
	}
}

// TestAccessibleIface exercises the Accessible D-Bus interface wrapper.
func TestAccessibleIface(t *testing.T) {
	obj := &AccessibleObject{
		id:          5,
		parentID:    0,
		role:        RolePanel,
		name:        "test-panel",
		description: "a test panel",
		enabled:     true,
		childIDs:    []uint64{6, 7},
		manager: &Manager{
			objects: map[uint64]*AccessibleObject{},
		},
	}
	obj.manager.objects[5] = obj

	iface := &accessibleIface{obj}

	name, err := iface.GetName()
	if err != nil || name != "test-panel" {
		t.Errorf("GetName: got %q %v", name, err)
	}

	role, err := iface.GetRole()
	if err != nil || role != uint32(RolePanel) {
		t.Errorf("GetRole: got %d %v", role, err)
	}

	count, err := iface.GetChildCount()
	if err != nil || count != 2 {
		t.Errorf("GetChildCount: got %d %v", count, err)
	}

	children, err := iface.GetChildren()
	if err != nil || len(children) != 2 {
		t.Errorf("GetChildren: got %v %v", children, err)
	}

	state, err := iface.GetState()
	if err != nil || len(state) != 2 {
		t.Errorf("GetState: got %v %v", state, err)
	}
}

// TestComponentIface exercises the Component D-Bus interface wrapper.
func TestComponentIface(t *testing.T) {
	obj := &AccessibleObject{x: 10, y: 20, width: 100, height: 50}
	c := &componentIface{obj}

	inside, err := c.Contains(50, 40, 0)
	if err != nil || !inside {
		t.Errorf("Contains(50,40): want true, got %v %v", inside, err)
	}

	outside, err := c.Contains(200, 200, 0)
	if err != nil || outside {
		t.Errorf("Contains(200,200): want false, got %v %v", outside, err)
	}

	ext, err := c.GetExtents(0)
	if err != nil {
		t.Fatalf("GetExtents: %v", err)
	}
	if ext.X != 10 || ext.Y != 20 || ext.Width != 100 || ext.Height != 50 {
		t.Errorf("GetExtents: got %+v", ext)
	}

	sz, err := c.GetSize()
	if err != nil || sz.Width != 100 || sz.Height != 50 {
		t.Errorf("GetSize: got %+v %v", sz, err)
	}
}

// TestActionIface exercises the Action D-Bus interface wrapper.
func TestActionIface(t *testing.T) {
	activated := false
	obj := &AccessibleObject{
		actions: []objectAction{
			{name: "click", description: "press the button", do: func() bool {
				activated = true
				return true
			}},
		},
	}
	a := &actionIface{obj}

	n, err := a.GetNActions()
	if err != nil || n != 1 {
		t.Errorf("GetNActions: got %d %v", n, err)
	}

	ok, err := a.DoAction(0)
	if err != nil || !ok || !activated {
		t.Errorf("DoAction: got %v %v, activated=%v", ok, err, activated)
	}

	name, err := a.GetName(0)
	if err != nil || name != "click" {
		t.Errorf("GetName: got %q %v", name, err)
	}

	_, dbusErr := a.DoAction(99)
	if dbusErr == nil {
		t.Error("expected error for out-of-range action index")
	}
}

// TestTextIface exercises the Text D-Bus interface wrapper.
func TestTextIface(t *testing.T) {
	obj := &AccessibleObject{text: "hello world"}
	tx := &textIface{obj}

	count, err := tx.GetCharacterCount()
	if err != nil || count != 11 {
		t.Errorf("GetCharacterCount: got %d %v", count, err)
	}

	text, err := tx.GetText(0, 5)
	if err != nil || text != "hello" {
		t.Errorf("GetText(0,5): got %q %v", text, err)
	}

	full, err := tx.GetText(0, -1)
	if err != nil || full != "hello world" {
		t.Errorf("GetText full: got %q %v", full, err)
	}

	ok, err := tx.SetCaret(3)
	if err != nil || !ok {
		t.Errorf("SetCaret: got %v %v", ok, err)
	}

	caret, err := tx.GetCaret()
	if err != nil || caret != 3 {
		t.Errorf("GetCaret: got %d %v", caret, err)
	}

	at, start, end, err := tx.GetTextAtOffset(2, 0)
	if err != nil || at != "l" || start != 2 || end != 3 {
		t.Errorf("GetTextAtOffset(2): got %q [%d,%d] %v", at, start, end, err)
	}
}

// TestClampOffsets verifies offset clamping behaviour.
func TestClampOffsets(t *testing.T) {
	cases := []struct {
		text         string
		start, end   int32
		wantS, wantE int
	}{
		{"hello", 0, 5, 0, 5},
		{"hello", 0, -1, 0, 5},
		{"hello", -1, 3, 0, 3},
		{"hello", 3, 1, 3, 3}, // start > end → clamp start to end
	}
	for _, tc := range cases {
		s, e := clampOffsets(tc.text, tc.start, tc.end)
		if s != tc.wantS || e != tc.wantE {
			t.Errorf("clampOffsets(%q,%d,%d) = (%d,%d), want (%d,%d)",
				tc.text, tc.start, tc.end, s, e, tc.wantS, tc.wantE)
		}
	}
}
