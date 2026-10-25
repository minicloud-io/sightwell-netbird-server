package dex

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func write(t *testing.T, p, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
}

// A projected volume at the theme root: real files under ..<timestamp>,
// a ..data symlink to it, and top-level symlinks into ..data.
func TestLoadThemeFS_ProjectedVolumeLayout(t *testing.T) {
	dir := t.TempDir()
	ts := filepath.Join(dir, "..2026_10_25_08_00_00.123")
	write(t, filepath.Join(ts, "templates", "login.html"), "login")
	write(t, filepath.Join(ts, "static", "main.css"), "css")
	write(t, filepath.Join(ts, "themes", "sightwell", "favicon.svg"), "svg")
	write(t, filepath.Join(ts, "robots.txt"), "robots")
	require.NoError(t, os.Symlink(ts, filepath.Join(dir, "..data")))
	for _, name := range []string{"templates", "static", "themes", "robots.txt"} {
		require.NoError(t, os.Symlink(filepath.Join("..data", name), filepath.Join(dir, name)))
	}

	fsys, err := loadThemeFS(dir)
	require.NoError(t, err)

	entries, err := fs.ReadDir(fsys, "templates")
	require.NoError(t, err)
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Equal(t, []string{"login.html"}, names, "no ..data in the listing Dex parses")

	data, err := fs.ReadFile(fsys, "robots.txt")
	require.NoError(t, err)
	assert.Equal(t, "robots", string(data))
	_, err = fs.Stat(fsys, "themes/sightwell/favicon.svg")
	require.NoError(t, err)
	_, err = fs.Stat(fsys, "..data")
	assert.Error(t, err)
}

// A ConfigMap mounted directly at templates/: the kubelet entries sit inside
// the directory Dex lists.
func TestLoadThemeFS_ConfigMapAtTemplates(t *testing.T) {
	dir := t.TempDir()
	tpl := filepath.Join(dir, "templates")
	ts := filepath.Join(tpl, "..2026_10_25_08_00_00.123")
	write(t, filepath.Join(ts, "header.html"), "header")
	write(t, filepath.Join(ts, "login.html"), "login")
	require.NoError(t, os.Symlink(ts, filepath.Join(tpl, "..data")))
	require.NoError(t, os.Symlink(filepath.Join("..data", "header.html"), filepath.Join(tpl, "header.html")))
	require.NoError(t, os.Symlink(filepath.Join("..data", "login.html"), filepath.Join(tpl, "login.html")))
	write(t, filepath.Join(dir, "static", "main.css"), "css")
	write(t, filepath.Join(dir, "robots.txt"), "robots")

	fsys, err := loadThemeFS(dir)
	require.NoError(t, err)
	entries, err := fs.ReadDir(fsys, "templates")
	require.NoError(t, err)
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Equal(t, []string{"header.html", "login.html"}, names)
	data, err := fs.ReadFile(fsys, "templates/login.html")
	require.NoError(t, err)
	assert.Equal(t, "login", string(data))
}

func TestLoadThemeFS_Errors(t *testing.T) {
	_, err := loadThemeFS(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
	empty := t.TempDir()
	_, err = loadThemeFS(empty)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no files")
}
