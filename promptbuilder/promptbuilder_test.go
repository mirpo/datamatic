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
	assert.Empty(t, builder.stepData)
	assert.Equal(t, map[string]PlaceholderInfo{
		".Name":    {Step: "Name"},
		".User.ID": {Step: "User", Key: "ID"},
	}, builder.placeholders)
}

func TestPromptBuilder_AddValue(t *testing.T) {
	builder := NewPromptBuilder("Test prompt with {{.Step1.KeyA}} and {{.Step2}}.")

	builder.AddValue("val1", "Step1", "KeyA", "valueA")
	builder.AddValue("val2", "Step2", "", "valueB")

	assert.Len(t, builder.stepData, 2)
	assert.Contains(t, builder.stepData, "Step1")
	assert.Equal(t, StepValue{ID: "val1", Content: "valueA"}, builder.stepData["Step1"]["KeyA"])
	assert.Contains(t, builder.stepData, "Step2")
	assert.Equal(t, StepValue{ID: "val2", Content: "valueB"}, builder.stepData["Step2"][""])
}

func TestPromptBuilder_AddStepValues(t *testing.T) {
	builder := NewPromptBuilder("Test prompt with {{.Step1.field1}} and {{.Step1.nested.field2}}.")

	values := map[string]StepValue{
		"field1":        {ID: "id1", Content: "value1"},
		"nested.field2": {ID: "id2", Content: "value2"},
	}
	builder.AddStepValues("Step1", values)

	assert.Len(t, builder.stepData, 1)
	assert.Contains(t, builder.stepData, "Step1")
	assert.Equal(t, StepValue{ID: "id1", Content: "value1"}, builder.stepData["Step1"]["field1"])
	assert.Equal(t, StepValue{ID: "id2", Content: "value2"}, builder.stepData["Step1"]["nested.field2"])
}

func TestPromptBuilder_executeTemplate(t *testing.T) {
	testCases := []struct {
		name           string
		promptTemplate string
		setupFunc      func(*PromptBuilder)
		expectedOutput string
		expectErr      bool
	}{
		{
			name:           "Simple value replacement",
			promptTemplate: "Hello, {{.Name}}!",
			setupFunc: func(pb *PromptBuilder) {
				pb.AddValue("1", "Name", "", "World")
			},
			expectedOutput: "Hello, World!",
		},
		{
			name:           "Single-level nested",
			promptTemplate: "The API key is {{.Config.APIKey}}.",
			setupFunc: func(pb *PromptBuilder) {
				pb.AddValue("2", "Config", "APIKey", "secret123")
			},
			expectedOutput: "The API key is secret123.",
		},
		{
			name:           "Deep nested fields",
			promptTemplate: "Value: {{.Data.Deep.Nested.Field}}",
			setupFunc: func(pb *PromptBuilder) {
				pb.AddValue("3", "Data", "Deep.Nested.Field", "deep_value")
			},
			expectedOutput: "Value: deep_value",
		},
		{
			name:           "Multiple nested in same step",
			promptTemplate: "User: {{.User.Name}}, Age: {{.User.Profile.Age}}",
			setupFunc: func(pb *PromptBuilder) {
				pb.AddValue("4", "User", "Name", "Alice")
				pb.AddValue("5", "User", "Profile.Age", "30")
			},
			expectedOutput: "User: Alice, Age: 30",
		},
		{
			name:           "Mixed simple and nested",
			promptTemplate: "Hello {{.Name}}, your score is {{.Stats.Score}}.",
			setupFunc: func(pb *PromptBuilder) {
				pb.AddValue("6", "Name", "", "Bob")
				pb.AddValue("7", "Stats", "Score", "95")
			},
			expectedOutput: "Hello Bob, your score is 95.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewPromptBuilder(tc.promptTemplate)
			if tc.setupFunc != nil {
				tc.setupFunc(builder)
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

func TestPromptBuilder_GroupPlaceholdersByStep(t *testing.T) {
	builder := NewPromptBuilder("{{.Step1.field1}} {{.Step1.nested.field}} {{.Step2.field2}}")
	groups := builder.GroupPlaceholdersByStep()

	// Check that each step has the right fields (order may vary)
	assert.Len(t, groups, 2)
	assert.Contains(t, groups["Step1"], "field1")
	assert.Contains(t, groups["Step1"], "nested.field")
	assert.Contains(t, groups["Step2"], "field2")
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

	expected := map[string]ValueShort{
		".USER.email":  {ID: "2", Value: "john.doe@example.com"},
		".USER.name":   {ID: "1", Value: "John Doe"},
		".INPUT.query": {ID: "4", Value: "search term"},
	}

	actual := builder.GetValues()
	assert.Equal(t, expected, actual)
}
