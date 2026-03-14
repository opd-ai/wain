//go:build atspi
// +build atspi

package a11y

import (
	"testing"
)

// newTestObject builds an AccessibleObject suitable for headless unit tests.
// The manager field is populated with a minimal stub that has an empty objects
// map so that lookupObject calls during GetIndexInParent return nil gracefully.
func newTestObject(id uint64, role Role) *AccessibleObject {
	mgr := &Manager{objects: map[uint64]*AccessibleObject{}}
	obj := &AccessibleObject{
		id:          id,
		parentID:    0,
		role:        role,
		name:        "widget",
		description: "a test widget",
		enabled:     true,
		text:        "some text",
		x:           5,
		y:           10,
		width:       80,
		height:      40,
		manager:     mgr,
	}
	mgr.objects[id] = obj
	return obj
}

// --- object.go ----------------------------------------------------------

// TestObjectPath verifies the D-Bus path format.
func TestObjectPath(t *testing.T) {
	obj := &AccessibleObject{id: 42}
	want := "/org/a11y/atspi/accessible/42"
	if got := obj.objectPath(); got != want {
		t.Errorf("objectPath: got %q, want %q", got, want)
	}
}

// TestObjectSetBounds verifies that bounds are stored correctly.
func TestObjectSetBounds(t *testing.T) {
	obj := &AccessibleObject{}
	obj.SetBounds(1, 2, 300, 400)
	if obj.x != 1 || obj.y != 2 || obj.width != 300 || obj.height != 400 {
		t.Errorf("SetBounds: got (%d,%d,%d,%d)", obj.x, obj.y, obj.width, obj.height)
	}
}

// TestObjectSetFocused verifies the focused flag toggle.
func TestObjectSetFocused(t *testing.T) {
	obj := &AccessibleObject{}
	obj.SetFocused(true)
	if !obj.focused {
		t.Error("expected focused=true")
	}
	obj.SetFocused(false)
	if obj.focused {
		t.Error("expected focused=false")
	}
}

// TestObjectSetText verifies text content update.
func TestObjectSetText(t *testing.T) {
	obj := &AccessibleObject{}
	obj.SetText("hello")
	if obj.text != "hello" {
		t.Errorf("SetText: got %q, want %q", obj.text, "hello")
	}
}

// TestObjectSetName verifies accessible name update.
func TestObjectSetName(t *testing.T) {
	obj := &AccessibleObject{}
	obj.SetName("new-name")
	if obj.name != "new-name" {
		t.Errorf("SetName: got %q, want %q", obj.name, "new-name")
	}
}

// --- accessible_iface.go ------------------------------------------------

// TestAccessibleIfaceGetDescription verifies description is returned.
func TestAccessibleIfaceGetDescription(t *testing.T) {
	obj := newTestObject(1, RolePanel)
	iface := &accessibleIface{obj}
	desc, err := iface.GetDescription()
	if err != nil {
		t.Fatalf("GetDescription: unexpected error %v", err)
	}
	if desc != "a test widget" {
		t.Errorf("GetDescription: got %q, want %q", desc, "a test widget")
	}
}

// TestAccessibleIfaceGetParentRoot verifies root objects return basePath.
func TestAccessibleIfaceGetParentRoot(t *testing.T) {
	obj := newTestObject(1, RolePanel) // parentID == 0
	iface := &accessibleIface{obj}
	path, err := iface.GetParent()
	if err != nil {
		t.Fatalf("GetParent: unexpected error %v", err)
	}
	if string(path) != basePath {
		t.Errorf("GetParent root: got %q, want %q", path, basePath)
	}
}

// TestAccessibleIfaceGetParentNonRoot verifies child objects return parent path.
func TestAccessibleIfaceGetParentNonRoot(t *testing.T) {
	obj := newTestObject(5, RolePanel)
	obj.parentID = 2
	iface := &accessibleIface{obj}
	path, err := iface.GetParent()
	if err != nil {
		t.Fatalf("GetParent: unexpected error %v", err)
	}
	want := basePath + "/2"
	if string(path) != want {
		t.Errorf("GetParent non-root: got %q, want %q", path, want)
	}
}

