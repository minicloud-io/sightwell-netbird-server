package version

import (
	"os/exec"
	"runtime"
)

const (
	// Sightwell fork: until the signed macOS packages exist (phase 2) both
	// point at the releases page; then at the per-arch .pkg assets.
	urlMacIntel = "https://github.com/minicloud-io/sightwell-netbird-server/releases/latest"
	urlMacM1M2  = "https://github.com/minicloud-io/sightwell-netbird-server/releases/latest"
)

// DownloadUrl return with the proper download link
func DownloadUrl() string {
	cmd := exec.Command("brew", "list --formula | grep -i netbird")
	if err := cmd.Start(); err != nil {
		goto PKGINSTALL
	}

	if err := cmd.Wait(); err == nil {
		return downloadURL
	}

PKGINSTALL:
	switch runtime.GOARCH {
	case "amd64":
		return urlMacIntel
	case "arm64":
		return urlMacM1M2
	default:
		return downloadURL
	}
}
