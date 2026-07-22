package fs

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// unionKeys returns the sorted union of keys across all row objects, so column
// order is deterministic regardless of per-row key variation.
func unionKeys(rows []map[string]interface{}) []string {
	seen := map[string]struct{}{}
	for _, r := range rows {
		for k := range r {
			seen[k] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// cellString renders a value for a CSV/Markdown cell: strings verbatim, nil as
// empty, everything else (numbers, bools, nested objects/arrays) as JSON — which
// keeps numbers exact and avoids scientific notation.
func cellString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// WriteCSV writes rows as CSV; the header is the sorted union of keys and nested
// values are JSON-encoded into their cell.
func WriteCSV(path string, rows []map[string]interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create '%s': %w", path, err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	keys := unionKeys(rows)
	if err := w.Write(keys); err != nil {
		return err
	}
	for _, row := range rows {
		record := make([]string, len(keys))
		for i, k := range keys {
			record[i] = cellString(row[k])
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// WriteJSONArray writes all rows as one pretty-printed JSON array.
func WriteJSONArray(path string, rows []interface{}) error {
	if rows == nil {
		rows = []interface{}{}
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal rows: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("failed to write '%s': %w", path, err)
	}
	return nil
}

// WriteMarkdownTable writes rows as a GitHub-flavored Markdown table; the header
// is the sorted union of keys, and cell newlines/pipes are escaped.
func WriteMarkdownTable(path string, rows []map[string]interface{}) error {
	keys := unionKeys(rows)

	var b strings.Builder
	b.WriteString("| " + strings.Join(keys, " | ") + " |\n")
	b.WriteString("|" + strings.Repeat(" --- |", len(keys)) + "\n")
	for _, row := range rows {
		cells := make([]string, len(keys))
		for i, k := range keys {
			cells[i] = mdEscape(cellString(row[k]))
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write '%s': %w", path, err)
	}
	return nil
}

func mdEscape(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "|", "\\|")
}
