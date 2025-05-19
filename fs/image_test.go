package fs

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPickImageFile(t *testing.T) {
	tmpDir := t.TempDir()
	files := []string{"img1.jpg", "img2.jpg", "img3.jpg"}

	for _, name := range files {
		fullPath := filepath.Join(tmpDir, name)
		err := os.WriteFile(fullPath, []byte("dummy image data"), 0o644)
		assert.NoError(t, err)
	}

	pattern := filepath.Join(tmpDir, "*.jpg")

	t.Run("valid index", func(t *testing.T) {
		path, err := PickImageFile(pattern, 1)
		assert.NoError(t, err)
		assert.Contains(t, path, "img2.jpg")
	})

	t.Run("index wraps around", func(t *testing.T) {
		path, err := PickImageFile(pattern, 7)
		assert.NoError(t, err)
		assert.Contains(t, path, "img2.jpg")
	})

	t.Run("no matches", func(t *testing.T) {
		emptyPattern := filepath.Join(tmpDir, "*.png")
		_, err := PickImageFile(emptyPattern, 0)
		assert.Error(t, err)
		_, err = PickImageFile(emptyPattern, 5)
		assert.Error(t, err)
	})
}

func TestImageToBase64(t *testing.T) {
	t.Run("valid image file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "test.jpg")
		originalData := []byte("fake-image-data")
		err := os.WriteFile(tmpFile, originalData, 0o644)
		assert.NoError(t, err)

		encoded, err := ImageToBase64(tmpFile)
		assert.NoError(t, err)

		expected := base64.StdEncoding.EncodeToString(originalData)
		assert.Equal(t, expected, encoded)
	})

	t.Run("file does not exist", func(t *testing.T) {
		_, err := ImageToBase64("non-existent.jpg")
		assert.Error(t, err)
	})

	t.Run("empty path", func(t *testing.T) {
		_, err := ImageToBase64("")
		assert.Error(t, err)
	})
}
