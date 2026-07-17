package utils

import (
	"fmt"
	"os"

	"github.com/mirpo/datamatic/config"
	"gopkg.in/yaml.v3"
)

// LoadConfigFile runs the full config pipeline for cfg.ConfigFile: read,
// fail-fast env-var expansion, strict YAML parse, preprocessing and
// validation. On success cfg holds the canonical, ready-to-run config.
func LoadConfigFile(cfg *config.Config) error {
	raw, err := os.ReadFile(cfg.ConfigFile)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	// pre-pass with a lenient decoder: only envVars is needed before expansion
	var envOnly struct {
		EnvVars []string `yaml:"envVars"`
	}
	if err := yaml.Unmarshal(raw, &envOnly); err != nil {
		return fmt.Errorf("parsing config file for env vars: %w", err)
	}

	expanded, err := ExpandEnv(string(raw), envOnly.EnvVars)
	if err != nil {
		return fmt.Errorf("expanding environment variables: %w", err)
	}

	if err := config.ParseYAML([]byte(expanded), cfg); err != nil {
		return err
	}

	if err := PreprocessConfig(cfg); err != nil {
		return err
	}

	return cfg.Validate()
}
