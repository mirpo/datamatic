package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				// no count/forEach: iterations resolved at runtime
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

	// Iteration settings
	assert.Equal(t, 0, cfg.Steps[0].Count, "no count/forEach: iterations resolved at runtime")
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
			"exactly one of 'prompt', 'run', 'jq', 'read' or 'write' must be defined",
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

func TestPreprocessConfig_Concurrency(t *testing.T) {
	base := func() *config.Config {
		cfg := config.NewConfig()
		cfg.OutputFolder = t.TempDir()
		cfg.Steps = []config.Step{
			{Name: "seed", Prompt: "p", Model: "ollama:m", Count: 2},
		}
		return cfg
	}

	t.Run("unset concurrency defaults to 1", func(t *testing.T) {
		cfg := base()
		assert.NoError(t, PreprocessConfig(cfg))
		assert.Equal(t, 1, cfg.Steps[0].Concurrency)
	})

	t.Run("explicit concurrency is preserved", func(t *testing.T) {
		cfg := base()
		cfg.Steps[0].Concurrency = 4
		assert.NoError(t, PreprocessConfig(cfg))
		assert.Equal(t, 4, cfg.Steps[0].Concurrency)
	})

	t.Run("negative concurrency fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[0].Concurrency = -1
		assert.ErrorContains(t, PreprocessConfig(cfg), "concurrency")
	})

	t.Run("concurrency on transform step fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps = append(cfg.Steps, config.Step{
			Name: "t", JQ: ".", From: "seed", Concurrency: 2,
		})
		assert.ErrorContains(t, PreprocessConfig(cfg), "concurrency")
	})

	t.Run("concurrency on shell step fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps = append(cfg.Steps, config.Step{
			Name: "sh", Run: "echo hi > x.jsonl", OutputFilename: "x.jsonl", Concurrency: 2,
		})
		assert.ErrorContains(t, PreprocessConfig(cfg), "concurrency")
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

func TestPreprocessConfig_SourceFormat(t *testing.T) {
	base := func() *config.Config {
		cfg := config.NewConfig()
		cfg.OutputFolder = t.TempDir()
		cfg.Steps = []config.Step{
			{Name: "src", Run: "echo '[]' > src.json", OutputFilename: "src.json"},
			{Name: "pick", JQ: `.[]`, From: "src", SourceFormat: "json"},
		}
		return cfg
	}

	t.Run("json format on transform passes", func(t *testing.T) {
		assert.NoError(t, PreprocessConfig(base()))
	})

	t.Run("unknown format fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].SourceFormat = "yaml"
		assert.ErrorContains(t, PreprocessConfig(cfg), "sourceFormat")
	})

	t.Run("sourceFormat on prompt step fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps = append(cfg.Steps, config.Step{
			Name: "gen", Prompt: "p", Model: "ollama:m", Count: 1, SourceFormat: "json",
		})
		assert.ErrorContains(t, PreprocessConfig(cfg), "sourceFormat")
	})

	t.Run("collect with json source fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].Collect = true
		assert.ErrorContains(t, PreprocessConfig(cfg), "collect")
	})
}

