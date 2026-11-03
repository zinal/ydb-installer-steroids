package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ErrNotBundled is returned by Resolve when the binary was built without a real
// runtime archive (the committed placeholder). Callers should fall back to the
// host's system python3/ansible.
var ErrNotBundled = errors.New("runtime: no bundled runtime archive (using system tools)")

// envRuntimeDir lets operators point at an external, pre-extracted runtime
// directory. When set and valid, it wins over the embedded archive.
const envRuntimeDir = "YDB_INSTALLER_RUNTIME_DIR"

const envCacheDir = "YDB_INSTALLER_RUNTIME_CACHE"

const cacheDirName = ".ydb-installer-runtime"

// extractedMarker is written into a version dir once extraction fully succeeds,
// making re-resolution idempotent and crash-safe (partial dirs are ignored).
const extractedMarker = ".extracted.ok"

type Manager struct {
	mu       sync.Mutex
	resolved *Paths
	meta     *Metadata
}

func NewManager() *Manager { return &Manager{} }

func (m *Manager) Info() (*Metadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.meta != nil {
		return m.meta, nil
	}
	meta, err := ReadMetadata()
	if err != nil {
		return nil, err
	}
	m.meta = meta
	return meta, nil
}

// Resolve locates a usable runtime, extracting the embedded archive on first
// call. Safe for concurrent use and idempotent across launches.
//
// Resolution order:
//  1. YDB_INSTALLER_RUNTIME_DIR (external pre-extracted runtime), if valid.
//  2. Already-extracted version dir in the cache (marker present), if valid.
//  3. Extract the embedded archive into a fresh version dir.
//
// Returns ErrNotBundled if the archive is the placeholder (embedded:false).
func (m *Manager) Resolve() (*Paths, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.resolved != nil {
		return m.resolved, nil
	}

	meta, err := ReadMetadata()
	if err != nil {
		return nil, fmt.Errorf("read runtime metadata: %w", err)
	}
	m.meta = meta

	if dir := os.Getenv(envRuntimeDir); dir != "" {
		p := newPaths(dir, filepath.Join(dir, meta.CollectionsSubdir))
		if isUsableRuntime(p) {
			m.resolved = p
			return p, nil
		}
		return nil, fmt.Errorf("runtime: %s=%q is not a usable runtime", envRuntimeDir, dir)
	}

	if !meta.Embedded {
		return nil, ErrNotBundled
	}

	versionDir, err := m.versionDir(meta)
	if err != nil {
		return nil, err
	}

	root := filepath.Join(versionDir, meta.RootSubdir)
	collections := filepath.Join(versionDir, meta.CollectionsSubdir)
	p := newPaths(root, collections)

	if fileExists(filepath.Join(versionDir, extractedMarker)) && isUsableRuntime(p) {
		m.resolved = p
		return p, nil
	}

	if err := m.extractEmbedded(versionDir, meta); err != nil {
		return nil, err
	}
	if !isUsableRuntime(p) {
		return nil, fmt.Errorf("runtime: extracted tree is missing expected files under %s", root)
	}

	m.resolved = p
	return p, nil
}

// versionDir prefers a directory next to the binary (portable / air-gap friendly)
// and falls back to XDG cache if that is not writable.
func (m *Manager) versionDir(meta *Metadata) (string, error) {
	key := meta.SHA256
	if key == "" {
		key = "unknown"
	}
	if len(key) > 16 {
		key = key[:16]
	}

	if override := os.Getenv(envCacheDir); override != "" {
		return filepath.Join(override, key), nil
	}

	if exe, err := os.Executable(); err == nil {
		base := filepath.Join(filepath.Dir(exe), cacheDirName, key)
		if dirWritable(filepath.Dir(filepath.Dir(base))) {
			return base, nil
		}
	}

	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("runtime: no writable cache dir: %w", err)
	}
	return filepath.Join(cache, "ydb-installer", "runtime", key), nil
}

// extractEmbedded verifies archive integrity, extracts to a sibling temp dir,
// then atomically renames it into place.
func (m *Manager) extractEmbedded(versionDir string, meta *Metadata) error {
	if meta.SHA256 != "" {
		got, err := archiveSHA256()
		if err != nil {
			return fmt.Errorf("hash runtime archive: %w", err)
		}
		if got != meta.SHA256 {
			return fmt.Errorf("runtime: archive sha256 mismatch (have %s want %s)", got, meta.SHA256)
		}
	}

	parent := filepath.Dir(versionDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp(parent, ".extract-*")
	if err != nil {
		return err
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(tmp)
		}
	}()

	rc, err := archiveReader()
	if err != nil {
		return err
	}
	defer rc.Close()

	if err := extractTarGz(rc, tmp); err != nil {
		return fmt.Errorf("extract runtime: %w", err)
	}

	if err := os.WriteFile(filepath.Join(tmp, extractedMarker), []byte(meta.SHA256), 0o644); err != nil {
		return err
	}

	// If another process won the race and already created versionDir, keep it.
	if fileExists(filepath.Join(versionDir, extractedMarker)) {
		success = true
		return nil
	}
	if err := os.Rename(tmp, versionDir); err != nil {
		// Lost the race — treat as success if marker present.
		if fileExists(filepath.Join(versionDir, extractedMarker)) {
			success = true
			return nil
		}
		return fmt.Errorf("install runtime dir: %w", err)
	}
	success = true
	return nil
}

func isUsableRuntime(p *Paths) bool {
	return fileExists(p.Python) &&
		fileExists(p.AnsiblePlaybook) &&
		fileExists(p.AnsibleGalaxy)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirWritable(dir string) bool {
	for d := dir; d != "" && d != string(os.PathSeparator); d = filepath.Dir(d) {
		info, err := os.Stat(d)
		if err == nil {
			return info.IsDir() && unixWritable(d)
		}
	}
	return unixWritable(string(os.PathSeparator))
}

func unixWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".wtest-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// VerifyArchiveSHA recomputes the embedded archive digest and compares it to
// the metadata. Exposed for tests and a future `--verify-runtime` flag.
func VerifyArchiveSHA() (ok bool, got, want string, err error) {
	meta, err := ReadMetadata()
	if err != nil {
		return false, "", "", err
	}
	got, err = archiveSHA256()
	if err != nil {
		return false, "", meta.SHA256, err
	}
	return got == meta.SHA256, got, meta.SHA256, nil
}

func equalBytes(a, b []byte) bool { return bytes.Equal(a, b) }
