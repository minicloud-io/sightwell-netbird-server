package jobexec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/netbirdio/netbird/client/internal/debug"
)

// Sightwell fork: a remote bundle job without an upload URL fails before a
// bundle is generated, instead of uploading to NetBird's service.
func TestBundleJobWithoutUploadURL(t *testing.T) {
	_, err := NewExecutor().BundleJob(context.Background(), debug.GeneratorDependencies{}, debug.BundleConfig{}, 0, "https://mgm.example", "")
	require.ErrorIs(t, err, debug.ErrNoUploadService)
}
