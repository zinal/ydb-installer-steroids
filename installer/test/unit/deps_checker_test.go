package unit_test

import (
	"testing"

	"ydb-installer/internal/services/checker"
)

// TestMeetsMinVersion verifies the version comparison helper covers all cases.
func TestMeetsMinVersion(t *testing.T) {
	cases := []struct {
		version string
		min     string
		want    bool
	}{
		{"3.10.12", "3.8", true},
		{"3.8.0", "3.8", true},
		{"3.7.9", "3.8", false},
		{"2.15.3", "2.12", true},
		{"2.12.0", "2.12", true},
		{"2.11.9", "2.12", false},
		{"1.0.0", "1.0.0", true},
		{"2.0.0", "1.0.0", true},
		{"0.9.0", "1.0.0", false},
	}

	for _, tc := range cases {
		got := checker.MeetsMinVersion(tc.version, tc.min)
		if got != tc.want {
			t.Errorf("MeetsMinVersion(%q, %q) = %v, want %v", tc.version, tc.min, got, tc.want)
		}
	}
}

// TestExtractVersion verifies the version extraction regex.
func TestExtractVersion(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Python 3.10.12", "3.10.12"},
		{"ansible [core 2.15.3]", "2.15.3"},
		{"ydb_platform.ydb  2.3.0", "2.3.0"},
		{"no version here", ""},
		{"v1.2.3 something", "1.2.3"},
	}

	for _, tc := range cases {
		got := checker.ExtractVersion(tc.input)
		if got != tc.want {
			t.Errorf("ExtractVersion(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestDepsChecker_ReturnsThreeDeps verifies that system-mode Check() always
// returns three dependency entries in the correct order.
func TestDepsChecker_ReturnsThreeDeps(t *testing.T) {
	c := &checker.DepsChecker{}
	deps := c.Check()

	expectedNames := []string{"python", "ansible", "ydb_platform.ydb"}
	if len(deps) != len(expectedNames) {
		t.Fatalf("expected %d dependencies, got %d", len(expectedNames), len(deps))
	}

	for i, dep := range deps {
		if dep.Name != expectedNames[i] {
			t.Errorf("deps[%d].Name = %q, want %q", i, dep.Name, expectedNames[i])
		}
	}
}

// TestDepsChecker_FieldsPopulated verifies that each dependency has the
// required fields set regardless of whether it is installed.
func TestDepsChecker_FieldsPopulated(t *testing.T) {
	c := &checker.DepsChecker{}
	deps := c.Check()

	for _, dep := range deps {
		if dep.Name == "" {
			t.Errorf("dependency has empty Name")
		}
		if dep.Required == "" {
			t.Errorf("dependency %q has empty Required", dep.Name)
		}
		// If installed, version must be non-empty.
		if dep.Installed && dep.Version == "" {
			t.Errorf("dependency %q is installed but has empty Version", dep.Name)
		}
		// OK implies Installed.
		if dep.OK && !dep.Installed {
			t.Errorf("dependency %q has OK=true but Installed=false", dep.Name)
		}
	}
}
