package pulse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// List names one of the lists pactl prints; an event of the server says which to read again.
type List string

const (
	Sinks         List = "sinks"
	Sources       List = "sources"
	SinkInputs    List = "sink-inputs"
	SourceOutputs List = "source-outputs"
	Cards         List = "cards"
	Info          List = "info"
)

// AllLists is what a window needs when it opens or the server comes back.
var AllLists = []List{Info, Sinks, Sources, SinkInputs, SourceOutputs, Cards}

// ErrNoServer means pactl found no sound server to talk to.
var ErrNoServer = errors.New("no sound server is running")

// pactl runs pactl in the C locale: the commands below parse no text, but its error
// messages go to the window as they are.
func pactl(args ...string) ([]byte, error) {
	command := exec.Command("pactl", args...)
	command.Env = append(os.Environ(), "LC_ALL=C")
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if strings.Contains(message, "Connection refused") || strings.Contains(message, "Connection failure") {
			return nil, ErrNoServer
		}
		if message == "" {
			return nil, fmt.Errorf("pactl %s: %w", args[0], err)
		}
		return nil, fmt.Errorf("pactl %s: %s", args[0], message)
	}
	return output, nil
}

// State is everything the window shows.
type State struct {
	Server    Server
	Outputs   []Device
	Inputs    []Device
	Playback  []Stream
	Recording []Stream
	Cards     []Card
}

