# Examples

Each folder is a self-contained `config.yaml` + `README.md`. New to datamatic? Read them top to bottom.

| Example | Features shown | Backend |
| --- | --- | --- |
| [basics](./basics) | text generation, JSON schema | Ollama |
| [process-my-files](./process-my-files) | `read` your own local files (glob/dir/CSV/JSONL) → rows | Ollama |
| [csv-enrichment](./csv-enrichment) | `read` CSV → enrich with LLM → `write` CSV (the full office loop) | Ollama |
| [inbox-triage](./inbox-triage) | `read` folder of emails → SGR triage → draft replies → `write` CSV + Markdown | Ollama |
| [linked-steps](./linked-steps) | step chaining, native template values (`if`/`range`/`len`) | Ollama |
| [structured-extraction](./structured-extraction) | nested schema, both schema formats (YAML / JSON-string), native templates | Ollama |
| [dataset-pipeline](./dataset-pipeline) | transform fan-out, fan-in (`collect`, `$parent`), rating pipeline | Ollama |
| [sgr-reasoning](./sgr-reasoning) | schema-guided reasoning, `sourceFormat: json` | Ollama |
| [document-classification](./document-classification) | SGR classification, `collect` fan-in QA | Ollama |
| [document-qa](./document-qa) | RAG-style Q&A from a document, rating filter | Ollama |
| [external-data](./external-data) | HuggingFace download + transform + shell tools | Ollama |
| [env-and-workdir](./env-and-workdir) | env vars, `workDir`, `$PROVIDER`, DuckDB | Ollama |
| [vision](./vision) | image → structured output (`imagePath`) | Ollama, LM Studio |
| [cloud-providers](./cloud-providers) | provider selection, `concurrency`, `retryConfig` | OpenAI / OpenRouter / Gemini |

Most local examples default to `ollama:qwen3:1.7b` — pull it with `ollama pull qwen3:1.7b`.
