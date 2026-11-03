package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/ansible"
	"ydb-installer/internal/services/checker"
	"ydb-installer/internal/services/config"
	"ydb-installer/internal/services/session"
)

// installYdbTasks defines the ordered task list for the YDB installation.
var installYdbTasks = []models.Task{
	{ID: 1, Name: "Running ydb_platform.ydb.initial_setup", Status: models.TaskPending},
}

// emCollection is the Ansible collection namespace that provides the YDB EM
// playbooks. Declared once so a collection rename is a single-line change.
const emCollection = "ydb_platform.ydb_em"

// playbookStep pairs an Ansible playbook with the user-visible task label
// shown on the installation screen.
type playbookStep struct {
	// Playbook is the fully-qualified Ansible playbook name.
	Playbook string
	// TaskName is the label rendered in the task list.
	TaskName string
	// ExtraVars is an optional JSON string passed to ansible-playbook via
	// --extra-vars. Empty means no extra vars.
	ExtraVars string
}

// installEmSteps is the ordered YDB EM installation sequence and the single
// source of truth for it: the task list, the progress percentage and the log
// prefixes are all derived from this slice. Adding, removing or reordering a
// step is therefore a one-line change here.
var installEmSteps = []playbookStep{
	{Playbook: emCollection + ".initial_setup", TaskName: "Running " + emCollection + ".initial_setup"},
	{Playbook: emCollection + ".install_agent", TaskName: "Running " + emCollection + ".install_agent"},
	{Playbook: emCollection + ".register_agents", TaskName: "Running " + emCollection + ".register_agents"},
}

// buildTasks converts an ordered playbook sequence into a fresh task list.
// Returns a new slice on every call so concurrent operations never share
// task state.
func buildTasks(steps []playbookStep) []models.Task {
	tasks := make([]models.Task, len(steps))
	for i, s := range steps {
		tasks[i] = models.Task{ID: i + 1, Name: s.TaskName, Status: models.TaskPending}
	}
	return tasks
}

// deleteYdbTasks defines the ordered task list for the delete-ydb operation.
var deleteYdbTasks = []models.Task{
	{ID: 1, Name: "Running ydb_platform.ydb.format_cluster", Status: models.TaskPending},
}

// deleteEmSteps is the ordered YDB EM deletion sequence (EM-only; does not
// wipe the YDB cluster). unregister_agents runs first so CP drops agent
// registrations while EM is still up; uninstall then removes EM services
// and files. uninstall defaults to Ansible group "ydb", so ExtraVars pins
// the target to ydb_em hosts only.
var deleteEmSteps = []playbookStep{
	{Playbook: emCollection + ".unregister_agents", TaskName: "Running " + emCollection + ".unregister_agents"},
	{
		Playbook:  emCollection + ".uninstall",
		TaskName:  "Running " + emCollection + ".uninstall",
		ExtraVars: `{"playbook_hosts":"ydb_em"}`,
	},
}

// createDatabaseTasks defines the task list for creating a single database.
// The task name is set dynamically with the database name by the handler.
var createDatabaseTasks = []models.Task{
	{ID: 1, Name: "Creating database", Status: models.TaskPending},
}

// createDatabaseRequest is the JSON body for POST /api/install/ydb/create-database.
type createDatabaseRequest struct {
	Name     string   `json:"name"`
	Groups   int      `json:"groups"`
	Hosts    []string `json:"hosts,omitempty"`
	Affinity string   `json:"affinity,omitempty"`
}

