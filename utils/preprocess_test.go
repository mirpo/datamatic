package utils

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/llm"
	"github.com/stretchr/testify/assert"
)

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid simple", "file.txt", false},
		{"Valid dot", ".", false},
		{"Valid long name", strings.Repeat("a", 255), false},
		{"Empty", "", true},
		{"Too long", strings.Repeat("a", 256), true},
		{"Invalid char <", "bad<name", true},
		{"Ends with space", "bad ", true},
		{"Ends with period", "bad.", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isValidName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidProvider(t *testing.T) {
	assert.True(t, isValidProvider(llm.ProviderOllama))
	assert.True(t, isValidProvider(llm.ProviderOpenAI))
	assert.False(t, isValidProvider(llm.ProviderUnknown))
	assert.False(t, isValidProvider(llm.ProviderType("INVALID")))
}

func TestPreprocessConfig_Success(t *testing.T) {
	outputFolder := filepath.Join("tmp", "test")
	cfg := &config.Config{
		OutputFolder: outputFolder,
		Steps: []config.Step{
			{
				Name:           "prompt1",
				Model:          "ollama:llama3.2",
				Prompt:         "Generate something",
				OutputFilename: "custom",
				ImagePath:      "images/photo.jpg",
				// no count/forEach: image step, iterations resolved at runtime
			},
			{
				Name:           "cli1",
				Run:            "echo hi",
				OutputFilename: "cli1",
			},
			{
				Name:   "prompt2",
				Model:  "openai:gpt-4",
				Prompt: "More text",
				Count:  5,
			},
			{
				Name:    "prompt3",
				Model:   "gemini:gemini-pro",
				Prompt:  "Dynamic",
				ForEach: "prompt1",
			},
		},
	}

	err := PreprocessConfig(cfg)
	assert.NoError(t, err)

	// Step types
	assert.Equal(t, config.PromptStepType, cfg.Steps[0].Type)
	assert.Equal(t, config.ShellStepType, cfg.Steps[1].Type)

	// Providers + models
	assert.Equal(t, llm.ProviderOllama, cfg.Steps[0].ModelConfig.ModelProvider)
	assert.Equal(t, "llama3.2", cfg.Steps[0].ModelConfig.ModelName)
	assert.Equal(t, llm.ProviderOpenAI, cfg.Steps[2].ModelConfig.ModelProvider)
	assert.Equal(t, "gpt-4", cfg.Steps[2].ModelConfig.ModelName)

	// Filenames - use absolute paths that work cross-platform
	// Prompt steps get .jsonl appended, CLI steps don't
	expectedCustom, _ := filepath.Abs(filepath.Join(outputFolder, "custom.jsonl"))
	expectedCli, _ := filepath.Abs(filepath.Join(outputFolder, "cli1"))
	assert.Equal(t, expectedCustom, cfg.Steps[0].OutputFilename)
	assert.Equal(t, expectedCli, cfg.Steps[1].OutputFilename) // CLI steps get absolute path but no extension change

	// Image path - use absolute paths that work cross-platform
	expectedImage, _ := filepath.Abs(filepath.Join(outputFolder, "images", "photo.jpg"))
	assert.Equal(t, expectedImage, cfg.Steps[0].ImagePath)

	// Iteration settings
	assert.Equal(t, 0, cfg.Steps[0].Count, "image step: iterations resolved at runtime, no default count")
	assert.Equal(t, 0, cfg.Steps[1].Count, "shell steps have no count")
	assert.Equal(t, 5, cfg.Steps[2].Count)
	assert.Equal(t, "prompt1", cfg.Steps[3].ForEach)
	assert.Equal(t, 0, cfg.Steps[3].Count)
}

func TestSetWorkDir(t *testing.T) {
	absDataset, _ := filepath.Abs(filepath.Join("tmp", "dataset"))
	absVarTmp, _ := filepath.Abs(filepath.Join(string(filepath.Separator), "var", "tmp"))

	tests := []struct {
		name         string
		workDir      string
		outputFolder string
		wantPath     string
	}{
		{
			name:         "Empty workDir defaults to outputFolder",
			workDir:      "",
			outputFolder: absDataset,
			wantPath:     absDataset,
		},
		{
			name:         "Relative workDir joined with outputFolder",
			workDir:      "subdir",
			outputFolder: absDataset,
			wantPath:     filepath.Join(absDataset, "subdir"),
		},
		{
			name:         "Absolute workDir preserved",
			workDir:      absVarTmp,
			outputFolder: absDataset,
			wantPath:     absVarTmp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &config.Step{WorkDir: tt.workDir}
			err := setWorkDir(step, tt.outputFolder)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantPath, step.WorkDir)
			assert.True(t, filepath.IsAbs(step.WorkDir), "workDir should be absolute")
		})
	}
}

