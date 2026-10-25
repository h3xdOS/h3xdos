package main

import (
	"fmt"
	"log"
	"os"

	"github.com/h3xdOS/h3xdos/volume/internal/pulse"
)

// keyStep is how much one press of a volume key changes the volume, in per cent.
const keyStep = 5

const keyUsage = "usage: h3xd-volume [--raise | --lower | --mute | --mic-mute]"

// runKey serves a volume key of the keyboard (xfce4-keyboard-shortcuts): it changes the
// default output or input and shows the result on screen. It needs no window, so a key
// works the same whether H3XD Volume is open or not.
func runKey(argument string) int {
	kind := pulse.Output
	switch argument {
	case "--raise", "--lower", "--mute":
	case "--mic-mute":
		kind = pulse.Input
	default:
		fmt.Fprintln(os.Stderr, keyUsage)
		return 2
	}
	list := pulse.Sinks
	if kind == pulse.Input {
		list = pulse.Sources
	}
	var state pulse.State
	if err := pulse.Load(&state, pulse.Info, list); err != nil {
		log.Printf("h3xd-volume: %v", err)
		return 1
	}
	device, ok := state.Default(kind)
	if !ok {
		log.Printf("h3xd-volume: the sound server has no default device")
		return 1
	}
	limit := loadSettings().limit()
	percent := pulse.Percent(pulse.Loudest(device.Channels))
	muted := device.Mute
	var err error
	switch argument {
	case "--raise":
		// Raising the volume of a muted output is meant to make it heard.
		if muted {
			muted = false
			err = pulse.SetDeviceMute(kind, device.Name, false)
		}
		if target := stepUp(percent, limit); err == nil && target != percent {
			percent = target
			err = pulse.SetDeviceVolume(kind, device.Name, pulse.Scale(device.Channels, pulse.Raw(target)))
		}
	case "--lower":
		if target := stepDown(percent); target != percent {
			percent = target
			err = pulse.SetDeviceVolume(kind, device.Name, pulse.Scale(device.Channels, pulse.Raw(target)))
		}
	case "--mute", "--mic-mute":
		muted = !muted
		err = pulse.SetDeviceMute(kind, device.Name, muted)
	}
	if err != nil {
		log.Printf("h3xd-volume: %v", err)
		return 1
	}
	return showOSD(osdState{Kind: kind, Percent: percent, Muted: muted, Limit: limit})
}

// stepUp goes to the next multiple of keyStep, so that the volume lands on round values
// even after the slider left it at 43 %; a volume above the limit stays where it is.
func stepUp(percent, limit int) int {
	if percent >= limit {
		return percent
	}
	return min((percent/keyStep+1)*keyStep, limit)
}

func stepDown(percent int) int {
	return max(((percent+keyStep-1)/keyStep-1)*keyStep, 0)
}
