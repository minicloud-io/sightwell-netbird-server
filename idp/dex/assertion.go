package dex

import (
	"sync"

	"github.com/netbirdio/dex/server"
	"github.com/netbirdio/dex/storage"

	"github.com/netbirdio/netbird/idp/dex/oidcassertion"
)

var assertionRegistryMu sync.Mutex

// registerAssertionConnector lets Dex open connectors of type oidc-assertion,
// signing their client assertions with the keys in stor. Dex keeps its
// connector factories in a process-wide map, so the storage registered last
// wins; one embedded Dex per process is the supported setup.
func registerAssertionConnector(stor storage.Storage) {
	assertionRegistryMu.Lock()
	defer assertionRegistryMu.Unlock()
	server.ConnectorsConfig[oidcassertion.Type] = func() server.ConnectorConfig {
		return &oidcassertion.Config{Keys: stor}
	}
}
