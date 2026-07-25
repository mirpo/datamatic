package fs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureFolder pins the rerun contract: the output folder is reused in
// place, never rotated to a "_vN" copy. Steps overwrite their own files, so a
// rerun keeps stable paths (deliverables included) and leaves room for resume.
func TestEnsureFolder(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_folder")

	require.NoError(t, EnsureFolder(path), "first call creates the folder")
	assert.DirExists(t, path)

	// a previous run left a file behind
	leftover := filepath.Join(path, "step.jsonl")
	require.NoError(t, os.WriteFile(leftover, []byte("{}\n"), 0o644))

	require.NoError(t, EnsureFolder(path), "second call reuses the folder")

	assert.DirExists(t, path)
	assert.NoDirExists(t, filepath.Join(tmpDir, "test_folder_v1"), "must not rotate the folder to a versioned copy")
	assert.FileExists(t, leftover, "reusing the folder must not wipe it; steps overwrite their own files")
}

func TestEnsureFolder_RejectsRelativePath(t *testing.T) {
	assert.Error(t, EnsureFolder("relative/path"), "the output folder is resolved to an absolute path before use")
}

func TestEnsureFolder_CreationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	var pathToCreate string

	if runtime.GOOS == "windows" {
		invalidFolderName := "test_folder*"
		pathToCreate = filepath.Join(tmpDir, invalidFolderName)
	} else {
		pathToCreate = filepath.Join(tmpDir, "test_folder")

		err := os.Chmod(tmpDir, 0o444)
		if err != nil {
			t.Skipf("Failed to change permissions on temporary directory: %v", err)
		}
		defer os.Chmod(tmpDir, 0o755) //nolint:golint,errcheck
	}

	err := EnsureFolder(pathToCreate)

	assert.Error(t, err, "Expected error during folder creation")
}
