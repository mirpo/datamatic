# Document Classification with Schema-Guided Reasoning

Example demonstrates Schema-Guided Reasoning (SGR) for document classification using structured JSON schemas to force systematic analysis.
Inspired by https://abdullin.com/schema-guided-reasoning/examples

## Overview

This example shows how to use structured schemas to improve document classification accuracy by forcing the LLM to think through the task in predefined steps:

1. **Identify document type** - Forces selection from predefined categories
2. **Summarize content** - Creates mental model of document
3. **Extract key entities** - Identifies business-relevant entities from controlled vocabulary
4. **Generate keywords** - Produces searchable terms for retrieval

The schema acts as a reasoning framework that guides the LLM through systematic analysis rather than jumping directly to classification.

## The SGR Pattern

```python
class DocumentClassification(BaseModel):
  document_type: Literal["invoice", "contract", "receipt", ...]
  brief_summary: str
  key_entities_mentioned: List[Literal["payment", "risk", "regulator", ...]]
  keywords: List[str] = Field(..., description="Up to 10 keywords")
```

The first two fields (`document_type` and `brief_summary`) force the LLM to analyze the document before identifying entities and keywords. This structured thinking improves classification accuracy.

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull llama3.2`
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)
- [DuckDB CLI](https://duckdb.org/docs/installation/)

## Run classification

`datamatic --config ./config.yaml --verbose`

## Example Output

```json
{
  "document_type": "contract",
  "brief_summary": "Service agreement between vendor and customer for cloud infrastructure services",
  "key_entities_mentioned": ["vendor", "customer", "service", "legal", "financial"],
  "keywords": ["cloud", "infrastructure", "SLA", "agreement", "services", "pricing", "terms", "liability"]
}
```

## Use Cases

This pattern is particularly useful for:

- **RAG systems** - Improves document retrieval through better classification and keyword extraction
- **Document management** - Automates filing and categorization of business documents
- **Compliance** - Identifies documents requiring regulatory attention
- **Search enhancement** - Generates rich metadata for document discovery
