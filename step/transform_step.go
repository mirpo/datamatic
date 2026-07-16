package step

import (
	"context"
	"fmt"
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

	written := 0
	scanner := fs.NewLineScanner(src)
	for lineNo := 0; scanner.Scan(); lineNo++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if step.Limit > 0 && written >= step.Limit {
			log.Debug().Msgf("limit %d reached, stopping", step.Limit)
			return nil
		}

		value, _, err := getSourceDataFromLine(*srcStep, scanner.Text())
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}

		results, err := step.JQProgram.Run(value)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}

		for _, result := range results {
			if step.Limit > 0 && written >= step.Limit {
				break // mid-fan-out cap; outer check stops the scan
			}
			if err := writer.WriteJSON(result); err != nil {
				return fmt.Errorf("failed to write output line: %w", err)
			}
			written++
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	log.Info().Msgf("transform produced %d rows", written)
	return nil
}
