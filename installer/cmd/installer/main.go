// Command installer is the single binary that runs the YDB Installer.
//
// It serves a Vue.js SPA (embedded via internal/embed) plus a small REST API
// on a local HTTP port.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"ydb-installer/internal/api"
	"ydb-installer/internal/api/handlers"
	"ydb-installer/internal/banner"
	installerembed "ydb-installer/internal/embed"
	"ydb-installer/internal/models"
	"ydb-installer/internal/runtime"
	"ydb-installer/internal/services/ansible"
	"ydb-installer/internal/services/checker"
	"ydb-installer/internal/services/config"
	langdetect "ydb-installer/internal/services/lang"
	"ydb-installer/internal/services/session"
)

func main() {
	port := flag.Int("port", 8080, "HTTP port to listen on")
	host := flag.String("host", "127.0.0.1", "bind address (use 0.0.0.0 for external access)")
	noBrowser := flag.Bool("no-browser", false, "do not try to open the system browser")
	useSystemDeps := flag.Bool("use-system-deps", false, "ignore the bundled runtime and use the host's system python/ansible")
	extractRuntimeOnly := flag.Bool("extract-runtime-only", false, "extract the bundled runtime and exit (pre-warm cache)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "YDB Installer version %s.\n", Version)
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "EN. Place %s into the folder with existing ansible configuration or empty folder for future configuration of your cluster.\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "RU: Разместите %s в каталоге с существующей конфигурацией ansible или в пустой каталог для создания конфигурации вашего кластера.\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	_ = noBrowser // browser auto-open is out of scope for v1

	log.Printf("YDB Installer version %s", Version)

	// Resolve the embedded Python+Ansible runtime. When bundled, Ansible runs
	// through it and no host dependencies are required. When not bundled (or
	// --use-system-deps), we fall back to the host's system tools.
	bundledDeps, runtimeInfo, ansibleRuntime := setupRuntime(*useSystemDeps)

	if *extractRuntimeOnly {
		if ansibleRuntime != nil {
			log.Printf("runtime extracted to %s", runtimeInfo.Path)
		} else {
			log.Printf("no bundled runtime to extract: %s", runtimeInfo.Error)
		}
		return
	}

	defaultLang := langdetect.DetectFromOS()
	store := session.New(defaultLang)

	// Create separate InstallState instances for each operation type.
	// This prevents state from leaking between flows (e.g. delete success
	// appearing as install success on screen 9).
	states := models.NewOperationStates()
	states.Install.AnsibleRuntime = ansibleRuntime
	states.DeleteYdb.AnsibleRuntime = ansibleRuntime

	cwd, _ := os.Getwd()

	// Write embedded ansible.cfg to the working directory if absent — this
	// guarantees Ansible always uses the correct defaults (host_key_checking
	// disabled, custom SSH args, pipelining, etc.) regardless of the CWD.
	if err := installerembed.WriteAnsibleCfg(cwd); err != nil {
		log.Printf("warning: cannot write embedded ansible.cfg: %v", err)
	}

	configWatcher := config.NewConfigWatcher()
	discovery := configWatcher.Discover(cwd)

	// Always set a default inventory path, even if the file doesn't exist yet.
	// The inventory writer will create it on the first PUT /api/inventory.
	if discovery.InventoryPath == "" {
		discovery.InventoryPath = filepath.Join(cwd, "inventory", "50-inventory.yaml")
	}

	if discovery.InventoryData != nil {
		store.SetInventory(discovery.InventoryData, discovery.InventoryFound, discovery.ConfigFound)
		log.Printf("inventory loaded from %s (%d hosts)", discovery.InventoryPath, len(discovery.InventoryData.Hosts))
	} else {
		log.Printf("no inventory file found, starting with empty session")
	}

	inventoryWriter := config.NewInventoryWriter()

	inventoryWSHub := handlers.NewInventoryWSHub()

	fileWatcher, err := config.NewInventoryFileWatcher(cwd)
	if err != nil {
		log.Printf("warning: cannot start inventory file watcher: %v", err)
	} else {
		changeCh := fileWatcher.AddListener()
		go func() {
			for range changeCh {
				log.Println("inventory file changed externally, reloading...")
				newDiscovery := configWatcher.Discover(cwd)
				if newDiscovery.InventoryData != nil {
					store.SetInventory(newDiscovery.InventoryData, newDiscovery.InventoryFound, newDiscovery.ConfigFound)
				}
				inventoryWSHub.Broadcast([]byte(`{"event":"inventory_changed"}`))
			}
		}()
		defer fileWatcher.Close()
	}

	spa, err := installerembed.FS()
	if err != nil {
		log.Printf("warning: cannot open embedded SPA: %v (serving dev placeholder)", err)
		spa = nil
	}

	// Set inventory path for Ansible operations. runAnsibleOperation reduces a
	// *.yaml path to its parent directory, so Ansible loads every inventory
	// file in that directory.
	states.Install.InventoryPath = discovery.InventoryPath
	states.DeleteYdb.InventoryPath = discovery.InventoryPath

	handler := api.New(api.Options{
		SessionStore:    store,
		OperationStates: states,
		SPA:             spa,
		InventoryWriter: inventoryWriter,
		InventoryWSHub:  inventoryWSHub,
		InventoryPath:   discovery.InventoryPath,
		ConfigPath:      filepath.Join(cwd, "files", "config.yaml"),
		BundledDeps:     bundledDeps,
		RuntimeInfo:     runtimeInfo,
	})

	addr := joinHostPort(*host, *port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Print the prominent startup banner to stdout (not the log stream) so the
	// user cannot miss the URL. The working directory is shown because cluster
	// data (inventory/, config.yaml, ansible.cfg) is written to the CWD, not
	// next to the binary — the binary is portable and reusable from any folder.
	banner.Print("http://"+addr, cwd)

	go func() {
		log.Printf("YDB Installer listening on http://%s (lang=%s)", addr, defaultLang)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for SIGINT / SIGTERM and shutdown gracefully.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Println("shutdown signal received, draining...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

// setupRuntime resolves the embedded Python+Ansible runtime and returns:
//   - bundledDeps: non-nil when deps should be reported as bundled
//   - runtimeInfo: a snapshot for GET /api/system/runtime
//   - ansibleRuntime: non-nil when Ansible should run through the bundle
//
// The third return is typed as ansible.Runtime (interface) so that returning
// nil in the system-deps / not-bundled paths produces a true nil interface
// value, not a typed nil pointer inside a non-nil interface
// (see Go "typed nil in interface" pitfall).
//
// When useSystem is true, or no runtime is bundled, all return values indicate
// the system-tools fallback.
func setupRuntime(useSystem bool) (*checker.BundledInfo, handlers.RuntimeInfo, ansible.Runtime) {
	mgr := runtime.NewManager()

	meta, err := mgr.Info()
	if err != nil {
		log.Printf("warning: cannot read runtime metadata: %v", err)
		return nil, handlers.RuntimeInfo{Embedded: false, Error: err.Error()}, nil
	}

	info := handlers.RuntimeInfo{
		Embedded:    meta.Embedded,
		Python:      meta.Python,
		Ansible:     meta.Ansible,
		AnsibleCore: meta.AnsibleCore,
		Collection:  meta.Collection,
	}

	if !meta.Embedded {
		log.Printf("no bundled runtime in this build — using system python/ansible")
		return nil, info, nil
	}

	if useSystem {
		log.Printf("--use-system-deps set — ignoring bundled runtime")
		return nil, info, nil
	}

	paths, err := mgr.Resolve()
	if err != nil {
		log.Printf("warning: cannot prepare bundled runtime: %v — falling back to system tools", err)
		info.Error = err.Error()
		return nil, info, nil
	}

	info.Extracted = true
	info.Path = paths.Root
	log.Printf("bundled runtime ready: python=%s ansible=%s ydb=%s (%s)",
		meta.Python, meta.Ansible, meta.Collection, paths.Root)

	bundled := &checker.BundledInfo{
		Python:      meta.Python,
		Ansible:     meta.Ansible,
		AnsibleCore: meta.AnsibleCore,
		Collection:  meta.Collection,
	}
	return bundled, info, paths
}

func joinHostPort(host string, port int) string {
	return host + ":" + itoa(port)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
