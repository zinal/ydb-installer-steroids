package api

import (
	"io/fs"
	"net/http"
	"strings"

	"ydb-installer/internal/api/handlers"
	"ydb-installer/internal/models"
	"ydb-installer/internal/services/checker"
	"ydb-installer/internal/services/config"
	"ydb-installer/internal/services/session"
)

type Options struct {
	SessionStore *session.Store
	// OperationStates holds separate InstallState instances per flow so that
	// state never leaks between flows.
	OperationStates *models.OperationStates
	SPA             fs.FS
	InventoryWriter *config.InventoryWriter
	InventoryWSHub  *handlers.InventoryWSHub
	InventoryPath   string
	ConfigPath      string
	// BundledDeps, when non-nil, makes /api/system/deps report the embedded
	// runtime as the source of python/ansible/collection.
	BundledDeps *checker.BundledInfo
	RuntimeInfo handlers.RuntimeInfo
}

// New routes /api/* to JSON handlers, /ws/* to WebSocket handlers, and
// everything else to SPA static assets with a fallback to index.html.
func New(opts Options) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/session", handlers.Session(opts.SessionStore))

	if opts.InventoryWriter != nil {
		mux.HandleFunc("/api/inventory", handlers.Inventory(opts.SessionStore, opts.InventoryWriter))
	}
	if opts.InventoryWSHub != nil {
		mux.HandleFunc("/ws/inventory", opts.InventoryWSHub.InventoryWebSocket())
	}

	mux.HandleFunc("/api/hosts/check", handlers.HandleCheckHosts)

	mux.HandleFunc("/api/disks/check", handlers.HandleCheckDisks)
	mux.HandleFunc("/api/disks/validate", handlers.HandleValidateDisks)

	mux.HandleFunc("/api/versions", handlers.HandleVersions)
	mux.HandleFunc("/api/versions/validate-file", handlers.HandleValidateFile)

	mux.HandleFunc("/api/tls/generate", handlers.HandleTLGenerate)
	mux.HandleFunc("/api/tls/validate", handlers.HandleTLValidate)
	mux.HandleFunc("/api/tls/exists", handlers.HandleTLSExists)

	mux.HandleFunc("/api/system/deps", handlers.SystemDeps(opts.BundledDeps))
	mux.HandleFunc("/api/system/deps/instruction/", handlers.DepsInstruction())
	mux.HandleFunc("/api/system/configs", handlers.SystemConfigs())
	mux.HandleFunc("/api/system/runtime", handlers.SystemRuntime(opts.RuntimeInfo))
	mux.HandleFunc("/api/system/connectivity", handlers.HandleConnectivity)

	if opts.OperationStates != nil {
		mux.HandleFunc("/api/install/ydb/config-status", handlers.ConfigStatus(opts.SessionStore, opts.InventoryPath, opts.ConfigPath))
		mux.HandleFunc("/api/install/ydb/generate-config", handlers.GenerateConfig(opts.SessionStore, opts.InventoryPath, opts.ConfigPath))
		mux.HandleFunc("/api/install/ydb/start", handlers.YdbStart(opts.OperationStates, opts.SessionStore))
		mux.HandleFunc("/api/install/ydb/create-database", handlers.CreateDatabase(opts.OperationStates))
		mux.HandleFunc("/api/install/delete-ydb/start", handlers.DeleteYdbStart(opts.OperationStates))
		mux.HandleFunc("/api/install/status", handlers.InstallStatus(opts.OperationStates, opts.SessionStore))
		mux.HandleFunc("/api/install/cancel", handlers.InstallCancel(opts.OperationStates, opts.SessionStore))
		mux.HandleFunc("/api/install/logs", handlers.GetLogs(opts.OperationStates, opts.SessionStore))
	}

	mux.Handle("/api/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))

	if opts.OperationStates != nil {
		mux.HandleFunc("/ws/logs", handlers.LogsWebSocket(opts.OperationStates, opts.SessionStore))
	}

	mux.Handle("/", spaHandler(opts.SPA))
	return withLogging(mux)
}

// spaHandler falls back to index.html for missing files so Vue Router can
// take over on hard refresh / direct URLs.
func spaHandler(spa fs.FS) http.Handler {
	if spa == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(devPlaceholder))
		})
	}
	files := http.FS(spa)
	fileServer := http.FileServer(files)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" {
			clean = "index.html"
		}
		f, err := spa.Open(clean)
		if err != nil {
			if idx, err := spa.Open("index.html"); err == nil {
				_ = idx.Close()
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/"
				fileServer.ServeHTTP(w, r2)
				return
			}
			http.Error(w, "SPA not built", http.StatusNotFound)
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

const devPlaceholder = `<!doctype html>
<html><head><meta charset="utf-8"><title>YDB Installer</title></head>
<body style="font-family:system-ui;padding:2rem;color:#102a43">
<h1 style="color:#1f6dd9">YDB Installer — backend is up</h1>
<p>The embedded SPA bundle is empty. Build the frontend with
<code>cd ui &amp;&amp; npm install &amp;&amp; npm run build</code> or use the dev server
(<code>make dev</code>).</p>
<ul>
<li><a href="/api/session">/api/session</a></li>
<li><a href="/api/system/deps">/api/system/deps</a></li>
<li><a href="/api/system/configs">/api/system/configs</a></li>
<li><a href="/api/install/status">/api/install/status</a></li>
<li><a href="/api/healthz">/api/healthz</a></li>
</ul></body></html>`
