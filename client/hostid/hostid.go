// Package hostid holds the machine-level identity of this client build
// (Sightwell fork).
//
// Sightwell must run on the same machine as an unmodified NetBird client, at
// the same time, without either one touching the other's service, socket,
// files, network interface or managed settings. Every host-level name that
// upstream spells "netbird" (or "Netbird", "io.netbird.*") is defined here
// instead, and the upstream call sites refer to these constants, so an
// upstream sync shows each one as a one-line conflict rather than a silent
// collision.
//
// User-visible product naming lives in client/ui/i18n/brand.json; this
// package is only about what the operating system sees.
package hostid

const (
	// Name is the lower-case machine name: service (Linux, macOS), CLI
	// binary (/usr/local/bin/sightwell), socket and directory names.
	Name = "sightwell"
	// TitleName is the Windows service name and ProgramData folder.
	TitleName = "Sightwell"

	// DaemonSocket is the daemon's default Unix control socket.
	DaemonSocket = "/var/run/" + Name + ".sock"
	// DaemonAddr is DaemonSocket as a gRPC dial address.
	DaemonAddr = "unix://" + DaemonSocket
	// SocketDir holds per-instance sockets (Linux systemd template units).
	SocketDir = "/var/run/" + Name

	// ConfigDir, LogDir and StateDir are the Unix daemon directories.
	ConfigDir = "/etc/" + Name + "/"
	LogDir    = "/var/log/" + Name + "/"
	StateDir  = "/var/lib/" + Name
	// BSDStateDir replaces /var/db/netbird on the BSDs.
	BSDStateDir = "/var/db/" + Name

	// UserConfigDirName is the per-user folder under os.UserConfigDir()
	// (macOS: ~/Library/Application Support/sightwell).
	UserConfigDirName = Name

	// BundleID is the macOS app bundle identifier, also the managed
	// preferences (MDM) domain the daemon and UI read policy from.
	BundleID = "ai.sightwell.connectivity"
	// MDMDomain replaces io.netbird.client.
	MDMDomain = BundleID
	// UIUniqueID is the desktop UI's single-instance lock; distinct from
	// NetBird's io.netbird.ui so both UIs can run at once.
	UIUniqueID = BundleID + ".ui"
	// PkgID is the macOS installer receipt identifier.
	PkgID = BundleID + ".pkg"
	// AppPath is where the macOS package installs the app bundle.
	AppPath = "/Applications/Sightwell.app"
	// UIProcess is the desktop UI's executable name (CFBundleExecutable).
	UIProcess = Name + "-ui"

	// InterfaceDarwin and InterfaceOther are the default WireGuard interface
	// names (NetBird: utun100, wt0). macOS opens the exact utun unit and fails
	// if it is taken; Linux kernel mode deletes an existing interface of the
	// same name. "wt1" keeps NetBird's "wt" ICE blacklist prefix.
	InterfaceDarwin = "utun110"
	InterfaceOther  = "wt1"
	// WgPort is the default WireGuard listen port for new configs (NetBird:
	// 51820). Peers still assume 51820 for remotes that don't advertise one.
	WgPort = 51821

	// SCKeyPrefix prefixes this daemon's macOS SystemConfiguration DNS keys
	// (State:/Network/Service/<prefix>-.../DNS) and Windows NRPT rules.
	SCKeyPrefix = TitleName

	// ReleasesURL is where this fork publishes releases and installers.
	ReleasesURL = "https://github.com/minicloud-io/sightwell-netbird-server/releases"
)
