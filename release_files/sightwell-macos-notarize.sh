#!/bin/sh
# Sightwell fork: notarises and staples a Developer ID signed package built by
# sightwell-macos-pkg.sh, then checks it the way Gatekeeper does.
#
#   release_files/sightwell-macos-notarize.sh PKG
#
# Credentials: an App Store Connect API key of MINICLOUD FZCO's team
# (US5JKY7Z7F) with the Developer role:
#   ASC_KEY_PATH   the .p8 file
#   ASC_KEY_ID     its key ID
#   ASC_ISSUER_ID  the issuer ID shown with the team keys
# Stapling changes the file, so PKG.sha256 is rewritten afterwards.
set -eu

[ "$#" -eq 1 ] || { echo "usage: $0 PKG" >&2; exit 2; }
PKG=$1
[ -f "$PKG" ] || { echo "no such package: $PKG" >&2; exit 2; }
: "${ASC_KEY_PATH:?set ASC_KEY_PATH to the App Store Connect API key (.p8)}"
: "${ASC_KEY_ID:?set ASC_KEY_ID}"
: "${ASC_ISSUER_ID:?set ASC_ISSUER_ID}"

WORK=$(mktemp -d "${TMPDIR:-/tmp}/sightwell-notarize.XXXXXX")
trap 'rm -rf "$WORK"' EXIT HUP INT TERM

echo "==> signature before submitting"
pkgutil --check-signature "$PKG"

echo "==> notarytool submit (waits for Apple)"
# A rejected submission still exits 0 here; the status decides.
xcrun notarytool submit "$PKG" \
    --key "$ASC_KEY_PATH" --key-id "$ASC_KEY_ID" --issuer "$ASC_ISSUER_ID" \
    --wait --timeout 30m --output-format json >"$WORK/submit.json"
STATUS=$(plutil -extract status raw -o - "$WORK/submit.json")
ID=$(plutil -extract id raw -o - "$WORK/submit.json")
echo "submission $ID: $STATUS"
if [ "$STATUS" != "Accepted" ]; then
    # The log names each file Apple rejected and why.
    xcrun notarytool log "$ID" \
        --key "$ASC_KEY_PATH" --key-id "$ASC_KEY_ID" --issuer "$ASC_ISSUER_ID" || true
    exit 1
fi

echo "==> staple"
xcrun stapler staple "$PKG"
xcrun stapler validate "$PKG"

echo "==> Gatekeeper checks"
spctl --assess --type install --verbose=2 "$PKG"
pkgutil --expand-full "$PKG" "$WORK/expanded"
APP=$(find "$WORK/expanded" -maxdepth 4 -type d -name Sightwell.app | head -n 1)
[ -n "$APP" ] || { echo "Sightwell.app not found in $PKG" >&2; exit 1; }
codesign --verify --strict --deep --verbose=2 "$APP"
spctl --assess --type execute --verbose=2 "$APP"

shasum -a 256 "$PKG" | tee "$PKG.sha256"
echo "Notarised $PKG"
