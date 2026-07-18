package step

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/internal/llmtest"
	"github.com/mirpo/datamatic/jsonschema"
	"github.com/mirpo/datamatic/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const titleSchema = `{
	"type": "object",
	"properties": {"title": {"type": "string"}},
	"required": ["title"],
	"additionalProperties": false
}`

func testSchema(t *testing.T, raw string) jsonschema.Schema {
	t.Helper()
	s, err := jsonschema.LoadSchema(raw)
	require.NoError(t, err)
	return *s
}

func promptStepConfig(t *testing.T, srvURL string) (*config.Config, config.Step, string) {
	t.Helper()
	dir := t.TempDir()

	cfg := config.NewConfig()
	cfg.OutputFolder = dir

	step := config.Step{
		Name:           "gen",
		Type:           config.PromptStepType,
		Prompt:         "generate something",
		ResolvedCount:  3,
		OutputFilename: filepath.Join(dir, "gen.jsonl"),
		ModelConfig: config.ModelConfig{
			ModelProvider: llm.ProviderOllama,
			ModelName:     "test-model",
			BaseURL:       srvURL,
		},
	}
	return cfg, step, dir
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	lines, err := fs.CountLinesInFile(path)
	require.NoError(t, err)
	return lines
}

func TestPromptStepRun_WritesResolvedCountLines(t *testing.T) {
	srv := llmtest.NewServer(t, "hello world")
	cfg, step, dir := promptStepConfig(t, srv.URL)

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.NoError(t, err)
	assert.Equal(t, 3, countLines(t, step.OutputFilename))
	assert.Equal(t, 3, srv.CallCount())
}

func TestPromptStepRun_UnknownRefStepReturnsError(t *testing.T) {
	srv := llmtest.NewServer(t, "never reached")
	cfg, step, dir := promptStepConfig(t, srv.URL)
	step.Prompt = "use {{.ghost.field}}" // no step named "ghost" in cfg.Steps

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
	assert.Equal(t, 0, srv.CallCount())
}

func TestPromptStepRun_RefValuesReadFailureFailsStep(t *testing.T) {
	srv := llmtest.NewServer(t, "never reached")
	cfg, step, dir := promptStepConfig(t, srv.URL)

	// "prev" exists in config but its output file does not exist on disk
	cfg.Steps = []config.Step{
		{Name: "prev", Type: config.PromptStepType, OutputFilename: filepath.Join(dir, "missing.jsonl")},
	}
	step.Prompt = "use {{.prev}}"

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.Error(t, err, "must fail, not send '<no value>' prompts to the LLM")
	assert.Contains(t, err.Error(), "prev")
	assert.Equal(t, 0, srv.CallCount(), "no LLM call with a broken prompt")
}

func TestPromptStepRun_MissingImageFailsStep(t *testing.T) {
	srv := llmtest.NewServer(t, "never reached")
	cfg, step, dir := promptStepConfig(t, srv.URL)
	step.Image = filepath.Join(dir, "no-such.jpg") // file does not exist

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.Error(t, err)
	assert.Equal(t, 0, srv.CallCount(), "no LLM call when the image can't be read")
}

func TestPromptStepRun_AttachesImageFromRowPath(t *testing.T) {
	// image: renders a per-row path template and attaches the file as base64
	srv := llmtest.NewServer(t, "described")
	cfg, step, dir := promptStepConfig(t, srv.URL)

	imgPath := filepath.Join(dir, "pic.jpg")
	require.NoError(t, os.WriteFile(imgPath, []byte("fake-image-bytes"), 0o644))

	srcPath := filepath.Join(dir, "src.jsonl")
	require.NoError(t, os.WriteFile(srcPath, []byte(`{"path":"`+imgPath+`"}`+"\n"), 0o644))
	cfg.Steps = []config.Step{{Name: "imgs", Type: config.ReadStepType, OutputFilename: srcPath}}

	step.ForEach = "imgs"
	step.ResolvedCount = 1
	step.Prompt = "Describe the image."
	step.Image = "{{.item.path}}"

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)
	require.NoError(t, err)

	// the request carried the base64 of the file
	messages := srv.Requests()[0]["messages"].([]interface{})
	last := messages[len(messages)-1].(map[string]interface{})
	assert.NotNil(t, last["content"], "vision content attached")
	assert.Equal(t, 1, srv.CallCount())
}

func TestPromptStepRun_FailsAfterRepeatedInvalidResponses(t *testing.T) {
	// server always returns JSON that violates the schema (missing "title")
	srv := llmtest.NewServer(t, `{"wrong": true}`)
	cfg, step, dir := promptStepConfig(t, srv.URL)
	step.JSONSchema = testSchema(t, titleSchema)

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.Error(t, err, "must not loop forever on a persistently invalid model")
	assert.Contains(t, err.Error(), "invalid response")
	assert.Equal(t, cfg.RetryConfig.MaxAttempts, srv.CallCount(),
		"one LLM call per allowed attempt, then stop")
}

func TestPromptStepRun_RecoversAfterTransientInvalidResponse(t *testing.T) {
	// first response invalid, then valid ones — step must succeed
	srv := llmtest.NewServer(t, `{"wrong": true}`, `{"title": "ok"}`, `{"title": "ok2"}`, `{"title": "ok3"}`)
	cfg, step, dir := promptStepConfig(t, srv.URL)
	step.JSONSchema = testSchema(t, titleSchema)

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.NoError(t, err)
	assert.Equal(t, 3, countLines(t, step.OutputFilename))
	assert.Equal(t, 4, srv.CallCount()) // 1 failed + 3 good
}

func TestPromptStepRun_ConcurrencyLimitsInFlight(t *testing.T) {
	srv := llmtest.NewServer(t, "ok")
	srv.Delay = 40 * time.Millisecond
	cfg, step, dir := promptStepConfig(t, srv.URL)
	step.ResolvedCount = 8
	step.Concurrency = 3

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)

	require.NoError(t, err)
	assert.Equal(t, 8, countLines(t, step.OutputFilename))
	assert.Equal(t, 8, srv.CallCount())
	assert.LessOrEqual(t, srv.MaxConcurrent(), 3, "never more than concurrency requests at once")
	assert.Greater(t, srv.MaxConcurrent(), 1, "actually runs in parallel")
}

func TestPromptStepRun_OrderedOutputRegardlessOfCompletion(t *testing.T) {
	// forEach source with recognizable rows; echo mode makes each response equal
	// to its prompt, and the delay shuffles completion order.
	srv := llmtest.NewServer(t)
	srv.EchoPrompt = true
	srv.Delay = 20 * time.Millisecond
	cfg, step, dir := promptStepConfig(t, srv.URL)

	srcPath := filepath.Join(dir, "src.jsonl")
	var lines string
	for i := range 6 {
		lines += `{"id":"r` + string(rune('0'+i)) + `","format":"text","prompt":"p","response":"item-` + string(rune('0'+i)) + `"}` + "\n"
	}
	require.NoError(t, os.WriteFile(srcPath, []byte(lines), 0o644))

	cfg.Steps = []config.Step{
		{Name: "src", Type: config.PromptStepType, OutputFilename: srcPath},
	}
	step.ForEach = "src"
	step.ResolvedCount = 6
	step.Concurrency = 4
	step.Prompt = "{{.src}}"

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)
	require.NoError(t, err)

	for i, line := range readOutput(t, step.OutputFilename) {
		assert.Contains(t, line, "item-"+string(rune('0'+i)),
			"output row %d must correspond to source row %d despite parallel completion", i, i)
	}
}

func TestPromptStepRun_ConcurrentRowFailureCancelsAndPreservesPrefix(t *testing.T) {
	// row 3's prompt reads a field that does not exist -> that row errors;
	// the step must fail and stop, not hang.
	srv := llmtest.NewServer(t, `{"title":"ok"}`)
	cfg, step, dir := promptStepConfig(t, srv.URL)

	srcPath := filepath.Join(dir, "src.jsonl")
	var lines string
	for i := range 6 {
		// row 3 has no "title" field, others do
		if i == 3 {
			lines += `{"id":"r3","format":"json","prompt":"p","response":{"other":1}}` + "\n"
		} else {
			lines += `{"id":"r` + string(rune('0'+i)) + `","format":"json","prompt":"p","response":{"title":"t` + string(rune('0'+i)) + `"}}` + "\n"
		}
	}
	require.NoError(t, os.WriteFile(srcPath, []byte(lines), 0o644))

	cfg.Steps = []config.Step{
		{Name: "src", Type: config.PromptStepType, OutputFilename: srcPath, JSONSchema: testSchema(t, titleSchema)},
	}
	step.ForEach = "src"
	step.ResolvedCount = 6
	step.Concurrency = 2
	step.Prompt = "use {{.src.title}}"

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)
	require.Error(t, err, "a failing row must fail the whole step")
	assert.Contains(t, err.Error(), "title")
}

func TestPromptStepRun_NativeTemplateRendering(t *testing.T) {
	srv := llmtest.NewServer(t, "summary written")
	cfg, step, dir := promptStepConfig(t, srv.URL)

	prevPath := filepath.Join(dir, "prev.jsonl")
	prevLine := `{"id":"r1","format":"json","prompt":"p","response":{"member":false,"pop":6184000,"langs":["Kyrgyz","Russian"],"jobs":[{"name":"Acme","months":26},{"name":"Globex","months":14}]}}` + "\n"
	require.NoError(t, os.WriteFile(prevPath, []byte(prevLine), 0o644))

	cfg.Steps = []config.Step{
		{Name: "prev", Type: config.PromptStepType, OutputFilename: prevPath, JSONSchema: testSchema(t, `{
			"type":"object",
			"properties":{"member":{"type":"boolean"},"pop":{"type":"integer"},"langs":{"type":"array"},"jobs":{"type":"array"}},
			"required":["member","pop","langs","jobs"],
			"additionalProperties":false
		}`)},
	}
	step.ForEach = "prev"
	step.ResolvedCount = 1
	step.Prompt = `{{if .prev.member}}member{{else}}not-member{{end}};pop={{.prev.pop}};n={{len .prev.jobs}};{{range .prev.jobs}}{{.name}}({{.months}}mo) {{end}};langs={{.prev.langs}}`

	err := (&PromptStep{}).Run(context.Background(), cfg, step, dir)
	require.NoError(t, err)

	messages := srv.Requests()[0]["messages"].([]interface{})
	content := messages[len(messages)-1].(map[string]interface{})["content"].(string)
	assert.Equal(t, "not-member;pop=6184000;n=2;Acme(26mo) Globex(14mo) ;langs=Kyrgyz, Russian", content)

	// lineage keeps native types
	data, err := os.ReadFile(step.OutputFilename)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"value":6184000`, "numbers stay numeric in values")
	assert.Contains(t, string(data), `"value":["Kyrgyz","Russian"]`, "arrays stay arrays in values")
	assert.Contains(t, string(data), `"value":false`, "booleans stay boolean in values")
}
