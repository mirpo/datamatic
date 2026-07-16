// Package jq wraps gojq so the rest of the codebase depends on one small API.
package jq

import (
	"fmt"

	"github.com/itchyny/gojq"
)

type Program struct {
	source string
	code   *gojq.Code
}

// Compile parses and compiles a jq program; errors are config-time errors.
func Compile(source string) (*Program, error) {
	query, err := gojq.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("invalid jq program %q: %w", source, err)
	}

	code, err := gojq.Compile(query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile jq program %q: %w", source, err)
	}

	return &Program{source: source, code: code}, nil
}

// Run applies the program to one input value and returns all emitted values
// (0..N — select() filters to 0, .items[] fans out to N).
func (p *Program) Run(input interface{}) ([]interface{}, error) {
	var results []interface{}

	iter := p.code.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return nil, fmt.Errorf("jq program %q failed: %w", p.source, err)
		}
		results = append(results, v)
	}

	return results, nil
}