// prepareDynamicNode calculates a new dynamic node for the given database,
// patches inventory.yaml with the merged node list, and returns the
// extra-vars JSON string for ansible-playbook --extra-vars.
func prepareDynamicNode(inventoryPath, dbName string, groups int) (string, error) {
	invService := config.NewInventoryService()
	invData, err := invService.Parse(inventoryPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse inventory: %w", err)
	}

	existingNodes := invData.DynamicNodes

	// Check if a node with this dbname already exists (user pre-configured it).
	// If found, reuse its instance/offset instead of computing new ones.
	// This respects the user's pre-configured ydb_dynnodes mapping and
	// prevents creating duplicate instance letters for the same database.
	var instance string
	var offset int
	found := false
	for _, node := range existingNodes {
		if node.Dbname == dbName {
			instance = node.Instance
			offset = node.Offset
			found = true
			break
		}
	}

	if !found {
		startIdx := len(existingNodes)
		instance = string(rune('a' + startIdx))
		offset = startIdx + 1
	}

	// Build extra-vars with only this node (not the full list).
	// The Ansible playbook receives a single node and merges it with
	// whatever is already in the inventory's ydb_dynnodes.
	extraVarsMap := map[string]interface{}{
		"ydb_dynnodes": []map[string]interface{}{{
			"instance": instance,
			"offset":   offset,
			"dbname":   dbName,
		}},
		"ydb_database_groups": groups,
		"ydb_dbname":          dbName,
	}
	extraVarsBytes, err := json.Marshal(extraVarsMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal extra-vars: %w", err)
	}

	// Inventory is NOT patched here. ydb_dynnodes is kept in sync with
	// Database.Entries by buildVars (inventory_writer.go) during normal
	// UI → inventory flow. Keeping inventory as "desired configuration"
	// avoids stale entries when a playbook fails after patching.
	return string(extraVarsBytes), nil
}

// installStatusResponse is the JSON body for GET /api/install/status.
type installStatusResponse struct {
	Status   models.InstallStatus `json:"status"`
	Progress int                  `json:"progress"`
	Tasks    []models.Task        `json:"tasks"`
	Error    *models.InstallError `json:"error,omitempty"`
}

// generateConfigResponse is the JSON body for POST /api/install/ydb/generate-config.
type generateConfigResponse struct {
	Success bool   `json:"success"`
	Path    string `json:"path,omitempty"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ConfigMatchStatus describes whether on-disk config.yaml matches the current
// inventory-driven expected output.
type ConfigMatchStatus string

const (
	ConfigMatchStatusMissing  ConfigMatchStatus = "missing"
	ConfigMatchStatusMatches  ConfigMatchStatus = "matches"
	ConfigMatchStatusOutdated ConfigMatchStatus = "outdated"
)

// configStatusResponse is the JSON body for GET /api/install/ydb/config-status.
type configStatusResponse struct {
	Success bool              `json:"success"`
	Status  ConfigMatchStatus `json:"status,omitempty"`
	Path    string            `json:"path,omitempty"`
	Content string            `json:"content,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// generateConfigRequest is the optional JSON body for POST /api/install/ydb/generate-config.
type generateConfigRequest struct {
	Topology   string `json:"topology,omitempty"`
	SkipBackup bool   `json:"skip_backup,omitempty"`
}

// enrichInventoryFromSession merges DC host lists and topology from the
// session into inv. DC1/DC2/DC3 and the topology are kept in Pinia/Session
// only — NOT in the inventory YAML. Without this merge, buildHostsData falls
// back to round-robin distribution (wrong zones, broken fail_domains) and the
// config renders with the default mirror-3-dc erasure. Used by both
// GenerateConfig and ConfigStatus (dry-run compare).
func enrichInventoryFromSession(inv *models.InventoryData, store *session.Store) {
	if store == nil {
		return
	}
	sessionInv := store.GetInventory()
	if sessionInv == nil {
		return
	}
	if len(sessionInv.DC1) > 0 {
		inv.DC1 = sessionInv.DC1
	}
	if len(sessionInv.DC2) > 0 {
		inv.DC2 = sessionInv.DC2
	}
	if len(sessionInv.DC3) > 0 {
		inv.DC3 = sessionInv.DC3
	}
	// Fill only when unset: GenerateConfig applies the request-body topology
	// before this call and that choice must win.
	if inv.Topology == nil && sessionInv.Topology != nil && sessionInv.Topology.Type != "" {
		inv.Topology = &models.TopologyConfig{Type: sessionInv.Topology.Type}
	}
}

// normalizeConfigContent trims surrounding whitespace and ensures a single
// trailing newline for deterministic string comparison.
func normalizeConfigContent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return s + "\n"
}

