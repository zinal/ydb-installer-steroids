package unit_test

import (
	"strings"
	"testing"

	"ydb-installer/internal/services/checker"
)

func TestInstructionProvider_Python(t *testing.T) {
	p := &checker.InstructionProvider{}

	cases := []struct {
		os      string
		wantSub string
	}{
		{"ubuntu", "apt"},
		{"centos", "yum"},
		{"macos", "brew"},
	}

	for _, tc := range cases {
		got := p.ForDep("python", tc.os)
		if got == "" {
			t.Errorf("ForDep(python, %s) returned empty string", tc.os)
		}
		if !strings.Contains(got, tc.wantSub) {
			t.Errorf("ForDep(python, %s) = %q, want substring %q", tc.os, got, tc.wantSub)
		}
	}
}

func TestInstructionProvider_Ansible(t *testing.T) {
	p := &checker.InstructionProvider{}

	cases := []struct {
		os      string
		wantSub string
	}{
		{"ubuntu", "ansible"},
		{"centos", "ansible"},
		{"macos", "ansible"},
	}

	for _, tc := range cases {
		got := p.ForDep("ansible", tc.os)
		if got == "" {
			t.Errorf("ForDep(ansible, %s) returned empty string", tc.os)
		}
		if !strings.Contains(got, tc.wantSub) {
			t.Errorf("ForDep(ansible, %s) = %q, want substring %q", tc.os, got, tc.wantSub)
		}
	}
}

func TestInstructionProvider_Collection(t *testing.T) {
	p := &checker.InstructionProvider{}

	for _, osName := range []string{"ubuntu", "centos", "macos"} {
		got := p.ForDep("ydb_platform.ydb", osName)
		if got == "" {
			t.Errorf("ForDep(ydb_platform.ydb, %s) returned empty string", osName)
		}
		if !strings.Contains(got, "ansible-galaxy") {
			t.Errorf("ForDep(ydb_platform.ydb, %s) = %q, want ansible-galaxy", osName, got)
		}
		if !strings.Contains(got, "git+") {
			t.Errorf("ForDep(ydb_platform.ydb, %s) = %q, want git+ install URL", osName, got)
		}
	}
}

func TestInstructionProvider_EMCollection(t *testing.T) {
	p := &checker.InstructionProvider{}

	for _, osName := range []string{"ubuntu", "centos", "macos"} {
		got := p.ForDep("ydb_platform.ydb_em", osName)
		if got == "" {
			t.Errorf("ForDep(ydb_platform.ydb_em, %s) returned empty string", osName)
		}
		if !strings.Contains(got, "ansible-galaxy") {
			t.Errorf("ForDep(ydb_platform.ydb_em, %s) missing ansible-galaxy", osName)
		}
		if !strings.Contains(got, "ydb_em") {
			t.Errorf("ForDep(ydb_platform.ydb_em, %s) missing ydb_em", osName)
		}
		if !strings.Contains(got, "git+") {
			t.Errorf("ForDep(ydb_platform.ydb_em, %s) missing git+ URL", osName)
		}
		if !strings.Contains(got, "pyyaml") {
			t.Errorf("ForDep(ydb_platform.ydb_em, %s) missing pyyaml", osName)
		}
	}
}

func TestInstructionProvider_UnknownDep(t *testing.T) {
	p := &checker.InstructionProvider{}
	got := p.ForDep("unknown-tool", "ubuntu")
	if got != "" {
		t.Errorf("ForDep(unknown-tool, ubuntu) = %q, want empty string", got)
	}
}

func TestInstructionProvider_UnknownOS(t *testing.T) {
	p := &checker.InstructionProvider{}
	got := p.ForDep("ansible", "windows")
	if got != "" {
		t.Errorf("ForDep(ansible, windows) = %q, want empty string", got)
	}
}

func TestInstructionProvider_AllInstructions(t *testing.T) {
	p := &checker.InstructionProvider{}
	all := p.AllInstructions("ubuntu")

	expectedKeys := []string{"python", "ansible", "ydb_platform.ydb", "ydb_platform.ydb_em"}
	for _, key := range expectedKeys {
		if _, ok := all[key]; !ok {
			t.Errorf("AllInstructions(ubuntu) missing key %q", key)
		}
		if all[key] == "" {
			t.Errorf("AllInstructions(ubuntu)[%q] is empty", key)
		}
	}
}

func TestInstructionProvider_CaseInsensitive(t *testing.T) {
	p := &checker.InstructionProvider{}
	lower := p.ForDep("ansible", "ubuntu")
	upper := p.ForDep("ANSIBLE", "UBUNTU")
	if lower != upper {
		t.Errorf("ForDep should be case-insensitive: %q != %q", lower, upper)
	}
}
