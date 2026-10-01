package builder

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// Render produces the contents of a Nix machines file for the given builders,
// each connected to via sshKeyPath.
func Render(builders []*Builder, sshKeyPath string) []byte {
	lines := make([]string, 0, len(builders))
	for _, b := range builders {
		lines = append(lines, b.MachinesLine(sshKeyPath))
	}

	data := strings.Join(lines, "\n")
	if data != "" {
		data += "\n"
	}
	return []byte(data)
}

// WriteFileAtomic atomically replaces path's contents with data, skipping the
// write entirely if the file already has the same contents.
func WriteFileAtomic(path string, data []byte) error {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		return nil
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, filepath.Clean(path))
}
