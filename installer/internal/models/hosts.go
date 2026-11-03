package models

type CheckHostsRequest struct {
	Hosts []string   `json:"hosts"`
	SSH   *SSHConfig `json:"ssh"`
	Ports []int      `json:"ports,omitempty"` // defaults to {2135, 8765}
}

type PortCheckResult struct {
	Port      int    `json:"port"`
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
}

type TimeSyncResult struct {
	NTPActive     bool   `json:"ntpActive"`
	NTPDaemon     string `json:"ntpDaemon"`     // "chrony" | "ntpd" | ""
	NTPSynced     bool   `json:"ntpSynced"`
	ClockOffsetMs int64  `json:"clockOffsetMs"`
	Error         string `json:"error,omitempty"`
}

type TimeSyncSummary struct {
	MaxDriftMs   int64  `json:"maxDriftMs"`
	Status       string `json:"status"`       // "ok" | "warning" | "error"
	NTPNotSynced bool   `json:"ntpNotSynced"` // true when NTP daemon is active but not synchronized on ≥1 host (distinct from actual drift)
	Threshold    struct {
		WarningMs int64 `json:"warningMs"`
		ErrorMs   int64 `json:"errorMs"`
	} `json:"threshold"`
}

type CheckHostResult struct {
	Host         string            `json:"host"`
	SSH          bool              `json:"ssh"`
	SSHError     string            `json:"sshError,omitempty"`
	Python       bool              `json:"python"`
	PythonVer    string            `json:"pythonVersion,omitempty"`
	PythonError  string            `json:"pythonError,omitempty"`
	CPUCores     int               `json:"cpuCores"`
	Ports        bool              `json:"ports"`
	PortsError   string            `json:"portsError,omitempty"`
	PortDetails  []PortCheckResult `json:"portDetails,omitempty"`
	TimeSync     TimeSyncResult    `json:"timeSync"`
	ClockEpochNs int64             `json:"clockEpochNs"` // raw epoch nanoseconds for drift calc
}

type CheckHostsResponse struct {
	Results  []CheckHostResult `json:"results"`
	TimeSync *TimeSyncSummary  `json:"timeSync,omitempty"`
}
