// Package pulse reads and changes the sound server through pactl, the PulseAudio client
// that PipeWire answers through pipewire-pulse. pactl prints JSON since PulseAudio 16, so
// H3XD Volume needs no bindings to libpulse and follows whatever PipeWire reports.
package pulse

import (
	"math"
	"strings"
)

// Norm is PA_VOLUME_NORM: the raw volume of 100 %.
const Norm = 65536

// Channel is the volume of one speaker position, such as front-left.
type Channel struct {
	Position string
	Value    uint32
}

// Percent turns a raw volume into whole per cent, as pactl and pavucontrol show it.
func Percent(value uint32) int {
	return int(math.Round(float64(value) * 100 / Norm))
}

// Raw turns per cent into a raw volume.
func Raw(percent int) uint32 {
	if percent <= 0 {
		return 0
	}
	return uint32(math.Round(float64(percent) * Norm / 100))
}

// Loudest is the volume a single slider shows for several channels, as pa_cvolume_max.
func Loudest(channels []Channel) uint32 {
	var loudest uint32
	for _, channel := range channels {
		loudest = max(loudest, channel.Value)
	}
	return loudest
}

// Scale sets the loudest channel to value and the others in proportion, so that the
// balance stays, as pa_cvolume_scale.
func Scale(channels []Channel, value uint32) []Channel {
	loudest := Loudest(channels)
	scaled := make([]Channel, len(channels))
	for i, channel := range channels {
		scaled[i] = channel
		if loudest == 0 {
			scaled[i].Value = value
		} else {
			scaled[i].Value = uint32(math.Round(float64(channel.Value) * float64(value) / float64(loudest)))
		}
	}
	return scaled
}

type side int

const (
	center side = iota
	left
	right
)

func sideOf(position string) side {
	switch {
	case strings.HasSuffix(position, "-left"), strings.HasSuffix(position, "-left-of-center"):
		return left
	case strings.HasSuffix(position, "-right"), strings.HasSuffix(position, "-right-of-center"):
		return right
	}
	return center
}

// HasBalance reports whether the channels have a left and a right side to balance.
func HasBalance(channels []Channel) bool {
	var hasLeft, hasRight bool
	for _, channel := range channels {
		switch sideOf(channel.Position) {
		case left:
			hasLeft = true
		case right:
			hasRight = true
		}
	}
	return hasLeft && hasRight
}

func sides(channels []Channel) (leftAverage, rightAverage float64) {
	var leftSum, rightSum, leftCount, rightCount float64
	for _, channel := range channels {
		switch sideOf(channel.Position) {
		case left:
			leftSum += float64(channel.Value)
			leftCount++
		case right:
			rightSum += float64(channel.Value)
			rightCount++
		}
	}
	if leftCount > 0 {
		leftAverage = leftSum / leftCount
	}
	if rightCount > 0 {
		rightAverage = rightSum / rightCount
	}
	return leftAverage, rightAverage
}

// Balance is -1 for only the left side, 0 for both alike and 1 for only the right side,
// as pa_cvolume_get_balance.
func Balance(channels []Channel) float64 {
	leftAverage, rightAverage := sides(channels)
	switch {
	case leftAverage == rightAverage:
		return 0
	case leftAverage > rightAverage:
		return rightAverage/leftAverage - 1
	default:
		return 1 - leftAverage/rightAverage
	}
}

// SetBalance moves the sound between left and right and keeps the louder side as loud as
// it was, as pa_cvolume_set_balance; centre channels stay.
func SetBalance(channels []Channel, balance float64) []Channel {
	balance = math.Max(-1, math.Min(1, balance))
	leftAverage, rightAverage := sides(channels)
	loudest := math.Max(leftAverage, rightAverage)
	newLeft, newRight := loudest, loudest
	if balance < 0 {
		newRight = (1 + balance) * loudest
	} else {
		newLeft = (1 - balance) * loudest
	}
	balanced := make([]Channel, len(channels))
	for i, channel := range channels {
		balanced[i] = channel
		switch sideOf(channel.Position) {
		case left:
			balanced[i].Value = rescale(channel.Value, leftAverage, newLeft)
		case right:
			balanced[i].Value = rescale(channel.Value, rightAverage, newRight)
		}
	}
	return balanced
}

func rescale(value uint32, from, to float64) uint32 {
	if from == 0 {
		return uint32(math.Round(to))
	}
	return uint32(math.Round(float64(value) * to / from))
}
