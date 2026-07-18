# Linked steps

Generate structured records, then feed them into a second step — showing step chaining and native template values (a referenced value keeps its real JSON type, so `if`/`len` work).

**Features:** `forEach` · `jsonSchema` · `native-templates`

## Steps

1. `about_country` — generate structured facts about a country (`isUNMember` bool, `languages[]`, numbers)
2. `text_about_country` — `forEach` country, write a short brief using `{{if .item.isUNMember}}`, `{{len .item.languages}}`

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
```
