# Structured extraction

One candidate-profile pipeline that shows three schema/template features together:

1. **Nested JSON schema** — `profile` generates objects and arrays nested inside arrays (`jobs[].achievements[]`, `skills.technical[]`).
2. **Both schema formats** — `profile` writes its schema as YAML-native; `seniority` writes the exact same kind of schema as a JSON string (`jsonSchema: |`). Both are supported.
3. **Native template values** — `brief` uses `{{len .item.skills.technical}}`, `{{if .item.jobs}}`, and `{{range .item.jobs}}` directly over the real JSON types (no transform step needed).

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
```
