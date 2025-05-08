package promptbuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTemplatePlaceholders(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected map[string]PlaceholderInfo
	}{
		{
			name:  "Single simple placeholder",
			input: "Hello, {{.Name}}!",
			expected: map[string]PlaceholderInfo{
				".Name": {Step: "Name"},
			},
		},
		{
			name:  "Multiple simple placeholders",
			input: "Hello, {{.FirstName}} {{.LastName}}!",
			expected: map[string]PlaceholderInfo{
				".FirstName": {Step: "FirstName"},
				".LastName":  {Step: "LastName"},
			},
		},
		{
			name:  "Placeholder with key",
			input: "The value is {{.Config.APIKey}}.",
			expected: map[string]PlaceholderInfo{
				".Config.APIKey": {Step: "Config", Key: "APIKey"},
			},
		},
		{
			name:  "Multiple placeholders with and without keys",
			input: "Hello {{.User.Name}}, your ID is {{.UserID}} and config is {{.Config.Value}}.",
			expected: map[string]PlaceholderInfo{
				".User.Name":    {Step: "User", Key: "Name"},
				".UserID":       {Step: "UserID"},
				".Config.Value": {Step: "Config", Key: "Value"},
			},
		},
		{
			name:  "Placeholder with multiple key parts",
			input: "Access token: {{.Auth.Token.Value}}",
			expected: map[string]PlaceholderInfo{
				".Auth.Token.Value": {Step: "Auth", Key: "Token.Value"},
			},
		},
		{
			name:     "No placeholders",
			input:    "This is a plain string.",
			expected: map[string]PlaceholderInfo{},
		},
		{
			name:     "Invalid placeholder format",
			input:    "Hello, {.Name}!",
			expected: map[string]PlaceholderInfo{},
		},
		{
			name:  "Placeholder with extra spaces",
			input: "Value: {{ .Data.Field }}",
			expected: map[string]PlaceholderInfo{
				".Data.Field": {Step: "Data", Key: "Field"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := parseTemplatePlaceholders(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestNewPromptBuilder(t *testing.T) {
	prompt := "Hello, {{.Name}}! Your ID is {{.User.ID}}."
	builder := NewPromptBuilder(prompt)

	assert.Equal(t, prompt, builder.prompt)
	assert.Empty(t, builder.newValues)
	assert.Equal(t, map[string]PlaceholderInfo{
		".Name":    {Step: "Name"},
		".User.ID": {Step: "User", Key: "ID"},
	}, builder.placeholders)
}

func TestPromptBuilder_AddValue(t *testing.T) {
	builder := NewPromptBuilder("Test prompt with {{.Step1.KeyA}} and {{.Step2}}.")

	builder.AddValue("val1", "Step1", "KeyA", "valueA")
	builder.AddValue("val2", "Step2", "", "valueB")

	assert.Len(t, builder.newValues, 2)
	assert.Contains(t, builder.newValues, "Step1-KeyA")
	assert.Equal(t, Value{ID: "val1", Step: "Step1", Key: "KeyA", Content: "valueA"}, builder.newValues["Step1-KeyA"])
	assert.Contains(t, builder.newValues, "Step2-")
	assert.Equal(t, Value{ID: "val2", Step: "Step2", Key: "", Content: "valueB"}, builder.newValues["Step2-"])
}

func TestPromptBuilder_executeTemplate(t *testing.T) {
	testCases := []struct {
		name           string
		promptTemplate string
		addedValues    []Value
		expectedOutput string
		expectErr      bool
	}{
		{
			name:           "Simple value replacement",
			promptTemplate: "Hello, {{.Name}}!",
			addedValues: []Value{
				{ID: "1", Step: "Name", Key: "", Content: "World"},
			},
			expectedOutput: "Hello, World!",
		},
		{
			name:           "Nested value replacement",
			promptTemplate: "The API key is {{.Config.APIKey}}.",
			addedValues: []Value{
				{ID: "2", Step: "Config", Key: "APIKey", Content: "secret123"},
			},
			expectedOutput: "The API key is secret123.",
		},
		{
			name:           "Multiple value replacements",
			promptTemplate: "User: {{.User.Name}}, ID: {{.User.ID}}",
			addedValues: []Value{
				{ID: "3", Step: "User", Key: "Name", Content: "Alice"},
				{ID: "4", Step: "User", Key: "ID", Content: "42"},
			},
			expectedOutput: "User: Alice, ID: 42",
		},
		{
			name:           "Missing key handling",
			promptTemplate: "Value: {{.Missing}}",
			addedValues:    []Value{},
			expectedOutput: "Value: <no value>",
		},
		{
			name:           "Mixed simple and nested values",
			promptTemplate: "Hello {{.Name}}, config value is {{.Settings.Value}}.",
			addedValues: []Value{
				{ID: "5", Step: "Name", Key: "", Content: "Bob"},
				{ID: "6", Step: "Settings", Key: "Value", Content: "enabled"},
			},
			expectedOutput: "Hello Bob, config value is enabled.",
		},
		{
			name:           "Invalid template syntax",
			promptTemplate: "Hello, {{.Name!",
			addedValues: []Value{
				{ID: "7", Step: "Name", Key: "", Content: "Charlie"},
			},
			expectErr: true,
		},
		{
			name:           "Empty template",
			promptTemplate: "",
			addedValues:    []Value{},
			expectedOutput: "",
		},
		{
			name:           "No matching values",
			promptTemplate: "Nothing to replace here.",
			addedValues:    []Value{},
			expectedOutput: "Nothing to replace here.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewPromptBuilder(tc.promptTemplate)
			for _, val := range tc.addedValues {
				builder.AddValue(val.ID, val.Step, val.Key, val.Content)
			}
			output, err := builder.executeTemplate(tc.promptTemplate)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedOutput, output)
			}
		})
	}
}

func TestPromptBuilder_BuildPrompt(t *testing.T) {
	builder := NewPromptBuilder("The item is {{.Item.Name}} with price {{.Item.Price}}.")
	builder.AddValue("item1", "Item", "Name", "Laptop")
	builder.AddValue("item2", "Item", "Price", "$1200")

	result, err := builder.BuildPrompt()
	assert.NoError(t, err)
	assert.Equal(t, "The item is Laptop with price $1200.", result)
}

func TestPromptBuilder_GetPlaceholders(t *testing.T) {
	prompt := "Name: {{.Person.Name}}, Age: {{.Person.Age}}, City: {{.Location}}."
	builder := NewPromptBuilder(prompt)
	expected := map[string]PlaceholderInfo{
		".Person.Name": {Step: "Person", Key: "Name"},
		".Person.Age":  {Step: "Person", Key: "Age"},
		".Location":    {Step: "Location"},
	}
	assert.Equal(t, expected, builder.GetPlaceholders())
}

func TestPromptBuilder_HasPlaceholders(t *testing.T) {
	builder1 := NewPromptBuilder("This prompt has {{.Placeholder}}.")
	assert.True(t, builder1.HasPlaceholders())

	builder2 := NewPromptBuilder("This prompt has no placeholders.")
	assert.False(t, builder2.HasPlaceholders())
}

func TestPromptBuilder_GetValues(t *testing.T) {
	builder := NewPromptBuilder("Test prompt.")
	builder.AddValue("1", "USER", "name", "John Doe")
	builder.AddValue("2", "USER", "email", "john.doe@example.com")
	builder.AddValue("3", "SYSTEM", "version", "v1.0")
	builder.AddValue("4", "INPUT", "query", "search term")

	expected := []ValueShort{
		{ID: "1", ComplexKey: "USER.name", Content: "John Doe"},
		{ID: "2", ComplexKey: "USER.email", Content: "john.doe@example.com"},
		{ID: "4", ComplexKey: "INPUT.query", Content: "search term"},
	}

	actual := builder.GetValues()
	assert.ElementsMatch(t, expected, actual)
}
