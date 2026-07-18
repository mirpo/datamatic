# Using Huggingface CLI and Ollama

Example shows complex dataset generation using dataset from Huggingface, filtering with a built-in jq transform step (no external jq needed) and four linked steps and Ollama

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull qwen3:1.7b`
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)

## Run dataset generation

`datamatic --config ./config.yaml --verbose`
