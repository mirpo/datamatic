# Inbox triage

A real support-desk loop, no shell: **read a folder of incoming emails →
classify each with schema-guided reasoning → draft a suggested reply → write a
triage board (CSV), a drafts digest (Markdown), and one editable reply file per
ticket**. Drop your own `.txt` emails into `inbox/` and rerun.

**Features:** `read` (folder of files) · `SGR` · `forEach` · `transform` · `write` (aggregate + per-row)

## Steps

1. `emails` — `read: inbox/*.txt` → one row per file (`{path, name, content}`)
2. `triage` — `forEach` email → SGR `{reasoning, subject, category, priority, sentiment, summary}`
3. `board_rows` — **transform** drops the reasoning, keeping the scannable columns
4. `board` — `write: board.csv` → the triage board
5. `drafts` — `forEach` triage row → `{subject, reply}` (drafted from the summary, not the raw email)
6. `reply_digest` — `write: replies.md` → all drafts as one Markdown table (aggregate mode)
7. `reply_files` — `forEach` draft → `replies/<subject>.md`, one file per ticket whose body is the reply itself (per-row mode)

Steps 6 and 7 show the two write modes side by side: `from:` for one file with
every row, `forEach:` + `content:` for a folder of documents.

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
cat ./dataset/board.csv
cat ./dataset/replies.md
ls ./dataset/replies/
```
