# Sightwell desktop client (macOS first)

Branch `feat/client-branding` (on `feat/idp-theme`, upstream `v0.81.4`)
rebrands the NetBird desktop UI, `client/ui` (Wails v3, Go + React), as
**Sightwell**. Phase 1 is the visual rebrand and a local, unsigned macOS
build. Signing, notarisation, the installer and the daemon are phase 2.

## Decisions taken (reversible)

| | Value | Why |
| --- | --- | --- |
| Name in menu bar, Finder, windows | **Sightwell** | "Sightwell Connectivity" is too long for the menu bar; the About page uses the full name |
| Bundle identifier | `ai.sightwell.connectivity` | reverse of sightwell.ai; makes it a distinct app to macOS (notifications, login item, permissions) |
| App bundle | `Sightwell.app`, binary still `netbird-ui` | installer scripts and launchd jobs refer to the binary name |
| Single-instance ID | `io.netbird.ui` (unchanged) | a NetBird UI and the Sightwell UI must never drive the same daemon at once |
| MDM domain, installer receipt | `io.netbird.client` (unchanged) | both belong to the daemon, which is NetBird's in phase 1 |

## What changed

| Surface | Files | Change |
| --- | --- | --- |
| Menu bar and tray icons | `client/ui/assets/netbird-systemtray-*.png` (7 states × light, dark, macOS template, mono, mono-dark) | the Sightwell sight-glass mark with upstream's badge language: check (connected), dots (connecting), cross (error, needs login), arrow (update), none (disconnected). Template variants inset the horizon so the mark stays readable as one colour at 18 pt |
| macOS menu bar while connected | `client/ui/assets/sightwell-menubar-connected.png`, `-update-connected.png`, `client/ui/icons_sightwell.go` (+ test), 3 lines in `tray_icon.go` | full-colour Sightwell mark (ink ring, white glass, orange horizon, hairline paper rim for dark and tinted bars), no badge; with an update waiting, a small orange arrow. All other states keep the monochrome template. Preview: `screenshots/client/menu-bar-connected.png` |
| Wails (UI toolkit) | patched fork `sightwell-wails`, branch `sightwell/tray-template-toggle` on NetBird's pinned commit `4e087a8bfd6d` | NetBird's Wails never clears the macOS "template" flag once a template icon was set, so a later colour icon would render as a silhouette. The patch clears it in `setIcon` and sets the flag explicitly in the native code (2 files, 6 lines). Until the fork is published and `go.mod`'s replace points at it, local builds use an uncommitted `go.work` |
| Menu row icons | `netbird-menu-16/24.png`, `netbird-menu-about-18.png`, `netbird.png` | neutral ring, open glass, orange horizon, readable on light and dark menus |
| Status dots | `netbird-menu-dot-*` | unchanged: status colours, not brand |
| App icon | `build/appicon.png` (1024, Apple grid), `build/appicon.icon` (Icon Composer: ink fill + mark layer) | `wails3 generate icons` derives `icons.icns`, `Assets.car` and the Windows `icon.ico` |
| Generator | `client/ui/build/sightwell/icons.py` | draws every icon above from the mark's geometry (Pillow, 4x supersampling); rerun after any change |
| Strings, all 13 locales | `client/ui/i18n/brand.json` (single source), applied by `i18n/brand.go` (+ test, one line in `bundle.go`) for the tray, notifications and window titles, and by `frontend/src/lib/i18n.ts` for the React UI, which imports the locale files at build time | 36 keys where "NetBird" names *this app* get "Sightwell" (and "Sightwell Connectivity" for the client name) at load time; locale files untouched. Technology names stay: the NetBird service/daemon, NetBird IP, NetBird Cloud, NetBird servers and support, setup keys, the copyright line. `TestBrandedKeysExist` fails if an upstream sync renames a key |
| App and window names | `client/ui/main.go` | `Name`, `Description`, main window title from `i18n.AppName` |
| Bundle identity | `build/config.yml`, `build/darwin/Info.plist`, `Info.dev.plist` | name, identifier, company, copyright line ("Built on NetBird, Copyright NetBird GmbH and contributors (BSD-3-Clause)") |
| App bundle name | `client/ui/Taskfile.yml` (`BUNDLE_NAME`), `build/darwin/Taskfile.yml` | `bin/Sightwell.app` |
| Frontend palette | `frontend/tailwind.config.ts`, `frontend/src/globals.css`, `frontend/index.html` | signal-orange ramp under the `netbird` token, ink (dark) and paper (light) grey ramps under `nb-gray`, window background |
| Primary button | `frontend/src/components/buttons/Button.tsx` | ink text on orange (7.6:1); white was 2.9:1 |
| Logos | `frontend/src/assets/logos/sightwell*.svg`, `MainConnectionStatusSwitch.tsx`, `SettingsAbout.tsx`, `SettingsAccent.tsx` | Sightwell lockup on the main and About screens; the accent effect traces the mark. The "NetBird Cloud" option keeps the bird, because it names NetBird's service |
| Welcome screenshot (macOS) | `frontend/src/assets/img/tray-darwin.png` | menu-bar icon and the "Open …" / "Quit …" rows replaced on the original screenshot, matched to SF 13 pt at weight 550 |
| Self-hosted only | `i18n/brand.json` (`features.netbirdCloud: false`, `selfHostedMessages`), `frontend/src/hooks/useManagementUrl.ts` (`CLOUD_MANAGEMENT_ENABLED`, `defaultManagementMode`), `ManagementServerSwitch.tsx`, `WelcomeStepManagement.tsx`, `ProfileCreationModal.tsx` | the Cloud / Self-hosted switch is gone from the welcome step, Settings › General and the new-profile dialog; every form starts self-hosted and asks for the server address. A configured NetBird Cloud URL is still shown truthfully in Settings. The three help texts that offered Cloud get new wording in all 13 locales (new translations, reusing each locale's terms; worth a native-speaker review). Set `netbirdCloud` to `true` to restore upstream's behaviour and texts |
| Update sources | `version/update.go`, `version/url.go`, `version/url_darwin.go` | latest version from `…/releases/latest/download/version` of this fork; download links to the fork's releases page until signed packages exist |
| Release asset | `.github/workflows/sightwell-server-image.yml` | each release gets a plain-text `version` file for the check above |

## Known limits of phase 1

- **The daemon is still NetBird's.** The UI drives whatever `netbird`
  service is installed. Update prompts in the UI come from that daemon,
  which checks NetBird's package server and, when the management server
  enables auto-update, downloads and installs NetBird's signed package
  (`client/internal/updater`, keys from `publickeys.netbird.io`). With
  NetBird's daemon underneath, users can therefore still be offered NetBird
  updates. Phase 2 ships the daemon from this fork and points the updater at
  this fork's artifacts and signing keys.
- **Self-hosted only, but not preset.** NetBird Cloud is not offered (see
  below), so first run asks for the management server address. Presetting a
  customer's address in the installer is phase 2. The daemon's own default
  is still NetBird Cloud; the UI does not prefill it.
- **Windows and Linux** get the new tray and app icons automatically (shared
  PNGs), but their welcome screenshots (`tray-windows.png`, `tray-linux.png`)
  still show NetBird, and their packaging is untouched.
- The devtools console banner (`frontend/src/lib/welcome.ts`) still prints
  NetBird's ASCII art; it is not visible to users.

## Build locally (unsigned)

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3   # inside this module: uses the go.mod replace (NetBird's Wails fork)
git restore go.sum                                   # the install above touches it
cd client/ui && PATH="$HOME/go/bin:$PATH" task package
open bin/Sightwell.app
```

Requires Xcode (with `actool` for `Assets.car`), Go per `go.mod`, Node and
pnpm. The bundle is ad-hoc signed by the task; it runs on the machine that
built it but cannot be distributed (Gatekeeper). It drives the locally
installed `netbird` service; with NetBird's own UI running, the Sightwell UI
hands off to it (shared single-instance ID) and exits.

Regenerate icons after changing the mark: `python3 client/ui/build/sightwell/icons.py`.

## Verification (2026-10-25)

| Check | Result |
| --- | --- |
| `task package` (bindings, frontend, icons incl. `Assets.car` via `actool`, Go, bundle) | `bin/Sightwell.app`, `CFBundleIdentifier` `ai.sightwell.connectivity`, display name Sightwell, ad-hoc signed |
| `go test ./client/ui/i18n/ ./version/` | pass, including `TestBrandedKeysExist` and branding across every shipped locale |
| frontend `tsc`, `eslint`, `prettier` on changed files | clean (`tailwind.config.ts` was already unformatted upstream and is left as is) |
| App launched on this Mac | runs; isolated test copy (separate instance ID, scratch home, non-existent daemon socket) so the installed NetBird UI and daemon were not touched |
| Windows (welcome, daemon overlay) in dark and light | captured from a Wails server-mode build of the same code: `screenshots/client/welcome-*.png`, `daemon-unavailable-dark.png` |
| Icons | rendered sheets: `screenshots/client/tray-icons.png` (upstream: `tray-icons-upstream.png`), `menu-bar-icons.png` (template icons at 2x and 1x on light and dark bars), `app-and-menu-icons.png` |

Not verified: the real macOS menu bar and the connected main window with the
logo. This session had no Screen Recording permission, and showing the
connected view needs a daemon; the test copy was deliberately kept away from
the installed one. Check both by running `bin/Sightwell.app` after quitting
NetBird's UI.

`client/ui/build/darwin/Assets.car` is generated by the build (from
`build/appicon.icon`) and, as upstream, not committed.
