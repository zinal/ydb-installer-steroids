package unit_test

import (
	"testing"

	"ydb-installer/internal/services/checker"
)

// TestDepsChecker_BundledMode verifies that, when a bundled runtime is set, the
// checker reports the dependencies as satisfied by the bundle without probing
// the host. EMCollection is empty here, so only the original three appear.
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

// TestDepsChecker_BundledReportsEMCollection verifies the EM collection is
// reported as a fourth bundled dependency when the runtime carries it.
func TestDepsChecker_BundledReportsEMCollection(t *testing.T) {
	c := &checker.DepsChecker{Bundled: &checker.BundledInfo{
		Python:       "3.12.5",
		Ansible:      "10.3.0",
		AnsibleCore:  "2.17.3",
		Collection:   "1.2.3",
		EMCollection: "1.0.13",
	}}

	deps := c.Check()
	if len(deps) != 4 {
		t.Fatalf("expected 4 deps when the EM collection is bundled, got %d", len(deps))
	}

	var em *checker.Dependency
	for i := range deps {
		if deps[i].Name == "ydb_platform.ydb_em" {
			em = &deps[i]
			break
		}
	}
	if em == nil {
		t.Fatal("ydb_platform.ydb_em should be reported when bundled")
	}
	if em.Version != "1.0.13" {
		t.Errorf("em collection version = %q, want 1.0.13", em.Version)
	}
	if !em.OK || !em.Installed {
		t.Error("bundled em collection should be OK and Installed")
	}
	if em.Source != checker.SourceBundled {
		t.Errorf("em collection source = %q, want %q", em.Source, checker.SourceBundled)
	}
}

// TestDepsChecker_BundledOmitsMissingEMCollection verifies that a runtime built
// before EM bundling keeps the original three-dependency shape rather than
// advertising a collection it does not ship.
func TestDepsChecker_BundledOmitsMissingEMCollection(t *testing.T) {
	c := &checker.DepsChecker{Bundled: &checker.BundledInfo{
		Python:       "3.12.5",
		Ansible:      "10.3.0",
		AnsibleCore:  "2.17.3",
		Collection:   "1.2.3",
		EMCollection: "",
	}}

	for _, d := range c.Check() {
		if d.Name == "ydb_platform.ydb_em" {
			t.Fatal("ydb_platform.ydb_em must not be reported when not bundled")
		}
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

// TestDepsChecker_SystemModeIncludesEMCollection verifies the non-bundled path
// probes four named deps including ydb_platform.ydb_em.
func TestDepsChecker_SystemModeIncludesEMCollection(t *testing.T) {
	c := &checker.DepsChecker{} // Bundled == nil
	deps := c.Check()
	want := []string{"python", "ansible", "ydb_platform.ydb", "ydb_platform.ydb_em"}
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
