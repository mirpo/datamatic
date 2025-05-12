# Multi step git commands dataset using Ollama

Example shows dataset generation of git commands in natural language using Ollama.

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama run deepseek-r1:1.5b`
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
