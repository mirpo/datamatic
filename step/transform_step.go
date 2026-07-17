package step

import (
	"context"
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

	src, err := os.Open(srcStep.OutputFilename)
	if err != nil {
		return fmt.Errorf("failed to open source '%s': %w", srcStep.OutputFilename, err)
	}
	defer src.Close()

	writer, err := jsonl.NewWriter(step.OutputFilename)
	if err != nil {
		return fmt.Errorf("failed to create JSONL writer: %w", err)
	}
	defer writer.Close()

	if step.Collect {
		return runCollect(ctx, *srcStep, step, src, writer)
	}
	return runPerRow(ctx, *srcStep, step, src, writer)
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

		if step.UsesParent {
			err = step.JQProgram.RunEach(value, emit, jsonl.UnfoldLineage(lineage))
		} else {
			err = step.JQProgram.RunEach(value, emit)
		}
		if err != nil {
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
	var collected []interface{}
	scanner := fs.NewLineScanner(src)
	for lineNo := 0; scanner.Scan(); lineNo++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		value, _, _, err := getSourceDataFromLine(srcStep, scanner.Text())
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}

		collected = append(collected, value)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	written := 0
	if err := step.JQProgram.RunEach(collected, limitedEmit(writer, step.Limit, &written)); err != nil {
		return fmt.Errorf("collect: %w", err)
	}

	log.Info().Msgf("transform produced %d rows", written)
	return nil
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
