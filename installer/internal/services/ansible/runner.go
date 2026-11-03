// Package ansible provides services for running Ansible playbooks.
package ansible

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"reflect"
	"sync"
)

// Runtime describes a bundled Python+Ansible runtime. When provided to a Runner,
// ansible-playbook is executed via the bundled interpreter and environment
// instead of relying on the host's system tools.
//
// internal/runtime.Paths satisfies this interface.
type Runtime interface {
	// PythonPath is the absolute path to the bundled python3 interpreter.
	PythonPath() string
	// AnsiblePlaybookPath is the absolute path to the ansible-playbook script.
	AnsiblePlaybookPath() string
	// ExtraEnv returns runtime-specific environment overrides to append to the
	// host environment (PATH, PYTHONHOME, ANSIBLE_COLLECTIONS_PATH, ...).
	ExtraEnv() []string
}

// Runner manages the execution of ansible-playbook processes.
type Runner struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	exitErr error // captured from cmd.Wait()
	runtime Runtime
}

// NewRunner creates a new AnsibleRunner instance that uses the host's
// system ansible-playbook.
func NewRunner() *Runner {
	return &Runner{}
}

// NewRunnerWithRuntime creates a Runner that executes ansible-playbook through
// the supplied bundled runtime. A nil runtime falls back to system tools.
//
// Uses reflect to guard against the Go "typed nil in interface" pitfall:
// when a nil concrete pointer (e.g. (*Paths)(nil)) is assigned to a Runtime
// interface variable, the interface is non-nil even though the underlying
// value is nil. This check detects both an untyped nil interface AND a nil
// pointer wrapped inside a non-nil interface.
func NewRunnerWithRuntime(rt Runtime) *Runner {
	if rt == nil {
		// Untyped nil interface.
		return &Runner{}
	}
	rv := reflect.ValueOf(rt)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		// Nil pointer wrapped in a non-nil interface (typed-nil pitfall).
		return &Runner{}
	}
	return &Runner{runtime: rt}
}

// Start begins execution of an Ansible playbook.
// It returns immediately after starting the process.
// The logCb function is called for each stdout/stderr line in real-time.
// The done channel is closed when the process completes.
// extraVars is an optional JSON string passed via --extra-vars; pass "" to skip.
func (r *Runner) Start(
	inventoryPath string,
	playbook string,
	logCb func(string),
	done chan<- struct{},
	extraVars ...string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	args := []string{"-i", inventoryPath, playbook}
	if len(extraVars) > 0 && extraVars[0] != "" {
		args = append(args, "--extra-vars", extraVars[0])
	}

	var cmd *exec.Cmd
	if r.runtime != nil {
		// Run via the bundled interpreter: "<python> <ansible-playbook> ...".
		// Invoking the script with the interpreter explicitly bypasses the
		// absolute shebang baked into the console script, keeping the runtime
		// relocatable.
		cmd = exec.CommandContext(ctx, r.runtime.PythonPath(),
			append([]string{r.runtime.AnsiblePlaybookPath()}, args...)...,
		)
		cmd.Env = append(os.Environ(), r.runtime.ExtraEnv()...)
	} else {
		cmd = exec.CommandContext(ctx, "ansible-playbook", args...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return err
	}

	r.cmd = cmd

	// Stream stdout and stderr to the log callback concurrently.
	go streamOutput(ctx, stdout, logCb)
	go streamOutput(ctx, stderr, logCb)

	// Wait for the process to complete in a goroutine.
	go func() {
		defer close(done)
		r.exitErr = cmd.Wait()
	}()

	return nil
}

// Cancel stops the running process by cancelling the context.
func (r *Runner) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		r.cancel()
	}
}

// WaitForCompletion returns nil if the process has already been waited on.
func (r *Runner) WaitForCompletion() error {
	r.mu.Lock()
	cmd := r.cmd
	r.mu.Unlock()

	if cmd == nil {
		return nil
	}

	return r.exitErr
}

// streamOutput reads from the reader and calls logCb for each line.
// It stops when the context is cancelled or the reader is closed.
func streamOutput(ctx context.Context, reader io.Reader, logCb func(string)) {
	scanner := bufio.NewScanner(reader)

	// Increase scanner buffer to handle long Ansible output lines
	const maxLineSize = 1024 * 256 // 256KB
	scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !scanner.Scan() {
				return
			}
			line := scanner.Text()

			// Call the callback (non-blocking is the caller's responsibility)
			logCb(line)
		}
	}
}
