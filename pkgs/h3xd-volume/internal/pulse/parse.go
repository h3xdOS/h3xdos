package pulse

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Kind tells outputs from inputs, and the programs that play from those that record.
type Kind int

const (
	Output Kind = iota
	Input
)

// MeterClient is the client name of the level meters of H3XD Volume; their recording
// streams are not programs the user started and stay out of the lists.
const MeterClient = "h3xd-volume-meter"

// unavailable is how pactl marks a port with nothing plugged in.
const unavailable = "not available"

// Port is a connector of a device, such as speakers or headphones.
type Port struct {
	Name        string
	Description string
	Available   bool
}

// Device is an output (a sink) or an input (a source).
type Device struct {
	Kind        Kind
	Index       uint32
	Name        string
	Description string
	Mute        bool
	Channels    []Channel
	Ports       []Port
	ActivePort  string
	// Monitor marks an input that records what an output plays.
	Monitor bool
	// MonitorName is the input that records an output.
	MonitorName string
	Bus         string
}

// Stream is a program that plays (Output) or records (Input).
type Stream struct {
	Kind     Kind
	Index    uint32
	Device   uint32
	App      string
	Media    string
	Icon     string
	Binary   string
	Mute     bool
	Corked   bool
	Channels []Channel
}

// Profile is one way of using a sound card, such as stereo output with microphone input.
type Profile struct {
	Name        string
	Description string
	Priority    int
	Available   bool
}

// Card is a sound card with its profiles.
type Card struct {
	Index         uint32
	Name          string
	Description   string
	Bus           string
	Profiles      []Profile
	ActiveProfile string
}

// Server names the sound server and its default devices.
type Server struct {
	Name          string
	DefaultOutput string
	DefaultInput  string
}

type rawVolume map[string]struct {
	Value uint32 `json:"value"`
}

