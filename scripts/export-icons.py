#!/usr/bin/env python3
"""Exports every icon H3XD shows, for drawing and checking the H3XD icon theme.

Collects the icon names of the menu entries and menu categories, the desktop, the file
manager, the session and dialog icons, and resolves each through the icon theme chain of
the system (H3XD and what it inherits) to the largest file there; the theme column tells
which names H3XD does not draw yet. H3XD's own artwork (mark, lockups, title-bar buttons,
workspace hexagons) is copied as it is.

Writes <out>/<category>/<name>.<ext>, icons.csv and index.html, a contact sheet.

  export-icons.py <out> <root> [<root> ...]

Each root is a system tree, such as the airootfs of an ISO build or extracted packages.
"""

import configparser
import csv
import html
import os
import shutil
import sys

THEME = "H3XD"
# Size directories in order of preference: large bitmaps of the design sizes first.
SIZE_PREFERENCE = ("64x64", "48x48", "scalable", "96x96", "128x128", "256x256", "512x512",
                   "32x32", "24x24", "22x22", "16x16")
EXTENSIONS = (".svg", ".png", ".xpm")

# Icons Xfce and H3XD use by name outside the menu, with where they appear.
UI_ICONS = {
    "Desktop": ["user-home", "drive-harddisk", "user-trash", "user-trash-full", "user-desktop",
                "drive-removable-media", "media-optical", "computer"],
    "File manager places": ["folder", "folder-open", "folder-documents", "folder-download",
                            "folder-music", "folder-pictures", "folder-videos",
                            "folder-publicshare", "folder-templates", "folder-remote",
                            "document-open-recent", "network-workgroup", "network-server"],
    "File types": ["text-x-generic", "text-x-script", "text-x-python", "text-html",
                   "application-x-executable", "application-x-shellscript",
                   "application-pdf", "application-json", "image-x-generic", "image-png",
                   "image-jpeg", "image-svg+xml", "video-x-generic", "audio-x-generic",
                   "application-zip", "application-x-compressed-tar", "package-x-generic",
                   "application-x-cd-image", "x-office-document", "x-office-spreadsheet",
                   "x-office-presentation", "font-x-generic", "unknown"],
    "Session and system": ["system-log-out", "system-lock-screen", "system-shutdown",
                           "system-reboot", "system-suspend", "system-hibernate",
                           "system-switch-user", "system-search", "system-software-update",
                           "software-update-available", "preferences-system",
                           "preferences-desktop"],
    # The Whisker menu, the panel actions and the logout dialog ask for these first.
    "Session actions": ["xfsm-lock", "xfsm-logout", "xfsm-shutdown", "xfsm-reboot",
                        "xfsm-suspend", "xfsm-hibernate", "xfsm-switch-user"],
    "Dialogs": ["dialog-information", "dialog-warning", "dialog-error", "dialog-question",
                "dialog-password"],
    "Notifications": ["network-wireless-signal-excellent", "network-wireless-signal-good",
                      "network-wireless-signal-ok", "network-wireless-signal-weak",
                      "network-wireless-signal-none", "network-wired", "network-offline",
                      "audio-volume-high", "audio-volume-medium", "audio-volume-low",
                      "audio-volume-muted", "battery-full", "battery-good", "battery-low",
                      "battery-caution", "battery-empty", "battery-full-charging",
                      "battery-good-charging", "battery-low-charging"],
}

# H3XD's own artwork, relative to a root, by the category it is filed under.
OWN_ARTWORK = {
    "h3xd/brand": ["usr/share/h3xd", "usr/share/icons/hicolor/scalable/apps"],
    "h3xd/window-buttons": ["usr/share/themes/h3xd-dark/xfwm4"],
    "h3xd/workspaces": ["usr/share/themes/h3xd-dark/gtk-3.0/assets"],
}


