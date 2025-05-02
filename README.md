# datamatic

Generate high-quality synthetic data using local Large Language Models (LLMs)

## Features

- LLM integration with popular LLM providers and all models they have under the hood (thanks all for the great tools!):
  - [Ollama](https://ollama.com/download)
  - [LM Studio](https://lmstudio.ai/download)
- Customizable text and JSON generation.

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

### From Source

```bash
# Clone the repository
git clone https://github.com/mirpo/datamatic
cd datamatic

# Build the binary
go build -o datamatic

# Move to your PATH (optional)
sudo mv datamatic /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/mirpo/datamatic@latest
```

## Quick Start

1. Create a configuration file `news_titles.yaml`:
```yaml
version: 1.0
steps:
  - name: generate_titles
    model: ollama:llama3.1
    prompt: |
      Generate a catchy and one unique news title. Come up with a wildly different and surprising news headline. Return only one news title per request, without any extra thinking.
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
```

2. Run the generation:
```bash
datamatic -config news_titles.yaml
```

or to enable debug messages

```bash
datamatic -config news_titles.yaml -verbose
```

## Output format

`Datamatic` outputs in JSONl. Structure can be found in `jsonl/line.go`.

```go
type LineEntity struct {
	ID       string      `json:"id"`
	Format   string      `json:"format"`
	Prompt   string      `json:"prompt"`
	Response interface{} `json:"response"`
}
```

**Important notes:**
  - Format can be: text/JSON.
  - In case of text, response is a text.
  - In case of JSON, response is a JSON object.

### Examples of JSONl results

**Text line**:
```json
{"id":"b8f4ffbd-0d68-4caf-b2a4-6d4840a7df18","format":"text","prompt":"Generate a catchy and one unique news title. Come up with a wildly different and surprising news headline. Return only one news title per request, without any extra thinking.","response":"BIGFOOT CONFIRMED AS NEW CEO OF MAJOR TECH COMPANY IN SHOCKING STOCK MARKET SWOOP"}
```

**JSON line**:
```json
{"id":"7eb40ee7-fca2-4f5e-bd28-5bdb0d86ebcc","format":"json","prompt":"Generate a catchy and one unique news title. Come up with a wildly different and surprising news headline. Return only one news title per request, without any extra thinking.","response":{"tags":["robot","insects","bees","honey","pollination"],"title":"Robot Bees: The Buzz in Sustainable Agriculture"}}
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
  -validate-response
        Validate JSON response from server to match the schema (default true)
  -verbose
        Enable DEBUG logging level
  -version
        Get current version of datamatic
```

## More examples

| Name                                                                                                                                 | Provider (-s)     |
| ------------------------------------------------------------------------------------------------------------------------------------ | ----------------- |
| [Simple text generation using Ollama and LM Studio](./examples/v1/1.%20simple%20text%20generation,%20not%20linked%20steps/README.md) | Ollama, LM Studio |
| [Simple JSON generation using Ollama and LM Studio](./examples/v1/2.%20simple%20json%20generation,%20not%20linked%20steps/README.md) | Ollama, LM Studio |
