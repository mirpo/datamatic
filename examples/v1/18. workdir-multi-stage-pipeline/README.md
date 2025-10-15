# Multi-Stage Pipeline with workDir

This example demonstrates the `workDir` feature for organizing shell-step outputs across multiple processing stages.

## Overview

The `workDir` parameter (available for **shell steps only**) allows you to control where shell commands execute and where their outputs are written. This enables clean separation of:
- Raw downloads
- Intermediate transformations
- Final datasets

## Requirements

Install:

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull llama3.2`
- [hf](https://huggingface.co/docs/huggingface_hub/main/en/guides/cli)
- [DuckDB](https://duckdb.org/docs/installation/)

## Directory Structure

After running this pipeline:

```
dataset/
├── downloads/                   # Stage 1: Raw data
│   └── prompts.csv              # 204 rows (104 KB)
│
├── processed/                   # Stage 2 & 3: Transformations
│   └── all_prompts.jsonl        # 204 items
│
└── analyze_complexity.jsonl     # Stage 4: LLM output (outputFolder)
```

## Key Concepts

### workDir Behavior

| Path Type           | Example              | Resolves To                |
| ------------------- | -------------------- | -------------------------- |
| **Empty** (default) | `workDir: ""`        | `{outputFolder}`           |
| **Relative**        | `workDir: downloads` | `{outputFolder}/downloads` |
| **Absolute**        | `workDir: /tmp/data` | `/tmp/data` (unchanged)    |

### Shell vs Prompt Steps

| Feature              | Shell Steps                  | Prompt Steps                         |
| -------------------- | ---------------------------- | ------------------------------------ |
| `workDir` support    | ✅ Yes                        | ❌ No                                 |
| Output location      | `{workDir}/{outputFilename}` | `{outputFolder}/{outputFilename}`    |
| Cross-directory refs | `../other_dir/file.txt`      | Uses step chaining `{{.step.field}}` |

## Use Cases

### 1. Download Isolation
Keep raw downloads separate from processed data:
```yaml
- name: download
  run: wget https://data.source/file.zip
  workDir: downloads
```

### 2. Stage Separation
Organize multi-stage pipelines:
```yaml
- name: stage1
  workDir: raw_data
- name: stage2
  workDir: processed
- name: stage3
  workDir: final
```

### 3. Git Repository Processing
Clone and process repos without polluting the output folder:
```yaml
- name: clone_repo
  run: git clone https://github.com/user/repo.git
  workDir: repos/repo

- name: extract_data
  run: git log --oneline > commits.txt
  workDir: repos/repo
```

### 4. Temporary Scratch Space
Isolate intermediate files:
```yaml
- name: preprocessing
  run: |
    cat ../data/raw.jsonl | jq -c 'select(.valid)' > filtered.jsonl
    sort filtered.jsonl | uniq > result.jsonl
  workDir: temp
```

### 5. Absolute Paths for In-Place Processing
Process existing directories:
```yaml
- name: batch_convert
  run: for f in *.png; do convert "$f" "${f%.png}.jpg"; done
  workDir: /absolute/path/to/images
```

## Run dataset generation

`datamatic --config ./config.yaml --verbose`
