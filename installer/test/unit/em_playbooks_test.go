package unit_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ydb-installer/internal/api/handlers"
)

// TestEmInstall_PlaybookFQCNs pins the fully-qualified collection names the EM
// install flow invokes.
//
// These must stay in lockstep with the collection bundled by
// scripts/build-runtime.sh (ydb_platform.ydb_em, whose playbooks/ directory
// contains initial_setup, install_agent and register_agents). If the collection
// namespace or a playbook is ever renamed, this test fails loudly instead of the
// installer failing at runtime with an unresolvable playbook.
func TestEmInstall_PlaybookFQCNs(t *testing.T) {
	content := `all:
  children:
    ydb_em:
      hosts:
        em1.example.com:
`
	path := writeTempInventory(t, content)
	defer os.RemoveAll(filepath.Dir(path))

	states := newStates()
	states.InstallEM.InventoryPath = path

	rec := httptest.NewRecorder()
	handlers.EmInstallStart(states).ServeHTTP(
		rec, httptest.NewRequest(http.MethodPost, "/api/install/ydb-em/start", nil))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	_, _, tasks, _ := states.InstallEM.Snapshot()

	// The task labels embed the FQCN, so they are the observable surface of the
	// playbook sequence from outside the handlers package.
	want := []string{
		"ydb_platform.ydb_em.initial_setup",
		"ydb_platform.ydb_em.install_agent",
		"ydb_platform.ydb_em.register_agents",
	}
	if len(tasks) != len(want) {
		t.Fatalf("expected %d tasks, got %d", len(want), len(tasks))
	}
	for i, fqcn := range want {
		if !strings.Contains(tasks[i].Name, fqcn) {
			t.Errorf("task %d = %q, want it to reference %q", i, tasks[i].Name, fqcn)
		}
	}
}

// TestEmDelete_PlaybookFQCNs pins the fully-qualified collection names the EM
// delete flow invokes (unregister_agents → uninstall). Must not touch the YDB
// collection or format_cluster.
func TestEmDelete_PlaybookFQCNs(t *testing.T) {
	content := `all:
  children:
    ydb_em:
      hosts:
        em1.example.com:
`
	path := writeTempInventory(t, content)
	defer os.RemoveAll(filepath.Dir(path))

	states := newStates()
	states.DeleteEM.InventoryPath = path

	rec := httptest.NewRecorder()
	handlers.DeleteEmYdbStart(states).ServeHTTP(
		rec, httptest.NewRequest(http.MethodPost, "/api/install/delete-em-ydb/start", nil))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	_, _, tasks, _ := states.DeleteEM.Snapshot()

	want := []string{
		"ydb_platform.ydb_em.unregister_agents",
		"ydb_platform.ydb_em.uninstall",
	}
	if len(tasks) != len(want) {
		t.Fatalf("expected %d tasks, got %d", len(want), len(tasks))
	}
	for i, fqcn := range want {
		if !strings.Contains(tasks[i].Name, fqcn) {
			t.Errorf("task %d = %q, want it to reference %q", i, tasks[i].Name, fqcn)
		}
		if strings.Contains(tasks[i].Name, "ydb_platform.ydb.") {
			t.Errorf("task %d = %q must not reference ydb_platform.ydb", i, tasks[i].Name)
		}
		if strings.Contains(tasks[i].Name, "format_cluster") {
			t.Errorf("task %d = %q must not reference format_cluster", i, tasks[i].Name)
		}
	}
}

// TestEmInstall_UsesEmCollectionNamespace guards against accidentally pointing
// the EM flow at the ydb_platform.ydb collection, which does not contain the EM
// playbooks.
func TestEmInstall_UsesEmCollectionNamespace(t *testing.T) {
	content := `all:
  children:
    ydb_em:
      hosts:
        em1.example.com:
`
	path := writeTempInventory(t, content)
	defer os.RemoveAll(filepath.Dir(path))

	states := newStates()
	states.InstallEM.InventoryPath = path

	rec := httptest.NewRecorder()
	handlers.EmInstallStart(states).ServeHTTP(
		rec, httptest.NewRequest(http.MethodPost, "/api/install/ydb-em/start", nil))

	_, _, tasks, _ := states.InstallEM.Snapshot()
	for _, task := range tasks {
		if !strings.Contains(task.Name, "ydb_platform.ydb_em.") {
			t.Errorf("EM task %q must use the ydb_platform.ydb_em collection", task.Name)
		}
	}
}
