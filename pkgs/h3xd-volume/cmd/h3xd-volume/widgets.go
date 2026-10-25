package main

import (
	"log"
	"strings"

	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

// check stops the app when GTK cannot create a widget, which only happens when GTK itself
// does not work.
func check[T any](widget T, err error) T {
	if err != nil {
		log.Fatalf("h3xd-volume: %v", err)
	}
	return widget
}

func styled[T gtk.IWidget](widget T, classes ...string) T {
	context := check(widget.ToWidget().GetStyleContext())
	for _, class := range classes {
		context.AddClass(class)
	}
	return widget
}

func setClass(widget gtk.IWidget, class string, on bool) {
	context := check(widget.ToWidget().GetStyleContext())
	if on {
		context.AddClass(class)
	} else {
		context.RemoveClass(class)
	}
}

func newLabel(text string, classes ...string) *gtk.Label {
	label := check(gtk.LabelNew(text))
	label.SetXAlign(0)
	return styled(label, classes...)
}

// newText is a label that wraps instead of widening the window.
func newText(text string, classes ...string) *gtk.Label {
	label := newLabel(text, classes...)
	label.SetLineWrap(true)
	label.SetLineWrapMode(pango.WRAP_WORD_CHAR)
	return label
}

// newName is a label for names of devices and songs, which can be long: it ends in an
// ellipsis instead of widening the window.
func newName(text string, classes ...string) *gtk.Label {
	label := newLabel(text, classes...)
	label.SetEllipsize(pango.ELLIPSIZE_END)
	label.SetHExpand(true)
	return label
}

// sentence starts a message from pactl with a capital, as the rest of the UI.
func sentence(message string) string {
	if message == "" {
		return message
	}
	return strings.ToUpper(message[:1]) + message[1:]
}

func newBox(vertical bool, spacing int, classes ...string) *gtk.Box {
	orientation := gtk.ORIENTATION_HORIZONTAL
	if vertical {
		orientation = gtk.ORIENTATION_VERTICAL
	}
	return styled(check(gtk.BoxNew(orientation, spacing)), classes...)
}

func newButton(text string, onClick func(), classes ...string) *gtk.Button {
	button := styled(check(gtk.ButtonNewWithLabel(text)), append([]string{"h3xd-button"}, classes...)...)
	button.SetVAlign(gtk.ALIGN_CENTER)
	button.Connect("clicked", onClick)
	return button
}

func chip(text string) *gtk.Label {
	label := newLabel(text, "h3xd-chip")
	label.SetXAlign(0.5)
	label.SetVAlign(gtk.ALIGN_CENTER)
	return label
}

func card(spacing int) *gtk.Box {
	return newBox(true, spacing, "h3xd-vol-card")
}

func heading(box *gtk.Box, text string) {
	box.PackStart(newLabel(text, "h3xd-vol-heading"), false, false, 0)
}

// newSwitch reports a change and shows it at once; settings of the window itself have no
// server that could refuse them.
func newSwitch(on bool, onChange func(bool)) *gtk.Switch {
	toggle := check(gtk.SwitchNew())
	toggle.SetActive(on)
	toggle.SetVAlign(gtk.ALIGN_CENTER)
	toggle.Connect("state-set", func(_ *gtk.Switch, state bool) bool {
		onChange(state)
		return false
	})
	return toggle
}

// segmented is a row of joined buttons of which one is selected.
type segmented struct {
	box      *gtk.Box
	buttons  []*gtk.Button
	values   []string
	selected string
}

func newSegmented(labels, values []string, selected string, onChange func(string)) *segmented {
	s := &segmented{box: newBox(false, 0, "h3xd-seg", "linked"), values: values,
		selected: selected}
	s.box.SetHAlign(gtk.ALIGN_START)
	for i := range labels {
		value := values[i]
		button := newButton(labels[i], func() {
			if s.selected == value {
				return
			}
			s.selected = value
			s.mark()
			onChange(value)
		})
		s.buttons = append(s.buttons, button)
		s.box.PackStart(button, false, false, 0)
	}
	s.mark()
	return s
}

func (s *segmented) mark() {
	for i, button := range s.buttons {
		setClass(button, "selected", s.values[i] == s.selected)
	}
}

// choice is an item of a drop-down list.
type choice struct{ id, text string }

// dropDown lists choices; setting the selection from the server does not report a change.
type dropDown struct {
	combo    *gtk.ComboBoxText
	updating bool
}

func newDropDown(choices []choice, active string, onChange func(string)) *dropDown {
	d := &dropDown{combo: check(gtk.ComboBoxTextNew())}
	for _, item := range choices {
		d.combo.Append(item.id, item.text)
	}
	d.combo.SetActiveID(active)
	d.combo.SetVAlign(gtk.ALIGN_CENTER)
	d.combo.Connect("changed", func() {
		if !d.updating {
			onChange(d.combo.GetActiveID())
		}
	})
	return d
}

func (d *dropDown) set(active string) {
	if d.combo.GetActiveID() == active {
		return
	}
	d.updating = true
	d.combo.SetActiveID(active)
	d.updating = false
}
