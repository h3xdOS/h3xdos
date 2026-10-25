package pulse

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// testdata holds pactl output of PipeWire 1.6 in a QEMU VM with an HDA card, a program
// playing and one recording; properties are cut down to the ones H3XD Volume reads.
func read(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestParseDevices(t *testing.T) {
	outputs, err := ParseDevices(Output, read(t, "sinks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(outputs) != 1 {
		t.Fatalf("got %d outputs, want 1", len(outputs))
	}
	output := outputs[0]
	want := Device{Kind: Output, Index: 73, Name: "alsa_output.pci-0000_00_05.0.analog-stereo",
		Description: "Built-in Audio Analog Stereo",
		Channels:    []Channel{{"front-left", 26214}, {"front-right", 26214}},
		Ports:       []Port{{"analog-output-lineout", "Line Out", true}},
		ActivePort:  "analog-output-lineout",
		MonitorName: "alsa_output.pci-0000_00_05.0.analog-stereo.monitor", Bus: "pci"}
	if !reflect.DeepEqual(output, want) {
		t.Errorf("got %+v\nwant %+v", output, want)
	}

	inputs, err := ParseDevices(Input, read(t, "sources.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 2 || !inputs[0].Monitor || inputs[1].Monitor {
		t.Fatalf("want the monitor first and the line in second, got %+v", inputs)
	}
	if inputs[1].Name != "alsa_input.pci-0000_00_05.0.analog-stereo" {
		t.Errorf("input name %q", inputs[1].Name)
	}
}

func TestParseStreams(t *testing.T) {
	playback, err := ParseStreams(Output, read(t, "sink-inputs.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Stream{{Kind: Output, Index: 123, Device: 73, App: "Firefox", Media: "YouTube video",
		Binary: "pacat", Channels: []Channel{{"front-left", Norm}, {"front-right", Norm}}}}
	if !reflect.DeepEqual(playback, want) {
		t.Errorf("got %+v\nwant %+v", playback, want)
	}
	recording, err := ParseStreams(Input, read(t, "source-outputs.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(recording) != 1 || recording[0].Device != 74 || recording[0].App != "Discord" {
		t.Errorf("recording %+v", recording)
	}
}

func TestMeterStreamsLeftOut(t *testing.T) {
	data := []byte(`[{"index":5,"source":74,"channel_map":"mono","mute":false,
		"volume":{"mono":{"value":65536}},"properties":{"application.name":"h3xd-volume-meter"}}]`)
	streams, err := ParseStreams(Input, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 0 {
		t.Errorf("meter listed: %+v", streams)
	}
}

func TestParseCards(t *testing.T) {
	cards, err := ParseCards(read(t, "cards.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 {
		t.Fatalf("got %d cards", len(cards))
	}
	card := cards[0]
	if card.Description != "Built-in Audio" || card.ActiveProfile != "output:analog-stereo+input:analog-stereo" {
		t.Errorf("card %+v", card)
	}
	var names []string
	for _, profile := range card.Profiles {
		names = append(names, profile.Name)
	}
	want := []string{"output:analog-stereo+input:analog-stereo", "output:analog-stereo",
		"input:analog-stereo", "pro-audio", "off"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("profiles %v, want %v", names, want)
	}
}

func TestParseServer(t *testing.T) {
	server, err := ParseServer(read(t, "info.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := Server{Name: "PipeWire 1.6.10", DefaultOutput: "alsa_output.pci-0000_00_05.0.analog-stereo",
		DefaultInput: "alsa_input.pci-0000_00_05.0.analog-stereo"}
	if server != want {
		t.Errorf("got %+v, want %+v", server, want)
	}
}

func TestParseEvent(t *testing.T) {
	cases := map[string][]List{
		"Event 'change' on sink #73":           {Sinks},
		"Event 'new' on sink-input #123":       {SinkInputs},
		"Event 'remove' on source-output #142": {SourceOutputs},
		"Event 'change' on server #-1":         {Info, Sinks, Sources},
		"Event 'change' on server #4294967295": {Info, Sinks, Sources},
		"Event 'new' on client #120":           nil,
		"Got SIGINT, exiting.":                 nil,
	}
	for line, want := range cases {
		if got := ParseEvent(line); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %v, want %v", line, got, want)
		}
	}
}

func TestPercent(t *testing.T) {
	if Percent(26214) != 40 || Percent(Norm) != 100 || Raw(45) != 29491 || Raw(-5) != 0 {
		t.Errorf("percent conversion: %d %d %d", Percent(26214), Percent(Norm), Raw(45))
	}
}

func TestScaleKeepsBalance(t *testing.T) {
	channels := []Channel{{"front-left", 40000}, {"front-right", 20000}}
	scaled := Scale(channels, 20000)
	want := []Channel{{"front-left", 20000}, {"front-right", 10000}}
	if !reflect.DeepEqual(scaled, want) {
		t.Errorf("got %v, want %v", scaled, want)
	}
	silent := Scale([]Channel{{"mono", 0}}, Norm)
	if silent[0].Value != Norm {
		t.Errorf("silent channel scaled to %d", silent[0].Value)
	}
}

func TestBalance(t *testing.T) {
	even := []Channel{{"front-left", Norm}, {"front-right", Norm}, {"front-center", Norm}}
	if Balance(even) != 0 || !HasBalance(even) {
		t.Errorf("even balance %v", Balance(even))
	}
	right := SetBalance(even, 0.5)
	want := []Channel{{"front-left", Norm / 2}, {"front-right", Norm}, {"front-center", Norm}}
	if !reflect.DeepEqual(right, want) {
		t.Errorf("got %v, want %v", right, want)
	}
	if got := Balance(right); got != 0.5 {
		t.Errorf("balance %v, want 0.5", got)
	}
	left := SetBalance(even, -1)
	if left[0].Value != Norm || left[1].Value != 0 || Balance(left) != -1 {
		t.Errorf("only left: %v", left)
	}
	if HasBalance([]Channel{{"mono", Norm}}) {
		t.Error("mono has no balance")
	}
}

func TestLevel(t *testing.T) {
	if Level([]float32{0, 0}) != 0 || Level([]float32{1, -0.2}) != 1 {
		t.Errorf("silence %v, full %v", Level([]float32{0}), Level([]float32{1}))
	}
	// -30 dB is half way up a meter that starts at -60 dB.
	if got := Level([]float32{0.0316228}); got < 0.49 || got > 0.51 {
		t.Errorf("-30 dB at %v", got)
	}
}
