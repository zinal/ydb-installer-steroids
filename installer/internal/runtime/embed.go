package runtime

import (
	"bytes"
	"embed"
	"encoding/json"
	"io"
)

// assetsFS embeds the runtime archive and its version metadata.
// Both files are always present: runtime.version (JSON metadata) and
// runtime-linux-amd64.tar.gz. Before `make runtime`, the placeholder archive
// is tiny and runtime.version has "embedded": false, so Resolve() reports the
// runtime as not bundled and callers fall back to system tools.
//
//go:embed assets/runtime.version
//go:embed assets/runtime-linux-amd64.tar.gz
var assetsFS embed.FS

const (
	versionAsset = "assets/runtime.version"
	archiveAsset = "assets/runtime-linux-amd64.tar.gz"
)

type Metadata struct {
	Embedded    bool   `json:"embedded"`
	Python      string `json:"python"`
	Ansible     string `json:"ansible"`
	AnsibleCore string `json:"ansible_core"`
	Collection  string `json:"collection"`
	// EMCollection is required by the EM install flow (screen 24). Empty on
	// placeholder builds and on runtimes built before the EM collection was
	// bundled.
	EMCollection      string `json:"em_collection"`
	SHA256            string `json:"sha256"`
	CollectionsSubdir string `json:"collections_subdir"`
	RootSubdir        string `json:"root_subdir"`
}

func ReadMetadata() (*Metadata, error) {
	b, err := assetsFS.ReadFile(versionAsset)
	if err != nil {
		return nil, err
	}
	var m Metadata
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.CollectionsSubdir == "" {
		m.CollectionsSubdir = "collections"
	}
	if m.RootSubdir == "" {
		m.RootSubdir = "python"
	}
	return &m, nil
}

func archiveReader() (io.ReadCloser, error) {
	return assetsFS.Open(archiveAsset)
}

func archiveBytes() ([]byte, error) {
	return assetsFS.ReadFile(archiveAsset)
}

func archiveSHA256() (string, error) {
	b, err := archiveBytes()
	if err != nil {
		return "", err
	}
	return sha256Hex(bytes.NewReader(b))
}
