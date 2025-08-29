package fs

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/mirpo/datamatic/defaults"
	"github.com/stretchr/testify/assert"
)

func TestCreateFolder(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_folder")

	err := createFolder(path)
	assert.NoError(t, err, "Expected no error when creating folder")

	_, err = os.Stat(path)
	assert.False(t, os.IsNotExist(err), "Expected folder %s to be created, but it does not exist", path)
}

func TestGetNextFolderVersion(t *testing.T) {
	tmpDir := t.TempDir()
	base := "test_folder"

	_ = os.Mkdir(filepath.Join(tmpDir, "test_folder_v1"), defaults.FolderPerm)
	_ = os.Mkdir(filepath.Join(tmpDir, "test_folder_v2"), defaults.FolderPerm)
	_ = os.Mkdir(filepath.Join(tmpDir, "test_folder_v5"), defaults.FolderPerm)
	_ = os.Mkdir(filepath.Join(tmpDir, "test_folder_v19"), defaults.FolderPerm)

	nextVersion, err := getNextFolderVersion(tmpDir, base)
	assert.NoError(t, err, "Expected no error when retrieving next folder version")

	expectedVersion := 20
	assert.Equal(t, expectedVersion, nextVersion, "Expected next version to be %d, got %d", expectedVersion, nextVersion)
}

func TestParseVersion(t *testing.T) {
	re := regexp.MustCompile(`^test_folder_v(\d+)$`)

	version, err := parseVersion(re, "test_folder_v3")
	assert.NoError(t, err, "Expected no error for valid folder name")
	assert.Equal(t, 3, version, "Expected version 3, got %d", version)

	_, err = parseVersion(re, "test_folder")
	assert.Error(t, err, "Expected error for folder without version")

	_, err = parseVersion(re, "test_folder_vx")
	assert.Error(t, err, "Expected error for folder with invalid version format")
}

func TestCreateVersionedFolder(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_folder")

	err := CreateVersionedFolder(path)
	assert.NoError(t, err, "Expected no error on first call to CreateVersionedFolder")

	_, err = os.Stat(path)
	assert.False(t, os.IsNotExist(err), "Expected folder %s to be created, but it does not exist", path)

	err = CreateVersionedFolder(path)
	assert.NoError(t, err, "Expected no error on second call to CreateVersionedFolder")

	versionedPath := filepath.Join(tmpDir, "test_folder_v1")
	_, err = os.Stat(versionedPath)
	assert.False(t, os.IsNotExist(err), "Expected versioned folder %s to be created, but it does not exist", versionedPath)

	_, err = os.Stat(path)
	assert.False(t, os.IsNotExist(err), "Expected new folder %s to be created, but it does not exist", path)
}

func TestCreateFolderNegative(t *testing.T) {
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

	err := createFolder(pathToCreate)

	assert.Error(t, err, "Expected error during folder creation")
}

func TestGetNextFolderVersionNoFolders(t *testing.T) {
	tmpDir := t.TempDir()
	base := "test_folder"

	nextVersion, err := getNextFolderVersion(tmpDir, base)
	assert.NoError(t, err, "Expected no error when retrieving version for an empty directory")
	assert.Equal(t, 1, nextVersion, "Expected version to start at 1 for empty directory")
}

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"valid_name", "valid_name"},
		{"name with spaces", "name_with_spaces"},
		{"name/with/slash", "name_with_slash"},
		{">invalid<chars", "invalid_chars"}, // Leading invalid character should add "_"
		{"   trailing spaces  ", "trailing_spaces"},
		{"..trailing.dots..", "trailing_dots"}, // Dots replaced
		{"special|<>:*?chars", "special_chars"},
		{"multiple___underscores", "multiple_underscores"},
		{"", "_"}, // Empty name should return "_"
	}

	for _, test := range tests {
		result := sanitizeFolderName(test.name)
		assert.Equal(t, test.expected, result, "Expected sanitized name to be %s, got %s", test.expected, result)
	}
}