func TestLoadConfigFile(t *testing.T) {
	write := func(t *testing.T, content string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	valid := `
version: 1.0
steps:
  - name: gen
    model: ollama:m
    prompt: hi
    count: 2
  - name: pick
    from: gen
    jq: '.'
`

	t.Run("valid config loads, preprocesses and validates", func(t *testing.T) {
		cfg := config.NewConfig()
		cfg.ConfigFile = write(t, valid)
		cfg.OutputFolder = t.TempDir()

		err := LoadConfigFile(cfg)

		assert.NoError(t, err)
		assert.Len(t, cfg.Steps, 2)
		assert.Equal(t, config.TransformStepType, cfg.Steps[1].Type)
		assert.NotNil(t, cfg.Steps[1].JQProgram, "preprocessing ran")
	})

	t.Run("missing declared env var fails", func(t *testing.T) {
		cfg := config.NewConfig()
		cfg.ConfigFile = write(t, "version: 1.0\nenvVars:\n  - DATAMATIC_TEST_MISSING_VAR\nsteps:\n  - name: g\n    model: ollama:m\n    prompt: p\n")
		cfg.OutputFolder = t.TempDir()

		assert.ErrorContains(t, LoadConfigFile(cfg), "DATAMATIC_TEST_MISSING_VAR")
	})

	t.Run("missing file fails", func(t *testing.T) {
		cfg := config.NewConfig()
		cfg.ConfigFile = filepath.Join(t.TempDir(), "nope.yaml")
		cfg.OutputFolder = t.TempDir()

		assert.Error(t, LoadConfigFile(cfg))
	})
}

func TestPreprocessConfig_ReadStep(t *testing.T) {
	base := func(read, format string) *config.Config {
		cfg := config.NewConfig()
		cfg.OutputFolder = t.TempDir()
		cfg.Steps = []config.Step{{Name: "src", Read: read, Format: format}}
		return cfg
	}

	t.Run("infers read type and files format", func(t *testing.T) {
		cfg := base("./docs/*.md", "")
		assert.NoError(t, PreprocessConfig(cfg))
		assert.Equal(t, config.ReadStepType, cfg.Steps[0].Type)
		assert.Equal(t, config.ReadFormatFiles, cfg.Steps[0].Format)
	})
	t.Run("infers csv from extension", func(t *testing.T) {
		cfg := base("./leads.csv", "")
		assert.NoError(t, PreprocessConfig(cfg))
		assert.Equal(t, config.ReadFormatCSV, cfg.Steps[0].Format)
	})
	t.Run("infers jsonl from extension", func(t *testing.T) {
		cfg := base("./seed.jsonl", "")
		assert.NoError(t, PreprocessConfig(cfg))
		assert.Equal(t, config.ReadFormatJSONL, cfg.Steps[0].Format)
	})
	t.Run("explicit format overrides extension", func(t *testing.T) {
		cfg := base("./data.txt", config.ReadFormatCSV)
		assert.NoError(t, PreprocessConfig(cfg))
		assert.Equal(t, config.ReadFormatCSV, cfg.Steps[0].Format)
	})
	t.Run("unknown format fails", func(t *testing.T) {
		assert.ErrorContains(t, PreprocessConfig(base("./x", "parquet")), "unknown format")
	})
	t.Run("read plus count fails", func(t *testing.T) {
		cfg := base("./x/*.md", "")
		cfg.Steps[0].Count = 3
		assert.ErrorContains(t, PreprocessConfig(cfg), "only valid on prompt")
	})
	t.Run("read plus image fails", func(t *testing.T) {
		cfg := base("./x/*.md", "")
		cfg.Steps[0].Image = "*.jpg"
		assert.ErrorContains(t, PreprocessConfig(cfg), "image")
	})
}

func TestPreprocessConfig_WriteStep(t *testing.T) {
	base := func() *config.Config {
		cfg := config.NewConfig()
		cfg.OutputFolder = t.TempDir()
		cfg.Steps = []config.Step{
			{Name: "gen", Prompt: "p", Model: "ollama:m", Count: 2},
			{Name: "out", From: "gen", Write: "./out.csv"},
		}
		return cfg
	}

	t.Run("valid write infers csv", func(t *testing.T) {
		cfg := base()
		assert.NoError(t, PreprocessConfig(cfg))
		assert.Equal(t, config.WriteStepType, cfg.Steps[1].Type)
		assert.Equal(t, config.WriteFormatCSV, cfg.Steps[1].Format)
	})
	t.Run("missing from fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].From = ""
		assert.ErrorContains(t, PreprocessConfig(cfg), "'from' is required")
	})
	t.Run("unknown extension without format fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].Write = "./out.parquet"
		assert.ErrorContains(t, PreprocessConfig(cfg), "cannot infer output format")
	})
	t.Run("explicit format overrides extension", func(t *testing.T) {
		cfg := base()
		cfg.Steps[1].Write = "./out.dat"
		cfg.Steps[1].Format = config.WriteFormatJSON
		assert.NoError(t, PreprocessConfig(cfg))
	})
	t.Run("write step cannot be a from source", func(t *testing.T) {
		cfg := base()
		cfg.Steps = append(cfg.Steps, config.Step{Name: "again", From: "out", JQ: "."})
		assert.ErrorContains(t, PreprocessConfig(cfg), "cannot use write step 'out'")
	})
	t.Run("write step cannot be a forEach source", func(t *testing.T) {
		cfg := base()
		cfg.Steps = append(cfg.Steps, config.Step{Name: "again", ForEach: "out", Prompt: "p", Model: "ollama:m"})
		assert.ErrorContains(t, PreprocessConfig(cfg), "cannot use write step 'out'")
	})
	t.Run("format on a prompt step fails", func(t *testing.T) {
		cfg := base()
		cfg.Steps[0].Format = "csv"
		assert.ErrorContains(t, PreprocessConfig(cfg), "'format' is only valid on read and write")
	})
}

func TestPreprocessConfig_DataPathsResolveToConfigDir(t *testing.T) {
	// a genuinely absolute path for the current platform (a bare "/abs/x" is
	// NOT absolute on Windows, which needs a drive letter)
	absIn := filepath.Join(t.TempDir(), "x.csv")

	cfg := config.NewConfig()
	cfg.OutputFolder = t.TempDir()
	cfg.ConfigFile = filepath.Join("some", "dir", "config.yaml")
	cfg.Steps = []config.Step{
		{Name: "in", Read: "./data/*.md"},
		{Name: "rel_out", From: "in", Write: "results.csv"},
		{Name: "abs_in", Read: absIn},
	}

	require.NoError(t, PreprocessConfig(cfg))

	assert.Equal(t, filepath.Join("some", "dir", "data", "*.md"), cfg.Steps[0].Read, "relative read → config dir")
	assert.Equal(t, filepath.Join("some", "dir", "results.csv"), cfg.Steps[1].Write, "relative write → config dir")
	assert.Equal(t, absIn, cfg.Steps[2].Read, "absolute read unchanged")
}
