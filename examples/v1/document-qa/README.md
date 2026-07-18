# Fine-tuning Q&A dataset from a document (RAG-style) using Ollama

Example builds a fine-tuning dataset from a source document: it downloads an essay, splits it into chunks, generates cognitive-level (Bloom's taxonomy) questions with supporting text evidence per chunk, then validates and rates each question and keeps only the high-quality ones.

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull deepseek-r1:1.5b`
- Install chopdoc to split the document: 
  ```shell
  brew tap mirpo/homebrew-tools
  brew install chopdoc
  ```
  or
  ```shell
  go install github.com/mirpo/chopdoc@latest
  ```

## Run dataset generation

`datamatic --config ./config.yaml --verbose`
