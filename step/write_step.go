package step

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mirpo/datamatic/config"
	"github.com/mirpo/datamatic/fs"
	"github.com/mirpo/datamatic/jsonl"
	"github.com/mirpo/datamatic/promptbuilder"
	"github.com/rs/zerolog/log"
)

type WriteStep struct{}

// Run exports a source step's rows to a file. With `from:` it writes one
// aggregate file in the configured format (csv / json / md / jsonl); with
// `forEach:` it writes one file per row (see runPerRow). Either way the step is
// terminal: it reads the source's JSONL and produces no pipeline rows itself.
func (p *WriteStep) Run(ctx context.Context, cfg *config.Config, step config.Step, outputFolder string) error {
	if step.ForEach != "" {
		return p.runPerRow(ctx, cfg, step)
	}

	src := cfg.GetStepByName(step.From)
	if src == nil {
		return fmt.Errorf("'from' references unknown step '%s'", step.From)
	}

	// the deliverable may nest (e.g. write: reports/board.csv), and the output
	// folder starts out fresh, so its parent won't exist yet. Done before
	// reading the source so an unwritable destination fails immediately.
	if err := fs.EnsureFolder(filepath.Dir(step.Write)); err != nil {
		return fmt.Errorf("step '%s': %w", step.Name, err)
	}

	file, err := os.Open(src.OutputFilename)
	if err != nil {
		return fmt.Errorf("step '%s': failed to open source '%s': %w", step.Name, src.OutputFilename, err)
	}
	defer file.Close()

	rows, err := collectRows(ctx, *src, file)
	if err != nil {
		return fmt.Errorf("step '%s': %w", step.Name, err)
	}

	if err := serializeRows(step.Format, step.Write, rows); err != nil {
		return fmt.Errorf("step '%s': %w", step.Name, err)
	}

	log.Info().Msgf("write exported %d rows to %s", len(rows), step.Write)
	return nil
}

// runPerRow writes one file per source row: the `write:` path is a template
// rendered against the row (so the row's own fields name the file), and
// `content:` — when set — is the file body, written as raw text rather than
// serialized. This is the "folder of documents" mode, next to the aggregate
// "one table" mode.
func (p *WriteStep) runPerRow(ctx context.Context, cfg *config.Config, step config.Step) error {
	src := cfg.GetStepByName(step.ForEach)
	if src == nil {
		return fmt.Errorf("'forEach' references unknown step '%s'", step.ForEach)
	}

	file, err := os.Open(src.OutputFilename)
	if err != nil {
		return fmt.Errorf("step '%s': failed to open source '%s': %w", step.Name, src.OutputFilename, err)
	}
	defer file.Close()

	rows, err := collectRows(ctx, *src, file)
	if err != nil {
		return fmt.Errorf("step '%s': %w", step.Name, err)
	}

	// step.OutputFilename carries the output folder for per-row writes: the
	// path itself can only be resolved once rendered
	written := map[string]int{}

	for i, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}

		pb, err := rowBuilder(step, src.Name, row)
		if err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}

		// the path is rendered from a copy of the row whose values are already
		// filename-safe, so a slash written in the template still nests while a
		// slash inside the data cannot redirect the file
		pathBuilder, err := rowBuilder(step, src.Name, sanitizeForPath(row))
		if err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}

		path, err := renderRowPath(pathBuilder, step, i)
		if err != nil {
			return fmt.Errorf("step '%s' row %d: %w", step.Name, i, err)
		}
		path = uniquePath(path, written)

		if err := fs.EnsureFolder(filepath.Dir(path)); err != nil {
			return fmt.Errorf("step '%s': %w", step.Name, err)
		}

		if step.Content != "" {
			body, err := pb.RenderString(step.Content)
			if err != nil {
				return fmt.Errorf("step '%s' row %d: failed to render content: %w", step.Name, i, err)
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				return fmt.Errorf("step '%s': failed to write '%s': %w", step.Name, path, err)
			}
		} else if err := serializeRows(step.Format, path, rows[i:i+1]); err != nil {
			return fmt.Errorf("step '%s' row %d: %w", step.Name, i, err)
		}
	}

	log.Info().Msgf("write exported %d rows to %d file(s) under %s", len(rows), len(written), step.OutputFilename)
	return nil
}

