# Retry Configuration Example

This example demonstrates how to configure retry logic in datamatic for handling temporary failures when calling LLM APIs.

## What is Retry Configuration?

Retry configuration allows datamatic to automatically retry failed API calls with intelligent backoff strategies. This is especially useful when:

## Configuration Parameters

### `retryConfig` Section

| Parameter           | Default | Description                                       |
| ------------------- | ------- | ------------------------------------------------- |
| `enabled`           | `true`  | Enable/disable retry functionality                |
| `maxAttempts`       | `3`     | Maximum number of retry attempts (1 = no retries) |
| `initialDelay`      | `1s`    | Delay before first retry attempt                  |
| `maxDelay`          | `10s`   | Maximum delay between retries                     |
| `backoffMultiplier` | `2.0`   | Exponential backoff multiplier                    |

### Retry Strategy

### Error Types

Datamatic automatically retries these error types:
- **Rate limiting** (429 Too Many Requests)
- **Server errors** (500, 502, 503, 504)
- **Network timeouts**

It will **NOT** retry:
- **Authentication errors** (401, 403)
- **Bad requests** (400, 404, 422)
- **Model/schema errors**

## When to Adjust Settings

### Conservative Settings (Stable APIs)
```yaml
retryConfig:
  maxAttempts: 2
  initialDelay: 1s
  maxDelay: 5s
  backoffMultiplier: 2.0
```

### Aggressive Settings (Unstable Networks)
```yaml
retryConfig:
  maxAttempts: 5
  initialDelay: 3s
  maxDelay: 60s
  backoffMultiplier: 1.5
```

### Disable Retries (Testing/Development)
```yaml
retryConfig:
  enabled: false
```

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull llama3.2`

## Run Example

```bash
datamatic --config ./config.yaml --verbose
```
