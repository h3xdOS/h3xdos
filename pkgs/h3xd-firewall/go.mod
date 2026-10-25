module github.com/h3xdOS/h3xdos/firewall

go 1.23

// v0.6.4, the last tag, does not build: gdk/gdk_since_3_22.go lacks an import. The commit
// of 2025-11-24 adds it and changes nothing else that H3XD Firewall uses.
require github.com/gotk3/gotk3 v0.6.5-0.20251124190141-e7a9e823ca35
