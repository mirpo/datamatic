# SQL Query Generation with Schema-Guided Reasoning

Example demonstrates structured chain-of-thought reasoning for SQL query generation using the gretelai/synthetic_text_to_sql dataset.
Inspired by https://abdullin.com/schema-guided-reasoning/examples

## Overview

This example shows how to generate SQL queries with explicit reasoning steps. The model first completes a "solution checklist" that analyzes:
- Which tables and columns are needed
- Type of table relationships (direct/indirect)
- Whether recursive queries or subqueries are required
- Query direction and filtering requirements

Only after completing this analysis does the model generate the final SQL query.

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull llama3.2`
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)
- [DuckDB CLI](https://duckdb.org/docs/installation/)

## Run dataset generation

`datamatic --config ./config.yaml --verbose`

## Example Output

For a query like "What is the total volume of timber sold by each salesperson?", the model will generate:

```json
{
  "solution_checklist": {
    "tables_to_query": ["salesperson", "timber_sales"],
    "columns_to_query": ["salesperson_id", "name", "volume"],
    "dependency_kind": "direct",
    "is_subject_system_from_or_to": "N/A",
    "does_this_require_recursive_query": false,
    "does_this_require_subquery": false,
    "is_this_forward_or_backward_pass": "forward",
    "should_we_filter_out_subject_system_from_results_to_avoid_overcounting": false
  },
  "sql_query": "SELECT salesperson_id, name, SUM(volume) as total_volume FROM timber_sales JOIN salesperson ON timber_sales.salesperson_id = salesperson.salesperson_id GROUP BY salesperson_id, name ORDER BY total_volume DESC;"
}
```
