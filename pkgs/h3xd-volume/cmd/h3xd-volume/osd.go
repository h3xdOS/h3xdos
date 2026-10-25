package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	"github.com/h3xdOS/h3xdos/volume/internal/pulse"
)

const (
	// osdTimeout is how long the level stays on screen after the last key press.
	osdTimeout   = 1500
	osdWidth     = 320
	osdBottom    = 120
	osdIconSize  = 32
	osdDialLimit = 200 * time.Millisecond
)

// osdState is what a key press shows: the volume, or for the microphone only whether it
// is muted.
type osdState struct {
	Kind    pulse.Kind
	Percent int
	Muted   bool
	Limit   int
}

func (s osdState) encode() string {
	return fmt.Sprintf("%d %d %t %d\n", s.Kind, s.Percent, s.Muted, s.Limit)
}

func decodeOSD(line string) (osdState, error) {
	var s osdState
	_, err := fmt.Sscanf(line, "%d %d %t %d", &s.Kind, &s.Percent, &s.Muted, &s.Limit)
	return s, err
}

// osdSocket is an abstract socket, which disappears with the process that shows the level,
// so a crash leaves nothing behind that would block the next key press.
func osdSocket() string {
	return fmt.Sprintf("@h3xd-volume-osd-%d", os.Getuid())
}

// showOSD hands the state to the process that already shows the level, so that held or
// repeated keys update one display; without one, this process becomes it.
func showOSD(state osdState) int {
	if sendOSD(state) == nil {
		return 0
	}
	listener, err := net.Listen("unix", osdSocket())
	if err != nil {
		// Another key press became the display a moment ago.
		if err := sendOSD(state); err != nil {
			log.Printf("h3xd-volume: on-screen level: %v", err)
			return 1
		}
		return 0
	}
	runOSD(listener.(*net.UnixListener), state)
	return 0
}

func sendOSD(state osdState) error {
	conn, err := net.DialTimeout("unix", osdSocket(), osdDialLimit)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(state.encode()))
	return err
}

// sameUser tells whether the other end of conn runs as this user: an abstract socket has
// no file permissions, and other accounts must not drive this screen.
func sameUser(conn *net.UnixConn) bool {
	raw, err := conn.SyscallConn()
	if err != nil {
		return false
	}
	var credentials *syscall.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		credentials, credErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil || credErr != nil {
		return false
	}
	return int(credentials.Uid) == os.Getuid()
}

type osd struct {
	window     *gtk.Window
	icon       *gtk.Image
	bar        *gtk.LevelBar
	value      *gtk.Label
	box        *gtk.Box
	generation int
	listener   *net.UnixListener
}

func runOSD(listener *net.UnixListener, state osdState) {
	gtk.Init(nil)
	installCSS()
	o := &osd{listener: listener}
	o.build()
	go o.serve()
	o.show(state)
	gtk.Main()
}

func (o *osd) build() {
	o.window = check(gtk.WindowNew(gtk.WINDOW_POPUP))
	o.window.SetTypeHint(gdk.WINDOW_TYPE_HINT_NOTIFICATION)
	o.window.SetAcceptFocus(false)
	styled(o.window, "h3xd-osd-window")
	// Round corners need a compositor; without one the window has the colour of the card.
	if screen, err := gdk.ScreenGetDefault(); err == nil && screen.IsComposited() {
		if visual, err := screen.GetRGBAVisual(); err == nil && visual != nil {
			o.window.SetVisual(visual)
			o.window.SetAppPaintable(true)
			styled(o.window, "composited")
		}
	}
	o.box = newBox(false, 14, "h3xd-osd")
	o.box.SetSizeRequest(osdWidth, -1)
	o.icon = check(gtk.ImageNew())
	o.icon.SetPixelSize(osdIconSize)
	o.box.PackStart(o.icon, false, false, 0)
	o.bar = check(gtk.LevelBarNewForInterval(0, 1))
	for _, offset := range []string{"low", "high", "full"} {
		o.bar.RemoveOffsetValue(offset)
	}
	o.bar.SetVAlign(gtk.ALIGN_CENTER)
	o.box.PackStart(o.bar, true, true, 0)
	o.value = newLabel("", "h3xd-osd-value")
	// Filled, so that halign can move the words of the microphone to the start.
	o.box.PackStart(o.value, false, true, 0)
	o.window.Add(o.box)
}

// serve takes the states that later key presses send.
func (o *osd) serve() {
	for {
		conn, err := o.listener.AcceptUnix()
		if err != nil {
			// The listener closes when the display ends.
			return
		}
		if !sameUser(conn) {
			conn.Close()
			continue
		}
		line, err := bufio.NewReader(conn).ReadString('\n')
		conn.Close()
		if err != nil {
			continue
		}
		state, err := decodeOSD(line)
		if err != nil {
			log.Printf("h3xd-volume: on-screen level: %v", err)
			continue
		}
		glib.IdleAdd(func() bool {
			o.show(state)
			return false
		})
	}
}

func (o *osd) show(state osdState) {
	o.box.ShowAll()
	setClass(o.box, "muted", state.Muted)
	// The microphone has words instead of a bar, in the type of the UI.
	setClass(o.value, "h3xd-osd-words", state.Kind == pulse.Input)
	if state.Kind == pulse.Input {
		o.bar.Hide()
		o.value.SetHExpand(true)
		o.value.SetHAlign(gtk.ALIGN_START)
		if state.Muted {
			o.icon.SetFromIconName("h3xd-microphone-muted", gtk.ICON_SIZE_DND)
			o.value.SetText("Microphone muted")
		} else {
			o.icon.SetFromIconName("h3xd-microphone", gtk.ICON_SIZE_DND)
			o.value.SetText("Microphone on")
		}
	} else {
		o.icon.SetFromIconName(volumeIcon(state), gtk.ICON_SIZE_DND)
		o.bar.SetMaxValue(float64(max(state.Limit, state.Percent)) / 100)
		o.bar.SetValue(float64(state.Percent) / 100)
		o.value.SetHExpand(false)
		o.value.SetHAlign(gtk.ALIGN_FILL)
		if state.Muted {
			o.value.SetText("Muted")
		} else {
			o.value.SetText(fmt.Sprintf("%d %%", state.Percent))
		}
	}
	o.place()
	o.window.Show()

	o.generation++
	generation := o.generation
	glib.TimeoutAdd(osdTimeout, func() bool {
		if generation == o.generation {
			// No new press came; key presses from now on start a display of their own.
			o.listener.Close()
			gtk.MainQuit()
		}
		return false
	})
}

// place puts the display near the bottom of the screen, in the middle, away from the
// panel at the top.
func (o *osd) place() {
	display, err := gdk.DisplayGetDefault()
	if err != nil {
		return
	}
	monitor, err := display.GetPrimaryMonitor()
	if err != nil || monitor == nil {
		if monitor, err = display.GetMonitor(0); err != nil || monitor == nil {
			return
		}
	}
	area := monitor.GetWorkarea()
	_, height := o.window.GetPreferredHeight()
	o.window.Move(area.GetX()+(area.GetWidth()-osdWidth)/2, area.GetY()+area.GetHeight()-osdBottom-height)
}

// volumeIcon picks the speaker of the H3XD icons that matches the volume.
func volumeIcon(state osdState) string {
	switch {
	case state.Muted || state.Percent == 0:
		return "audio-volume-muted"
	case state.Percent < 34:
		return "audio-volume-low"
	case state.Percent < 67:
		return "audio-volume-medium"
	default:
		return "audio-volume-high"
	}
}
