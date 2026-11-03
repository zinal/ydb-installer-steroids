package runtime

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxDecompressedSize caps total extracted bytes to guard against malformed or
// malicious archives (defense in depth — the archive is our own build output).
const maxDecompressedSize = 2 << 30 // 2 GiB

// extractTarGz extracts a gzip-compressed tar stream into destDir.
// Path traversal (".." escapes, absolute paths) is rejected.
func extractTarGz(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var written int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}

		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			n, err := writeFile(target, tr, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			written += n
			if written > maxDecompressedSize {
				return fmt.Errorf("archive exceeds max decompressed size")
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := safeSymlink(hdr.Linkname, target, destDir); err != nil {
				return err
			}
		default:
		}
	}
	return nil
}

func writeFile(target string, src io.Reader, mode os.FileMode) (int64, error) {
	if mode == 0 {
		mode = 0o644
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, src)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return n, err
}

func safeJoin(base, name string) (string, error) {
	clean := filepath.Clean("/" + name) // force-absolute then strip leading /
	target := filepath.Join(base, clean)
	if target != base && !strings.HasPrefix(target, base+string(os.PathSeparator)) {
		return "", fmt.Errorf("illegal path in archive: %q", name)
	}
	return target, nil
}

func safeSymlink(linkname, linkPath, base string) error {
	var resolved string
	if filepath.IsAbs(linkname) {
		resolved = filepath.Clean(linkname)
	} else {
		resolved = filepath.Clean(filepath.Join(filepath.Dir(linkPath), linkname))
	}
	if resolved != base && !strings.HasPrefix(resolved, base+string(os.PathSeparator)) {
		return fmt.Errorf("illegal symlink target: %q -> %q", linkPath, linkname)
	}
	_ = os.Remove(linkPath) // overwrite if exists
	return os.Symlink(linkname, linkPath)
}

func sha256Hex(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
