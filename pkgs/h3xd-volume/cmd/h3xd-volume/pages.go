package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"github.com/h3xdOS/h3xdos/volume/internal/pulse"
)

const (
	labelWidth = 96
	// sliderInset is the side padding of sliders in css: the meter starts where the
	// trough of the volume slider above it starts.
	sliderInset    = 9
	streamIconSize = 32
	// meterDecay is how far the meter falls per reading (20 a second): from full to
	// silence in about a second, slow enough for the eye to follow.
	meterDecay = 0.05
)

// words are the texts that differ between the output and the input page.
type words struct{ device, others, none, level string }

func wordsFor(kind pulse.Kind) words {
	if kind == pulse.Input {
		return words{"Input device", "Other inputs",
			"No microphone or other input was found. Check that it is plugged in, or turn its " +
				"input on under Devices.", "Level"}
	}
	return words{"Output device", "Other outputs",
		"No speakers or headphones were found. Check that they are plugged in, or turn the " +
			"sound card on under Devices.", "Level"}
}

// pageShape describes everything that the widgets of the page are built from, apart from
// values that the updaters change in place: volumes, mute, the selected device of a
// program and the title of what it plays.
func (a *app) pageShape() string {
	var shape strings.Builder
	fmt.Fprintf(&shape, "%s|%t|%t|%d|", a.page, a.loaded, a.serverError != nil, a.limit())
	switch a.page {
	case pageOutput, pageInput:
		kind := pulse.Output
		if a.page == pageInput {
			kind = pulse.Input
		}
		main, _ := a.mainDevice(kind)
		fmt.Fprintf(&shape, "main=%s|", main.Name)
		for _, device := range a.visibleDevices(kind) {
			fmt.Fprintf(&shape, "%s/%s/%s/%t;", device.Name, device.Description, device.ActivePort,
				pulse.HasBalance(device.Channels))
			for _, port := range device.Ports {
				fmt.Fprintf(&shape, "%s/%s/%t,", port.Name, port.Description, port.Available)
			}
		}
	case pageApps:
		for _, device := range append(append([]pulse.Device{}, a.state.Outputs...), a.state.Inputs...) {
			fmt.Fprintf(&shape, "%d/%s/%s;", device.Kind, device.Name, device.Description)
		}
		for _, stream := range append(append([]pulse.Stream{}, a.state.Playback...), a.state.Recording...) {
			fmt.Fprintf(&shape, "%d/%d/%s/%s;", stream.Kind, stream.Index, stream.App, stream.Icon)
		}
	case pageDevices:
		for _, card := range a.state.Cards {
			fmt.Fprintf(&shape, "%s/%s/%s;", card.Name, card.Description, card.ActiveProfile)
			for _, profile := range card.Profiles {
				fmt.Fprintf(&shape, "%s/%s/%t,", profile.Name, profile.Description, profile.Available)
			}
		}
	}
	return shape.String()
}

// visibleDevices leaves out the monitors, which record what an output plays and are no
// microphones; programs can still record from them (Applications).
func (a *app) visibleDevices(kind pulse.Kind) []pulse.Device {
	var visible []pulse.Device
	for _, device := range a.state.Devices(kind) {
		if !device.Monitor {
			visible = append(visible, device)
		}
	}
	return visible
}

// mainDevice is the default device, or the first one while the server has no default.
func (a *app) mainDevice(kind pulse.Kind) (pulse.Device, bool) {
	if device, ok := a.state.Default(kind); ok && !device.Monitor {
		return device, true
	}
	if visible := a.visibleDevices(kind); len(visible) > 0 {
		return visible[0], true
	}
	return pulse.Device{}, false
}

// labeled puts a field name in front of a control, in one column for the whole card.
func labeled(name string, control gtk.IWidget) *gtk.Box {
	row := newBox(false, 12)
	label := newLabel(name, "h3xd-vol-label")
	label.SetSizeRequest(labelWidth, -1)
	label.SetVAlign(gtk.ALIGN_CENTER)
	row.PackStart(label, false, false, 0)
	row.PackStart(control, true, true, 0)
	return row
}

