# Vision

Transcribe handwritten-math images to LaTeX with a vision model, then explain each formula — a multimodal dataset pipeline.

**Features:** `imagePath` · `shell` · `forEach`

## Steps

1. `download_images` — `hf download` + unzip + `magick` convert (BMP → JPEG)
2. `to_latex` — vision step over an `imagePath` glob → LaTeX transcription
3. `explain` — `forEach` → a step-by-step explanation of the formula

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) (or [LM Studio](https://lmstudio.ai/download)) + a vision model: `ollama pull qwen2.5vl:3b` (or `gemma3:4b`)
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli), [magick](https://imagemagick.org/script/download.php)

## Run

```bash
datamatic --config ./config.yaml --verbose
```