// GenerateConfig handles POST /api/install/ydb/generate-config.
//
// inventoryPath and configPath are filesystem paths. store provides session
// data (including DC1/DC2/DC3 host lists) that is NOT persisted in the
// inventory YAML file but MUST be merged before config generation.
func GenerateConfig(store *session.Store, inventoryPath, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if inventoryPath == "" {
			writeJSON(w, http.StatusOK, generateConfigResponse{
				Success: false,
				Error:   "inventory path is empty",
			})
			return
		}

		if configPath == "" {
			writeJSON(w, http.StatusOK, generateConfigResponse{
				Success: false,
				Error:   "config path is empty",
			})
			return
		}

		var reqBody generateConfigRequest
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&reqBody)
		}

		invService := config.NewInventoryService()
		inv, err := invService.Parse(inventoryPath)
		if err != nil {
			writeJSON(w, http.StatusOK, generateConfigResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to read inventory: %v", err),
			})
			return
		}

		// Topology comes from the Pinia store via the frontend, not the inventory YAML.
		if reqBody.Topology != "" {
			inv.Topology = &models.TopologyConfig{Type: reqBody.Topology}
		}

		enrichInventoryFromSession(inv, store)

		gen := config.NewGenerator(configPath)

		if !reqBody.SkipBackup {
			backupPath, err := gen.Backup()
			if err != nil {
				// Backup is non-critical — log but don't fail.
				fmt.Printf("Warning: failed to backup existing config: %v\n", err)
			} else if backupPath != "" {
				fmt.Printf("Backed up existing config to %s\n", backupPath)
			}
		} else {
			fmt.Printf("Skipping backup for regeneration\n")
		}

		if err := gen.Generate(inv); err != nil {
			writeJSON(w, http.StatusOK, generateConfigResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to generate config: %v", err),
			})
			return
		}

		content, err := gen.Read()
		if err != nil {
			writeJSON(w, http.StatusOK, generateConfigResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to read generated config: %v", err),
			})
			return
		}

		writeJSON(w, http.StatusOK, generateConfigResponse{
			Success: true,
			Path:    configPath,
			Content: content,
		})
	}
}

// ConfigStatus handles GET /api/install/ydb/config-status.
//
// It performs a dry-run compare: it renders the expected config.yaml from the
// current inventory+session state and compares it to the on-disk file.
// The response status field is one of: missing | matches | outdated.
//
// inventoryPath and store are needed to enrich the inventory with session DC
// lists (same parity as GenerateConfig). ConfigPath is where config.yaml lives.
func ConfigStatus(store *session.Store, inventoryPath, configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		gen := config.NewGenerator(configPath)

		// gen.Read returns "" when the file does not exist.
		diskContent, err := gen.Read()
		if err != nil {
			writeJSON(w, http.StatusOK, configStatusResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to read config: %v", err),
			})
			return
		}

		if diskContent == "" {
			writeJSON(w, http.StatusOK, configStatusResponse{
				Success: true,
				Status:  ConfigMatchStatusMissing,
				Path:    configPath,
			})
			return
		}

		// Enrich for parity with GenerateConfig.
		if inventoryPath == "" {
			// No inventory path — can't compare; report exists but status unknown.
			writeJSON(w, http.StatusOK, configStatusResponse{
				Success: true,
				Status:  ConfigMatchStatusOutdated,
				Path:    configPath,
				Content: diskContent,
			})
			return
		}

		invService := config.NewInventoryService()
		inv, err := invService.Parse(inventoryPath)
		if err != nil {
			writeJSON(w, http.StatusOK, configStatusResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to read inventory: %v", err),
			})
			return
		}
		enrichInventoryFromSession(inv, store)

		expected, err := gen.Render(inv)
		if err != nil {
			writeJSON(w, http.StatusOK, configStatusResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to render expected config: %v", err),
			})
			return
		}

		matchStatus := ConfigMatchStatusMatches
		if normalizeConfigContent(expected) != normalizeConfigContent(diskContent) {
			matchStatus = ConfigMatchStatusOutdated
		}

		writeJSON(w, http.StatusOK, configStatusResponse{
			Success: true,
			Status:  matchStatus,
			Path:    configPath,
			Content: diskContent,
		})
	}
}

