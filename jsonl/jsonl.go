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

func NewWriter(path string) (*Writer, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	log.Debug().Msgf("created file for output: %s", path)

	return &Writer{file: file}, nil
}

func (w *Writer) WriteStringLine(line string) error {
	log.Debug().Msgf("writing raw data: %s", line)

	if _, err := w.file.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("failed to write string line to file: %w", err)
	}
	return nil
}

func (w *Writer) WriteLine(entity LineEntity) error {
	jsonData, err := json.Marshal(entity)
	if err != nil {
		return fmt.Errorf("failed to marshal line entity: %w", err)
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
