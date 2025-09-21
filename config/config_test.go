package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "", cfg.ConfigFile)
	assert.False(t, cfg.Verbose)
	assert.True(t, cfg.LogPretty)
	assert.Equal(t, "dataset", cfg.OutputFolder)
	assert.Equal(t, 300, cfg.HTTPTimeout)
	assert.Equal(t, "", cfg.Version)
	assert.Nil(t, cfg.Steps)
	assert.True(t, cfg.RetryConfig.Enabled)
	assert.Equal(t, 3, cfg.RetryConfig.MaxAttempts)
	assert.Equal(t, time.Second, cfg.RetryConfig.InitialDelay)
	assert.Equal(t, 10*time.Second, cfg.RetryConfig.MaxDelay)
	assert.Equal(t, 2.0, cfg.RetryConfig.BackoffMultiplier)
}

func TestGetStepByName(t *testing.T) {
	step1 := Step{Name: "step1"}
	step2 := Step{Name: "step2"}
	config := &Config{Steps: []Step{step1, step2}}

	t.Run("Step exists", func(t *testing.T) {
		step := config.GetStepByName("step1")
		assert.NotNil(t, step)
		assert.Equal(t, "step1", step.Name)
	})

	t.Run("Step does not exist", func(t *testing.T) {
		step := config.GetStepByName("nonexistent")
		assert.Nil(t, step)
	})
}
