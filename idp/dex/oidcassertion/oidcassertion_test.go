package oidcassertion

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netbirdio/dex/connector"
	"github.com/netbirdio/dex/storage"
)

const (
	testClientID    = "client-id"
	testCallbackURL = "https://netbird.example/oauth2/callback"
	testSubject     = "netbird-identity-provider:entra-test"
	testDexIssuer   = "https://netbird.example/oauth2"
)

// staticKeys is a KeySource holding one signing key, like Dex storage does.
type staticKeys struct {
	keys storage.Keys
}

func (s staticKeys) GetKeys(context.Context) (storage.Keys, error) {
	return s.keys, nil
}

func newDexKeys(t *testing.T) (staticKeys, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return staticKeys{keys: storage.Keys{
		SigningKey:    &jose.JSONWebKey{Key: priv, KeyID: "dex-kid", Algorithm: "RS256", Use: "sig"},
		SigningKeyPub: &jose.JSONWebKey{Key: &priv.PublicKey, KeyID: "dex-kid", Algorithm: "RS256", Use: "sig"},
	}}, &priv.PublicKey
}

// fakeProvider is an upstream OIDC provider that records every token
// request and verifies its client assertion against the Dex public key.
type fakeProvider struct {
	srv          *httptest.Server
	key          *rsa.PrivateKey
	dexPublicKey *rsa.PublicKey

	mu       sync.Mutex
	requests []url.Values
	kids     []string
}

func newFakeProvider(t *testing.T, dexPublicKey *rsa.PublicKey) *fakeProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	f := &fakeProvider{key: key, dexPublicKey: dexPublicKey}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		issuer := "http://" + r.Host
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/token",
			"jwks_uri":                              issuer + "/keys",
			"userinfo_endpoint":                     issuer + "/userinfo",
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"code_challenge_methods_supported":      []string{"S256"},
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: &key.PublicKey, KeyID: "idp-kid", Algorithm: "RS256", Use: "sig"},
		}})
	})
	mux.HandleFunc("/token", f.handleToken)
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeProvider) handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.requests = append(f.requests, r.PostForm)
	f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if _, _, ok := r.BasicAuth(); ok {
		f.reject(w, "basic authentication not accepted")
		return
	}
	if r.PostForm.Get("client_assertion_type") != ClientAssertionType {
		f.reject(w, "missing client_assertion_type")
		return
	}
	kid, err := f.verifyAssertion(r.PostForm.Get("client_assertion"))
	if err != nil {
		f.reject(w, err.Error())
		return
	}
	f.mu.Lock()
	f.kids = append(f.kids, kid)
	f.mu.Unlock()

	refresh := "refresh-1"
	if r.PostForm.Get("grant_type") == "refresh_token" {
		refresh = "refresh-2"
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token":  "access-" + r.PostForm.Get("grant_type"),
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": refresh,
		"id_token":      f.idToken("http://" + r.Host),
	})
}

func (f *fakeProvider) reject(w http.ResponseWriter, description string) {
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_client", "error_description": description})
}

// verifyAssertion checks the assertion the way Entra does: signature against
// the trusted issuer's key, then issuer, subject, audience and expiry.
func (f *fakeProvider) verifyAssertion(raw string) (string, error) {
	tok, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return "", err
	}
	var claims jwt.Claims
	if err := tok.Claims(f.dexPublicKey, &claims); err != nil {
		return "", err
	}
	if err := claims.Validate(jwt.Expected{
		Issuer:      testDexIssuer,
		Subject:     testSubject,
		AnyAudience: jwt.Audience{DefaultAudience},
		Time:        time.Now(),
	}); err != nil {
		return "", err
	}
	if claims.ID == "" {
		return "", assert.AnError
	}
	return tok.Headers[0].KeyID, nil
}

func (f *fakeProvider) idToken(issuer string) string {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: f.key}, (&jose.SignerOptions{}).WithHeader("kid", "idp-kid"))
	if err != nil {
		panic(err)
	}
	raw, err := jwt.Signed(signer).Claims(map[string]any{
		"iss":            issuer,
		"sub":            "pairwise-sub",
		"oid":            "object-id-1",
		"aud":            testClientID,
		"exp":            time.Now().Add(time.Hour).Unix(),
		"name":           "Jane Doe",
		"email":          "jane@example.com",
		"email_verified": true,
	}).Serialize()
	if err != nil {
		panic(err)
	}
	return raw
}

func (f *fakeProvider) tokenRequests() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]url.Values(nil), f.requests...)
}

func (f *fakeProvider) assertionKids() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.kids...)
}

func openConnector(t *testing.T, cfg Config) *oidcConnector {
	t.Helper()
	conn, err := cfg.Open("entra-test", slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.(*oidcConnector).Close() })
	return conn.(*oidcConnector)
}

func federatedConfig(issuer string, keys KeySource) Config {
	return Config{
		Issuer:           issuer,
		ClientID:         testClientID,
		RedirectURI:      testCallbackURL,
		AssertionIssuer:  testDexIssuer,
		AssertionSubject: testSubject,
		UserIDKey:        "oid",
		Keys:             keys,
	}
}

