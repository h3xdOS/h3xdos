package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"

	"github.com/h3xdOS/h3xdos/firewall/internal/protocol"
	"github.com/h3xdOS/h3xdos/firewall/internal/ufw"
)

const (
	heroMark    = 72
	indexWidth  = 3
	maxActivity = 100
)

// policyWords are the plain words for ufw's default policies.
var policyWords = map[string]string{"allow": "Allowed", "deny": "Blocked", "reject": "Refused"}

func card(spacing int) *gtk.Box {
	return newBox(true, spacing, "h3xd-fw-card")
}

func heading(box *gtk.Box, text string) {
	box.PackStart(newLabel(text, "h3xd-fw-heading"), false, false, 0)
}

func (a *app) lockedPage(box *gtk.Box) {
	panel := card(12)
	if a.starting {
		panel.PackStart(newText("Waiting for the password in the authentication window.",
			"h3xd-fw-big"), false, false, 0)
	} else {
		panel.PackStart(newText("H3XD Firewall needs your password to read and change the firewall.",
			"h3xd-fw-big"), false, false, 0)
		panel.PackStart(newText("The password opens it until you close the window.",
			"h3xd-muted"), false, false, 0)
		unlock := newButton("Unlock", a.unlock, "h3xd-primary")
		unlock.SetHAlign(gtk.ALIGN_START)
		panel.PackStart(unlock, false, false, 0)
	}
	box.PackStart(panel, false, false, 0)
}

func (a *app) overviewPage(box *gtk.Box) {
	status := a.status
	hero := newBox(false, 22, "h3xd-fw-card")
	hero.PackStart(hexagon(status.Enabled, heroMark), false, false, 0)
	text := newBox(true, 6)
	text.SetVAlign(gtk.ALIGN_CENTER)
	var title, summary string
	if status.Enabled {
		title = "Firewall is on"
		summary = "Connections from other devices are " + strings.ToLower(policyWords[status.Incoming]) +
			" unless a rule or a service lets them in."
		if !status.Boot {
			summary += " It is not set to start with the computer."
		}
	} else {
		title = "Firewall is off"
		summary = "Every connection gets through. Turn the firewall on to keep other devices " +
			"out of the services on this computer."
	}
	text.PackStart(newLabel(title, "h3xd-fw-hero-title"), false, false, 0)
	text.PackStart(newText(summary, "h3xd-muted"), false, false, 0)
	hero.PackStart(text, true, true, 0)
	if status.Enabled {
		hero.PackEnd(newButton("Turn off", func() {
			a.request(protocol.Request{Op: protocol.OpDisable}, nil)
		}), false, false, 0)
	} else {
		hero.PackEnd(newButton("Turn on", func() {
			a.request(protocol.Request{Op: protocol.OpEnable}, nil)
		}, "h3xd-primary"), false, false, 0)
	}
	box.PackStart(hero, false, false, 0)

	tiles := newBox(false, 14)
	tile := func(name, value, detail string) {
		panel := card(4)
		panel.PackStart(newLabel(name, "h3xd-fw-label"), false, false, 0)
		panel.PackStart(newLabel(value, "h3xd-fw-big"), false, false, 0)
		panel.PackStart(newText(detail, "h3xd-fw-meta"), false, false, 0)
		tiles.PackStart(panel, true, true, 0)
	}
	tile("Incoming", policyWords[status.Incoming], "without a rule")
	tile("Outgoing", policyWords[status.Outgoing], "without a rule")
	tile("Rules", strconv.Itoa(len(status.Rules)),
		fmt.Sprintf("%d switched off", len(status.Disabled)))
	box.PackStart(tiles, false, false, 0)

	heading(box, "Services other devices can reach")
	var enabled []string
	for _, preset := range ufw.Presets {
		if preset.Enabled(status.Rules) {
			enabled = append(enabled, preset.Name)
		}
	}
	services := newBox(false, 8)
	if len(enabled) == 0 {
		services.PackStart(newLabel("None of the listed services.", "h3xd-muted"), false, false, 0)
	}
	for _, name := range enabled {
		services.PackStart(chip(name, "allow"), false, false, 0)
	}
	services.PackEnd(newButton("Choose services", func() { a.navigate(pageServices) }), false, false, 0)
	box.PackStart(services, false, false, 0)
}

