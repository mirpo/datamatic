# Complex dataset using DuckDb and LM studio

Example shows how to download parquet dataset from Huggingface, convert to Parquet to JSONL using DuckDb, take top 100 and analyze the problem using LM Studio.

## Requirements

Install:

- `datamatic`
- [LM Studio](https://lmstudio.ai/download)
- Install model: `ollama pull llama3.2`
- Open LM Studio, find and download `hermes-3-llama-3.2-3b`
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)
- [jq](https://github.com/jqlang/jq)
- [DuckDB](https://duckdb.org/docs/installation/)

## Run dataset generation

`datamatic --config ./config.yaml --verbose`
