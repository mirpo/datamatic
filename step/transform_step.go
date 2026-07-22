package step

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/rs/zerolog/log"
)

type TransformStep struct{}

func (p *TransformStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	srcStep := cfg.GetStepByName(step.From)
	if srcStep == nil {
		return fmt.Errorf("'from' references unknown step '%s'", step.From)
	}

	writer, err := jsonl.NewWriter(step.OutputFilename)
	if err != nil {
		return fmt.Errorf("failed to create JSONL writer: %w", err)
	}
	defer writer.Close()

	if step.SourceFormat == config.SourceFormatJSON {
		return runWholeJSON(step, srcStep.OutputFilename, writer)
	}

	src, err := os.Open(srcStep.OutputFilename)
	if err != nil {
		return fmt.Errorf("failed to open source '%s': %w", srcStep.OutputFilename, err)
	}
	defer src.Close()

	if step.Collect {
		return runCollect(ctx, *srcStep, step, src, writer)
	}
	return runPerRow(ctx, *srcStep, step, src, writer)
}

// runWholeJSON decodes the source file as a single JSON value (e.g. a
// pretty-printed array from an API dump) and runs the program once over it.
func runWholeJSON(step config.Step, path string, writer *jsonl.Writer) error {
	data, err := os.ReadFile(path) // presized by file stat, unlike io.ReadAll
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("sourceFormat json: failed to parse source: %w", err)
	}

	written := 0
	if err := runProgram(step, value, limitedEmit(writer, step.Limit, &written), nil); err != nil {
		return err
	}

	log.Info().Msgf("transform produced %d rows", written)
	return nil
}

// runProgram runs the compiled program over one input, passing $parent only
// when it was declared at compile time (the argument count must match the
// variables declared in preprocessing).
func runProgram(step config.Step, input interface{}, emit func(interface{}) (bool, error), parent interface{}) error {
	if step.UsesParent {
		return step.JQProgram.RunEach(input, emit, parent)
	}
	return step.JQProgram.RunEach(input, emit)
}

// runPerRow streams the source: the jq program runs once per row, each
// emitted value becomes an output row (fan-out), limit caps output.
func runPerRow(ctx context.Context, srcStep config.Step, step config.Step, src io.Reader, writer *jsonl.Writer) error {
	written := 0
	emit := limitedEmit(writer, step.Limit, &written)

	scanner := fs.NewLineScanner(src)
	for lineNo := 0; scanner.Scan(); lineNo++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if step.Limit > 0 && written >= step.Limit {
			log.Debug().Msgf("limit %d reached, stopping", step.Limit)
			return nil
		}

		value, _, lineage, err := getSourceDataFromLine(srcStep, scanner.Text())
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}

		if err := runProgram(step, value, emit, jsonl.UnfoldLineage(lineage)); err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	log.Info().Msgf("transform produced %d rows", written)
	return nil
}

// runCollect gathers all source rows into one array and runs the jq program
// once over it (fan-in: unique, group_by, sort_by across the whole dataset).
func runCollect(ctx context.Context, srcStep config.Step, step config.Step, src io.Reader, writer *jsonl.Writer) error {
	collected, err := collectRows(ctx, srcStep, src)
	if err != nil {
		return err
	}

	written := 0
	if err := step.JQProgram.RunEach(collected, limitedEmit(writer, step.Limit, &written)); err != nil {
		return fmt.Errorf("collect: %w", err)
	}

	log.Info().Msgf("transform produced %d rows", written)
	return nil
}

// collectRows scans a source step's output, decoding every line into its row
// value via the shared decoder. Used by transform collect and write steps.
func collectRows(ctx context.Context, srcStep config.Step, r io.Reader) ([]interface{}, error) {
	var rows []interface{}
	scanner := fs.NewLineScanner(r)
	for lineNo := 0; scanner.Scan(); lineNo++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value, _, _, err := getSourceDataFromLine(srcStep, scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		rows = append(rows, value)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read source: %w", err)
	}
	return rows, nil
}

// limitedEmit returns a RunEach callback that writes emitted values until the
// output limit is reached (0 = no limit).
func limitedEmit(writer *jsonl.Writer, limit int, written *int) func(interface{}) (bool, error) {
	return func(v interface{}) (bool, error) {
		if limit > 0 && *written >= limit {
			return true, nil
		}
		if err := writer.WriteJSON(v); err != nil {
			return false, fmt.Errorf("failed to write output line: %w", err)
		}
		*written++
		return false, nil
	}
}
