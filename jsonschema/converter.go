package jsonschema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func ConvertValueToString(value interface{}) string {
	if value == nil {
		return ""
	}

	val := reflect.ValueOf(value)

	switch val.Kind() {
	case reflect.String:
		return val.String()
	case reflect.Float32, reflect.Float64:
		v := val.Float()
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', 2, 64)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10)
	case reflect.Bool:
		return strconv.FormatBool(val.Bool())
	case reflect.Slice:
		var elements []string
		for i := range val.Len() {
			elements = append(elements, ConvertValueToString(val.Index(i).Interface()))
		}
		return strings.Join(elements, ", ")
	case reflect.Map:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("error marshalling map: %v", err)
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", value)
	}
}
