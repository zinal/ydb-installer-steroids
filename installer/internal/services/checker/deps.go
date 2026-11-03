// Package checker provides system dependency verification for the YDB installer.
package checker

import (
	"os/exec"
	"regexp"
	"strings"
)

const (
	SourceBundled = "bundled"
	SourceSystem  = "system"
	SourceMissing = "missing"
)

// Dependency describes a single system dependency.
type Dependency struct {
	Name      string `json:"name"`
	OK        bool   `json:"ok"`
	Version   string `json:"version,omitempty"`
	Required  string `json:"required,omitempty"`
	Installed bool   `json:"installed"`
	Source    string `json:"source,omitempty"`
}

type BundledInfo struct {
	Python      string
	Ansible     string
	AnsibleCore string
	Collection  string
}

type DepsChecker struct {
	Bundled *BundledInfo
}

// Check runs all dependency probes. When a bundled runtime is configured,
// dependencies are reported as satisfied by the bundle. In system mode,
// python, ansible, and the ydb_platform.ydb collection are probed on the host.
func (c *DepsChecker) Check() []Dependency {
	if c.Bundled != nil {
		return c.bundledDeps()
	}
	return []Dependency{
		c.checkPython(),
		c.checkAnsible(),
		c.checkAnsibleCollection("ydb_platform.ydb"),
	}
}

func (c *DepsChecker) bundledDeps() []Dependency {
	b := c.Bundled
	ansibleVer := b.Ansible
	if ansibleVer == "" {
		ansibleVer = b.AnsibleCore
	}
	return []Dependency{
		{Name: "python", OK: true, Installed: true, Required: ">=3.8", Version: b.Python, Source: SourceBundled},
		{Name: "ansible", OK: true, Installed: true, Required: ">=2.12", Version: ansibleVer, Source: SourceBundled},
		{Name: "ydb_platform.ydb", OK: true, Installed: true, Required: "any", Version: b.Collection, Source: SourceBundled},
	}
}

func (c *DepsChecker) checkPython() Dependency {
	dep := Dependency{
		Name:     "python",
		Required: ">=3.8",
	}

	out, err := exec.Command("python3", "--version").CombinedOutput()
	if err != nil {
		out, err = exec.Command("python", "--version").CombinedOutput()
		if err != nil {
			dep.OK = false
			dep.Installed = false
			dep.Source = SourceMissing
			return dep
		}
	}

	version := ExtractVersion(string(out))
	dep.Version = version
	dep.Installed = version != ""
	dep.OK = dep.Installed && MeetsMinVersion(version, "3.8")
	dep.Source = sourceFor(dep.Installed)
	return dep
}

func (c *DepsChecker) checkAnsible() Dependency {
	dep := Dependency{
		Name:     "ansible",
		Required: ">=2.12",
	}

	out, err := exec.Command("ansible", "--version").CombinedOutput()
	if err != nil {
		dep.OK = false
		dep.Installed = false
		dep.Source = SourceMissing
		return dep
	}

	version := ExtractVersion(string(out))
	dep.Version = version
	dep.Installed = version != ""
	dep.OK = dep.Installed && MeetsMinVersion(version, "2.12")
	dep.Source = sourceFor(dep.Installed)
	return dep
}

func (c *DepsChecker) checkAnsibleCollection(name string) Dependency {
	dep := Dependency{
		Name:     name,
		Required: "any",
	}

	out, err := exec.Command("ansible-galaxy", "collection", "list", name).CombinedOutput()
	if err != nil {
		dep.OK = false
		dep.Installed = false
		dep.Source = SourceMissing
		return dep
	}

	text := string(out)
	// Match the FQCN as the first field on a line so a shorter name does not
	// match a longer collection that shares its prefix.
	line := collectionListLine(text, name)
	if line == "" || strings.Contains(strings.ToLower(text), "unable to find") {
		dep.OK = false
		dep.Installed = false
		dep.Source = SourceMissing
		return dep
	}

	version := ExtractVersion(line)
	dep.Version = version
	dep.Installed = true
	dep.OK = true
	dep.Source = SourceSystem
	return dep
}

func collectionListLine(text, name string) string {
	for _, raw := range strings.Split(text, "\n") {
		fields := strings.Fields(raw)
		if len(fields) >= 1 && fields[0] == name {
			return raw
		}
	}
	return ""
}

func sourceFor(installed bool) string {
	if installed {
		return SourceSystem
	}
	return SourceMissing
}

var versionRe = regexp.MustCompile(`(\d+\.\d+(?:\.\d+)?)`)

// ExtractVersion pulls the first semver-like token from text.
func ExtractVersion(text string) string {
	return versionRe.FindString(text)
}

// MeetsMinVersion returns true if version >= min using major.minor comparison.
func MeetsMinVersion(version, min string) bool {
	vParts := splitVersion(version)
	mParts := splitVersion(min)

	for i := 0; i < len(mParts) && i < len(vParts); i++ {
		if vParts[i] > mParts[i] {
			return true
		}
		if vParts[i] < mParts[i] {
			return false
		}
	}
	return len(vParts) >= len(mParts)
}

func splitVersion(v string) []int {
	parts := strings.Split(v, ".")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		n := 0
		for _, ch := range p {
			if ch >= '0' && ch <= '9' {
				n = n*10 + int(ch-'0')
			} else {
				break
			}
		}
		result = append(result, n)
	}
	return result
}
