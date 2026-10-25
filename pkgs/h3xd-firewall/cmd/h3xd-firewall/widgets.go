package main

import (
	"log"
	"math"
	"strings"

	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

// check stops the app when GTK cannot create a widget, which only happens when GTK itself
// does not work.
func check[T any](widget T, err error) T {
	if err != nil {
		log.Fatalf("h3xd-firewall: %v", err)
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

// sentence starts a message from ufw or the checks with a capital, as the rest of the UI.
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

func chip(text, kind string) *gtk.Label {
	label := newLabel(text, "h3xd-chip", "h3xd-chip-"+kind)
	label.SetXAlign(0.5)
	label.SetVAlign(gtk.ALIGN_CENTER)
	return label
}

// newSwitch reports a change but leaves the switch as it is; the page draws it again from
// the firewall's answer, so it never shows a state the firewall does not have.
func newSwitch(on bool, onChange func(bool)) *gtk.Switch {
	toggle := check(gtk.SwitchNew())
	toggle.SetActive(on)
	toggle.SetVAlign(gtk.ALIGN_CENTER)
	toggle.Connect("state-set", func(_ *gtk.Switch, state bool) bool {
		onChange(state)
		return true
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
			if onChange != nil {
				onChange(value)
			}
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

// hexagon draws the state mark of the overview: a regular hexagon like the H3XD mark,
// filled with the accent and crossed by the dark bar of the logo when the firewall is on,
// an outline when it is off.
func hexagon(on bool, size int) *gtk.DrawingArea {
	area := check(gtk.DrawingAreaNew())
	area.SetSizeRequest(size, size)
	area.SetVAlign(gtk.ALIGN_CENTER)
	area.Connect("draw", func(_ *gtk.DrawingArea, cr *cairo.Context) bool {
		const lineWidth = 3
		center := float64(size) / 2
		radius := center - lineWidth
		half := radius * math.Sqrt(3) / 2
		cr.MoveTo(center, center-radius)
		cr.LineTo(center+half, center-radius/2)
		cr.LineTo(center+half, center+radius/2)
		cr.LineTo(center, center+radius)
		cr.LineTo(center-half, center+radius/2)
		cr.LineTo(center-half, center-radius/2)
		cr.ClosePath()
		bar := radius / 4
		if on {
			cr.SetSourceRGB(0x17/255.0, 0x93/255.0, 0xd1/255.0)
			cr.Fill()
			cr.SetSourceRGB(0x0f/255.0, 0x13/255.0, 0x20/255.0)
		} else {
			cr.SetSourceRGB(0x4b/255.0, 0x53/255.0, 0x66/255.0)
			cr.SetLineWidth(lineWidth)
			cr.Stroke()
		}
		cr.Rectangle(center-half/1.6, center-bar/2, 2*half/1.6, bar)
		cr.Fill()
		return false
	})
	return area
}
