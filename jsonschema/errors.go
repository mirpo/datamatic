package jsonschema

import (
	"fmt"
	"strings"
)

// ValidationError show multiple validation errors from JSON schema validation
type ValidationError struct {
	Errors []string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if len(e.Errors) == 1 {
		return fmt.Sprintf("validation failed: %s", e.Errors[0])
	}

	var b strings.Builder
	fmt.Fprintf(&b, "validation failed with %d errors:", len(e.Errors))
	for _, err := range e.Errors {
		fmt.Fprintf(&b, "\n  - %s", err)
	}
	return b.String()
}
