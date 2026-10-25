package ufw

import (
	"bufio"
	"encoding/json"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// tokens splits a "ufw ..." line as ufw prints it: words, with single-quoted parts such
// as comments and application names kept whole.
func tokens(line string) []string {
	var out []string
	var current strings.Builder
	quoted, started := false, false
	for _, char := range line {
		switch {
		case char == '\'':
			quoted = !quoted
			started = true
		case char == ' ' && !quoted:
			if started {
				out = append(out, current.String())
				current.Reset()
				started = false
			}
		default:
			current.WriteRune(char)
			started = true
		}
	}
	if started {
		out = append(out, current.String())
	}
	return out
}

// RawArgs are the words of the "ufw ..." line a rule was read from, without "ufw", for
// removing rules H3XD cannot otherwise express.
func (r Rule) RawArgs() []string {
	return tokens(strings.TrimPrefix(r.Raw, "ufw "))
}

var simpleTarget = regexp.MustCompile(`^([0-9,:]+)(?:/(tcp|udp))?$`)

func anyToEmpty(address string) string {
	if address == "any" {
		return ""
	}
	return address
}

// parseRule reads the words after "ufw" in both the simple ("allow 22/tcp") and the full
// syntax ("allow in on eth0 from 10.0.0.0/8 to any port 22 proto tcp comment 'x'").
func parseRule(words []string) (Rule, bool) {
	var rule Rule
	if len(words) == 0 || !oneOf(words[0], Actions) {
		return rule, false
	}
	rule.Action, rule.Direction = words[0], "in"
	i := 1
	next := func() (string, bool) {
		if i >= len(words) {
			return "", false
		}
		i++
		return words[i-1], true
	}
	for i < len(words) {
		word, _ := next()
		var ok bool
		switch word {
		case "in", "out":
			rule.Direction = word
		case "on":
			rule.Interface, ok = next()
		case "log", "log-all":
			rule.Log = word
		case "proto":
			rule.Proto, ok = next()
		case "from":
			var address string
			address, ok = next()
			rule.From = anyToEmpty(address)
			if ok && i < len(words) && words[i] == "port" {
				next()
				rule.FromPort, ok = next()
			} else if ok && i < len(words) && words[i] == "app" {
				// Source application profiles are rare; H3XD shows such rules as text.
				return rule, false
			}
		case "to":
			var address string
			address, ok = next()
			rule.To = anyToEmpty(address)
			if ok && i < len(words) && words[i] == "port" {
				next()
				rule.Port, ok = next()
			} else if ok && i < len(words) && words[i] == "app" {
				next()
				rule.App, ok = next()
			}
		case "comment":
			rule.Comment, ok = next()
		default:
			// The simple syntax names a port or an application profile right away.
			if match := simpleTarget.FindStringSubmatch(word); match != nil {
				rule.Port, rule.Proto = match[1], match[2]
			} else {
				rule.App = word
			}
			ok = true
		}
		if word == "in" || word == "out" || word == "log" || word == "log-all" {
			ok = true
		}
		if !ok {
			return rule, false
		}
	}
	return rule, true
}

// ParseAdded reads "ufw show added", which lists the rules in order whether or not the
// firewall is running.
func ParseAdded(output string) []Rule {
	var rules []Rule
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "ufw ") {
			continue
		}
		rule, ok := parseRule(tokens(strings.TrimPrefix(line, "ufw ")))
		if !ok || rule.Validate() != nil {
			rule = Rule{Unsupported: true}
		}
		rule.Raw = line
		rules = append(rules, rule)
	}
	return rules
}

// ParseConfig reads KEY=value files such as /etc/ufw/ufw.conf and /etc/default/ufw.
func ParseConfig(text string) map[string]string {
	values := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if found {
			values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
		}
	}
	return values
}

// Policy turns an iptables target from /etc/default/ufw into ufw's word for it.
func Policy(target string) string {
	switch target {
	case "ACCEPT":
		return "allow"
	case "REJECT":
		return "reject"
	default:
		return "deny"
	}
}

// Listener is a program waiting for connections, from ss.
type Listener struct {
	Process string `json:"process"`
	Proto   string `json:"proto"`
	Port    int    `json:"port"`
	// LocalOnly listeners only take connections from this computer.
	LocalOnly bool `json:"local_only"`
}

var processName = regexp.MustCompile(`users:\(\("([^"]+)"`)

func loopback(address string) bool {
	address = strings.Trim(address, "[]")
	if cut := strings.IndexByte(address, '%'); cut >= 0 {
		address = address[:cut]
	}
	ip := net.ParseIP(address)
	return ip != nil && ip.IsLoopback()
}

// ParseListening reads "ss -H -tulnp", one entry per program, protocol and port.
func ParseListening(output string) []Listener {
	index := map[string]int{}
	var listeners []Listener
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}
		proto := fields[0]
		if proto != "tcp" && proto != "udp" {
			continue
		}
		local := fields[4]
		cut := strings.LastIndexByte(local, ':')
		if cut < 0 {
			continue
		}
		port, err := strconv.Atoi(local[cut+1:])
		if err != nil || port == 0 {
			continue
		}
		process := "unknown program"
		if match := processName.FindStringSubmatch(scanner.Text()); match != nil {
			process = match[1]
		}
		localOnly := loopback(local[:cut])
		key := process + "|" + proto + "|" + strconv.Itoa(port)
		if at, seen := index[key]; seen {
			listeners[at].LocalOnly = listeners[at].LocalOnly && localOnly
			continue
		}
		index[key] = len(listeners)
		listeners = append(listeners, Listener{Process: process, Proto: proto, Port: port,
			LocalOnly: localOnly})
	}
	return listeners
}

// Blocked is a packet the firewall dropped, from the kernel log.
type Blocked struct {
	Time     time.Time `json:"time"`
	Incoming bool      `json:"incoming"`
	Iface    string    `json:"iface"`
	Source   string    `json:"source"`
	Dest     string    `json:"dest"`
	Proto    string    `json:"proto"`
	Port     int       `json:"port"`
}

var logField = regexp.MustCompile(`\b(IN|OUT|SRC|DST|PROTO|DPT)=(\S*)`)

// ParseBlocked reads "journalctl -k -o json" lines with UFW BLOCK messages, newest last.
func ParseBlocked(output string) []Blocked {
	var blocked []Blocked
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry struct {
			Message   interface{} `json:"MESSAGE"`
			Timestamp string      `json:"__REALTIME_TIMESTAMP"`
		}
		if json.Unmarshal(scanner.Bytes(), &entry) != nil {
			continue
		}
		// journald stores messages that are not valid UTF-8 as byte arrays; UFW's are text.
		message, isText := entry.Message.(string)
		if !isText || !strings.Contains(message, "UFW BLOCK") {
			continue
		}
		fields := map[string]string{}
		for _, match := range logField.FindAllStringSubmatch(message, -1) {
			fields[match[1]] = match[2]
		}
		micros, _ := strconv.ParseInt(entry.Timestamp, 10, 64)
		port, _ := strconv.Atoi(fields["DPT"])
		item := Blocked{Time: time.UnixMicro(micros), Iface: fields["IN"], Source: fields["SRC"],
			Dest: fields["DST"], Proto: strings.ToLower(fields["PROTO"]), Port: port}
		item.Incoming = fields["IN"] != "" && fields["OUT"] == ""
		if !item.Incoming {
			item.Iface = fields["OUT"]
		}
		blocked = append(blocked, item)
	}
	return blocked
}
