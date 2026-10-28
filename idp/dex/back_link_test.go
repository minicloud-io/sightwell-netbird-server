package dex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/netbirdio/dex/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// passwordPage drives the sign-in flow to the password page and returns the
// rendered HTML, following the redirects a browser would.
func passwordPage(t *testing.T, p *Provider, clientID string) string {
	t.Helper()
	q := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {"http://127.0.0.1/callback"},
		"response_type": {"code"},
		"scope":         {"openid"},
		"state":         {"s"},
	}
	target := "/oauth2/auth/local?" + q.Encode()
	for range 3 {
		rec := httptest.NewRecorder()
		p.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusFound {
			require.Equal(t, http.StatusOK, rec.Code, "password page must render: %s", rec.Body.String())
			return rec.Body.String()
		}
		target = rec.Header().Get("Location")
	}
	t.Fatalf("too many redirects, last %s", target)
	return ""
}

func hasBackLink(page string) bool {
	return strings.Contains(strings.ToLower(page), "another login method")
}

func newSignInProvider(t *testing.T) *Provider {
	t.Helper()
	ctx := context.Background()
	tmpDir := t.TempDir()
	p, err := NewProvider(ctx, &Config{Issuer: testIssuer, Port: 5556, DataDir: tmpDir})
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Stop(ctx) })
	for _, c := range []storage.Client{
		{ID: "any-client", RedirectURIs: []string{"http://127.0.0.1/callback"}, Public: true},
		{ID: "local-only", RedirectURIs: []string{"http://127.0.0.1/callback"}, Public: true, AllowedConnectors: []string{"local"}},
	} {
		require.NoError(t, p.storage.CreateClient(ctx, c))
	}
	return p
}

// openInDex loads a stored connector into the running Dex server, as the
// first login through it would.
func openInDex(t *testing.T, p *Provider, id string) {
	t.Helper()
	conn, err := p.storage.GetConnector(context.Background(), id)
	require.NoError(t, err)
	_, err = p.dexServer.OpenConnector(conn)
	require.NoError(t, err, "the registered oidc-assertion factory must open the connector")
}

func TestPasswordPage_BackLinkOnlyWithAnotherLoginMethod(t *testing.T) {
	ctx := context.Background()
	p := newSignInProvider(t)

	assert.False(t, hasBackLink(passwordPage(t, p, "any-client")), "the password connector alone offers no other login method")

	issuer := newFakeIssuer(t)
	_, err := p.CreateConnector(ctx, federatedEntra(issuer.URL))
	require.NoError(t, err)
	openInDex(t, p, "entra-fed")

	assert.True(t, hasBackLink(passwordPage(t, p, "any-client")), "a second connector makes the chooser worth linking")
	assert.False(t, hasBackLink(passwordPage(t, p, "local-only")), "a client limited to the password connector has no chooser")
}

func TestPasswordPage_NoBackLinkForConnectorGoneFromStorage(t *testing.T) {
	ctx := context.Background()
	p := newSignInProvider(t)

	issuer := newFakeIssuer(t)
	_, err := p.CreateConnector(ctx, federatedEntra(issuer.URL))
	require.NoError(t, err)
	openInDex(t, p, "entra-fed")

	// Removed behind the provider's back, as when another replica or a restart
	// of the storage owner deletes it: Dex's cache still holds the connector.
	require.NoError(t, p.storage.DeleteConnector(ctx, "entra-fed"))
	assert.False(t, hasBackLink(passwordPage(t, p, "any-client")), "a connector missing from storage must not count")
}

func TestDeleteConnector_DropsDexCache(t *testing.T) {
	ctx := context.Background()
	p := newSignInProvider(t)

	issuer := newFakeIssuer(t)
	_, err := p.CreateConnector(ctx, federatedEntra(issuer.URL))
	require.NoError(t, err)
	openInDex(t, p, "entra-fed")
	require.NoError(t, p.DeleteConnector(ctx, "entra-fed"))

	// The request skips the provider's filter, so only Dex's own count decides.
	q := url.Values{
		"client_id":     {"any-client"},
		"redirect_uri":  {"http://127.0.0.1/callback"},
		"response_type": {"code"},
		"scope":         {"openid"},
		"state":         {"s"},
	}
	rec := httptest.NewRecorder()
	p.dexServer.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/oauth2/auth/local?"+q.Encode(), nil))
	require.Equal(t, http.StatusFound, rec.Code)
	loc, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	assert.Empty(t, loc.Query().Get("back"), "a deleted connector must leave Dex's cache")
}