func actionChip(rule ufw.Rule) *gtk.Label {
	if rule.Unsupported {
		return chip("RULE", "neutral")
	}
	return chip(strings.ToUpper(rule.Action), rule.Action)
}

func (a *app) ruleRow(index int, rule ufw.Rule, on bool) *gtk.Box {
	row := newBox(false, 14, "h3xd-fw-row")
	setClass(row, "off", !on)
	number := "-"
	if on {
		number = strconv.Itoa(index)
	}
	position := newLabel(number, "h3xd-fw-meta")
	position.SetWidthChars(indexWidth)
	row.PackStart(position, false, false, 0)
	row.PackStart(actionChip(rule), false, false, 0)
	if !rule.Unsupported {
		direction := chip(map[string]string{"in": "IN", "out": "OUT"}[rule.Direction], "neutral")
		row.PackStart(direction, false, false, 0)
	}
	text := newBox(true, 2)
	text.SetVAlign(gtk.ALIGN_CENTER)
	target := newLabel(rule.Target(), "h3xd-fw-target")
	target.SetEllipsize(pango.ELLIPSIZE_END)
	text.PackStart(target, false, false, 0)
	var details []string
	if rule.Comment != "" {
		details = append(details, rule.Comment)
	}
	if rule.Log != "" {
		details = append(details, map[string]string{"log": "logs new connections",
			"log-all": "logs every packet"}[rule.Log])
	}
	if !on {
		details = append(details, "switched off")
	}
	if len(details) > 0 {
		text.PackStart(newLabel(strings.Join(details, "  ·  "), "h3xd-fw-meta"), false, false, 0)
	}
	row.PackStart(text, true, true, 0)
	row.PackEnd(newButton("Delete", func() {
		if a.confirm("Delete this rule?", rule.Target(), "Delete") {
			a.request(protocol.Request{Op: protocol.OpDelete, Rule: rule}, nil)
		}
	}, "h3xd-danger"), false, false, 0)
	if !rule.Unsupported {
		row.PackEnd(newSwitch(on, func(state bool) {
			a.request(protocol.Request{Op: protocol.OpSetRule, Rule: rule, On: state}, nil)
		}), false, false, 0)
	}
	return row
}

func (a *app) rulesPage(box *gtk.Box) {
	toolbar := newBox(false, 12)
	toolbar.PackStart(newButton("Add rule", func() { a.addRuleDialog(ufw.Rule{}) }, "h3xd-primary"),
		false, false, 0)
	if !a.status.Enabled {
		toolbar.PackStart(newLabel("The firewall is off, so no rule applies now.", "h3xd-muted"),
			false, false, 0)
	}
	box.PackStart(toolbar, false, false, 0)
	if len(a.status.Rules)+len(a.status.Disabled) == 0 {
		box.PackStart(newText("No rules yet. Connections follow the defaults in Settings.",
			"h3xd-muted"), false, false, 0)
		return
	}
	list := newBox(true, 8)
	for i, rule := range a.status.Rules {
		list.PackStart(a.ruleRow(i+1, rule, true), false, false, 0)
	}
	if len(a.status.Disabled) > 0 {
		heading(list, "Switched off")
		for _, rule := range a.status.Disabled {
			list.PackStart(a.ruleRow(0, rule, false), false, false, 0)
		}
	}
	box.PackStart(list, false, false, 0)
}

func presetPorts(preset ufw.Preset) string {
	var ports []string
	for _, rule := range preset.Rules {
		ports = append(ports, rule.Port+"/"+rule.Proto)
	}
	return strings.Join(ports, "  ")
}

