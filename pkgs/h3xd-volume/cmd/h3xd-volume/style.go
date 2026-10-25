package main

import (
	"log"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// css follows sheet 2f of design v2 and the layout of the settings window (2d), as H3XD
// Firewall does: navigation on the left, cards with 4 px corners, IBM Plex Sans for text,
// JetBrains Mono for numbers. Slider knobs are the hexagon of the H3XD toggle, drawn by
// the PKGBUILD into /usr/share/h3xd/volume.
const css = `
.h3xd-vol {
    background-color: #181c25;
    color: #e6edf3;
    font-family: "IBM Plex Sans";
    font-size: 13px;
}
.h3xd-vol-side {
    background-color: #151922;
    padding: 20px 12px;
}
button.h3xd-nav {
    padding: 9px 12px;
    border-radius: 4px;
    border: none;
    box-shadow: none;
    background-image: none;
    background-color: transparent;
    color: #c6cbd5;
    font-size: 14px;
}
button.h3xd-nav:hover {
    background-color: #1f2430;
}
button.h3xd-nav.selected {
    background-color: #12304a;
    color: #e6edf3;
    font-weight: 600;
}
.h3xd-vol-main {
    padding: 36px 0 0 48px;
}
.h3xd-vol-title, .h3xd-vol-subtitle, .h3xd-vol-content {
    margin-right: 48px;
}
.h3xd-vol-title {
    font-size: 26px;
    font-weight: 600;
}
.h3xd-vol-subtitle {
    margin-top: 6px;
    margin-bottom: 24px;
}
.h3xd-vol-content {
    margin-bottom: 36px;
}
.h3xd-vol-heading {
    font-size: 14px;
    font-weight: 600;
    margin-top: 10px;
}
.h3xd-muted {
    color: #8b95a5;
}
.h3xd-vol-meta {
    font-family: "JetBrains Mono";
    font-size: 11px;
    color: #8b95a5;
}
.h3xd-vol-card {
    background-color: #1a202b;
    border-radius: 4px;
    padding: 16px 18px;
}
.h3xd-vol-big {
    font-size: 18px;
    font-weight: 600;
}
.h3xd-vol-name {
    font-size: 14px;
    font-weight: 500;
}
.h3xd-vol-label {
    color: #8b95a5;
    font-size: 12px;
}
.h3xd-vol-mono {
    font-family: "JetBrains Mono";
    font-size: 12px;
}
.h3xd-vol-empty {
    padding: 4px 0 8px 0;
}
.h3xd-chip {
    font-family: "JetBrains Mono";
    font-size: 11px;
    font-weight: 600;
    border-radius: 3px;
    padding: 3px 8px;
    background-color: #2b3240;
    color: #c6cbd5;
}
.h3xd-notice {
    background-color: #12304a;
    color: #e6edf3;
    border-radius: 4px;
    padding: 10px 14px;
    margin-bottom: 18px;
}
.h3xd-notice.error {
    background-color: #3a1d22;
    color: #f5b5b8;
}
button.h3xd-button {
    background-image: none;
    background-color: #2b3240;
    border: none;
    border-radius: 3px;
    box-shadow: none;
    color: #e6edf3;
    padding: 6px 14px;
}
button.h3xd-button:hover {
    background-color: #3a4253;
}
/* Mute is a switch with words: red while the sound is off. */
button.h3xd-vol-mute {
    min-width: 64px;
}
button.h3xd-vol-mute.on {
    background-color: #3a1d22;
    color: #f08a8e;
    font-weight: 600;
}
button.h3xd-vol-mute.on:hover {
    background-color: #4a252b;
}
.h3xd-seg button {
    border-radius: 0;
    padding: 6px 14px;
}
.h3xd-seg button:first-child {
    border-radius: 3px 0 0 3px;
}
.h3xd-seg button:last-child {
    border-radius: 0 3px 3px 0;
}
.h3xd-seg button.selected {
    background-color: #1793d1;
    color: #0f1320;
    font-weight: 600;
}
combobox.h3xd-vol-big-combo button {
    font-size: 16px;
    font-weight: 600;
    padding: 6px 12px;
}
.h3xd-vol scale {
    padding: 10px 9px;
}
.h3xd-vol scale trough {
    min-height: 6px;
    border: none;
    border-radius: 3px;
    background-color: #2b3240;
}
.h3xd-vol scale highlight {
    min-height: 6px;
    border: none;
    border-radius: 3px;
    background-image: none;
    background-color: #1793d1;
}
.h3xd-vol scale.muted highlight, .h3xd-vol scale.h3xd-vol-balance highlight {
    background-color: #4b5366;
}
.h3xd-vol scale slider {
    min-width: 18px;
    min-height: 16px;
    margin: -5px -9px;
    border: none;
    border-radius: 0;
    box-shadow: none;
    background-color: transparent;
    background-image: url("file:///usr/share/h3xd/volume/knob.svg");
    background-size: 18px 16px;
    background-repeat: no-repeat;
    background-position: center;
}
.h3xd-vol scale slider:hover, .h3xd-vol scale slider:active {
    background-image: url("file:///usr/share/h3xd/volume/knob-active.svg");
}
.h3xd-vol scale.muted slider {
    background-image: url("file:///usr/share/h3xd/volume/knob-muted.svg");
}
.h3xd-vol scale marks indicator {
    min-height: 6px;
    min-width: 2px;
    background-color: #4b5366;
}
levelbar trough {
    min-height: 6px;
    padding: 0;
    border: none;
    border-radius: 3px;
    background-color: #10141b;
}
levelbar block.filled {
    border: none;
    border-radius: 3px;
    background-image: none;
    background-color: #1793d1;
}
levelbar block.empty {
    border: none;
    background-color: transparent;
}
.h3xd-vol-meter block.filled {
    background-color: #46c08a;
}
.h3xd-vol-meter.muted block.filled {
    background-color: #4b5366;
}
/* On-screen level of the volume keys. */
window.h3xd-osd-window {
    background-color: #1a202b;
}
window.h3xd-osd-window.composited {
    background-color: transparent;
}
.h3xd-osd {
    background-color: #1a202b;
    border-radius: 4px;
    padding: 14px 18px;
    color: #e6edf3;
    font-family: "IBM Plex Sans";
    font-size: 13px;
}
.h3xd-osd levelbar trough {
    background-color: #2b3240;
}
.h3xd-osd.muted levelbar block.filled {
    background-color: #4b5366;
}
.h3xd-osd-value {
    font-family: "JetBrains Mono";
    font-size: 13px;
    min-width: 52px;
}
.h3xd-osd-value.h3xd-osd-words {
    font-family: "IBM Plex Sans";
    font-weight: 500;
}
.h3xd-osd.muted .h3xd-osd-value {
    color: #f08a8e;
}
`

func installCSS() {
	provider, err := gtk.CssProviderNew()
	if err != nil {
		log.Fatalf("h3xd-volume: %v", err)
	}
	if err := provider.LoadFromData(css); err != nil {
		log.Fatalf("h3xd-volume: the stylesheet does not load: %v", err)
	}
	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		log.Fatalf("h3xd-volume: %v", err)
	}
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
