//go:build !js

package metrics

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Sightwell fork: management can't start a push to NetBird's ingest server.
func TestUpdatePushFromMgm_ManagementCannotEnablePush(t *testing.T) {
	t.Setenv(EnvMetricsPushEnabled, "")
	os.Unsetenv(EnvMetricsPushEnabled)

	c := NewClientMetrics(AgentInfo{})
	c.UpdatePushFromMgm(context.Background(), true)
	defer c.StopPush()

	assert.Nil(t, c.push.Load(), "push must not start when only management enables it")
}

func TestUpdatePushFromMgm_ManagementDisableIsNoOpWithoutPush(t *testing.T) {
	t.Setenv(EnvMetricsPushEnabled, "")
	os.Unsetenv(EnvMetricsPushEnabled)

	c := NewClientMetrics(AgentInfo{})
	c.UpdatePushFromMgm(context.Background(), false)

	assert.Nil(t, c.push.Load())
}
