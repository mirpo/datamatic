package fs

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// GlobFiles expands a glob pattern or a directory into a sorted list of regular
// files (directories excluded). Results are always sorted ascending so runs are
// deterministic. An empty match is an error.
func GlobFiles(pattern string) ([]string, error) {
	var candidates []string

	if info, err := os.Stat(pattern); err == nil && info.IsDir() {
		entries, err := os.ReadDir(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory '%s': %w", pattern, err)
		}
		for _, e := range entries {
			candidates = append(candidates, filepath.Join(pattern, e.Name()))
		}
	} else {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob '%s': %w", pattern, err)
		}
		candidates = matches
	}

	files := candidates[:0]
	for _, path := range candidates {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("failed to stat '%s': %w", path, err)
		}
		if info.Mode().IsRegular() {
			files = append(files, path)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files matched '%s'", pattern)
	}

	sort.Strings(files)
	return files, nil
}

// ReadTextFile returns a file's base name and its full text content.
func ReadTextFile(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("failed to read '%s': %w", path, err)
	}
	return filepath.Base(path), string(data), nil
}

// ReadCSV parses a CSV file into one map per record, keyed by the header row.
func ReadCSV(path string) ([]map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open '%s': %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV '%s': %w", path, err)
	}
	if len(records) == 0 {
		return nil, nil
	}

	header := records[0]
	rows := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		row := make(map[string]string, len(header))
		for i, col := range header {
			if i < len(record) {
				row[col] = record[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}
