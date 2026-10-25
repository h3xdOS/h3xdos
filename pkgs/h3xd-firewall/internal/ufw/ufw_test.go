package ufw

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	good := []Rule{
		{Action: "allow", Direction: "in", Proto: "tcp", Port: "22"},
		{Action: "deny", Direction: "in", From: "192.168.1.0/24"},
		{Action: "limit", Direction: "in", Proto: "tcp", Port: "6000:6007"},
		{Action: "reject", Direction: "out", Proto: "udp", Port: "53,123", To: "2001:db8::1"},
		{Action: "allow", Direction: "in", Interface: "wlan0", Log: "log", Comment: "home lan"},
	}
	for _, rule := range good {
		if err := rule.Validate(); err != nil {
			t.Errorf("%+v: unexpected error %v", rule, err)
		}
	}
	bad := map[string]Rule{
		"action":        {Action: "accept", Direction: "in"},
		"direction":     {Action: "allow", Direction: "both"},
		"port zero":     {Action: "allow", Direction: "in", Proto: "tcp", Port: "0"},
		"port too high": {Action: "allow", Direction: "in", Proto: "tcp", Port: "65536"},
		"range down":    {Action: "allow", Direction: "in", Proto: "tcp", Port: "90:80"},
		"list no proto": {Action: "allow", Direction: "in", Port: "80,443"},
		"shell":         {Action: "allow", Direction: "in", Port: "22; rm -rf /"},
		"address":       {Action: "allow", Direction: "in", From: "example.com"},
		"interface":     {Action: "allow", Direction: "in", Interface: "eth0 to any"},
		"quote":         {Action: "allow", Direction: "in", Comment: "it's"},
		"newline":       {Action: "allow", Direction: "in", Comment: "a\nb"},
		"long comment":  {Action: "allow", Direction: "in", Comment: strings.Repeat("x", 65)},
	}
	for name, rule := range bad {
		if rule.Validate() == nil {
			t.Errorf("%s: %+v passed validation", name, rule)
		}
	}
}

func TestArgs(t *testing.T) {
	rule := Rule{Action: "allow", Direction: "in", Interface: "eth0", Proto: "tcp",
		From: "10.0.0.0/8", To: "", Port: "22", Comment: "admins"}
	want := []string{"allow", "in", "on", "eth0", "proto", "tcp", "from", "10.0.0.0/8",
		"to", "any", "port", "22", "comment", "admins"}
	if got := rule.Args(); !reflect.DeepEqual(got, want) {
		t.Errorf("Args() = %q, want %q", got, want)
	}
	if got := rule.DeleteArgs(); reflect.DeepEqual(got, want) || got[len(got)-1] != "22" {
		t.Errorf("DeleteArgs() kept the comment: %q", got)
	}
}

const showAdded = `Added user rules (see 'ufw status' for running firewall):
ufw allow 22/tcp
ufw limit 22/tcp comment 'h3xd ssh'
ufw deny out 25
ufw allow from 192.168.1.0/24 to any port 445 proto tcp comment 'smb share'
ufw allow in on eth0 log to any port 80,443 proto tcp
ufw allow 'Nginx Full'
ufw route allow in on eth1 out on eth2
`

func TestParseAdded(t *testing.T) {
	rules := ParseAdded(showAdded)
	if len(rules) != 7 {
		t.Fatalf("got %d rules, want 7: %+v", len(rules), rules)
	}
	check := func(index int, want Rule) {
		got := rules[index]
		got.Raw = ""
		if !reflect.DeepEqual(got, want) {
			t.Errorf("rule %d = %+v, want %+v", index, got, want)
		}
	}
	check(0, Rule{Action: "allow", Direction: "in", Proto: "tcp", Port: "22"})
	check(1, Rule{Action: "limit", Direction: "in", Proto: "tcp", Port: "22", Comment: "h3xd ssh"})
	check(2, Rule{Action: "deny", Direction: "out", Port: "25"})
	check(3, Rule{Action: "allow", Direction: "in", Proto: "tcp", From: "192.168.1.0/24",
		Port: "445", Comment: "smb share"})
	check(4, Rule{Action: "allow", Direction: "in", Interface: "eth0", Log: "log", Proto: "tcp",
		Port: "80,443"})
	check(5, Rule{Action: "allow", Direction: "in", App: "Nginx Full"})
	if !rules[6].Unsupported || rules[6].Raw != "ufw route allow in on eth1 out on eth2" {
		t.Errorf("route rule = %+v, want it kept as unsupported text", rules[6])
	}
	// The simple and the full syntax of the same rule are the same rule.
	full := Rule{Action: "allow", Direction: "in", Proto: "tcp", Port: "22"}
	if rules[0].Key() != full.Key() {
		t.Errorf("keys differ: %q and %q", rules[0].Key(), full.Key())
	}
}

