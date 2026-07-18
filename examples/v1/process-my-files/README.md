# Process my files

Run an LLM over your **own local files** — no download, no shell step. `read`
turns a glob/directory/`.csv`/`.jsonl` into rows; here each `.md` support ticket
becomes a row, and each is triaged into a structured record.

**Features:** `read` · `forEach` · `jsonSchema`

## Steps

1. `tickets` — `read: ./docs/*.md` → one row per file: `{path, name, content}`
2. `triage` — `forEach` ticket → `{category, priority, summary}`

Point `read` at your own path to process your data. Other forms:
`read: ./data.csv` (one row per record), `read: ./seed.jsonl` (one row per line),
`read: ./docs/` (a whole directory). Override detection with `format:`.

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
```
