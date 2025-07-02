# Using OpenRouter provider

Example shows dataset generation using OpenRouter models with both simple text generation and structured JSON output

## Requirements

Install:

- `datamatic`
- OpenRouter API key (set as `OPENROUTER_API_KEY` environment variable)
- Models for the text generation you can find https://openrouter.ai/models
- For the JSON generation you need to use models which support "structured_outputs" https://openrouter.ai/models?fmt=cards&supported_parameters=structured_outputs

## Run dataset generation

`export OPENROUTER_API_KEY=sk-or-v1-XXXX datamatic --config ./config.yaml --verbose`