func (a *app) servicesPage(box *gtk.Box) {
	list := newBox(true, 8)
	for _, preset := range ufw.Presets {
		id := preset.ID
		row := newBox(false, 14, "h3xd-fw-row")
		text := newBox(true, 2)
		text.PackStart(newLabel(preset.Name, "h3xd-fw-target"), false, false, 0)
		text.PackStart(newText(preset.Summary, "h3xd-muted"), false, false, 0)
		text.PackStart(newLabel(presetPorts(preset), "h3xd-fw-meta"), false, false, 0)
		row.PackStart(text, true, true, 0)
		row.PackEnd(newSwitch(preset.Enabled(a.status.Rules), func(state bool) {
			a.request(protocol.Request{Op: protocol.OpPreset, Preset: id, On: state}, nil)
		}), false, false, 0)
		list.PackStart(row, false, false, 0)
	}
	box.PackStart(list, false, false, 0)

	header := newBox(false, 12)
	heading(header, "Programs waiting for connections")
	header.PackEnd(newButton("Refresh", func() {
		a.request(protocol.Request{Op: protocol.OpListening}, nil)
	}), false, false, 0)
	box.PackStart(header, false, false, 0)
	if len(a.listening) == 0 {
		box.PackStart(newLabel("No program waits for connections.", "h3xd-muted"), false, false, 0)
		return
	}
	// Programs other devices can reach come first, then by port.
	listening := append([]ufw.Listener(nil), a.listening...)
	sort.SliceStable(listening, func(i, j int) bool {
		if listening[i].LocalOnly != listening[j].LocalOnly {
			return !listening[i].LocalOnly
		}
		if listening[i].Port != listening[j].Port {
			return listening[i].Port < listening[j].Port
		}
		return listening[i].Proto < listening[j].Proto
	})
	programs := newBox(true, 8)
	for _, listener := range listening {
		item := listener
		row := newBox(false, 14, "h3xd-fw-row")
		row.PackStart(newLabel(item.Process, "h3xd-fw-target"), false, false, 0)
		row.PackStart(newLabel(fmt.Sprintf("%d/%s", item.Port, item.Proto), "h3xd-fw-mono"),
			true, true, 0)
		allowed := ufw.Allows(a.status.Rules, item.Proto, item.Port)
		switch {
		case item.LocalOnly:
			row.PackEnd(chip("THIS COMPUTER ONLY", "neutral"), false, false, 0)
		case allowed:
			row.PackEnd(chip("REACHABLE", "allow"), false, false, 0)
		case !a.status.Enabled || a.status.Incoming == "allow":
			row.PackEnd(chip("REACHABLE", "reject"), false, false, 0)
		default:
			row.PackEnd(newButton("Allow", func() {
				a.addRuleDialog(ufw.Rule{Action: "allow", Direction: "in", Proto: item.Proto,
					Port: strconv.Itoa(item.Port), Comment: item.Process})
			}), false, false, 0)
			row.PackEnd(chip("BLOCKED", "deny"), false, false, 0)
		}
		programs.PackStart(row, false, false, 0)
	}
	box.PackStart(programs, false, false, 0)
}

func (a *app) activityPage(box *gtk.Box) {
	toolbar := newBox(false, 12)
	if a.status.Logging == "off" {
		toolbar.PackStart(newText("Logging is off, so blocked connections are not recorded.",
			"h3xd-muted"), true, true, 0)
		toolbar.PackEnd(newButton("Turn on logging", func() {
			a.request(protocol.Request{Op: protocol.OpLogging, Level: "low"}, func(protocol.Response) {
				a.request(protocol.Request{Op: protocol.OpBlocked}, nil)
			})
		}, "h3xd-primary"), false, false, 0)
	} else {
		toolbar.PackStart(newLabel("Logging: "+a.status.Logging, "h3xd-fw-meta"), true, true, 0)
		toolbar.PackEnd(newButton("Refresh", func() {
			a.request(protocol.Request{Op: protocol.OpBlocked}, nil)
		}), false, false, 0)
	}
	box.PackStart(toolbar, false, false, 0)
	if len(a.blocked) == 0 {
		box.PackStart(newLabel("Nothing blocked recently.", "h3xd-muted"), false, false, 0)
		return
	}
	list := newBox(true, 8)
	shown := 0
	for i := len(a.blocked) - 1; i >= 0 && shown < maxActivity; i-- {
		entry := a.blocked[i]
		shown++
		row := newBox(false, 14, "h3xd-fw-row")
		row.PackStart(newLabel(entry.Time.Local().Format("Jan 2 15:04:05"), "h3xd-fw-meta"),
			false, false, 0)
		direction := "IN"
		if !entry.Incoming {
			direction = "OUT"
		}
		row.PackStart(chip(direction, "neutral"), false, false, 0)
		what := fmt.Sprintf("%s to %s", entry.Source, entry.Dest)
		if entry.Port > 0 {
			what = fmt.Sprintf("%s to port %d/%s", entry.Source, entry.Port, entry.Proto)
		}
		target := newLabel(what, "h3xd-fw-target")
		target.SetEllipsize(pango.ELLIPSIZE_END)
		row.PackStart(target, true, true, 0)
		row.PackStart(newLabel(entry.Iface, "h3xd-fw-meta"), false, false, 0)
		if entry.Incoming && entry.Port > 0 && (entry.Proto == "tcp" || entry.Proto == "udp") {
			row.PackEnd(newButton("Allow", func() {
				a.addRuleDialog(ufw.Rule{Action: "allow", Direction: "in", Proto: entry.Proto,
					Port: strconv.Itoa(entry.Port)})
			}), false, false, 0)
		}
		list.PackStart(row, false, false, 0)
	}
	box.PackStart(list, false, false, 0)
}

