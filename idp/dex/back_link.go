package dex

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/netbirdio/dex/server"
)

// Grant types and the device callback path as Dex derives them in its
// connector chooser.
const (
	grantTypeAuthorizationCode = "authorization_code"
	grantTypeImplicit          = "implicit"
	grantTypeDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
	deviceCallbackPath         = "/device/callback"
)

var passwordLoginPath = regexp.MustCompile(`/auth/[^/]+/login$`)

// withLoginBackLink drops the "back" link from password login requests when
// the chooser it points to would offer a single login method. Dex decides on
// the link by counting its in-memory connector cache, which also holds
// connectors the chooser filters out for the client and connectors deleted
// from storage, so the password page could link to a chooser that only
// redirects back to it. The templates render the link only when it is set.
func (p *Provider) withLoginBackLink(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, p.dropSingleMethodBackLink(r))
	})
}

func (p *Provider) dropSingleMethodBackLink(r *http.Request) *http.Request {
	if !passwordLoginPath.MatchString(r.URL.Path) {
		return r
	}
	q := r.URL.Query()
	if q.Get("back") == "" {
		return r
	}

	count, err := p.loginMethodCount(r.Context(), q.Get("state"))
	if err != nil {
		p.logger.Debug("keeping login back link", "error", err)
		return r
	}
	if count > 1 {
		return r
	}

	q.Del("back")
	out := r.Clone(r.Context())
	out.URL.RawQuery = q.Encode()
	out.RequestURI = out.URL.RequestURI()
	return out
}

// loginMethodCount returns how many connectors the chooser would offer for
// the auth request, applying the same grant type and client filters as Dex.
func (p *Provider) loginMethodCount(ctx context.Context, authReqID string) (int, error) {
	if authReqID == "" {
		return 0, fmt.Errorf("no auth request")
	}
	authReq, err := p.storage.GetAuthRequest(ctx, authReqID)
	if err != nil {
		return 0, fmt.Errorf("get auth request: %w", err)
	}
	client, err := p.storage.GetClient(ctx, authReq.ClientID)
	if err != nil {
		return 0, fmt.Errorf("get client: %w", err)
	}
	connectors, err := p.storage.ListConnectors(ctx)
	if err != nil {
		return 0, fmt.Errorf("list connectors: %w", err)
	}

	grantType := authRequestGrantType(authReq.RedirectURI, authReq.ResponseTypes)
	count := 0
	for _, c := range connectors {
		if !server.GrantTypeAllowed(c.GrantTypes, grantType) {
			continue
		}
		if len(client.AllowedConnectors) > 0 && !slices.Contains(client.AllowedConnectors, c.ID) {
			continue
		}
		count++
	}
	return count, nil
}

// authRequestGrantType mirrors how Dex's chooser derives the grant type of an
// authorization request.
func authRequestGrantType(redirectURI string, responseTypes []string) string {
	if strings.HasSuffix(redirectURI, deviceCallbackPath) {
		return grantTypeDeviceCode
	}
	for _, rt := range responseTypes {
		if rt == "token" || rt == "id_token" {
			return grantTypeImplicit
		}
	}
	return grantTypeAuthorizationCode
}
