package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

// NOTE ON ASSET STATE
//
// The runtime archive under assets/ has two legitimate states:
//
//   - placeholder  — what is committed to git (`embedded:false`, empty sha256)
//   - real runtime — produced locally by `make runtime` (`embedded:true`)
//
// Both are valid working trees, so these tests must not hardcode either one.
// They assert the invariants that hold in BOTH states and branch on
// meta.Embedded only where the two genuinely differ. Hardcoding "placeholder"
// used to make `make test` fail for any developer who had run `make runtime`.

func TestReadMetadata_Defaults(t *testing.T) {
	meta, err := ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}

	if meta.CollectionsSubdir != "collections" {
		t.Errorf("CollectionsSubdir default = %q", meta.CollectionsSubdir)
	}
	if meta.RootSubdir != "python" {
		t.Errorf("RootSubdir default = %q", meta.RootSubdir)
	}

	// Internal consistency: a real bundled runtime must carry the digest that
	// acts as its cache key and integrity check; a placeholder must not claim
	// to be embedded.
	if meta.Embedded {
		if meta.SHA256 == "" {
			t.Error("embedded runtime must carry a sha256 digest")
		}
		if meta.Python == "" {
			t.Error("embedded runtime must report a python version")
		}
	} else if meta.SHA256 != "" {
		t.Errorf("placeholder should have an empty sha256, got %q", meta.SHA256)
	}
}

func TestManager_Info(t *testing.T) {
	m := NewManager()
	meta, err := m.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	want, err := ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if meta.Embedded != want.Embedded {
		t.Errorf("Embedded = %v, want %v", meta.Embedded, want.Embedded)
	}
	if meta.SHA256 != want.SHA256 {
		t.Errorf("SHA256 = %q, want %q", meta.SHA256, want.SHA256)
	}

	meta2, _ := m.Info()
	if meta2 != meta {
		t.Error("Info should cache metadata")
	}
}

func TestManager_Resolve_NotBundled(t *testing.T) {
	t.Setenv(envRuntimeDir, "")

	meta, err := ReadMetadata()
	if err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if meta.Embedded {
		// A real runtime is bundled, so Resolve() correctly extracts instead of
		// reporting ErrNotBundled. The fallback path under test only exists for
		// placeholder builds — run `make runtime-clean` to exercise it.
		t.Skip("real runtime is bundled; ErrNotBundled path not applicable")
	}

	m := NewManager()
	if _, err := m.Resolve(); err != ErrNotBundled {
		t.Fatalf("Resolve() = %v, want ErrNotBundled", err)
	}
}

func fakeRuntimeTree(t *testing.T, root string) {
	t.Helper()
	bin := filepath.Join(root, "python", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"python3", "ansible-playbook", "ansible-galaxy"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!stub"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "collections"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestManager_Resolve_ExternalDir(t *testing.T) {
	dir := t.TempDir()
	// External dir points directly at the python root (per envRuntimeDir
	// semantics in Resolve: root = dir, collections = dir/collections_subdir).
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"python3", "ansible-playbook", "ansible-galaxy"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!stub"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "collections"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envRuntimeDir, dir)
	m := NewManager()
	p, err := m.Resolve()
	if err != nil {
		t.Fatalf("Resolve with external dir: %v", err)
	}
	if p.Root != dir {
		t.Errorf("Root = %q, want %q", p.Root, dir)
	}
	if !isUsableRuntime(p) {
		t.Error("external runtime should be usable")
	}

	p2, _ := m.Resolve()
	if p2 != p {
		t.Error("Resolve should memoize result")
	}
}

func TestManager_Resolve_ExternalDir_Invalid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envRuntimeDir, dir)
	m := NewManager()
	if _, err := m.Resolve(); err == nil {
		t.Fatal("expected error for invalid external runtime dir")
	}
}

func TestIsUsableRuntime(t *testing.T) {
	root := t.TempDir()
	fakeRuntimeTree(t, root)
	p := newPaths(filepath.Join(root, "python"), filepath.Join(root, "collections"))
	if !isUsableRuntime(p) {
		t.Error("complete tree should be usable")
	}

	_ = os.Remove(p.AnsibleGalaxy)
	if isUsableRuntime(p) {
		t.Error("missing ansible-galaxy should make runtime unusable")
	}
}

func TestDirWritable(t *testing.T) {
	dir := t.TempDir()
	if !dirWritable(dir) {
		t.Error("temp dir should be writable")
	}
}

func TestVerifyArchiveSHA(t *testing.T) {
	ok, got, want, err := VerifyArchiveSHA()
	if err != nil {
		t.Fatalf("VerifyArchiveSHA: %v", err)
	}

	if got == "" {
		t.Error("got sha should be non-empty (archive exists)")
	}

	if want == "" {
		// Placeholder: nothing to compare against, so it cannot verify.
		if ok {
			t.Error("placeholder (empty want) should not verify as ok")
		}
		return
	}

	// Real runtime: the committed digest MUST match the archive on disk,
	// otherwise Resolve() would refuse to extract it at runtime.
	if !ok {
		t.Errorf("bundled archive failed integrity check: got %s want %s", got, want)
	}
}

func TestEqualBytes(t *testing.T) {
	if !equalBytes([]byte("a"), []byte("a")) {
		t.Error("equalBytes should be true for equal slices")
	}
	if equalBytes([]byte("a"), []byte("b")) {
		t.Error("equalBytes should be false for different slices")
	}
}
