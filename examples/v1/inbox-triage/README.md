# Inbox triage

A real support-desk loop, no shell: **read a folder of incoming emails →
classify each with schema-guided reasoning → draft a suggested reply → write a
triage board (CSV) and a drafts digest (Markdown)**. Drop your own `.txt`
emails into `inbox/` and rerun.

**Features:** `read` (folder of files) · `SGR` · `forEach` · `transform` · `write` (csv + md)

## Steps

1. `emails` — `read: ./inbox/*.txt` → one row per file (`{path, name, content}`)
2. `triage` — `forEach` email → SGR `{reasoning, subject, category, priority, sentiment, summary}`
3. `board_rows` — **transform** drops the reasoning, keeping the scannable columns
4. `board` — `write: ./board.csv` → the triage board
5. `drafts` — `forEach` triage row → `{subject, reply}` (drafted from the summary, not the raw email)
6. `reply_digest` — `write: ./replies.md` → the suggested replies as a Markdown table

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
cat ./board.csv
cat ./replies.md
```
