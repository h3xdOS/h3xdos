"""Shared look and placement of the H3XD shell windows (design v2, sheet 2f)."""

import gi

gi.require_version("Gdk", "3.0")
gi.require_version("Gtk", "3.0")
from gi.repository import Gdk, Gtk  # noqa: E402

# Space between the panel, windows and the screen edges (design 2a).
GAP = 10

# Sheet 2f tokens used by every shell window.
BASE_CSS = """
window.h3xd-popup {
    background-color: transparent;
}
.h3xd-card {
    background-color: #1a1e27;
    border-radius: 4px;
    color: #e6edf3;
    font-family: "IBM Plex Sans";
    font-size: 13px;
}
.h3xd-muted {
    color: #8b95a5;
}
.h3xd-mono {
    font-family: "JetBrains Mono";
}
.h3xd-rule {
    border-top: 1px solid #2b3240;
}
.h3xd-row {
    padding: 10px 12px;
    border-radius: 4px;
    background-color: transparent;
    color: #e6edf3;
}
.h3xd-row:hover, .h3xd-row.selected {
    background-color: #12304a;
}
.h3xd-card button {
    background-image: none;
    background-color: #2b3240;
    border: none;
    border-radius: 3px;
    color: #e6edf3;
    padding: 6px 14px;
    box-shadow: none;
}
.h3xd-card button:hover {
    background-color: #3a4253;
}
.h3xd-card button.h3xd-primary {
    background-color: #1793d1;
    color: #0f1320;
}
.h3xd-card button.h3xd-primary:hover {
    background-color: #5cb6e4;
}
.h3xd-card button.h3xd-flat {
    background-color: transparent;
    padding: 4px 8px;
}
.h3xd-card button.h3xd-flat:hover {
    background-color: #1f2430;
}
.h3xd-card entry {
    background-image: none;
    background-color: #10141b;
    border: 2px solid #2b3240;
    border-radius: 4px;
    color: #e6edf3;
    caret-color: #1793d1;
    padding: 8px 12px;
    box-shadow: none;
}
.h3xd-card entry:focus {
    border-color: #1793d1;
}
.h3xd-error {
    color: #f07075;
}
"""


def apply_css(css=""):
    """Installs the shared styles plus the window's own at application priority."""
    provider = Gtk.CssProvider()
    provider.load_from_data((BASE_CSS + css).encode())
    Gtk.StyleContext.add_provider_for_screen(
        Gdk.Screen.get_default(), provider, Gtk.STYLE_PROVIDER_PRIORITY_APPLICATION
    )


def pointer_monitor():
    """Returns the monitor under the pointer and the pointer position."""
    display = Gdk.Display.get_default()
    _screen, x, y = display.get_default_seat().get_pointer().get_position()
    return display.get_monitor_at_point(x, y), x, y


def make_popup(app, title):
    """An undecorated window that can carry rounded corners when a compositor runs."""
    window = Gtk.ApplicationWindow(application=app, title=title)
    window.set_decorated(False)
    window.set_resizable(False)
    window.set_skip_taskbar_hint(True)
    window.set_skip_pager_hint(True)
    window.set_keep_above(True)
    window.get_style_context().add_class("h3xd-popup")
    screen = window.get_screen()
    visual = screen.get_rgba_visual()
    if visual is not None and screen.is_composited():
        window.set_visual(visual)
    return window


def place_under_panel(window, width):
    """Puts a popup right below the top panel, centred on the pointer, inside its monitor."""
    monitor, x, _y = pointer_monitor()
    area = monitor.get_workarea()
    left = min(max(x - width // 2, area.x + GAP), area.x + area.width - width - GAP)
    # The workarea starts below the panel's own bottom gap.
    window.move(left, area.y)


def center_on_pointer_monitor(window, width, height):
    monitor, _x, _y = pointer_monitor()
    geometry = monitor.get_geometry()
    window.move(geometry.x + (geometry.width - width) // 2,
                geometry.y + (geometry.height - height) // 2)


def close_on_escape(window):
    def on_key(_widget, event):
        if event.keyval == Gdk.KEY_Escape:
            window.close()
            return True
        return False

    window.connect("key-press-event", on_key)


def close_on_focus_out(window):
    window.connect("focus-out-event", lambda *_args: window.close())
