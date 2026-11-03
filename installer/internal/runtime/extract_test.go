package runtime

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func makeTarGz(t *testing.T, files map[string]string, exec map[string]bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for name, content := range files {
		mode := int64(0o644)
		if exec[name] {
			mode = 0o755
		}
		hdr := &tar.Header{
			Name:     name,
			Mode:     mode,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write body: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}

func TestExtractTarGz_RoundTrip(t *testing.T) {
	files := map[string]string{
		"python/bin/python3":          "#!fake interpreter",
		"python/bin/ansible-playbook": "fake playbook script",
		"collections/marker.txt":      "ok",
	}
	data := makeTarGz(t, files, map[string]bool{"python/bin/python3": true})

	dest := t.TempDir()
	if err := extractTarGz(bytes.NewReader(data), dest); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}

	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}

	info, err := os.Stat(filepath.Join(dest, "python/bin/python3"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("python3 should be executable, mode=%v", info.Mode())
	}
}

func TestSafeJoin_NeutralizesTraversal(t *testing.T) {
	base := t.TempDir()
	cases := []string{"../escape", "../../etc/passwd", "/abs/path", "python/bin/python3"}
	for _, name := range cases {
		got, err := safeJoin(base, name)
		if err != nil {
			t.Fatalf("safeJoin(%q) unexpected error: %v", name, err)
		}
		if got != base && !bytes.HasPrefix([]byte(got), []byte(base+string(os.PathSeparator))) {
			t.Errorf("safeJoin(%q) = %q escaped base %q", name, got, base)
		}
	}
}

func TestExtractTarGz_NeutralizesTraversalEntry(t *testing.T) {
	data := makeTarGz(t, map[string]string{"../evil.txt": "x"}, nil)
	dest := t.TempDir()
	if err := extractTarGz(bytes.NewReader(data), dest); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "evil.txt")); err != nil {
		t.Errorf("traversal entry should be contained in dest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "evil.txt")); err == nil {
		t.Error("traversal entry escaped dest into parent directory")
	}
}

func TestSHA256Hex_Stable(t *testing.T) {
	h1, err := sha256Hex(bytes.NewReader([]byte("hello")))
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := sha256Hex(bytes.NewReader([]byte("hello")))
	if h1 != h2 {
		t.Errorf("hash not stable: %s != %s", h1, h2)
	}
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if h1 != want {
		t.Errorf("sha256(hello) = %s, want %s", h1, want)
	}
}
