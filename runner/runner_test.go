package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestValidateConfigFromExamples(t *testing.T) {
	matches, err := filepath.Glob("../examples/**/**/*.yaml")
	assert.NoError(t, err, "Failed to glob YAML files")

	assert.NotEmpty(t, matches, "No YAML files found in examples directory")

	for _, path := range matches {
		t.Logf("Testing file: %s", path)

		testName := filepath.ToSlash(path)

		t.Run(testName, func(t *testing.T) {
			setupTestEnvVars(t, path)

			data, err := os.ReadFile(path)
			assert.NoError(t, err, "Failed to read file: %s", path)

			expandedYaml, err := utils.ExpandEnv(string(data), nil)
			assert.NoError(t, err, "Failed to expand env vars for file: %s", path)

			var cfg config.Config
			err = yaml.Unmarshal([]byte(expandedYaml), &cfg)
			assert.NoError(t, err, "Failed to unmarshal YAML: %s", path)

			cfg.OutputFolder = "test"
			cfg.SkipCliWarning = true

			err = utils.PreprocessConfig(&cfg)
			assert.NoError(t, err, "Preprocessing failed for file: %s", path)

			err = cfg.Validate()
			assert.NoError(t, err, "Validation failed for file: %s", path)
		})
	}
}

// setupTestEnvVars sets up environment variables needed for specific test configs
func setupTestEnvVars(t *testing.T, path string) {
	// For the workdir-multi-stage-pipeline example that uses env vars
	if filepath.Base(filepath.Dir(path)) == "18. workdir-multi-stage-pipeline" {
		os.Setenv("REQUIRED_FILE", "prompts.csv")
		os.Setenv("DOWNLOAD_DIR", "downloads")
		os.Setenv("PROVIDER", "ollama")
		os.Setenv("MODEL", "llama3.2")
		t.Cleanup(func() {
			os.Unsetenv("REQUIRED_FILE")
			os.Unsetenv("DOWNLOAD_DIR")
			os.Unsetenv("PROVIDER")
			os.Unsetenv("MODEL")
		})
	}
}
