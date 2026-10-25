// Command h3xd-firewall-helper changes the firewall as root on behalf of H3XD Firewall.
// The app starts it through pkexec; it reads one JSON request per line on standard input,
// answers each on standard output and ends when the app closes the pipe. Every request is
// checked here, and ufw only ever gets an argument list, never text for a shell.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/h3xdOS/h3xdos/firewall/internal/protocol"
	"github.com/h3xdOS/h3xdos/firewall/internal/ufw"
)

const (
	ufwConf     = "/etc/ufw/ufw.conf"
	ufwDefaults = "/etc/default/ufw"
	// ufw has no switched-off rules; H3XD keeps them here and gives them back to ufw.
	disabledStore = "/var/lib/h3xd-firewall/disabled.json"
	blockedLimit  = 200
	sshPort       = 22
)

var (
	directions = []string{"incoming", "outgoing", "routed"}
	policies   = []string{"allow", "deny", "reject"}
	logLevels  = []string{"off", "low", "medium", "high", "full"}
)

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// run starts a program with a fixed environment and returns its output; on failure the
// error carries ufw's own "ERROR:" line when there is one.
func run(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	command.Env = []string{"PATH=/usr/bin", "LC_ALL=C"}
	output, err := command.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err == nil {
		return text, nil
	}
	message := err.Error()
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "ERROR: ") {
			message = strings.TrimPrefix(line, "ERROR: ")
			break
		}
		if strings.TrimSpace(line) != "" {
			message = line
		}
	}
	return text, errors.New(message)
}

func readConfig(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "h3xd-firewall-helper: %v\n", err)
		return map[string]string{}
	}
	return ufw.ParseConfig(string(data))
}

func activeRules() ([]ufw.Rule, error) {
	output, err := run("ufw", "show", "added")
	if err != nil {
		return nil, err
	}
	return ufw.ParseAdded(output), nil
}

func loadDisabled() ([]ufw.Rule, error) {
	data, err := os.ReadFile(disabledStore)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rules []ufw.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("%s: %w", disabledStore, err)
	}
	return rules, nil
}

func saveDisabled(rules []ufw.Rule) error {
	if err := os.MkdirAll(filepath.Dir(disabledStore), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}
	temporary := disabledStore + ".new"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, disabledStore)
}

func indexOf(rules []ufw.Rule, rule ufw.Rule) int {
	for i, item := range rules {
		if item.Key() == rule.Key() {
			return i
		}
	}
	return -1
}

func status() (*protocol.Status, error) {
	rules, err := activeRules()
	if err != nil {
		return nil, err
	}
	disabled, err := loadDisabled()
	if err != nil {
		return nil, err
	}
	conf := readConfig(ufwConf)
	defaults := readConfig(ufwDefaults)
	logging := conf["LOGLEVEL"]
	if logging == "" {
		logging = "low"
	}
	boot := exec.Command("systemctl", "is-enabled", "--quiet", "ufw.service").Run() == nil
	return &protocol.Status{
		Enabled:  conf["ENABLED"] == "yes",
		Boot:     boot,
		Logging:  logging,
		Incoming: ufw.Policy(defaults["DEFAULT_INPUT_POLICY"]),
		Outgoing: ufw.Policy(defaults["DEFAULT_OUTPUT_POLICY"]),
		Routed:   ufw.Policy(defaults["DEFAULT_FORWARD_POLICY"]),
		Rules:    rules,
		Disabled: disabled,
	}, nil
}

// addRule appends a rule, or puts it first when atTop asks for it and there is a first.
func addRule(rule ufw.Rule, atTop bool) error {
	if err := rule.Validate(); err != nil {
		return err
	}
	if atTop {
		rules, err := activeRules()
		if err != nil {
			return err
		}
		if len(rules) > 0 {
			_, err := run("ufw", append([]string{"insert", "1"}, rule.Args()...)...)
			// ufw numbers IPv4 and IPv6 rules apart and refuses some positions; such a
			// rule then goes last, which is where ufw would put it anyway.
			if err == nil || !strings.Contains(err.Error(), "position") {
				return err
			}
		}
	}
	_, err := run("ufw", rule.Args()...)
	return err
}

func deleteActive(rule ufw.Rule) error {
	if rule.Unsupported {
		rules, err := activeRules()
		if err != nil {
			return err
		}
		known := false
		for _, item := range rules {
			known = known || item.Raw == rule.Raw
		}
		if !known {
			return errors.New("this rule is no longer in the firewall")
		}
		words := rule.RawArgs()
		if len(words) > 0 && words[0] == "route" {
			_, err = run("ufw", append([]string{"route", "delete"}, words[1:]...)...)
		} else {
			_, err = run("ufw", append([]string{"delete"}, words...)...)
		}
		return err
	}
	if err := rule.Validate(); err != nil {
		return err
	}
	_, err := run("ufw", append([]string{"delete"}, rule.DeleteArgs()...)...)
	return err
}

func sshServerOn() bool {
	active := exec.Command("systemctl", "is-active", "--quiet", "sshd.service").Run() == nil
	enabled := exec.Command("systemctl", "is-enabled", "--quiet", "sshd.service").Run() == nil
	return active || enabled
}

