#!/bin/sh
# Sightwell fork: removes Sightwell from this Mac and nothing else. A NetBird
# install (netbird service, /usr/local/bin/netbird, NetBird.app) is left as is.
# Run with sudo. Shipped in Sightwell.app/Contents/Resources/uninstall.sh.
set -eu

[ "$(id -u)" -eq 0 ] || { echo "Run with sudo: sudo $0" >&2; exit 1; }

AGENT=/usr/local/bin/sightwell
APP=/Applications/Sightwell.app

pkill -x sightwell-ui 2>/dev/null || true
if [ -x "$AGENT" ]; then
    "$AGENT" service stop 2>/dev/null || true
    "$AGENT" service uninstall 2>/dev/null || true
fi
# Only Sightwell-owned paths below.
rm -f "$AGENT"
rm -rf "$APP"
rm -rf /var/lib/sightwell /etc/sightwell /var/log/sightwell /var/run/sightwell /var/run/sightwell.sock
pkgutil --forget ai.sightwell.connectivity.pkg 2>/dev/null || true
echo "Sightwell removed. User settings remain in ~/Library/Application Support/sightwell for each user."
