// Package ufw models the firewall rules H3XD Firewall manages: what a rule may contain, how
// it is checked before it reaches ufw as root, the ufw arguments that express it, and the
// parsing of what ufw, ss and the kernel log report.
package ufw

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

// Rule is one ufw rule. Empty addresses mean any address.
type Rule struct {
	Action    string `json:"action"`
	Direction string `json:"direction"`
	Interface string `json:"interface,omitempty"`
	Log       string `json:"log,omitempty"`
	Proto     string `json:"proto,omitempty"`
	From      string `json:"from,omitempty"`
	FromPort  string `json:"from_port,omitempty"`
	To        string `json:"to,omitempty"`
	Port      string `json:"port,omitempty"`
	// App is a ufw application profile; H3XD does not create them, but keeps those it finds.
	App     string `json:"app,omitempty"`
	Comment string `json:"comment,omitempty"`
	// Raw is the "ufw ..." line the rule was read from; Unsupported rules only have that.
	Raw         string `json:"raw,omitempty"`
	Unsupported bool   `json:"unsupported,omitempty"`
}

var (
	Actions    = []string{"allow", "deny", "reject", "limit"}
	Directions = []string{"in", "out"}
	Protocols  = []string{"", "tcp", "udp"}
	LogModes   = []string{"", "log", "log-all"}
)

const (
	maxPort        = 65535
	maxComment     = 64
	maxPortEntries = 15
)

var (
	// Interface names as the kernel allows them, at most 15 characters.
	interfacePattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,15}$`)
	appPattern       = regexp.MustCompile(`^[A-Za-z0-9 ._-]{1,64}$`)
	portEntry        = regexp.MustCompile(`^([0-9]{1,5})(?::([0-9]{1,5}))?$`)
)

func oneOf(value string, allowed []string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

// CheckPorts accepts "22", "6000:6007" and lists such as "80,443"; multi reports whether
// the value names more than one port, which ufw only takes with a protocol.
func CheckPorts(value string) (multi bool, err error) {
	entries := strings.Split(value, ",")
	if len(entries) > maxPortEntries {
		return false, fmt.Errorf("at most %d ports or ranges", maxPortEntries)
	}
	for _, entry := range entries {
		match := portEntry.FindStringSubmatch(entry)
		if match == nil {
			return false, fmt.Errorf("%q is not a port, a range such as 6000:6007 or a list such as 80,443", value)
		}
		low, _ := strconv.Atoi(match[1])
		if low < 1 || low > maxPort {
			return false, fmt.Errorf("port %d is outside 1 to %d", low, maxPort)
		}
		if match[2] != "" {
			high, _ := strconv.Atoi(match[2])
			if high <= low || high > maxPort {
				return false, fmt.Errorf("range %s must go up and end at %d at most", entry, maxPort)
			}
			multi = true
		}
	}
	return multi || len(entries) > 1, nil
}

// CheckAddress accepts an IPv4 or IPv6 address or network; empty means any.
func CheckAddress(value string) error {
	if value == "" {
		return nil
	}
	if net.ParseIP(value) != nil {
		return nil
	}
	if _, _, err := net.ParseCIDR(value); err == nil {
		return nil
	}
	return fmt.Errorf("%q is not an IP address or a network such as 192.168.1.0/24", value)
}

// Validate checks everything a rule carries before ufw sees it as root.
func (r Rule) Validate() error {
	if r.Unsupported {
		return errors.New("this rule can only be removed")
	}
	if !oneOf(r.Action, Actions) {
		return fmt.Errorf("unknown action %q", r.Action)
	}
	if !oneOf(r.Direction, Directions) {
		return fmt.Errorf("unknown direction %q", r.Direction)
	}
	if !oneOf(r.Proto, Protocols) {
		return fmt.Errorf("unknown protocol %q", r.Proto)
	}
	if !oneOf(r.Log, LogModes) {
		return fmt.Errorf("unknown log mode %q", r.Log)
	}
	if r.Interface != "" && !interfacePattern.MatchString(r.Interface) {
		return fmt.Errorf("%q is not a network interface name", r.Interface)
	}
	if r.App != "" && (!appPattern.MatchString(r.App) || r.Port != "") {
		return fmt.Errorf("%q is not an application profile", r.App)
	}
	for _, port := range []string{r.Port, r.FromPort} {
		if port == "" {
			continue
		}
		multi, err := CheckPorts(port)
		if err != nil {
			return err
		}
		if multi && r.Proto == "" {
			return errors.New("several ports need TCP or UDP")
		}
	}
	if err := CheckAddress(r.From); err != nil {
		return err
	}
	if err := CheckAddress(r.To); err != nil {
		return err
	}
	if len(r.Comment) > maxComment {
		return fmt.Errorf("the comment is longer than %d characters", maxComment)
	}
	for _, char := range r.Comment {
		if char < ' ' || char == 0x7f || char == '\'' || char == '"' {
			return errors.New("the comment may not contain quotes or control characters")
		}
	}
	return nil
}

func orAny(address string) string {
	if address == "" {
		return "any"
	}
	return address
}

func (r Rule) args(withComment bool) []string {
	args := []string{r.Action, r.Direction}
	if r.Interface != "" {
		args = append(args, "on", r.Interface)
	}
	if r.Log != "" {
		args = append(args, r.Log)
	}
	if r.Proto != "" {
		args = append(args, "proto", r.Proto)
	}
	args = append(args, "from", orAny(r.From))
	if r.FromPort != "" {
		args = append(args, "port", r.FromPort)
	}
	args = append(args, "to", orAny(r.To))
	if r.Port != "" {
		args = append(args, "port", r.Port)
	}
	if r.App != "" {
		args = append(args, "app", r.App)
	}
	if withComment && r.Comment != "" {
		args = append(args, "comment", r.Comment)
	}
	return args
}

// Args is the rule in ufw's full syntax, ready to follow "ufw" or "ufw insert N".
func (r Rule) Args() []string {
	return r.args(true)
}

// DeleteArgs follows "ufw delete"; ufw matches rules without their comment.
func (r Rule) DeleteArgs() []string {
	return r.args(false)
}

// Key identifies what a rule does, without its comment and logging.
func (r Rule) Key() string {
	if r.Unsupported {
		return "raw:" + r.Raw
	}
	return strings.Join([]string{r.Action, r.Direction, r.Interface, r.Proto, r.From,
		r.FromPort, r.To, r.Port, r.App}, "|")
}

// Target describes what the rule matches, for the rule list.
func (r Rule) Target() string {
	if r.Unsupported {
		return strings.TrimPrefix(r.Raw, "ufw ")
	}
	var what string
	switch {
	case r.App != "":
		what = r.App
	case r.Port != "" && r.Proto != "":
		what = r.Port + "/" + r.Proto
	case r.Port != "":
		what = "port " + r.Port
	case r.Proto != "":
		what = "all " + strings.ToUpper(r.Proto)
	default:
		what = "all traffic"
	}
	parts := []string{what}
	if r.From != "" {
		from := "from " + r.From
		if r.FromPort != "" {
			from += " port " + r.FromPort
		}
		parts = append(parts, from)
	}
	if r.To != "" {
		parts = append(parts, "to "+r.To)
	}
	if r.Interface != "" {
		parts = append(parts, "on "+r.Interface)
	}
	return strings.Join(parts, " ")
}
