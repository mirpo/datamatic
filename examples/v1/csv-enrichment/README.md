# CSV enrichment

The full office loop with no shell: **read a CSV → enrich each row with an LLM → write a CSV**. Point it at your own spreadsheet of leads/rows to add labeled columns.

**Features:** `read` · `forEach` · `jsonSchema` · `write`

## Steps

1. `leads` — `read: ./leads.csv` → one row per record (columns become fields)
2. `classified` — `forEach` lead → `{company, industry, size}`
3. `report` — `write: ./enriched.csv` → the enriched rows as CSV

Output format is inferred from the extension (`.csv`); use `.json` for a JSON array, `.md` for a Markdown table, or set `format:` explicitly.

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
cat ./enriched.csv
```
