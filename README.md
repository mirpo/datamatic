# datamatic

Generate high-quality synthetic data using local Large Language Models (LLMs)

## Features

- LLM integration with popular LLM providers and all models they have under the hood (thanks all for the great tools!):
  - [Ollama](https://ollama.com/download)
  - [LM Studio](https://lmstudio.ai/download)
- Customizable text

## ⚠️ Important note

Before using this tool for synthetic data generation, please:

1. Review the licenses of the specific LLM models you plan to use
2. Check the model's terms of use regarding synthetic data generation
3. Be aware that some models may have restrictions on:
   - Commercial use of generated data
   - Generation of certain types of content
   - Usage in specific industries or applications
4. Verify the quality and accuracy of generated data before using it in production
5. Consider potential biases in the generated data

**Note**: The responsibility for checking and complying with model licenses and terms of use lies with the end user. This tool is provided as-is, and we recommend thoroughly reviewing the licensing terms of each model before deployment.

## Installation

[Homebrew](https://brew.sh/):
```shell
brew tap mirpo/homebrew-tools
brew install datamatic
```

Using `go install`:
```shell
go install github.com/mirpo/datamatic@latest
```

### Local Build
```shell
git clone https://github.com/mirpo/datamatic.git
cd datamatic
make build
```

## Quick Start

1. Create a configuration file `news_titles.yaml`:
```yaml
version: 1.0

steps:
  - name: generate_titles_simple
    model: ollama:llama3.1
    prompt: |
      Generate a catchy and one unique news title. Come up with a wildly different and surprising news headline. Return only one news title per request, without any extra thinking.
```

2. Run the generation:
```bash
datamatic -config news_titles.yaml
```

or to enable debug messages

```bash
datamatic -config news_titles.yaml -verbose
```

## CLI Flags

```
Usage of datamatic:
  -config string
      Config file path
  -http-timeout int
        HTTP timeout: 0 - no timeout, if number - recommended to put high on poor hardware (default 300)
  -log-pretty
        Enable pretty logging, JSON when false (default true)
  -output string
        Output folder path (default "dataset")
  -verbose
        Enable DEBUG logging level
  -version
        Get current version of datamatic
```

## More examples

| Name                                                                                                           | Provider (-s)     |
| -------------------------------------------------------------------------------------------------------------- | ----------------- |
| [Simple not linked text using Ollama and LM Studio](./examples/v1/1.%20simple%20not%20linked%20text/README.md) | Ollama, Lm Studio |
