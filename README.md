# datamatic

[![Tests](https://github.com/mirpo/datamatic/actions/workflows/tests.yml/badge.svg)](https://github.com/mirpo/datamatic/actions/workflows/tests.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mirpo/datamatic)](https://golang.org/)
[![Release](https://img.shields.io/github/v/release/mirpo/datamatic)](https://github.com/mirpo/datamatic/releases)
[![License](https://img.shields.io/github/license/mirpo/datamatic)](https://github.com/mirpo/datamatic/blob/main/LICENSE)

Build multi-step AI workflows with schema-guided reasoning. Works with Ollama, LMStudio, OpenAI, OpenRouter, Gemini, and all the latest models for structured generation, chaining, and data processing.

## Features

### AI Provider Support
- **[Ollama](https://ollama.com/download)** - Local model inference
- **[LM Studio](https://lmstudio.ai/download)** - Local model management
- **[OpenAI](https://openai.com/)** - Cloud-based models
- **[OpenRouter](https://openrouter.ai/)** - Multi-provider access
- **[Gemini](https://deepmind.google/models/gemini/)** - Google DeepMind's multimodal LLMs

### Workflow Capabilities
- **JSON Schema Validation** - Structured output with type safety (YAML-native or JSON string formats)
- **Text Generation** - Flexible content creation
- **Explicit Iteration** - `count: N` for generators, `forEach: step` to run once per row of an earlier step; reference the current row as `{{.item.field}}`
- **Parallel Rows** - `concurrency: N` generates rows of a prompt step in parallel while keeping output in row order
- **Native Template Values** - referenced values keep their JSON types: `{{range .item.companies}}`, `{{len .item.tags}}`, `{{if .item.isActive}}` all work; arrays still print as `a, b` and numbers verbatim
- **Schema-Guided Reasoning (SGR)** - Guide LLMs through systematic analysis using structured schemas
- **Image Analysis** - Visual model integration

### Extensibility
- **CLI Integration** - Use any command-line tool as a step
- **Dataset Loading** - Import from [Huggingface](https://huggingface.co/datasets)
- **Transform Steps** - Embedded [jq](https://jqlang.github.io/jq/) (via gojq): filter, reshape, and fan out data between steps — no external binary needed
- **Environment Variables** - Dynamic configuration with `$VAR` syntax
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

## Use Cases

- **Synthetic Data Generation** - Create training datasets for fine-tuning LLMs
- **Document Classification** - Systematic analysis with structured reasoning
- **SQL Query Generation** - Chain-of-thought reasoning for complex queries
- **Multi-step Processing Pipelines** - CV analysis, data transformation, content generation
- **Vision Workflows** - Image analysis combined with text generation
- **Data Integration** - Combine HuggingFace datasets with LLM processing

## Quick Start

Create a configuration file and run datamatic:

```yaml
# config.yaml
version: 1.0
steps:
  - name: generate_titles
    model: ollama:llama3.2
    count: 5                    # generate 5 rows
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
      additionalProperties: false

  - name: analyze_title
    model: ollama:llama3.2
    forEach: generate_titles    # one iteration per generated title
    prompt: |
      Analyze this news title and provide sentiment and category analysis:
      Title: {{.item.title}}
    jsonSchema: |
      {
        "type": "object",
        "properties": {
          "sentiment": {"type": "string", "enum": ["positive", "negative", "neutral"]},
          "category": {"type": "string", "description": "News category"},
          "clickbait_score": {"type": "number", "minimum": 0, "maximum": 10}
        },
        "required": ["sentiment", "category", "clickbait_score"]
      }
```

```bash
# Generate data
datamatic --config config.yaml

# With debug output
datamatic --config config.yaml --verbose --log-pretty

# Check a config without running anything (great as a CI step for
# committed workflows): parses, preprocesses and validates — schemas,
# cross-step references, jq programs — and exits non-zero on any error
datamatic validate --config config.yaml
```

**Other providers:**
- OpenAI: `model: openai:gpt-4o-mini` + `export OPENAI_API_KEY=sk-...`
- OpenRouter: `model: openrouter:meta-llama/llama-3.2-3b` + `export OPENROUTER_API_KEY=sk-...`
- Gemini: `model: gemini:gemini-2.0-flash` + `export GEMINI_API_KEY=...`

### Parallel Generation

Rows of a prompt step are independent, so they can be generated in parallel:

```yaml
steps:
  - name: analyze
    model: openai:gpt-4o-mini
    forEach: documents
    concurrency: 5   # up to 5 rows generated at once (default: 1)
```

- Applies to **prompt steps only** (`count` or `forEach`); using it on transform or shell steps is a config error.
- Output stays in row order regardless of which request finishes first, so datasets remain deterministic.
- Raise it for cloud providers, which handle many parallel requests. Keep it low (or `1`) for a single local GPU — Ollama/LM Studio serve only a few requests at a time, so a high value won't help and may thrash.

### Transform Steps

Reshape, filter, and fan out data between steps with embedded [jq](https://jqlang.github.io/jq/) (via [gojq](https://github.com/itchyny/gojq) — no external binary needed):

```yaml
steps:
  - name: picked
    from: source_step
    jq: 'select(.score > 5) | {q: .question, a: .answer}'
    limit: 100
```

- `from` — source step; the jq program sees each row's value (for prompt steps: the `response`)
- `jq` — any jq program; emitting multiple values fans out (1 row → N rows), `select()` filters rows out
- `collect: true` — fan-in: the program runs once over an **array of all source rows** (`unique`, `group_by`, `sort_by` across the whole dataset)
- `sourceFormat: json` — the source file is a single JSON value (e.g. a pretty-printed array from an API dump) instead of JSONL
- `$parent` — per-row programs can reach the source row's lineage as `$parent.step.field` (e.g. carry the original chunk while fanning out extracted questions); not available with `collect`, where there is no single parent row
- `limit` — optional cap on output rows

Always wrap jq programs in single quotes: unquoted YAML silently truncates at `#`, misparses `{...}` object construction, and jq's own strings use double quotes anyway.

jq programs are validated when the config loads. Transform steps run instantly, produce regular JSONL, and don't trigger the external-CLI warning. See the [dataset-pipeline example](./examples/v1/dataset-pipeline/README.md), which uses fan-out and fan-in.

### Environment Variables

Configure your pipelines dynamically using `$VAR` syntax:

```yaml
version: 1.0

envVars:
  - PROVIDER
  - MODEL

steps:
  - name: generate
    model: $PROVIDER:$MODEL
    prompt: Generate a creative story
```

```bash
PROVIDER=ollama MODEL=llama3.2 datamatic --config config.yaml
```

Variables listed in `envVars` are validated before execution (fail-fast). See [env-and-workdir example](./examples/v1/env-and-workdir/README.md) for more details.

## Output Format

Datamatic outputs structured data in JSONl format:

```go
type LineEntity struct {
	ID       string                              `json:"id"`
	Format   string                              `json:"format"`
	Prompt   string                              `json:"prompt"`
	Response interface{}                         `json:"response"`
	Values   map[string]promptbuilder.ValueShort `json:"values,omitempty"`
}
```

- **Format**: `text` or `json`
- **Response**: Generated content (text string or JSON object)
- **Values**: Linked step values for traceability

### Output Examples

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
  "prompt":"Write nice tourist brochure about country Kyrgyzstan (a UN member state), which capital is Bishkek, area 199912, independenceYear: 1991 and official languages (2 total): Kyrgyz, Russian.",
  "response":"**Discover the Hidden Gem of Central Asia: Kyrgyzstan**\n\nTucked away in the heart of Central Asia, Kyrgyzstan is a land of breathtaking beauty, rich history, and warm hospitality. Our capital city, Bishkek, is a bustling metropolis surrounded by the stunning Tian Shan mountains, waiting to be explored.\n\n**A Brief History**\n\nKyrgyzstan gained its independence on August 31, 1991...",
  "values":{".about_country.capitalCity":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","value":"Bishkek"},".about_country.independenceYear":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","value":1991},".about_country.isUNMember":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","value":true},".about_country.languages":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","value":["Kyrgyz","Russian"]},".about_country.name":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","value":"Kyrgyzstan"},".about_country.totalCountryArea":{"id":"cc437b10-63c6-443a-9b3e-a7d6c51fc0a0","value":199912}}
}
```

## CLI Reference

```bash
datamatic [OPTIONS]            # run the workflow
datamatic validate [OPTIONS]   # check the config and exit (0 = valid)

Options:
  -config string
        Config file path
  -http-timeout int
        HTTP timeout: 0 - no timeout, if number - recommended to put high on poor hardware (default 300)
  -log-pretty
        Enable pretty logging, JSON when false (default true)
  -output string
        Output folder path (default "dataset")
  -validate-response
        Validate JSON response from server to match the schema (default true)
  -verbose
        Enable DEBUG logging level
  -version
        Get current version of datamatic
```

## Examples

See [`examples/v1/`](./examples/v1/) for the full feature matrix. Start with `basics`, then `linked-steps`.

### Getting started
| Example | Features shown | Backend |
| --- | --- | --- |
| [basics](./examples/v1/basics/README.md) | text generation, JSON schema | Ollama |
| [linked-steps](./examples/v1/linked-steps/README.md) | step chaining, native template values (`if`/`range`/`len`) | Ollama |
| [structured-extraction](./examples/v1/structured-extraction/README.md) | nested schema, both schema formats (YAML/JSON-string), native templates | Ollama |

### Datasets & reasoning
| Example | Features shown | Backend |
| --- | --- | --- |
| [dataset-pipeline](./examples/v1/dataset-pipeline/README.md) | transform fan-out, fan-in (`collect`, `$parent`), rating pipeline | Ollama |
| [sgr-reasoning](./examples/v1/sgr-reasoning/README.md) | schema-guided reasoning, `sourceFormat: json` | Ollama |
| [document-classification](./examples/v1/document-classification/README.md) | SGR classification, `collect` fan-in QA | Ollama |
| [document-qa](./examples/v1/document-qa/README.md) | RAG-style Q&A from a document, rating filter | Ollama |

### Data integration & infrastructure
| Example | Features shown | Backend |
| --- | --- | --- |
| [external-data](./examples/v1/external-data/README.md) | HuggingFace download + transform + shell tools | Ollama |
| [env-and-workdir](./examples/v1/env-and-workdir/README.md) | env vars, `workDir`, `$PROVIDER`, DuckDB | Ollama |
| [vision](./examples/v1/vision/README.md) | image → structured output (`imagePath`) | Ollama, LM Studio |

### Cloud
| Example | Features shown | Backend |
| --- | --- | --- |
| [cloud-providers](./examples/v1/cloud-providers/README.md) | provider selection, `concurrency`, `retryConfig` | OpenAI / OpenRouter / Gemini |
