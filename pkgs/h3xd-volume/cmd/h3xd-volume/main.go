// Command h3xd-volume is H3XD Volume: the volume of outputs, inputs and programs and the
// use of the sound cards, in the look of the H3XD settings (design 2d). It talks to
// PipeWire through pactl and follows `pactl subscribe`, so changes made elsewhere, such as
// a headset plugged in, show at once.
//
// With an argument it serves a volume key instead (keys.go) and shows the level on screen.
package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"github.com/h3xdOS/h3xdos/volume/internal/pulse"
)

const (
	appID         = "org.h3xd.Volume"
	windowWidth   = 1000
	windowHeight  = 720
	sidebarWidth  = 230
	pageOutput    = "output"
	pageInput     = "input"
	pageApps      = "apps"
	pageDevices   = "devices"
	iconName      = "h3xd-volume"
	noticeTimeout = 8000
	// eventDelay gathers the burst of events that one change causes into one reload.
	eventDelay = 60 * time.Millisecond
	// retryDelay is how long the window waits before it looks for the sound server again.
	retryDelay = 3 * time.Second
)

var pages = []struct{ id, name, subtitle string }{
	{pageOutput, "Output", "Where sound plays and how loud"},
	{pageInput, "Input", "Microphones and other inputs, and how much they pick up"},
	{pageApps, "Applications", "The volume of each program, and where it plays or records"},
	{pageDevices, "Devices", "Sound cards and how they are used"},
}

type app struct {
	application *gtk.Application
	window      *gtk.ApplicationWindow
	state       pulse.State
	loaded      bool
	serverError error
	settings    settings
	setter      *setter
	stopWatch   context.CancelFunc
	page        string
	notice      string
	noticeError bool
	noticeID    int

	nav       map[string]*gtk.Button
	sideState *gtk.Label
	title     *gtk.Label
	subtitle  *gtk.Label
	holder    *gtk.Box
	content   *gtk.Box

	// shape describes what the page was built from; while it stays, the updaters bring
	// the built widgets up to date instead of building the page again.
	shape    string
	updaters []func()
	// held counts sliders under the mouse; stale marks a page to build again after them.
	held      int
	stale     bool
	stopMeter context.CancelFunc
}

func main() {
	if len(os.Args) > 1 {
		os.Exit(runKey(os.Args[1]))
	}
	application := check(gtk.ApplicationNew(appID, glib.APPLICATION_FLAGS_NONE))
	a := &app{application: application, page: pageOutput, settings: loadSettings()}
	a.setter = newSetter(func(err error) {
		a.setNotice(err.Error(), true)
		a.renderSoon()
	})
	application.Connect("activate", a.activate)
	application.Connect("shutdown", func() {
		if a.stopWatch != nil {
			a.stopWatch()
		}
		a.endMeter()
	})
	os.Exit(application.Run(os.Args))
}

func (a *app) activate() {
	if a.window != nil {
		a.window.Present()
		return
	}
	installCSS()
	a.buildWindow()
	ctx, cancel := context.WithCancel(context.Background())
	a.stopWatch = cancel
	go a.watch(ctx)
}

func (a *app) limit() int {
	return a.settings.limit()
}

func (a *app) buildWindow() {
	a.window = check(gtk.ApplicationWindowNew(a.application))
	a.window.SetTitle("H3XD Volume")
	a.window.SetIconName(iconName)
	a.window.SetDefaultSize(windowWidth, windowHeight)
	a.window.SetPosition(gtk.WIN_POS_CENTER)

	root := newBox(false, 0, "h3xd-vol")
	side := newBox(true, 2, "h3xd-vol-side")
	side.SetSizeRequest(sidebarWidth, -1)
	a.nav = map[string]*gtk.Button{}
	for _, page := range pages {
		id := page.id
		button := styled(check(gtk.ButtonNew()), "h3xd-nav")
		button.Add(newLabel(page.name))
		button.Connect("clicked", func() { a.navigate(id) })
		a.nav[id] = button
		side.PackStart(button, false, false, 0)
	}
	a.sideState = newLabel("", "h3xd-vol-meta")
	a.sideState.SetMarginStart(12)
	side.PackEnd(a.sideState, false, false, 0)
	root.PackStart(side, false, false, 0)

	body := newBox(true, 0, "h3xd-vol-main")
	a.title = newLabel("", "h3xd-vol-title")
	body.PackStart(a.title, false, false, 0)
	a.subtitle = newText("", "h3xd-muted", "h3xd-vol-subtitle")
	body.PackStart(a.subtitle, false, false, 0)
	scroller := check(gtk.ScrolledWindowNew(nil, nil))
	scroller.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	a.holder = newBox(true, 0)
	scroller.Add(a.holder)
	body.PackStart(scroller, true, true, 0)
	root.PackStart(body, true, true, 0)

	a.window.Add(root)
	a.render()
	a.window.ShowAll()
}

