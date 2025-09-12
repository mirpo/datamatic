# CV Processing Pipeline with Flexible Schema Formats

Example shows 3-step CV processing: text generation, company extraction, education extraction. Demonstrates both YAML-native and JSON string schema formats in the same `jsonSchema` field.

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull llama3.2`

## Run dataset generation

`datamatic --config ./config.yaml --verbose`

## Features

- **Step 2**: YAML-native schema (clean & readable)
- **Step 3**: JSON string schema (copy-paste friendly)
- Multi-step pipeline with `{{.generate_cv}}` references
- Single file approach (no external schema files)
