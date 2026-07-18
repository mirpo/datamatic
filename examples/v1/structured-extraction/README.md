# Structured extraction

One candidate-profile pipeline that combines the three schema/template features: a deeply nested schema, both schema formats, and native template values over real JSON types.

**Features:** `jsonSchema (nested)` · `YAML + JSON-string schema` · `native-templates` · `forEach`

## Steps

1. `profile` — generate a candidate profile against a **deeply nested, YAML-native** schema (`jobs[].achievements[]`, `skills.technical[]`)
2. `seniority` — classify seniority using the same kind of schema written as a **JSON string** (`jsonSchema: |`)
3. `brief` — `forEach` profile, write a summary using `{{len .item.skills.technical}}`, `{{if .item.jobs}}`, `{{range .item.jobs}}`

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
```
