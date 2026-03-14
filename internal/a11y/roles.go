package a11y

// Role identifies the AT-SPI2 accessible role of a widget.
// Values match the AtspiRole enumeration from the AT-SPI2 specification.
type Role uint32

const (
	// RoleApplication is the top-level application object.
	RoleApplication Role = 75
	// RoleFrame is a top-level window with a title bar.
	RoleFrame Role = 68
	// RolePanel is a generic container panel.
	RolePanel Role = 27
	// RolePushButton is a clickable button.
	RolePushButton Role = 28
	// RoleLabel is a static text label.
	RoleLabel Role = 34
	// RoleEntry is an editable text field.
	RoleEntry Role = 51
	// RoleScrollPane is a scrollable container.
	RoleScrollPane Role = 47
	// RoleScrollBar is a scroll bar widget.
	RoleScrollBar Role = 46
	// RoleRow is a horizontal layout container.
	RoleRow Role = 73
	// RoleColumn is a vertical layout container.
	RoleColumn Role = 20
	// RoleUnknown is the fallback role for unclassified widgets.
	RoleUnknown Role = 0
)

// roleName maps each Role to its human-readable AT-SPI2 name.
var roleName = map[Role]string{
	RoleApplication: "application",
	RoleFrame:       "frame",
	RolePanel:       "panel",
	RolePushButton:  "push button",
	RoleLabel:       "label",
	RoleEntry:       "entry",
	RoleScrollPane:  "scroll pane",
	RoleScrollBar:   "scroll bar",
	RoleRow:         "row",
	RoleColumn:      "column list",
	RoleUnknown:     "unknown",
}

// String returns the human-readable name for the role.
func (r Role) String() string {
	if name, ok := roleName[r]; ok {
		return name
	}
	return "unknown"
}