// YdbStart handles POST /api/install/ydb/start.
// Always uses the Install state (hard-coded — this endpoint is only for install flow).
func YdbStart(states *models.OperationStates, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var tasks []models.Task
		taskID := 1

		tasks = append(tasks, models.Task{
			ID: taskID, Name: "Running ydb_platform.ydb.initial_setup",
			Status: models.TaskPending,
		})
		taskID++

		if store != nil {
			if inv := store.GetInventory(); inv != nil && inv.Database != nil {
				for i, entry := range inv.Database.Entries {
					if i == 0 {
						continue // initial_setup creates the first DB.
					}
					tasks = append(tasks, models.Task{
						ID:     taskID,
						Name:   fmt.Sprintf("Creating database %q", entry.Name),
						Status: models.TaskPending,
					})
					taskID++
				}

				if inv.Database.ConfigVariant == "single-plus-em" {
					for _, step := range installEmSteps {
						tasks = append(tasks, models.Task{
							ID:     taskID,
							Name:   step.TaskName,
							Status: models.TaskPending,
						})
						taskID++
					}
				}
			}
		}

		cancelCh := make(chan struct{})
		once := make(chan struct{})
		cancelFn := func() {
			select {
			case <-once:
			default:
				close(once)
				close(cancelCh)
			}
		}

		states.Install.SetRunning(tasks, cancelFn)
		go runYdbInstall(states.Install, store, tasks, cancelCh)

		writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
	}
}

// EmInstallStart handles POST /api/install/ydb-em/start.
// Always uses the InstallEM state (hard-coded — this endpoint is only for the
// ydb-em flow).
//
// Unlike the YDB install there is no config.yaml generation step: the whole EM
// configuration lives in 90-inventory.yaml, which screens 21–23 already keep in
// sync. Screen 24 therefore only runs the playbook sequence.
func EmInstallStart(states *models.OperationStates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if st, _, _, _ := states.InstallEM.Snapshot(); st == models.StatusRunning {
			writeJSON(w, http.StatusConflict, map[string]string{
				"status": "already_running",
				"error":  "EM installation already running",
			})
			return
		}

		tasks := buildTasks(installEmSteps)

		cancelCh := make(chan struct{})
		once := make(chan struct{})
		cancelFn := func() {
			select {
			case <-once:
			default:
				close(once)
				close(cancelCh)
			}
		}

		states.InstallEM.SetRunning(tasks, cancelFn)
		go runPlaybookSequence(states.InstallEM, tasks, installEmSteps, cancelCh, "install-em")

		writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
	}
}

