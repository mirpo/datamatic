package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnv(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		envVars      map[string]string
		requiredVars []string
		want         string
		wantError    bool
	}{
		{
			name:    "simple variable",
			input:   "model: $PROVIDER",
			envVars: map[string]string{"PROVIDER": "ollama"},
			want:    "model: ollama",
		},
		{
			name:    "braced variable",
			input:   "model: ${PROVIDER}",
			envVars: map[string]string{"PROVIDER": "openai"},
			want:    "model: openai",
		},
		{
			name:    "multiple variables",
			input:   "model: $PROVIDER:$MODEL",
			envVars: map[string]string{"PROVIDER": "ollama", "MODEL": "llama3.2"},
			want:    "model: ollama:llama3.2",
		},
		{
			name:    "mixed syntax",
			input:   "path: ${DIR}/$FILE",
			envVars: map[string]string{"DIR": "downloads", "FILE": "data.csv"},
			want:    "path: downloads/data.csv",
		},
		{
			name:    "undefined variable is preserved (no required vars)",
			input:   "model: $UNDEFINED",
			envVars: map[string]string{},
			want:    "model: $UNDEFINED",
		},
		{
			name:         "missing required variable",
			input:        "model: $PROVIDER",
			envVars:      map[string]string{},
			requiredVars: []string{"PROVIDER"},
			wantError:    true,
		},
		{
			name:         "missing one of multiple required variables",
			input:        "model: $PROVIDER:$MODEL",
			envVars:      map[string]string{"PROVIDER": "ollama"},
			requiredVars: []string{"PROVIDER", "MODEL"},
			wantError:    true,
		},
		{
			name:    "empty string value is valid",
			input:   "value: $EMPTY",
			envVars: map[string]string{"EMPTY": ""},
			want:    "value: ",
		},
		{
			name:    "no variables",
			input:   "model: ollama:llama3.2",
			envVars: map[string]string{},
			want:    "model: ollama:llama3.2",
		},
		{
			name:    "variable in yaml structure",
			input:   "  workDir: $DOWNLOAD_DIR\n  outputFilename: $REQUIRED_FILE",
			envVars: map[string]string{"DOWNLOAD_DIR": "downloads", "REQUIRED_FILE": "prompts.csv"},
			want:    "  workDir: downloads\n  outputFilename: prompts.csv",
		},
		{
			name:    "jq variable preserved (not in env)",
			input:   `jq '.data as $chunk | {chunk: $chunk}'`,
			envVars: map[string]string{},
			want:    `jq '.data as $chunk | {chunk: $chunk}'`,
		},
		{
			name:    "mix of env and jq variables",
			input:   `jq -c '.value as $chunk | { dir: "$DIR" }'`,
			envVars: map[string]string{"DIR": "/tmp"},
			want:    `jq -c '.value as $chunk | { dir: "/tmp" }'`,
		},
		{
			name:    "datamatic internal syntax preserved",
			input:   "maxResults: convert_to_jsonl.$length",
			envVars: map[string]string{},
			want:    "maxResults: convert_to_jsonl.$length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got, err := ExpandEnv(tt.input, tt.requiredVars)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExpandEnvMultipleMissingRequiredVars(t *testing.T) {
	os.Clearenv()
	input := "model: $PROVIDER:$MODEL, dir: $DIR"
	requiredVars := []string{"PROVIDER", "MODEL", "DIR"}

	_, err := ExpandEnv(input, requiredVars)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "required environment variables not set")
	assert.Contains(t, err.Error(), "DIR, MODEL, PROVIDER")
}
