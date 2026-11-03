package unit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ydb-installer/internal/api/handlers"
	"ydb-installer/internal/models"
	"ydb-installer/internal/services/config"
	"ydb-installer/internal/services/session"
)

// newStates returns fresh OperationStates for tests.
func newStates() *models.OperationStates {
	return models.NewOperationStates()
}

// newStatesWithFlow returns states + a store with the given flow already set.
func newStatesWithFlow(flow models.Flow) (*models.OperationStates, *session.Store) {
	states := models.NewOperationStates()
	store := session.New("en")
	store.Update(models.SessionUpdate{Flow: &flow})
	return states, store
}

// TestInstallStatus_Idle verifies the status endpoint returns "idle" on a fresh state.
func TestInstallStatus_Idle(t *testing.T) {
	states, store := newStatesWithFlow(models.FlowYDB)
	h := handlers.InstallStatus(states, store)

	req := httptest.NewRequest(http.MethodGet, "/api/install/status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if body["status"] != string(models.StatusIdle) {
		t.Errorf("status = %v, want %q", body["status"], models.StatusIdle)
	}
	if body["progress"].(float64) != 0 {
		t.Errorf("progress = %v, want 0", body["progress"])
	}
}

// TestInstallStatus_MethodNotAllowed verifies POST is rejected.
func TestInstallStatus_MethodNotAllowed(t *testing.T) {
	states, store := newStatesWithFlow(models.FlowYDB)
	h := handlers.InstallStatus(states, store)

	req := httptest.NewRequest(http.MethodPost, "/api/install/status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

// TestDeleteYdbStart_ReturnsAccepted verifies the start endpoint returns 202.
func TestDeleteYdbStart_ReturnsAccepted(t *testing.T) {
	states := newStates()
	states.DeleteYdb.InventoryPath = "/tmp/test-inventory.yaml"
	h := handlers.DeleteYdbStart(states)

	req := httptest.NewRequest(http.MethodPost, "/api/install/delete-ydb/start", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["status"] != "started" {
		t.Errorf("body.status = %q, want %q", body["status"], "started")
	}
}

// TestDeleteYdbStart_MethodNotAllowed verifies GET is rejected.
func TestDeleteYdbStart_MethodNotAllowed(t *testing.T) {
	states := newStates()
	h := handlers.DeleteYdbStart(states)

	req := httptest.NewRequest(http.MethodGet, "/api/install/delete-ydb/start", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

// TestDeleteYdbStart_SetsRunningState verifies the state transitions to running.
func TestDeleteYdbStart_SetsRunningState(t *testing.T) {
	states := newStates()
	states.DeleteYdb.InventoryPath = "/tmp/test-inventory.yaml"
	h := handlers.DeleteYdbStart(states)

	req := httptest.NewRequest(http.MethodPost, "/api/install/delete-ydb/start", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	// Give the goroutine a moment to start.
	time.Sleep(10 * time.Millisecond)

	status, _, tasks, _ := states.DeleteYdb.Snapshot()
	if status != models.StatusRunning {
		t.Errorf("state.Status = %q, want %q", status, models.StatusRunning)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task for delete-ydb, got %d", len(tasks))
	}
}

// TestDeleteYdbStart_AlreadyRunning verifies second start returns 409.
func TestDeleteYdbStart_AlreadyRunning(t *testing.T) {
	states := newStates()
	states.DeleteYdb.InventoryPath = "/tmp/test-inventory.yaml"

	h := handlers.DeleteYdbStart(states)
	req := httptest.NewRequest(http.MethodPost, "/api/install/delete-ydb/start", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("first start: expected 202, got %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/install/delete-ydb/start", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Errorf("second start: expected 409, got %d", rec2.Code)
	}
}

// TestInstallCancel_NoActiveJob verifies cancel returns 200 even with no active job.
func TestInstallCancel_NoActiveJob(t *testing.T) {
	states, store := newStatesWithFlow(models.FlowYDB)
	h := handlers.InstallCancel(states, store)

	req := httptest.NewRequest(http.MethodPost, "/api/install/cancel", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// TestInstallCancel_MethodNotAllowed verifies GET is rejected.
func TestInstallCancel_MethodNotAllowed(t *testing.T) {
	states, store := newStatesWithFlow(models.FlowYDB)
	h := handlers.InstallCancel(states, store)

	req := httptest.NewRequest(http.MethodGet, "/api/install/cancel", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

// TestDepsInstruction_ReturnsCommands verifies the instruction endpoint.
func TestDepsInstruction_ReturnsCommands(t *testing.T) {
	h := handlers.DepsInstruction()

	req := httptest.NewRequest(http.MethodGet, "/api/system/deps/instruction/ubuntu?dep=ansible", nil)
	// Simulate the path that the mux would set.
	req.URL.Path = "/api/system/deps/instruction/ubuntu"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["commands"] == "" {
		t.Error("commands field is empty")
	}
	if body["dep"] != "ansible" {
		t.Errorf("dep = %q, want %q", body["dep"], "ansible")
	}
	if body["os"] != "ubuntu" {
		t.Errorf("os = %q, want %q", body["os"], "ubuntu")
	}
}

// TestDepsInstruction_MissingDep verifies 400 when dep param is absent.
func TestDepsInstruction_MissingDep(t *testing.T) {
	h := handlers.DepsInstruction()

	req := httptest.NewRequest(http.MethodGet, "/api/system/deps/instruction/ubuntu", nil)
	req.URL.Path = "/api/system/deps/instruction/ubuntu"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// TestInstallState_ProgressCalculation verifies progress is computed correctly.
func TestInstallState_ProgressCalculation(t *testing.T) {
	states := newStates()
	state := states.Install
	tasks := []models.Task{
		{ID: 1, Name: "Task 1", Status: models.TaskPending},
		{ID: 2, Name: "Task 2", Status: models.TaskPending},
		{ID: 3, Name: "Task 3", Status: models.TaskPending},
		{ID: 4, Name: "Task 4", Status: models.TaskPending},
	}

	cancelCh := make(chan struct{})
	state.SetRunning(tasks, func() { close(cancelCh) })

	state.UpdateTask(1, models.TaskDone)
	state.UpdateTask(2, models.TaskDone)

	_, progress, _, _ := state.Snapshot()
	if progress != 50 {
		t.Errorf("progress = %d, want 50", progress)
	}

	state.UpdateTask(3, models.TaskDone)
	state.UpdateTask(4, models.TaskDone)
	state.SetDone()

	_, progress, _, _ = state.Snapshot()
	if progress != 100 {
		t.Errorf("progress after SetDone = %d, want 100", progress)
	}
}

// TestInstallState_HybridProgress verifies the bar moves inside a running
// playbook as ansible TASK lines stream in, instead of sitting at 0%.
func TestInstallState_HybridProgress(t *testing.T) {
	states := newStates()
	state := states.Install
	tasks := []models.Task{
		{ID: 1, Name: "Running ydb_platform.ydb.initial_setup", Status: models.TaskPending},
	}
	cancelCh := make(chan struct{})
	state.SetRunning(tasks, func() { close(cancelCh) })

	// Playbook starts; no sub-tasks yet → still 0%.
	state.UpdateTask(1, models.TaskRunning)
	_, progress, _, _ := state.Snapshot()
	if progress != 0 {
		t.Errorf("progress at playbook start = %d, want 0", progress)
	}

	// A second TASK line means the first ansible task finished → bar moves.
	state.WriteLog("TASK [Check python version]")
	state.WriteLog("TASK [Check config]")
	_, progress, _, _ = state.Snapshot()
	if progress <= 0 {
		t.Errorf("progress after first sub-task = %d, want > 0", progress)
	}

	// Playbook completes → 100%.
	state.UpdateTask(1, models.TaskDone)
	state.SetDone()
	_, progress, _, _ = state.Snapshot()
	if progress != 100 {
		t.Errorf("progress after SetDone = %d, want 100", progress)
	}
}

// TestInstallState_ProgressMonotonic verifies progress never regresses across
// WriteLog and UpdateTask calls.
func TestInstallState_ProgressMonotonic(t *testing.T) {
	state := newStates().Install
	tasks := []models.Task{
		{ID: 1, Name: "Task 1", Status: models.TaskPending},
		{ID: 2, Name: "Task 2", Status: models.TaskPending},
	}
	cancelCh := make(chan struct{})
	state.SetRunning(tasks, func() { close(cancelCh) })

	state.UpdateTask(1, models.TaskRunning)
	state.WriteLog("TASK [A]")
	state.WriteLog("ok: [h1]")
	_, p1, _, _ := state.Snapshot()

	// A later playbook boundary resets the parser; the monotonic clamp must
	// keep progress from dropping below the previous value.
	state.UpdateTask(1, models.TaskDone)
	state.UpdateTask(2, models.TaskRunning)
	_, p2, _, _ := state.Snapshot()

	if p2 < p1 {
		t.Errorf("progress regressed: %d -> %d", p1, p2)
	}
}

// writeTempInventory writes content to a temp file and returns its path.
func writeTempInventory(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp inventory: %v", err)
	}
	return path
}

// TestCreateDatabase_AllowsSameDbname verifies that CreateDatabase
// accepts a call for a dbname that already has a dynamic node
// (deduplication check was removed — the caller manages uniqueness).
func TestCreateDatabase_AllowsSameDbname(t *testing.T) {
	content := `all:
  children:
    ydb:
      hosts:
        node1.example.com:
      vars:
        ydb_dynnodes:
          - instance: a
            offset: 1
          - instance: b
            offset: 2
            dbname: database2
`
	path := writeTempInventory(t, content)
	defer os.RemoveAll(filepath.Dir(path))

	states := newStates()
	states.Install.InventoryPath = path

	body := `{"name":"database2","groups":4}`
	req := httptest.NewRequest(http.MethodPost, "/api/install/ydb/create-database", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h := handlers.CreateDatabase(states)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", rec.Code)
	}
}

// TestCreateDatabase_AllowsNewDbname verifies that CreateDatabase proceeds
// normally when no existing node has the same dbname.
func TestCreateDatabase_AllowsNewDbname(t *testing.T) {
	content := `all:
  children:
    ydb:
      hosts:
        node1.example.com:
      vars:
        ydb_dynnodes:
          - instance: a
            offset: 1
          - instance: b
            offset: 2
            dbname: database2
`
	path := writeTempInventory(t, content)
	defer os.RemoveAll(filepath.Dir(path))

	states := newStates()
	states.Install.InventoryPath = path

	body := `{"name":"database3","groups":4}`
	req := httptest.NewRequest(http.MethodPost, "/api/install/ydb/create-database", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h := handlers.CreateDatabase(states)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", rec.Code)
	}

	// Verify inventory is unchanged (prepareDynamicNode no longer patches).
	// ydb_dynnodes is managed by buildVars during UI sync, not by installation.
	invService := config.NewInventoryService()
	invData, err := invService.Parse(path)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(invData.DynamicNodes) != 2 {
		t.Errorf("expected 2 dynamic nodes (unchanged), got %d", len(invData.DynamicNodes))
	}
}

// TestYdbStart_SingleDb_TaskCount verifies that with 1 DB entry,
// the task list has exactly 1 task (initial_setup only).
func TestYdbStart_SingleDb_TaskCount(t *testing.T) {
	content := `all:
  children:
    ydb:
      hosts:
        node1.example.com:
`
	path := writeTempInventory(t, content)
	defer os.RemoveAll(filepath.Dir(path))

	states := newStates()
	states.Install.InventoryPath = path

	store := session.New("en")
	store.SetInventory(&models.InventoryData{
		Database: &models.DatabaseConfig{
			ConfigVariant: "single",
			Entries: []models.DatabaseEntry{
				{Name: "database", Groups: 8},
			},
		},
	}, true, false)

	h := handlers.YdbStart(states, store)
	req := httptest.NewRequest(http.MethodPost, "/api/install/ydb/start", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	status, _, tasks, _ := states.Install.Snapshot()
	if status != models.StatusRunning {
		t.Fatalf("expected running status, got %s", status)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Name != "Running ydb_platform.ydb.initial_setup" {
		t.Errorf("task name = %q", tasks[0].Name)
	}
}

// TestYdbStart_MultiDb_TaskCount verifies that with 3 DB entries,
// the task list has 3 tasks with correct names.
func TestYdbStart_MultiDb_TaskCount(t *testing.T) {
	content := `all:
  children:
    ydb:
      hosts:
        node1.example.com:
`
	path := writeTempInventory(t, content)
	defer os.RemoveAll(filepath.Dir(path))

	states := newStates()
	states.Install.InventoryPath = path

	store := session.New("en")
	store.SetInventory(&models.InventoryData{
		Database: &models.DatabaseConfig{
			ConfigVariant: "multiple",
			Entries: []models.DatabaseEntry{
				{Name: "sales_db", Groups: 8},
				{Name: "analytics_db", Groups: 4},
				{Name: "test_db", Groups: 2},
			},
		},
	}, true, false)

	h := handlers.YdbStart(states, store)
	req := httptest.NewRequest(http.MethodPost, "/api/install/ydb/start", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	status, _, tasks, _ := states.Install.Snapshot()
	if status != models.StatusRunning {
		t.Fatalf("expected running status, got %s", status)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].Name != "Running ydb_platform.ydb.initial_setup" {
		t.Errorf("task 0 name = %q", tasks[0].Name)
	}
	if tasks[1].Name != `Creating database "analytics_db"` {
		t.Errorf("task 1 name = %q", tasks[1].Name)
	}
	if tasks[2].Name != `Creating database "test_db"` {
		t.Errorf("task 2 name = %q", tasks[2].Name)
	}
}

// TestPrepareDynamicNode_CalculatesInstance verifies that instance/offset
// are computed correctly from existing nodes.
func TestPrepareDynamicNode_CalculatesInstance(t *testing.T) {
	content := `all:
  children:
    ydb:
      hosts:
        node1.example.com:
      vars:
        ydb_dynnodes:
          - instance: a
            offset: 1
          - instance: b
            offset: 2
            dbname: analytics_db
`
	path := writeTempInventory(t, content)
	defer os.RemoveAll(filepath.Dir(path))

	states := newStates()
	states.Install.InventoryPath = path

	body := `{"name":"test_db","groups":4}`
	req := httptest.NewRequest(http.MethodPost, "/api/install/ydb/create-database", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h := handlers.CreateDatabase(states)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	// Verify inventory is unchanged (prepareDynamicNode no longer patches).
	// ydb_dynnodes is managed by buildVars during UI sync.
	invService := config.NewInventoryService()
	invData, err := invService.Parse(path)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(invData.DynamicNodes) != 2 {
		t.Fatalf("expected 2 dynamic nodes (unchanged), got %d", len(invData.DynamicNodes))
	}
	if invData.DynamicNodes[0].Instance != "a" || invData.DynamicNodes[0].Offset != 1 {
		t.Errorf("expected first node {a, 1}, got {%s, %d}",
			invData.DynamicNodes[0].Instance, invData.DynamicNodes[0].Offset)
	}
	if invData.DynamicNodes[1].Dbname != "analytics_db" {
		t.Errorf("expected second node dbname analytics_db, got %q", invData.DynamicNodes[1].Dbname)
	}
}

// TestPrepareDynamicNode_ReusesPreconfiguredInstance verifies that
// when a dbname already exists in ydb_dynnodes (user pre-configured it),
// prepareDynamicNode reuses the existing instance/offset and does NOT
// add a duplicate entry to the inventory.
func TestPrepareDynamicNode_ReusesPreconfiguredInstance(t *testing.T) {
	content := "all:\n" +
		"  children:\n" +
		"    ydb:\n" +
		"      hosts:\n" +
		"        node1.example.com:\n" +
		"      vars:\n" +
		"        ydb_dynnodes:\n" +
		"          - instance: a\n" +
		"            offset: 1\n" +
		"          - instance: b\n" +
		"            offset: 2\n" +
		"            dbname: database2\n" +
		"          - instance: c\n" +
		"            offset: 3\n" +
		"            dbname: database3\n"
	path := writeTempInventory(t, content)
	defer os.RemoveAll(filepath.Dir(path))

	states := newStates()
	states.Install.InventoryPath = path

	body := `{"name":"database2","groups":8}`
	req := httptest.NewRequest(http.MethodPost, "/api/install/ydb/create-database", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h := handlers.CreateDatabase(states)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d: %s", rec.Code, rec.Body.String())
	}

	invService := config.NewInventoryService()
	invData, err := invService.Parse(path)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(invData.DynamicNodes) != 3 {
		t.Fatalf("expected 3 dynamic nodes (unchanged), got %d", len(invData.DynamicNodes))
	}

	db2 := invData.DynamicNodes[1]
	if db2.Instance != "b" || db2.Offset != 2 {
		t.Errorf("expected database2 node {b, 2}, got {%s, %d}", db2.Instance, db2.Offset)
	}
	if db2.Dbname != "database2" {
		t.Errorf("expected dbname database2, got %q", db2.Dbname)
	}
}

// TestPrepareDynamicNode_AddsNewDbEntry verifies that when a dbname
// does NOT already exist in ydb_dynnodes, prepareDynamicNode creates a
// new node and patches the inventory.
func TestPrepareDynamicNode_AddsNewDbEntry(t *testing.T) {
	content := "all:\n" +
		"  children:\n" +
		"    ydb:\n" +
		"      hosts:\n" +
		"        node1.example.com:\n" +
		"      vars:\n" +
		"        ydb_dynnodes:\n" +
		"          - instance: a\n" +
		"            offset: 1\n" +
		"          - instance: b\n" +
		"            offset: 2\n" +
		"            dbname: database2\n"
	path := writeTempInventory(t, content)
	defer os.RemoveAll(filepath.Dir(path))

	states := newStates()
	states.Install.InventoryPath = path

	body := `{"name":"database3","groups":4}`
	req := httptest.NewRequest(http.MethodPost, "/api/install/ydb/create-database", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h := handlers.CreateDatabase(states)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", rec.Code)
	}

	// Verify inventory is unchanged (prepareDynamicNode no longer patches).
	// ydb_dynnodes is managed by buildVars during UI sync.
	invService := config.NewInventoryService()
	invData, err := invService.Parse(path)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(invData.DynamicNodes) != 2 {
		t.Fatalf("expected 2 dynamic nodes (unchanged), got %d", len(invData.DynamicNodes))
	}

	if invData.DynamicNodes[1].Dbname != "database2" {
		t.Errorf("expected second node dbname database2, got %q", invData.DynamicNodes[1].Dbname)
	}
}

// TestInstallStatus_FlowAware verifies that InstallStatus returns the
// correct state for each flow (install vs delete).
func TestInstallStatus_FlowAware(t *testing.T) {
	states := newStates()

	states.Install.SetRunning([]models.Task{
		{ID: 1, Name: "install-task", Status: models.TaskRunning},
	}, func() {})

	states.DeleteYdb.SetRunning([]models.Task{
		{ID: 1, Name: "delete-task", Status: models.TaskDone},
	}, func() {})
	states.DeleteYdb.SetDone()

	storeInstall := session.New("en")
	flowYdb := models.FlowYDB
	storeInstall.Update(models.SessionUpdate{Flow: &flowYdb})

	h := handlers.InstallStatus(states, storeInstall)
	req := httptest.NewRequest(http.MethodGet, "/api/install/status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != string(models.StatusRunning) {
		t.Errorf("install flow: expected running, got %v", body["status"])
	}

	storeDelete := session.New("en")
	flowDelete := models.FlowDeleteYDB
	storeDelete.Update(models.SessionUpdate{Flow: &flowDelete})

	h2 := handlers.InstallStatus(states, storeDelete)
	req2 := httptest.NewRequest(http.MethodGet, "/api/install/status", nil)
	rec2 := httptest.NewRecorder()
	h2.ServeHTTP(rec2, req2)

	var body2 map[string]any
	json.NewDecoder(rec2.Body).Decode(&body2)
	if body2["status"] != string(models.StatusSuccess) {
		t.Errorf("delete-ydb flow: expected success, got %v", body2["status"])
	}

	// Test 3: The ydb-em flow has its own state and must stay idle even though
	// both the install and delete states are non-idle. Before InstallEM existed
	// ForFlow returned nil here, which made the EM screens permanently idle.
	storeEM := session.New("en")
	flowEM := models.FlowYDBEM
	storeEM.Update(models.SessionUpdate{Flow: &flowEM})

	h3 := handlers.InstallStatus(states, storeEM)
	req3 := httptest.NewRequest(http.MethodGet, "/api/install/status", nil)
	rec3 := httptest.NewRecorder()
	h3.ServeHTTP(rec3, req3)

	var body3 map[string]any
	json.NewDecoder(rec3.Body).Decode(&body3)
	if body3["status"] != string(models.StatusIdle) {
		t.Errorf("ydb-em flow: expected idle, got %v", body3["status"])
	}
}

// TestInstallStatus_EmFlowIsolated verifies an EM install is visible to the
// ydb-em flow and does NOT leak into the ydb flow.
func TestInstallStatus_EmFlowIsolated(t *testing.T) {
	states := newStates()

	states.InstallEM.SetRunning([]models.Task{
		{ID: 1, Name: "em-task", Status: models.TaskRunning},
	}, func() {})

	storeEM := session.New("en")
	flowEM := models.FlowYDBEM
	storeEM.Update(models.SessionUpdate{Flow: &flowEM})

	rec := httptest.NewRecorder()
	handlers.InstallStatus(states, storeEM).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, "/api/install/status", nil))

	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != string(models.StatusRunning) {
		t.Errorf("ydb-em flow: expected running, got %v", body["status"])
	}

	// The ydb flow must remain idle — no cross-flow leakage.
	storeYdb := session.New("en")
	flowYdb := models.FlowYDB
	storeYdb.Update(models.SessionUpdate{Flow: &flowYdb})

	rec2 := httptest.NewRecorder()
	handlers.InstallStatus(states, storeYdb).ServeHTTP(
		rec2, httptest.NewRequest(http.MethodGet, "/api/install/status", nil))

	var body2 map[string]any
	json.NewDecoder(rec2.Body).Decode(&body2)
	if body2["status"] != string(models.StatusIdle) {
		t.Errorf("ydb flow should stay idle, got %v", body2["status"])
	}
}

// TestGetLogs_EmFlow verifies GetLogs reads the InstallEM broadcaster for the
// ydb-em flow and does not return the install flow's logs.
func TestGetLogs_EmFlow(t *testing.T) {
	states := newStates()
	states.InstallEM.WriteLog("[install-em] em line")
	states.Install.WriteLog("[install] ydb line")

	storeEM := session.New("en")
	flowEM := models.FlowYDBEM
	storeEM.Update(models.SessionUpdate{Flow: &flowEM})

	rec := httptest.NewRecorder()
	handlers.GetLogs(states, storeEM).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, "/api/install/logs", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body struct {
		Logs []string `json:"logs"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	joined := strings.Join(body.Logs, "\n")
	if !strings.Contains(joined, "em line") {
		t.Errorf("expected EM log line, got %q", joined)
	}
	if strings.Contains(joined, "ydb line") {
		t.Errorf("EM logs must not contain install-flow lines, got %q", joined)
	}
}

// TestEmInstallStart_BuildsTaskSequence verifies the EM start handler responds
// 202 and seeds the task list from the ordered playbook sequence.
func TestEmInstallStart_BuildsTaskSequence(t *testing.T) {
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

	status, _, tasks, _ := states.InstallEM.Snapshot()
	if status != models.StatusRunning {
		t.Fatalf("expected running status, got %s", status)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	wantOrder := []string{"initial_setup", "install_agent", "register_agents"}
	for i, want := range wantOrder {
		if !strings.Contains(tasks[i].Name, want) {
			t.Errorf("task %d name = %q, want it to mention %q", i, tasks[i].Name, want)
		}
		if tasks[i].ID != i+1 {
			t.Errorf("task %d ID = %d, want %d", i, tasks[i].ID, i+1)
		}
	}

	if st, _, _, _ := states.Install.Snapshot(); st != models.StatusIdle {
		t.Errorf("install state should stay idle, got %s", st)
	}
}

// TestEmInstallStart_RejectsSecondStart verifies a second start while running
// is rejected with HTTP 409 instead of spawning a parallel run.
func TestEmInstallStart_RejectsSecondStart(t *testing.T) {
	states := newStates()
	states.InstallEM.SetRunning([]models.Task{
		{ID: 1, Name: "em-task", Status: models.TaskRunning},
	}, func() {})

	rec := httptest.NewRecorder()
	handlers.EmInstallStart(states).ServeHTTP(
		rec, httptest.NewRequest(http.MethodPost, "/api/install/ydb-em/start", nil))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

// TestEmInstallStart_MethodNotAllowed verifies GET is rejected.
func TestEmInstallStart_MethodNotAllowed(t *testing.T) {
	states := newStates()

	rec := httptest.NewRecorder()
	handlers.EmInstallStart(states).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, "/api/install/ydb-em/start", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

// TestOperationStates_ForFlow verifies ForFlow returns the correct state.
func TestOperationStates_ForFlow(t *testing.T) {
	states := newStates()

	if states.ForFlow(models.FlowYDB) != states.Install {
		t.Error("FlowYDB should return Install state")
	}
	if states.ForFlow(models.FlowYDBEM) != states.InstallEM {
		t.Error("FlowYDBEM should return InstallEM state")
	}
	if states.ForFlow(models.FlowDeleteYDB) != states.DeleteYdb {
		t.Error("FlowDeleteYDB should return DeleteYdb state")
	}
	if states.ForFlow(models.FlowDeleteEMYDB) != states.DeleteEM {
		t.Error("FlowDeleteEMYDB should return DeleteEM state")
	}
	if states.ForFlow(models.FlowNone) != nil {
		t.Error("FlowNone should return nil")
	}
	if states.ForFlow("bogus") != nil {
		t.Error("unknown flow should return nil")
	}
}

// TestDeleteEmYdbStart_BuildsTaskSequence verifies the EM delete handler
// responds 202 and seeds tasks from unregister_agents → uninstall only
// (no YDB format_cluster).
func TestDeleteEmYdbStart_BuildsTaskSequence(t *testing.T) {
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

	status, _, tasks, _ := states.DeleteEM.Snapshot()
	if status != models.StatusRunning {
		t.Fatalf("expected running status, got %s", status)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	wantOrder := []string{"unregister_agents", "uninstall"}
	for i, want := range wantOrder {
		if !strings.Contains(tasks[i].Name, want) {
			t.Errorf("task %d name = %q, want it to mention %q", i, tasks[i].Name, want)
		}
		if !strings.Contains(tasks[i].Name, "ydb_platform.ydb_em") {
			t.Errorf("task %d name = %q, want ydb_platform.ydb_em namespace", i, tasks[i].Name)
		}
		if strings.Contains(tasks[i].Name, "format_cluster") {
			t.Errorf("task %d must not reference format_cluster, got %q", i, tasks[i].Name)
		}
		if tasks[i].ID != i+1 {
			t.Errorf("task %d ID = %d, want %d", i, tasks[i].ID, i+1)
		}
	}

	if st, _, _, _ := states.Install.Snapshot(); st != models.StatusIdle {
		t.Errorf("install state should stay idle, got %s", st)
	}
	if st, _, _, _ := states.DeleteYdb.Snapshot(); st != models.StatusIdle {
		t.Errorf("delete-ydb state should stay idle, got %s", st)
	}
}

// TestDeleteEmYdbStart_RejectsSecondStart verifies a second start while running
// is rejected with HTTP 409.
func TestDeleteEmYdbStart_RejectsSecondStart(t *testing.T) {
	states := newStates()
	states.DeleteEM.SetRunning([]models.Task{
		{ID: 1, Name: "em-delete-task", Status: models.TaskRunning},
	}, func() {})

	rec := httptest.NewRecorder()
	handlers.DeleteEmYdbStart(states).ServeHTTP(
		rec, httptest.NewRequest(http.MethodPost, "/api/install/delete-em-ydb/start", nil))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

// TestDeleteEmYdbStart_MethodNotAllowed verifies GET is rejected.
func TestDeleteEmYdbStart_MethodNotAllowed(t *testing.T) {
	states := newStates()

	rec := httptest.NewRecorder()
	handlers.DeleteEmYdbStart(states).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, "/api/install/delete-em-ydb/start", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

// TestInstallStatus_DeleteEmFlowIsolated verifies delete-em-ydb status does not
// leak install or delete-ydb state.
func TestInstallStatus_DeleteEmFlowIsolated(t *testing.T) {
	states := newStates()

	states.Install.SetRunning([]models.Task{
		{ID: 1, Name: "ydb-task", Status: models.TaskRunning},
	}, func() {})
	states.DeleteYdb.SetRunning([]models.Task{
		{ID: 1, Name: "delete-ydb-task", Status: models.TaskDone},
	}, func() {})
	states.DeleteYdb.SetDone()
	states.DeleteEM.SetRunning([]models.Task{
		{ID: 1, Name: "delete-em-task", Status: models.TaskRunning},
	}, func() {})

	store := session.New("en")
	flow := models.FlowDeleteEMYDB
	store.Update(models.SessionUpdate{Flow: &flow})

	rec := httptest.NewRecorder()
	handlers.InstallStatus(states, store).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, "/api/install/status", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != string(models.StatusRunning) {
		t.Errorf("status = %v, want running", body["status"])
	}
	tasks, _ := body["tasks"].([]any)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task from DeleteEM, got %d", len(tasks))
	}
	task0, _ := tasks[0].(map[string]any)
	if name, _ := task0["name"].(string); name != "delete-em-task" {
		t.Errorf("task name = %v, want delete-em-task", task0["name"])
	}
}

// TestOperationStates_EmIsDistinctInstance verifies InstallEM is its own
// instance with its own Broadcaster — the guarantee that prevents EM state
// from leaking into the other flows.
func TestOperationStates_EmIsDistinctInstance(t *testing.T) {
	states := newStates()

	if states.InstallEM == nil {
		t.Fatal("InstallEM must be initialised")
	}
	if states.InstallEM == states.Install {
		t.Error("InstallEM must not alias Install")
	}
	if states.InstallEM == states.DeleteYdb || states.InstallEM == states.DeleteEM {
		t.Error("InstallEM must not alias a delete state")
	}
	if states.InstallEM.Broadcaster == nil {
		t.Error("InstallEM must have its own Broadcaster")
	}
	if states.InstallEM.Broadcaster == states.Install.Broadcaster {
		t.Error("InstallEM must not share the Install Broadcaster")
	}
}

// TestInstallStatus_OpQueryOverridesSessionFlow verifies ?op=ydb-em returns
// InstallEM state even when the session flow is still ydb (screen 9.1 phase).
func TestInstallStatus_OpQueryOverridesSessionFlow(t *testing.T) {
	states, store := newStatesWithFlow(models.FlowYDB)

	states.InstallEM.SetRunning([]models.Task{
		{ID: 1, Name: "em-task", Status: models.TaskPending},
	}, func() {})
	states.InstallEM.SetDone()

	rec := httptest.NewRecorder()
	handlers.InstallStatus(states, store).ServeHTTP(
		rec, httptest.NewRequest(http.MethodGet, "/api/install/status", nil))
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != string(models.StatusIdle) {
		t.Errorf("without op: status = %v, want idle", body["status"])
	}

	rec2 := httptest.NewRecorder()
	handlers.InstallStatus(states, store).ServeHTTP(
		rec2, httptest.NewRequest(http.MethodGet, "/api/install/status?op=ydb-em", nil))
	var body2 map[string]any
	if err := json.NewDecoder(rec2.Body).Decode(&body2); err != nil {
		t.Fatalf("decode op: %v", err)
	}
	if body2["status"] != string(models.StatusSuccess) {
		t.Errorf("with op=ydb-em: status = %v, want success", body2["status"])
	}
}
