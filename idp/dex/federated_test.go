package dex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netbirdio/netbird/idp/dex/oidcassertion"
)

const testIssuer = "https://netbird.example"

// newFakeIssuer serves just enough OIDC discovery for Dex to open a connector
// against it; no login ever completes here.
func newFakeIssuer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		issuer := "http://" + r.Host
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/token",
			"jwks_uri":                              issuer + "/keys",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"keys":[]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func federatedEntra(issuer string) *ConnectorConfig {
	return &ConnectorConfig{
		ID:         "entra-fed",
		Name:       "Entra",
		Type:       "entra",
		Issuer:     issuer,
		ClientID:   "client-id",
		ClientAuth: ClientAuthFederated,
	}
}

func decodeConfig(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func TestBuildStorageConnector_FederatedEntra(t *testing.T) {
	p := &Provider{config: &Config{Issuer: testIssuer}}

	conn, err := p.buildStorageConnector(federatedEntra("https://login.microsoftonline.com/tid/v2.0"))
	require.NoError(t, err)

	assert.Equal(t, oidcassertion.Type, conn.Type, "federated entra must use the assertion connector")
	m := decodeConfig(t, conn.Config)
	_, hasSecret := m["clientSecret"]
	assert.False(t, hasSecret, "no secret may be stored for a federated connector")
	assert.Equal(t, testIssuer+"/oauth2", m["assertionIssuer"], "the assertion is issued by this server's Dex issuer")
	assert.Equal(t, "netbird-identity-provider:client-id", m["assertionSubject"], "the subject follows the app's client ID")
	assert.Equal(t, oidcassertion.DefaultAudience, m["assertionAudience"])
	assert.Equal(t, "oid", m["userIDKey"], "entra defaults must still apply")
	assert.Equal(t, map[string]any{"email": "preferred_username"}, m["claimMapping"])
}

func TestBuildStorageConnector_FederatedRequiresEntraAndIssuer(t *testing.T) {
	p := &Provider{config: &Config{Issuer: testIssuer}}
	for _, typ := range []string{"oidc", "okta", "google", "microsoft"} {
		cfg := federatedEntra("https://idp.example")
		cfg.Type = typ
		_, err := p.buildStorageConnector(cfg)
		assert.Error(t, err, "%s must not accept federated client authentication", typ)
	}

	noIssuer := &Provider{}
	_, err := noIssuer.buildStorageConnector(federatedEntra("https://idp.example"))
	assert.Error(t, err, "without this server's issuer the assertion cannot name a trusted issuer")

	secret := federatedEntra("https://idp.example")
	secret.ClientAuth = ClientAuthSecret
	secret.ClientSecret = "s"
	conn, err := p.buildStorageConnector(secret)
	require.NoError(t, err)
	assert.Equal(t, "oidc", conn.Type, "secret authentication keeps the plain oidc connector")
}

func TestConnectorRoundTrip_ClientAuth(t *testing.T) {
	ctx := context.Background()
	p, cleanup := newTestProvider(t)
	defer cleanup()
	p.config = &Config{Issuer: testIssuer}

	_, err := p.CreateConnector(ctx, federatedEntra("https://login.microsoftonline.com/tid/v2.0"))
	require.NoError(t, err)
	secret := &ConnectorConfig{ID: "entra-secret", Name: "Entra", Type: "entra", Issuer: "https://login.microsoftonline.com/tid/v2.0", ClientID: "c", ClientSecret: "s"}
	_, err = p.CreateConnector(ctx, secret)
	require.NoError(t, err)

	got, err := p.GetConnector(ctx, "entra-fed")
	require.NoError(t, err)
	assert.Equal(t, "entra", got.Type, "the provider type is still inferred from the connector ID")
	assert.Equal(t, ClientAuthFederated, got.ClientAuth)
	assert.Empty(t, got.ClientSecret)

	got, err = p.GetConnector(ctx, "entra-secret")
	require.NoError(t, err)
	assert.Equal(t, ClientAuthSecret, got.ClientAuth, "a stored secret connector reports secret authentication")

	list, err := p.ListConnectors(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestUpdateConnector_SwitchesClientAuth(t *testing.T) {
	ctx := context.Background()
	p, cleanup := newTestProvider(t)
	defer cleanup()
	p.config = &Config{Issuer: testIssuer}

	secret := &ConnectorConfig{ID: "entra-1", Name: "Entra", Type: "entra", Issuer: "https://login.microsoftonline.com/tid/v2.0", ClientID: "c", ClientSecret: "s", RedirectURI: "https://netbird.example/oauth2/callback"}
	_, err := p.CreateConnector(ctx, secret)
	require.NoError(t, err)

	// Secret to federated: the secret is dropped and the stored issuer, client
	// ID and redirect URI survive even though the update leaves them empty.
	require.NoError(t, p.UpdateConnector(ctx, &ConnectorConfig{ID: "entra-1", Type: "entra", ClientAuth: ClientAuthFederated}))
	raw, err := p.storage.GetConnector(ctx, "entra-1")
	require.NoError(t, err)
	assert.Equal(t, oidcassertion.Type, raw.Type)
	m := decodeConfig(t, raw.Config)
	_, hasSecret := m["clientSecret"]
	assert.False(t, hasSecret, "switching to federated must drop the secret")
	assert.Equal(t, "https://login.microsoftonline.com/tid/v2.0", m["issuer"])
	assert.Equal(t, "c", m["clientID"])
	assert.Equal(t, "https://netbird.example/oauth2/callback", m["redirectURI"])
	assert.Equal(t, "netbird-identity-provider:c", m["assertionSubject"])
	assert.Equal(t, "Entra", raw.Name, "the name is kept when the update leaves it empty")

	// A new client ID moves the subject with it, or Entra would reject the
	// assertion for the new app.
	require.NoError(t, p.UpdateConnector(ctx, &ConnectorConfig{ID: "entra-1", ClientID: "c2"}))
	raw, err = p.storage.GetConnector(ctx, "entra-1")
	require.NoError(t, err)
	m = decodeConfig(t, raw.Config)
	assert.Equal(t, "c2", m["clientID"])
	assert.Equal(t, "netbird-identity-provider:c2", m["assertionSubject"])

	// Federated to secret needs a secret to switch to.
	err = p.UpdateConnector(ctx, &ConnectorConfig{ID: "entra-1", ClientAuth: ClientAuthSecret})
	assert.Error(t, err, "switching back without a secret must be refused")

	require.NoError(t, p.UpdateConnector(ctx, &ConnectorConfig{ID: "entra-1", ClientAuth: ClientAuthSecret, ClientSecret: "new"}))
	raw, err = p.storage.GetConnector(ctx, "entra-1")
	require.NoError(t, err)
	assert.Equal(t, "oidc", raw.Type)
	m = decodeConfig(t, raw.Config)
	assert.Equal(t, "new", m["clientSecret"])
	_, hasAssertion := m["assertionSubject"]
	assert.False(t, hasAssertion, "assertion claims must not leak into a secret connector")

	// An update without client auth keeps the current mode.
	require.NoError(t, p.UpdateConnector(ctx, &ConnectorConfig{ID: "entra-1", Name: "Renamed"}))
	raw, err = p.storage.GetConnector(ctx, "entra-1")
	require.NoError(t, err)
	assert.Equal(t, "oidc", raw.Type)
	assert.Equal(t, "Renamed", raw.Name)
}
