# datamatic

Generate high-quality synthetic data using local Large Language Models (LLMs)

## Features

- LLM integration with popular LLM providers and all models they have under the hood (thanks all for the great tools!):
  - [Ollama](https://ollama.com/download)
  - [LM Studio](https://lmstudio.ai/download)
  - Support for other LLMs is planned.
- Customizable text and JSON generation.
- Multi step chaining.
- Use any CLI as a step. For example:
  - Load datasets from [Huggingface](https://huggingface.co/datasets).
  - Run [jq](https://github.com/jqlang/jq) to transform data between steps.
- Image analysis using visual models.
- Automatic retry logic with smart error handling for improved reliability.

## ⚠️ Important notes

Before using this tool for synthetic data generation, please ensure you:

1. Review the licenses for the specific LLM models you plan to use.
2. Check the models' terms of service regarding synthetic data generation.
3. Be aware of potential restrictions, which may include limitations on:
  - Commercial use of generated data.
  - Generating specific content types.
  - Usage in certain industries or applications.
4. Verify the quality and accuracy of generated data before using it in production.
5. Consider potential biases in the generated data.

**Important**: The end user is responsible for checking and complying with model licenses and terms. This tool is provided "as-is," and we strongly recommend reviewing the licensing terms of each model before deployment.

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

1. Create a configuration file `news_titles.yaml`:
```yaml
version: 1.0
steps:
  - name: generate_titles
    model: ollama:llama3.2
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
	Values   interface{} `json:"values"`
}
```

**Important notes:**
  - Format can be: text/JSON.
  - In case of text, response is a text.
  - In case of JSON, response is a JSON object.
  - When steps are linked, `values` contains values from linked steps for traceability.

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
  -skip-cli-warning
        Skip external CLI warning (default true)
  -validate-response
        Validate JSON response from server to match the schema (default true)
  -verbose
        Enable DEBUG logging level
  -version
        Get current version of datamatic
```

## More examples

| Name                                                                                                                                                                     | Provider (-s)     |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------- |
| [Simple text generation, not linked steps](./examples/v1/1.%20simple%20text%20generation,%20not%20linked%20steps/README.md)                                              | Ollama, LM Studio |
| [Simple JSON generation, not linked steps](./examples/v1/2.%20simple%20json%20generation,%20not%20linked%20steps/README.md)                                              | Ollama, LM Studio |
| [Complex JSON generation, linked steps](./examples/v1/3.%20complex%20json,%20linked%20steps/README.md)                                                                   | Ollama            |
| [Using Huggingface dataset and jq cli, linked steps](./examples/v1/4.%20using%20huggingface%20and%20jq%20cli/README.md)                                                  | Ollama            |
| [Complex dataset using DuckDb, Huggingface and LM studio](./examples/v1/5.%20using%20duckdb%20to%20convert%20parquet%20huggingface%20dataset%20and%20lmstudio/README.md) | LM Studio         |
| [Git dataset](./examples/v1/6.%20git%20dataset/README.md)                                                                                                                | Ollama            |
| [Creating dataset for fine-tuning](./examples/v1/7.%20fine-tuning%20dataset/README.md)                                                                                   | Ollama            |
| [Creating dataset using vision model](./examples/v1/8.%20hugginface%20images%20and%20qwen2.5vl%20or%20gemma3/README.md)                                                  | Ollama, LM Studio |
