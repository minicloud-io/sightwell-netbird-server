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
