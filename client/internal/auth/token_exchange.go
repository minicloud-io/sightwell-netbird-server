package auth

// Sightwell fork: sign-in with a token from the upstream identity provider
// (for example an Entra ID token obtained natively on iOS with MSAL). The token
// is exchanged (RFC 8693) at the token endpoint the management server announces
// for device sign-in, with the same client, scope and audience rules as the
// browser (PKCE) flow, and the result is used like a browser sign-in token.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/netbirdio/netbird/shared/management/client/common"

	mgm "github.com/netbirdio/netbird/shared/management/client"
)

const (
	grantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
	// TokenTypeIDToken is the RFC 8693 type of an OpenID Connect ID token.
	TokenTypeIDToken     = "urn:ietf:params:oauth:token-type:id_token"
	tokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"
	exchangeTimeout      = 30 * time.Second
)

// UpstreamToken is a token issued by the identity provider behind the
// management server's IdP (for Dex: by the connector's upstream).
type UpstreamToken struct {
	SubjectToken     string
	SubjectTokenType string // defaults to TokenTypeIDToken
	ConnectorID      string // Dex connector that trusts the upstream token; optional for other IdPs
}

// ExchangeUpstreamToken exchanges an upstream token for a token the management
// server accepts, using the device sign-in configuration it announces.
func (a *Auth) ExchangeUpstreamToken(ctx context.Context, up UpstreamToken) (TokenInfo, error) {
	var cfg PKCEAuthProviderConfig
	err := a.withRetry(ctx, func(client *mgm.GrpcClient) error {
		c, err := a.pkceProviderConfig(client)
		if err != nil {
			return err
		}
		cfg = c
		return nil
	})
	if err != nil {
		return TokenInfo{}, fmt.Errorf("get sign-in configuration: %w", err)
	}
	return exchangeToken(ctx, httpClientFor(cfg), cfg, up)
}

// pkceProviderConfig reads the provider configuration of the PKCE flow
// without opening a loopback listener (the exchange needs none).
func (a *Auth) pkceProviderConfig(client *mgm.GrpcClient) (PKCEAuthProviderConfig, error) {
	protoFlow, err := client.GetPKCEAuthorizationFlow(false)
	if err != nil {
		return PKCEAuthProviderConfig{}, err
	}
	pc := protoFlow.GetProviderConfig()
	cfg := PKCEAuthProviderConfig{
		Audience:              pc.GetAudience(),
		ClientID:              pc.GetClientID(),
		ClientSecret:          pc.GetClientSecret(), //nolint:staticcheck
		TokenEndpoint:         pc.GetTokenEndpoint(),
		AuthorizationEndpoint: pc.GetAuthorizationEndpoint(),
		Scope:                 pc.GetScope(),
		RedirectURLs:          pc.GetRedirectURLs(),
		UseIDToken:            pc.GetUseIDToken(),
		ClientCertPair:        a.config.ClientCertKeyPair,
		LoginFlag:             common.LoginFlag(pc.GetLoginFlag()),
	}
	if err := validatePKCEConfig(&cfg); err != nil {
		return PKCEAuthProviderConfig{}, err
	}
	return cfg, nil
}

func httpClientFor(cfg PKCEAuthProviderConfig) *http.Client {
	client := &http.Client{Timeout: exchangeTimeout}
	if cfg.ClientCertPair != nil {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{*cfg.ClientCertPair}},
		}
	}
	return client
}

type exchangeResponse struct {
	AccessToken     string `json:"access_token"`
	IDToken         string `json:"id_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
	Error           string `json:"error"`
	ErrorDesc       string `json:"error_description"`
}

func exchangeToken(ctx context.Context, client *http.Client, cfg PKCEAuthProviderConfig, up UpstreamToken) (TokenInfo, error) {
	if up.SubjectToken == "" {
		return TokenInfo{}, fmt.Errorf("upstream token is empty")
	}
	subjectType := up.SubjectTokenType
	if subjectType == "" {
		subjectType = TokenTypeIDToken
	}
	requested := tokenTypeAccessToken
	if cfg.UseIDToken {
		requested = TokenTypeIDToken
	}
	scope := cfg.Scope
	if scope == "" {
		scope = "openid profile email"
	}

	form := url.Values{
		"grant_type":           {grantTypeTokenExchange},
		"client_id":            {cfg.ClientID},
		"subject_token":        {up.SubjectToken},
		"subject_token_type":   {subjectType},
		"requested_token_type": {requested},
		"scope":                {scope},
	}
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}
	if up.ConnectorID != "" {
		form.Set("connector_id", up.ConnectorID)
	}

	ctx, cancel := context.WithTimeout(ctx, exchangeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenInfo{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return TokenInfo{}, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return TokenInfo{}, fmt.Errorf("read token exchange response: %w", err)
	}
	var r exchangeResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return TokenInfo{}, fmt.Errorf("token exchange: HTTP %d, unreadable response", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || r.Error != "" {
		msg := r.Error
		if r.ErrorDesc != "" {
			msg += ": " + r.ErrorDesc
		}
		return TokenInfo{}, fmt.Errorf("token exchange refused (HTTP %d): %s", resp.StatusCode, msg)
	}

	// RFC 8693 returns the issued token in access_token whatever its type.
	info := TokenInfo{TokenType: r.TokenType, ExpiresIn: r.ExpiresIn, UseIDToken: cfg.UseIDToken}
	if cfg.UseIDToken {
		info.IDToken = r.AccessToken
		if r.IDToken != "" {
			info.IDToken = r.IDToken
		}
	} else {
		info.AccessToken = r.AccessToken
		info.IDToken = r.IDToken
	}

	audience := cfg.Audience
	if audience == "" {
		audience = cfg.ClientID
	}
	if err := validateTokenAudience(info.GetTokenToUse(), audience); err != nil {
		return TokenInfo{}, fmt.Errorf("token exchange returned an unusable token: %w", err)
	}

	emailSource := info.IDToken
	if emailSource == "" {
		emailSource = info.GetTokenToUse()
	}
	if email, fromEmailClaim, err := parseEmailFromIDToken(emailSource); err == nil {
		info.Email = email
		if fromEmailClaim {
			info.EmailClaim = email
		}
	}
	return info, nil
}
