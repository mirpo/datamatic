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
			name: "Shell step - valid JSON",
			step: config.Step{Type: config.ShellStepType},
			line: `{"name": "test", "value": 123}`,
			wantData: map[string]interface{}{
				"name":  "test",
				"value": float64(123),
			},
		},
		{
			name:    "Shell step - invalid JSON",
			step:    config.Step{Type: config.ShellStepType},
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
			name: "Transform step - object row",
			step: config.Step{Type: config.TransformStepType},
			line: `{"q": "question", "a": "answer"}`,
			wantData: map[string]interface{}{
				"q": "question",
				"a": "answer",
			},
		},
		{
			name:     "Transform step - scalar row",
			step:     config.Step{Type: config.TransformStepType},
			line:     `"just a string"`,
			wantData: "just a string",
		},
		{
			name:    "Transform step - invalid JSON",
			step:    config.Step{Type: config.TransformStepType},
			line:    `{oops`,
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
			gotData, gotID, _, err := getSourceDataFromLine(tt.step, tt.line)

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
			name:       "Shell step - multiple fields",
			step:       config.Step{Type: config.ShellStepType},
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
			step:       config.Step{Type: config.ShellStepType},
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

func TestReadStepValuesBatch_NativeValues(t *testing.T) {
	origRead := readLineFromFile
	defer func() { readLineFromFile = origRead }()
	readLineFromFile = func(_ string, _ int) (string, error) {
		return `{"id":"r1","format":"json","prompt":"p","response":{"pop":6184000,"member":false,"langs":["a","b"],"jobs":[{"name":"Acme","months":26}]}}`, nil
	}

	step := config.Step{Type: config.PromptStepType, JSONSchema: testSchema(t, `{
		"type":"object",
		"properties":{"pop":{"type":"integer"},"member":{"type":"boolean"},"langs":{"type":"array"},"jobs":{"type":"array"}},
		"required":["pop","member","langs","jobs"],
		"additionalProperties":false
	}`)}

	result, err := readStepValuesBatch(step, "", 0, []string{"pop", "member", "langs", "jobs"})
	require.NoError(t, err)

	assert.Equal(t, promptbuilder.Number(6184000), result["pop"].Content, "numbers stay numeric")
	assert.Equal(t, false, result["member"].Content, "booleans stay boolean")
	assert.Equal(t, promptbuilder.List{"a", "b"}, result["langs"].Content, "arrays stay iterable")
	jobs, ok := result["jobs"].Content.(promptbuilder.List)
	require.True(t, ok)
	assert.Equal(t, promptbuilder.Object{"name": "Acme", "months": promptbuilder.Number(26)}, jobs[0])
}

func TestExtractFieldByPath(t *testing.T) {
	testData := map[string]interface{}{
		"name": "John",
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"age":  30,
				"tags": []interface{}{"developer", "golang"},
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected interface{}
		errorMsg string
	}{
		{"top-level string", "name", "John", ""},
		{"nested integer stays native", "user.profile.age", 30, ""},
		{"nested array stays native", "user.profile.tags", []interface{}{"developer", "golang"}, ""},
		{"nested object stays native", "user.profile", testData["user"].(map[string]interface{})["profile"], ""},
		{"empty path returns whole value", "", testData, ""},
		{"non-existent field", "missing", nil, "field 'missing' not found at path 'missing'"},
		{"traverse non-object", "name.field", nil, "cannot traverse field 'field' on non-object type string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractFieldByPath(testData, tt.path)
			if tt.errorMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSourceDataFromLine_LineageValues(t *testing.T) {
	t.Run("prompt row returns raw lineage values", func(t *testing.T) {
		line := `{"id":"r1","format":"json","prompt":"p","response":{"ok":true},` +
			`"values":{".chopdoc.chunk":{"id":"c1","value":"source text"}}}`

		_, _, lineage, err := getSourceDataFromLine(config.Step{Type: config.PromptStepType}, line)

		require.NoError(t, err)
		assert.Equal(t, "source text", lineage[".chopdoc.chunk"].Value)
	})

	t.Run("shell row has no lineage", func(t *testing.T) {
		_, _, lineage, err := getSourceDataFromLine(config.Step{Type: config.ShellStepType}, `{"a":1}`)
		require.NoError(t, err)
		assert.Nil(t, lineage)
	})
}
