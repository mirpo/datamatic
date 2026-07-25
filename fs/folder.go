package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

const folderPerm = 0o755

// EnsureFolder creates a folder for generated output if it does not exist yet,
// and is a no-op when it does — reruns reuse the folder in place rather than
// archiving it into a "<name>_vN" copy, so output paths stay stable. The path
// must be absolute: every folder here is derived from the resolved output
// folder, so a relative one means a caller skipped that resolution.
func EnsureFolder(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path is not absolute %s", path)
	}

	if err := os.MkdirAll(path, folderPerm); err != nil {
		return fmt.Errorf("failed to create folder %s: %w", path, err)
	}
	return nil
}
