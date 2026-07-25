package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

const folderPerm = 0o755

// EnsureFolder makes sure the output folder exists, reusing it if it already
// does. Runs are not archived into "<name>_vN" copies: each step overwrites its
// own output file, so paths stay stable across reruns (deliverables included)
// and an existing folder is what a future resume can build on.
func EnsureFolder(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path is not absolute %s", path)
	}

	return createFolder(path)
}

func createFolder(path string) error {
	if err := os.MkdirAll(path, folderPerm); err != nil {
		return fmt.Errorf("failed to create folder %s: %w", path, err)
	}
	return nil
}