// TestAccessibleIfaceGetChildAtIndex verifies valid and out-of-range indices.
func TestAccessibleIfaceGetChildAtIndex(t *testing.T) {
	obj := newTestObject(1, RolePanel)
	obj.childIDs = []uint64{10, 20}
	iface := &accessibleIface{obj}

	path, err := iface.GetChildAtIndex(0)
	if err != nil {
		t.Fatalf("GetChildAtIndex(0): unexpected error %v", err)
	}
	want := basePath + "/10"
	if string(path) != want {
		t.Errorf("GetChildAtIndex(0): got %q, want %q", path, want)
	}

	_, dbusErr := iface.GetChildAtIndex(-1)
	if dbusErr == nil {
		t.Error("expected error for index -1")
	}
	_, dbusErr = iface.GetChildAtIndex(99)
	if dbusErr == nil {
		t.Error("expected error for out-of-range index")
	}
}

// TestAccessibleIfaceGetIndexInParentRoot verifies root returns 0.
func TestAccessibleIfaceGetIndexInParentRoot(t *testing.T) {
	obj := newTestObject(1, RolePanel) // parentID == 0
	iface := &accessibleIface{obj}
	idx, err := iface.GetIndexInParent()
	if err != nil {
		t.Fatalf("GetIndexInParent root: unexpected error %v", err)
	}
	if idx != 0 {
		t.Errorf("GetIndexInParent root: got %d, want 0", idx)
	}
}

// TestAccessibleIfaceGetIndexInParentNotFound verifies nil parent returns 0.
func TestAccessibleIfaceGetIndexInParentNotFound(t *testing.T) {
	obj := newTestObject(1, RolePanel)
	obj.parentID = 99 // parent not in manager.objects
	iface := &accessibleIface{obj}
	idx, err := iface.GetIndexInParent()
	if err != nil {
		t.Fatalf("GetIndexInParent not-found: unexpected error %v", err)
	}
	if idx != 0 {
		t.Errorf("GetIndexInParent not-found: got %d, want 0", idx)
	}
}

// TestAccessibleIfaceGetIndexInParentFound verifies correct index when parent is registered.
func TestAccessibleIfaceGetIndexInParentFound(t *testing.T) {
	mgr := &Manager{objects: map[uint64]*AccessibleObject{}}
	parent := &AccessibleObject{id: 1, childIDs: []uint64{2, 3, 4}, manager: mgr}
	child := &AccessibleObject{id: 3, parentID: 1, manager: mgr}
	mgr.objects[1] = parent
	mgr.objects[3] = child

	iface := &accessibleIface{child}
	idx, err := iface.GetIndexInParent()
	if err != nil {
		t.Fatalf("GetIndexInParent found: unexpected error %v", err)
	}
	if idx != 1 {
		t.Errorf("GetIndexInParent found: got %d, want 1", idx)
	}
}

// TestAccessibleIfaceGetRelationSet verifies empty relation set.
func TestAccessibleIfaceGetRelationSet(t *testing.T) {
	obj := newTestObject(1, RolePanel)
	iface := &accessibleIface{obj}
	rels, err := iface.GetRelationSet()
	if err != nil {
		t.Fatalf("GetRelationSet: unexpected error %v", err)
	}
	if len(rels) != 0 {
		t.Errorf("GetRelationSet: got %d relations, want 0", len(rels))
	}
}

// TestAccessibleIfaceGetAttributes verifies empty attribute map.
func TestAccessibleIfaceGetAttributes(t *testing.T) {
	obj := newTestObject(1, RolePanel)
	iface := &accessibleIface{obj}
	attrs, err := iface.GetAttributes()
	if err != nil {
		t.Fatalf("GetAttributes: unexpected error %v", err)
	}
	if len(attrs) != 0 {
		t.Errorf("GetAttributes: got %d attrs, want 0", len(attrs))
	}
}