func TestPresets(t *testing.T) {
	for _, p := range Presets {
		for _, rule := range p.Rules {
			if err := rule.Validate(); err != nil {
				t.Errorf("preset %s: %v", p.ID, err)
			}
		}
	}
	ssh, _ := FindPreset("ssh")
	if !ssh.Enabled(ParseAdded(showAdded)) {
		t.Error("the ssh preset should count as enabled")
	}
	web, _ := FindPreset("web")
	if web.Enabled(ParseAdded(showAdded)) {
		t.Error("the web preset should not count as enabled")
	}
}

func TestAllows(t *testing.T) {
	rules := ParseAdded(showAdded)
	if !Allows(rules, "tcp", 22) {
		t.Error("22/tcp is allowed")
	}
	if Allows(rules, "tcp", 445) {
		t.Error("445/tcp is only allowed from one network")
	}
	if Allows(rules, "udp", 22) {
		t.Error("22/udp is not allowed")
	}
}

func TestParseListening(t *testing.T) {
	output := `tcp   LISTEN 0      128          0.0.0.0:22        0.0.0.0:*    users:(("sshd",pid=520,fd=3))
tcp   LISTEN 0      128             [::]:22           [::]:*    users:(("sshd",pid=520,fd=4))
udp   UNCONN 0      0            0.0.0.0:5353      0.0.0.0:*    users:(("avahi-daemon",pid=410,fd=12))
tcp   LISTEN 0      4096   127.0.0.53%lo:53        0.0.0.0:*    users:(("systemd-resolve",pid=300,fd=15))
tcp   LISTEN 0      4096       127.0.0.1:631       0.0.0.0:*    users:(("cupsd",pid=600,fd=7))
tcp   LISTEN 0      4096           [::1]:631          [::]:*    users:(("cupsd",pid=600,fd=6))
`
	want := []Listener{
		{Process: "sshd", Proto: "tcp", Port: 22},
		{Process: "avahi-daemon", Proto: "udp", Port: 5353},
		{Process: "systemd-resolve", Proto: "tcp", Port: 53, LocalOnly: true},
		{Process: "cupsd", Proto: "tcp", Port: 631, LocalOnly: true},
	}
	if got := ParseListening(output); !reflect.DeepEqual(got, want) {
		t.Errorf("ParseListening() = %+v, want %+v", got, want)
	}
}

func TestParseBlocked(t *testing.T) {
	output := `{"__REALTIME_TIMESTAMP":"1792915200000000","MESSAGE":"[UFW BLOCK] IN=enp1s0 OUT= MAC=52:54 SRC=10.0.2.2 DST=10.0.2.15 LEN=60 PROTO=TCP SPT=51515 DPT=22 WINDOW=64240"}
{"__REALTIME_TIMESTAMP":"1792915201000000","MESSAGE":[91,85,70,87]}
{"__REALTIME_TIMESTAMP":"1792915202000000","MESSAGE":"[UFW BLOCK] IN= OUT=enp1s0 SRC=10.0.2.15 DST=1.1.1.1 LEN=60 PROTO=UDP SPT=40000 DPT=53"}
`
	got := ParseBlocked(output)
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(got), got)
	}
	if !got[0].Incoming || got[0].Source != "10.0.2.2" || got[0].Proto != "tcp" ||
		got[0].Port != 22 || got[0].Iface != "enp1s0" {
		t.Errorf("first entry = %+v", got[0])
	}
	if got[1].Incoming || got[1].Iface != "enp1s0" || got[1].Port != 53 {
		t.Errorf("second entry = %+v", got[1])
	}
}

func TestParseConfig(t *testing.T) {
	values := ParseConfig("# comment\nENABLED=yes\nDEFAULT_INPUT_POLICY=\"DROP\"\n")
	if values["ENABLED"] != "yes" || Policy(values["DEFAULT_INPUT_POLICY"]) != "deny" {
		t.Errorf("ParseConfig() = %v", values)
	}
}
