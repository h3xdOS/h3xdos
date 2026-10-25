package main

import (
	"fmt"
	"math"
	"sync"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"github.com/h3xdOS/h3xdos/volume/internal/pulse"
)

const (
	normalLimit = 100
	// louderLimit is the top of the slider with "Louder than 100 %" on; pavucontrol stops
	// at about the same 153 %, past which most sound only distorts.
	louderLimit  = 150
	percentChars = 5
	balanceSteps = 100
)

// setter runs changes for the server in the background, one at a time per target: while
// pactl sets one volume, only the newest of the values that come in meanwhile waits, so a
// dragged slider never queues up a backlog of old positions.
type setter struct {
	mu      sync.Mutex
	running map[string]bool
	waiting map[string]func() error
	failed  func(error)
}

func newSetter(failed func(error)) *setter {
	return &setter{running: map[string]bool{}, waiting: map[string]func() error{}, failed: failed}
}

func (s *setter) do(target string, change func() error) {
	s.mu.Lock()
	if s.running[target] {
		s.waiting[target] = change
		s.mu.Unlock()
		return
	}
	s.running[target] = true
	s.mu.Unlock()
	go func() {
		for {
			if err := change(); err != nil {
				glib.IdleAdd(func() bool {
					s.failed(err)
					return false
				})
			}
			s.mu.Lock()
			next, ok := s.waiting[target]
			delete(s.waiting, target)
			if !ok {
				s.running[target] = false
				s.mu.Unlock()
				return
			}
			s.mu.Unlock()
			change = next
		}
	}()
}

// volumeControl is a slider with its value in per cent and a mute button.
type volumeControl struct {
	a       *app
	box     *gtk.Box
	scale   *gtk.Scale
	percent *gtk.Label
	mute    *gtk.Button
	// base keeps the proportions of the channels while the slider sits at 0, where the
	// channels themselves no longer have any.
	base     []pulse.Channel
	muted    bool
	held     bool
	updating bool
}

func (a *app) newVolumeControl(channels []pulse.Channel, muted bool,
	setVolume func([]pulse.Channel) error, setMute func(bool) error, target string) *volumeControl {
	c := &volumeControl{a: a, box: newBox(false, 12, "h3xd-vol-control")}
	c.scale = check(gtk.ScaleNewWithRange(gtk.ORIENTATION_HORIZONTAL, 0, float64(a.limit()), 1))
	c.scale.SetDrawValue(false)
	c.scale.SetHExpand(true)
	c.scale.SetVAlign(gtk.ALIGN_CENTER)
	if a.limit() > normalLimit {
		c.scale.AddMark(normalLimit, gtk.POS_BOTTOM, "")
	}
	c.percent = newLabel("", "h3xd-vol-mono")
	c.percent.SetWidthChars(percentChars)
	c.percent.SetXAlign(1)
	c.mute = newButton("Mute", func() {
		c.muted = !c.muted
		c.showMute()
		a.setter.do(target+"/mute", func() error { return setMute(c.muted) })
	}, "h3xd-vol-mute")
	c.update(channels, muted)

	c.scale.Connect("value-changed", func() {
		value := int(math.Round(c.scale.GetValue()))
		c.percent.SetText(fmt.Sprintf("%d %%", value))
		if c.updating {
			return
		}
		levels := pulse.Scale(c.base, pulse.Raw(value))
		a.setter.do(target+"/volume", func() error { return setVolume(levels) })
	})
	a.holdWhilePressed(c.scale.ToWidget(), &c.held)

	c.box.PackStart(c.scale, true, true, 0)
	c.box.PackStart(c.percent, false, false, 0)
	c.box.PackStart(c.mute, false, false, 0)
	return c
}

// update shows the server's volume, except on a slider the mouse is holding: the server
// lags behind the hand, and following it would make the slider jump back.
func (c *volumeControl) update(channels []pulse.Channel, muted bool) {
	if pulse.Loudest(channels) > 0 {
		c.base = channels
	} else if c.base == nil {
		c.base = channels
	}
	c.muted = muted
	c.showMute()
	if c.held {
		return
	}
	value := pulse.Percent(pulse.Loudest(channels))
	c.updating = true
	// A volume set elsewhere above the limit stays reachable instead of being cut down.
	c.scale.SetRange(0, float64(max(c.a.limit(), value)))
	c.scale.SetValue(float64(value))
	c.updating = false
	c.percent.SetText(fmt.Sprintf("%d %%", value))
}

func (c *volumeControl) showMute() {
	if c.muted {
		c.mute.SetLabel("Muted")
	} else {
		c.mute.SetLabel("Mute")
	}
	setClass(c.mute, "on", c.muted)
	setClass(c.scale, "muted", c.muted)
}

// holdWhilePressed marks a slider as held between press and release of the mouse; the
// page is not built again while any slider is held, since that would end the drag.
func (a *app) holdWhilePressed(widget *gtk.Widget, held *bool) {
	widget.Connect("button-press-event", func() bool {
		if !*held {
			*held = true
			a.held++
		}
		return false
	})
	widget.Connect("button-release-event", func() bool {
		if *held {
			*held = false
			a.held--
			if a.held == 0 && a.stale {
				// Not from inside the handler of a widget that the rebuild destroys.
				glib.IdleAdd(func() bool {
					if a.held == 0 && a.stale {
						a.render()
					}
					return false
				})
			}
		}
		return false
	})
}

// balanceControl moves the sound between the left and the right speakers.
type balanceControl struct {
	box      *gtk.Box
	scale    *gtk.Scale
	channels []pulse.Channel
	held     bool
	updating bool
}

func (a *app) newBalanceControl(channels []pulse.Channel, setVolume func([]pulse.Channel) error,
	target string) *balanceControl {
	b := &balanceControl{box: newBox(false, 12)}
	b.scale = check(gtk.ScaleNewWithRange(gtk.ORIENTATION_HORIZONTAL, -balanceSteps, balanceSteps, 1))
	b.scale.SetDrawValue(false)
	b.scale.SetHExpand(true)
	b.scale.AddMark(0, gtk.POS_BOTTOM, "")
	styled(b.scale, "h3xd-vol-balance")
	b.update(channels)
	b.scale.Connect("value-changed", func() {
		if b.updating {
			return
		}
		levels := pulse.SetBalance(b.channels, b.scale.GetValue()/balanceSteps)
		a.setter.do(target+"/volume", func() error { return setVolume(levels) })
	})
	a.holdWhilePressed(b.scale.ToWidget(), &b.held)
	b.box.PackStart(newLabel("Left", "h3xd-muted"), false, false, 0)
	b.box.PackStart(b.scale, true, true, 0)
	b.box.PackStart(newLabel("Right", "h3xd-muted"), false, false, 0)
	return b
}

func (b *balanceControl) update(channels []pulse.Channel) {
	b.channels = channels
	if b.held {
		return
	}
	b.updating = true
	b.scale.SetValue(math.Round(pulse.Balance(channels) * balanceSteps))
	b.updating = false
}
