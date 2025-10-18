package utils

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// ExpandEnv replaces ${VAR} or $VAR in the input string with environment variable values.
func ExpandEnv(s string, requiredVars []string) (string, error) {
	if len(requiredVars) > 0 {
		var missingVars []string
		for _, varName := range requiredVars {
			if _, exists := os.LookupEnv(varName); !exists {
				missingVars = append(missingVars, varName)
			}
		}

		if len(missingVars) > 0 {
			sort.Strings(missingVars)
			return "", fmt.Errorf("required environment variables not set: %s", strings.Join(missingVars, ", "))
		}
	}

	expanded := os.Expand(s, func(v string) string {
		if val, ok := os.LookupEnv(v); ok {
			return val
		}
		return "$" + v
	})

	return expanded, nil
}
