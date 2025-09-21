# Recipe Generation with Nested Fields Example

This example demonstrates **nested field access** in datamatic step references, showcasing how to use complex JSON schemas with deep object structures and reference nested fields in subsequent steps.

## What This Example Shows

This example creates a **two-step recipe generation pipeline**:

1. **Step 1 (`recipe_data`)**: Generate structured recipe data with nested objects
2. **Step 2 (`cooking_guide`)**: Create cooking instructions using nested field references

### Nested Field Access Demonstrated

The second step uses advanced nested field references like:
- `{{.recipe_data.ingredients.proteins}}` - Access protein list
- `{{.recipe_data.ingredients.vegetables}}` - Access vegetable list
- `{{.recipe_data.nutrition.calories}}` - Access calorie information
- `{{.recipe_data.nutrition.prep_time_minutes}}` - Access prep time
- `{{.recipe_data.details.cuisine_type}}` - Access cuisine type
- `{{.recipe_data.details.difficulty_level}}` - Access difficulty level

This showcases datamatic's ability to traverse deeply nested JSON structures using dot notation.

## JSON Schema Structure

The recipe data uses a complex nested schema:

```yaml
ingredients:
  proteins: [array]     # {{.recipe_data.ingredients.proteins}}
  vegetables: [array]   # {{.recipe_data.ingredients.vegetables}}
  grains: [array]       # {{.recipe_data.ingredients.grains}}
  spices: [array]       # {{.recipe_data.ingredients.spices}}
nutrition:
  calories: number      # {{.recipe_data.nutrition.calories}}
  protein_grams: number # {{.recipe_data.nutrition.protein_grams}}
  prep_time_minutes: number # {{.recipe_data.nutrition.prep_time_minutes}}
details:
  cuisine_type: string  # {{.recipe_data.details.cuisine_type}}
  difficulty_level: enum # {{.recipe_data.details.difficulty_level}}
  serving_size: number  # {{.recipe_data.details.serving_size}}
```

## Requirements

- `datamatic`
- [Ollama](https://ollama.com/download)
- Install model: `ollama pull llama3.2`

## Run Example

```bash
datamatic --config ./config.yaml --verbose
```
