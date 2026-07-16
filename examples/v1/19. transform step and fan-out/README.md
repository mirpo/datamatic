# Transform Step and Fan-Out

Shows the built-in `jq` transform step turning **one structured LLM answer into many rows** (fan-out), then running a prompt per row:

1. `team_roster` — LLM generates 3 team rosters, each with an array of members (JSON schema enforced)
2. `members` — transform step explodes `members[]`: 3 roster rows become N member rows, no external tools
3. `bio` — one LLM call per member (`forEach: members`), referencing the current row as `{{.item.name}}` and `{{.item.role}}`

This 1-row → N-rows pattern was impossible without shell + external `jq` before; now it's one YAML step and jq programs are validated at config load.

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull llama3.2`

## Run dataset generation

`datamatic -config ./config.yaml -verbose`
