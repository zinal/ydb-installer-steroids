package unit_test

import (
	"testing"

	"ydb-installer/internal/services/checker"
)

// TestDepsChecker_BundledMode verifies that, when a bundled runtime is set, the
// checker reports the dependencies as satisfied by the bundle without probing
// the host.
func TestDepsChecker_BundledMode(t *testing.T) {
	c := &checker.DepsChecker{Bundled: &checker.BundledInfo{
		Python:      "3.12.5",
		Ansible:     "10.3.0",
		AnsibleCore: "2.17.3",
		Collection:  "1.2.3",
	}}

	deps := c.Check()
	if len(deps) != 3 {
		t.Fatalf("expected 3 deps, got %d", len(deps))
	}

	for _, d := range deps {
		if !d.OK {
			t.Errorf("bundled dep %q should be OK", d.Name)
		}
		if !d.Installed {
			t.Errorf("bundled dep %q should be Installed", d.Name)
		}
		if d.Source != checker.SourceBundled {
			t.Errorf("dep %q source = %q, want %q", d.Name, d.Source, checker.SourceBundled)
		}
	}

	byName := map[string]checker.Dependency{}
	for _, d := range deps {
		byName[d.Name] = d
	}
	if byName["python"].Version != "3.12.5" {
		t.Errorf("python version = %q", byName["python"].Version)
	}
	if byName["ansible"].Version != "10.3.0" {
		t.Errorf("ansible version = %q", byName["ansible"].Version)
	}
	if byName["ydb_platform.ydb"].Version != "1.2.3" {
		t.Errorf("collection version = %q", byName["ydb_platform.ydb"].Version)
	}
}

// TestDepsChecker_BundledAnsibleFallsBackToCore verifies ansible version falls
// back to ansible-core when the community package version is empty.
func TestDepsChecker_BundledAnsibleFallback(t *testing.T) {
	c := &checker.DepsChecker{Bundled: &checker.BundledInfo{
		Python:      "3.12.5",
		Ansible:     "",
		AnsibleCore: "2.17.3",
		Collection:  "1.0.0",
	}}
	deps := c.Check()
	for _, d := range deps {
		if d.Name == "ansible" && d.Version != "2.17.3" {
			t.Errorf("ansible version fallback = %q, want 2.17.3", d.Version)
		}
	}
}

// TestDepsChecker_SystemModeNames verifies the non-bundled path probes python,
// ansible, and the ydb collection.
func TestDepsChecker_SystemModeNames(t *testing.T) {
	c := &checker.DepsChecker{} // Bundled == nil
	deps := c.Check()
	want := []string{"python", "ansible", "ydb_platform.ydb"}
	if len(deps) != len(want) {
		t.Fatalf("expected %d deps in system mode, got %d", len(want), len(deps))
	}
	for i, d := range deps {
		if d.Name != want[i] {
			t.Errorf("deps[%d] = %q, want %q", i, d.Name, want[i])
		}
		// System probes must set a source (system or missing).
		if d.Source != checker.SourceSystem && d.Source != checker.SourceMissing {
			t.Errorf("dep %q has unexpected source %q", d.Name, d.Source)
		}
	}
}
