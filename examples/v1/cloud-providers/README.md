# Cloud providers

Dataset generation against hosted APIs — an e-commerce catalog batch — showing what matters for cloud runs: provider selection, parallel requests, and retries.

**Features:** `providers` · `concurrency` · `retry` · `jsonSchema`

## Steps

1. `product_descriptions` — free-text marketing copy, `concurrency: 5`
2. `product_specs` — structured catalog records (JSON schema), `concurrency: 3`

`retryConfig` retries transient failures (429 / 5xx / timeouts) with exponential backoff and fails fast on permanent errors (auth / bad request).

## Providers

All providers go through one OpenAI-compatible client, so only the model string and API key change:

| Provider | `model:` string | API key env var | Notes |
|---|---|---|---|
| OpenAI | `openai:gpt-4o-mini` | `OPENAI_API_KEY` | |
| OpenRouter | `openrouter:meta-llama/llama-3.2-3b` | `OPENROUTER_API_KEY` | for JSON, pick a model that supports [structured outputs](https://openrouter.ai/models?supported_parameters=structured_outputs) |
| Gemini | `gemini:gemini-2.0-flash` | `GEMINI_API_KEY` | |

This example uses OpenAI; swap the `model:` string and the key env var to use another.

## Requirements

- `datamatic`
- API key for your chosen provider

## Run

```bash
export OPENAI_API_KEY=sk-XXXXX
datamatic --config ./config.yaml --verbose
```
