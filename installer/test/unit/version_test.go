package unit_test

import (
	"strings"
	"testing"

	"ydb-installer/internal/version"
)

func TestVersion_Format(t *testing.T) {
	if version.Version == "" {
		t.Fatal("version.Version is empty")
	}
	if !strings.HasPrefix(version.Version, "v") {
		t.Errorf("version %q must start with 'v' (semver tag convention)", version.Version)
	}
	if strings.ContainsAny(version.Version, " \t") {
		t.Errorf("version %q must not contain whitespace", version.Version)
	}
}

func TestVersion_Current(t *testing.T) {
	const want = "v0.4.3"
	if version.Version != want {
		t.Errorf("version.Version = %q, want %q", version.Version, want)
	}
}
