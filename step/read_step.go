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
// (csv), or per line (jsonl). The read path resolves relative to the current
// working directory; rows materialize to OutputFilename like a transform step.
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
	for _, path := range files {
		if err := ctx.Err(); err != nil {
			return err
		}

		switch step.Format {
		case config.ReadFormatFiles:
			name, content, err := fs.ReadTextFile(path)
			if err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
			if err := writer.WriteJSON(map[string]interface{}{"path": path, "name": name, "content": content}); err != nil {
				return err
			}
			written++

		case config.ReadFormatCSV:
			rows, err := fs.ReadCSV(path)
			if err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
			for _, row := range rows {
				if err := writer.WriteJSON(row); err != nil {
					return err
				}
				written++
			}

		case config.ReadFormatJSONL:
			n, err := emitJSONLines(path, writer)
			if err != nil {
				return fmt.Errorf("step '%s': %w", step.Name, err)
			}
			written += n

		default:
			return fmt.Errorf("step '%s': unknown read format '%s'", step.Name, step.Format)
		}
	}

	log.Info().Msgf("read produced %d rows from %d file(s)", written, len(files))
	return nil
}

// emitJSONLines re-emits each JSON line of a file as an output row (validating
// that each line is well-formed JSON).
func emitJSONLines(path string, writer *jsonl.Writer) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to open '%s': %w", path, err)
	}
	defer f.Close()

	written := 0
	scanner := fs.NewLineScanner(f)
	for lineNo := 0; scanner.Scan(); lineNo++ {
		var value interface{}
		if err := json.Unmarshal([]byte(scanner.Text()), &value); err != nil {
			return written, fmt.Errorf("%s line %d: invalid JSON: %w", path, lineNo, err)
		}
		if err := writer.WriteJSON(value); err != nil {
			return written, err
		}
		written++
	}
	if err := scanner.Err(); err != nil {
		return written, fmt.Errorf("failed to read '%s': %w", path, err)
	}
	return written, nil
}
