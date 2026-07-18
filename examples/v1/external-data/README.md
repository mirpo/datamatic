# External data (HuggingFace + shell + transform)

Turn a HuggingFace code dataset into a code-explanation and unit-test dataset: download with a shell tool, filter with a built-in jq transform (no external jq), then two chained LLM steps.

**Features:** `huggingface` · `shell` · `transform` · `forEach`

## Steps

1. `download_dataset` — `hf download` a Python-code dataset
2. `pick_first_20` — **transform**: filter rows with jq, cap at 20
3. `explain_code` — `forEach` snippet → an explanation
4. `unit_test` — `forEach` → a runnable pytest file

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)

## Run

```bash
datamatic --config ./config.yaml --verbose
```
