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
	assertDistinct(t, "macOS template", map[string][]byte{
		"connected": iconConnectedMacOS, "disconnected": iconDisconnectedMacOS,
		"connecting": iconConnectingMacOS, "error": iconErrorMacOS, "needs-login": iconNeedsLoginMacOS,
		"update-connected": iconUpdateConnectedMacOS, "update-disconnected": iconUpdateDisconnectedMacOS,
	})
}

// Windows and Linux show the state by the level alone (no badges), so every
// state needs its own icon in each set.
func TestGaugeTrayIconsDistinct(t *testing.T) {
	assertDistinct(t, "Windows light", map[string][]byte{
		"connected": iconConnected, "disconnected": iconDisconnected, "connecting": iconConnecting,
		"error": iconError, "needs-login": iconNeedsLogin,
		"update-connected": iconUpdateConnected, "update-disconnected": iconUpdateDisconnected,
	})
	assertDistinct(t, "Windows dark", map[string][]byte{
		"connected": iconConnectedDark, "disconnected": iconDisconnectedDark, "connecting": iconConnectingDark,
		"error": iconErrorDark, "needs-login": iconNeedsLoginDark,
		"update-connected": iconUpdateConnectedDark, "update-disconnected": iconUpdateDisconnectedDark,
	})
	assertDistinct(t, "Linux mono", map[string][]byte{
		"connected": iconConnectedMono, "disconnected": iconDisconnectedMono, "connecting": iconConnectingMono,
		"error": iconErrorMono, "needs-login": iconNeedsLoginMono,
		"update-connected": iconUpdateConnectedMono, "update-disconnected": iconUpdateDisconnectedMono,
	})
	assertDistinct(t, "Linux mono-dark", map[string][]byte{
		"connected": iconConnectedMonoDark, "disconnected": iconDisconnectedMonoDark, "connecting": iconConnectingMonoDark,
		"error": iconErrorMonoDark, "needs-login": iconNeedsLoginMonoDark,
		"update-connected": iconUpdateConnectedMonoDark, "update-disconnected": iconUpdateDisconnectedMonoDark,
	})
}

func assertDistinct(t *testing.T, set string, icons map[string][]byte) {
	t.Helper()
	seen := map[string]string{}
	for name, b := range icons {
		key := string(b)
		if other, ok := seen[key]; ok {
			t.Errorf("%s icons %q and %q are identical", set, name, other)
		}
		seen[key] = name
	}
}
