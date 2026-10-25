package main

import (
	"log"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

// css follows sheet 2f of design v2 and the layout of the settings window (2d), as the
// other H3XD windows do: navigation on the left, cards with 4 px corners, IBM Plex Sans for
// text, JetBrains Mono for ports, addresses and states.
const css = `
.h3xd-fw {
    background-color: #181c25;
    color: #e6edf3;
    font-family: "IBM Plex Sans";
    font-size: 13px;
}
.h3xd-fw-side {
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
.h3xd-fw-main {
    padding: 36px 0 0 48px;
}
.h3xd-fw-header, .h3xd-fw-subtitle, .h3xd-fw-content {
    margin-right: 48px;
}
.h3xd-fw-title {
    font-size: 26px;
    font-weight: 600;
}
.h3xd-fw-subtitle {
    margin-top: 6px;
    margin-bottom: 24px;
}
.h3xd-fw-content {
    margin-bottom: 36px;
}
.h3xd-fw-heading {
    font-size: 14px;
    font-weight: 600;
    margin-top: 10px;
}
.h3xd-muted {
    color: #8b95a5;
}
.h3xd-fw-card {
    background-color: #1a202b;
    border-radius: 4px;
    padding: 16px 18px;
}
/* Rows keep one height whether they end in a button, a switch or a chip. */
.h3xd-fw-row {
    background-color: #1a202b;
    border-radius: 4px;
    padding: 12px 16px;
    min-height: 36px;
}
.h3xd-fw-row.off {
    background-color: #151922;
}
.h3xd-fw-row.off .h3xd-fw-target {
    color: #8b95a5;
}
.h3xd-fw-hero-title {
    font-size: 22px;
    font-weight: 600;
}
.h3xd-fw-big {
    font-size: 18px;
    font-weight: 600;
}
.h3xd-fw-target {
    font-size: 14px;
    font-weight: 500;
}
.h3xd-fw-mono {
    font-family: "JetBrains Mono";
    font-size: 12px;
}
.h3xd-fw-meta {
    font-family: "JetBrains Mono";
    font-size: 11px;
    color: #8b95a5;
}
.h3xd-chip {
    font-family: "JetBrains Mono";
    font-size: 11px;
    font-weight: 600;
    border-radius: 3px;
    padding: 3px 8px;
}
.h3xd-chip-allow { background-color: #16301f; color: #6fd6a6; }
.h3xd-chip-limit { background-color: #12304a; color: #5cb6e4; }
.h3xd-chip-deny { background-color: #3a1d22; color: #f08a8e; }
.h3xd-chip-reject { background-color: #33291a; color: #efcb83; }
.h3xd-chip-neutral { background-color: #2b3240; color: #c6cbd5; }
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
.h3xd-error {
    color: #f07075;
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
button.h3xd-button:disabled {
    background-color: #1f2430;
    color: #8b95a5;
}
button.h3xd-primary {
    background-color: #1793d1;
    color: #0f1320;
    font-weight: 600;
}
button.h3xd-primary:hover {
    background-color: #5cb6e4;
}
button.h3xd-danger {
    background-color: transparent;
    color: #f07075;
}
button.h3xd-danger:hover {
    background-color: #3a1d22;
}
button.h3xd-danger-fill {
    background-color: #e5484d;
    color: #14080a;
    font-weight: 600;
}
button.h3xd-danger-fill:hover {
    background-color: #f07075;
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
.h3xd-fw entry {
    background-image: none;
    background-color: #10141b;
    border: 2px solid #2b3240;
    border-radius: 4px;
    color: #e6edf3;
    caret-color: #1793d1;
    padding: 8px 12px;
    box-shadow: none;
}
.h3xd-fw entry:focus {
    border-color: #1793d1;
}
.h3xd-fw-dialog {
    padding: 22px 24px 8px 24px;
}
.h3xd-fw-label {
    color: #8b95a5;
    font-size: 12px;
}
`

func installCSS() {
	provider, err := gtk.CssProviderNew()
	if err != nil {
		log.Fatalf("h3xd-firewall: %v", err)
	}
	if err := provider.LoadFromData(css); err != nil {
		log.Fatalf("h3xd-firewall: the stylesheet does not load: %v", err)
	}
	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		log.Fatalf("h3xd-firewall: %v", err)
	}
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
