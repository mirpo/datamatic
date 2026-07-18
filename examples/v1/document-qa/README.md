# Document Q&A dataset

Build a question-answer dataset from your own document — the kind of set used to fine-tune or evaluate a RAG system. Chunk a source document, generate grounded questions, answer them from the chunk, judge each pair, and keep only the well-grounded ones.

**Features:** `SGR` · `fan-out` · `$parent` · `LLM-judge` · `filter` · `shell`

## Steps

1. `download_document` — fetch the source document (any shell tool)
2. `chunk_document` — split into overlapping chunks (`chopdoc`)
3. `sample_chunks` — cap chunks so the demo runs quickly (raise/remove for a full run)
4. `questions` — **SGR**: summarize the chunk, then emit 3 questions tagged by cognitive level + evidence
5. `flatten` — **fan-out**: one row per question, carrying its chunk via `$parent`
6. `answer` — `forEach` question → an answer grounded strictly in the chunk
7. `qa_pair` — assemble `{question, answer, cognitive_level, textEvidence, chunk}`
8. `judge` — **LLM-as-judge**: rate 1-10 how well the answer is grounded
9. `high_quality` — keep only pairs with `rating >= 7`

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download) + `ollama pull qwen3:1.7b`
- `curl`, and [chopdoc](https://github.com/mirpo/chopdoc) for chunking:
  ```bash
  brew tap mirpo/homebrew-tools && brew install chopdoc
  # or: go install github.com/mirpo/chopdoc@latest
  ```

## Run

```bash
datamatic --config ./config.yaml --verbose
```
