package models

type DiskInfo struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Type         string `json:"type"`      // "SSD", "HDD", or "NVMe"
	Size         string `json:"size"`
	SizeBytes    int64  `json:"sizeBytes"`
	Mounted      bool   `json:"mounted"`
	MountPoint   string `json:"mountPoint"`
	Filesystem   string `json:"filesystem"`
	IsSystemDisk bool   `json:"isSystemDisk"`
}

type HostDisksResult struct {
	Host  string     `json:"host"`
	Disks []DiskInfo `json:"disks"`
	Error string     `json:"error,omitempty"`
}

type GetDisksResponse struct {
	Results []HostDisksResult `json:"results"`
}

type DiskConsistencyIssue struct {
	Type    string   `json:"type"`    // "count_mismatch", "type_mismatch", "size_mismatch"
	Hosts   []string `json:"hosts"`
	Message string   `json:"message"`
}

type DiskConsistencyResult struct {
	IsConsistent bool                   `json:"isConsistent"`
	Issues       []DiskConsistencyIssue `json:"issues"`
	CommonDisks  []DiskInfo             `json:"commonDisks"` // disks available on ALL hosts
}

type CheckDisksRequest struct {
	Hosts []string   `json:"hosts"`
	SSH   *SSHConfig `json:"ssh"`
}

type ValidateDisksRequest struct {
	Hosts []string   `json:"hosts"`
	SSH   *SSHConfig `json:"ssh"`
	Paths []string   `json:"paths"`
}

type DiskValidation struct {
	Path  string `json:"path"`
	Found bool   `json:"found"`
}

type DiskValidationResult struct {
	Host  string           `json:"host"`
	Valid bool             `json:"valid"`
	Disks []DiskValidation `json:"disks"`
	Error string           `json:"error,omitempty"`
}

type ValidateDisksResponse struct {
	Results []DiskValidationResult `json:"results"`
}
