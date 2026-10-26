#!/usr/bin/env python3
"""Sightwell fork: generates every client icon from the Sightwell mark.

Replaces, in place and under their upstream file names (so the Go embeds and
build tasks stay unchanged):

  client/ui/assets/netbird-systemtray-<state>{,-dark,-macos,-mono,-mono-dark}.png
  client/ui/assets/netbird-menu-16.png, -24.png, netbird-menu-about-18.png, netbird.png
  client/ui/build/appicon.png  (1024 px; wails3 derives icons.icns and icon.ico)

  client/ui/assets/sightwell-menubar-connected.png, -update-connected.png
      (macOS, full colour, used instead of the template icon while connected)
  client/ui/assets/sightwell-menubar-connecting-NN.png
      (macOS, full colour spinner frames shown while connecting, the
      dashboard's BrandSpinner: the mark inside a track with a turning arc)

The status dots (netbird-menu-dot-*) are status colours, not brand, and stay.

Every tray reads the sight glass like a gauge instead of badges (the Sightwell
design reference, sightwell.ai/brand/): the level at rest = disconnected, an
empty glass = sign-in needed (nothing flows until you sign in), a tilted level
= error or service not running (something is off). The only badge is the
update arrow, top right, in signal orange.

Windows (colour): connected = the mark with its orange level; connecting = the
mark inside a signal-orange arc (one frame of the macOS spinner); the other
states show a grey level, since orange liquid means traffic is flowing.

Linux (mono, black or white by panel theme): no colour to tell connected from
disconnected, so connected is a full glass; connecting is the level at rest
inside an arc.

macOS menu bar (template icons): connected and connecting are the colour icon
and the spinner. The connected and connecting templates keep their badges:
they are never shown, but the app tells states apart by these bytes, so every
macOS template must stay distinct (icons_sightwell_test.go).

Run from the repository root:  python3 client/ui/build/sightwell/icons.py
Requires Pillow. Drawn at 4x and downsampled, so no SVG renderer is needed.
"""
import math
import os
import sys

from PIL import Image, ImageDraw

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", ".."))
ASSETS = os.path.join(ROOT, "client", "ui", "assets")
BUILD = os.path.join(ROOT, "client", "ui", "build")

INK = (0x11, 0x14, 0x17, 255)
PAPER = (0xEC, 0xEA, 0xE4, 255)
GLASS_DARK = (0x1A, 0x1E, 0x22, 255)
WHITE = (255, 255, 255, 255)
ORANGE = (0xF0, 0x7A, 0x3A, 255)
NEUTRAL = (0x8C, 0x91, 0x97, 255)
CLEAR = (0, 0, 0, 0)

SS = 4  # supersampling factor


def circle(d, cx, cy, r, fill):
    d.ellipse((cx - r, cy - r, cx + r, cy + r), fill=fill)


