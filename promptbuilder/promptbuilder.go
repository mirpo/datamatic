package promptbuilder

import (
	"bytes"
	"text/template"
)

func GetCompiledPrompt(promptTemplate string, values map[string]interface{}) (string, error) {
	tmpl, err := template.New("prompt").Option("missingkey=zero").Parse(promptTemplate)
	if err != nil {
		return "", err
	}

	var output bytes.Buffer
	err = tmpl.Execute(&output, values)
	if err != nil {
		return "", err
	}

	return output.String(), nil
}
