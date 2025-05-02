package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirpo/datamatic/config"
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
			data, err := os.ReadFile(path)
			assert.NoError(t, err, "Failed to read file: %s", path)

			var cfg config.Config
			err = yaml.Unmarshal(data, &cfg)
			assert.NoError(t, err, "Failed to unmarshal YAML: %s", path)

			cfg.OutputFolder = "test"

			err = cfg.Validate()
			assert.NoError(t, err, "Validation failed for file: %s", path)
		})
	}
}