def desktop_entries(roots, folder, suffix):
    """(file name, Name, Icon) of the entries shown in menus."""
    seen = {}
    for root in roots:
        directory = os.path.join(root, folder)
        if not os.path.isdir(directory):
            continue
        for name in sorted(os.listdir(directory)):
            if not name.endswith(suffix) or name in seen:
                continue
            parser = configparser.ConfigParser(interpolation=None, strict=False)
            parser.optionxform = str
            try:
                parser.read(os.path.join(directory, name), encoding="utf-8")
            except (configparser.Error, UnicodeDecodeError) as error:
                print(f"export-icons: skipping {name}: {error}", file=sys.stderr)
                continue
            section = "Desktop Entry"
            if not parser.has_section(section):
                continue
            entry = parser[section]
            if entry.get("NoDisplay") == "true" or entry.get("Hidden") == "true":
                continue
            if entry.get("Icon"):
                seen[name] = (entry.get("Name", name), entry["Icon"])
    return seen


def theme_chain(roots, theme, chain=None):
    chain = chain if chain is not None else []
    if theme in chain:
        return chain
    chain.append(theme)
    for root in roots:
        index = os.path.join(root, "usr/share/icons", theme, "index.theme")
        if os.path.exists(index):
            parser = configparser.ConfigParser(interpolation=None, strict=False)
            parser.read(index, encoding="utf-8")
            for parent in parser.get("Icon Theme", "Inherits", fallback="").split(","):
                if parent.strip():
                    theme_chain(roots, parent.strip(), chain)
            break
    if "hicolor" not in chain:
        chain.append("hicolor")
    return chain


def index_themes(roots, chain):
    """{name: [(theme, size, context, path)]} for every icon file of the chain."""
    found = {}
    for theme in chain:
        for root in roots:
            base = os.path.join(root, "usr/share/icons", theme)
            if not os.path.isdir(base):
                continue
            for size in sorted(os.listdir(base)):
                size_dir = os.path.join(base, size)
                if not os.path.isdir(size_dir) or "symbolic" in size:
                    continue
                for context in sorted(os.listdir(size_dir)):
                    context_dir = os.path.join(size_dir, context)
                    if not os.path.isdir(context_dir):
                        continue
                    for file in os.listdir(context_dir):
                        stem, ext = os.path.splitext(file)
                        if ext in EXTENSIONS and not stem.endswith("-symbolic"):
                            found.setdefault(stem, []).append(
                                (theme, size, context, os.path.join(context_dir, file)))
    return found


def resolve(name, roots, chain, found):
    """(path, theme, context, sizes) of the best file for an icon name, or None."""
    if os.path.isabs(name):
        for root in roots:
            path = os.path.join(root, name.lstrip("/"))
            if os.path.exists(path):
                return path, "file", "apps", []
        return None
    candidates = found.get(name, [])
    for theme in chain:
        ours = [c for c in candidates if c[0] == theme and os.path.exists(c[3])]
        if not ours:
            continue
        sizes = sorted({c[1] for c in ours})
        for size in SIZE_PREFERENCE:
            for candidate in ours:
                if candidate[1] == size:
                    return candidate[3], theme, candidate[2], sizes
        return ours[0][3], theme, ours[0][2], sizes
    for root in roots:
        for ext in EXTENSIONS:
            path = os.path.join(root, "usr/share/pixmaps", name + ext)
            if os.path.exists(path):
                return path, "pixmaps", "apps", []
    return None


def copy_icon(path, out, category, name):
    ext = os.path.splitext(path)[1]
    target_dir = os.path.join(out, category)
    os.makedirs(target_dir, exist_ok=True)
    target = os.path.join(target_dir, name + ext)
    # Themes are full of symlinks; the export holds the files themselves.
    shutil.copyfile(os.path.realpath(path), target)
    return os.path.relpath(target, out)


