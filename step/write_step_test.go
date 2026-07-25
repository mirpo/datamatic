package step

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/mirpo/datamatic/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeStepSetup(t *testing.T, srcLines, format, ext string) (config.Step, string) {
	t.Helper()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.jsonl")
	require.NoError(t, os.WriteFile(srcPath, []byte(srcLines), 0o644))

	out := filepath.Join(dir, "out."+ext)
	cfg := config.NewConfig()
	cfg.Steps = []config.Step{{Name: "data", Type: config.TransformStepType, OutputFilename: srcPath}}
	step := config.Step{
		Name: "report", Type: config.WriteStepType, From: "data",
		Write: out, Format: format, OutputFilename: out,
	}
	require.NoError(t, (&WriteStep{}).Run(context.Background(), cfg, step, dir))
	return step, out
}

func TestWriteStepRun_CSV(t *testing.T) {
	_, out := writeStepSetup(t,
		`{"name":"Acme","score":9}`+"\n"+`{"name":"Globex","score":4}`+"\n",
		config.WriteFormatCSV, "csv")

	f, err := os.Open(out)
	require.NoError(t, err)
	defer f.Close()
	recs, err := csv.NewReader(f).ReadAll()
	require.NoError(t, err)

	assert.Equal(t, []string{"name", "score"}, recs[0])
	assert.Equal(t, []string{"Acme", "9"}, recs[1])
	assert.Equal(t, []string{"Globex", "4"}, recs[2])
}

func TestWriteStepRun_JSONLPassthrough(t *testing.T) {
	_, out := writeStepSetup(t,
		`{"a":1}`+"\n"+`{"a":2}`+"\n",
		config.WriteFormatJSONL, "jsonl")

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, `{"a":1}`+"\n"+`{"a":2}`+"\n", string(data))
}

// TestWriteStepRun_CreatesParentDir covers a nested deliverable path: the
// output folder is created fresh, so a subdirectory in the write path won't
// exist yet and the step has to make it.
func TestWriteStepRun_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.jsonl")
	require.NoError(t, os.WriteFile(srcPath, []byte(`{"a":1}`+"\n"), 0o644))

	cfg := config.NewConfig()
	cfg.Steps = []config.Step{{Name: "data", Type: config.TransformStepType, OutputFilename: srcPath}}
	out := filepath.Join(dir, "reports", "2026", "out.csv")
	step := config.Step{
		Name: "report", Type: config.WriteStepType, From: "data",
		Write: out, Format: config.WriteFormatCSV, OutputFilename: out,
	}

	require.NoError(t, (&WriteStep{}).Run(context.Background(), cfg, step, dir))
	assert.FileExists(t, out)
}

func TestWriteStepRun_UnknownSourceFails(t *testing.T) {
	cfg := config.NewConfig()
	step := config.Step{Name: "report", Type: config.WriteStepType, From: "ghost", Write: "/tmp/x.csv", Format: config.WriteFormatCSV, OutputFilename: "/tmp/x.csv"}
	err := (&WriteStep{}).Run(context.Background(), cfg, step, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
}

func TestWriteStepRun_NonObjectRowFailsCSV(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.jsonl")
	require.NoError(t, os.WriteFile(srcPath, []byte(`"just a string"`+"\n"), 0o644))
	cfg := config.NewConfig()
	cfg.Steps = []config.Step{{Name: "data", Type: config.TransformStepType, OutputFilename: srcPath}}
	out := filepath.Join(dir, "out.csv")
	step := config.Step{Name: "report", Type: config.WriteStepType, From: "data", Write: out, Format: config.WriteFormatCSV, OutputFilename: out}

	err := (&WriteStep{}).Run(context.Background(), cfg, step, dir)
	require.Error(t, err, "csv needs object rows")
}

// perRowSetup runs a per-row write step over the given JSONL source rows and
// returns the directory the files landed in.
func perRowSetup(t *testing.T, srcLines, writeTmpl, content, format string) string {
	t.Helper()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.jsonl")
	require.NoError(t, os.WriteFile(srcPath, []byte(srcLines), 0o644))

	cfg := config.NewConfig()
	cfg.OutputFolder = dir
	cfg.Steps = []config.Step{{Name: "data", Type: config.TransformStepType, OutputFilename: srcPath}}
	step := config.Step{
		Name: "files", Type: config.WriteStepType, ForEach: "data",
		Write: writeTmpl, Content: content, Format: format, OutputFilename: dir,
	}

	require.NoError(t, (&WriteStep{}).Run(context.Background(), cfg, step, dir))
	return dir
}

func TestWriteStepRun_PerRowContent(t *testing.T) {
	dir := perRowSetup(t,
		`{"id":"alpha","body":"first draft"}`+"\n"+`{"id":"beta","body":"second draft"}`+"\n",
		"drafts/{{.item.id}}.md", "{{.item.body}}", "")

	for name, want := range map[string]string{"alpha.md": "first draft", "beta.md": "second draft"} {
		data, err := os.ReadFile(filepath.Join(dir, "drafts", name))
		require.NoError(t, err, "expected one file per row")
		assert.Equal(t, want, string(data), "body is the rendered content, not JSON")
	}
}

func TestWriteStepRun_PerRowSanitizesAndDedupes(t *testing.T) {
	dir := perRowSetup(t,
		`{"id":"a/b","body":"one"}`+"\n"+`{"id":"a/b","body":"two"}`+"\n"+`{"id":"","body":"three"}`+"\n",
		"{{.item.id}}.txt", "{{.item.body}}", "")

	assert.FileExists(t, filepath.Join(dir, "a_b.txt"), "separators in the name are replaced")
	assert.FileExists(t, filepath.Join(dir, "a_b-2.txt"), "a colliding name gets a suffix instead of clobbering")
	assert.FileExists(t, filepath.Join(dir, "3.txt"), "an empty name falls back to the row number")
}

func TestWriteStepRun_PerRowSerializesWithoutContent(t *testing.T) {
	dir := perRowSetup(t,
		`{"id":"alpha","score":7}`+"\n",
		"{{.item.id}}.json", "", config.WriteFormatJSON)

	data, err := os.ReadFile(filepath.Join(dir, "alpha.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"score": 7`, "without content the row itself is serialized")
}

func TestWriteStepRun_PerRowUnknownSourceFails(t *testing.T) {
	cfg := config.NewConfig()
	step := config.Step{
		Name: "files", Type: config.WriteStepType, ForEach: "ghost",
		Write: "{{.item.id}}.txt", Content: "x", OutputFilename: t.TempDir(),
	}
	err := (&WriteStep{}).Run(context.Background(), cfg, step, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
}
