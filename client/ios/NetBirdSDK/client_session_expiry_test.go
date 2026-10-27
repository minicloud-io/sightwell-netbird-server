//go:build ios

package NetBirdSDK

import (
	"context"
	"testing"
	"time"

	"github.com/netbirdio/netbird/client/internal/peer"
)

// running starts a fake Run on c (the run lifecycle only, no engine) and
// returns the function that ends it.
func running(t *testing.T, c *Client) func() {
	t.Helper()
	_, cancel := context.WithCancel(context.Background())
	done, err := c.startRun(cancel)
	if err != nil {
		t.Fatalf("startRun: %v", err)
	}
	return func() { cancel(); c.finishRun(done) }
}

func TestClientSessionExpiresAt(t *testing.T) {
	deadline := time.Now().Add(18 * time.Hour).Truncate(time.Second)

	t.Run("set", func(t *testing.T) {
		c := &Client{recorder: peer.NewRecorder("")}
		defer running(t, c)()
		c.recorder.SetSessionExpiresAt(deadline)
		if got := c.SessionExpiresAt(); got != deadline.Unix() {
			t.Errorf("SessionExpiresAt() = %d, want %d", got, deadline.Unix())
		}
	})

	t.Run("zero", func(t *testing.T) {
		c := &Client{recorder: peer.NewRecorder("")}
		defer running(t, c)()
		if got := c.SessionExpiresAt(); got != 0 {
			t.Errorf("SessionExpiresAt() = %d, want 0", got)
		}
		c.recorder.SetSessionExpiresAt(deadline)
		c.recorder.SetSessionExpiresAt(time.Time{})
		if got := c.SessionExpiresAt(); got != 0 {
			t.Errorf("SessionExpiresAt() after clearing = %d, want 0", got)
		}
	})

	t.Run("not running", func(t *testing.T) {
		c := &Client{recorder: peer.NewRecorder("")}
		c.recorder.SetSessionExpiresAt(deadline)
		if got := c.SessionExpiresAt(); got != 0 {
			t.Errorf("SessionExpiresAt() before Run = %d, want 0", got)
		}
		running(t, c)()
		if got := c.SessionExpiresAt(); got != 0 {
			t.Errorf("SessionExpiresAt() after Run = %d, want 0", got)
		}
	})

	t.Run("nil", func(t *testing.T) {
		var c *Client
		if got := c.SessionExpiresAt(); got != 0 {
			t.Errorf("nil client: SessionExpiresAt() = %d, want 0", got)
		}
		if got := (&Client{}).SessionExpiresAt(); got != 0 {
			t.Errorf("no recorder: SessionExpiresAt() = %d, want 0", got)
		}
	})
}
