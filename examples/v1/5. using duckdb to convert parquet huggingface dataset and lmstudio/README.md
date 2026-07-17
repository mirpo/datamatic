# Complex dataset using DuckDb and LM studio

Example shows how to download a parquet dataset from Huggingface, convert Parquet to JSONL using DuckDB, take the top 100 rows and analyze the problem using LM Studio.

## Requirements

Install:

- `datamatic`
- [LM Studio](https://lmstudio.ai/download)
- Open LM Studio, find and download `hermes-3-llama-3.2-3b`
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)
- [DuckDB](https://duckdb.org/docs/installation/)

## Run dataset generation

`datamatic --config ./config.yaml --verbose`
