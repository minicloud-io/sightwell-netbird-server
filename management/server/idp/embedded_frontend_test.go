package idp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeThemeDir creates a minimal, valid Dex web directory for the tests.
func writeThemeDir(t *testing.T, theme string, templates ...string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "themes", theme), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "static"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "robots.txt"), []byte("User-agent: *\nDisallow: /\n"), 0o644))
	for _, name := range templates {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", name), []byte("{{ issuer }}"), 0o644))
	}
	return dir
}

func TestBuildFrontend_DefaultsWhenUnset(t *testing.T) {
	frontend, err := buildFrontend(nil)
	require.NoError(t, err)
	assert.Equal(t, "NetBird", frontend.Issuer)
	assert.Equal(t, "light", frontend.Theme)
	assert.Empty(t, frontend.Dir)
}

func TestBuildFrontend_DisplayNameWithoutDir(t *testing.T) {
	frontend, err := buildFrontend(&EmbeddedIdPFrontend{DisplayName: "Sightwell Connectivity"})
	require.NoError(t, err)
	assert.Equal(t, "Sightwell Connectivity", frontend.Issuer)
	assert.Equal(t, "light", frontend.Theme)
	assert.Empty(t, frontend.Dir, "no dir means the embedded pages stay")
}

func TestBuildFrontend_ValidDir(t *testing.T) {
	dir := writeThemeDir(t, "sightwell", requiredFrontendTemplates...)
	frontend, err := buildFrontend(&EmbeddedIdPFrontend{
		Dir:         dir,
		Theme:       "sightwell",
		DisplayName: "Sightwell Connectivity",
		LogoURL:     "https://example.com/logo.svg",
	})
	require.NoError(t, err)
	assert.Equal(t, dir, frontend.Dir)
	assert.Equal(t, "sightwell", frontend.Theme)
	assert.Equal(t, "Sightwell Connectivity", frontend.Issuer)
	assert.Equal(t, "https://example.com/logo.svg", frontend.LogoURL)
}

func TestBuildFrontend_MissingDir(t *testing.T) {
	_, err := buildFrontend(&EmbeddedIdPFrontend{Dir: filepath.Join(t.TempDir(), "nope")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "frontend dir")
}

func TestBuildFrontend_MissingTemplate(t *testing.T) {
	dir := writeThemeDir(t, "light", "header.html", "footer.html", "login.html")
	_, err := buildFrontend(&EmbeddedIdPFrontend{Dir: dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "templates/logout.html")
}

func TestBuildFrontend_MissingStaticFolder(t *testing.T) {
	dir := writeThemeDir(t, "light", requiredFrontendTemplates...)
	require.NoError(t, os.Remove(filepath.Join(dir, "static")))
	_, err := buildFrontend(&EmbeddedIdPFrontend{Dir: dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no static directory")
}

func TestBuildFrontend_RobotsTxtMissingOrDirectory(t *testing.T) {
	dir := writeThemeDir(t, "light", requiredFrontendTemplates...)
	require.NoError(t, os.Remove(filepath.Join(dir, "robots.txt")))
	_, err := buildFrontend(&EmbeddedIdPFrontend{Dir: dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "robots.txt")

	// A Kubernetes file mount that lands on a directory must be reported too.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "robots.txt"), 0o755))
	_, err = buildFrontend(&EmbeddedIdPFrontend{Dir: dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "directory mounted")
}

func TestBuildFrontend_MissingThemeFolder(t *testing.T) {
	dir := writeThemeDir(t, "light", requiredFrontendTemplates...)
	_, err := buildFrontend(&EmbeddedIdPFrontend{Dir: dir, Theme: "sightwell"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "themes/sightwell")
}

func TestToYAMLConfig_FrontendPassesThrough(t *testing.T) {
	dir := writeThemeDir(t, "sightwell", requiredFrontendTemplates...)
	cfg := &EmbeddedIdPConfig{
		Enabled:  true,
		Issuer:   "https://example.com/oauth2",
		Storage:  EmbeddedStorageConfig{Type: "sqlite3", Config: EmbeddedStorageTypeConfig{File: filepath.Join(t.TempDir(), "idp.db")}},
		Frontend: &EmbeddedIdPFrontend{Dir: dir, Theme: "sightwell", DisplayName: "Sightwell Connectivity"},
	}
	yamlCfg, err := cfg.ToYAMLConfig()
	require.NoError(t, err)
	assert.Equal(t, dir, yamlCfg.Frontend.Dir)
	assert.Equal(t, "sightwell", yamlCfg.Frontend.Theme)
	assert.Equal(t, "Sightwell Connectivity", yamlCfg.Frontend.Issuer)

	cfg.Frontend = nil
	yamlCfg, err = cfg.ToYAMLConfig()
	require.NoError(t, err)
	assert.Equal(t, "NetBird", yamlCfg.Frontend.Issuer)
	assert.Equal(t, "light", yamlCfg.Frontend.Theme)
	assert.Empty(t, yamlCfg.Frontend.Dir)
}
