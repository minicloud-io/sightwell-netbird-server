#!/bin/sh
# Sightwell fork: builds the macOS installer package.
#
#   release_files/sightwell-macos-pkg.sh VERSION [OUTPUT_DIR]
#
# Produces OUTPUT_DIR/Sightwell-VERSION-unsigned.pkg (default OUTPUT_DIR: dist/sightwell-macos),
# or OUTPUT_DIR/Sightwell-VERSION.pkg when Developer ID identities are given:
#   SIGN_APP_IDENTITY  "Developer ID Application: MINICLOUD - FZCO (US5JKY7Z7F)"
#   SIGN_PKG_IDENTITY  "Developer ID Installer: MINICLOUD - FZCO (US5JKY7Z7F)"
#   SIGN_KEYCHAIN      keychain holding them (optional; default: the search list)
# Both or neither. Notarise the signed package with sightwell-macos-notarize.sh.
# installing /Applications/Sightwell.app, which holds:
#   Contents/MacOS/sightwell-ui   the desktop UI (CFBundleExecutable)
#   Contents/MacOS/sightwell      the daemon and CLI, linked to /usr/local/bin/sightwell
#   Contents/Resources/licenses/  NetBird's BSD-3-Clause licence and third-party licences
#   Contents/Resources/uninstall.sh
# Both binaries are universal (arm64 + amd64). The package receipt is
# ai.sightwell.connectivity.pkg; NetBird's (io.netbird.client) is never touched.
# Needs macOS with Xcode command line tools, Go, Node/pnpm and wails3 (task package).
set -eu

[ "$#" -ge 1 ] || { echo "usage: $0 VERSION [OUTPUT_DIR]" >&2; exit 2; }
VERSION=${1#v}
ROOT=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
OUT=${2:-$ROOT/dist/sightwell-macos}
mkdir -p "$OUT"
OUT=$(CDPATH='' cd -- "$OUT" && pwd)
WORK=$(mktemp -d "${TMPDIR:-/tmp}/sightwell-pkg.XXXXXX")
trap 'rm -rf "$WORK"' EXIT HUP INT TERM

LDFLAGS="-s -w -X github.com/netbirdio/netbird/version.version=$VERSION"

SIGN_APP_IDENTITY=${SIGN_APP_IDENTITY:-}
SIGN_PKG_IDENTITY=${SIGN_PKG_IDENTITY:-}
SIGN_KEYCHAIN=${SIGN_KEYCHAIN:-}
case "${SIGN_APP_IDENTITY:+app}${SIGN_PKG_IDENTITY:+pkg}" in
    app | pkg)
        echo "set both SIGN_APP_IDENTITY and SIGN_PKG_IDENTITY, or neither" >&2
        exit 2
        ;;
esac

# devid_codesign PATH: Developer ID signature with the hardened runtime and a
# secure timestamp, both required for notarisation.
devid_codesign() {
    if [ -n "$SIGN_KEYCHAIN" ]; then
        codesign --force --options runtime --timestamp --keychain "$SIGN_KEYCHAIN" \
            --sign "$SIGN_APP_IDENTITY" "$1"
    else
        codesign --force --options runtime --timestamp --sign "$SIGN_APP_IDENTITY" "$1"
    fi
}

echo "==> daemon (arm64 + amd64)"
for arch in arm64 amd64; do
    (cd "$ROOT/client" && CGO_ENABLED=0 GOOS=darwin GOARCH=$arch \
        go build -trimpath -ldflags "$LDFLAGS" -o "$WORK/sightwell-$arch" .)
done
lipo -create -output "$WORK/sightwell" "$WORK/sightwell-arm64" "$WORK/sightwell-amd64"

echo "==> UI app (universal)"
# Start from a clean bundle: create:app:bundle copies into an existing one,
# so a stale binary from an earlier build would ship too.
rm -rf "$ROOT/client/ui/bin/Sightwell.app"
(cd "$ROOT/client/ui" && task darwin:package:universal VERSION="$VERSION")
APP_SRC=$ROOT/client/ui/bin/Sightwell.app
[ -x "$APP_SRC/Contents/MacOS/sightwell-ui" ] || { echo "UI build did not produce $APP_SRC" >&2; exit 1; }

