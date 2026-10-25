package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/h3xdOS/h3xdos/firewall/internal/protocol"
)

const helperPath = "/usr/lib/h3xd/firewall-helper"

// pkexec exits 126 when the password dialog is dismissed and 127 when not authorised.
const (
	pkexecDismissed     = 126
	pkexecNotAuthorised = 127
)

var errDeclined = errors.New("the password was not given")

// helper is the root process that changes the firewall; one runs while the app is open.
type helper struct {
	command *exec.Cmd
	stdin   io.WriteCloser
	encoder *json.Encoder
	decoder *json.Decoder
	mutex   sync.Mutex
	waited  bool
	exitErr error
}

// startHelper asks for the password through polkit and returns the helper with its first
// answer, which shows that it runs.
func startHelper() (*helper, protocol.Response, error) {
	command := exec.Command("pkexec", helperPath)
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, protocol.Response{}, err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, protocol.Response{}, err
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return nil, protocol.Response{}, fmt.Errorf("cannot start pkexec: %w", err)
	}
	h := &helper{command: command, stdin: stdin, encoder: json.NewEncoder(stdin),
		decoder: json.NewDecoder(stdout)}
	response, err := h.call(protocol.Request{Op: protocol.OpStatus})
	if err != nil {
		return nil, protocol.Response{}, err
	}
	return h, response, nil
}

func (h *helper) call(request protocol.Request) (protocol.Response, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	var response protocol.Response
	if err := h.encoder.Encode(request); err != nil {
		return response, h.stopped(err)
	}
	if err := h.decoder.Decode(&response); err != nil {
		return response, h.stopped(err)
	}
	return response, nil
}

// stopped explains why the helper went away; pkexec's exit status tells a declined
// password from a failure.
func (h *helper) stopped(cause error) error {
	if !h.waited {
		h.waited = true
		h.exitErr = h.command.Wait()
	}
	var exit *exec.ExitError
	if errors.As(h.exitErr, &exit) &&
		(exit.ExitCode() == pkexecDismissed || exit.ExitCode() == pkexecNotAuthorised) {
		return errDeclined
	}
	return fmt.Errorf("the firewall helper stopped: %v", cause)
}

// close ends the helper by closing its input; it exits when it reads the end.
func (h *helper) close() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if err := h.stdin.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "h3xd-firewall: closing the helper: %v\n", err)
	}
	if !h.waited {
		h.waited = true
		h.exitErr = h.command.Wait()
	}
}