func TestOpen_RequiresAssertionConfig(t *testing.T) {
	keys, pub := newDexKeys(t)
	idp := newFakeProvider(t, pub)
	logger := slog.New(slog.DiscardHandler)

	noKeys := federatedConfig(idp.srv.URL, nil)
	_, err := noKeys.Open("entra-test", logger)
	assert.Error(t, err, "a connector without a key source cannot sign assertions")

	noSubject := federatedConfig(idp.srv.URL, keys)
	noSubject.AssertionSubject = ""
	_, err = noSubject.Open("entra-test", logger)
	assert.Error(t, err, "the federated credential needs a fixed subject")
}

func TestHandleCallback_AuthenticatesWithClientAssertion(t *testing.T) {
	keys, pub := newDexKeys(t)
	idp := newFakeProvider(t, pub)
	conn := openConnector(t, federatedConfig(idp.srv.URL, keys))

	_, connData, err := conn.LoginURL(connector.Scopes{OfflineAccess: true}, testCallbackURL, "state-1")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, testCallbackURL+"?code=code-1&state=state-1", nil)
	identity, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true}, connData, req)
	require.NoError(t, err)

	assert.Equal(t, "object-id-1", identity.UserID, "identity must come from the verified ID token")
	assert.Equal(t, "jane@example.com", identity.Email)

	requests := idp.tokenRequests()
	require.Len(t, requests, 1, "one code exchange expected")
	form := requests[0]
	assert.Equal(t, "authorization_code", form.Get("grant_type"))
	assert.Equal(t, testClientID, form.Get("client_id"), "client_id travels in the body next to the assertion")
	assert.Equal(t, ClientAssertionType, form.Get("client_assertion_type"))
	assert.NotEmpty(t, form.Get("client_assertion"))
	assert.NotContains(t, form, "client_secret", "no secret may be sent")
	assert.NotEmpty(t, form.Get("code_verifier"), "PKCE must keep working")
	assert.Equal(t, []string{"dex-kid"}, idp.assertionKids(), "kid must name the key published in Dex's JWKS")
}

func TestRefresh_AuthenticatesWithClientAssertion(t *testing.T) {
	keys, pub := newDexKeys(t)
	idp := newFakeProvider(t, pub)
	conn := openConnector(t, federatedConfig(idp.srv.URL, keys))

	_, connData, err := conn.LoginURL(connector.Scopes{OfflineAccess: true}, testCallbackURL, "state-1")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, testCallbackURL+"?code=code-1&state=state-1", nil)
	identity, err := conn.HandleCallback(connector.Scopes{OfflineAccess: true}, connData, req)
	require.NoError(t, err)

	refreshed, err := conn.Refresh(context.Background(), connector.Scopes{OfflineAccess: true}, identity)
	require.NoError(t, err)
	assert.Equal(t, "object-id-1", refreshed.UserID)

	var cd connectorData
	require.NoError(t, json.Unmarshal(refreshed.ConnectorData, &cd))
	assert.Equal(t, "refresh-2", string(cd.RefreshToken), "the rotated refresh token must be stored")

	requests := idp.tokenRequests()
	require.Len(t, requests, 2, "code exchange then refresh expected")
	form := requests[1]
	assert.Equal(t, "refresh_token", form.Get("grant_type"))
	assert.Equal(t, "refresh-1", form.Get("refresh_token"))
	assert.Equal(t, testClientID, form.Get("client_id"))
	assert.Equal(t, ClientAssertionType, form.Get("client_assertion_type"))
	assert.NotContains(t, form, "client_secret", "no secret may be sent")
	assert.NotEqual(t, requests[0].Get("client_assertion"), form.Get("client_assertion"), "every request gets a fresh assertion")
	assert.Equal(t, []string{"dex-kid", "dex-kid"}, idp.assertionKids())
}

func TestRefresh_SurfacesTokenEndpointError(t *testing.T) {
	keys, pub := newDexKeys(t)
	idp := newFakeProvider(t, pub)
	conn := openConnector(t, federatedConfig(idp.srv.URL, keys))

	// A key the provider does not trust makes the assertion fail verification.
	otherKeys, _ := newDexKeys(t)
	conn.keys = otherKeys

	cd, err := json.Marshal(connectorData{RefreshToken: []byte("refresh-1")})
	require.NoError(t, err)
	_, err = conn.Refresh(context.Background(), connector.Scopes{}, connector.Identity{ConnectorData: cd})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_client", "the provider's error must reach the caller")
}

func TestClientAssertion_Claims(t *testing.T) {
	keys, pub := newDexKeys(t)
	idp := newFakeProvider(t, pub)
	conn := openConnector(t, federatedConfig(idp.srv.URL, keys))

	raw, err := conn.clientAssertion(context.Background())
	require.NoError(t, err)

	tok, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.RS256})
	require.NoError(t, err)
	require.Len(t, tok.Headers, 1)
	assert.Equal(t, "dex-kid", tok.Headers[0].KeyID)
	assert.Equal(t, "JWT", tok.Headers[0].ExtraHeaders[jose.HeaderType])

	var claims jwt.Claims
	require.NoError(t, tok.Claims(pub, &claims))
	assert.Equal(t, testDexIssuer, claims.Issuer)
	assert.Equal(t, testSubject, claims.Subject)
	assert.Equal(t, jwt.Audience{DefaultAudience}, claims.Audience, "audience defaults to the Entra token exchange audience")
	assert.NotEmpty(t, claims.ID, "jti is required for a one-time assertion")
	require.NotNil(t, claims.Expiry)
	require.NotNil(t, claims.IssuedAt)
	assert.Equal(t, assertionLifetime, claims.Expiry.Time().Sub(claims.IssuedAt.Time()))
}
