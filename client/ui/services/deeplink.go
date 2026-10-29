//go:build !android && !ios && !freebsd && !js

package services

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"unicode"

	log "github.com/sirupsen/logrus"
)

// Sightwell fork: one-click tenant set-up. A tenant's dashboard links to
//
//	sightwell://add-profile?name=<display name>&url=https://<tenant host>
//
// and the app offers to add that tenant as a profile, select it and sign in.
// The link comes from a web page, so everything in it is untrusted: it is
// parsed here, and nothing changes until the user confirms in the main window.
// The daemon applies its own gates to the management URL (MDM, the
// update-settings kill switch, and caller identity while SSH is enabled), so
// the link can do nothing the Add Profile form cannot.

const (
	// LinkScheme is the URL scheme the app registers with the OS.
	LinkScheme = "sightwell"
	// linkActionAddProfile is the only action the app accepts.
	linkActionAddProfile = "add-profile"

	// EventLinkReceived tells the frontend a link is waiting; it fetches it
	// with DeepLink.TakePending, so a link that arrives before the window
	// mounts is not lost.
	EventLinkReceived = "sightwell:link:received"

	// maxLinkLen bounds what is parsed at all; real links are well under it.
	maxLinkLen = 2048
	// maxLinkNameLen matches the Add Profile form's name limit.
	maxLinkNameLen = 128
)

// LinkRefusal names why a link was refused. The frontend maps each to a
// translated message.
type LinkRefusal string

const (
	LinkRefusedInvalid       LinkRefusal = "invalid"
	LinkRefusedUnknownAction LinkRefusal = "unknown-action"
	LinkRefusedNotHTTPS      LinkRefusal = "not-https"
	LinkRefusedMissingHost   LinkRefusal = "missing-host"
)

// LinkError is returned for a refused link.
type LinkError struct {
	Refusal LinkRefusal
	Detail  string
}

func (e *LinkError) Error() string {
	return fmt.Sprintf("link refused (%s): %s", e.Refusal, e.Detail)
}

func refuse(r LinkRefusal, format string, args ...any) error {
	return &LinkError{Refusal: r, Detail: fmt.Sprintf(format, args...)}
}

// AddProfileLink is a parsed, validated add-profile link.
type AddProfileLink struct {
	// Name is the profile's display name: the link's name, else the host.
	Name string `json:"name"`
	// ManagementURL is https://host[:port], with nothing else.
	ManagementURL string `json:"managementUrl"`
	// Host is shown in the confirmation so the user sees where the profile points.
	Host string `json:"host"`
}

// ParseLink parses a sightwell:// link. Only add-profile with an https
// management URL on a DNS name is accepted.
func ParseLink(raw string) (AddProfileLink, error) {
	if len(raw) > maxLinkLen {
		return AddProfileLink{}, refuse(LinkRefusedInvalid, "longer than %d bytes", maxLinkLen)
	}

	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return AddProfileLink{}, refuse(LinkRefusedInvalid, "parse: %v", err)
	}
	if u.Scheme != LinkScheme {
		return AddProfileLink{}, refuse(LinkRefusedInvalid, "scheme %q", u.Scheme)
	}
	// sightwell://add-profile?… puts the action in the host. Accept a bare
	// trailing slash, which some browsers add.
	if u.Opaque != "" || u.User != nil || (u.Path != "" && u.Path != "/") {
		return AddProfileLink{}, refuse(LinkRefusedInvalid, "unexpected link form")
	}
	if action := strings.ToLower(u.Host); action != linkActionAddProfile {
		return AddProfileLink{}, refuse(LinkRefusedUnknownAction, "action %q", action)
	}

	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return AddProfileLink{}, refuse(LinkRefusedInvalid, "query: %v", err)
	}
	// A repeated parameter is ambiguous: refuse rather than pick one.
	for key, values := range q {
		if len(values) > 1 {
			return AddProfileLink{}, refuse(LinkRefusedInvalid, "parameter %q repeated", key)
		}
	}

	managementURL, host, err := parseTenantURL(q.Get("url"))
	if err != nil {
		return AddProfileLink{}, err
	}

	name := cleanLinkName(q.Get("name"))
	if name == "" {
		name = host
	}

	return AddProfileLink{Name: name, ManagementURL: managementURL, Host: host}, nil
}

