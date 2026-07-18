package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobFiles_SortsAndExcludesDirs(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"c.md", "a.md", "b.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644))
	}
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))

	got, err := GlobFiles(filepath.Join(dir, "*.md"))

	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(dir, "a.md"),
		filepath.Join(dir, "b.md"),
		filepath.Join(dir, "c.md"),
	}, got, "sorted ascending, directories excluded")
}

func TestGlobFiles_Directory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("y"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))

	got, err := GlobFiles(dir) // a directory, not a glob

	require.NoError(t, err)
	assert.Equal(t, []string{
		filepath.Join(dir, "a.txt"),
		filepath.Join(dir, "b.txt"),
	}, got)
}

func TestGlobFiles_EmptyMatchErrors(t *testing.T) {
	dir := t.TempDir()
	_, err := GlobFiles(filepath.Join(dir, "*.md"))
	require.Error(t, err)
}

func TestReadTextFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "doc.md")
	require.NoError(t, os.WriteFile(p, []byte("hello world"), 0o644))

	name, content, err := ReadTextFile(p)

	require.NoError(t, err)
	assert.Equal(t, "doc.md", name)
	assert.Equal(t, "hello world", content)
}

func TestReadCSV(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "leads.csv")
	require.NoError(t, os.WriteFile(p, []byte("company,website\nAcme,acme.com\nGlobex,globex.io\n"), 0o644))

	rows, err := ReadCSV(p)

	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "Acme", rows[0]["company"])
	assert.Equal(t, "acme.com", rows[0]["website"])
	assert.Equal(t, "Globex", rows[1]["company"])
	assert.Equal(t, "globex.io", rows[1]["website"])
}
