package fs

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