// watch follows the server's events and reads again what they change. When the server is
// missing or goes away, it looks for it again every few seconds.
func (a *app) watch(ctx context.Context) {
	for {
		changes := make(chan []pulse.List)
		ended := make(chan error, 1)
		go func() { ended <- pulse.Subscribe(ctx, changes) }()
		a.reload(pulse.AllLists)
		stale := map[pulse.List]bool{}
		var due <-chan time.Time
	events:
		for {
			select {
			case lists := <-changes:
				for _, list := range lists {
					stale[list] = true
				}
				if due == nil {
					due = time.After(eventDelay)
				}
			case <-due:
				due = nil
				var lists []pulse.List
				for _, list := range pulse.AllLists {
					if stale[list] {
						lists = append(lists, list)
					}
				}
				stale = map[pulse.List]bool{}
				a.reload(lists)
			case <-ended:
				break events
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(retryDelay):
		}
	}
}

// reload reads lists away from the GTK thread and hands them to it.
func (a *app) reload(lists []pulse.List) {
	var fresh pulse.State
	err := pulse.Load(&fresh, lists...)
	glib.IdleAdd(func() bool {
		a.merge(&fresh, lists, err)
		return false
	})
}

func (a *app) merge(fresh *pulse.State, lists []pulse.List, err error) {
	switch {
	case errors.Is(err, pulse.ErrNoServer):
		a.loaded, a.serverError = false, err
	case err != nil:
		a.setNotice(err.Error(), true)
	default:
		for _, list := range lists {
			switch list {
			case pulse.Info:
				a.state.Server = fresh.Server
			case pulse.Sinks:
				a.state.Outputs = fresh.Outputs
			case pulse.Sources:
				a.state.Inputs = fresh.Inputs
			case pulse.SinkInputs:
				a.state.Playback = fresh.Playback
			case pulse.SourceOutputs:
				a.state.Recording = fresh.Recording
			case pulse.Cards:
				a.state.Cards = fresh.Cards
			}
		}
		if len(lists) == len(pulse.AllLists) {
			a.loaded, a.serverError = true, nil
		}
	}
	a.refresh()
}

func (a *app) navigate(page string) {
	a.page = page
	a.render()
}

// setNotice shows a message above the page until the next one, or for a while.
func (a *app) setNotice(text string, isError bool) {
	a.notice, a.noticeError = text, isError
	a.noticeID++
	id := a.noticeID
	glib.TimeoutAdd(noticeTimeout, func() bool {
		if a.noticeID == id && a.notice != "" {
			a.notice = ""
			a.renderSoon()
		}
		return false
	})
}

// renderSoon builds the page again, or once the mouse lets go of the slider it holds.
func (a *app) renderSoon() {
	if a.held > 0 {
		a.stale = true
		return
	}
	a.render()
}

// refresh brings the page up to date with the state: in place while the page has the same
// shape, built again when devices or programs came or went.
func (a *app) refresh() {
	a.renderSide()
	if a.content != nil && a.pageShape() == a.shape {
		for _, update := range a.updaters {
			update()
		}
		return
	}
	if a.held > 0 {
		a.stale = true
		for _, update := range a.updaters {
			update()
		}
		return
	}
	a.render()
}

func (a *app) renderSide() {
	for _, page := range pages {
		setClass(a.nav[page.id], "selected", page.id == a.page)
		if page.id == a.page {
			a.title.SetText(page.name)
			a.subtitle.SetText(page.subtitle)
		}
	}
	switch {
	case a.serverError != nil:
		a.sideState.SetText("no sound server")
	case a.loaded:
		a.sideState.SetText(a.state.Server.Name)
	default:
		a.sideState.SetText("")
	}
}

func (a *app) render() {
	a.renderSide()
	a.endMeter()
	a.stale = false
	a.updaters = nil
	a.shape = a.pageShape()
	if a.content != nil {
		a.holder.Remove(a.content)
		a.content.Destroy()
	}
	a.content = newBox(true, 14, "h3xd-vol-content")
	if a.notice != "" {
		notice := newText(sentence(a.notice), "h3xd-notice")
		setClass(notice, "error", a.noticeError)
		a.content.PackStart(notice, false, false, 0)
	}
	switch {
	case a.serverError != nil:
		a.noServerPage(a.content)
	case !a.loaded:
		// The first answer of the server comes within a moment.
	default:
		switch a.page {
		case pageOutput:
			a.devicePage(a.content, pulse.Output)
		case pageInput:
			a.devicePage(a.content, pulse.Input)
		case pageApps:
			a.appsPage(a.content)
		case pageDevices:
			a.cardsPage(a.content)
		}
	}
	a.holder.PackStart(a.content, false, false, 0)
	a.content.ShowAll()
}