// Load reads the given lists into state; lists not named keep what state had.
func Load(state *State, lists ...List) error {
	for _, list := range lists {
		args := []string{"-f", "json", "list", string(list)}
		if list == Info {
			args = []string{"-f", "json", "info"}
		}
		data, err := pactl(args...)
		if err != nil {
			return err
		}
		switch list {
		case Info:
			state.Server, err = ParseServer(data)
		case Sinks:
			state.Outputs, err = ParseDevices(Output, data)
		case Sources:
			state.Inputs, err = ParseDevices(Input, data)
		case SinkInputs:
			state.Playback, err = ParseStreams(Output, data)
		case SourceOutputs:
			state.Recording, err = ParseStreams(Input, data)
		case Cards:
			state.Cards, err = ParseCards(data)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Default is the default device of a kind, if the server has one.
func (s *State) Default(kind Kind) (Device, bool) {
	name := s.Server.DefaultOutput
	if kind == Input {
		name = s.Server.DefaultInput
	}
	return s.Device(kind, name)
}

// Device finds a device by name.
func (s *State) Device(kind Kind, name string) (Device, bool) {
	for _, device := range s.Devices(kind) {
		if device.Name == name {
			return device, true
		}
	}
	return Device{}, false
}

// Devices lists the outputs or the inputs, monitors included.
func (s *State) Devices(kind Kind) []Device {
	if kind == Input {
		return s.Inputs
	}
	return s.Outputs
}

// Streams lists the programs that play or record.
func (s *State) Streams(kind Kind) []Stream {
	if kind == Input {
		return s.Recording
	}
	return s.Playback
}

func deviceWord(kind Kind) string {
	if kind == Input {
		return "source"
	}
	return "sink"
}

func streamWord(kind Kind) string {
	if kind == Input {
		return "source-output"
	}
	return "sink-input"
}

func levels(channels []Channel) []string {
	values := make([]string, len(channels))
	for i, channel := range channels {
		values[i] = strconv.FormatUint(uint64(channel.Value), 10)
	}
	return values
}

func flag(on bool) string {
	if on {
		return "1"
	}
	return "0"
}

// SetDeviceVolume sets every channel of a device; target is a name or @DEFAULT_SINK@.
func SetDeviceVolume(kind Kind, target string, channels []Channel) error {
	args := append([]string{"set-" + deviceWord(kind) + "-volume", target}, levels(channels)...)
	_, err := pactl(args...)
	return err
}

func SetDeviceMute(kind Kind, target string, mute bool) error {
	_, err := pactl("set-"+deviceWord(kind)+"-mute", target, flag(mute))
	return err
}

func SetDefault(kind Kind, name string) error {
	_, err := pactl("set-default-"+deviceWord(kind), name)
	return err
}

func SetPort(kind Kind, device, port string) error {
	_, err := pactl("set-"+deviceWord(kind)+"-port", device, port)
	return err
}

func SetStreamVolume(kind Kind, index uint32, channels []Channel) error {
	args := append([]string{"set-" + streamWord(kind) + "-volume",
		strconv.FormatUint(uint64(index), 10)}, levels(channels)...)
	_, err := pactl(args...)
	return err
}

func SetStreamMute(kind Kind, index uint32, mute bool) error {
	_, err := pactl("set-"+streamWord(kind)+"-mute", strconv.FormatUint(uint64(index), 10), flag(mute))
	return err
}

// MoveStream sends a playing program to another output, or a recording one to another input.
func MoveStream(kind Kind, index uint32, device string) error {
	_, err := pactl("move-"+streamWord(kind), strconv.FormatUint(uint64(index), 10), device)
	return err
}

func SetProfile(card, profile string) error {
	_, err := pactl("set-card-profile", card, profile)
	return err
}

// eventLine is a line of `pactl subscribe`, such as "Event 'change' on sink #73"; the
// server has no index of its own and comes as #-1 or as the invalid index 4294967295.
var eventLine = regexp.MustCompile(`^Event '(\w+)' on ([a-z-]+) #-?\d+$`)

// eventLists says which list an event changes. Clients, modules and samples come and go
// with every pactl call and change nothing the window shows.
var eventLists = map[string][]List{
	"sink":          {Sinks},
	"source":        {Sources},
	"sink-input":    {SinkInputs},
	"source-output": {SourceOutputs},
	"card":          {Cards},
	"server":        {Info, Sinks, Sources},
}

// ParseEvent tells which lists a line of `pactl subscribe` makes stale.
func ParseEvent(line string) []List {
	match := eventLine.FindStringSubmatch(strings.TrimSpace(line))
	if match == nil {
		return nil
	}
	return eventLists[match[2]]
}

// Subscribe sends the lists that changed until ctx ends or the server goes away.
func Subscribe(ctx context.Context, changed chan<- []List) error {
	command := exec.CommandContext(ctx, "pactl", "subscribe")
	command.Env = append(os.Environ(), "LC_ALL=C")
	output, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		if lists := ParseEvent(scanner.Text()); lists != nil {
			select {
			case changed <- lists:
			case <-ctx.Done():
			}
		}
	}
	if err := command.Wait(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("pactl subscribe: %w", err)
	}
	return ctx.Err()
}

const (
	meterRate = 8000
	// meterChunk is 50 ms of samples: 20 readings a second, as smooth as the eye needs.
	meterChunk = meterRate / 20
	// meterFloor is the quietest level the meter shows, in dB.
	meterFloor = -60.0
)

// Meter reads the level of a device (an input, or the monitor of an output) and reports it
// from 0 for silence to 1 for full scale, on the dB scale that ears hear, until ctx ends.
// report runs on the goroutine of Meter.
func Meter(ctx context.Context, device string, report func(level float64)) error {
	command := exec.CommandContext(ctx, "parec", "--raw", "--format=float32le", "--channels=1",
		"--rate="+strconv.Itoa(meterRate), "--latency-msec=50", "--client-name="+MeterClient,
		"--stream-name=Level meter", "--device="+device)
	output, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		return err
	}
	samples := make([]float32, meterChunk)
	for {
		if err := binary.Read(output, binary.LittleEndian, samples); err != nil {
			waitErr := command.Wait()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return fmt.Errorf("parec stopped: %w", waitErr)
			}
			return err
		}
		// Once ctx ends, the read above fails on the killed process, which reaps it.
		if ctx.Err() == nil {
			report(Level(samples))
		}
	}
}

// Level is the peak of samples on the meter's scale.
func Level(samples []float32) float64 {
	var peak float64
	for _, sample := range samples {
		peak = math.Max(peak, math.Abs(float64(sample)))
	}
	if peak <= 0 {
		return 0
	}
	decibels := 20 * math.Log10(peak)
	return math.Max(0, math.Min(1, (decibels-meterFloor)/-meterFloor))
}