// TestAccessibleIfaceGetApplication verifies the application root path.
func TestAccessibleIfaceGetApplication(t *testing.T) {
	obj := newTestObject(1, RolePanel)
	iface := &accessibleIface{obj}
	path, err := iface.GetApplication()
	if err != nil {
		t.Fatalf("GetApplication: unexpected error %v", err)
	}
	want := basePath + "/root"
	if string(path) != want {
		t.Errorf("GetApplication: got %q, want %q", path, want)
	}
}

// --- action_iface.go ----------------------------------------------------

// TestActionIfaceGetDescription covers the description getter.
func TestActionIfaceGetDescription(t *testing.T) {
	obj := &AccessibleObject{
		actions: []objectAction{
			{name: "click", description: "activate the widget"},
		},
	}
	a := &actionIface{obj}
	desc, err := a.GetDescription(0)
	if err != nil || desc != "activate the widget" {
		t.Errorf("GetDescription(0): got %q %v", desc, err)
	}
	_, dbusErr := a.GetDescription(99)
	if dbusErr == nil {
		t.Error("expected error for out-of-range index")
	}
}

// TestActionIfaceGetKeyBinding covers the key-binding getter.
func TestActionIfaceGetKeyBinding(t *testing.T) {
	obj := &AccessibleObject{
		actions: []objectAction{
			{name: "activate", keyBinding: "Return"},
		},
	}
	a := &actionIface{obj}
	kb, err := a.GetKeyBinding(0)
	if err != nil || kb != "Return" {
		t.Errorf("GetKeyBinding(0): got %q %v", kb, err)
	}
	_, dbusErr := a.GetKeyBinding(-1)
	if dbusErr == nil {
		t.Error("expected error for index -1")
	}
}

// TestActionIfaceDoActionNilCallback verifies nil callback returns false without error.
func TestActionIfaceDoActionNilCallback(t *testing.T) {
	obj := &AccessibleObject{
		actions: []objectAction{
			{name: "noop", do: nil},
		},
	}
	a := &actionIface{obj}
	ok, err := a.DoAction(0)
	if err != nil {
		t.Fatalf("DoAction(nil callback): unexpected error %v", err)
	}
	if ok {
		t.Error("DoAction(nil callback): expected false")
	}
}

// --- component_iface.go -------------------------------------------------

// TestComponentIfaceGetAccessibleAtPoint verifies inside/outside cases.
func TestComponentIfaceGetAccessibleAtPoint(t *testing.T) {
	obj := newTestObject(7, RolePanel)
	c := &componentIface{obj}

	// Inside: returns the object's own path.
	path, err := c.GetAccessibleAtPoint(20, 30, 0)
	if err != nil {
		t.Fatalf("GetAccessibleAtPoint inside: %v", err)
	}
	want := basePath + "/7"
	if string(path) != want {
		t.Errorf("GetAccessibleAtPoint inside: got %q, want %q", path, want)
	}

	// Outside: returns root path "/".
	path, err = c.GetAccessibleAtPoint(999, 999, 0)
	if err != nil {
		t.Fatalf("GetAccessibleAtPoint outside: %v", err)
	}
	if string(path) != "/" {
		t.Errorf("GetAccessibleAtPoint outside: got %q, want /", path)
	}
}

// TestComponentIfaceGetPosition verifies position extraction.
func TestComponentIfaceGetPosition(t *testing.T) {
	obj := newTestObject(1, RolePanel)
	c := &componentIface{obj}
	pos, err := c.GetPosition(0)
	if err != nil {
		t.Fatalf("GetPosition: %v", err)
	}
	if pos.X != 5 || pos.Y != 10 {
		t.Errorf("GetPosition: got (%d,%d), want (5,10)", pos.X, pos.Y)
	}
}

// TestComponentIfaceGrabFocus verifies GrabFocus sets focused=true.
func TestComponentIfaceGrabFocus(t *testing.T) {
	obj := &AccessibleObject{focused: false}
	c := &componentIface{obj}
	ok, err := c.GrabFocus()
	if err != nil || !ok {
		t.Errorf("GrabFocus: got %v %v", ok, err)
	}
	if !obj.focused {
		t.Error("GrabFocus: expected focused=true after call")
	}
}

