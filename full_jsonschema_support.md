# Full JSON Schema Support Implementation Plan

## Overview
This document outlines the plan to enhance datamatic with comprehensive JSON Schema validation using the `github.com/xeipuuv/gojsonschema` library. The implementation will add support for external schemas, config validation, and improved error handling while maintaining backward compatibility.

## Current State Analysis

### Existing Implementation
- **Duplicated Code**: Schema definitions exist in both `/jsonschema` and `/jsonl` packages
- **Limited Validation**: Custom validation logic with basic JSON Schema feature support
- **Embedded Only**: Schemas must be defined inline within YAML configs
- **No Config Validation**: The datamatic config itself is not validated against a schema
- **Basic Error Messages**: Limited context in validation errors

### Current Schema Features Supported
- Basic types: `string`, `number`, `integer`, `boolean`, `array`, `object`
- Validation constraints: `minimum`, `maximum`, `minLength`, `maxLength`, `pattern`
- Array item schemas and nested objects
- Required fields validation
- Additional properties control
- Enum values

## Implementation Phases

## Phase 1: Create New Schema Package with gojsonschema

### 1.1 Package Structure
Create `/schema` package as the unified schema handler:

```
/schema/
├── loader.go          # Schema loading from various sources
├── validator.go       # gojsonschema-based validation
├── converter.go       # YAML to JSON schema conversion
├── marshaler.go       # Schema to JSON text for LLM prompts
├── config_schema.json # Datamatic config validation schema
├── errors.go          # Enhanced error reporting
├── features.go        # Feature support documentation
└── types.go           # Common types and interfaces
```

### 1.2 Schema Loader Implementation
```go
// loader.go
package schema

import (
    "github.com/xeipuuv/gojsonschema"
)

type SchemaSource interface {
    Load() (*gojsonschema.Schema, error)
}

type Loader struct {
    cache map[string]*gojsonschema.Schema
}

func (l *Loader) LoadFromYAML(yamlSchema interface{}) (*gojsonschema.Schema, error)
func (l *Loader) LoadFromFile(path string) (*gojsonschema.Schema, error)
func (l *Loader) LoadFromString(jsonSchema string) (*gojsonschema.Schema, error)
func (l *Loader) LoadFromStep(step config.Step) (*gojsonschema.Schema, error)
```

### 1.3 Dependencies
Add to `go.mod`:
```
github.com/xeipuuv/gojsonschema v1.2.0
```

## Phase 2: Config Structure Extension

### 2.1 Extended Step Configuration
Update `/config/config.go`:

```go
type Step struct {
    Type               StepType
    Name               string                `yaml:"name"`
    Model              string                `yaml:"model"`
    Prompt             string                `yaml:"prompt"`
    Cmd                string                `yaml:"cmd"`
    SystemPrompt       string                `yaml:"systemPrompt"`
    MaxResults         interface{}           `yaml:"maxResults"`
    ModelConfig        ModelConfig           `yaml:"modelConfig"`
    OutputFilename     string                `yaml:"outputFilename"`
    
    // Schema options (priority: Raw > File > Embedded)
    JSONSchema         jsonschema.JSONSchema `yaml:"jsonSchema"`      // Existing embedded
    JSONSchemaFile     string                `yaml:"jsonSchemaFile"`  // New: external file
    JSONSchemaRaw      string                `yaml:"jsonSchemaRaw"`   // New: raw JSON string
    
    ImagePath          string                `yaml:"imagePath"`
    ResolvedMaxResults int
}
```

### 2.2 Schema Resolution Priority
1. `jsonSchemaRaw` - Raw JSON schema string (highest priority)
2. `jsonSchemaFile` - Path to external schema file
3. `jsonSchema` - Embedded YAML schema (backward compatibility)

## Phase 3: Config Validation

