//go:build !android && !ios && !freebsd && !js

package i18n

import (
	_ "embed"
	"encoding/json"
	"strings"
)

// Sightwell fork: product-name branding for the desktop UI.
//
// brand.json is the single source: this file applies it to the Go bundle
// (tray, notifications, window titles) and frontend/src/lib/i18n.ts applies
// it to the React strings, which the frontend imports at build time. The
// locale files stay upstream's: they change with every NetBird release and
// would otherwise conflict on each sync.
//
// Only keys where "NetBird" names *this app* are listed. Where it names the
// technology underneath (the NetBird service/daemon, NetBird IP, NetBird
// Cloud, NetBird servers and support, setup keys, the copyright line) the
// text stays as it is, so it remains true.

//go:embed brand.json
var brandJSON []byte

type brandConfig struct {
	AppName     string      `json:"appName"`
	ProductName string      `json:"productName"`
	Replace     [][2]string `json:"replace"`
	Keys        []string    `json:"keys"`
	Features    struct {
		NetBirdCloud bool `json:"netbirdCloud"`
	} `json:"features"`
	// SelfHostedMessages replace keys that offer NetBird Cloud, per locale,
	// when features.netbirdCloud is false.
	SelfHostedMessages map[string]map[string]string `json:"selfHostedMessages"`
}

var (
	// AppName is the product name shown in the menu bar, windows and notifications.
	AppName string
	// ProductName is the full product name used on the About page.
	ProductName string

	brandedKeys   map[string]bool
	brandReplacer *strings.Replacer
	// selfHostedMessages is nil when NetBird Cloud is offered.
	selfHostedMessages map[string]map[string]string
)

func init() {
	var c brandConfig
	if err := json.Unmarshal(brandJSON, &c); err != nil {
		panic("i18n: invalid brand.json: " + err.Error())
	}
	AppName, ProductName = c.AppName, c.ProductName
	brandedKeys = make(map[string]bool, len(c.Keys))
	for _, k := range c.Keys {
		brandedKeys[k] = true
	}
	pairs := make([]string, 0, 2*len(c.Replace))
	for _, r := range c.Replace { // longest first: strings.Replacer compares in argument order
		pairs = append(pairs, r[0], r[1])
	}
	brandReplacer = strings.NewReplacer(pairs...)
	if !c.Features.NetBirdCloud {
		selfHostedMessages = c.SelfHostedMessages
	}
}

// applySelfHostedMessages replaces, in one locale's bundle, the texts that
// offer NetBird Cloud when this build is self-hosted only.
func applySelfHostedMessages(code LanguageCode, bundle map[string]string) {
	for key, byLocale := range selfHostedMessages {
		if msg, ok := byLocale[string(code)]; ok {
			bundle[key] = msg
		}
	}
}

// brandMessage applies the product name to one bundle entry.
func brandMessage(key, message string) string {
	if !brandedKeys[key] {
		return message
	}
	return brandReplacer.Replace(message)
}
