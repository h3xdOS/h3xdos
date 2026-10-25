package ufw

import "strings"

// Preset is a service the Services page switches on with one click: incoming rules for its
// ports, marked with a comment so they can be told apart from rules added by hand.
type Preset struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Rules   []Rule `json:"rules"`
}

// PresetCommentPrefix starts the comment of every rule a preset adds.
const PresetCommentPrefix = "h3xd "

func preset(id, name, summary, action string, ports ...string) Preset {
	p := Preset{ID: id, Name: name, Summary: summary}
	for _, port := range ports {
		number, proto, _ := strings.Cut(port, "/")
		p.Rules = append(p.Rules, Rule{Action: action, Direction: "in", Proto: proto,
			Port: number, Comment: PresetCommentPrefix + id})
	}
	return p
}

// Presets are the services of the Services page. SSH is limited rather than allowed: ufw
// then blocks an address that opens six connections within 30 seconds.
var Presets = []Preset{
	preset("ssh", "SSH", "Remote shell and file transfer into this computer", "limit", "22/tcp"),
	preset("web", "Web server", "Websites served from this computer, HTTP and HTTPS", "allow",
		"80,443/tcp"),
	preset("kdeconnect", "KDE Connect", "Phone integration with KDE Connect and GSConnect",
		"allow", "1714:1764/tcp", "1714:1764/udp"),
	preset("localsend", "LocalSend", "Send files between devices on the local network", "allow",
		"53317/tcp", "53317/udp"),
	preset("samba", "Windows file sharing", "Shared folders over SMB (Samba)", "allow",
		"139,445/tcp", "137,138/udp"),
	preset("printing", "Printer sharing", "Printers shared from this computer (IPP)", "allow",
		"631/tcp"),
	preset("mdns", "Local discovery", "Find printers and devices on the network (mDNS)", "allow",
		"5353/udp"),
	preset("syncthing", "Syncthing", "Folder sync between your devices", "allow",
		"22000/tcp", "22000/udp", "21027/udp"),
	preset("steam", "Steam Remote Play", "Stream games and copy them between your computers",
		"allow", "27036,27037/tcp", "27031:27036/udp"),
	preset("minecraft", "Minecraft server", "A Minecraft server hosted on this computer",
		"allow", "25565/tcp"),
	preset("rdp", "Remote desktop", "Remote desktop connections over RDP", "allow", "3389/tcp"),
}

// FindPreset returns the preset with the id, or false.
func FindPreset(id string) (Preset, bool) {
	for _, p := range Presets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}

// Enabled reports whether every rule of the preset is among the active rules.
func (p Preset) Enabled(active []Rule) bool {
	keys := map[string]bool{}
	for _, rule := range active {
		keys[rule.Key()] = true
	}
	for _, rule := range p.Rules {
		if !keys[rule.Key()] {
			return false
		}
	}
	return true
}

// Allows reports whether the rules let incoming connections reach a port from anywhere,
// for the list of listening programs.
func Allows(rules []Rule, proto string, port int) bool {
	for _, rule := range rules {
		if rule.Unsupported || rule.Direction != "in" || rule.From != "" || rule.To != "" ||
			rule.Interface != "" || (rule.Action != "allow" && rule.Action != "limit") {
			continue
		}
		if rule.Proto != "" && rule.Proto != proto {
			continue
		}
		if rule.Port == "" && rule.App == "" {
			return true
		}
		if rule.Port != "" && portListed(rule.Port, port) {
			return true
		}
	}
	return false
}

func portListed(list string, port int) bool {
	for _, entry := range strings.Split(list, ",") {
		match := portEntry.FindStringSubmatch(entry)
		if match == nil {
			continue
		}
		low := atoi(match[1])
		high := low
		if match[2] != "" {
			high = atoi(match[2])
		}
		if port >= low && port <= high {
			return true
		}
	}
	return false
}

func atoi(text string) int {
	value := 0
	for _, char := range text {
		value = value*10 + int(char-'0')
	}
	return value
}
