package promptbuilder

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustBuilder(t *testing.T, prompt string) *PromptBuilder {
	t.Helper()
	pb, err := NewPromptBuilder(prompt, "")
	require.NoError(t, err)
	return pb
}

func parsePlaceholders(t *testing.T, input string) map[string]PlaceholderInfo {
	t.Helper()
	return mustBuilder(t, input).GetPlaceholders()
}

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
			actual := parsePlaceholders(t, tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestNewPromptBuilder(t *testing.T) {
	prompt := "Hello, {{.Name}}! Your ID is {{.User.ID}}."
	builder := mustBuilder(t, prompt)

	assert.Empty(t, builder.stepData)
	assert.Equal(t, map[string]PlaceholderInfo{
		".Name":    {Step: "Name"},
		".User.ID": {Step: "User", Key: "ID"},
	}, builder.placeholders)
}

func TestPromptBuilder_AddValue(t *testing.T) {
	builder := mustBuilder(t, "Test prompt with {{.Step1.KeyA}} and {{.Step2}}.")

	builder.AddValue("val1", "Step1", "KeyA", "valueA")
	builder.AddValue("val2", "Step2", "", "valueB")

	assert.Len(t, builder.stepData, 2)
	assert.Contains(t, builder.stepData, "Step1")
	assert.Equal(t, StepValue{ID: "val1", Content: "valueA"}, builder.stepData["Step1"]["KeyA"])
	assert.Contains(t, builder.stepData, "Step2")
	assert.Equal(t, StepValue{ID: "val2", Content: "valueB"}, builder.stepData["Step2"][""])
}

func TestPromptBuilder_AddStepValues(t *testing.T) {
	builder := mustBuilder(t, "Test prompt with {{.Step1.field1}} and {{.Step1.nested.field2}}.")

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
			builder := mustBuilder(t, tc.promptTemplate)
			if tc.setupFunc != nil {
				tc.setupFunc(builder)
			}
			output, err := builder.BuildPrompt()
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
	builder := mustBuilder(t, "{{.Step1.field1}} {{.Step1.nested.field}} {{.Step2.field2}}")
	groups := builder.GroupPlaceholdersByStep()

	// Check that each step has the right fields (order may vary)
	assert.Len(t, groups, 2)
	assert.Contains(t, groups["Step1"], "field1")
	assert.Contains(t, groups["Step1"], "nested.field")
	assert.Contains(t, groups["Step2"], "field2")
}

func TestPromptBuilder_BuildPrompt(t *testing.T) {
	builder := mustBuilder(t, "The item is {{.Item.Name}} with price {{.Item.Price}}.")
	builder.AddValue("item1", "Item", "Name", "Laptop")
	builder.AddValue("item2", "Item", "Price", "$1200")

	result, err := builder.BuildPrompt()
	assert.NoError(t, err)
	assert.Equal(t, "The item is Laptop with price $1200.", result)
}

func TestPromptBuilder_GetPlaceholders(t *testing.T) {
	prompt := "Name: {{.Person.Name}}, Age: {{.Person.Age}}, City: {{.Location}}."
	builder := mustBuilder(t, prompt)
	expected := map[string]PlaceholderInfo{
		".Person.Name": {Step: "Person", Key: "Name"},
		".Person.Age":  {Step: "Person", Key: "Age"},
		".Location":    {Step: "Location"},
	}
	assert.Equal(t, expected, builder.GetPlaceholders())
}

func TestPromptBuilder_HasPlaceholders(t *testing.T) {
	builder1 := mustBuilder(t, "This prompt has {{.Placeholder}}.")
	assert.True(t, builder1.HasPlaceholders())

	builder2 := mustBuilder(t, "This prompt has no placeholders.")
	assert.False(t, builder2.HasPlaceholders())
}

func TestPromptBuilder_GetValues(t *testing.T) {
	builder := mustBuilder(t, "Test prompt.")
	builder.AddValue("1", "USER", "name", "John Doe")
	builder.AddValue("2", "USER", "email", "john.doe@example.com")
	builder.AddValue("4", "INPUT", "query", "search term")

	expected := map[string]ValueShort{
		".USER.email":  {ID: "2", Value: "john.doe@example.com"},
		".USER.name":   {ID: "1", Value: "John Doe"},
		".INPUT.query": {ID: "4", Value: "search term"},
	}

	actual := builder.GetValues()
	assert.Equal(t, expected, actual)
}

func TestPromptBuilder_GetValues_WholeStepReference(t *testing.T) {
	builder := mustBuilder(t, "Test prompt.")
	builder.AddValue("1", "generate_cv", "", "full cv text") // whole-response reference

	actual := builder.GetValues()

	assert.Equal(t, map[string]ValueShort{
		".generate_cv": {ID: "1", Value: "full cv text"},
	}, actual, "no trailing dot for whole-step references")
}

func TestNewPromptBuilder_ItemAlias(t *testing.T) {
	t.Run("item placeholders point at the source step", func(t *testing.T) {
		pb, err := NewPromptBuilder(
			`{{.item.title}} {{len .item.tags}} {{range .item.jobs}}{{.name}}{{end}}`, "src")
		assert.NoError(t, err)
		groups := pb.GroupPlaceholdersByStep()
		assert.Contains(t, groups, "src")
		assert.NotContains(t, groups, "item")
		assert.ElementsMatch(t, []string{"title", "tags", "jobs"}, groups["src"])
	})

	t.Run("item without forEach fails", func(t *testing.T) {
		_, err := NewPromptBuilder("use {{.item.x}}", "")
		assert.ErrorContains(t, err, "no forEach")
	})

	t.Run("item renders the source step values", func(t *testing.T) {
		pb, err := NewPromptBuilder("v={{.item.title}} n={{len .item.tags}}", "src")
		assert.NoError(t, err)
		pb.AddValue("1", "src", "title", "hello")
		pb.AddValue("1", "src", "tags", List{"a", "b"})
		out, err := pb.BuildPrompt()
		assert.NoError(t, err)
		assert.Equal(t, "v=hello n=2", out)
	})
}

func TestWrap_RenderingMatchesLegacyStringification(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want string
	}{
		{"string", "hello", "hello"},
		{"integer stays verbatim", float64(6184000), "6184000"},
		{"float stays verbatim", 1700.5, "1700.5"},
		{"bool", true, "true"},
		{"array joined with comma", []interface{}{"Kyrgyz", "Russian"}, "Kyrgyz, Russian"},
		{"array of numbers", []interface{}{float64(1), float64(2)}, "1, 2"},
		{"object compact json", map[string]interface{}{"a": float64(1), "b": "x"}, `{"a":1,"b":"x"}`},
		{"nested object in array", []interface{}{map[string]interface{}{"n": "Acme"}}, `{"n":"Acme"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fmt.Sprint(Wrap(tt.in)))
		})
	}
}

func TestWrap_TemplateTraversal(t *testing.T) {
	row := Wrap(map[string]interface{}{
		"population": float64(6184000),
		"isUNMember": false,
		"languages":  []interface{}{"Kyrgyz", "Russian"},
		"companies": []interface{}{
			map[string]interface{}{"name": "Acme", "months": float64(26)},
		},
	})

	builder := mustBuilder(t,
		`{{if .row.isUNMember}}member{{else}}not-member{{end}};`+
			`n={{len .row.languages}};`+
			`{{range .row.companies}}{{.name}}/{{.months}}mo{{end}};`+
			`big={{if gt .row.population 1000000.0}}big{{else}}small{{end}}`)
	builder.AddValue("1", "row", "", row)

	out, err := builder.BuildPrompt()

	assert.NoError(t, err)
	assert.Equal(t, "not-member;n=2;Acme/26mo;big=big", out)
}

func TestNumber_MarshalsAsLiteral(t *testing.T) {
	b, err := json.Marshal(Wrap(map[string]interface{}{"pop": float64(6184000), "gdp": 1700.5}))
	assert.NoError(t, err)
	assert.JSONEq(t, `{"pop":6184000,"gdp":1700.5}`, string(b))
	assert.Contains(t, string(b), "6184000", "no scientific notation in marshaled output")
}

func TestParseTemplatePlaceholders_ScopedDotIsNotAStepRef(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]PlaceholderInfo
	}{
		{
			name:  "range body fields are element-relative, not step refs",
			input: `{{range .prev.jobs}}{{.name}} ({{.months}}mo){{end}}`,
			expected: map[string]PlaceholderInfo{
				".prev.jobs": {Step: "prev", Key: "jobs"},
			},
		},
		{
			name:  "if body keeps root scope",
			input: `{{if .prev.member}}{{.prev.name}}{{end}}`,
			expected: map[string]PlaceholderInfo{
				".prev.member": {Step: "prev", Key: "member"},
				".prev.name":   {Step: "prev", Key: "name"},
			},
		},
		{
			name:  "nested range only collects pipelines",
			input: `{{range .a.xs}}{{range .ys}}{{.z}}{{end}}{{end}}`,
			expected: map[string]PlaceholderInfo{
				".a.xs": {Step: "a", Key: "xs"},
			},
		},
		{
			name:  "len and gt argument fields are collected",
			input: `{{len .src.items}} {{if gt .src.count 5.0}}many{{end}}`,
			expected: map[string]PlaceholderInfo{
				".src.items": {Step: "src", Key: "items"},
				".src.count": {Step: "src", Key: "count"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parsePlaceholders(t, tt.input))
		})
	}
}
