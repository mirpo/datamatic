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
  document_type: Literal["receipt", "blog_post", "article", "news", "invoice", ...]
  brief_summary: str
  key_entities_mentioned: List[str]
  keywords: List[str] = Field(..., description="Up to 10 keywords")
```

The first two fields (`document_type` and `brief_summary`) force the LLM to analyze the document before identifying entities and keywords. This structured thinking improves classification accuracy.

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull llama3.2`

## Example Output

```json
{
  "document_type": "invoice",
  "brief_summary": "Invoice from a cloud provider billing a customer for monthly infrastructure usage",
  "key_entities_mentioned": ["vendor", "customer", "invoice number", "amount due", "billing period"],
  "keywords": ["invoice", "cloud", "billing", "infrastructure", "payment", "due date", "services"]
}
```

## Run dataset generation

`datamatic --config ./config.yaml --verbose`