// CreateDatabase handles POST /api/install/ydb/create-database.
// Always uses the Install state (hard-coded — this endpoint is only for install flow).
//
// Starts an Ansible playbook (ydb_platform.ydb.install_dynamic) to create a
// single database on the already-running cluster. The frontend polls
// /api/install/status for completion, then proceeds to the next database.
//
// The handler calculates instance/offset for new dynamic nodes based on the
// existing ydb_dynnodes in the inventory, patches the inventory file with the
// updated node list, and passes extra-vars (ydb_dynnodes, ydb_dbname,
// ydb_database_groups) to the playbook.
func CreateDatabase(states *models.OperationStates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req createDatabaseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}

		inventoryPath := states.Install.InventoryPath
		if inventoryPath == "" {
			writeError(w, http.StatusInternalServerError, "inventory path not configured")
			return
		}

		extraVars, err := prepareDynamicNode(inventoryPath, req.Name, req.Groups)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		tasks := make([]models.Task, len(createDatabaseTasks))
		copy(tasks, createDatabaseTasks)
		tasks[0].Name = fmt.Sprintf("Creating database: %s", req.Name)

		cancelCh := make(chan struct{})
		once := make(chan struct{})
		cancelFn := func() {
			select {
			case <-once:
			default:
				close(once)
				close(cancelCh)
			}
		}

		states.Install.SetRunning(tasks, cancelFn)

		states.Install.WriteLog(fmt.Sprintf("[create-db] Creating database %q (groups=%d)", req.Name, req.Groups))

		go runAnsibleOperation(states.Install, tasks, cancelCh, "ydb_platform.ydb.install_dynamic", "create-database", extraVars)

		writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
	}
}

// DeleteYdbStart handles POST /api/install/delete-ydb/start.
// Always uses the DeleteYdb state (hard-coded — this endpoint is only for delete-ydb flow).
func DeleteYdbStart(states *models.OperationStates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if st, _, _, _ := states.DeleteYdb.Snapshot(); st == models.StatusRunning {
			writeJSON(w, http.StatusConflict, map[string]string{"status": "already_running", "error": "deletion already running"})
			return
		}

		tasks := make([]models.Task, len(deleteYdbTasks))
		copy(tasks, deleteYdbTasks)

		cancelCh := make(chan struct{})
		once := make(chan struct{})
		cancelFn := func() {
			select {
			case <-once:
			default:
				close(once)
				close(cancelCh)
			}
		}

		states.DeleteYdb.SetRunning(tasks, cancelFn)
		go runAnsibleOperation(states.DeleteYdb, tasks, cancelCh, "ydb_platform.ydb.format_cluster", "delete-ydb", "")

		writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
	}
}

// DeleteEmYdbStart handles POST /api/install/delete-em-ydb/start.
// Always uses the DeleteEM state (hard-coded — this endpoint is only for the
// delete-em-ydb flow). Removes YDB EM only (unregister agents + uninstall);
// does not format/wipe the YDB cluster.
func DeleteEmYdbStart(states *models.OperationStates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		if st, _, _, _ := states.DeleteEM.Snapshot(); st == models.StatusRunning {
			writeJSON(w, http.StatusConflict, map[string]string{
				"status": "already_running",
				"error":  "EM deletion already running",
			})
			return
		}

		tasks := buildTasks(deleteEmSteps)

		cancelCh := make(chan struct{})
		once := make(chan struct{})
		cancelFn := func() {
			select {
			case <-once:
			default:
				close(once)
				close(cancelCh)
			}
		}

		states.DeleteEM.SetRunning(tasks, cancelFn)
		go runPlaybookSequence(states.DeleteEM, tasks, deleteEmSteps, cancelCh, "delete-em-ydb")

		writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
	}
}

// resolveInstallState picks the InstallState for a request.
// Optional query param op=<flow> overrides the session flow so an in-flow
// secondary phase (e.g. EM install while session.flow is still ydb) can
// poll/cancel/stream the correct OperationStates slot.
func resolveInstallState(states *models.OperationStates, store *session.Store, r *http.Request) *models.InstallState {
	if op := r.URL.Query().Get("op"); op != "" {
		f := models.Flow(op)
		if models.IsValidFlow(f) && f != models.FlowNone {
			if st := states.ForFlow(f); st != nil {
				return st
			}
		}
	}
	return states.ForFlow(store.Get().Flow)
}

