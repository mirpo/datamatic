package promptbuilder

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
)

type StepValue struct {
	ID      string
	Content interface{}
}

type ValueShort struct {
	ID    string      `json:"id"`
	Value interface{} `json:"value"`
}

type PromptBuilder struct {
	prompt       string
	stepData     map[string]map[string]StepValue // step -> fieldPath -> value
	placeholders map[string]PlaceholderInfo
}

type PlaceholderInfo struct {
	Step string
	Key  string
}

func parseTemplatePlaceholders(input string) map[string]PlaceholderInfo {
	re := regexp.MustCompile(`{{\s*(\.[^\s}]+)\s*}}`)
	matches := re.FindAllStringSubmatch(input, -1)

	placeholders := make(map[string]PlaceholderInfo)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		placeholder := strings.TrimPrefix(match[1], ".")
		if placeholder == "" {
			continue
		}

		parts := strings.SplitN(placeholder, ".", 2)
		info := PlaceholderInfo{Step: parts[0]}
		if len(parts) > 1 {
			info.Key = parts[1]
		}

		placeholders[match[1]] = info
	}

	return placeholders
}

func NewPromptBuilder(prompt string) *PromptBuilder {
	placeholders := parseTemplatePlaceholders(prompt)

	return &PromptBuilder{
		prompt:       prompt,
		stepData:     make(map[string]map[string]StepValue),
		placeholders: placeholders,
	}
}

// AddStepValues adds multiple values for a step in one batch operation
func (pb *PromptBuilder) AddStepValues(stepName string, values map[string]StepValue) {
	if pb.stepData[stepName] == nil {
		pb.stepData[stepName] = make(map[string]StepValue)
	}
	for fieldPath, value := range values {
		pb.stepData[stepName][fieldPath] = value
	}
}

// AddValue maintains backward compatibility for individual value additions
func (pb *PromptBuilder) AddValue(id string, step string, key string, value interface{}) {
	if pb.stepData[step] == nil {
		pb.stepData[step] = make(map[string]StepValue)
	}
	pb.stepData[step][key] = StepValue{
		ID:      id,
		Content: value,
	}
}

func setNestedValue(target map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := target
	for _, part := range parts[:len(parts)-1] {
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			next = make(map[string]interface{})
			current[part] = next
			current = next
		}
	}
	current[parts[len(parts)-1]] = value
}

func (pb *PromptBuilder) executeTemplate(tmplString string) (string, error) {
	tmpl, err := template.New("prompt").Option("missingkey=zero").Parse(tmplString)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	values := make(map[string]interface{})
	for stepName, stepFields := range pb.stepData {
		stepObj := make(map[string]interface{})
		for fieldPath, stepValue := range stepFields {
			if fieldPath == "" {
				values[stepName] = stepValue.Content
			} else {
				setNestedValue(stepObj, fieldPath, stepValue.Content)
			}
		}
		if len(stepObj) > 0 {
			values[stepName] = stepObj
		}
	}

	log.Debug().Msgf("using values: %+v", values)

	var output bytes.Buffer
	err = tmpl.Execute(&output, values)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return output.String(), nil
}

func (pb *PromptBuilder) BuildPrompt() (string, error) {
	return pb.executeTemplate(pb.prompt)
}

func (pb *PromptBuilder) GetPlaceholders() map[string]PlaceholderInfo {
	return pb.placeholders
}

func (pb *PromptBuilder) HasPlaceholders() bool {
	return len(pb.placeholders) > 0
}

// GroupPlaceholdersByStep groups placeholders by step name for batch processing
func (pb *PromptBuilder) GroupPlaceholdersByStep() map[string][]string {
	groups := make(map[string][]string)
	for _, placeholder := range pb.placeholders {
		groups[placeholder.Step] = append(groups[placeholder.Step], placeholder.Key)
	}
	return groups
}

func (pb *PromptBuilder) GetValues() map[string]ValueShort {
	resultValues := map[string]ValueShort{}

	for stepName, stepFields := range pb.stepData {
		if strings.HasPrefix(stepName, "SYSTEM") {
			continue
		}

		for fieldPath, stepValue := range stepFields {
			key := "." + strings.Join(append([]string{stepName}, fieldPath), ".")
			resultValues[key] = ValueShort{
				ID:    stepValue.ID,
				Value: stepValue.Content,
			}
		}
	}

	return resultValues
}