func (a *app) settingRow(box *gtk.Box, name, help string, control gtk.IWidget) {
	row := newBox(false, 18, "h3xd-fw-row")
	text := newBox(true, 2)
	text.PackStart(newLabel(name, "h3xd-fw-target"), false, false, 0)
	text.PackStart(newText(help, "h3xd-muted"), false, false, 0)
	row.PackStart(text, true, true, 0)
	row.PackEnd(control, false, false, 0)
	box.PackStart(row, false, false, 0)
}

func (a *app) policyControl(direction, current string, choices []string) gtk.IWidget {
	labels := make([]string, len(choices))
	for i, choice := range choices {
		labels[i] = map[string]string{"allow": "Allow", "deny": "Block", "reject": "Refuse"}[choice]
	}
	control := newSegmented(labels, choices, current, func(policy string) {
		a.request(protocol.Request{Op: protocol.OpDefault, Direction: direction, Policy: policy}, nil)
	})
	control.box.SetVAlign(gtk.ALIGN_CENTER)
	return control.box
}

func (a *app) settingsPage(box *gtk.Box) {
	status := a.status
	heading(box, "When no rule matches")
	a.settingRow(box, "Incoming",
		"Connections other devices open to this computer. Block drops them silently, Refuse "+
			"answers that the port is closed.",
		a.policyControl("incoming", status.Incoming, []string{"deny", "reject", "allow"}))
	a.settingRow(box, "Outgoing", "Connections this computer opens to others.",
		a.policyControl("outgoing", status.Outgoing, []string{"allow", "deny", "reject"}))
	a.settingRow(box, "Routed", "Traffic this computer forwards between networks, for example "+
		"for virtual machines and containers.",
		a.policyControl("routed", status.Routed, []string{"deny", "reject", "allow"}))

	heading(box, "Logging")
	levels := []string{"off", "low", "medium", "high", "full"}
	logging := newSegmented([]string{"Off", "Low", "Medium", "High", "Full"}, levels, status.Logging,
		func(level string) {
			a.request(protocol.Request{Op: protocol.OpLogging, Level: level}, nil)
		})
	logging.box.SetVAlign(gtk.ALIGN_CENTER)
	a.settingRow(box, "Blocked connections in the log",
		"Low records blocked connections and is enough for Activity; higher levels record "+
			"more and fill the log faster.", logging.box)

	heading(box, "Start")
	boot := "The firewall starts with the computer."
	if !status.Boot {
		boot = "The firewall does not start with the computer; turning it on sets that up."
	}
	a.settingRow(box, "At startup", boot, newLabel(map[bool]string{true: "ON", false: "OFF"}[status.Boot],
		"h3xd-fw-meta"))

	heading(box, "Reset")
	a.settingRow(box, "Reset the firewall",
		"Removes every rule, switches the firewall off and restores the defaults. ufw keeps "+
			"a copy of the old rules in /etc/ufw.",
		newButton("Reset", func() {
			if a.confirm("Reset the firewall?",
				"Every rule is removed and the firewall is switched off.", "Reset") {
				a.request(protocol.Request{Op: protocol.OpReset}, nil)
			}
		}, "h3xd-danger-fill"))
}