func (a *app) noServerPage(box *gtk.Box) {
	panel := card(10)
	panel.PackStart(newText("The sound server is not running.", "h3xd-vol-big"), false, false, 0)
	panel.PackStart(newText("H3XD plays sound through PipeWire, which starts when you log in. "+
		"Logging out and in again usually brings it back; this window connects as soon as it runs.",
		"h3xd-muted"), false, false, 0)
	box.PackStart(panel, false, false, 0)
}

func (a *app) deviceControl(device pulse.Device) *volumeControl {
	kind, name := device.Kind, device.Name
	control := a.newVolumeControl(device.Channels, device.Mute,
		func(levels []pulse.Channel) error { return pulse.SetDeviceVolume(kind, name, levels) },
		func(mute bool) error { return pulse.SetDeviceMute(kind, name, mute) }, name)
	a.updaters = append(a.updaters, func() {
		if fresh, ok := a.state.Device(kind, name); ok {
			control.update(fresh.Channels, fresh.Mute)
		}
	})
	return control
}

func (a *app) setDefault(kind pulse.Kind, name string) {
	a.setter.do(fmt.Sprintf("default/%d", kind), func() error { return pulse.SetDefault(kind, name) })
}

// devicePage is the output or the input page: the default device with its volume, level,
// connector and balance, then the other devices.
func (a *app) devicePage(box *gtk.Box, kind pulse.Kind) {
	text := wordsFor(kind)
	main, ok := a.mainDevice(kind)
	if !ok {
		panel := card(8)
		panel.PackStart(newText(text.none, "h3xd-vol-big"), false, false, 0)
		box.PackStart(panel, false, false, 0)
		return
	}
	visible := a.visibleDevices(kind)
	hero := card(14)
	hero.PackStart(newLabel(text.device, "h3xd-vol-label"), false, false, 0)
	if len(visible) > 1 {
		var choices []choice
		for _, device := range visible {
			choices = append(choices, choice{device.Name, device.Description})
		}
		drop := newDropDown(choices, main.Name, func(name string) { a.setDefault(kind, name) })
		drop.combo.SetHAlign(gtk.ALIGN_START)
		styled(drop.combo, "h3xd-vol-big-combo")
		hero.PackStart(drop.combo, false, false, 0)
	} else {
		hero.PackStart(newName(main.Description, "h3xd-vol-big"), false, false, 0)
	}

	hero.PackStart(labeled("Volume", a.deviceControl(main).box), false, false, 0)

	meterDevice := main.Name
	if kind == pulse.Output {
		meterDevice = main.MonitorName
	}
	if meterDevice != "" {
		bar := check(gtk.LevelBarNewForInterval(0, 1))
		// The meter is one colour; GTK's offsets would colour its loud end as a warning.
		for _, offset := range []string{"low", "high", "full"} {
			bar.RemoveOffsetValue(offset)
		}
		styled(bar, "h3xd-vol-meter")
		bar.SetVAlign(gtk.ALIGN_CENTER)
		bar.SetMarginStart(sliderInset)
		hero.PackStart(labeled(text.level, bar), false, false, 0)
		a.startMeter(meterDevice, bar)
		// PipeWire's monitor hears the sound before the mute, so a muted output still
		// shows a level; grey says that it does not reach the speakers.
		name := main.Name
		update := func() {
			if fresh, ok := a.state.Device(kind, name); ok {
				setClass(bar, "muted", fresh.Mute)
			}
		}
		update()
		a.updaters = append(a.updaters, update)
	}

	if len(main.Ports) > 1 {
		var labels, values []string
		for _, port := range main.Ports {
			label := port.Description
			if !port.Available {
				label += " (unplugged)"
			}
			labels = append(labels, label)
			values = append(values, port.Name)
		}
		name := main.Name
		ports := newSegmented(labels, values, main.ActivePort, func(port string) {
			a.setter.do(name+"/port", func() error { return pulse.SetPort(kind, name, port) })
		})
		hero.PackStart(labeled("Connector", ports.box), false, false, 0)
	}

	if pulse.HasBalance(main.Channels) {
		name := main.Name
		balance := a.newBalanceControl(main.Channels,
			func(levels []pulse.Channel) error { return pulse.SetDeviceVolume(kind, name, levels) }, name)
		a.updaters = append(a.updaters, func() {
			if fresh, ok := a.state.Device(kind, name); ok {
				balance.update(fresh.Channels)
			}
		})
		hero.PackStart(labeled("Balance", balance.box), false, false, 0)
	}
	box.PackStart(hero, false, false, 0)

	first := true
	for _, device := range visible {
		if device.Name == main.Name {
			continue
		}
		if first {
			heading(box, text.others)
			first = false
		}
		box.PackStart(a.otherDevice(device), false, false, 0)
	}

	if kind == pulse.Output {
		box.PackStart(a.louderRow(), false, false, 0)
	}
}

