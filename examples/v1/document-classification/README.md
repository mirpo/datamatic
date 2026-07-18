# Document classification (schema-guided reasoning)

Generate documents, classify each one, then QA the whole dataset's label distribution in a single fan-in step. The schema forces systematic analysis: identify type and summarize *before* extracting entities and keywords.

**Features:** `SGR` · `collect` · `forEach` · `jsonSchema`

## Steps

1. `generate_documents` — generate diverse documents across categories
2. `classify_documents` — `forEach` document → `{document_type (enum), brief_summary, key_entities_mentioned[], keywords[]}`
3. `label_distribution` — fan-in (`collect: true`): one jq expression counts labels across the whole dataset

## The SGR pattern

```python
class DocumentClassification(BaseModel):
  document_type: Literal["receipt", "blog_post", "article", "news", "invoice", ...]
  brief_summary: str
  key_entities_mentioned: List[str]
  keywords: List[str]
```

`document_type` and `brief_summary` come first, forcing the model to analyze the document before it extracts entities and keywords — which improves accuracy.

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`

## Run

```bash
datamatic --config ./config.yaml --verbose
```