### 3.1 Datamatic Config Schema
Create `/schema/config_schema.json`:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+$"
    },
    "steps": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "minLength": 1,
            "maxLength": 255
          },
          "model": {
            "type": "string",
            "pattern": "^(ollama|lmstudio|openai|openrouter|gemini):.+"
          },
          "prompt": {"type": "string"},
          "cmd": {"type": "string"},
          "systemPrompt": {"type": "string"},
          "maxResults": {
            "oneOf": [
              {"type": "integer", "minimum": 1},
              {"type": "string"}
            ]
          },
          "jsonSchema": {"type": "object"},
          "jsonSchemaFile": {"type": "string"},
          "jsonSchemaRaw": {"type": "string"},
          "imagePath": {"type": "string"}
        },
        "required": ["name"],
        "oneOf": [
          {"required": ["prompt", "model"]},
          {"required": ["cmd"]}
        ]
      }
    },
    "retryConfig": {
      "type": "object",
      "properties": {
        "enabled": {"type": "boolean"},
        "maxAttempts": {"type": "integer", "minimum": 1},
        "initialDelay": {"type": "string"},
        "maxDelay": {"type": "string"},
        "backoffMultiplier": {"type": "number", "minimum": 1}
      }
    }
  },
  "required": ["version", "steps"]
}
```

### 3.2 Config Validation Implementation
```go
// validator.go
func ValidateConfig(configData []byte) error {
    schemaLoader := gojsonschema.NewReferenceLoader("file:///schema/config_schema.json")
    documentLoader := gojsonschema.NewBytesLoader(configData)
    
    result, err := gojsonschema.Validate(schemaLoader, documentLoader)
    if err != nil {
        return fmt.Errorf("config validation failed: %w", err)
    }
    
    if !result.Valid() {
        return formatValidationErrors(result.Errors())
    }
    
    return nil
}
```

## Phase 4: Enhanced Validation & Error Handling

### 4.1 Response Validator
```go
// validator.go
type Validator struct {
    loader *Loader
}

func (v *Validator) ValidateResponse(schema *gojsonschema.Schema, response string) error {
    responseLoader := gojsonschema.NewStringLoader(response)
    result, err := gojsonschema.Validate(schema, responseLoader)
    
    if err != nil {
        return fmt.Errorf("validation error: %w", err)
    }
    
    if !result.Valid() {
        return v.formatErrors(result.Errors())
    }
    
    return nil
}

func (v *Validator) formatErrors(errors []gojsonschema.ResultError) error {
    var details []string
    for _, err := range errors {
        details = append(details, fmt.Sprintf(
            "- Field: %s\n  Error: %s\n  Value: %v",
            err.Field(), err.Description(), err.Value(),
        ))
    }
    return fmt.Errorf("validation failed:\n%s", strings.Join(details, "\n"))
}
```

### 4.2 Feature Support Documentation
```go
// features.go
type FeatureSupport struct {
    Feature     string
    Supported   bool
    Providers   []string // LLM providers that support this
    Note        string
}

var SchemaFeatures = []FeatureSupport{
    {
        Feature:   "$ref",
        Supported: true,
        Providers: []string{"openai", "gemini"},
        Note:      "Internal references only, no external refs",
    },
    {
        Feature:   "allOf/anyOf/oneOf",
        Supported: true,
        Providers: []string{"openai"},
        Note:      "Limited support in other providers",
    },
    {
        Feature:   "patternProperties",
        Supported: false,
        Providers: []string{},
        Note:      "Not supported by most LLM providers",
    },
    // ... more features
}

