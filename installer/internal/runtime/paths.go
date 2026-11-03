// Package runtime manages the embedded, self-contained Ansible control-node
// runtime (Python + Ansible + the ydb_platform.ydb collection).
//
// The installer ships this runtime inside the binary. On first launch it is
// extracted to a version-stamped cache directory next to the binary, and all
// Ansible/Python invocations are pointed at it. When the binary is built
// without a real runtime archive (placeholder, embedded:false), Resolve
// returns ErrNotBundled and callers fall back to the host's system tools.
package runtime

import (
	"os"
	"path/filepath"
	"strings"
)

type Paths struct {
	Root            string
	Python          string
	AnsiblePlaybook string
	AnsibleGalaxy   string
	CollectionsPath string
}

func newPaths(root, collectionsPath string) *Paths {
	bin := filepath.Join(root, "bin")
	return &Paths{
		Root:            root,
		Python:          filepath.Join(bin, "python3"),
		AnsiblePlaybook: filepath.Join(bin, "ansible-playbook"),
		AnsibleGalaxy:   filepath.Join(bin, "ansible-galaxy"),
		CollectionsPath: collectionsPath,
	}
}

func (p *Paths) PythonPath() string { return p.Python }

func (p *Paths) AnsiblePlaybookPath() string { return p.AnsiblePlaybook }

// ExtraEnv returns only the runtime-specific environment overrides (no host
// environment). Callers typically do: append(os.Environ(), p.ExtraEnv()...).
//
// We deliberately invoke scripts as "<python> <script>" so the absolute shebang
// baked into the console scripts is bypassed and the runtime stays relocatable.
func (p *Paths) ExtraEnv() []string {
	bin := filepath.Join(p.Root, "bin")
	env := []string{
		"PATH=" + bin + pathListSep + envPath(),
		"PYTHONHOME=" + p.Root,
		"ANSIBLE_COLLECTIONS_PATH=" + p.CollectionsPath,
	}
	return env
}

func (p *Paths) Env() []string { return p.ExtraEnv() }

// envPath returns the current process PATH so the bundled bin/ is prepended
// rather than replacing system tools like ssh.
var envPath = func() string {
	if v := os.Getenv("PATH"); v != "" {
		return v
	}
	return "/usr/local/bin:/usr/bin:/bin"
}

const pathListSep = ":"

func sanitizeVersion(s string) string { return strings.TrimSpace(s) }
