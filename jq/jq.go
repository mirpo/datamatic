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
// Optional variable names (e.g. "$parent") declare named variables whose
// values are passed positionally to Run/RunEach; using an undeclared
// variable in the program fails here, at compile time.
func Compile(source string, variables ...string) (*Program, error) {
	query, err := gojq.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("invalid jq program %q: %w", source, err)
	}

	var opts []gojq.CompilerOption
	if len(variables) > 0 {
		opts = append(opts, gojq.WithVariables(variables))
	}

	code, err := gojq.Compile(query, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to compile jq program %q: %w", source, err)
	}

	return &Program{source: source, code: code}, nil
}

// Run applies the program to one input value and returns all emitted values
// (0..N — select() filters to 0, .items[] fans out to N).
func (p *Program) Run(input interface{}, variableValues ...interface{}) ([]interface{}, error) {
	var results []interface{}
	err := p.RunEach(input, func(v interface{}) (bool, error) {
		results = append(results, v)
		return false, nil
	}, variableValues...)
	return results, err
}

// RunEach applies the program and streams emitted values to emit; emit
// returning stop=true ends iteration early without materializing the rest.
// variableValues match the variables declared at Compile, in order.
func (p *Program) RunEach(input interface{}, emit func(v interface{}) (stop bool, err error), variableValues ...interface{}) error {
	iter := p.code.Run(input, variableValues...)
	for {
		v, ok := iter.Next()
		if !ok {
			return nil
		}
		if err, isErr := v.(error); isErr {
			return fmt.Errorf("jq program %q failed: %w", p.source, err)
		}

		stop, err := emit(v)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}
}
