package config

import (
	"testing"

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
}
