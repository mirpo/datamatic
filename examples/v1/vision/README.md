# Vision

Run a vision model over your **own local images**: `read` enumerates a folder of
images, and each prompt call attaches the current row's file as a vision image
via `image: {{.item.path}}`. Here each image gets a structured alt-text record
(description + colors + tags) — the kind of set used for cataloging or
accessibility.

**Features:** `read` · `image:` (vision attach) · `forEach` · `jsonSchema`

## Steps

1. `images` — `read: ./images/*.jpg` → one row per image: `{path, name, content}`
2. `describe` — `forEach` image, `image: {{.item.path}}` attaches it → `{description, main_colors[], tags[]}`

Point `read` at your own image folder to process your files.

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) (or [LM Studio](https://lmstudio.ai/download)) + a vision model: `ollama pull qwen2.5vl:3b` (or `gemma3:4b`)

## Run

```bash
datamatic --config ./config.yaml --verbose
```
