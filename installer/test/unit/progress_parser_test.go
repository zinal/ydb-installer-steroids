package unit_test

import (
	"testing"

	"ydb-installer/internal/services/progress"
)

// TestParser_CountsTaskStart verifies a "TASK [" line increments the started count.
func TestParser_CountsTaskStart(t *testing.T) {
	p := progress.NewParser()
	p.Feed("TASK [Check python version]")
	if got := p.Fraction(); got != 1.0/10.0 {
		t.Errorf("fraction after first start = %v, want %v", got, 1.0/10.0)
	}
}

// TestParser_FractionAdvancesWithTasks verifies each new TASK line advances
// the fraction along the started/(started+9) curve: 1/10, 2/11, 3/12, ...
func TestParser_FractionAdvancesWithTasks(t *testing.T) {
	p := progress.NewParser()
	p.Feed("TASK [Task 1]")
	if got := p.Fraction(); got != 1.0/10.0 {
		t.Errorf("fraction after 1 task = %v, want %v", got, 1.0/10.0)
	}
	p.Feed("TASK [Task 2]")
	if got := p.Fraction(); got != 2.0/11.0 {
		t.Errorf("fraction after 2 tasks = %v, want %v", got, 2.0/11.0)
	}
	p.Feed("TASK [Task 3]")
	if got := p.Fraction(); got != 3.0/12.0 {
		t.Errorf("fraction after 3 tasks = %v, want %v", got, 3.0/12.0)
	}
}

// TestParser_IgnoresResultLines verifies ok:/changed:/failed: lines do NOT
// advance the fraction (they are per-host noise, not task completions).
func TestParser_IgnoresResultLines(t *testing.T) {
	p := progress.NewParser()
	p.Feed("ok: [host1]")
	p.Feed("ok: [host2]")
	p.Feed("changed: [host1]")
	if got := p.Fraction(); got != 0 {
		t.Errorf("fraction after result lines = %v, want 0", got)
	}
}

// TestParser_FractionZeroWhenNoTasks verifies an empty stream yields 0.
func TestParser_FractionZeroWhenNoTasks(t *testing.T) {
	p := progress.NewParser()
	if got := p.Fraction(); got != 0 {
		t.Errorf("fraction = %v, want 0", got)
	}
}

// TestParser_Reset verifies Reset clears all counters.
func TestParser_Reset(t *testing.T) {
	p := progress.NewParser()
	p.Feed("TASK [A]")
	p.Feed("TASK [B]")
	p.Reset()
	if got := p.Fraction(); got != 0 {
		t.Errorf("fraction after reset = %v, want 0", got)
	}
}

// TestParser_StripsAnsi verifies ANSI color codes do not break marker matching.
func TestParser_StripsAnsi(t *testing.T) {
	p := progress.NewParser()
	p.Feed("\x1b[32mTASK [A]\x1b[0m")
	p.Feed("\x1b[32mTASK [B]\x1b[0m")
	if got := p.Fraction(); got != 2.0/11.0 {
		t.Errorf("fraction = %v, want %v", got, 2.0/11.0)
	}
}

// TestParser_IgnoresNonMarkers verifies unrelated log lines are ignored.
func TestParser_IgnoresNonMarkers(t *testing.T) {
	p := progress.NewParser()
	p.Feed("PLAYBOOK: ydb_platform.ydb.initial_setup")
	p.Feed("ok: not a task line")
	if got := p.Fraction(); got != 0 {
		t.Errorf("fraction = %v, want 0", got)
	}
}
