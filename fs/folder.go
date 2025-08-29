package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mirpo/datamatic/defaults"
)

func CreateVersionedFolder(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path is not absolute %s", path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return createFolder(path)
	} else if err != nil {
		return fmt.Errorf("failed to stat folder %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	base := sanitizeFolderName(filepath.Base(path))

	nextVersion, err := getNextFolderVersion(dir, base)
	if err != nil {
		return fmt.Errorf("failed to get next version for folder %s: %w", path, err)
	}

	versionedPath := filepath.Join(dir, fmt.Sprintf("%s_v%d", base, nextVersion))

	if err := os.Rename(path, versionedPath); err != nil {
		return fmt.Errorf("failed to rename folder from %s to %s: %w", path, versionedPath, err)
	}

	return createFolder(path)
}

func createFolder(path string) error {
	if err := os.MkdirAll(path, defaults.FolderPerm); err != nil {
		return fmt.Errorf("failed to create folder %s: %w", path, err)
	}
	return nil
}

func getNextFolderVersion(dir, base string) (int, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	maxVersion := 0
	re := regexp.MustCompile(fmt.Sprintf(`^%s_v(\d+)$`, regexp.QuoteMeta(base)))

	for _, file := range files {
		if file.IsDir() {
			if version, err := parseVersion(re, file.Name()); err == nil {
				if version > maxVersion {
					maxVersion = version
				}
			}
		}
	}

	return maxVersion + 1, nil
}

func parseVersion(re *regexp.Regexp, name string) (int, error) {
	matches := re.FindStringSubmatch(name)
	if len(matches) != 2 {
		return 0, fmt.Errorf("folder name %s does not match version pattern", name)
	}

	version, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse version number from folder name %s: %w", name, err)
	}

	return version, nil
}

func sanitizeFolderName(name string) string {
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F. ]`)
	sanitized := invalidChars.ReplaceAllString(name, "_")

	consecutiveUnderscores := regexp.MustCompile(`_+`)
	sanitized = consecutiveUnderscores.ReplaceAllString(sanitized, "_")

	sanitized = strings.Trim(sanitized, "_")

	if sanitized == "" {
		return "_"
	}

	return sanitized
}
