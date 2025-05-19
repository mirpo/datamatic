# Multi step git commands dataset using Ollama

Example shows dataset generation of git commands in natural language using Ollama
Inspired by: https://towardsdatascience.com/create-a-synthetic-dataset-using-llama-3-1-405b-for-instruction-fine-tuning-9afc22fb6eef
But with 0 coding.

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull llama3.2`

## Run dataset generation

`datamatic --config ./config.yaml --verbose`
