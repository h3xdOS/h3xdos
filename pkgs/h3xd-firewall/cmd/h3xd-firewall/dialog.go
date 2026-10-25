package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gotk3/gotk3/gtk"

	"github.com/h3xdOS/h3xdos/firewall/internal/protocol"
	"github.com/h3xdOS/h3xdos/firewall/internal/ufw"
)

const (
	dialogWidth = 560
	anyIface    = "Any"
	netClassDir = "/sys/class/net"
)

func entryText(entry *gtk.Entry) string {
	return strings.TrimSpace(check(entry.GetText()))
}

func newEntry(text, placeholder string) *gtk.Entry {
	entry := check(gtk.EntryNew())
	entry.SetText(text)
	entry.SetPlaceholderText(placeholder)
	return entry
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// interfaces lists the network interfaces for the advanced part of the rule dialog.
func interfaces() []string {
	entries, err := os.ReadDir(netClassDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "h3xd-firewall: %v\n", err)
		return nil
	}
	var names []string
	for _, entry := range entries {
		if entry.Name() != "lo" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}

// addRuleDialog asks for a new rule, prefilled for example from a blocked connection; the
// rule is checked here as well as in the helper, so mistakes show next to the fields.
func (a *app) addRuleDialog(prefill ufw.Rule) {
	dialog := styled(check(gtk.DialogNew()), "h3xd-fw")
	dialog.SetTitle("Add rule")
	dialog.SetTransientFor(a.window)
	dialog.SetModal(true)
	dialog.SetDefaultSize(dialogWidth, -1)
	area := styled(check(dialog.GetContentArea()), "h3xd-fw-dialog")
	area.SetSpacing(14)

	field := func(box *gtk.Box, name string, widget gtk.IWidget, help string) {
		group := newBox(true, 6)
		group.PackStart(newLabel(name, "h3xd-fw-label"), false, false, 0)
		group.PackStart(widget, false, false, 0)
		if help != "" {
			group.PackStart(newText(help, "h3xd-fw-meta"), false, false, 0)
		}
		box.PackStart(group, false, false, 0)
	}
	action := newSegmented([]string{"Allow", "Deny", "Reject", "Limit"}, ufw.Actions,
		orDefault(prefill.Action, "allow"), nil)
	field(area, "Action", action.box,
		"Deny drops silently, Reject answers that the port is closed, Limit allows but blocks an "+
			"address that connects six times in 30 seconds.")
	direction := newSegmented([]string{"Incoming", "Outgoing"}, ufw.Directions,
		orDefault(prefill.Direction, "in"), nil)
	field(area, "Direction", direction.box, "")
	port := newEntry(prefill.Port, "22, 80,443 or 6000:6007; empty for every port")
	field(area, "Port", port, "")
	proto := newSegmented([]string{"Any", "TCP", "UDP"}, ufw.Protocols, prefill.Proto, nil)
	field(area, "Protocol", proto.box, "Several ports or a range need TCP or UDP.")
	from := newEntry(prefill.From, "Any address, or for example 192.168.1.0/24")
	field(area, "From", from, "")
	comment := newEntry(prefill.Comment, "What the rule is for")
	field(area, "Comment", comment, "")

	advanced := check(gtk.ExpanderNew("Advanced"))
	more := newBox(true, 14)
	more.SetMarginTop(12)
	to := newEntry(prefill.To, "Any address of this computer")
	field(more, "To", to, "")
	iface := check(gtk.ComboBoxTextNew())
	iface.AppendText(anyIface)
	iface.SetActive(0)
	for i, name := range interfaces() {
		iface.AppendText(name)
		if name == prefill.Interface {
			iface.SetActive(i + 1)
		}
	}
	field(more, "Network interface", iface, "")
	logMode := newSegmented([]string{"Off", "New connections", "Every packet"}, ufw.LogModes,
		prefill.Log, nil)
	field(more, "Log", logMode.box, "")
	position := newSegmented([]string{"At the end", "At the top"}, []string{"end", "top"}, "end", nil)
	field(more, "Position", position.box,
		"Rules are checked from the top; put a rule at the top to make it win over others.")
	advanced.Add(more)
	advanced.SetExpanded(prefill.To != "" || prefill.Interface != "" || prefill.Log != "")
	area.PackStart(advanced, false, false, 0)

	problem := newText("", "h3xd-error")
	area.PackStart(problem, false, false, 0)
	styled(check(dialog.AddButton("Cancel", gtk.RESPONSE_CANCEL)), "h3xd-button")
	styled(check(dialog.AddButton("Add rule", gtk.RESPONSE_OK)), "h3xd-button", "h3xd-primary")
	dialog.ShowAll()

	for dialog.Run() == gtk.RESPONSE_OK {
		chosenIface := iface.GetActiveText()
		if chosenIface == anyIface {
			chosenIface = ""
		}
		rule := ufw.Rule{Action: action.selected, Direction: direction.selected,
			Proto: proto.selected, Port: entryText(port), From: entryText(from), To: entryText(to),
			Interface: chosenIface, Log: logMode.selected, Comment: entryText(comment)}
		if err := rule.Validate(); err != nil {
			problem.SetText(sentence(err.Error()))
			continue
		}
		a.request(protocol.Request{Op: protocol.OpAdd, Rule: rule, AtTop: position.selected == "top"},
			nil)
		break
	}
	dialog.Destroy()
}

// confirm asks before a change that cannot be taken back with a click.
func (a *app) confirm(question, detail, action string) bool {
	dialog := styled(gtk.MessageDialogNew(a.window, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION,
		gtk.BUTTONS_NONE, "%s", question), "h3xd-fw")
	dialog.FormatSecondaryText("%s", detail)
	styled(check(dialog.AddButton("Cancel", gtk.RESPONSE_CANCEL)), "h3xd-button")
	styled(check(dialog.AddButton(action, gtk.RESPONSE_ACCEPT)), "h3xd-button", "h3xd-danger-fill")
	response := dialog.Run()
	dialog.Destroy()
	return response == gtk.RESPONSE_ACCEPT
}
