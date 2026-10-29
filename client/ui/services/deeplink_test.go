//go:build !android && !ios && !freebsd && !js

package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLinkAccepts(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want AddProfileLink
	}{
		{
			name: "name and url",
			raw:  "sightwell://add-profile?name=Alder%20Cove%20Energy&url=https://aldercove-energy.sightwell.ai",
			want: AddProfileLink{
				Name:          "Alder Cove Energy",
				ManagementURL: "https://aldercove-energy.sightwell.ai",
				Host:          "aldercove-energy.sightwell.ai",
			},
		},
		{
			name: "url encoded url and trailing slashes",
			raw:  "sightwell://add-profile/?url=https%3A%2F%2Faldercove-energy.sightwell.ai%2F&name=Alder",
			want: AddProfileLink{
				Name:          "Alder",
				ManagementURL: "https://aldercove-energy.sightwell.ai",
				Host:          "aldercove-energy.sightwell.ai",
			},
		},
		{
			// Without a name the host is the profile name, never an empty one.
			name: "no name",
			raw:  "sightwell://add-profile?url=https://tenant.example.com",
			want: AddProfileLink{
				Name:          "tenant.example.com",
				ManagementURL: "https://tenant.example.com",
				Host:          "tenant.example.com",
			},
		},
		{
			name: "case is normalised and 443 dropped",
			raw:  "SIGHTWELL://Add-Profile?url=https://Tenant.Example.COM:443&name=T",
			want: AddProfileLink{
				Name:          "T",
				ManagementURL: "https://tenant.example.com",
				Host:          "tenant.example.com",
			},
		},
		{
			name: "non-default port kept",
			raw:  "sightwell://add-profile?url=https://tenant.example.com:8443&name=T",
			want: AddProfileLink{
				Name:          "T",
				ManagementURL: "https://tenant.example.com:8443",
				Host:          "tenant.example.com:8443",
			},
		},
		{
			// Bidi overrides and newlines could disguise the name in the
			// confirmation, so they are removed.
			name: "name is cleaned",
			raw:  "sightwell://add-profile?url=https://tenant.example.com&name=%20Evil%E2%80%AE%0Acorp%20%20",
			want: AddProfileLink{
				Name:          "Evil corp",
				ManagementURL: "https://tenant.example.com",
				Host:          "tenant.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLink(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "parsed link for %q", tt.raw)
		})
	}
}

