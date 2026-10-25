//go:build !devartifactsign

package installer

const (
	// Sightwell fork: MINICLOUD FZCO's artifact-signing keys (root, artifact
	// keys, revocation list, per-release signatures under tag/v<version>/).
	// Not published yet: until they are, and until reposign/certs carries
	// MINICLOUD's root, every update fails verification (fails closed).
	DefaultSigningKeysBaseURL = "https://sightwell.ai/connectivity/artifact-signatures"
)
