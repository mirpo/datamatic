# Simple Math Reasoning with MathInstruct Dataset

Example demonstrates structured step-by-step reasoning using MathInstruct dataset with nested JSON schemas and solution comparison.
Inspired by https://abdullin.com/schema-guided-reasoning/examples

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull qwen3:1.7b`
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)

## Run dataset generation

`datamatic --config ./config.yaml --verbose`