func WarnUnsupportedFeatures(schema *gojsonschema.Schema, provider string) []string
```

## Phase 5: Integration

### 5.1 Update Prompt Step
Modify `/step/prompt_step.go`:

```go
func (p *PromptStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
    // ... existing code ...
    
    // Load schema using new system
    schemaLoader := schema.NewLoader()
    jsonSchema, err := schemaLoader.LoadFromStep(step)
    if err != nil {
        return fmt.Errorf("failed to load schema: %w", err)
    }
    
    if jsonSchema != nil {
        // Marshal for LLM prompt
        schemaText, err := schema.MarshalForPrompt(jsonSchema)
        if err != nil {
            return fmt.Errorf("failed to marshal schema: %w", err)
        }
        
        promptBuilder.AddValue("-", defaults.SystemStepName, "JSON_SCHEMA", schemaText)
        
        // Warn about unsupported features
        warnings := schema.WarnUnsupportedFeatures(jsonSchema, step.ModelConfig.ModelProvider)
        for _, warning := range warnings {
            log.Warn().Msg(warning)
        }
    }
    
    // ... rest of execution ...
    
    // Validate response
    if jsonSchema != nil && cfg.ValidateResponse {
        validator := schema.NewValidator()
        if err := validator.ValidateResponse(jsonSchema, response); err != nil {
            log.Error().Err(err).Msg("Response validation failed")
            // Handle based on config (strict vs lenient mode)
        }
    }
}
```

### 5.2 Update Config Validation
Modify `/config/validate.go`:

```go
func ValidateConfig(cfg *Config) error {
    // Convert config to JSON for validation
    configJSON, err := json.Marshal(cfg)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }
    
    // Validate against schema
    if err := schema.ValidateConfig(configJSON); err != nil {
        return fmt.Errorf("config validation failed: %w", err)
    }
    
    // ... existing validation logic ...
    
    return nil
}
```

## Phase 6: Migration & Cleanup

### 6.1 Migration Steps
1. **Create new `/schema` package** with all functionality
2. **Add compatibility layer** to support existing code
3. **Update imports** gradually from old packages to new
4. **Test thoroughly** with existing configs
5. **Deprecate old packages** with warnings
6. **Remove legacy code** after grace period

### 6.2 Backward Compatibility
- All existing configs continue to work without changes
- New features are opt-in via new config fields
- Graceful fallback for unsupported features
- Clear migration documentation

### 6.3 Testing Strategy
```go
// schema/validator_test.go
func TestBackwardCompatibility(t *testing.T)
func TestExternalSchemaLoading(t *testing.T)
func TestRawSchemaString(t *testing.T)
func TestSchemaPriority(t *testing.T)
func TestConfigValidation(t *testing.T)
func TestResponseValidation(t *testing.T)
func TestErrorFormatting(t *testing.T)
func TestFeatureWarnings(t *testing.T)
```

## Example Usage

### Example 1: External Schema File
```yaml
version: 1.0
steps:
  - name: generate_data
    model: openai:gpt-4
    prompt: "Generate test data"
    jsonSchemaFile: "./schemas/test_data.json"
```

### Example 2: Raw Schema String
```yaml
version: 1.0
steps:
  - name: generate_data
    model: ollama:llama3.2
    prompt: "Generate a person object"
    jsonSchemaRaw: |
      {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "age": {"type": "integer", "minimum": 0, "maximum": 150}
        },
        "required": ["name", "age"]
      }
```

### Example 3: Existing Embedded Schema (Backward Compatible)
```yaml
version: 1.0
steps:
  - name: generate_data
    model: lmstudio:hermes-3
    prompt: "Generate data"
    jsonSchema:
      type: object
      properties:
        title:
          type: string
      required:
        - title
```

## Benefits

1. **Full JSON Schema Support**: Leverage complete JSON Schema specification via gojsonschema
2. **Flexibility**: Support for inline, external file, and raw string schemas
3. **Config Validation**: Ensure configs are valid before execution
4. **Better Errors**: Detailed validation errors with field paths and descriptions
5. **Provider Awareness**: Warnings for features not supported by specific LLM providers
6. **Code Consolidation**: Single source of truth for all schema operations
7. **Backward Compatibility**: All existing configs continue to work
8. **Extensibility**: Easy to add new schema sources or validation rules

## Timeline Estimate

- **Phase 1**: 2-3 days - Create new schema package
- **Phase 2**: 1 day - Extend config structure
- **Phase 3**: 1-2 days - Implement config validation
- **Phase 4**: 2 days - Enhanced validation and error handling
- **Phase 5**: 2 days - Integration with existing code
- **Phase 6**: 1-2 days - Migration and cleanup

**Total**: 9-12 days for complete implementation

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing configs | Extensive backward compatibility testing |
| Performance impact | Schema caching and lazy loading |
| Complex migration | Gradual rollout with compatibility layer |
| LLM provider incompatibilities | Feature detection and warnings |
| Large refactoring scope | Incremental implementation phases |

## Success Criteria

- ✅ All existing configs work without modification
- ✅ External schema files can be loaded and used
- ✅ Config validation catches errors before execution
- ✅ Detailed error messages with field context
- ✅ No code duplication between packages
- ✅ Clear documentation of supported features
- ✅ Performance comparable to current implementation
- ✅ 100% test coverage for new schema package