// Package protocol is what H3XD Firewall and its root helper send each other: one JSON
// request per line on the helper's standard input, one JSON response per line back.
package protocol

import "github.com/h3xdOS/h3xdos/firewall/internal/ufw"

// Operations the helper carries out.
const (
	OpStatus    = "status"
	OpEnable    = "enable"
	OpDisable   = "disable"
	OpAdd       = "add"
	OpDelete    = "delete"
	OpSetRule   = "set-rule"
	OpPreset    = "preset"
	OpDefault   = "default"
	OpLogging   = "logging"
	OpReset     = "reset"
	OpListening = "listening"
	OpBlocked   = "blocked"
)

// Request is one operation; only the fields it needs are set.
type Request struct {
	Op        string   `json:"op"`
	Rule      ufw.Rule `json:"rule,omitempty"`
	AtTop     bool     `json:"at_top,omitempty"`
	On        bool     `json:"on,omitempty"`
	Preset    string   `json:"preset,omitempty"`
	Direction string   `json:"direction,omitempty"`
	Policy    string   `json:"policy,omitempty"`
	Level     string   `json:"level,omitempty"`
}

// Status is the state of the firewall as the pages show it.
type Status struct {
	Enabled  bool       `json:"enabled"`
	Boot     bool       `json:"boot"`
	Logging  string     `json:"logging"`
	Incoming string     `json:"incoming"`
	Outgoing string     `json:"outgoing"`
	Routed   string     `json:"routed"`
	Rules    []ufw.Rule `json:"rules"`
	// Disabled rules are kept by H3XD, since ufw has no switched-off rules.
	Disabled []ufw.Rule `json:"disabled"`
}

// Response answers a request; Note tells the user about something the helper did on its
// own, such as keeping SSH reachable.
type Response struct {
	OK        bool           `json:"ok"`
	Error     string         `json:"error,omitempty"`
	Note      string         `json:"note,omitempty"`
	Status    *Status        `json:"status,omitempty"`
	Listening []ufw.Listener `json:"listening,omitempty"`
	Blocked   []ufw.Blocked  `json:"blocked,omitempty"`
}