// InstallStatus handles GET /api/install/status.
// Returns the state for the current session flow so that install and delete
// operations never leak into each other. Optional ?op=<flow> overrides the
// session flow (used by ydb-flow EM phase on screen 9.1).
func InstallStatus(states *models.OperationStates, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		state := resolveInstallState(states, store, r)
		if state == nil {
			writeJSON(w, http.StatusOK, installStatusResponse{
				Status: models.StatusIdle,
			})
			return
		}

		status, progress, tasks, installErr := state.Snapshot()
		writeJSON(w, http.StatusOK, installStatusResponse{
			Status:   status,
			Progress: progress,
			Tasks:    tasks,
			Error:    installErr,
		})
	}
}

// InstallCancel handles POST /api/install/cancel.
// Cancels the operation for the current session flow (or ?op= override).
func InstallCancel(states *models.OperationStates, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		state := resolveInstallState(states, store, r)
		if state == nil {
			writeError(w, http.StatusBadRequest, "no active flow")
			return
		}
		state.Cancel()
		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
	}
}

// DepsInstruction handles GET /api/system/deps/instruction/{os}?dep=<name>.
// Delegates to checker.InstructionProvider (single source of truth).
func DepsInstruction() http.HandlerFunc {
	provider := &checker.InstructionProvider{}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/system/deps/instruction/")
		osName := strings.SplitN(path, "/", 2)[0]
		if osName == "" {
			writeError(w, http.StatusBadRequest, "os is required in path")
			return
		}

		dep := r.URL.Query().Get("dep")
		if dep == "" {
			writeError(w, http.StatusBadRequest, "dep query param is required")
			return
		}

		commands := provider.ForDep(dep, osName)
		writeJSON(w, http.StatusOK, map[string]string{
			"dep":      dep,
			"os":       osName,
			"commands": commands,
		})
	}
}

// runSinglePlaybook starts an Ansible playbook, waits for completion or
// cancellation, and returns an error if the playbook failed.
// Does NOT manage task status or broadcaster — caller handles those.
func runSinglePlaybook(
	state *models.InstallState,
	playbook string,
	extraVars string,
	cancelCh <-chan struct{},
) error {
	inventoryPath := state.InventoryPath
	if filepath.Ext(inventoryPath) == ".yaml" || filepath.Ext(inventoryPath) == ".yml" {
		inventoryPath = filepath.Dir(inventoryPath)
	}

	runner := ansible.NewRunnerWithRuntime(state.AnsibleRuntime)
	done := make(chan struct{})

	err := runner.Start(inventoryPath, playbook, state.WriteLog, done, extraVars)
	if err != nil {
		return fmt.Errorf("failed to start %s: %w", playbook, err)
	}

	select {
	case <-cancelCh:
		runner.Cancel()
		return fmt.Errorf("cancelled")
	case <-done:
	}

	if err := runner.WaitForCompletion(); err != nil {
		return fmt.Errorf("%s failed: %w", playbook, err)
	}

	return nil
}

