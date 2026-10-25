//go:build darwin

package configurer

import "github.com/netbirdio/netbird/client/hostid"

// WgInterfaceDefault is the default interface name (Sightwell fork: not
// NetBird's utun100, so both can run at once).
const WgInterfaceDefault = hostid.InterfaceDarwin
