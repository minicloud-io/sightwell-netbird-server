package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The server section is copied into the management config wholesale
// (ApplySimplifiedDefaults), so the theme block only has to parse and map.
func TestAuthThemeConfig_ParsesAndMaps(t *testing.T) {
	var cfg CombinedConfig
	require.NoError(t, yaml.Unmarshal([]byte(`
server:
  dataDir: /var/lib/netbird
  auth:
    issuer: https://example.com/oauth2
    theme:
      dir: idp-theme
      name: sightwell
      displayName: Sightwell Connectivity
`), &cfg))
	require.NotNil(t, cfg.Server.Auth.Theme)

	mgmt := ManagementConfig{DataDir: cfg.Server.DataDir, Auth: cfg.Server.Auth}
	idpCfg, err := cfg.buildEmbeddedIdPConfig(mgmt)
	require.NoError(t, err)
	require.NotNil(t, idpCfg.Frontend)
	assert.Equal(t, filepath.Join("/var/lib/netbird", "idp-theme"), idpCfg.Frontend.Dir, "a relative dir resolves against dataDir")
	assert.Equal(t, "sightwell", idpCfg.Frontend.Theme)
	assert.Equal(t, "Sightwell Connectivity", idpCfg.Frontend.DisplayName)
	assert.Empty(t, idpCfg.Frontend.LogoURL)
}

func TestAuthThemeConfig_AbsoluteDirKept(t *testing.T) {
	var cfg CombinedConfig
	mgmt := ManagementConfig{
		DataDir: "/var/lib/netbird",
		Auth:    AuthConfig{Issuer: "https://example.com/oauth2", Theme: &AuthThemeConfig{Dir: "/etc/netbird/idp-theme"}},
	}
	idpCfg, err := cfg.buildEmbeddedIdPConfig(mgmt)
	require.NoError(t, err)
	require.NotNil(t, idpCfg.Frontend)
	assert.Equal(t, "/etc/netbird/idp-theme", idpCfg.Frontend.Dir)
}

func TestAuthThemeConfig_Absent(t *testing.T) {
	var cfg CombinedConfig
	mgmt := ManagementConfig{DataDir: "/var/lib/netbird", Auth: AuthConfig{Issuer: "https://example.com/oauth2"}}
	idpCfg, err := cfg.buildEmbeddedIdPConfig(mgmt)
	require.NoError(t, err)
	assert.Nil(t, idpCfg.Frontend, "no theme block keeps the embedded pages")
}
