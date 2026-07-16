package fs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
)

const maxScanLineSize = 16 * 1024 * 1024 // long LLM responses can exceed bufio's 64KB default

// NewLineScanner returns a line scanner sized for datamatic JSONL files,
// where a single line may hold a long LLM response.
func NewLineScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, maxScanLineSize)
	return scanner
}

// lineCountCache caches line counts per path. Safe because step outputs are
// immutable once the producing step completes and steps run sequentially.
var lineCountCache sync.Map // path -> int

func ReadLineFromFile(path string, lineNumber int) (string, error) {
	lineCount, err := cachedLineCount(path)
	if err != nil {
		return "", err
	}
	if lineCount == 0 {
		return "", errors.New("file is empty")
	}

	effectiveIndex := normalizeIndex(lineNumber, lineCount)
	if effectiveIndex != lineNumber {
		log.Warn().Msgf("requested line %d but '%s' has only %d lines — wrapping to line %d (check for data misalignment)",
			lineNumber, path, lineCount, effectiveIndex)
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := NewLineScanner(file)
	for i := 0; scanner.Scan(); i++ {
		if i == effectiveIndex {
			return scanner.Text(), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning file to read target line: %w", err)
	}

	return "", errors.New("unexpected error: target line not found")
}

func cachedLineCount(path string) (int, error) {
	if v, ok := lineCountCache.Load(path); ok {
		return v.(int), nil
	}

	count, err := CountLinesInFile(path)
	if err != nil {
		return 0, err
	}

	lineCountCache.Store(path, count)
	return count, nil
}

func normalizeIndex(index, length int) int {
	return (index%length + length) % length
}

func CountLinesInFile(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := NewLineScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error while reading file %s: %w", filePath, err)
	}

	return lineCount, nil
}