def mark(size, ring, glass, horizon, bolts=False, highlight=None, scale=1.0, offset=(0, 0), tilt=0, drained=False):
    """The sight-glass mark on a transparent canvas, geometry from sightwell-mark.svg
    (120-unit box). glass=None punches the glass out (template icons).
    tilt turns the level (degrees, positive = higher on the left); drained
    leaves the glass empty."""
    S = size * SS
    u = S / 120.0 * scale
    ox, oy = offset[0] * SS, offset[1] * SS
    cx, cy = ox + 60 * u + (S - 120 * u) / 2, oy + 60 * u + (S - 120 * u) / 2
    img = Image.new("RGBA", (S, S), CLEAR)
    d = ImageDraw.Draw(img)
    r_ring = (52 if bolts else 56) * u
    r_glass = (34 if bolts else 33.4) * u
    circle(d, cx, cy, r_ring, ring)
    if bolts:
        for k in range(8):
            a = math.radians(22.5 + 45 * k)
            circle(d, cx + 43 * u * math.cos(a), cy + 43 * u * math.sin(a), 3.4 * u, CLEAR)
    # glass, then horizon clipped to the glass
    glass_mask = Image.new("L", (S, S), 0)
    circle(ImageDraw.Draw(glass_mask), cx, cy, r_glass, 255)
    img.paste(glass if glass else CLEAR, (0, 0), glass_mask)
    # In single-colour icons (glass=None) the horizon is the same colour as the
    # ring, so it is inset by a gap; otherwise the two fuse into a dome.
    gap = 0 if glass else 5.5 * u
    horizon_clip = Image.new("L", (S, S), 0)
    circle(ImageDraw.Draw(horizon_clip), cx, cy, r_glass - gap, 255)
    horizon_mask = Image.new("L", (S, S), 0)
    level = cy + 6 * u + gap * 0.4
    rise = math.tan(math.radians(tilt)) * S / 2
    if not drained:
        ImageDraw.Draw(horizon_mask).polygon([(0, level - rise), (S, level + rise), (S, S), (0, S)], fill=255)
    hm = Image.new("L", (S, S), 0)
    hm.paste(horizon_mask, (0, 0), horizon_clip)
    img.paste(horizon, (0, 0), hm)
    if highlight:
        # small glint arc in the upper-left of the glass (M40 44 A26 26 0 0 1 58 36)
        bbox = (cx - 26 * u, cy - 26 * u, cx + 26 * u, cy + 26 * u)
        d.arc(bbox, start=205, end=265, fill=highlight, width=max(1, int(3 * u)))
    return img


def knock(img, cx, cy, r):
    """Transparent circle (the gap around a badge)."""
    m = Image.new("L", img.size, 0)
    circle(ImageDraw.Draw(m), cx, cy, r, 255)
    img.paste(CLEAR, (0, 0), m)


def glyph(img, kind, cx, cy, r, color):
    d = ImageDraw.Draw(img)
    w = max(1, int(r * 0.42))
    if kind == "check":
        pts = [(cx - r * 0.78, cy + r * 0.02), (cx - r * 0.22, cy + r * 0.58), (cx + r * 0.85, cy - r * 0.62)]
        d.line(pts, fill=color, width=w, joint="curve")
        for p in pts:
            circle(d, p[0], p[1], w / 2, color)
    elif kind == "cross":
        a = r * 0.72
        d.line([(cx - a, cy - a), (cx + a, cy + a)], fill=color, width=w)
        d.line([(cx - a, cy + a), (cx + a, cy - a)], fill=color, width=w)
        for p in [(cx - a, cy - a), (cx + a, cy + a), (cx - a, cy + a), (cx + a, cy - a)]:
            circle(d, p[0], p[1], w / 2, color)
    elif kind == "dots":
        for i in (-1, 0, 1):
            circle(d, cx + i * r * 0.78, cy, r * 0.27, color)
    elif kind == "arrow":
        d.line([(cx, cy + r * 0.85), (cx, cy - r * 0.2)], fill=color, width=w)
        circle(d, cx, cy + r * 0.85, w / 2, color)
        d.polygon([(cx - r * 0.75, cy - r * 0.05), (cx + r * 0.75, cy - r * 0.05), (cx, cy - r * 0.95)], fill=color)


# macOS template icons: the level carries the state instead of a badge.
MACOS_LEVEL = {
    "error": {"tilt": 35},
    "needs-login": {"drained": True},
}

# macOS templates that keep a badge. Connected and connecting are never shown
# (colour icon, spinner) but must stay distinct; the update arrow is real.
BADGES = {
    "connected": [("check", "br")],
    "connecting": [("dots", "br")],
    "disconnected": [],
    "error": [],
    "needs-login": [],
    "update-connected": [("arrow", "tr"), ("check", "br")],
    "update-disconnected": [("arrow", "br")],
}


