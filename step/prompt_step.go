package step

import (
	"context"
	"fmt"
	"os"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/mirpo/datamatic/llm"
	"github.com/mirpo/datamatic/promptbuilder"
	"github.com/mirpo/datamatic/retry"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

func newProviderConfigFromStep(step config.Step, httpTimeout int) llm.ProviderConfig {
	return llm.ProviderConfig{
		BaseURL:      step.ModelConfig.BaseURL,
		ProviderType: step.ModelConfig.ModelProvider,
		ModelName:    step.ModelConfig.ModelName,
		AuthToken:    "token",
		HTTPTimeout:  httpTimeout,
		Temperature:  step.ModelConfig.Temperature,
		MaxTokens:    step.ModelConfig.MaxTokens,
	}
}

type PromptStep struct{}

// sourceRows holds every line of a referenced step's output, read once so the
// per-row workers only do in-memory lookups instead of re-scanning the file.
type sourceRows struct {
	step       config.Step
	fieldPaths []string
	lines      []string
}

func (p *PromptStep) retryLLMGeneration(ctx context.Context, cfg *config.Config, provider llm.Provider, req llm.GenerateRequest, response **llm.GenerateResponse) error {
	return retry.Do(ctx, cfg.RetryConfig, func() error {
		resp, err := provider.Generate(ctx, req)
		if err == nil {
			*response = resp
			return nil
		}
		return err
	}, retry.ShouldRetryHTTPError)
}

func (p *PromptStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	total := step.ResolvedCount
	// PreprocessConfig resolves the default (0 → 1); this clamp only guards
	// direct Run calls in unit tests that bypass preprocessing, where a
	// zero-capacity errgroup limit would deadlock.
	workers := step.Concurrency
	if workers < 1 {
		workers = 1
	}

	writer, err := jsonl.NewWriter(step.OutputFilename)
	if err != nil {
		return fmt.Errorf("failed to create JSONL writer: %w", err)
	}
	defer writer.Close()

	provider, err := llm.NewProvider(newProviderConfigFromStep(step, cfg.HTTPTimeout))
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}

	hasSchema := step.JSONSchema.HasSchemaDefinition()

	// parse the prompt once to discover which steps it references, then read
	// each referenced file a single time up front (rows only differ by values)
	base, err := promptbuilder.NewPromptBuilder(step.Prompt, step.ForEach)
	if err != nil {
		return err
	}
	sources, err := loadSources(base, cfg, total)
	if err != nil {
		return err
	}

	runRow := func(ctx context.Context, i int) (jsonl.LineEntity, error) {
		return p.runRow(ctx, cfg, step, hasSchema, provider, sources, i)
	}

	return generate(ctx, total, workers, writer, runRow)
}

