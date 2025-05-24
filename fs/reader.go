package fs

import (
	"bufio"
	"errors"
	"fmt"
	"os"
)

func ReadLineFromFile(path string, lineNumber int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lineCount int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning file to count lines: %w", err)
	}
	if lineCount == 0 {
		return "", errors.New("file is empty")
	}

	effectiveIndex := normalizeIndex(lineNumber, lineCount)

	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("error seeking to start of file: %w", err)
	}
	scanner = bufio.NewScanner(file)
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

func normalizeIndex(index, length int) int {
	return (index%length + length) % length
}

func CountLinesInFile(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error while reading file %s: %w", filePath, err)
	}

	return lineCount, nil
}
