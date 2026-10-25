package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testJWT builds an unsigned JWT; the exchange only reads claims (the
// management server verifies signatures).
func testJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	enc := func(v any) string {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return enc(map[string]string{"alg": "none"}) + "." + enc(claims) + ".sig"
}

func exchangeServer(t *testing.T, check func(url.Values), status int, resp map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, r.ParseForm())
		if check != nil {
			check(r.PostForm)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestExchangeTokenAccessToken(t *testing.T) {
	issued := testJWT(t, map[string]any{"aud": "netbird-cli", "email": "user@example.com"})
	srv := exchangeServer(t, func(f url.Values) {
		assert.Equal(t, grantTypeTokenExchange, f.Get("grant_type"))
		assert.Equal(t, "netbird-cli", f.Get("client_id"))
		assert.Equal(t, "upstream-id-token", f.Get("subject_token"))
		assert.Equal(t, TokenTypeIDToken, f.Get("subject_token_type"), "id_token is the default subject type")
		assert.Equal(t, tokenTypeAccessToken, f.Get("requested_token_type"))
		assert.Equal(t, "entra-1", f.Get("connector_id"))
		assert.Equal(t, "openid profile email", f.Get("scope"))
		assert.Empty(t, f.Get("client_secret"), "public client sends no secret")
	}, http.StatusOK, map[string]any{"access_token": issued, "issued_token_type": tokenTypeAccessToken, "token_type": "bearer", "expires_in": 3600})

	cfg := PKCEAuthProviderConfig{ClientID: "netbird-cli", TokenEndpoint: srv.URL, Scope: "openid profile email"}
	info, err := exchangeToken(context.Background(), srv.Client(), cfg, UpstreamToken{SubjectToken: "upstream-id-token", ConnectorID: "entra-1"})
	require.NoError(t, err)
	assert.Equal(t, issued, info.GetTokenToUse())
	assert.False(t, info.UseIDToken)
	assert.Equal(t, "user@example.com", info.Email)
	assert.Equal(t, 3600, info.ExpiresIn)
}

func TestExchangeTokenIDTokenMode(t *testing.T) {
	issued := testJWT(t, map[string]any{"aud": "netbird-cli", "email": "user@example.com"})
	srv := exchangeServer(t, func(f url.Values) {
		assert.Equal(t, TokenTypeIDToken, f.Get("requested_token_type"))
	}, http.StatusOK, map[string]any{"access_token": issued, "issued_token_type": TokenTypeIDToken})

	cfg := PKCEAuthProviderConfig{ClientID: "netbird-cli", TokenEndpoint: srv.URL, UseIDToken: true}
	info, err := exchangeToken(context.Background(), srv.Client(), cfg, UpstreamToken{SubjectToken: "t"})
	require.NoError(t, err)
	assert.True(t, info.UseIDToken)
	assert.Equal(t, issued, info.IDToken)
	assert.Equal(t, issued, info.GetTokenToUse())
}

func TestExchangeTokenAudienceChecked(t *testing.T) {
	srv := exchangeServer(t, nil, http.StatusOK, map[string]any{"access_token": testJWT(t, map[string]any{"aud": "someone-else"})})
	cfg := PKCEAuthProviderConfig{ClientID: "netbird-cli", TokenEndpoint: srv.URL}
	_, err := exchangeToken(context.Background(), srv.Client(), cfg, UpstreamToken{SubjectToken: "t"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unusable token")

	// An announced audience takes precedence over the client ID.
	srv2 := exchangeServer(t, nil, http.StatusOK, map[string]any{"access_token": testJWT(t, map[string]any{"aud": "api"})})
	cfg2 := PKCEAuthProviderConfig{ClientID: "netbird-cli", Audience: "api", TokenEndpoint: srv2.URL}
	_, err = exchangeToken(context.Background(), srv2.Client(), cfg2, UpstreamToken{SubjectToken: "t"})
	require.NoError(t, err)
}

func TestExchangeTokenRefused(t *testing.T) {
	srv := exchangeServer(t, nil, http.StatusBadRequest, map[string]any{"error": "access_denied", "error_description": "connector does not allow token exchange"})
	cfg := PKCEAuthProviderConfig{ClientID: "netbird-cli", TokenEndpoint: srv.URL}
	_, err := exchangeToken(context.Background(), srv.Client(), cfg, UpstreamToken{SubjectToken: "t"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 400")
	assert.Contains(t, err.Error(), "connector does not allow token exchange")
}

func TestExchangeTokenEmptySubject(t *testing.T) {
	_, err := exchangeToken(context.Background(), http.DefaultClient, PKCEAuthProviderConfig{ClientID: "c", TokenEndpoint: "http://unused"}, UpstreamToken{})
	require.Error(t, err)
}

func TestExchangeTokenConfidentialClient(t *testing.T) {
	srv := exchangeServer(t, func(f url.Values) {
		assert.Equal(t, "s3cret", f.Get("client_secret"))
		assert.Empty(t, f.Get("connector_id"), "connector is optional")
	}, http.StatusOK, map[string]any{"access_token": testJWT(t, map[string]any{"aud": "c"})})
	cfg := PKCEAuthProviderConfig{ClientID: "c", ClientSecret: "s3cret", TokenEndpoint: srv.URL}
	_, err := exchangeToken(context.Background(), srv.Client(), cfg, UpstreamToken{SubjectToken: "t"})
	require.NoError(t, err)
}
