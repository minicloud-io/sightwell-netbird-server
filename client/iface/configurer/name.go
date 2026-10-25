//go:build linux || windows || freebsd || js || wasip1

package configurer

import "github.com/netbirdio/netbird/client/hostid"

// WgInterfaceDefault is the default interface name (Sightwell fork: not
// NetBird's wt0, so both can run at once).
const WgInterfaceDefault = hostid.InterfaceOther
