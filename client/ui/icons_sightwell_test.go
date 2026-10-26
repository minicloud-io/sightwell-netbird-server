//go:build !android && !ios && !freebsd && !js

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMacOSColourIcon(t *testing.T) {
	assert.Equal(t, iconConnectedMacOSColour, macOSColourIcon(iconConnectedMacOS))
	assert.Equal(t, iconUpdateConnectedMacOSColour, macOSColourIcon(iconUpdateConnectedMacOS))
	for _, template := range [][]byte{iconDisconnectedMacOS, iconConnectingMacOS, iconErrorMacOS, iconNeedsLoginMacOS, iconUpdateDisconnectedMacOS} {
		assert.Nil(t, macOSColourIcon(template), "non-connected states keep the template icon")
	}
}

// The tray tells states apart by icon bytes (colour icon, spinner), so no two
// macOS templates may be identical.
func TestMacOSTemplatesDistinct(t *testing.T) {
	icons := map[string][]byte{
		"connected": iconConnectedMacOS, "disconnected": iconDisconnectedMacOS,
		"connecting": iconConnectingMacOS, "error": iconErrorMacOS, "needs-login": iconNeedsLoginMacOS,
		"update-connected": iconUpdateConnectedMacOS, "update-disconnected": iconUpdateDisconnectedMacOS,
	}
	seen := map[string]string{}
	for name, b := range icons {
		key := string(b)
		if other, ok := seen[key]; ok {
			t.Errorf("macOS template icons %q and %q are identical", name, other)
		}
		seen[key] = name
	}
}