// runYdbInstall runs initial_setup followed by install_dynamic for each
// additional database entry. Tasks are updated sequentially.
// Designed to run in a goroutine; YdbStart returns 202 immediately.
func runYdbInstall(
	state *models.InstallState,
	store *session.Store,
	tasks []models.Task,
	cancelCh <-chan struct{},
) {
	if state.Broadcaster != nil {
		state.Broadcaster.Reset("install-ydb")
	}

	inventoryPath := state.InventoryPath
	if inventoryPath == "" {
		state.WriteLog("[install] Error: inventory path not configured")
		state.SetError("install", "", "inventory path not configured")
		return
	}

	state.UpdateTask(tasks[0].ID, models.TaskRunning)
	state.WriteLog("[install] Starting initial_setup")

	if err := runSinglePlaybook(state, "ydb_platform.ydb.initial_setup", "", cancelCh); err != nil {
		state.SetError("initial_setup", "", err.Error())
		return
	}
	state.UpdateTask(tasks[0].ID, models.TaskDone)

	// We do NOT skip databases based on pre-existing ydb_dynnodes entries —
	// the presence of a dbname field in ydb_dynnodes is user configuration,
	// not proof that the database already exists on the cluster. The Ansible
	// playbook handles idempotency: if the database already exists, it
	// succeeds without side effects.
	if store != nil {
		if inv := store.GetInventory(); inv != nil && inv.Database != nil {
			for i, entry := range inv.Database.Entries {
				if i == 0 {
					continue // initial_setup creates the first DB.
				}
				taskIdx := i // 1-based for tasks[1..N]
				if taskIdx >= len(tasks) {
					break
				}

				extraVars, err := prepareDynamicNode(inventoryPath, entry.Name, entry.Groups)
				if err != nil {
					state.WriteLog(fmt.Sprintf("[install] Error preparing %q: %v", entry.Name, err))
					state.SetError("prepare", entry.Name, err.Error())
					return
				}

				state.UpdateTask(tasks[taskIdx].ID, models.TaskRunning)
				state.WriteLog(fmt.Sprintf("[install] Creating database %q", entry.Name))

				if err := runSinglePlaybook(state, "ydb_platform.ydb.install_dynamic", extraVars, cancelCh); err != nil {
					state.SetError("install_dynamic", entry.Name, err.Error())
					return
				}
				state.UpdateTask(tasks[taskIdx].ID, models.TaskDone)
			}
		}

		if inv := store.GetInventory(); inv != nil && inv.Database != nil && inv.Database.ConfigVariant == "single-plus-em" {
			emTaskOffset := 1 + (len(inv.Database.Entries) - 1)
			for j, step := range installEmSteps {
				taskIdx := emTaskOffset + j
				if taskIdx >= len(tasks) {
					state.WriteLog(fmt.Sprintf("[install] skipping %s — no task slot", step.Playbook))
					continue
				}
				state.UpdateTask(tasks[taskIdx].ID, models.TaskRunning)
				state.WriteLog(fmt.Sprintf("[install] %s", step.TaskName))
				if err := runSinglePlaybook(state, step.Playbook, step.ExtraVars, cancelCh); err != nil {
					state.SetError("em-install", "", err.Error())
					return
				}
				state.UpdateTask(tasks[taskIdx].ID, models.TaskDone)
			}
		}
	}

	state.SetDone()
}

// runPlaybookSequence runs an ordered list of playbooks one after another,
// updating the matching task as each one starts and completes.
//
// It is the generic counterpart to runYdbInstall: use it whenever an operation
// is a fixed sequence of playbooks with no data passed between the steps.
// runYdbInstall is deliberately NOT built on top of this because it computes
// per-database extra-vars between playbooks.
//
// Aborts on the first failure (SetError) and honours cancellation
// (SetCancelled). Designed to run in a goroutine; the caller returns 202
// immediately.
func runPlaybookSequence(
	state *models.InstallState,
	tasks []models.Task,
	steps []playbookStep,
	cancelCh <-chan struct{},
	operationType string,
) {
	if len(steps) == 0 {
		state.SetDone()
		return
	}

	// Reset once for the whole sequence — clears the ring buffer from any
	// previous run and rotates the log file.
	if state.Broadcaster != nil {
		state.Broadcaster.Reset(operationType)
	}

	if state.InventoryPath == "" {
		state.WriteLog("[" + operationType + "] Error: inventory path not configured")
		state.SetError(operationType, "", "inventory path not configured")
		return
	}

	for i, step := range steps {
		if i >= len(tasks) {
			break
		}

		// Cancellation may arrive between two playbooks, where no runner is
		// active to observe it — check explicitly before starting the next one.
		select {
		case <-cancelCh:
			state.WriteLog("[" + operationType + "] Cancelled by user")
			state.SetCancelled()
			return
		default:
		}

		state.UpdateTask(tasks[i].ID, models.TaskRunning)
		state.WriteLog("[" + operationType + "] Starting: " + step.Playbook)

		if err := runSinglePlaybook(state, step.Playbook, step.ExtraVars, cancelCh); err != nil {
			// runSinglePlaybook reports a cancellation as a plain error, so
			// distinguish the two here to set the correct terminal state.
			select {
			case <-cancelCh:
				state.WriteLog("[" + operationType + "] Cancelled by user")
				state.SetCancelled()
			default:
				state.WriteLog("[" + operationType + "] Error: " + err.Error())
				state.SetError(step.Playbook, "", err.Error())
			}
			return
		}

		state.UpdateTask(tasks[i].ID, models.TaskDone)
	}

	state.SetDone()
	state.WriteLog("[" + operationType + "] Operation completed successfully")
}

