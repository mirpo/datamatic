package jsonl

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

type Writer struct {
	file *os.File
}

// NewWriter opens a step's output file for writing, truncating it: a step
// materializes all of its rows in one pass, so the file always holds exactly
// the rows of the run that produced it (a rerun replaces, never appends).
func NewWriter(path string) (*Writer, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	log.Debug().Msgf("created file for output: %s", path)

	return &Writer{file: file}, nil
}

func (w *Writer) WriteLine(entity LineEntity) error {
	return w.WriteJSON(entity)
}

// WriteJSON writes any value as one compact JSON line (used by transform steps).
func (w *Writer) WriteJSON(v interface{}) error {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	log.Debug().Msgf("writing jsonl line: %s", string(jsonData))

	if _, err := w.file.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write json data to file: %w", err)
	}

	if _, err := w.file.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline after json data: %w", err)
	}

	return nil
}

func (w *Writer) Close() error {
	return w.file.Close()
}
