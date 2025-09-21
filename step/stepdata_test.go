package step

import (
	"fmt"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/promptbuilder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUUIDFromString(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"example", "16eda250-39c7-3c5d-b2f8-7eb6dfedff40"},
		{"", "596b79dc-00dd-3991-a72f-d3696c38c64f"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, uuidFromString(tt.input), "input=%q", tt.input)
	}
}

func TestGetSourceDataFromLine(t *testing.T) {
	tests := []struct {
		name     string
		step     config.Step
		line     string
		wantData interface{}
		wantID   string
		wantErr  bool
	}{
		{
			name: "CLI step - valid JSON",
			step: config.Step{Type: config.CliStepType},
			line: `{"name": "test", "value": 123}`,
			wantData: map[string]interface{}{
				"name":  "test",
				"value": float64(123),
			},
		},
		{
			name:    "CLI step - invalid JSON",
			step:    config.Step{Type: config.CliStepType},
			line:    `{"invalid": json}`,
			wantErr: true,
		},
		{
			name: "Prompt step - valid JSONL",
			step: config.Step{Type: config.PromptStepType},
			line: `{"id": "test-id", "response": {"result": "success"}, "prompt": "test"}`,
			wantData: map[string]interface{}{
				"result": "success",
			},
			wantID: "test-id",
		},
		{
			name:    "Prompt step - invalid JSONL",
			step:    config.Step{Type: config.PromptStepType},
			line:    `{"invalid": jsonl}`,
			wantErr: true,
		},
		{
			name:    "Unsupported step type",
			step:    config.Step{Type: "unknown"},
			line:    `{"valid": "json"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotData, gotID, err := getSourceDataFromLine(tt.step, tt.line)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantID, gotID)
			assert.Equal(t, tt.wantData, gotData)
		})
	}
}

func TestReadStepValuesBatch(t *testing.T) {
	tests := []struct {
		name       string
		step       config.Step
		mockLine   string
		mockErr    error
		fieldPaths []string
		wantResult map[string]promptbuilder.StepValue
		wantErr    bool
	}{
		{
			name:       "CLI step - multiple fields",
			step:       config.Step{Type: config.CliStepType},
			mockLine:   `{"name": "test", "value": 123, "nested": {"field": "data"}}`,
			fieldPaths: []string{"name", "nested.field"},
			wantResult: map[string]promptbuilder.StepValue{
				"name": {
					ID:      "61df151d-7508-321d-ada6-27936752b809",
					Content: "test",
				},
				"nested.field": {
					ID:      "40bdd27d-55b6-3c98-8b60-f6901ee4cfd6",
					Content: "data",
				},
			},
		},
		{
			name: "Prompt step - no schema (string response)",
			step: config.Step{
				Type:       config.PromptStepType,
				JSONSchema: jsonschema.Schema{},
			},
			mockLine:   `{"id": "test-id", "response": "simple string", "prompt": "test"}`,
			fieldPaths: []string{"any"},
			wantResult: map[string]promptbuilder.StepValue{
				"any": {
					ID:      "test-id",
					Content: "simple string",
				},
			},
		},
		{
			name:       "File read error",
			step:       config.Step{Type: config.CliStepType},
			mockErr:    fmt.Errorf("mocked read error"),
			fieldPaths: []string{"name"},
			wantErr:    true,
		},
	}

	origRead := readLineFromFile
	defer func() { readLineFromFile = origRead }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readLineFromFile = func(_ string, _ int) (string, error) {
				return tt.mockLine, tt.mockErr
			}

			result, err := readStepValuesBatch(tt.step, "", 0, tt.fieldPaths)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}
