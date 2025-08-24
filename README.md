# datamatic

[![Tests](https://github.com/mirpo/datamatic/actions/workflows/tests.yml/badge.svg)](https://github.com/mirpo/datamatic/actions/workflows/tests.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mirpo/datamatic)](https://golang.org/)
[![Release](https://img.shields.io/github/v/release/mirpo/datamatic)](https://github.com/mirpo/datamatic/releases)
[![License](https://img.shields.io/github/license/mirpo/datamatic)](https://github.com/mirpo/datamatic/blob/main/LICENSE)

**Generate high-quality synthetic data using local Large Language Models**

A powerful CLI tool for creating structured datasets with local LLMs, supporting JSON schema validation, multi-step chaining, and various AI providers.

## Features

### 🤖 AI Provider Support
- **[Ollama](https://ollama.com/download)** - Local model inference
- **[LM Studio](https://lmstudio.ai/download)** - Local model management
- **[OpenAI](https://openai.com/)** - Cloud-based models
- **[OpenRouter](https://openrouter.ai/)** - Multi-provider access
- **[Gemini](https://deepmind.google/models/gemini/)** - Gemini is a family of multimodal large language models (LLMs) developed by Google DeepMind

### 📊 Data Generation
- **JSON Schema Validation** - Structured output with type safety
- **Text Generation** - Flexible content creation
- **Multi-step Chaining** - Link generation steps together
- **Image Analysis** - Visual model integration

### 🔧 Extensibility
- **CLI Integration** - Use any command-line tool as a step
- **Dataset Loading** - Import from [Huggingface](https://huggingface.co/datasets)
- **Data Transformation** - Built-in [jq](https://github.com/jqlang/jq) support
- **Retry Logic** - Smart error handling and recovery

## Installation

### Homebrew

```shell
brew tap mirpo/homebrew-tools
brew install datamatic
```

### Using Go Install

```shell
go install github.com/mirpo/datamatic@latest
```

### From source

```bash
git clone https://github.com/mirpo/datamatic.git
cd datamatic
make build
```

## Quick Start

Create a configuration file and run datamatic:

```yaml
# config.yaml
version: 1.0
steps:
  - name: generate_titles
    model: ollama:llama3.2
    prompt: Generate a catchy news title
    jsonSchema:
      type: object
      properties:
        title:
          type: string
        tags:
          type: array
          items:
            type: string
      required:
        - title
        - tags
      additionalProperties: false  # Reject extra fields (default)
```

```bash
# Generate data
datamatic -config config.yaml

# With debug output
datamatic -config config.yaml -verbose -log-pretty
```

**Other providers:**
- OpenAI: `model: openai:gpt-4o-mini` + `export OPENAI_API_KEY=sk-...`
- OpenRouter: `model: openrouter:meta-llama/llama-3.2-3b` + `export OPENROUTER_API_KEY=sk-...`
- Gemini: `model: gemini:gemini-2.0-flash` + `export GEMINI_API_KEY=...`

## Output Format

Datamatic outputs structured data in JSONl format:

```go
type LineEntity struct {
	ID       string      `json:"id"`
	Format   string      `json:"format"`
	Prompt   string      `json:"prompt"`
	Response interface{} `json:"response"`
	Values   interface{} `json:"values"`
}
```

- **Format**: `text` or `json`
- **Response**: Generated content (text string or JSON object)
- **Values**: Linked step values for traceability

### Examples of JSONl results

**Text line**:

```json
{
  "id":"38082542-f352-44d2-88e9-6d68d28dcac4"
  "format":"text",
  "prompt":"Generate a catchy and one unique news title. Come up with a wildly different and surprising news headline. Return only one news title per request, without any extra thinking.",
  "response":"BREAKING: Giant Squid Found Wearing Tiny Top Hat and monocle in Remote Arctic Location"
}
```

**JSON line**:

```json
{
  "id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0",
  "format":"json",
  "prompt":"Provide up-to-date information about a randomly selected country, including its name, population, land area, UN membership status, capital city, GDP per capita, official languages, and year of independence. Return the data in a structured JSON format according to the schema below.",
  "response":{"capitalCity":"Bishkek","gdpPerCapita":1700,"independenceYear":1991,"isUNMember":true,"languages":["Kyr Kyrgyz","Russian"],"name":"Kyrgyzstan","population":6184000,"totalCountryArea":199912}
}
```

With values from linked steps:

```json
{
  "id":"dc140355-6c41-4ce7-9127-b8145cf1a23e",
  "format":"text",
  "prompt":"Write nice tourist brochure about country {{.about_country.name}}, which capital is {{.about_country.capitalCity}}, area {{.about_country.totalCountryArea}}, independenceYear: {{.about_country.independenceYear}} and official languages are {{.about_country.languages}}.",
  "response":"**Discover the Hidden Gem of Central Asia: Kyrgyzstan**\n\nTucked away in the heart of Central Asia, Kyrgyzstan is a land of breathtaking beauty, rich history, and warm hospitality. Our capital city, Bishkek, is a bustling metropolis surrounded by the stunning Tian Shan mountains, waiting to be explored.\n\n**A Brief History**\n\nKyrgyzstan gained its independence on August 31, 1991...",
  "values":{".about_country.capitalCity":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","content":"Bishkek"},".about_country.independenceYear":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","content":"1991"},".about_country.languages":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","content":"Kyr Kyrgyz, Russian"},".about_country.name":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","content":"Kyrgyzstan"},".about_country.totalCountryArea":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","content":"199912"}}
}
```

## CLI Reference

```bash
datamatic [OPTIONS]

Options:
  -config string
        Config file path
  -http-timeout int
        HTTP timeout: 0 - no timeout, if number - recommended to put high on poor hardware (default 300)
  -log-pretty
        Enable pretty logging, JSON when false (default true)
  -output string
        Output folder path (default "dataset")
  -skip-cli-warning
        Skip external CLI warning
  -validate-response
        Validate JSON response from server to match the schema (default true)
  -verbose
        Enable DEBUG logging level
  -version
        Get current version of datamatic
```

## Examples

| Example                                                                                                                             | Description                | Provider          |
| ----------------------------------------------------------------------------------------------------------------------------------- | -------------------------- | ----------------- |
| [Simple Text](./examples/v1/1.%20simple%20text%20generation,%20not%20linked%20steps/README.md)                                      | Basic text generation      | Ollama, LM Studio |
| [Simple JSON](./examples/v1/2.%20simple%20json%20generation,%20not%20linked%20steps/README.md)                                      | Basic JSON generation      | Ollama, LM Studio |
| [Linked Steps](./examples/v1/3.%20complex%20json,%20linked%20steps/README.md)                                                       | Multi-step JSON generation | Ollama            |
| [Huggingface + jq](./examples/v1/4.%20using%20huggingface%20and%20jq%20cli/README.md)                                               | Dataset transformation     | Ollama            |
| [DuckDB Integration](./examples/v1/5.%20using%20duckdb%20to%20convert%20parquet%20huggingface%20dataset%20and%20lmstudio/README.md) | Complex data processing    | LM Studio         |
| [Git Dataset](./examples/v1/6.%20git%20dataset/README.md)                                                                           | Version control data       | Ollama            |
| [Fine-tuning Data](./examples/v1/7.%20fine-tuning%20dataset/README.md)                                                              | Training dataset creation  | Ollama            |
| [Vision Models](./examples/v1/8.%20hugginface%20images%20and%20qwen2.5vl%20or%20gemma3/README.md)                                   | Image analysis             | Ollama, LM Studio |
| [OpenAI](./examples/v1/9.%20openai-example/README.md)                                                                               | Cloud provider usage       | OpenAI            |
| [Gemini](./examples/v1/10.%20openrouter-example/README.md)                                                                          | Cloud provider usage       | Gemini            |
