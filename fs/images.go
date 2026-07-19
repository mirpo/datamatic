package fs

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
)

// ImageToBase64 reads a file and returns its base64-encoded contents, used to
// attach an image to a vision model request.
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
