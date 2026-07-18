# Basics: text and JSON generation

The two fundamental step outputs in one config: free text, and structured JSON validated against a schema. Start here.

**Features:** `count` · `jsonSchema`

## Steps

1. `title_text` — plain text generation (no schema)
2. `title_json` — structured JSON validated against a `jsonSchema` (`{title, tags[]}`)

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
```
