package runtime

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPaths_Layout(t *testing.T) {
	root := "/opt/rt/python"
	collections := "/opt/rt/collections"
	p := newPaths(root, collections)

	if p.Python != filepath.Join(root, "bin", "python3") {
		t.Errorf("Python = %q", p.Python)
	}
	if p.AnsiblePlaybook != filepath.Join(root, "bin", "ansible-playbook") {
		t.Errorf("AnsiblePlaybook = %q", p.AnsiblePlaybook)
	}
	if p.AnsibleGalaxy != filepath.Join(root, "bin", "ansible-galaxy") {
		t.Errorf("AnsibleGalaxy = %q", p.AnsibleGalaxy)
	}
	if p.CollectionsPath != collections {
		t.Errorf("CollectionsPath = %q", p.CollectionsPath)
	}

	if p.PythonPath() != p.Python {
		t.Error("PythonPath mismatch")
	}
	if p.AnsiblePlaybookPath() != p.AnsiblePlaybook {
		t.Error("AnsiblePlaybookPath mismatch")
	}
}

func TestExtraEnv_ContainsExpectedKeys(t *testing.T) {
	p := newPaths("/opt/rt/python", "/opt/rt/collections")
	env := p.ExtraEnv()

	joined := strings.Join(env, "\n")
	mustContain := []string{
		"PATH=",
		"/opt/rt/python/bin",
		"PYTHONHOME=/opt/rt/python",
		"ANSIBLE_COLLECTIONS_PATH=/opt/rt/collections",
	}
	for _, frag := range mustContain {
		if !strings.Contains(joined, frag) {
			t.Errorf("ExtraEnv missing %q; got:\n%s", frag, joined)
		}
	}

	if strings.Join(p.Env(), "\n") != joined {
		t.Error("Env() should equal ExtraEnv()")
	}
}

func TestExtraEnv_PrependsBinToPath(t *testing.T) {
	p := newPaths("/opt/rt/python", "/opt/rt/collections")
	for _, e := range p.ExtraEnv() {
		if strings.HasPrefix(e, "PATH=") {
			val := strings.TrimPrefix(e, "PATH=")
			if !strings.HasPrefix(val, "/opt/rt/python/bin"+pathListSep) &&
				val != "/opt/rt/python/bin" {
				t.Errorf("bundled bin not prepended to PATH: %q", val)
			}
			return
		}
	}
	t.Fatal("no PATH entry in ExtraEnv")
}

func TestSanitizeVersion(t *testing.T) {
	if got := sanitizeVersion("  3.12.5  \n"); got != "3.12.5" {
		t.Errorf("sanitizeVersion = %q", got)
	}
}
