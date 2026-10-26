package types

const (
	// ClientHeader is the header used to identify the client
	ClientHeader = "x-nb-client"
	// ClientHeaderValue is the value of the ClientHeader
	ClientHeaderValue = "netbird"
	// GetURLPath is the path for the GetURL request
	GetURLPath = "/upload-url"

	// DefaultBundleURL is empty in the Sightwell fork: debug bundles never go to
	// NetBird's upload service (upload.debug.netbird.io). An upload needs an
	// explicit URL from the CLI flag, MDM or the management server.
	DefaultBundleURL = ""
)

// GetURLResponse is the response for the GetURL request
type GetURLResponse struct {
	URL string
	Key string
}
