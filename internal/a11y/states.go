package a11y

// StateIndex identifies a bit position in an AT-SPI2 state set.
// The state set is represented as two uint32 values where each bit corresponds
// to a state index. Values match the AtspiStateType enumeration.
type StateIndex uint

// State constants match the AtspiStateType enumeration.
// Used by atspi build-tagged code in object.go and manager.go.
const (
	stateActive    StateIndex = 0  //nolint:unused // used with -tags=atspi
	stateEnabled   StateIndex = 11 //nolint:unused // used with -tags=atspi
	stateFocusable StateIndex = 12 //nolint:unused // used with -tags=atspi
	stateFocused   StateIndex = 13 //nolint:unused // used with -tags=atspi
	stateEditable  StateIndex = 10 //nolint:unused // used with -tags=atspi
	stateVisible   StateIndex = 27 //nolint:unused // used with -tags=atspi
	stateShowing   StateIndex = 26 //nolint:unused // used with -tags=atspi
	stateSensitive StateIndex = 28 //nolint:unused // used with -tags=atspi
)

// StateSet encodes AT-SPI2 widget states as a pair of uint32 bitfields.
// Each StateIndex maps to a bit position (0–63) spread across two words.
type StateSet [2]uint32

// Set marks the given state as active.
func (s *StateSet) Set(idx StateIndex) {
	if idx < 32 {
		s[0] |= 1 << idx
	} else {
		s[1] |= 1 << (idx - 32)
	}
}

// Clear removes the given state.
func (s *StateSet) Clear(idx StateIndex) {
	if idx < 32 {
		s[0] &^= 1 << idx
	} else {
		s[1] &^= 1 << (idx - 32)
	}
}

// Has reports whether the given state is active.
func (s StateSet) Has(idx StateIndex) bool {
	if idx < 32 {
		return s[0]&(1<<idx) != 0
	}
	return s[1]&(1<<(idx-32)) != 0
}

// Uint32s returns the two-word representation expected by AT-SPI2.
func (s StateSet) Uint32s() []uint32 {
	return []uint32{s[0], s[1]}
}

// defaultStates returns the base StateSet applied to all visible widgets.
func defaultStates(focused, enabled bool) StateSet { //nolint:unused // used with -tags=atspi
	var ss StateSet
	ss.Set(stateVisible)
	ss.Set(stateShowing)
	ss.Set(stateSensitive)
	if enabled {
		ss.Set(stateEnabled)
	}
	if focused {
		ss.Set(stateFocused)
		ss.Set(stateFocusable)
	} else {
		ss.Set(stateFocusable)
	}
	return ss
}
