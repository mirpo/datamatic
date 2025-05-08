package promptbuilder

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
)

type Value struct {
	ID      string
	Step    string
	Key     string
	Content interface{}
}

type ValueShort struct {
	ID         string      `json:"id"`
	ComplexKey string      `json:"complexKey"`
	Content    interface{} `json:"content"`
}

type PromptBuilder struct {
	prompt       string
	newValues    map[string]Value
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
		parts := strings.Split(placeholder, ".")

		if len(parts) == 0 {
			continue
		}

		info := PlaceholderInfo{
			Step: parts[0],
		}

		if len(parts) > 1 {
			info.Key = strings.Join(parts[1:], ".")
		}

		placeholders[match[1]] = info
	}

	return placeholders
}

func NewPromptBuilder(prompt string) *PromptBuilder {
	placeholders := parseTemplatePlaceholders(prompt)

	return &PromptBuilder{
		prompt:       prompt,
		newValues:    make(map[string]Value),
		placeholders: placeholders,
	}
}

func (pb *PromptBuilder) AddValue(id string, step string, key string, value interface{}) {
	pb.newValues[fmt.Sprintf("%s-%s", step, key)] = Value{
		ID:      id,
		Step:    step,
		Key:     key,
		Content: value,
	}
}

func (pb *PromptBuilder) executeTemplate(tmplString string) (string, error) {
	tmpl, err := template.New("prompt").Option("missingkey=zero").Parse(tmplString)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// build map required by go template
	values := map[string]interface{}{}
	for _, value := range pb.newValues {
		if value.Key == "" {
			values[value.Step] = value.Content
		} else {
			existingVal, exist := values[value.Step]
			if !exist {
				values[value.Step] = map[string]interface{}{
					value.Key: value.Content,
				}
			} else {
				if stepMap, ok := existingVal.(map[string]interface{}); ok {
					stepMap[value.Key] = value.Content
				}
			}
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

func (pb *PromptBuilder) GetValues() []ValueShort {
	shortValues := []ValueShort{}

	for _, value := range pb.newValues {
		if strings.HasPrefix(value.Step, "SYSTEM") {
			continue
		}

		shortValues = append(shortValues, ValueShort{
			ID:         value.ID,
			ComplexKey: strings.Join([]string{value.Step, value.Key}, "."),
			Content:    value.Content,
		})
	}

	return shortValues
}
