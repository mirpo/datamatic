# Schema-guided reasoning

Generate step-by-step reasoning traces for math problems from a HuggingFace dataset. The schema forces the model to emit explicit reasoning steps before a final answer (SGR).

**Features:** `SGR` · `sourceFormat: json` · `transform` · `forEach`

## Steps

1. `download_dataset` — `hf download` the MathInstruct dataset (a pretty-printed JSON array)
2. `math_problems_10` — **transform** with `sourceFormat: json` reads the whole-file array and slices the first 10 (no external jq)
3. `solve_with_reasoning` — `forEach` problem → `{steps[]: {explanation, output}, final_answer}`

Inspired by <https://abdullin.com/schema-guided-reasoning/examples>.

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)

## Run

```bash
datamatic --config ./config.yaml --verbose
```