type rawDevice struct {
	Index         uint32            `json:"index"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	ChannelMap    string            `json:"channel_map"`
	Mute          bool              `json:"mute"`
	Volume        rawVolume         `json:"volume"`
	MonitorSource string            `json:"monitor_source"`
	ActivePort    string            `json:"active_port"`
	Properties    map[string]string `json:"properties"`
	Ports         []struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		Availability string `json:"availability"`
	} `json:"ports"`
}

type rawStream struct {
	Index      uint32            `json:"index"`
	Sink       *uint32           `json:"sink"`
	Source     *uint32           `json:"source"`
	ChannelMap string            `json:"channel_map"`
	Corked     bool              `json:"corked"`
	Mute       bool              `json:"mute"`
	Volume     rawVolume         `json:"volume"`
	Properties map[string]string `json:"properties"`
}

type rawCard struct {
	Index      uint32            `json:"index"`
	Name       string            `json:"name"`
	Properties map[string]string `json:"properties"`
	Profiles   map[string]struct {
		Description string `json:"description"`
		Priority    int    `json:"priority"`
		Available   bool   `json:"available"`
	} `json:"profiles"`
	ActiveProfile string `json:"active_profile"`
}

// channels puts the volume of each position in the order of the channel map, which is the
// order pactl expects when it sets them.
func channels(channelMap string, volume rawVolume) ([]Channel, error) {
	var result []Channel
	for _, position := range strings.Split(channelMap, ",") {
		if position == "" {
			continue
		}
		value, ok := volume[position]
		if !ok {
			return nil, fmt.Errorf("no volume for channel %q", position)
		}
		result = append(result, Channel{Position: position, Value: value.Value})
	}
	return result, nil
}

// ParseDevices reads `pactl -f json list sinks` (Output) or `list sources` (Input).
func ParseDevices(kind Kind, data []byte) ([]Device, error) {
	var raw []rawDevice
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("pactl device list: %w", err)
	}
	devices := make([]Device, 0, len(raw))
	for _, item := range raw {
		levels, err := channels(item.ChannelMap, item.Volume)
		if err != nil {
			return nil, fmt.Errorf("device %s: %w", item.Name, err)
		}
		device := Device{Kind: kind, Index: item.Index, Name: item.Name,
			Description: item.Description, Mute: item.Mute, Channels: levels,
			ActivePort: item.ActivePort, Bus: item.Properties["device.bus"]}
		// An output names its monitor in monitor_source; a monitor names its output there.
		if kind == Output {
			device.MonitorName = item.MonitorSource
		} else {
			device.Monitor = item.MonitorSource != "" || item.Properties["device.class"] == "monitor"
		}
		for _, port := range item.Ports {
			device.Ports = append(device.Ports, Port{Name: port.Name,
				Description: port.Description, Available: port.Availability != unavailable})
		}
		devices = append(devices, device)
	}
	return devices, nil
}

// ParseStreams reads `pactl -f json list sink-inputs` (Output) or `list source-outputs`
// (Input), leaving out the level meters.
func ParseStreams(kind Kind, data []byte) ([]Stream, error) {
	var raw []rawStream
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("pactl stream list: %w", err)
	}
	streams := make([]Stream, 0, len(raw))
	for _, item := range raw {
		properties := item.Properties
		if properties["application.name"] == MeterClient {
			continue
		}
		levels, err := channels(item.ChannelMap, item.Volume)
		if err != nil {
			return nil, fmt.Errorf("stream %d: %w", item.Index, err)
		}
		device := item.Sink
		if kind == Input {
			device = item.Source
		}
		if device == nil {
			return nil, fmt.Errorf("stream %d has no device", item.Index)
		}
		app := properties["application.name"]
		if app == "" {
			app = properties["node.name"]
		}
		streams = append(streams, Stream{Kind: kind, Index: item.Index, Device: *device,
			App: app, Media: properties["media.name"], Icon: properties["application.icon_name"],
			Binary: properties["application.process.binary"], Mute: item.Mute,
			Corked: item.Corked, Channels: levels})
	}
	return streams, nil
}

// ParseCards reads `pactl -f json list cards`, with the profiles in the order of their
// priority, the one the server would pick first.
func ParseCards(data []byte) ([]Card, error) {
	var raw []rawCard
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("pactl card list: %w", err)
	}
	cards := make([]Card, 0, len(raw))
	for _, item := range raw {
		card := Card{Index: item.Index, Name: item.Name, ActiveProfile: item.ActiveProfile,
			Description: item.Properties["device.description"], Bus: item.Properties["device.bus"]}
		if card.Description == "" {
			card.Description = item.Name
		}
		for name, profile := range item.Profiles {
			card.Profiles = append(card.Profiles, Profile{Name: name,
				Description: profile.Description, Priority: profile.Priority,
				Available: profile.Available})
		}
		sort.Slice(card.Profiles, func(i, j int) bool {
			a, b := card.Profiles[i], card.Profiles[j]
			if a.Priority != b.Priority {
				return a.Priority > b.Priority
			}
			return a.Name < b.Name
		})
		cards = append(cards, card)
	}
	return cards, nil
}

// ParseServer reads `pactl -f json info`.
func ParseServer(data []byte) (Server, error) {
	var raw struct {
		ServerName    string `json:"server_name"`
		DefaultSink   string `json:"default_sink_name"`
		DefaultSource string `json:"default_source_name"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Server{}, fmt.Errorf("pactl info: %w", err)
	}
	return Server{Name: serverName(raw.ServerName), DefaultOutput: raw.DefaultSink,
		DefaultInput: raw.DefaultSource}, nil
}

// serverName shortens "PulseAudio (on PipeWire 1.6.10)" to the server that really runs.
func serverName(name string) string {
	if start := strings.Index(name, "(on "); start >= 0 && strings.HasSuffix(name, ")") {
		return name[start+len("(on ") : len(name)-1]
	}
	return name
}