func (a *app) otherDevice(device pulse.Device) *gtk.Box {
	panel := card(10)
	top := newBox(false, 12)
	text := newBox(true, 2)
	text.PackStart(newName(device.Description, "h3xd-vol-name"), false, false, 0)
	for _, port := range device.Ports {
		if port.Name == device.ActivePort {
			text.PackStart(newName(port.Description, "h3xd-muted"), false, false, 0)
		}
	}
	top.PackStart(text, true, true, 0)
	kind, name := device.Kind, device.Name
	top.PackStart(newButton("Use", func() { a.setDefault(kind, name) }), false, false, 0)
	panel.PackStart(top, false, false, 0)
	panel.PackStart(a.deviceControl(device).box, false, false, 0)
	return panel
}

func (a *app) louderRow() *gtk.Box {
	row := newBox(false, 16, "h3xd-vol-card")
	text := newBox(true, 4)
	text.PackStart(newLabel("Louder than 100 %", "h3xd-vol-name"), false, false, 0)
	text.PackStart(newText(fmt.Sprintf("Lets the volume go up to %d %%. Quiet recordings get louder; "+
		"loud ones distort.", louderLimit), "h3xd-muted"), false, false, 0)
	row.PackStart(text, true, true, 0)
	row.PackStart(newSwitch(a.settings.Louder, func(on bool) {
		a.settings.Louder = on
		if err := a.settings.save(); err != nil {
			a.setNotice(fmt.Sprintf("the setting was not saved: %v", err), true)
		}
		// Every slider changes its range; not from inside the switch's own handler.
		glib.IdleAdd(func() bool {
			a.render()
			return false
		})
	}), false, false, 0)
	return row
}

func (a *app) appsPage(box *gtk.Box) {
	a.streamList(box, pulse.Output, "Playing", "No program is playing sound right now.")
	a.streamList(box, pulse.Input, "Recording", "No program is recording right now.")
}

func (a *app) streamList(box *gtk.Box, kind pulse.Kind, title, empty string) {
	heading(box, title)
	streams := a.state.Streams(kind)
	if len(streams) == 0 {
		box.PackStart(newText(empty, "h3xd-muted", "h3xd-vol-empty"), false, false, 0)
		return
	}
	for _, stream := range streams {
		box.PackStart(a.streamRow(stream), false, false, 0)
	}
}

// streamIcon is the program's own icon, or a general one for programs without.
func streamIcon(stream pulse.Stream) string {
	theme, err := gtk.IconThemeGetDefault()
	if err == nil {
		for _, name := range []string{stream.Icon, strings.ToLower(stream.App), stream.Binary} {
			if name != "" && theme.HasIcon(name) {
				return name
			}
		}
	}
	return "applications-multimedia"
}

func mediaText(stream pulse.Stream) string {
	text := ""
	if stream.Media != stream.App {
		text = stream.Media
	}
	if stream.Corked {
		if text == "" {
			return "Paused"
		}
		return text + " · paused"
	}
	return text
}

func (a *app) findStream(kind pulse.Kind, index uint32) (pulse.Stream, bool) {
	for _, stream := range a.state.Streams(kind) {
		if stream.Index == index {
			return stream, true
		}
	}
	return pulse.Stream{}, false
}

func (a *app) deviceName(kind pulse.Kind, index uint32) string {
	for _, device := range a.state.Devices(kind) {
		if device.Index == index {
			return device.Name
		}
	}
	return ""
}