def contact_sheet(rows):
    groups = {}
    for row in rows:
        groups.setdefault(row["category"], []).append(row)
    sections = []
    for category in sorted(groups):
        cards = []
        for row in sorted(groups[category], key=lambda r: r["name"]):
            if row["file"]:
                image = f'<img src="{html.escape(row["file"])}" alt="" width="64" height="64">'
            else:
                image = '<div class="missing">missing</div>'
            cards.append(
                f'<figure>{image}<figcaption><b>{html.escape(row["name"])}</b>'
                f'<span>{html.escape(row["used_by"])}</span></figcaption></figure>')
        sections.append(f'<h2>{html.escape(category)} <small>{len(cards)}</small></h2>'
                        f'<div class="grid">{"".join(cards)}</div>')
    return f"""<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>H3XD icons</title>
<style>
body {{ margin: 0; padding: 32px; background: #10141b; color: #e6edf3;
       font: 13px/1.4 "IBM Plex Sans", sans-serif; }}
h1 {{ font-weight: 600; font-size: 22px; margin: 0 0 4px; }}
p {{ color: #8b95a5; margin: 0 0 24px; }}
h2 {{ font-size: 15px; font-weight: 600; margin: 32px 0 12px; }}
h2 small {{ color: #8b95a5; font-weight: 400; }}
.grid {{ display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 8px; }}
figure {{ margin: 0; padding: 14px 10px; background: #181c25; border-radius: 4px;
         display: flex; flex-direction: column; align-items: center; gap: 8px; }}
figcaption {{ display: flex; flex-direction: column; align-items: center; text-align: center;
             word-break: break-all; }}
figcaption b {{ font: 500 12px "JetBrains Mono", monospace; }}
figcaption span {{ color: #8b95a5; font-size: 11px; }}
.missing {{ width: 64px; height: 64px; display: grid; place-items: center; color: #f07075;
           border: 1px dashed #3a4253; border-radius: 4px; font-size: 11px; }}
</style></head><body>
<h1>H3XD icons</h1>
<p>{len(rows)} icons the system shows, with the current file and where each one appears.</p>
{"".join(sections)}
</body></html>
"""


def main(argv):
    if len(argv) < 3:
        print(__doc__.strip(), file=sys.stderr)
        return 2
    out, roots = argv[1], argv[2:]
    if os.path.exists(out):
        shutil.rmtree(out)
    os.makedirs(out)

    chain = theme_chain(roots, THEME)
    found = index_themes(roots, chain)
    wanted = {}
    for file, (label, icon) in desktop_entries(roots, "usr/share/applications", ".desktop").items():
        wanted.setdefault(icon, set()).add(f"Menu: {label}")
    for file, (label, icon) in desktop_entries(roots, "usr/share/desktop-directories",
                                                ".directory").items():
        wanted.setdefault(icon, set()).add(f"Menu category: {label}")
    for place, names in UI_ICONS.items():
        for name in names:
            wanted.setdefault(name, set()).add(place)

    rows = []
    for name in sorted(wanted):
        result = resolve(name, roots, chain, found)
        label = os.path.basename(name).rsplit(".", 1)[0] if os.path.isabs(name) else name
        if result is None:
            rows.append({"name": label, "category": "missing", "file": "", "theme": "",
                         "sizes": "", "used_by": "; ".join(sorted(wanted[name]))})
            continue
        path, theme, context, sizes = result
        rows.append({"name": label, "category": context,
                     "file": copy_icon(path, out, context, label), "theme": theme,
                     "sizes": " ".join(sizes), "used_by": "; ".join(sorted(wanted[name]))})

    for category, folders in OWN_ARTWORK.items():
        for folder in folders:
            for root in roots:
                directory = os.path.join(root, folder)
                if not os.path.isdir(directory):
                    continue
                for file in sorted(os.listdir(directory)):
                    stem, ext = os.path.splitext(file)
                    if ext not in EXTENSIONS or not (stem.startswith(("h3xd", "ws-"))
                                                     or "xfwm4" in folder):
                        continue
                    rows.append({"name": stem, "category": category,
                                 "file": copy_icon(os.path.join(directory, file), out,
                                                   category, stem),
                                 "theme": "h3xd", "sizes": "", "used_by": "H3XD artwork"})
                break

    with open(os.path.join(out, "icons.csv"), "w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=["name", "category", "file", "theme",
                                                    "sizes", "used_by"])
        writer.writeheader()
        writer.writerows(rows)
    with open(os.path.join(out, "index.html"), "w", encoding="utf-8") as handle:
        handle.write(contact_sheet(rows))
    missing = sum(1 for row in rows if row["category"] == "missing")
    print(f"export-icons: {len(rows)} icons in {out}, {missing} not found in {', '.join(chain)}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
