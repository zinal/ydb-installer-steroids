package models

type Flow string

const (
	FlowNone      Flow = ""
	FlowYDB       Flow = "ydb"
	FlowDeleteYDB Flow = "delete-ydb"
)

// IsValidFlow reports whether the given flow string is a known flow.
func IsValidFlow(f Flow) bool {
	switch f {
	case FlowNone, FlowYDB, FlowDeleteYDB:
		return true
	default:
		return false
	}
}

type StepStatus string

const (
	StatusActive    StepStatus = "active"
	StatusDone      StepStatus = "done"
	StatusError     StepStatus = "error"
	StatusClickable StepStatus = "clickable"
	StatusDisabled  StepStatus = "disabled"
)

type Step struct {
	ID     string     `json:"id"`
	Status StepStatus `json:"status"`
}

type UIState struct {
	CurrentStep     string   `json:"current_step"`
	Visited         []string `json:"visited"`
	Errored         []string `json:"errored"`
	ConfigGenerated bool     `json:"config_generated"`
}

type Session struct {
	Flow           Flow           `json:"flow"`
	CurrentStep    string         `json:"current_step"`
	Steps          []Step         `json:"steps"`
	DefaultLang    string         `json:"default_lang"`
	Version        string         `json:"version"`
	Inventory      *InventoryData `json:"inventory,omitempty"`
	InventoryFound bool           `json:"inventory_found"`
	ConfigFound    bool           `json:"config_found"`
	UIState        *UIState       `json:"ui_state,omitempty"`
}

type SessionUpdate struct {
	Flow        *Flow          `json:"flow,omitempty"`
	CurrentStep *string        `json:"current_step,omitempty"`
	Steps       *[]Step        `json:"steps,omitempty"`
	Inventory   *InventoryData `json:"inventory,omitempty"`
	UIState     *UIState       `json:"ui_state,omitempty"`
}