// runAnsibleOperation executes an Ansible playbook for the given operation.
// extraVars is an optional JSON string passed to ansible-playbook via
// --extra-vars; pass "" to skip.
func runAnsibleOperation(state *models.InstallState, tasks []models.Task, cancelCh <-chan struct{}, playbook string, operationType string, extraVars string, prePlaybookFn ...func()) {
	if len(tasks) == 0 {
		state.SetDone()
		return
	}

	// Reset clears old ring buffer data and rotates the log file so new
	// subscribers start with a clean slate.
	if state.Broadcaster != nil {
		state.Broadcaster.Reset(operationType)
	}

	// Executed after Broadcaster.Reset() so log messages are preserved.
	for _, fn := range prePlaybookFn {
		fn()
	}

	inventoryPath := state.InventoryPath
	if inventoryPath == "" {
		state.WriteLog("[ansible] Error: inventory path not configured")
		state.SetError("AnsibleRunner", "", "inventory path not configured")
		return
	}
	if filepath.Ext(inventoryPath) == ".yaml" || filepath.Ext(inventoryPath) == ".yml" {
		inventoryPath = filepath.Dir(inventoryPath)
		state.WriteLog("[ansible] Using inventory directory: " + inventoryPath)
	}

	state.UpdateTask(tasks[0].ID, models.TaskRunning)
	state.WriteLog("[ansible] Starting: " + playbook)

	// When a bundled runtime is configured on the install state,
	// ansible-playbook runs through it (no host deps needed); otherwise it
	// falls back to the host's system ansible-playbook.
	runner := ansible.NewRunnerWithRuntime(state.AnsibleRuntime)
	done := make(chan struct{})

	// WriteLog delegates to Broadcaster.Publish() for fan-out to all subscribers.
	err := runner.Start(inventoryPath, playbook, state.WriteLog, done, extraVars)
	if err != nil {
		state.WriteLog("[ansible] Error starting playbook: " + err.Error())
		state.SetError("AnsibleRunner", "", err.Error())
		return
	}

	select {
	case <-cancelCh:
		runner.Cancel()
		state.WriteLog("[ansible] Cancelled by user")
		state.SetCancelled()
		return
	case <-done:
		state.WriteLog("[ansible] Playbook completed")
	}

	if err := runner.WaitForCompletion(); err != nil {
		state.WriteLog("[ansible] Error: " + err.Error())
		state.SetError(playbook, "", err.Error())
		return
	}

	state.UpdateTask(tasks[0].ID, models.TaskDone)
	state.SetDone()
	state.WriteLog("[ansible] Operation completed successfully")
}

// GetLogs handles GET /api/install/logs
// Returns the ring buffer snapshot from the broadcaster for the current
// session flow (in-memory). Optional ?op=<flow> overrides the session flow.
func GetLogs(states *models.OperationStates, store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		state := resolveInstallState(states, store, r)
		if state == nil || state.Broadcaster == nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"logs":   []string{},
				"source": "none",
			})
			return
		}

		lines := state.Broadcaster.Snapshot()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"logs":   lines,
			"source": "ringbuffer",
			"count":  len(lines),
		})
	}
}
