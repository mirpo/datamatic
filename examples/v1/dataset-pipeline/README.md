# Dataset pipeline (fan-out / fan-in / rate / assemble)

The full instruction-dataset pattern in one config — the kind of pipeline used to build synthetic fine-tuning data, with zero external tools:

1. `generate_subtopics` — generate a list of Git subtopics
2. `split_into_unique_subtopics` — **transform fan-out**: 1 row → one row per unique subtopic
3. `generate_instructions` — `forEach` subtopic → many instructions
4. `split_into_unique_instructions` — **fan-in** (`collect: true`): dedupe instructions across all rows
5. `generate_answer` — `forEach` instruction → an answer
6. `instruction_response` — join answer with its source instruction via `$parent`
7. `rate` — score each pair on five dimensions
8. `result` — assemble final `{instruction, response, evaluation}` rows via `$parent`

Shows fan-out, fan-in (`collect`), `$parent` lineage, and multi-step `forEach` chaining together.
Inspired by [this synthetic-dataset walkthrough](https://towardsdatascience.com/create-a-synthetic-dataset-using-llama-3-1-405b-for-instruction-fine-tuning-9afc22fb6eef), but with no code.

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
```
