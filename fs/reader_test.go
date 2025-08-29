package fs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeIndex(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		length   int
		expected int
	}{
		{"zero index", 0, 5, 0},
		{"positive in range", 2, 5, 2},
		{"positive wrap", 7, 5, 2},
		{"negative wrap", -1, 5, 4},
		{"negative wrap again", -6, 5, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeIndex(tt.index, tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func createTempFileWithContent(t *testing.T, lines []string) *os.File {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "testfile_*.txt")
	assert.NoError(t, err)

	for _, line := range lines {
		_, err := tmpfile.WriteString(line + "\n")
		assert.NoError(t, err)
	}
	err = tmpfile.Close()
	assert.NoError(t, err)

	return tmpfile
}

func TestReadLineFromFile(t *testing.T) {
	lines := []string{"first", "second", "third", "fourth"}
	tmpfile := createTempFileWithContent(t, lines)
	defer os.Remove(tmpfile.Name())

	tests := []struct {
		name     string
		index    int
		expected string
	}{
		{"first line", 0, "first"},
		{"last line with -1", -1, "fourth"},
		{"wrap around", 5, "second"},
		{"negative wrap", -5, "fourth"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ReadLineFromFile(tmpfile.Name(), tt.index)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReadLineFromFile_EmptyFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "emptyfile_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	assert.NoError(t, tmpfile.Close())

	result, err := ReadLineFromFile(tmpfile.Name(), 0)
	assert.Error(t, err)
	assert.Equal(t, "", result)
	assert.Contains(t, err.Error(), "file is empty")
}

func TestReadLineFromFile_FileNotExist(t *testing.T) {
	_, err := ReadLineFromFile("non_existent_file.txt", 0)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err), "Expected 'file not exist' error, but got: %v", err)
}

func TestCountLinesInFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{"empty file", "", 0},
		{"one line", "hello", 1},
		{"multiple lines", "line1\nline2\nline3", 3},
		{"trailing newline", "line1\nline2\n", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "testfile")
			assert.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.content)
			assert.NoError(t, err)
			tmpFile.Close()

			count, err := CountLinesInFile(tmpFile.Name())
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, count)
		})
	}
}
