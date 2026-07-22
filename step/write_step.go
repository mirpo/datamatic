package step

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/rs/zerolog/log"
)

type WriteStep struct{}

// Run exports a source step's rows to a single file in the configured format
// (csv / json / md / jsonl). It is terminal — it reads the source's JSONL and
// writes the deliverable; it does not produce pipeline rows itself.
func (p *WriteStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	src := cfg.GetStepByName(step.From)
	if src == nil {
		return fmt.Errorf("'from' references unknown step '%s'", step.From)
	}

	file, err := os.Open(src.OutputFilename)
	if err != nil {
		return fmt.Errorf("step '%s': failed to open source '%s': %w", step.Name, src.OutputFilename, err)
	}
	defer file.Close()

	rows, err := collectRows(ctx, *src, file)
	if err != nil {
		return fmt.Errorf("step '%s': %w", step.Name, err)
	}

	switch step.Format {
	case config.WriteFormatJSON:
		if err := fs.WriteJSONArray(step.Write, rows); err != nil {
			return err
		}
	case config.WriteFormatJSONL:
		if err := writeJSONL(step.Write, rows); err != nil {
			return err
		}
	case config.WriteFormatCSV, config.WriteFormatMarkdown:
		objs, err := asObjects(rows)
		if err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}
		// WriteCSV and WriteMarkdownTable share a signature — pick by format
		writeTable := fs.WriteCSV
		if step.Format == config.WriteFormatMarkdown {
			writeTable = fs.WriteMarkdownTable
		}
		if err := writeTable(step.Write, objs); err != nil {
			return err
		}
	default:
		return fmt.Errorf("step '%s': unknown write format '%s'", step.Name, step.Format)
	}

	log.Info().Msgf("write exported %d rows to %s", len(rows), step.Write)
	return nil
}

// asObjects asserts every row is a JSON object (required for csv/md columns).
func asObjects(rows []interface{}) ([]map[string]interface{}, error) {
	objs := make([]map[string]interface{}, 0, len(rows))
	for i, r := range rows {
		obj, ok := r.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("row %d is not a JSON object; csv/md output needs object rows (use json/jsonl instead)", i)
		}
		objs = append(objs, obj)
	}
	return objs, nil
}

func writeJSONL(path string, rows []interface{}) error {
	file, err := os.Create(path) // truncate: a fresh deliverable each run
	if err != nil {
		return fmt.Errorf("failed to create '%s': %w", path, err)
	}
	defer file.Close()

	enc := json.NewEncoder(file) // Encode writes one compact JSON value + newline
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}
	return nil
}