func (a *app) streamRow(stream pulse.Stream) *gtk.Box {
	kind, index := stream.Kind, stream.Index
	target := fmt.Sprintf("stream/%d/%d", kind, index)
	panel := card(10)
	top := newBox(false, 14)
	icon := check(gtk.ImageNewFromIconName(streamIcon(stream), gtk.ICON_SIZE_DND))
	icon.SetPixelSize(streamIconSize)
	top.PackStart(icon, false, false, 0)
	text := newBox(true, 2)
	text.SetVAlign(gtk.ALIGN_CENTER)
	text.PackStart(newName(stream.App, "h3xd-vol-name"), false, false, 0)
	media := newName(mediaText(stream), "h3xd-muted")
	text.PackStart(media, false, false, 0)
	top.PackStart(text, true, true, 0)

	// A recording program can also record what an output plays, so monitors are listed.
	devices := a.state.Devices(kind)
	var drop *dropDown
	if len(devices) > 1 {
		var choices []choice
		for _, device := range devices {
			choices = append(choices, choice{device.Name, device.Description})
		}
		drop = newDropDown(choices, a.deviceName(kind, stream.Device), func(name string) {
			a.setter.do(target+"/move", func() error { return pulse.MoveStream(kind, index, name) })
		})
		top.PackStart(drop.combo, false, false, 0)
	}
	panel.PackStart(top, false, false, 0)

	control := a.newVolumeControl(stream.Channels, stream.Mute,
		func(levels []pulse.Channel) error { return pulse.SetStreamVolume(kind, index, levels) },
		func(mute bool) error { return pulse.SetStreamMute(kind, index, mute) }, target)
	panel.PackStart(control.box, false, false, 0)
	a.updaters = append(a.updaters, func() {
		fresh, ok := a.findStream(kind, index)
		if !ok {
			return
		}
		control.update(fresh.Channels, fresh.Mute)
		media.SetText(mediaText(fresh))
		if drop != nil {
			drop.set(a.deviceName(kind, fresh.Device))
		}
	})
	return panel
}

func busWord(bus string) string {
	switch bus {
	case "":
		return ""
	case "bluetooth":
		return "Bluetooth"
	default:
		return strings.ToUpper(bus)
	}
}

func (a *app) cardsPage(box *gtk.Box) {
	if len(a.state.Cards) == 0 {
		panel := card(8)
		panel.PackStart(newText("No sound card was found.", "h3xd-vol-big"), false, false, 0)
		box.PackStart(panel, false, false, 0)
		return
	}
	box.PackStart(newText("A profile decides which outputs and inputs of a card are on, for "+
		"example sound over HDMI or a headset's microphone.", "h3xd-muted"), false, false, 0)
	for _, sound := range a.state.Cards {
		panel := card(12)
		top := newBox(false, 12)
		top.PackStart(newName(sound.Description, "h3xd-vol-name"), true, true, 0)
		if bus := busWord(sound.Bus); bus != "" {
			top.PackStart(chip(bus), false, false, 0)
		}
		panel.PackStart(top, false, false, 0)
		var choices []choice
		for _, profile := range sound.Profiles {
			text := profile.Description
			if !profile.Available {
				text += " (unplugged)"
			}
			choices = append(choices, choice{profile.Name, text})
		}
		name := sound.Name
		drop := newDropDown(choices, sound.ActiveProfile, func(profile string) {
			a.setter.do("card/"+name, func() error { return pulse.SetProfile(name, profile) })
		})
		panel.PackStart(labeled("Profile", drop.combo), false, false, 0)
		box.PackStart(panel, false, false, 0)
	}
}

// startMeter shows the level of a device on bar until the page is built again.
func (a *app) startMeter(device string, bar *gtk.LevelBar) {
	ctx, cancel := context.WithCancel(context.Background())
	a.stopMeter = cancel
	go func() {
		shown := 0.0
		err := pulse.Meter(ctx, device, func(level float64) {
			shown = math.Max(level, shown-meterDecay)
			value := shown
			glib.IdleAdd(func() bool {
				// The page may have been built again since; its bar is gone.
				if ctx.Err() == nil {
					bar.SetValue(value)
				}
				return false
			})
		})
		if err != nil && ctx.Err() == nil {
			log.Printf("h3xd-volume: level of %s: %v", device, err)
		}
	}()
}

func (a *app) endMeter() {
	if a.stopMeter != nil {
		a.stopMeter()
		a.stopMeter = nil
	}
}