// TestComponentIfaceScrollTo verifies ScrollTo always succeeds.
func TestComponentIfaceScrollTo(t *testing.T) {
	obj := &AccessibleObject{}
	c := &componentIface{obj}
	ok, err := c.ScrollTo(0)
	if err != nil || !ok {
		t.Errorf("ScrollTo: got %v %v", ok, err)
	}
}

// --- manager.go (D-Bus-free paths) --------------------------------------

// newHeadlessManager builds a Manager without a D-Bus connection.
// Only lookupObject and the Set* methods (which do not call conn) are safe to use.
func newHeadlessManager() *Manager {
	return &Manager{
		objects: map[uint64]*AccessibleObject{},
		nextID:  0,
	}
}

// TestManagerLookupObject verifies lookup returns nil for unknown IDs.
func TestManagerLookupObject(t *testing.T) {
	m := newHeadlessManager()
	if obj := m.lookupObject(42); obj != nil {
		t.Errorf("lookupObject: expected nil for unknown ID, got %v", obj)
	}
}

// TestManagerLookupObjectFound verifies lookup returns the registered object.
func TestManagerLookupObjectFound(t *testing.T) {
	m := newHeadlessManager()
	obj := &AccessibleObject{id: 5, manager: m}
	m.objects[5] = obj
	if got := m.lookupObject(5); got != obj {
		t.Errorf("lookupObject: expected %p, got %p", obj, got)
	}
}

// TestManagerSetBounds verifies bounds forwarding to the object.
func TestManagerSetBounds(t *testing.T) {
	m := newHeadlessManager()
	obj := &AccessibleObject{id: 1, manager: m}
	m.objects[1] = obj
	m.SetBounds(1, 10, 20, 100, 50)
	if obj.x != 10 || obj.y != 20 || obj.width != 100 || obj.height != 50 {
		t.Errorf("SetBounds: got (%d,%d,%d,%d)", obj.x, obj.y, obj.width, obj.height)
	}
}

// TestManagerSetBoundsUnknown verifies SetBounds is a no-op for unknown ID.
func TestManagerSetBoundsUnknown(t *testing.T) {
	m := newHeadlessManager()
	m.SetBounds(99, 1, 2, 3, 4) // must not panic
}

// TestManagerSetText verifies text forwarding.
func TestManagerSetText(t *testing.T) {
	m := newHeadlessManager()
	obj := &AccessibleObject{id: 1, manager: m}
	m.objects[1] = obj
	m.SetText(1, "new text")
	if obj.text != "new text" {
		t.Errorf("SetText: got %q, want %q", obj.text, "new text")
	}
}

// TestManagerSetTextUnknown verifies SetText is a no-op for unknown ID.
func TestManagerSetTextUnknown(t *testing.T) {
	m := newHeadlessManager()
	m.SetText(99, "x") // must not panic
}

// TestManagerSetName verifies name forwarding.
func TestManagerSetName(t *testing.T) {
	m := newHeadlessManager()
	obj := &AccessibleObject{id: 1, manager: m}
	m.objects[1] = obj
	m.SetName(1, "new-name")
	if obj.name != "new-name" {
		t.Errorf("SetName: got %q, want %q", obj.name, "new-name")
	}
}

// TestManagerSetNameUnknown verifies SetName is a no-op for unknown ID.
func TestManagerSetNameUnknown(t *testing.T) {
	m := newHeadlessManager()
	m.SetName(99, "x") // must not panic
}

// TestManagerSetFocusedFalse verifies SetFocused(false) does not emit D-Bus signals.
// This path is safe because emitFocusEvent is only called when focused==true.
func TestManagerSetFocusedFalse(t *testing.T) {
	m := newHeadlessManager()
	obj := &AccessibleObject{id: 1, focused: true, manager: m}
	m.objects[1] = obj
	m.SetFocused(1, false)
	if obj.focused {
		t.Error("SetFocused(false): expected focused=false")
	}
}

