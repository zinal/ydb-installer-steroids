package checker

import "strings"

const (
	OSUbuntu = "ubuntu"
	OSCentOS = "centos"
	OSMacOS  = "macos"
)

// Git sources for collections not published on Ansible Galaxy.
// Keep in sync with scripts/runtime.lock.
const (
	ydbCollectionGit = "git+https://github.com/ydb-platform/ydb-ansible"
)

type InstructionProvider struct{}

func (p *InstructionProvider) ForDep(dep, os string) string {
	os = strings.ToLower(strings.TrimSpace(os))
	dep = strings.ToLower(strings.TrimSpace(dep))

	switch dep {
	case "python":
		return p.pythonInstructions(os)
	case "ansible":
		return p.ansibleInstructions(os)
	case "ydb_platform.ydb":
		return p.ydbCollectionInstructions(os)
	}
	return ""
}

func (p *InstructionProvider) AllInstructions(os string) map[string]string {
	deps := []string{"python", "ansible", "ydb_platform.ydb"}
	result := make(map[string]string, len(deps))
	for _, dep := range deps {
		result[dep] = p.ForDep(dep, os)
	}
	return result
}

func (p *InstructionProvider) pythonInstructions(os string) string {
	switch os {
	case OSUbuntu:
		return strings.TrimSpace(`
# Ubuntu / Debian
sudo apt update
sudo apt install -y python3 python3-pip`)
	case OSCentOS:
		return strings.TrimSpace(`
# CentOS / RHEL
sudo yum install -y python3 python3-pip`)
	case OSMacOS:
		return strings.TrimSpace(`
# macOS (Homebrew)
brew install python3`)
	}
	return ""
}

func (p *InstructionProvider) ansibleInstructions(os string) string {
	switch os {
	case OSUbuntu:
		return strings.TrimSpace(`
# Ubuntu / Debian
sudo apt update
sudo apt install -y python3-pip
pip3 install "ansible>=2.12"`)
	case OSCentOS:
		return strings.TrimSpace(`
# CentOS / RHEL
sudo yum install -y python3-pip
pip3 install "ansible>=2.12"`)
	case OSMacOS:
		return strings.TrimSpace(`
# macOS (Homebrew)
brew install python3
pip3 install "ansible>=2.12"`)
	}
	return ""
}

func (p *InstructionProvider) ydbCollectionInstructions(os string) string {
	_ = os
	return strings.TrimSpace(`
# Requires Ansible to be installed first
# ydb_platform.ydb is installed from git (not Ansible Galaxy)
ansible-galaxy collection install ` + ydbCollectionGit)
}
