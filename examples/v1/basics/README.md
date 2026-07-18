# Basics: text and JSON generation

The two fundamental step outputs, in one config:

1. `title_text` — plain text generation (no schema)
2. `title_json` — structured JSON validated against a `jsonSchema` (`{title, tags[]}`)

Start here, then see [linked-steps](../linked-steps) for chaining steps together.

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
```