func TestPreprocessConfig_Failures(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		errMsg string
	}{
		{
			"Both prompt and run",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "bad", Prompt: "p", Run: "c"},
			}},
			"exactly one of 'prompt', 'run' or 'jq' must be defined",
		},
		{
			"Missing provider colon",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "bad", Prompt: "p", Model: "invalidmodel"},
			}},
			"model should follow pattern",
		},
		{
			"Invalid filename",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "bad<name>", Prompt: "p", Model: "ollama:llama3.2"},
			}},
			"filename contains invalid characters",
		},
		{
			"Empty step name",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "", Prompt: "p", Model: "ollama:llama3.2"},
			}},
			"name can't be empty",
		},
		{
			"Reserved name SYSTEM",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "SYSTEM", Prompt: "p", Model: "ollama:llama3.2"},
			}},
			"not allowed",
		},
		{
			"Duplicate step names",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "dup", Prompt: "p", Model: "ollama:llama3.2"},
				{Name: "dup", Prompt: "q", Model: "openai:gpt-4"},
			}},
			"duplicate step name",
		},
		{
			"Shell without output filename",
			&config.Config{OutputFolder: "/tmp", Steps: []config.Step{
				{Name: "cli1", Run: "echo hi"},
			}},
			"output filename is mandatory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PreprocessConfig(tt.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestSetStepType_ExplicitTypeValidation(t *testing.T) {
	tests := []struct {
		name    string
		step    config.Step
		wantErr string // empty = no error
		want    config.StepType
	}{
		{"explicit prompt matches", config.Step{Type: "prompt", Prompt: "p"}, "", config.PromptStepType},
		{"explicit shell matches", config.Step{Type: "shell", Run: "r"}, "", config.ShellStepType},
		{"explicit shell but prompt defined", config.Step{Type: "shell", Prompt: "p"}, "does not match", ""},
		{"explicit prompt but run defined", config.Step{Type: "prompt", Run: "r"}, "does not match", ""},
		{"unknown explicit type", config.Step{Type: "banana", Run: "r"}, "unknown step type", ""},
		{"jq field infers transform", config.Step{JQ: ".x"}, "", config.TransformStepType},
		{"explicit transform matches", config.Step{Type: "transform", JQ: ".x"}, "", config.TransformStepType},
		{"jq and prompt both defined", config.Step{JQ: ".x", Prompt: "p"}, "exactly one", ""},
		{"nothing defined", config.Step{}, "exactly one", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setStepType(&tt.step)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, tt.step.Type)
		})
	}
}

func TestPreprocessConfig_TransformStep(t *testing.T) {
	base := func() *config.Config {
		cfg := config.NewConfig()
		cfg.OutputFolder = t.TempDir()
		cfg.Steps = []config.Step{
			{Name: "src", Run: "echo '{}' > src.jsonl", OutputFilename: "src.jsonl"},
			{Name: "pick", JQ: `select(.ok)`, From: "src"},
		}
		return cfg
	}

	t.Run("valid transform step compiles and gets output path", func(t *testing.T) {
		cfg := base()
		err := PreprocessConfig(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, cfg.Steps[1].JQProgram)
		assert.Contains(t, cfg.Steps[1].OutputFilename, "pick.jsonl")
	})

	t.Run("missing from fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].From = ""
		err := PreprocessConfig(cfg)
		assert.ErrorContains(t, err, "'from' is required")
	})

	t.Run("from referencing unknown step fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].From = "ghost"
		err := PreprocessConfig(cfg)
		assert.ErrorContains(t, err, "unknown step 'ghost'")
	})

	t.Run("from referencing itself fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].From = "pick" // itself — not an earlier step
		err := PreprocessConfig(cfg)
		assert.ErrorContains(t, err, "unknown step 'pick'")
	})

	t.Run("invalid jq program fails at preprocess", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].JQ = ".foo | select("
		err := PreprocessConfig(cfg)
		assert.ErrorContains(t, err, "invalid jq program")
	})

	t.Run("negative limit fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].Limit = -1
		err := PreprocessConfig(cfg)
		assert.ErrorContains(t, err, "limit")
	})
}

