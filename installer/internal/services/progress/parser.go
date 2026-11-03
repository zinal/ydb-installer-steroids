// Package progress derives install progress from the Ansible log stream.
//
// Ansible's default text output marks each task start with a "TASK [name]"
// line. The Parser counts these to estimate how far the currently-running
// playbook has progressed, so the install progress bar moves continuously
// instead of sitting at 0% until a whole playbook finishes.
//
// The fraction is derived from TASK lines only, NOT from ok:/changed:/failed:
// result lines. Result lines are emitted per-host (a single task across N
// hosts emits N result lines), so counting them would make the first task's
// completion look like the whole playbook is done. A new TASK line instead
// means the previous task finished, giving a smooth monotonic signal.
package progress

import (
	"regexp"
	"sync"
)

// taskStartRe matches Ansible's "TASK [name]" header line. The leading
// whitespace and ANSI color codes are stripped before matching.
var taskStartRe = regexp.MustCompile(`^TASK \[`)

// ansiRe strips ANSI escape sequences (color codes) that ansible-playbook
// emits when color output is enabled, so the marker above matches reliably.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Parser tracks intra-playbook sub-task progress from ansible log lines.
// It is thread-safe: WriteLog may be called from the ansible output stream
// goroutine while the status endpoint reads progress concurrently.
type Parser struct {
	mu      sync.Mutex
	started int // TASK lines seen
}

// NewParser returns a Parser with all counters zeroed.
func NewParser() *Parser {
	return &Parser{}
}

// Reset clears all counters. Called at the start of a new operation so a
// previous run's sub-task counts never leak into the next one.
func (p *Parser) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.started = 0
}

// Feed consumes one log line and returns the current intra-playbook fraction
// in [0,1]. The fraction is monotonic: it never decreases across calls.
func (p *Parser) Feed(line string) float64 {
	clean := ansiRe.ReplaceAllString(line, "")
	p.mu.Lock()
	defer p.mu.Unlock()

	if taskStartRe.MatchString(clean) {
		p.started++
	}
	return p.fractionLocked()
}

// Fraction returns the current intra-playbook fraction in [0,1].
func (p *Parser) Fraction() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.fractionLocked()
}

// fractionLocked must be called with the lock held.
//
// Each TASK line marks the start of a new task. The fraction is
// started/(started+9): it starts at 1/10 after the first task, then 2/11,
// 3/12, ... — a gentle asymptotic curve that never reaches 1.0 until the
// playbook's task is marked done by the caller. The +9 offset keeps the bar
// from jumping early while still moving on every task.
func (p *Parser) fractionLocked() float64 {
	if p.started == 0 {
		return 0
	}
	return float64(p.started) / float64(p.started+9)
}