echo "==> assemble Sightwell.app"
PAYLOAD=$WORK/payload
mkdir -p "$PAYLOAD/Applications"
cp -R "$APP_SRC" "$PAYLOAD/Applications/Sightwell.app"
APP=$PAYLOAD/Applications/Sightwell.app
cp "$WORK/sightwell" "$APP/Contents/MacOS/sightwell"
cp "$ROOT/release_files/sightwell_darwin_pkg/uninstall.sh" "$APP/Contents/Resources/uninstall.sh"
chmod 755 "$APP/Contents/MacOS/sightwell" "$APP/Contents/Resources/uninstall.sh"
# Bundle versions must be numeric (X.Y.Z), so they carry the upstream base
# version; the binaries and the package carry the full fork version.
BUNDLE_VERSION=$(printf '%s' "$VERSION" | sed 's/[^0-9.].*$//; s/\.$//')
plutil -replace CFBundleShortVersionString -string "$BUNDLE_VERSION" "$APP/Contents/Info.plist"
plutil -replace CFBundleVersion -string "$BUNDLE_VERSION" "$APP/Contents/Info.plist"

echo "==> licences"
# Homebrew's Go omits GOROOT/LICENSE; golang.org/x/sys carries the same Go
# Authors licence text (CI uses the official Go distribution, which has it).
if [ ! -f "$(go env GOROOT)/LICENSE" ] && [ -z "${GO_STDLIB_TERMS_DIR:-}" ]; then
    GO_STDLIB_TERMS_DIR=$WORK/go-terms
    mkdir -p "$GO_STDLIB_TERMS_DIR"
    cp "$(cd "$ROOT" && go list -m -f '{{.Dir}}' golang.org/x/sys)/LICENSE" "$GO_STDLIB_TERMS_DIR/LICENSE"
    cp "$(go env GOROOT)/PATENTS" "$GO_STDLIB_TERMS_DIR/PATENTS"
    export GO_STDLIB_TERMS_DIR
fi
sh "$ROOT/release_files/collect-licenses.sh" -t production "$WORK/licenses" ./client/ui amd64 arm64
sh "$ROOT/release_files/collect-licenses.sh" "$WORK/licenses-daemon" ./client amd64 arm64
# Merge: the daemon's third-party set into the UI's (same layout).
cp -Rn "$WORK/licenses-daemon/." "$WORK/licenses/" 2>/dev/null || true
cp -R "$WORK/licenses" "$APP/Contents/Resources/licenses"
# The collected tree is private to the build user; installed, it must be
# readable by everyone (the BSD-3-Clause notices ship with the binaries).
chmod -R u+rwX,go+rX,go-w "$APP/Contents/Resources/licenses"

if [ -n "$SIGN_APP_IDENTITY" ]; then
    echo "==> Developer ID signing ($SIGN_APP_IDENTITY)"
    # Inside out, no --deep: the daemon first, then the bundle, which signs its
    # main executable (sightwell-ui) and seals the daemon and resources.
    devid_codesign "$APP/Contents/MacOS/sightwell"
    devid_codesign "$APP"
    codesign --verify --strict --deep --verbose=2 "$APP"
else
    # Ad-hoc signature over the final contents: runs on the building Mac only.
    codesign --force --deep --sign - "$APP"
fi

echo "==> package"
if [ -n "$SIGN_PKG_IDENTITY" ]; then
    PKG=$OUT/Sightwell-$VERSION.pkg
else
    PKG=$OUT/Sightwell-$VERSION-unsigned.pkg
fi
# Not relocatable: always install to /Applications, never into a copy of
# Sightwell.app found elsewhere on the disk.
pkgbuild --analyze --root "$PAYLOAD" "$WORK/components.plist"
plutil -replace 0.BundleIsRelocatable -bool NO "$WORK/components.plist"
pkgbuild --root "$PAYLOAD" \
    --component-plist "$WORK/components.plist" \
    --identifier ai.sightwell.connectivity.pkg \
    --version "$VERSION" \
    --install-location / \
    --scripts "$ROOT/release_files/sightwell_darwin_pkg" \
    --ownership recommended \
    "$WORK/component.pkg"
# The scripts directory also holds uninstall.sh; pkgbuild only runs preinstall/postinstall.
if [ -z "$SIGN_PKG_IDENTITY" ]; then
    productbuild --package "$WORK/component.pkg" --identifier ai.sightwell.connectivity.pkg --version "$VERSION" "$PKG"
elif [ -n "$SIGN_KEYCHAIN" ]; then
    productbuild --package "$WORK/component.pkg" --identifier ai.sightwell.connectivity.pkg --version "$VERSION" \
        --sign "$SIGN_PKG_IDENTITY" --timestamp --keychain "$SIGN_KEYCHAIN" "$PKG"
else
    productbuild --package "$WORK/component.pkg" --identifier ai.sightwell.connectivity.pkg --version "$VERSION" \
        --sign "$SIGN_PKG_IDENTITY" --timestamp "$PKG"
fi
shasum -a 256 "$PKG" | tee "$PKG.sha256"
echo "Built $PKG"
