# Sightwell beside NetBird (phase 2)

Branch `feat/client-side-by-side` (on `feat/client-branding`).

**Requirement (Richard, 2026-10-25):** anyone must be able to run NetBird
and Sightwell on the same machine, desktop and mobile, at the same time.
Sightwell never stops, replaces or reconfigures a NetBird install; an
enterprise customer may already run NetBird.

On phones the operating system allows one active VPN at a time: both apps
can be installed side by side, only one can be connected. The mobile apps
are separate projects and not part of this branch.

## Machine identity

Every host-level name the client used to spell "netbird" is defined in
`client/hostid/hostid.go`; upstream call sites refer to it (marked
`Sightwell fork`). `client/hostid/hostid_test.go` fails if a value equals
NetBird's or if client code (outside `knownOpen`) uses a NetBird host
literal again, which is how an upstream sync shows a new collision.

| | NetBird | Sightwell |
| --- | --- | --- |
| Service (launchd label, systemd unit) | `netbird` (Windows `Netbird`) | `sightwell` (Windows `Sightwell`) |
| CLI | `/usr/local/bin/netbird` | `/usr/local/bin/sightwell` |
| Daemon socket | `/var/run/netbird.sock`, `/var/run/netbird/` | `/var/run/sightwell.sock`, `/var/run/sightwell/` |
| JSON socket (opt-in) | `/var/run/netbird-http.sock` | `/var/run/sightwell-http.sock` |
| Windows pipe | `npipe://netbird` | `npipe://sightwell` |
| Config, state, log dirs | `/etc/netbird`, `/var/lib/netbird`, `/var/log/netbird`, `%PROGRAMDATA%\Netbird` | `/etc/sightwell`, `/var/lib/sightwell`, `/var/log/sightwell`, `%PROGRAMDATA%\Sightwell` |
| Legacy migration source | `/etc/netbird`, `/etc/wiretrustee` | Sightwell's own dirs only (never reads or moves NetBird's files) |
| Per-user folder (profiles, UI preferences, UI log) | `~/Library/Application Support/netbird` | `~/Library/Application Support/sightwell` |
| WireGuard interface | `utun100` (macOS), `wt0` | `utun110`, `wt1` |
| WireGuard port (new configs) | 51820 | 51821 (51820 is still assumed for remote peers that don't advertise a port) |
| macOS DNS keys | `State:/Network/Service/NetBird-*/DNS` | `State:/Network/Service/Sightwell-*/DNS` |
| macOS route tag (flushed at startup) | `RTF_PROTO1` | `RTF_PROTO2` (`NB_ROUTE_PROTO_FLAG=1`, `2` or `3` still overrides) |
| Managed settings (MDM) | `io.netbird.client`, `HKLM\Software\Policies\NetBird` | `ai.sightwell.connectivity`, `HKLM\Software\Policies\Sightwell` |
| SSH client config | `/etc/ssh/ssh_config.d/99-netbird.conf` | `99-sightwell.conf` |
| Client certificate store | `/etc/netbird/certs` | `/etc/sightwell/certs` |
| App, UI process, single-instance ID | `NetBird.app`, `netbird-ui`, `io.netbird.ui` | `Sightwell.app`, `sightwell-ui`, `ai.sightwell.connectivity.ui` |
| Installer receipt | `io.netbird.client` | `ai.sightwell.connectivity.pkg` |

The updater (`client/internal/updater/installer`, macOS) now checks our
receipt, downloads `Sightwell-<version>.pkg` from this fork's releases,
stops and starts only Sightwell, and refuses the Homebrew path (which would
upgrade NetBird's formula). Its signing-key base is
`https://sightwell.ai/connectivity/artifact-signatures`, not published yet,
and `reposign/certs` still embeds NetBird's root, so every update fails
verification until MINICLOUD FZCO's keys exist: updates fail closed.

## Installer (macOS, unsigned)

```bash
release_files/sightwell-macos-pkg.sh 0.81.4-sightwell.7-dev
```

Builds `dist/sightwell-macos/Sightwell-<version>-unsigned.pkg`:
`/Applications/Sightwell.app` with `sightwell-ui` (UI) and `sightwell`
(daemon, CLI), both universal, `Resources/licenses/` (NetBird's BSD-3-Clause
licence, Go's terms, third-party licences) and `Resources/uninstall.sh`.
Not relocatable. Scripts: `release_files/sightwell_darwin_pkg/`; they only
quit Sightwell's UI (by bundle id and process name) and stop Sightwell's
service, and drop upstream's refusal to install next to a Homebrew NetBird.
The workflow `.github/workflows/sightwell-client-macos.yml` (manual,
SHA-pinned) builds the same package as an artifact; nothing is published.

Remove with `sudo /Applications/Sightwell.app/Contents/Resources/uninstall.sh`
(Sightwell's service, command, app, dirs and receipt only).

## Not solved by names

- **Overlay address range.** Each management account gets a random /16
  inside 100.64.0.0/10. If a NetBird account and a Sightwell account land on
  overlapping ranges, the second tunnel cannot come up on macOS (the overlay
  route already exists). Two random picks collide 1 time in 64. No code
  change is needed: management accounts already accept a custom
  `network_range` (account settings API, `parseAndValidateNetworkRange`).
  Set Sightwell accounts to a fixed private range outside 100.64.0.0/10
  that the customer's LANs don't use; pick it per deployment.
  Both clients signed in to the same account always share its range, so
  that combination can't carry traffic on both tunnels; on the development
  Mac (2026-10-26, both on SAWGRASS-LNG) Sightwell reached management and
  signal, but `100.69/16` stayed on NetBird's `utun100`.
- **DNS for everything.** Two clients that both route all DNS (a
  primary/catch-all nameserver) fight over the default resolver. Match
  domains coexist if they don't overlap; Sightwell's DNS domain must differ
  from the customer's NetBird domain.
- **Exit nodes / default route.** Only one client can carry the default
  route at a time.
- **Scoped default reuse (macOS).** When NetBird already installed the
  scoped default route on the physical interface, Sightwell reuses it; if
  NetBird then disconnects it removes that route. Needs a real two-client
  test before deciding on a fix.

## Not done yet (other platforms)

Linux and Windows still collide at the network level even with the names
above: Linux route table 7120 and rule priorities, fwmark base and conntrack
zone, iptables chain names, nftables accept-rule tags in the shared filter
chains, ipsets, `/etc/resolv.conf` header and backup; Windows NRPT prefix
`NetBird-`, firewall rule `Netbird`, the fixed wintun adapter GUID,
`Global\` events, the MSI, and the Windows updater (still NetBird's MSI).
Linux and Windows packaging are untouched. Cosmetic: CLI help texts still
say `netbird`.

## Test plan (needs a clean Mac or VM, never the development Mac)

1. Install NetBird's current pkg, connect to a NetBird account.
2. Install the Sightwell pkg; NetBird's UI and service keep running.
3. Connect Sightwell to a Sightwell management server with a
   non-overlapping range; both tunnels up (`ifconfig utun100 utun110`),
   peers reachable on both, `scutil --dns` shows both resolvers.
4. Disconnect, reconnect and restart each in turn; the other stays up.
5. `sudo …/uninstall.sh`; NetBird untouched.
