package promptbuilder

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"text/template/parse"

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
	tmpl         *template.Template
	stepData     map[string]map[string]StepValue // step -> fieldPath -> value
	placeholders map[string]PlaceholderInfo
	itemSource   string // forEach source step that {{.item}} aliases, if any
}

type PlaceholderInfo struct {
	Step string
	Key  string
}

// collectPlaceholders walks every tree of a parsed template (including
// {{define}}/{{block}} bodies) and collects field chains evaluated against
// the root context — i.e. real step references. Fields inside
// {{range}}/{{with}} bodies are element-relative (dot is rebound) and are
// NOT step references; {{if}} bodies keep the root scope.
func collectPlaceholders(tmpl *template.Template) map[string]PlaceholderInfo {
	placeholders := make(map[string]PlaceholderInfo)
	for _, t := range tmpl.Templates() {
		if t.Tree != nil {
			collectRootFields(t.Root, true, placeholders)
		}
	}
	return placeholders
}

func collectRootFields(node parse.Node, rootDot bool, out map[string]PlaceholderInfo) {
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, child := range n.Nodes {
			collectRootFields(child, rootDot, out)
		}
	case *parse.ActionNode:
		collectPipeFields(n.Pipe, rootDot, out)
	case *parse.RangeNode:
		collectPipeFields(n.Pipe, rootDot, out)
		collectRootFields(n.List, false, out) // dot rebinds to the element
		collectRootFields(n.ElseList, rootDot, out)
	case *parse.WithNode:
		collectPipeFields(n.Pipe, rootDot, out)
		collectRootFields(n.List, false, out) // dot rebinds to the value
		collectRootFields(n.ElseList, rootDot, out)
	case *parse.IfNode:
		collectPipeFields(n.Pipe, rootDot, out)
		collectRootFields(n.List, rootDot, out) // dot unchanged inside if
		collectRootFields(n.ElseList, rootDot, out)
	case *parse.TemplateNode:
		collectPipeFields(n.Pipe, rootDot, out)
	}
}

func collectPipeFields(pipe *parse.PipeNode, rootDot bool, out map[string]PlaceholderInfo) {
	if pipe == nil {
		return
	}
	for _, cmd := range pipe.Cmds {
		for _, arg := range cmd.Args {
			switch a := arg.(type) {
			case *parse.FieldNode:
				if rootDot {
					recordFieldChain(a.Ident, out)
				}
			case *parse.VariableNode:
				// $.step.field is always root-relative, even inside range
				if a.Ident[0] == "$" && len(a.Ident) > 1 {
					recordFieldChain(a.Ident[1:], out)
				}
			case *parse.ChainNode:
				// e.g. (index .src.list 0).name — collect from the base expression
				if pipe, ok := a.Node.(*parse.PipeNode); ok {
					collectPipeFields(pipe, rootDot, out)
				}
				if field, ok := a.Node.(*parse.FieldNode); ok && rootDot {
					recordFieldChain(field.Ident, out)
				}
			case *parse.PipeNode:
				collectPipeFields(a, rootDot, out)
			}
		}
	}
}

func recordFieldChain(ident []string, out map[string]PlaceholderInfo) {
	if len(ident) == 0 {
		return
	}
	info := PlaceholderInfo{Step: ident[0]}
	if len(ident) > 1 {
		info.Key = strings.Join(ident[1:], ".")
	}
	out["."+strings.Join(ident, ".")] = info
}

// ItemAliasName is the reserved placeholder name for the forEach source row;
// step names must not shadow it (enforced during preprocessing).
const ItemAliasName = "item"

// NewPromptBuilder parses the prompt template once (malformed templates fail
// here, i.e. at config time) and collects step references. {{.item...}}
// refers to the forEach source step: collected placeholders point at the
// source, and at render time .item shares the source step's values. The alias
// is semantic — it works anywhere in the template, including {{len .item.x}}
// and {{range .item.xs}}. Pass forEachSource="" for steps without forEach.
func NewPromptBuilder(prompt string, forEachSource string) (*PromptBuilder, error) {
	tmpl, err := template.New("prompt").Option("missingkey=zero").Parse(prompt)
	if err != nil {
		return nil, fmt.Errorf("invalid prompt template: %w", err)
	}

	placeholders := collectPlaceholders(tmpl)

	for key, info := range placeholders {
		if info.Step != ItemAliasName {
			continue
		}
		if forEachSource == "" {
			return nil, errors.New("prompt uses {{.item}} but the step has no forEach")
		}
		info.Step = forEachSource
		placeholders[key] = info
	}

	return &PromptBuilder{
		tmpl:         tmpl,
		stepData:     make(map[string]map[string]StepValue),
		placeholders: placeholders,
		itemSource:   forEachSource,
	}, nil
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

func setNestedValue(target Object, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := target
	for _, part := range parts[:len(parts)-1] {
		if next, ok := current[part].(Object); ok {
			current = next
		} else {
			next = make(Object)
			current[part] = next
			current = next
		}
	}
	current[parts[len(parts)-1]] = value
}

func (pb *PromptBuilder) BuildPrompt() (string, error) {
	values := make(map[string]interface{})
	for stepName, stepFields := range pb.stepData {
		stepObj := make(Object)
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

	if pb.itemSource != "" {
		if v, ok := values[pb.itemSource]; ok {
			values[ItemAliasName] = v
		}
	}

	log.Debug().Msgf("using values: %+v", values)

	var output bytes.Buffer
	if err := pb.tmpl.Execute(&output, values); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return output.String(), nil
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
		for fieldPath, stepValue := range stepFields {
			key := "." + stepName
			if fieldPath != "" {
				key += "." + fieldPath
			}
			resultValues[key] = ValueShort{
				ID:    stepValue.ID,
				Value: stepValue.Content,
			}
		}
	}

	return resultValues
}
