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

## ⚠️ Important notes

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
{"id":"469cfc97-04de-47f6-a2b7-f8a31a3f893e","format":"text","prompt":"Generate a catchy and one unique news title. Come up with a wildly different and surprising news headline. Return only one news title per request, without any extra thinking.","response":"GIANT PURPLE PINEAPPLE DISAPPEARS FROM FRENCH QUARTER, LEAVING TOURISTS BAFFLED AND DELICIOUS-SMELLING CLOUD IN ITS WAKE.","values":[]}
```

**JSON line**:

```json
{"id":"edfdef51-6025-442b-8df2-91159edde0c7","format":"json","prompt":"Provide up-to-date information about a randomly selected country, including its name, population, land area, UN membership status, capital city, GDP per capita, official languages, and year of independence. Return the data in a structured JSON format according to the schema below.","response":{"capitalCity":"Bishkek","gdpPerCapita":1643.8,"independenceYear":1991,"isUNMember":true,"languages":["Kyrgyz","Russian"],"name":"Kyrgyzstan","population":6786000,"totalCountryArea":199900},"values":[]}
```

With values from linked steps:

```json
{"id":"1b9872d3-4eab-486c-924f-0ff74e18d3d6","format":"text","prompt":"Write nice tourist brochure about country Kyrgyzstan, which capital is Bishkek, area 199900, independenceYear: 1991 and official languages are Kyrgyz, Russian.","response":"...**A Brief History**\n\nKyrgyzstan declared its independence on August 31, 1991...","values":[{"id":"edfdef51-6025-442b-8df2-91159edde0c7","complexKey":"about_country.independenceYear","content":"1991"},{"id":"edfdef51-6025-442b-8df2-91159edde0c7","complexKey":"about_country.languages","content":"Kyrgyz, Russian"},{"id":"edfdef51-6025-442b-8df2-91159edde0c7","complexKey":"about_country.name","content":"Kyrgyzstan"},{"id":"edfdef51-6025-442b-8df2-91159edde0c7","complexKey":"about_country.capitalCity","content":"Bishkek"},{"id":"edfdef51-6025-442b-8df2-91159edde0c7","complexKey":"about_country.totalCountryArea","content":"199900"}]}
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

| Name                                                                                                                          | Provider (-s)     |
| ----------------------------------------------------------------------------------------------------------------------------- | ----------------- |
| [Simple text generation, not linked steps](./examples/v1/1.%20simple%20text%20generation,%20not%20linked%20steps/config.yaml) | Ollama, LM Studio |
| [Simple JSON generation, not linked steps](./examples/v1/2.%20simple%20json%20generation,%20not%20linked%20steps/config.yaml) | Ollama, LM Studio |
| [Complex JSON generation, linked steps](./examples/v1/3.%20complex%20json,%20linked%20steps/config.yaml)                      | Ollama            |
