package hostid

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sightwell's machine identity must differ from NetBird's everywhere, or the
// two clients collide when installed side by side.
func TestDistinctFromNetBird(t *testing.T) {
	netbird := map[string]string{
		"Name":            "netbird",
		"TitleName":       "Netbird",
		"DaemonSocket":    "/var/run/netbird.sock",
		"SocketDir":       "/var/run/netbird",
		"ConfigDir":       "/etc/netbird/",
		"LogDir":          "/var/log/netbird/",
		"StateDir":        "/var/lib/netbird",
		"MDMDomain":       "io.netbird.client",
		"UIUniqueID":      "io.netbird.ui",
		"PkgID":           "io.netbird.client",
		"AppPath":         "/Applications/NetBird.app",
		"UIProcess":       "netbird-ui",
		"InterfaceDarwin": "utun100",
		"InterfaceOther":  "wt0",
		"SCKeyPrefix":     "NetBird",
	}
	ours := map[string]string{
		"Name": Name, "TitleName": TitleName, "DaemonSocket": DaemonSocket, "SocketDir": SocketDir,
		"ConfigDir": ConfigDir, "LogDir": LogDir, "StateDir": StateDir, "MDMDomain": MDMDomain,
		"UIUniqueID": UIUniqueID, "PkgID": PkgID, "AppPath": AppPath, "UIProcess": UIProcess,
		"InterfaceDarwin": InterfaceDarwin, "InterfaceOther": InterfaceOther, "SCKeyPrefix": SCKeyPrefix,
	}
	for k, nb := range netbird {
		if strings.EqualFold(ours[k], nb) {
			t.Errorf("%s = %q collides with NetBird", k, ours[k])
		}
	}
	if WgPort == 51820 {
		t.Error("WgPort collides with NetBird's 51820")
	}
	if !strings.HasPrefix(InterfaceDarwin, "utun") || InterfaceDarwin == "utun" {
		t.Errorf("InterfaceDarwin %q must be an explicit utunN unit", InterfaceDarwin)
	}
}

// netbirdHostLiterals are NetBird's machine-level names. Client code must
// use this package instead, so an upstream sync that adds a new use shows up
// here rather than as a silent collision on a customer's machine.
var netbirdHostLiterals = []string{
	`/var/run/netbird`, `/etc/netbird`, `/var/lib/netbird`, `/var/log/netbird`,
	`io.netbird.client`, `io.netbird.ui`, `"netbird-ui"`, `npipe://netbird"`,
	`99-netbird`, `/Applications/NetBird.app`, `Service/NetBird-`, `Policies\NetBird`,
	`"utun100"`, `netbirdio/tap/`,
}

// knownOpen lists files where a NetBird literal is still expected: platforms
// whose side-by-side work is not done yet (see docs/sightwell/SIDE-BY-SIDE.md).
var knownOpen = map[string]bool{
	"internal/updater/installer/installer_run_darwin.go": true, // Homebrew branch, unreachable (guarded)
}

func TestNoNetBirdHostLiterals(t *testing.T) {
	root := ".."
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "bindings", "hostid", "frontend", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if knownOpen[filepath.ToSlash(rel)] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(data), "\n") {
			code := strings.TrimSpace(line)
			if strings.HasPrefix(code, "//") {
				continue
			}
			if idx := strings.Index(code, " //"); idx >= 0 {
				code = code[:idx]
			}
			for _, lit := range netbirdHostLiterals {
				if strings.Contains(code, lit) {
					t.Errorf("%s:%d uses NetBird's %s; use client/hostid", rel, i+1, lit)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