// parseTenantURL accepts https://host[:port] with an optional trailing slash
// and returns it normalised, plus the host (and non-default port) for display.
func parseTenantURL(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", refuse(LinkRefusedMissingHost, "no url parameter")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", "", refuse(LinkRefusedInvalid, "url: %v", err)
	}
	if u.Scheme != "https" {
		return "", "", refuse(LinkRefusedNotHTTPS, "url scheme %q", u.Scheme)
	}
	if u.Opaque != "" || u.User != nil || (u.Path != "" && u.Path != "/") ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", "", refuse(LinkRefusedInvalid, "url has more than a host")
	}

	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "" {
		return "", "", refuse(LinkRefusedMissingHost, "url has no host")
	}
	if !isDNSName(host) {
		return "", "", refuse(LinkRefusedInvalid, "host %q is not a DNS name", host)
	}

	hostPort := host
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", "", refuse(LinkRefusedInvalid, "port %q", port)
		}
		if n != 443 {
			hostPort = net.JoinHostPort(host, strconv.Itoa(n))
		}
	}

	return "https://" + hostPort, hostPort, nil
}

// isDNSName reports whether host is a name with at least two labels, each of
// letters, digits and inner hyphens. IP literals and single labels are not
// tenant addresses.
func isDNSName(host string) bool {
	if len(host) > 253 {
		return false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return false
	}
	for _, l := range labels {
		if l == "" || len(l) > 63 || l[0] == '-' || l[len(l)-1] == '-' {
			return false
		}
		for _, c := range l {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return false
			}
		}
	}
	// An all-numeric last label is an IPv4 address, not a name.
	last := labels[len(labels)-1]
	return strings.IndexFunc(last, func(r rune) bool { return r < '0' || r > '9' }) >= 0
}

// cleanLinkName drops control and format characters (which could reorder or
// hide text in the confirmation), collapses whitespace and caps the length.
func cleanLinkName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r):
		default:
			b.WriteRune(r)
		}
	}
	name := strings.Join(strings.Fields(b.String()), " ")
	if runes := []rune(name); len(runes) > maxLinkNameLen {
		name = strings.TrimSpace(string(runes[:maxLinkNameLen]))
	}
	return name
}

// PendingLink is what the frontend receives: a link to confirm, or the reason
// one was refused. Both are empty when nothing is waiting.
type PendingLink struct {
	Link    *AddProfileLink `json:"link"`
	Refusal LinkRefusal     `json:"refusal"`
}

// LinkInbox holds the most recent link until the frontend takes it. A newer
// link replaces an unconfirmed older one: the user acts on the last click.
type LinkInbox struct {
	mu      sync.Mutex
	pending PendingLink
	notify  func(event string)
}

// NewLinkInbox returns an inbox that calls notify (WindowManager.ShowMainAndEmit)
// when a link arrives.
func NewLinkInbox(notify func(event string)) *LinkInbox {
	return &LinkInbox{notify: notify}
}

// Receive parses raw and queues the result for the frontend. Anything that
// is not a sightwell:// link is ignored, since the same entry points carry
// other launch arguments.
func (i *LinkInbox) Receive(raw string) {
	if !IsLink(raw) {
		return
	}

	var pending PendingLink
	link, err := ParseLink(raw)
	if err != nil {
		pending.Refusal = LinkRefusedInvalid
		var le *LinkError
		if errors.As(err, &le) {
			pending.Refusal = le.Refusal
		}
		log.Warnf("deep link refused: %s", pending.Refusal)
		log.Debugf("deep link: %v", err)
	} else {
		log.Info("deep link: add-profile received")
		log.Debugf("deep link: add-profile for %s", link.Host)
		pending.Link = &link
	}

	i.mu.Lock()
	i.pending = pending
	i.mu.Unlock()

	if i.notify != nil {
		i.notify(EventLinkReceived)
	}
}

// Take returns the waiting link and clears it, so each link is offered once
// even when several windows listen.
func (i *LinkInbox) Take() PendingLink {
	i.mu.Lock()
	defer i.mu.Unlock()
	p := i.pending
	i.pending = PendingLink{}
	return p
}

// IsLink reports whether s is a sightwell:// link, so launch arguments can be
// told apart from it.
func IsLink(s string) bool {
	prefix := LinkScheme + "://"
	s = strings.TrimSpace(s)
	return len(s) > len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// FindLink returns the first sightwell:// link in args, e.g. the arguments a
// second instance was started with on Windows and Linux.
func FindLink(args []string) (string, bool) {
	for _, a := range args {
		if IsLink(a) {
			return a, true
		}
	}
	return "", false
}

// DeepLink is the Wails-bound side of LinkInbox: the frontend can take a
// link, not inject one.
type DeepLink struct {
	inbox *LinkInbox
}

func NewDeepLink(inbox *LinkInbox) *DeepLink {
	return &DeepLink{inbox: inbox}
}

// TakePending returns the waiting link or refusal and clears it.
func (d *DeepLink) TakePending() PendingLink {
	return d.inbox.Take()
}
