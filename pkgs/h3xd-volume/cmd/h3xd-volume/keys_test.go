package main

import (
	"testing"

	"github.com/h3xdOS/h3xdos/volume/internal/pulse"
)

func TestSteps(t *testing.T) {
	up := []struct{ percent, limit, want int }{
		{0, 100, 5}, {43, 100, 45}, {45, 100, 50}, {98, 100, 100}, {100, 100, 100},
		{100, 150, 105}, {140, 100, 140},
	}
	for _, c := range up {
		if got := stepUp(c.percent, c.limit); got != c.want {
			t.Errorf("stepUp(%d, %d) = %d, want %d", c.percent, c.limit, got, c.want)
		}
	}
	down := []struct{ percent, want int }{{0, 0}, {3, 0}, {43, 40}, {45, 40}, {100, 95}, {140, 135}}
	for _, c := range down {
		if got := stepDown(c.percent); got != c.want {
			t.Errorf("stepDown(%d) = %d, want %d", c.percent, got, c.want)
		}
	}
}

func TestOSDMessage(t *testing.T) {
	state := osdState{Kind: pulse.Input, Percent: 35, Muted: true, Limit: louderLimit}
	decoded, err := decodeOSD(state.encode())
	if err != nil {
		t.Fatal(err)
	}
	if decoded != state {
		t.Errorf("got %+v, want %+v", decoded, state)
	}
	if _, err := decodeOSD("garbage\n"); err == nil {
		t.Error("garbage decoded")
	}
}
