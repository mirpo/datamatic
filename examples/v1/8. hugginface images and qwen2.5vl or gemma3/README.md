# Using vision model

Example shows complex dataset generation using dataset from Huggingface and Gemma3 visual model using Ollama

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download) or [LM Studio](https://lmstudio.ai/download)
  - Ollama models:
    - `ollama pull qwen2.5vl:3b` or `ollama pull gemma3:4b` or any from this list https://ollama.com/search?c=vision&o=newest
  - LM Studio:
    - `gemma3:4b`
- [magick](https://imagemagick.org/script/download.php) (to convert dataset images from BMP => JPEG)
- [huggingface-cli](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)
- [jq](https://github.com/jqlang/jq)

## Run dataset generation

`datamatic --config ./config.yaml --verbose`
