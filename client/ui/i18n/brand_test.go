//go:build !android && !ios && !freebsd && !js

package i18n

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every branded key must exist in the English bundle; an upstream rename
// would otherwise silently bring "NetBird" back into the app.
func TestBrandedKeysExist(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("locales", "en", commonBundleFile))
	require.NoError(t, err)
	var entries map[string]bundleEntry
	require.NoError(t, json.Unmarshal(data, &entries))
	for key := range brandedKeys {
		_, ok := entries[key]
		assert.True(t, ok, "branded key %q is no longer in locales/en/common.json", key)
	}
}

func TestBrandMessage(t *testing.T) {
	assert.Equal(t, "Open Sightwell", brandMessage("tray.menu.open", "Open NetBird"))
	assert.Equal(t, "Launch Sightwell at Login", brandMessage("settings.general.autostart.label", "Launch NetBird UI at Login"))
	assert.Equal(t, "Sightwell Connectivity v{version}", brandMessage("settings.about.client", "NetBird Client v{version}"))
	// Technology references are left alone.
	assert.Equal(t, "NetBird IP", brandMessage("peers.details.netbirdIp", "NetBird IP"))
	assert.Equal(t, "NetBird Service Is Not Running", brandMessage("daemon.unavailable.title", "NetBird Service Is Not Running"))
}

// Branding applies to every shipped locale, through the real loader.
func TestBrandingAppliesToAllLocales(t *testing.T) {
	b, err := NewBundle(os.DirFS("locales"))
	require.NoError(t, err)
	for _, l := range b.Languages() {
		bundle, err := b.BundleFor(l.Code)
		require.NoError(t, err)
		for key := range brandedKeys {
			msg, ok := bundle[key]
			if !ok {
				continue // untranslated keys fall back to English at lookup time
			}
			assert.False(t, strings.Contains(msg, "NetBird"), "%s %s still says NetBird: %q", l.Code, key, msg)
		}
		assert.Contains(t, bundle["peers.details.netbirdIp"], "NetBird", "%s: technology names must stay", l.Code)
	}
}

// Self-hosted only: every shipped locale has every replacement text, and the
// replaced messages no longer offer NetBird Cloud.
func TestSelfHostedMessagesCoverEveryLocale(t *testing.T) {
	if selfHostedMessages == nil {
		t.Skip("NetBird Cloud is enabled in brand.json")
	}
	b, err := NewBundle(os.DirFS("locales"))
	require.NoError(t, err)
	for key, byLocale := range selfHostedMessages {
		for _, l := range b.Languages() {
			text, ok := byLocale[string(l.Code)]
			require.True(t, ok, "%s has no %s text", key, l.Code)
			assert.NotContains(t, text, "Cloud", "%s %s", key, l.Code)
			bundle, err := b.BundleFor(l.Code)
			require.NoError(t, err)
			assert.Equal(t, text, bundle[key], "%s %s not applied", key, l.Code)
		}
	}
}

// Sightwell fork: the About page reads links.* from brand.json; keep them
// present and on https so a typo can't ship a dead or insecure link.
func TestBrandLinks(t *testing.T) {
	var cfg struct {
		Links map[string]string `json:"links"`
	}
	if err := json.Unmarshal(brandJSON, &cfg); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"slack", "documentation", "releases"} {
		u, err := url.Parse(cfg.Links[k])
		if err != nil || u.Scheme != "https" || u.Host == "" {
			t.Errorf("links.%s = %q, want an https URL", k, cfg.Links[k])
		}
	}
}

// brand.json messages replace their key in every shipped locale; the
// copyright line carries the NetBird BSD-3-Clause notice.
func TestAllLocaleMessagesApplied(t *testing.T) {
	require.Contains(t, allLocaleMessages, "settings.about.copyright")
	assert.Contains(t, allLocaleMessages["settings.about.copyright"], "NetBird GmbH & AUTHORS")
	assert.Contains(t, allLocaleMessages["settings.about.copyright"], "BSD-3-Clause")
	b, err := NewBundle(os.DirFS("locales"))
	require.NoError(t, err)
	for key, text := range allLocaleMessages {
		for _, l := range b.Languages() {
			bundle, err := b.BundleFor(l.Code)
			require.NoError(t, err)
			if _, ok := bundle[key]; !ok {
				continue // untranslated: falls back to English, which is replaced
			}
			assert.Equal(t, text, bundle[key], "%s %s not applied", key, l.Code)
		}
	}
}

// The About page's legal row comes from brand.json legalLinks: every entry
// (if any) needs a translated label (settings.about.links.<key>) and an https URL.
func TestLegalLinks(t *testing.T) {
	var cfg struct {
		LegalLinks []struct {
			Key string `json:"key"`
			URL string `json:"url"`
		} `json:"legalLinks"`
	}
	require.NoError(t, json.Unmarshal(brandJSON, &cfg))
	// Empty is allowed: the About page then hides the legal row.
	data, err := os.ReadFile(filepath.Join("locales", "en", commonBundleFile))
	require.NoError(t, err)
	var entries map[string]bundleEntry
	require.NoError(t, json.Unmarshal(data, &entries))
	for _, l := range cfg.LegalLinks {
		_, ok := entries["settings.about.links."+l.Key]
		assert.True(t, ok, "legal link %q has no settings.about.links.%s label", l.Key, l.Key)
		u, err := url.Parse(l.URL)
		assert.True(t, err == nil && u.Scheme == "https" && u.Host != "", "legal link %q: %q is not an https URL", l.Key, l.URL)
	}
}