func TestParseLinkRefuses(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want LinkRefusal
	}{
		{name: "http url", raw: "sightwell://add-profile?url=http://tenant.example.com", want: LinkRefusedNotHTTPS},
		{name: "url without scheme", raw: "sightwell://add-profile?url=tenant.example.com", want: LinkRefusedNotHTTPS},
		{name: "javascript url", raw: "sightwell://add-profile?url=javascript:alert(1)", want: LinkRefusedNotHTTPS},
		{name: "missing url", raw: "sightwell://add-profile?name=T", want: LinkRefusedMissingHost},
		{name: "empty url", raw: "sightwell://add-profile?url=&name=T", want: LinkRefusedMissingHost},
		{name: "url without host", raw: "sightwell://add-profile?url=https://", want: LinkRefusedMissingHost},
		{name: "url with only a port", raw: "sightwell://add-profile?url=https://:443", want: LinkRefusedMissingHost},
		{name: "unknown action", raw: "sightwell://remove-profile?url=https://tenant.example.com", want: LinkRefusedUnknownAction},
		{name: "login action", raw: "sightwell://login", want: LinkRefusedUnknownAction},
		{name: "no action", raw: "sightwell://?url=https://tenant.example.com", want: LinkRefusedUnknownAction},
		{name: "other scheme", raw: "https://add-profile?url=https://tenant.example.com", want: LinkRefusedInvalid},
		{name: "action as path", raw: "sightwell:add-profile?url=https://tenant.example.com", want: LinkRefusedInvalid},
		{name: "extra path on link", raw: "sightwell://add-profile/x?url=https://tenant.example.com", want: LinkRefusedInvalid},
		{name: "repeated url", raw: "sightwell://add-profile?url=https://a.example.com&url=https://b.example.com", want: LinkRefusedInvalid},
		{name: "credentials in url", raw: "sightwell://add-profile?url=https://user:pw@tenant.example.com", want: LinkRefusedInvalid},
		{name: "path on url", raw: "sightwell://add-profile?url=https://tenant.example.com/api", want: LinkRefusedInvalid},
		{name: "query on url", raw: "sightwell://add-profile?url=https%3A%2F%2Ftenant.example.com%3Fx%3D1", want: LinkRefusedInvalid},
		{name: "fragment on url", raw: "sightwell://add-profile?url=https%3A%2F%2Ftenant.example.com%23x", want: LinkRefusedInvalid},
		{name: "ip address", raw: "sightwell://add-profile?url=https://10.0.0.1", want: LinkRefusedInvalid},
		{name: "ipv6 address", raw: "sightwell://add-profile?url=https://[::1]", want: LinkRefusedInvalid},
		{name: "single label", raw: "sightwell://add-profile?url=https://localhost", want: LinkRefusedInvalid},
		{name: "underscore in host", raw: "sightwell://add-profile?url=https://bad_host.example.com", want: LinkRefusedInvalid},
		{name: "port out of range", raw: "sightwell://add-profile?url=https://tenant.example.com:70000", want: LinkRefusedInvalid},
		{name: "too long", raw: "sightwell://add-profile?url=https://tenant.example.com&name=" + strings.Repeat("a", maxLinkLen), want: LinkRefusedInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseLink(tt.raw)
			require.Error(t, err, "link %q must be refused", tt.raw)
			var le *LinkError
			require.True(t, errors.As(err, &le), "refusal must be a *LinkError, got %T", err)
			assert.Equal(t, tt.want, le.Refusal, "refusal for %q", tt.raw)
		})
	}
}

func TestParseLinkCapsNameLength(t *testing.T) {
	got, err := ParseLink("sightwell://add-profile?url=https://tenant.example.com&name=" + strings.Repeat("é", 200))
	require.NoError(t, err)
	assert.Equal(t, maxLinkNameLen, len([]rune(got.Name)), "name is capped in runes, not bytes")
}

func TestLinkInbox(t *testing.T) {
	var events []string
	inbox := NewLinkInbox(func(event string) { events = append(events, event) })

	// Launch arguments that are not links are left alone.
	inbox.Receive("--log-level")
	assert.Empty(t, events, "a non-link argument must not notify")
	assert.Equal(t, PendingLink{}, inbox.Take(), "nothing pending after a non-link")

	inbox.Receive("sightwell://add-profile?url=http://tenant.example.com")
	assert.Equal(t, []string{EventLinkReceived}, events, "a refused link still notifies")
	assert.Equal(t, PendingLink{Refusal: LinkRefusedNotHTTPS}, inbox.Take(), "refusal is passed on")

	// The last click wins over one the user has not confirmed yet.
	inbox.Receive("sightwell://add-profile?url=https://a.example.com&name=A")
	inbox.Receive("sightwell://add-profile?url=https://b.example.com&name=B")
	got := inbox.Take()
	require.NotNil(t, got.Link, "a valid link is pending")
	assert.Equal(t, "B", got.Link.Name, "the newest link replaces an unconfirmed one")
	assert.Equal(t, PendingLink{}, inbox.Take(), "a taken link is offered once")
}

func TestFindLink(t *testing.T) {
	got, ok := FindLink([]string{"/Applications/Sightwell.app/Contents/MacOS/sightwell-ui", "Sightwell://add-profile?url=https://t.example.com"})
	assert.True(t, ok, "link found among arguments")
	assert.Equal(t, "Sightwell://add-profile?url=https://t.example.com", got)

	_, ok = FindLink([]string{"sightwell-ui", "--daemon-addr", "unix:///var/run/sightwell.sock"})
	assert.False(t, ok, "no link among ordinary arguments")
}