def tray(state, variant, size):
    """variant: light | dark (colour, Windows), template (macOS, black+alpha), mono | mono-dark (Linux SNI)."""
    if variant != "template":
        return gauge_tray(state, variant, size)
    gauge = MACOS_LEVEL.get(state)
    if gauge:
        return mark(size, INK, None, INK, **gauge).resize((size, size), Image.LANCZOS)
    S = size * SS
    img = mark(size, INK, None, INK)
    r = S * 0.19
    pos = {"br": (S * 0.79, S * 0.79), "tr": (S * 0.79, S * 0.21)}
    for kind, where in BADGES[state]:
        bx, by = pos[where]
        knock(img, bx, by, r * 1.32)
        glyph(img, kind, bx, by, r, INK)
    return img.resize((size, size), Image.LANCZOS)


# Windows and Linux: the level carries the state (see the module docstring).
GAUGE_LEVEL = {
    "connected": {},
    "disconnected": {},
    "needs-login": {"drained": True},
    "error": {"tilt": 35},
}


def full_glass(size, solid):
    """Single-colour mark with the glass filled (Linux mono, connected)."""
    img = mark(size, solid, None, solid, drained=True)
    S = size * SS
    u = S / 120.0
    circle(ImageDraw.Draw(img), S / 2, S / 2, (33.4 - 5.5) * u, solid)
    return img


def mono_connecting(size, solid):
    """Single-colour connecting icon: the level at rest inside an open arc."""
    S = size * SS
    u = S / 120.0
    c = S / 2
    img = Image.new("RGBA", (S, S), CLEAR)
    r, w = 53 * u, 10 * u
    ImageDraw.Draw(img).arc((c - r, c - r, c + r, c + r), -60, 210, fill=solid, width=int(w))
    img.alpha_composite(mark(size, solid, None, solid, scale=0.62))
    return img


def gauge_tray(state, variant, size):
    """Windows (light | dark) and Linux (mono | mono-dark) tray icons, no badges
    except the update arrow."""
    update = state.startswith("update-")
    base = state[len("update-"):] if update else state
    colour = variant in ("light", "dark")
    solid = None if colour else {"mono": INK, "mono-dark": WHITE}[variant]
    if base == "connecting":
        if colour:
            img = menubar_spinner(size, 1).resize((size * SS, size * SS), Image.LANCZOS)
        else:
            img = mono_connecting(size, solid)
    elif colour:
        ring, glass = (INK, WHITE) if variant == "light" else (PAPER, GLASS_DARK)
        level = ORANGE if base == "connected" else NEUTRAL
        img = mark(size, ring, glass, level, **GAUGE_LEVEL[base])
    elif base == "connected":
        img = full_glass(size, solid)
    else:
        img = mark(size, solid, None, solid, **GAUGE_LEVEL[base])
    if update:
        S = size * SS
        r = S * 0.19
        bx, by = S * 0.79, S * 0.21
        knock(img, bx, by, r * 1.32)
        glyph(img, "arrow", bx, by, r, ORANGE if colour else solid)
    return img.resize((size, size), Image.LANCZOS)


# Windows dark-taskbar variants that upstream lacks (it reused the light icon).
NEW_DARK = ("disconnected", "needs-login")


def save_tray():
    out = []
    for state in BADGES:
        base = os.path.join(ASSETS, f"netbird-systemtray-{state}")
        jobs = [("", "light", 256), ("-macos", "template", 366), ("-mono", "mono", 48), ("-mono-dark", "mono-dark", 48)]
        if os.path.exists(base + "-dark.png") or state in NEW_DARK:
            jobs.append(("-dark", "dark", 256))
        for suffix, variant, size in jobs:
            path = f"{base}{suffix}.png"
            img = tray(state, variant, size)
            if variant == "template":
                # upstream ships macOS template icons as grey+alpha
                img = img.convert("LA")
            img.save(path, optimize=True)
            out.append(path)
    return out


def menubar_colour(size, update=False):
    """Full-colour macOS menu-bar icon for the connected states (no badge).
    A colour image is not adapted by macOS like a template, so it carries a
    hairline paper rim to stay readable on dark and tinted menu bars."""
    S = size * SS
    img = Image.new("RGBA", (S, S), CLEAR)
    circle(ImageDraw.Draw(img), S / 2, S / 2, 56 * S / 120 + 7 * SS, PAPER)
    img.alpha_composite(mark(size, INK, WHITE, ORANGE))
    if update:
        r = S * 0.19
        bx, by = S * 0.79, S * 0.21
        knock(img, bx, by, r * 1.32)
        glyph(img, "arrow", bx, by, r, ORANGE)
    return img.resize((size, size), Image.LANCZOS)