// enable switches the firewall on now and at every boot. A running SSH server keeps a way
// in, since a remote user would otherwise lock themselves out.
func enable() (string, error) {
	note := ""
	rules, err := activeRules()
	if err != nil {
		return "", err
	}
	if sshServerOn() && !ufw.Allows(rules, "tcp", sshPort) {
		ssh, _ := ufw.FindPreset("ssh")
		for _, rule := range ssh.Rules {
			if err := addRule(rule, false); err != nil {
				return "", err
			}
		}
		note = "SSH stays reachable: the SSH rule was added before the firewall went on."
	}
	if _, err := run("systemctl", "enable", "ufw.service"); err != nil {
		return note, err
	}
	_, err = run("ufw", "--force", "enable")
	return note, err
}

func setRule(rule ufw.Rule, on bool) error {
	disabled, err := loadDisabled()
	if err != nil {
		return err
	}
	at := indexOf(disabled, rule)
	if on {
		if at < 0 {
			return errors.New("this rule is not switched off")
		}
		// Blocking rules go back first, so that they again win over the rules that allow.
		blocking := rule.Action == "deny" || rule.Action == "reject"
		if err := addRule(disabled[at], blocking); err != nil {
			return err
		}
		return saveDisabled(append(disabled[:at], disabled[at+1:]...))
	}
	if rule.Unsupported {
		return errors.New("this rule can only be removed")
	}
	if err := deleteActive(rule); err != nil {
		return err
	}
	if at < 0 {
		disabled = append(disabled, rule)
	}
	return saveDisabled(disabled)
}

func deleteRule(rule ufw.Rule) error {
	disabled, err := loadDisabled()
	if err != nil {
		return err
	}
	if at := indexOf(disabled, rule); at >= 0 {
		return saveDisabled(append(disabled[:at], disabled[at+1:]...))
	}
	return deleteActive(rule)
}

func setPreset(id string, on bool) error {
	preset, found := ufw.FindPreset(id)
	if !found {
		return fmt.Errorf("unknown service %q", id)
	}
	rules, err := activeRules()
	if err != nil {
		return err
	}
	disabled, err := loadDisabled()
	if err != nil {
		return err
	}
	for _, rule := range preset.Rules {
		active := indexOf(rules, rule) >= 0
		if on && !active {
			err = addRule(rule, false)
		} else if !on && active {
			err = deleteActive(rule)
		}
		if err != nil {
			return err
		}
		if at := indexOf(disabled, rule); at >= 0 {
			disabled = append(disabled[:at], disabled[at+1:]...)
		}
	}
	return saveDisabled(disabled)
}

func blocked() ([]ufw.Blocked, error) {
	command := exec.Command("journalctl", "-k", "--grep", "UFW BLOCK", "-o", "json",
		"-n", strconv.Itoa(blockedLimit), "--no-pager")
	command.Env = []string{"PATH=/usr/bin", "LC_ALL=C"}
	output, err := command.Output()
	// journalctl exits 1 when nothing matches, which only means nothing was blocked.
	var exit *exec.ExitError
	if err != nil && !(errors.As(err, &exit) && exit.ExitCode() == 1) {
		return nil, fmt.Errorf("journalctl: %w", err)
	}
	return ufw.ParseBlocked(string(output)), nil
}

func handle(request protocol.Request) protocol.Response {
	var err error
	var note string
	response := protocol.Response{}
	switch request.Op {
	case protocol.OpStatus:
	case protocol.OpEnable:
		note, err = enable()
	case protocol.OpDisable:
		_, err = run("ufw", "disable")
	case protocol.OpAdd:
		err = addRule(request.Rule, request.AtTop)
	case protocol.OpDelete:
		err = deleteRule(request.Rule)
	case protocol.OpSetRule:
		err = setRule(request.Rule, request.On)
	case protocol.OpPreset:
		err = setPreset(request.Preset, request.On)
	case protocol.OpDefault:
		if !contains(directions, request.Direction) || !contains(policies, request.Policy) {
			err = fmt.Errorf("unknown default %q for %q", request.Policy, request.Direction)
		} else {
			_, err = run("ufw", "default", request.Policy, request.Direction)
		}
	case protocol.OpLogging:
		if !contains(logLevels, request.Level) {
			err = fmt.Errorf("unknown logging level %q", request.Level)
		} else {
			_, err = run("ufw", "logging", request.Level)
		}
	case protocol.OpReset:
		if _, err = run("ufw", "--force", "reset"); err == nil {
			err = saveDisabled(nil)
		}
	case protocol.OpListening:
		var output string
		if output, err = run("ss", "-H", "-tulnp"); err == nil {
			response.Listening = ufw.ParseListening(output)
		}
	case protocol.OpBlocked:
		response.Blocked, err = blocked()
	default:
		err = fmt.Errorf("unknown request %q", request.Op)
	}
	if err != nil {
		response.Error = err.Error()
	}
	response.Note = note
	// Every answer brings the state as it is now, also after a failed change.
	current, statusErr := status()
	if statusErr != nil && response.Error == "" {
		response.Error = statusErr.Error()
	}
	response.Status = current
	response.OK = response.Error == ""
	return response
}

func main() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "h3xd-firewall-helper: H3XD Firewall starts this through pkexec")
		os.Exit(2)
	}
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for {
		var request protocol.Request
		if err := decoder.Decode(&request); err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintf(os.Stderr, "h3xd-firewall-helper: %v\n", err)
			}
			return
		}
		if err := encoder.Encode(handle(request)); err != nil {
			fmt.Fprintf(os.Stderr, "h3xd-firewall-helper: %v\n", err)
			return
		}
	}
}