func TestPreprocessConfig_CountAndForEach(t *testing.T) {
	base := func() *config.Config {
		cfg := config.NewConfig()
		cfg.OutputFolder = t.TempDir()
		cfg.Steps = []config.Step{
			{Name: "seed", Prompt: "p", Model: "ollama:m", Count: 2},
			{Name: "per_row", Prompt: "use {{.item}}", Model: "ollama:m", ForEach: "seed"},
		}
		return cfg
	}

	t.Run("valid count and forEach pass", func(t *testing.T) {
		cfg := base()
		assert.NoError(t, PreprocessConfig(cfg))
	})

	t.Run("generator count stays 0 (default applied at runtime)", func(t *testing.T) {
		cfg := base()
		cfg.Steps[0].Count = 0
		assert.NoError(t, PreprocessConfig(cfg))
		assert.Equal(t, 0, cfg.Steps[0].Count, "runner.resolveIterations owns the default")
	})

	t.Run("item alias validates against forEach source schema", func(t *testing.T) {
		cfg := base()
		cfg.Steps[0].JSONSchemaRaw = `{
			"type": "object",
			"properties": {"title": {"type": "string"}, "tag": {"type": "string"}},
			"required": ["title", "tag"],
			"additionalProperties": false
		}`
		cfg.Steps[1].Prompt = "use {{.item.title}} and {{if .item.tag}}tagged{{end}}"
		assert.NoError(t, PreprocessConfig(cfg))
		assert.Contains(t, cfg.Steps[1].Prompt, "{{.item.title}}", "prompt is not rewritten; alias is semantic")
	})

	t.Run("item without forEach fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].ForEach = ""
		cfg.Steps[1].Count = 2
		assert.ErrorContains(t, PreprocessConfig(cfg), "no forEach")
	})

	t.Run("count and forEach together fail", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].Count = 5
		assert.ErrorContains(t, PreprocessConfig(cfg), "either 'count' or 'forEach'")
	})

	t.Run("forEach referencing unknown step fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].ForEach = "ghost"
		assert.ErrorContains(t, PreprocessConfig(cfg), "unknown step 'ghost'")
	})

	t.Run("forEach referencing itself fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].ForEach = "per_row"
		assert.ErrorContains(t, PreprocessConfig(cfg), "unknown step 'per_row'")
	})

	t.Run("negative count fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[0].Count = -1
		assert.ErrorContains(t, PreprocessConfig(cfg), "count")
	})

	t.Run("forEach on shell step fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps = append(cfg.Steps, config.Step{
			Name: "sh", Run: "echo hi > x.jsonl", OutputFilename: "x.jsonl", ForEach: "seed",
		})
		assert.ErrorContains(t, PreprocessConfig(cfg), "forEach")
	})

	t.Run("step named item is reserved", func(t *testing.T) {
		cfg := base()
		cfg.Steps[0].Name = "item"
		cfg.Steps[1].ForEach = "item"
		assert.ErrorContains(t, PreprocessConfig(cfg), "not allowed")
	})
}

func TestPreprocessConfig_PromptPlaceholders(t *testing.T) {
	base := func() *config.Config {
		cfg := config.NewConfig()
		cfg.OutputFolder = t.TempDir()
		cfg.Steps = []config.Step{
			{
				Name: "src", Prompt: "gen", Model: "ollama:m", Count: 2,
				JSONSchemaRaw: `{
					"type": "object",
					"properties": {"title": {"type": "string"}},
					"required": ["title"],
					"additionalProperties": false
				}`,
			},
			{Name: "use", Prompt: "x {{.src.title}}", Model: "ollama:m", ForEach: "src"},
		}
		return cfg
	}

	t.Run("valid cross-step reference passes", func(t *testing.T) {
		assert.NoError(t, PreprocessConfig(base()))
	})

	t.Run("SYSTEM placeholder is rejected (feature removed)", func(t *testing.T) {
		cfg := base()
		cfg.Steps[0].Prompt = "follow this schema: {{.SYSTEM.JSON_SCHEMA}}"
		assert.ErrorContains(t, PreprocessConfig(cfg), "unknown step 'SYSTEM'")
	})

	t.Run("self reference fails at config time", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].Prompt = "continue {{.use}}"
		assert.ErrorContains(t, PreprocessConfig(cfg), "unknown step 'use'")
	})

	t.Run("reference to later step fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[0].Prompt = "peek ahead {{.use.title}}"
		assert.ErrorContains(t, PreprocessConfig(cfg), "unknown step 'use'")
	})

	t.Run("field missing from source schema fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].Prompt = "x {{.src.nope}}"
		assert.ErrorContains(t, PreprocessConfig(cfg), "nope")
	})

	t.Run("field reference to schema-less prompt step fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[0].JSONSchemaRaw = nil
		assert.ErrorContains(t, PreprocessConfig(cfg), "JSON schema")
	})

	t.Run("mixing whole and field references to one step fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].Prompt = "all: {{.src}} title: {{.src.title}}"
		assert.ErrorContains(t, PreprocessConfig(cfg), "both as a whole")
	})
}

func TestPreprocessConfig_CollectValidation(t *testing.T) {
	t.Run("collect only valid on transform steps", func(t *testing.T) {
		cfg := config.NewConfig()
		cfg.OutputFolder = t.TempDir()
		cfg.Steps = []config.Step{
			{Name: "gen", Prompt: "p", Model: "ollama:m", Count: 2, Collect: true},
		}
		assert.ErrorContains(t, PreprocessConfig(cfg), "collect")
	})
}