def save_menubar_colour():
    out = []
    for name, update in [("sightwell-menubar-connected.png", False), ("sightwell-menubar-update-connected.png", True)]:
        menubar_colour(366, update).save(os.path.join(ASSETS, name), optimize=True)
        out.append(name)
    return out


SPINNER_FRAMES = 12
SPINNER_SIZE = 176


def menubar_spinner(size, step, steps=SPINNER_FRAMES):
    """One frame of the macOS connecting spinner (colour, like the connected
    icon): the mark with its paper rim at 0.66 scale, a neutral track at the
    edge, and a 270 degree signal-orange arc rotated by step/steps of a turn."""
    S = size * SS
    u = S / 120.0
    c = S / 2
    img = Image.new("RGBA", (S, S), CLEAR)
    d = ImageDraw.Draw(img)
    r, w = 53 * u, 10 * u
    box = (c - r, c - r, c + r, c + r)
    # Track: mid grey at partial alpha reads on light and dark menu bars.
    d.arc(box, 0, 360, fill=(0x8C, 0x91, 0x97, 110), width=int(w))
    start = -90 + 360.0 * step / steps
    end = start + 270
    d.arc(box, start, end, fill=ORANGE, width=int(w))
    for a in (start, end):  # round caps
        t = math.radians(a)
        circle(d, c + (r - w / 2) * math.cos(t), c + (r - w / 2) * math.sin(t), w / 2, ORANGE)
    scale = 0.66
    circle(d, c, c, (56 * scale + 5) * u, PAPER)
    img.alpha_composite(mark(size, INK, WHITE, ORANGE, scale=scale))
    return img.resize((size, size), Image.LANCZOS)


def save_menubar_spinner():
    out = []
    for i in range(SPINNER_FRAMES):
        name = f"sightwell-menubar-connecting-{i:02d}.png"
        menubar_spinner(SPINNER_SIZE, i).save(os.path.join(ASSETS, name), optimize=True)
        out.append(name)
    return out


def save_menu_icons():
    out = []
    # Menu rows are shown on light and dark menus: neutral ring, open glass, orange horizon.
    for name, size in [("netbird-menu-16.png", 16), ("netbird-menu-24.png", 24), ("netbird-menu-about-18.png", 18)]:
        img = mark(size, NEUTRAL, None, ORANGE).resize((size, size), Image.LANCZOS)
        img.save(os.path.join(ASSETS, name), optimize=True)
        out.append(name)
    img = mark(256, INK, WHITE, ORANGE, bolts=True, highlight=(0x11, 0x14, 0x17, 140)).resize((256, 256), Image.LANCZOS)
    img.save(os.path.join(ASSETS, "netbird.png"), optimize=True)
    out.append("netbird.png")
    return out


def save_appicon():
    """1024 px macOS app icon on Apple's grid: 824 px squircle-ish tile at 100 px."""
    size = 1024
    S = size * SS
    tile = Image.new("RGBA", (S, S), CLEAR)
    d = ImageDraw.Draw(tile)
    m, rad = 100 * SS, 185 * SS
    d.rounded_rectangle((m, m, S - m, S - m), radius=rad, fill=INK)
    glyph_img = mark(size, PAPER, GLASS_DARK, ORANGE, bolts=True, highlight=(0xEC, 0xEA, 0xE4, 140), scale=0.66)
    tile.alpha_composite(glyph_img)
    tile = tile.resize((size, size), Image.LANCZOS)
    tile.save(os.path.join(BUILD, "appicon.png"), optimize=True)
    return ["appicon.png"]


if __name__ == "__main__":
    done = save_tray() + save_menubar_colour() + save_menubar_spinner() + save_menu_icons() + save_appicon()
    print(f"wrote {len(done)} icons")
    sys.exit(0)
