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
