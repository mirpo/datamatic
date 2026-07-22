package step

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/rs/zerolog/log"
)

type ReadStep struct{}

// Run loads local files into JSONL rows: one row per file (files), per record
// (csv), or per line (jsonl). The read path is resolved during preprocessing
// (relative to the config file's dir); rows materialize to OutputFilename like
// a transform step.
func (p *ReadStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	files, err := fs.GlobFiles(step.Read)
	if err != nil {
		return fmt.Errorf("step '%s': %w", step.Name, err)
	}

	writer, err := jsonl.NewWriter(step.OutputFilename)
	if err != nil {
		return fmt.Errorf("failed to create JSONL writer: %w", err)
	}
	defer writer.Close()

	written := 0
	emit := limitedEmit(writer, 0, &written) // read has no limit; reuses the transform writer

	// resolve the per-file loader once — the format is constant for the step
	var loadFile func(path string) error
	switch step.Format {
	case config.ReadFormatFiles:
		loadFile = func(path string) error {
			name, content, err := fs.ReadTextFile(path)
			if err != nil {
				return err
			}
			_, err = emit(map[string]interface{}{"path": path, "name": name, "content": content})
			return err
		}
	case config.ReadFormatCSV:
		loadFile = func(path string) error {
			rows, err := fs.ReadCSV(path)
			if err != nil {
				return err
			}
			for _, row := range rows {
				if _, err := emit(row); err != nil {
					return err
				}
			}
			return nil
		}
	case config.ReadFormatJSONL:
		loadFile = func(path string) error { return emitJSONLines(path, emit) }
	default:
		return fmt.Errorf("step '%s': unknown read format '%s'", step.Name, step.Format)
	}

	for _, path := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := loadFile(path); err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}
	}

	log.Info().Msgf("read produced %d rows from %d file(s)", written, len(files))
	return nil
}

// emitJSONLines emits each JSON line of a file as an output row (validating
// that each line is well-formed JSON).
func emitJSONLines(path string, emit func(interface{}) (bool, error)) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open '%s': %w", path, err)
	}
	defer f.Close()

	scanner := fs.NewLineScanner(f)
	for lineNo := 0; scanner.Scan(); lineNo++ {
		var value interface{}
		if err := json.Unmarshal([]byte(scanner.Text()), &value); err != nil {
			return fmt.Errorf("%s line %d: invalid JSON: %w", path, lineNo, err)
		}
		if _, err := emit(value); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read '%s': %w", path, err)
	}
	return nil
}
