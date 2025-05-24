package fs

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

func globFiles(pattern string) ([]string, error) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func CountFiles(pattern string) (int, error) {
	log.Debug().Msgf("searching for files matching pattern: %s", pattern)

	files, err := globFiles(pattern)
	if err != nil {
		return 0, fmt.Errorf("failed to search for files with pattern %s: %w", pattern, err)
	}

	totalFiles := len(files)
	log.Debug().Msgf("found %d files matching pattern: %s", totalFiles, pattern)

	return totalFiles, nil
}

func PickImageFile(pattern string, index int) (string, error) {
	log.Debug().Msgf("Searching for files matching pattern: %s", pattern)

	files, err := globFiles(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to search for files with pattern %s: %w", pattern, err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files matched pattern: %s", pattern)
	}

	index = index % len(files)
	selected := files[index]

	log.Debug().Msgf("Selected file [%d]: %s", index, selected)
	return selected, nil
}

func ImageToBase64(imagePath string) (string, error) {
	if imagePath == "" {
		return "", errors.New("image path is empty")
	}

	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", imagePath, err)
	}

	return base64.StdEncoding.EncodeToString(data), nil
}
