package models

import (
	"sync"

	"ydb-installer/internal/services/ansible"
	"ydb-installer/internal/services/logs"
	"ydb-installer/internal/services/progress"
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskDone      TaskStatus = "done"
	TaskError     TaskStatus = "error"
	TaskCancelled TaskStatus = "cancelled"
)

type Task struct {
	ID     int        `json:"id"`
	Name   string     `json:"name"`
	Status TaskStatus `json:"status"`
}

type InstallStatus string

const (
	StatusIdle         InstallStatus = "idle"
	StatusRunning      InstallStatus = "running"
	StatusSuccess      InstallStatus = "success"
	StatusCancelled    InstallStatus = "cancelled"
	StatusInstallError InstallStatus = "error"
)

type InstallError struct {
	Task   string `json:"task"`
	Host   string `json:"host"`
	Reason string `json:"reason"`
}

type InstallState struct {
	mu       sync.RWMutex
	Status   InstallStatus
	Progress int
	Tasks    []Task
	Error    *InstallError
	// Broadcaster is created once in NewInstallState and reused across operations.
	Broadcaster   *logs.LogBroadcaster
	cancelFn      func()
	InventoryPath string
	// AnsibleRuntime, when set, makes Ansible run through the bundled
	// Python+Ansible runtime instead of the host's system tools. nil falls
	// back to system ansible-playbook.
	AnsibleRuntime ansible.Runtime
	// parser derives intra-playbook sub-task progress from the Ansible log
	// stream so the progress bar moves continuously inside a long playbook.
	parser *progress.Parser
}

func NewInstallState() *InstallState {
	return &InstallState{
		Status:      StatusIdle,
		Broadcaster: logs.NewBroadcaster("install"),
		parser:      progress.NewParser(),
	}
}

func (s *InstallState) Snapshot() (InstallStatus, int, []Task, *InstallError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]Task, len(s.Tasks))
	copy(tasks, s.Tasks)

	var errCopy *InstallError
	if s.Error != nil {
		e := *s.Error
		errCopy = &e
	}
	return s.Status, s.Progress, tasks, errCopy
}

func (s *InstallState) SetRunning(tasks []Task, cancelFn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusRunning
	s.Progress = 0
	s.Tasks = tasks
	s.Error = nil
	s.cancelFn = cancelFn
	s.parser.Reset()
}

func (s *InstallState) UpdateTask(id int, status TaskStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Tasks {
		if s.Tasks[i].ID == id {
			s.Tasks[i].Status = status
			break
		}
	}
	// Each task is one playbook: reset the sub-task parser at playbook
	// boundaries so a finished playbook's fraction never fills the next
	// playbook's share.
	if status == TaskRunning || status == TaskDone {
		s.parser.Reset()
	}
	s.recalcProgress()
}

func (s *InstallState) SetDone() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusSuccess
	s.Progress = 100
	for i := range s.Tasks {
		if s.Tasks[i].Status != TaskDone {
			s.Tasks[i].Status = TaskDone
		}
	}
	s.parser.Reset()
}

func (s *InstallState) SetError(taskName, host, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusInstallError
	s.Error = &InstallError{Task: taskName, Host: host, Reason: reason}
	for i := range s.Tasks {
		if s.Tasks[i].Status == TaskRunning {
			s.Tasks[i].Status = TaskError
		}
	}
	s.parser.Reset()
}

func (s *InstallState) SetCancelled() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusCancelled
	s.Error = &InstallError{Task: "cancelled", Host: "", Reason: "Operation cancelled by user"}
	for i := range s.Tasks {
		if s.Tasks[i].Status == TaskRunning {
			s.Tasks[i].Status = TaskCancelled
		}
	}
	s.parser.Reset()
}

func (s *InstallState) Cancel() {
	s.mu.Lock()
	fn := s.cancelFn
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// recalcProgress must be called with the lock held.
//
// Progress blends playbook-level completion with intra-playbook sub-task
// progress parsed from the Ansible log stream: each done playbook is a full
// share, and the currently-running playbook fills part of its share based on
// how many of its Ansible tasks have completed. The result is clamped to be
// monotonic so the bar never regresses.
func (s *InstallState) recalcProgress() {
	if len(s.Tasks) == 0 {
		return
	}
	done := 0
	for _, t := range s.Tasks {
		if t.Status == TaskDone {
			done++
		}
	}
	base := done * 100 / len(s.Tasks)
	share := 100 / len(s.Tasks)
	blended := base + int(s.parser.Fraction()*float64(share))
	if blended < s.Progress {
		blended = s.Progress
	}
	s.Progress = blended
}

// StartLogging is a no-op kept for backward compatibility.
func (s *InstallState) StartLogging(_ string) error {
	return nil
}

func (s *InstallState) WriteLog(line string) {
	// Feed the parser and recompute progress so the bar moves as ansible
	// sub-task lines stream in from the output goroutine. The lock is held
	// across both so the status endpoint never reads a half-updated value.
	s.mu.Lock()
	s.parser.Feed(line)
	s.recalcProgress()
	s.mu.Unlock()
	if s.Broadcaster != nil {
		s.Broadcaster.Publish(line)
	}
}

// StopLogging is a no-op. Broadcaster manages its own lifecycle.
func (s *InstallState) StopLogging() {}

// OperationStates holds separate InstallState instances per flow so that
// state from one flow (e.g. delete-ydb success) never leaks into another
// (e.g. ydb install). Each instance has its own Broadcaster, task list,
// and status.
type OperationStates struct {
	Install   *InstallState
	InstallEM *InstallState
	DeleteYdb *InstallState
	DeleteEM  *InstallState
}

func NewOperationStates() *OperationStates {
	return &OperationStates{
		Install:   NewInstallState(),
		InstallEM: NewInstallState(),
		DeleteYdb: NewInstallState(),
		DeleteEM:  NewInstallState(),
	}
}

// ForFlow returns nil when flow is empty or unknown.
func (s *OperationStates) ForFlow(flow Flow) *InstallState {
	switch flow {
	case FlowYDB:
		return s.Install
	case FlowYDBEM:
		return s.InstallEM
	case FlowDeleteYDB:
		return s.DeleteYdb
	case FlowDeleteEMYDB:
		return s.DeleteEM
	default:
		return nil
	}
}
