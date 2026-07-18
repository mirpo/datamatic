# Dataset pipeline (fan-out / fan-in / rate / assemble)

The full synthetic instruction-dataset pattern in one config, with zero external tools — the shape used to build fine-tuning data.

**Features:** `transform` · `fan-out` · `collect` · `$parent` · `forEach`

## Steps

1. `generate_subtopics` — generate a list of Git subtopics
2. `split_into_unique_subtopics` — **fan-out**: 1 row → one row per unique subtopic
3. `generate_instructions` — `forEach` subtopic → many instructions
4. `split_into_unique_instructions` — **fan-in** (`collect: true`): dedupe across all rows
5. `generate_answer` — `forEach` instruction → an answer
6. `instruction_response` — join answer with its instruction via `$parent`
7. `rate` — score each pair on five dimensions
8. `result` — assemble final `{instruction, response, evaluation}` rows

Inspired by [this synthetic-dataset walkthrough](https://towardsdatascience.com/create-a-synthetic-dataset-using-llama-3-1-405b-for-instruction-fine-tuning-9afc22fb6eef), but with no code.

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
```
