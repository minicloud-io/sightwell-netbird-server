//go:build !android && !ios && !freebsd && !js

package main

import (
	"bytes"
	_ "embed"
)

// Sightwell fork: on macOS the menu-bar icon is a monochrome template while
// disconnected (and while connecting, failing or waiting for login) and the
// full-colour Sightwell mark while connected. Switching back to a colour icon
// after a template needs the patched Wails (setIcon clears the template
// flag); see docs/sightwell/CLIENT.md.

//go:embed assets/sightwell-menubar-connected.png
var iconConnectedMacOSColour []byte

//go:embed assets/sightwell-menubar-update-connected.png
var iconUpdateConnectedMacOSColour []byte

// macOSColourIcon returns the colour icon that replaces the given macOS
// template icon, or nil when the template should be shown.
func macOSColourIcon(template []byte) []byte {
	switch {
	case bytes.Equal(template, iconConnectedMacOS):
		return iconConnectedMacOSColour
	case bytes.Equal(template, iconUpdateConnectedMacOS):
		return iconUpdateConnectedMacOSColour
	}
	return nil
}
