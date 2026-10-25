# Sightwell fork of the NetBird server

This repository is `minicloud-io/sightwell-netbird-server`, a fork of
[netbirdio/netbird](https://github.com/netbirdio/netbird) kept for one
purpose: building the `netbird-server` image that Sightwell Connectivity
deploys, with the ability to theme the embedded identity provider's pages.
It is **not** proposed upstream and tracks upstream release tags.

| | |
| --- | --- |
| Upstream baseline | tag `v0.81.4`, commit `7e791d3e2972ddb763bb0e5c3bff4f9e14f70033` (selected 2026-10-25; the running `netbirdio/netbird-server:latest` carries no version label and its binary has no version flag, so the deployed revision is unverified) |
| Branch | `feat/idp-theme` |
| Licence | unchanged: BSD-3-Clause for the repository, AGPL-3.0 for `management/`, `signal/`, `relay/` and `combined/` (see the `LICENSE` files); this change touches `management/` and `combined/`, so builds of it are AGPL-3.0 and their source is this repository at the built commit |

## The change: `server.auth.theme`

Dex, the embedded IdP, can serve its sign-in, sign-out, password, device and
error pages from a directory on disk instead of the files compiled into the
binary (`frontend.dir` in Dex terms; NetBird's wrapper in `idp/dex` already
passes it through). Upstream's management server never exposes that: it
hard-codes `Frontend{Issuer: "NetBird", Theme: "light"}` when it builds the
Dex config. This fork adds the missing plumbing, backwards compatible:

| File | Change |
| --- | --- |
| `management/server/idp/embedded.go` | `EmbeddedIdPConfig.Frontend *EmbeddedIdPFrontend` (`Dir`, `Theme`, `DisplayName`, `LogoURL`); `buildFrontend` maps it to `dex.Frontend` and validates a custom `Dir` at start-up (must exist and contain `templates/header.html`, `footer.html`, `login.html`, `logout.html`, `password.html`, `error.html`, plus the `static/` and `themes/<Theme>/` directories and the `robots.txt` file Dex loads at start), so a broken theme or mount fails on boot with a clear message rather than on the first sign-in |
| `management/server/idp/embedded_frontend_test.go` | tests for defaults, mapping and each validation error |
| `idp/dex/theme_fs.go` (+ test), `idp/dex/provider.go` | the theme directory is loaded into an in-memory filesystem at start, following symlinks and skipping entries whose names start with `..`. Dex lists `templates/` and would otherwise parse kubelet's `..data` symlink as a template, so a ConfigMap or projected volume mounted in any layout works. A changed theme needs a restart |
| `combined/cmd/config.go` | `server.auth.theme` YAML block (`dir`, `name`, `displayName`, `logoURL`) mapped into `Frontend`; a relative `dir` resolves against `dataDir` |
| `combined/cmd/theme_config_test.go` | parsing and mapping tests |
| `combined/config.yaml.example` | documentation of the block |
| `management/server/instance/manager.go` (+ `update_sources_test.go`), `combined/cmd/config.go`, `combined/cmd/root.go` | `server.updates.{managementReleasesURL,dashboardReleasesURL}`: GitHub releases API documents the version check reads "latest" from, so a fork compares against its own releases (go-version orders `v0.81.4-sightwell.4` below `.5` and below `v0.81.5`); empty keeps NetBird's sources |

Nothing changes when the block is absent: the embedded NetBird pages are
served exactly as before.

### Configuration

```yaml
server:
  auth:
    issuer: "https://connectivity.example.internal/oauth2"
    theme:
      dir: /etc/netbird/idp-theme          # templates/, static/, themes/<name>/
      name: sightwell                      # themes/<name>/ (default "light")
      displayName: "Sightwell Connectivity" # page title (default "NetBird")
```

The Sightwell theme itself lives in the dashboard repository,
`minicloud-io/sightwell-netbird-dashboard`, under `deploy/idp-theme/`, with
Kubernetes mounting instructions in its README. The directory replaces Dex's
web directory entirely, so it must carry every template.

## Desktop client

Branch `feat/client-branding` rebrands the macOS desktop UI as Sightwell
(icons, palette, strings in all locales, bundle identity, update sources).
Phase 1, unsigned local builds; see `docs/sightwell/CLIENT.md`.

## Images

Built by `.github/workflows/sightwell-server-image.yml` on tags
`v*-sightwell.*` (native `linux/amd64` and `linux/arm64` builds from
`combined/Dockerfile.multistage`, merged into one manifest; every action
pinned to a commit SHA, and the repository requires that). Pin tag and
manifest digest together:

| Release | Image |
| --- | --- |
| `v0.81.4-sightwell.6` (commit `c0571104`, 2026-10-25): `server.updates` release sources for the version check | `ghcr.io/minicloud-io/sightwell-netbird-server:v0.81.4-sightwell.6@sha256:4ab658e5d133f46256d92d916c11347ba09c6a37fbdd588fa4075f8e6f552161` |
| `v0.81.4-sightwell.5`: tag exists but its tree had a failing unit test; the build was cancelled, no image or release | — |
| `v0.81.4-sightwell.4` (commit `06d6cc22`, 2026-10-25): theme served from an in-memory snapshot, ConfigMap mounts of any layout work | `ghcr.io/minicloud-io/sightwell-netbird-server:v0.81.4-sightwell.4@sha256:8baf1cce26f7b90dd95e29f068dfa17d35615f49b45a54bcddc31aa834999c09` |
| `v0.81.4-sightwell.3` (commit `358c7783`, 2026-10-25): validation also requires `robots.txt`; fails on kubelet's `..data` entries, superseded | `ghcr.io/minicloud-io/sightwell-netbird-server:v0.81.4-sightwell.3@sha256:a1a87bc9193564bf9e912755c4a15ae9ccf6fc1f35e6add56ca61827ef97624d` |
| `v0.81.4-sightwell.2` (commit `16abdec2`, 2026-10-25): validation also requires `static/`, superseded | `ghcr.io/minicloud-io/sightwell-netbird-server:v0.81.4-sightwell.2@sha256:547d03a7c4b958e6b8b4dafeb535f84f7d73fba474b224df033ab13ab1b57842` |
| `v0.81.4-sightwell.1` (commit `b29735b3`, 2026-10-25, run 40175893900), superseded | `ghcr.io/minicloud-io/sightwell-netbird-server:v0.81.4-sightwell.1@sha256:311c0b8e9bae21d24c97cecf37a21d3109cee17f971701ecc934a38a77cc9d33` |

Each release carries the corresponding source archive (AGPL-3.0 for the
`management/`, `signal/`, `relay/` and `combined/` parts). The GHCR package
is private by default; the cluster needs a pull secret or the package made
public. Upstream's `Release`, `update docs`, `sync tag` and Discourse
workflows are disabled in this fork; re-check with `gh workflow list` after
each upstream merge.

## Keeping up with upstream

```bash
git fetch upstream --tags
git switch -c feat/idp-theme-<tag> <tag>
git cherry-pick <the fork commits>
go build ./management/server/idp/ ./combined/... && go test ./management/server/idp/ ./combined/cmd/
```

The touched functions are small (`ToYAMLConfig`, `buildEmbeddedIdPConfig`,
two structs); a conflict there is a few lines. Never push to
`netbirdio/netbird`; the `upstream` remote's push URL is set to `no_push`.