// runRow produces a single output row: build its prompt from the preloaded
// source values, call the LLM, and retry within the per-row attempt budget
// when the response fails validation.
func (p *PromptStep) runRow(ctx context.Context, cfg *config.Config, step config.Step, hasSchema bool, provider llm.Provider, sources []sourceRows, i int) (jsonl.LineEntity, error) {
	log.Info().
		Str("step_name", step.Name).
		Str("step_type", string(step.Type)).
		Int("iteration", i).
		Msg("Running step")

	pb, err := promptbuilder.NewPromptBuilder(step.Prompt, step.ForEach)
	if err != nil {
		return jsonl.LineEntity{}, err
	}

	for _, src := range sources {
		if i >= len(src.lines) {
			return jsonl.LineEntity{}, fmt.Errorf("step '%s': row %d not found (only %d rows)", src.step.Name, i, len(src.lines))
		}
		values, err := extractStepValues(src.step, src.lines[i], src.fieldPaths)
		if err != nil {
			return jsonl.LineEntity{}, fmt.Errorf("failed to read values from step '%s' row %d: %w", src.step.Name, i, err)
		}
		pb.AddStepValues(src.step.Name, values)
	}

	var base64Image string
	if step.HasImages() {
		imagePath, err := fs.PickImageFile(step.ImagePath, i)
		if err != nil {
			return jsonl.LineEntity{}, fmt.Errorf("failed to find images by pattern '%s': %w", step.ImagePath, err)
		}

		base64Image, err = fs.ImageToBase64(imagePath)
		if err != nil {
			return jsonl.LineEntity{}, fmt.Errorf("failed to encode image '%s': %w", imagePath, err)
		}

		pb.AddValue(base64Image[:15], step.Name, "image", imagePath)
	}

	userPrompt, err := pb.BuildPrompt()
	if err != nil {
		return jsonl.LineEntity{}, fmt.Errorf("failed to build prompt: %w", err)
	}

	req := llm.GenerateRequest{
		UserMessage:   userPrompt,
		SystemMessage: step.SystemPrompt,
		IsJSON:        hasSchema,
		JSONSchema:    step.JSONSchema,
		Base64Image:   base64Image,
	}

	invalidAttempts := 0
	// registerInvalid records an unusable response (schema violation or
	// malformed line) and returns a terminal error once the attempt budget is
	// exhausted; nil means "retry this row".
	registerInvalid := func(cause error, responseText string) error {
		invalidAttempts++
		log.Warn().Err(cause).Msgf("row %d: invalid LLM response (attempt %d/%d): %s",
			i, invalidAttempts, cfg.RetryConfig.MaxAttempts, responseText)
		if invalidAttempts >= cfg.RetryConfig.MaxAttempts {
			return fmt.Errorf("row %d: LLM returned invalid response %d times in a row: %w", i, invalidAttempts, cause)
		}
		return nil
	}

	for {
		var response *llm.GenerateResponse
		if err := p.retryLLMGeneration(ctx, cfg, provider, req, &response); err != nil {
			return jsonl.LineEntity{}, fmt.Errorf("row %d: failed to get response from LLM after retries: %w", i, err)
		}

		if cfg.ValidateResponse && hasSchema {
			log.Debug().Msg("Validating response from LLM using JSON schema")
			if err := step.JSONSchema.ValidateJSONText(response.Text); err != nil {
				if failErr := registerInvalid(err, response.Text); failErr != nil {
					return jsonl.LineEntity{}, failErr
				}
				continue
			}
		}

		log.Info().Msgf("Response from LLM: '%s'", response.Text)

		lineEntity, err := jsonl.NewLineEntity(response.Text, userPrompt, hasSchema, pb.GetValues())
		if err != nil {
			if failErr := registerInvalid(err, response.Text); failErr != nil {
				return jsonl.LineEntity{}, failErr
			}
			continue
		}

		return lineEntity, nil
	}
}

// loadSources resolves the steps referenced by the prompt and reads each of
// their output files once into memory, indexed by row.
func loadSources(base *promptbuilder.PromptBuilder, cfg *config.Config, total int) ([]sourceRows, error) {
	var sources []sourceRows
	for stepName, fieldPaths := range base.GroupPlaceholdersByStep() {
		refStep := cfg.GetStepByName(stepName)
		if refStep == nil {
			return nil, fmt.Errorf("prompt references unknown step '%s'", stepName)
		}

		lines, err := readAllLines(refStep.OutputFilename, total)
		if err != nil {
			return nil, fmt.Errorf("failed to read values from step '%s': %w", stepName, err)
		}

		sources = append(sources, sourceRows{step: *refStep, fieldPaths: fieldPaths, lines: lines})
	}
	return sources, nil
}

// readAllLines reads up to `limit` non-empty lines from a JSONL file in a
// single pass (limit <= 0 reads them all).
func readAllLines(path string, limit int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := fs.NewLineScanner(file)
	for scanner.Scan() {
		if limit > 0 && len(lines) >= limit {
			break
		}
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// generate runs rows through runRow with up to `workers` in flight and writes
// their results to the writer in row order. A single collector goroutine keeps
// output deterministic and streams each row as soon as its predecessors are
// done, so a mid-run failure still leaves the completed prefix on disk.
func generate(ctx context.Context, total, workers int, writer *jsonl.Writer, runRow func(context.Context, int) (jsonl.LineEntity, error)) error {
	if total == 0 {
		return nil
	}

	results := make([]jsonl.LineEntity, total)
	done := make(chan int, total)
	writeErr := make(chan error, 1)

	go func() {
		arrived := make([]bool, total)
		next := 0
		for i := range done {
			arrived[i] = true
			for next < total && arrived[next] {
				if err := writer.WriteLine(results[next]); err != nil {
					writeErr <- fmt.Errorf("failed to write output line: %w", err)
					return
				}
				results[next] = jsonl.LineEntity{} // let the written row be GC'd
				next++
			}
		}
		writeErr <- nil
	}()

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)
	for i := range total {
		g.Go(func() error {
			line, err := runRow(gctx, i)
			if err != nil {
				return err
			}
			results[i] = line
			done <- i
			return nil
		})
	}

	runErr := g.Wait()
	close(done)
	if werr := <-writeErr; werr != nil {
		return werr
	}
	return runErr
}
