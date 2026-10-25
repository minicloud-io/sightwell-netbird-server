package installer

import (
	"context"
	"os/exec"

	"github.com/netbirdio/netbird/client/hostid"
)

var (
	TypeHomebrew = Type{name: "Homebrew", downloadable: false}
	TypePKG      = Type{name: "pkg", downloadable: true}
)

func TypeOfInstaller(ctx context.Context) Type {
	// Sightwell fork: our own package receipt, never NetBird's io.netbird.client.
	cmd := exec.CommandContext(ctx, "pkgutil", "--pkg-info", hostid.PkgID)
	_, err := cmd.Output()
	if err != nil && cmd.ProcessState.ExitCode() == 1 {
		// Not installed using pkg file, thus installed using Homebrew

		return TypeHomebrew
	}
	return TypePKG
}