// TestManagerSetFocusedUnknown verifies SetFocused is a no-op for unknown ID.
func TestManagerSetFocusedUnknown(t *testing.T) {
	m := newHeadlessManager()
	m.SetFocused(99, false) // must not panic
}

// --- states.go (high-bit branches) -------------------------------------

// TestStateSetHighIndex exercises Set, Clear, Has for indices >= 32.
func TestStateSetHighIndex(t *testing.T) {
	const highState StateIndex = 40 // >= 32
	var ss StateSet
	ss.Set(highState)
	if !ss.Has(highState) {
		t.Error("expected high-index state to be set")
	}
	if ss[1] == 0 {
		t.Error("expected second word to be non-zero")
	}
	ss.Clear(highState)
	if ss.Has(highState) {
		t.Error("expected high-index state to be cleared")
	}
}

// --- text_iface.go (remaining uncovered functions) ----------------------

// TestTextIfaceGetTextAfterOffset covers GetTextAfterOffset valid and out-of-range.
func TestTextIfaceGetTextAfterOffset(t *testing.T) {
	obj := &AccessibleObject{text: "abc"}
	tx := &textIface{obj}

	ch, start, end, err := tx.GetTextAfterOffset(1, 0)
	if err != nil || ch != "b" || start != 1 || end != 2 {
		t.Errorf("GetTextAfterOffset(1): got %q [%d,%d] %v", ch, start, end, err)
	}

	// Out of range — returns empty string, no error.
	ch, _, _, err = tx.GetTextAfterOffset(99, 0)
	if err != nil || ch != "" {
		t.Errorf("GetTextAfterOffset out-of-range: got %q %v", ch, err)
	}
}

// TestTextIfaceGetTextBeforeOffset covers GetTextBeforeOffset valid and edge cases.
func TestTextIfaceGetTextBeforeOffset(t *testing.T) {
	obj := &AccessibleObject{text: "abc"}
	tx := &textIface{obj}

	ch, start, end, err := tx.GetTextBeforeOffset(2, 0)
	if err != nil || ch != "b" || start != 1 || end != 2 {
		t.Errorf("GetTextBeforeOffset(2): got %q [%d,%d] %v", ch, start, end, err)
	}

	// Offset 0 → returns empty.
	ch, _, _, err = tx.GetTextBeforeOffset(0, 0)
	if err != nil || ch != "" {
		t.Errorf("GetTextBeforeOffset(0): got %q %v", ch, err)
	}

	// Offset beyond length → returns empty.
	ch, _, _, err = tx.GetTextBeforeOffset(99, 0)
	if err != nil || ch != "" {
		t.Errorf("GetTextBeforeOffset(99): got %q %v", ch, err)
	}
}

// TestTextIfaceGetDefaultAttributeSet covers the default attribute set getter.
func TestTextIfaceGetDefaultAttributeSet(t *testing.T) {
	obj := &AccessibleObject{text: "x"}
	tx := &textIface{obj}
	attrs, err := tx.GetDefaultAttributeSet()
	if err != nil || len(attrs) != 0 {
		t.Errorf("GetDefaultAttributeSet: got %v %v", attrs, err)
	}
}

// TestTextIfaceGetTextAtOffsetOutOfRange covers the offset >= n branch.
func TestTextIfaceGetTextAtOffsetOutOfRange(t *testing.T) {
	obj := &AccessibleObject{text: "hi"}
	tx := &textIface{obj}
	ch, start, end, err := tx.GetTextAtOffset(99, 0)
	if err != nil || ch != "" || start != 0 || end != 0 {
		t.Errorf("GetTextAtOffset(99): got %q [%d,%d] %v", ch, start, end, err)
	}
}

// TestTextIfaceSetCaretOutOfRange covers the out-of-range offset branch.
func TestTextIfaceSetCaretOutOfRange(t *testing.T) {
	obj := &AccessibleObject{text: "hi"}
	tx := &textIface{obj}
	ok, err := tx.SetCaret(99)
	if err != nil || ok {
		t.Errorf("SetCaret(99): expected (false, nil), got (%v, %v)", ok, err)
	}
}
