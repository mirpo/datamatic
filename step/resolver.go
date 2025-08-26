package step

import (
	"fmt"
	"strings"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/rs/zerolog/log"
)

func ResolveMaxResults(step *config.Step, cfg *config.Config) error {
	switch v := step.MaxResults.(type) {
	case int:
		step.ResolvedMaxResults = v
		return nil

	case string:
		if strings.HasSuffix(v, ".$length") {
			refStepName := strings.TrimSuffix(v, ".$length")
			refStep := cfg.GetStepByName(refStepName)

			if refStep == nil {
				return fmt.Errorf("reference step '%s' not found", refStepName)
			}

			isImageStep := step.HasImages()
			if isImageStep {
				imagesCount, err := fs.CountFiles(step.ImagePath)
				if err != nil {
					return fmt.Errorf("failed to count images in folder '%s': %w", step.ImagePath, err)
				}
				step.ResolvedMaxResults = imagesCount
				log.Debug().Msgf("Resolved MaxResults for step '%s' to %d from refStep: %s", step.Name, imagesCount, refStepName)
			} else {
				lines, err := fs.CountLinesInFile(refStep.OutputFilename)
				if err != nil {
					return fmt.Errorf("failed to count lines in '%s': %w", refStep.OutputFilename, err)
				}

				step.ResolvedMaxResults = lines
				log.Debug().Msgf("Resolved MaxResults for step '%s' to %d from refStep: %s", step.Name, lines, refStepName)
			}

			return nil
		}
	}

	return fmt.Errorf("unexpected MaxResults value: %v", step.MaxResults)
}
