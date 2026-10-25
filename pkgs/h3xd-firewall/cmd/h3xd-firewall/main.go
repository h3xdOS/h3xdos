// Command h3xd-firewall is H3XD Firewall: on and off, simple services and advanced ufw
// rules in the look of the H3XD settings (design 2d). Changes go through
// h3xd-firewall-helper, which pkexec starts as root once the password is given.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"github.com/h3xdOS/h3xdos/firewall/internal/protocol"
	"github.com/h3xdOS/h3xdos/firewall/internal/ufw"
)

const (
	appID         = "org.h3xd.Firewall"
	windowWidth   = 1080
	windowHeight  = 740
	sidebarWidth  = 230
	pageOverview  = "overview"
	pageRules     = "rules"
	pageServices  = "services"
	pageActivity  = "activity"
	pageSettings  = "settings"
	iconName      = "h3xd-firewall"
	noticeTimeout = 8000
)

var pages = []struct{ id, name, subtitle string }{
	{pageOverview, "Overview", "Whether the firewall is on and what it lets through"},
	{pageRules, "Rules", "Every ufw rule, checked from the top; the first that matches decides"},
	{pageServices, "Services", "Let other devices reach services on this computer"},
	{pageActivity, "Activity", "Connections the firewall blocked, newest first"},
	{pageSettings, "Settings", "What happens to connections no rule matches, and logging"},
}

type app struct {
	application *gtk.Application
	window      *gtk.ApplicationWindow
	helper      *helper
	status      *protocol.Status
	listening   []ufw.Listener
	blocked     []ufw.Blocked
	page        string
	busy        bool
	starting    bool
	notice      string
	noticeError bool
	noticeID    int

	nav       map[string]*gtk.Button
	sideState *gtk.Label
	title     *gtk.Label
	subtitle  *gtk.Label
	holder    *gtk.Box
	content   *gtk.Box
}

func main() {
	application := check(gtk.ApplicationNew(appID, glib.APPLICATION_FLAGS_NONE))
	a := &app{application: application, page: pageOverview}
	application.Connect("activate", a.activate)
	application.Connect("shutdown", func() {
		if a.helper != nil {
			a.helper.close()
		}
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
	a.unlock()
}

func (a *app) buildWindow() {
	a.window = check(gtk.ApplicationWindowNew(a.application))
	a.window.SetTitle("H3XD Firewall")
	a.window.SetIconName(iconName)
	a.window.SetDefaultSize(windowWidth, windowHeight)
	a.window.SetPosition(gtk.WIN_POS_CENTER)

	root := newBox(false, 0, "h3xd-fw")
	side := newBox(true, 2, "h3xd-fw-side")
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
	a.sideState = newLabel("", "h3xd-fw-meta")
	a.sideState.SetMarginStart(12)
	side.PackEnd(a.sideState, false, false, 0)
	root.PackStart(side, false, false, 0)

	body := newBox(true, 0, "h3xd-fw-main")
	header := newBox(false, 16, "h3xd-fw-header")
	a.title = newLabel("", "h3xd-fw-title")
	header.PackStart(a.title, true, true, 0)
	body.PackStart(header, false, false, 0)
	a.subtitle = newText("", "h3xd-muted", "h3xd-fw-subtitle")
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

// unlock asks for the password and starts the helper; without it the pages stay locked.
func (a *app) unlock() {
	if a.starting {
		return
	}
	a.starting = true
	a.render()
	go func() {
		started, response, err := startHelper()
		glib.IdleAdd(func() bool {
			a.starting = false
			if err != nil {
				if !errors.Is(err, errDeclined) {
					a.setNotice(err.Error(), true)
				}
				a.render()
				return false
			}
			a.helper = started
			a.apply(protocol.OpStatus, response)
			// The page may need more than the status, such as the listening programs.
			a.navigate(a.page)
			return false
		})
	}()
}

func (a *app) navigate(page string) {
	a.page = page
	if a.helper != nil {
		switch page {
		case pageServices:
			a.request(protocol.Request{Op: protocol.OpListening}, nil)
		case pageActivity:
			a.request(protocol.Request{Op: protocol.OpBlocked}, nil)
		}
	}
	a.render()
}

// request sends a change to the helper away from the GTK thread and draws the page again
// from the answer; then runs after a successful answer.
func (a *app) request(request protocol.Request, then func(protocol.Response)) {
	if a.helper == nil || a.busy {
		return
	}
	a.busy = true
	a.holder.SetSensitive(false)
	current := a.helper
	go func() {
		response, err := current.call(request)
		glib.IdleAdd(func() bool {
			a.busy = false
			a.holder.SetSensitive(true)
			if err != nil {
				// The helper is gone; the password is needed again.
				a.helper = nil
				a.setNotice(err.Error(), true)
				a.render()
				return false
			}
			a.apply(request.Op, response)
			if response.OK && then != nil {
				then(response)
			}
			a.render()
			return false
		})
	}()
}

// apply takes in an answer; lists come only with the request for them, and an empty list
// is left out of the answer.
func (a *app) apply(op string, response protocol.Response) {
	if response.Status != nil {
		a.status = response.Status
	}
	switch op {
	case protocol.OpListening:
		a.listening = response.Listening
	case protocol.OpBlocked:
		a.blocked = response.Blocked
	}
	switch {
	case response.Error != "":
		a.setNotice(response.Error, true)
	case response.Note != "":
		a.setNotice(response.Note, false)
	}
}

// setNotice shows a message above the page until the next one, or for a while.
func (a *app) setNotice(text string, isError bool) {
	a.notice, a.noticeError = text, isError
	a.noticeID++
	id := a.noticeID
	glib.TimeoutAdd(noticeTimeout, func() bool {
		if a.noticeID == id && a.notice != "" {
			a.notice = ""
			a.render()
		}
		return false
	})
}

func (a *app) render() {
	for _, page := range pages {
		setClass(a.nav[page.id], "selected", page.id == a.page)
		if page.id == a.page {
			a.title.SetText(page.name)
			a.subtitle.SetText(page.subtitle)
		}
	}
	switch {
	case a.helper == nil:
		a.sideState.SetText("locked")
	case a.status.Enabled:
		a.sideState.SetText(fmt.Sprintf("firewall on, %d rules", len(a.status.Rules)))
	default:
		a.sideState.SetText("firewall off")
	}

	if a.content != nil {
		a.holder.Remove(a.content)
		a.content.Destroy()
	}
	a.content = newBox(true, 14, "h3xd-fw-content")
	if a.notice != "" {
		notice := newText(sentence(a.notice), "h3xd-notice")
		setClass(notice, "error", a.noticeError)
		a.content.PackStart(notice, false, false, 0)
	}
	if a.helper == nil {
		a.lockedPage(a.content)
	} else {
		switch a.page {
		case pageOverview:
			a.overviewPage(a.content)
		case pageRules:
			a.rulesPage(a.content)
		case pageServices:
			a.servicesPage(a.content)
		case pageActivity:
			a.activityPage(a.content)
		case pageSettings:
			a.settingsPage(a.content)
		}
	}
	a.holder.PackStart(a.content, false, false, 0)
	a.content.ShowAll()
}
