package ansible

import (
	"testing"

	"ydb-installer/internal/runtime"
)

func TestNewRunnerWithRuntime_NilInterface(t *testing.T) {
	r := NewRunnerWithRuntime(nil)
	if r == nil {
		t.Fatal("NewRunnerWithRuntime returned nil")
	}
	if r.runtime != nil {
		t.Fatal("expected nil runtime when nil interface is passed")
	}
}

// TestNewRunnerWithRuntime_NilTypedPointer reproduces the "typed nil in interface"
// Go pitfall: a nil *runtime.Paths assigned to a Runtime interface variable
// creates a non-nil interface wrapping a nil pointer. NewRunnerWithRuntime must
// detect this via reflection and treat it as nil (system fallback).
func TestNewRunnerWithRuntime_NilTypedPointer(t *testing.T) {
	var p *runtime.Paths = nil
	var rt Runtime = p

	if rt == nil {
		t.Fatal("test precondition failed: expected non-nil interface wrapping nil pointer")
	}

	r := NewRunnerWithRuntime(rt)
	if r == nil {
		t.Fatal("NewRunnerWithRuntime returned nil")
	}
	if r.runtime != nil {
		t.Fatal("expected nil runtime when typed-nil pointer is passed")
	}
}

func TestNewRunner_NoRuntime(t *testing.T) {
	r := NewRunner()
	if r == nil {
		t.Fatal("NewRunner returned nil")
	}
	if r.runtime != nil {
		t.Fatal("expected nil runtime from NewRunner")
	}
}

func TestNewRunnerWithRuntime_NonNullWrapsCorrectly(t *testing.T) {
	p := &runtime.Paths{
		Python: "/tmp/fake-root/bin/python3",
	}
	var rt Runtime = p

	if rt == nil {
		t.Fatal("test precondition failed: expected non-nil interface")
	}

	r := NewRunnerWithRuntime(rt)
	if r == nil {
		t.Fatal("NewRunnerWithRuntime returned nil")
	}
	if r.runtime == nil {
		t.Fatal("expected non-nil runtime when valid Paths is passed")
	}

	pyPath := r.runtime.PythonPath()
	if pyPath == "" {
		t.Errorf("PythonPath returned empty string")
	}
}
