package instance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetUpdateSources_GitHubForBoth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mgmt":
			_, _ = w.Write([]byte(`{"tag_name":"v0.81.4-sightwell.9"}`))
		case "/dash":
			_, _ = w.Write([]byte(`{"tag_name":"v2.95.0-sightwell.7"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	saved := currentUpdateSources()
	t.Cleanup(func() {
		updateSourcesMu.Lock()
		updateSources = saved
		updateSourcesMu.Unlock()
	})

	SetUpdateSources(srv.URL+"/mgmt", srv.URL+"/dash")
	m := &DefaultManager{httpClient: srv.Client()}
	info, err := m.fetchVersionInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "0.81.4-sightwell.9", info.ManagementVersion, "fetchGitHubRelease strips the leading v, as for upstream")
	assert.Equal(t, "2.95.0-sightwell.7", info.DashboardVersion)
}

func TestSetUpdateSources_EmptyKeepsDefaults(t *testing.T) {
	saved := currentUpdateSources()
	t.Cleanup(func() {
		updateSourcesMu.Lock()
		updateSources = saved
		updateSourcesMu.Unlock()
	})
	SetUpdateSources("", "")
	s := currentUpdateSources()
	assert.Equal(t, managementVersionURL, s.ManagementURL)
	assert.True(t, s.ManagementPlainText)
	assert.Equal(t, dashboardReleasesURL, s.DashboardURL)
}

// go-version orders fork builds of one upstream release by the build number
// and any newer upstream release above them, which is what a fork needs.
func TestIsNewerVersion_ForkBuilds(t *testing.T) {
	assert.True(t, isNewerVersion("v0.81.4-sightwell.4", "v0.81.4-sightwell.5"))
	assert.False(t, isNewerVersion("v0.81.4-sightwell.5", "v0.81.4-sightwell.4"))
	assert.False(t, isNewerVersion("v0.81.4-sightwell.5", "v0.81.4-sightwell.5"))
	assert.True(t, isNewerVersion("v0.81.4-sightwell.5", "v0.81.5-sightwell.1"))
	assert.True(t, isNewerVersion("v0.81.4-sightwell.9", "v0.81.4-sightwell.10"))
}
