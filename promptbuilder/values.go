package promptbuilder

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Wrap converts a decoded JSON value into template-friendly types: templates
// traverse them natively (range, len, index, field access, comparisons) while
// printing stays human-readable and byte-identical to the legacy
// stringification (arrays joined with ", ", objects as compact JSON, numbers
// verbatim without scientific notation).
func Wrap(v interface{}) interface{} {
	switch t := v.(type) {
	case []interface{}:
		out := make(List, len(t))
		for i, e := range t {
			out[i] = Wrap(e)
		}
		return out
	case map[string]interface{}:
		out := make(Object, len(t))
		for k, e := range t {
			out[k] = Wrap(e)
		}
		return out
	case float64:
		return Number(t)
	default:
		return v
	}
}

// Number renders and marshals verbatim (no scientific notation) while keeping
// a float64 kind, so template comparisons (gt, lt, eq) keep working.
type Number float64

func (n Number) String() string {
	return strconv.FormatFloat(float64(n), 'f', -1, 64)
}

func (n Number) MarshalJSON() ([]byte, error) {
	return []byte(n.String()), nil
}

// List prints as a comma-separated line; range/len/index work natively.
type List []interface{}

func (l List) String() string {
	parts := make([]string, len(l))
	for i, v := range l {
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, ", ")
}

// Object prints as compact JSON; field access works natively.
type Object map[string]interface{}

func (o Object) String() string {
	b, err := json.Marshal(map[string]interface{}(o))
	if err != nil {
		return fmt.Sprintf("%v", map[string]interface{}(o))
	}
	return string(b)
}