// rowBuilder binds one source row to the step's templates, exposing it both as
// {{.item}} and under the source step's own name.
func rowBuilder(step config.Step, sourceName string, row interface{}) (*promptbuilder.PromptBuilder, error) {
	pb, err := promptbuilder.NewPromptBuilder(step.Write, step.ForEach, step.Content)
	if err != nil {
		return nil, err
	}
	pb.AddValue("-", sourceName, "", row)
	return pb, nil
}

// sanitizeForPath returns a copy of a row whose every string is safe as a single
// path segment, so values interpolated into a write path cannot introduce
// directories. Non-strings are untouched.
func sanitizeForPath(v interface{}) interface{} {
	switch t := v.(type) {
	case string:
		return fs.SanitizeFilename(t)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			out[k] = sanitizeForPath(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, val := range t {
			out[i] = sanitizeForPath(val)
		}
		return out
	default:
		return v
	}
}

// renderRowPath renders the write template for one row into a path under the
// output folder. A template may nest (e.g. "{{.item.kind}}/{{.item.id}}.md");
// a file name that renders to nothing falls back to the row number.
func renderRowPath(pb *promptbuilder.PromptBuilder, step config.Step, i int) (string, error) {
	rendered, err := pb.RenderString(step.Write)
	if err != nil {
		return "", fmt.Errorf("failed to render write path: %w", err)
	}

	dir, name := filepath.Split(rendered)
	ext := filepath.Ext(name)
	stem := fs.SanitizeFilename(strings.TrimSuffix(name, ext))
	if stem == "" {
		stem = strconv.Itoa(i + 1)
	}
	name = stem + ext

	if filepath.IsAbs(rendered) {
		return filepath.Join(dir, name), nil
	}
	return filepath.Join(step.OutputFilename, dir, name), nil
}

// uniquePath keeps two rows that render the same name from clobbering each
// other within a run, by numbering the later ones. Leftovers from previous runs
// are still overwritten — each run produces a fresh set of files.
func uniquePath(path string, written map[string]int) string {
	written[path]++
	if n := written[path]; n > 1 {
		ext := filepath.Ext(path)
		numbered := fmt.Sprintf("%s-%d%s", strings.TrimSuffix(path, ext), n, ext)
		log.Warn().Msgf("two rows render the same file name, writing '%s' instead of overwriting '%s'", numbered, path)
		return numbered
	}
	return path
}

// serializeRows writes rows to one file in the given format. Shared by the
// aggregate mode (every row) and the per-row mode (a single row per file).
func serializeRows(format, path string, rows []interface{}) error {
	switch format {
	case config.WriteFormatJSON:
		return fs.WriteJSONArray(path, rows)
	case config.WriteFormatJSONL:
		return writeJSONL(path, rows)
	case config.WriteFormatCSV, config.WriteFormatMarkdown:
		objs, err := asObjects(rows)
		if err != nil {
			return err
		}
		// WriteCSV and WriteMarkdownTable share a signature — pick by format
		writeTable := fs.WriteCSV
		if format == config.WriteFormatMarkdown {
			writeTable = fs.WriteMarkdownTable
		}
		return writeTable(path, objs)
	default:
		return fmt.Errorf("unknown write format '%s'", format)
	}
}

// asObjects asserts every row is a JSON object (required for csv/md columns).
func asObjects(rows []interface{}) ([]map[string]interface{}, error) {
	objs := make([]map[string]interface{}, 0, len(rows))
	for i, r := range rows {
		obj, ok := r.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("row %d is not a JSON object; csv/md output needs object rows (use json/jsonl instead)", i)
		}
		objs = append(objs, obj)
	}
	return objs, nil
}

// writeJSONL reuses the shared JSONL writer, so the deliverable's line format
// stays defined in one place (the writer truncates: a fresh file each run).
func writeJSONL(path string, rows []interface{}) error {
	writer, err := jsonl.NewWriter(path)
	if err != nil {
		return fmt.Errorf("failed to create '%s': %w", path, err)
	}
	defer writer.Close()

	for _, row := range rows {
		if err := writer.WriteJSON(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}
	return nil
}
