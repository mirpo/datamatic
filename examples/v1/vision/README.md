# Using vision model

Example shows complex dataset generation using a dataset from Huggingface and a vision model (Qwen2.5-VL or Gemma 3) via Ollama or LM Studio

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download) or [LM Studio](https://lmstudio.ai/download)
  - Ollama models:
    - `ollama pull qwen2.5vl:3b` or `ollama pull gemma3:4b` or any from this list https://ollama.com/search?c=vision&o=newest
  - LM Studio:
    - `gemma3:4b`
- [magick](https://imagemagick.org/script/download.php) (to convert dataset images from BMP => JPEG)
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)

## Run dataset generation

`datamatic --config ./config.yaml --verbose`
