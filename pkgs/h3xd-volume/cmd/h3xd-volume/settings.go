package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// settings are kept per account, read by the window and by the volume keys alike.
type settings struct {
	// Louder lets the volume go past 100 %, up to louderLimit.
	Louder bool
}

const louderKey = "louder"

func settingsPath() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(config, "h3xd", "volume.conf"), nil
}

// loadSettings falls back to the defaults when the file is missing or unreadable; an
// unreadable file is logged, as the volume keys must work all the same.
func loadSettings() settings {
	var loaded settings
	path, err := settingsPath()
	if err != nil {
		log.Printf("h3xd-volume: settings: %v", err)
		return loaded
	}
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return loaded
	}
	if err != nil {
		log.Printf("h3xd-volume: settings: %v", err)
		return loaded
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, found := strings.Cut(strings.TrimSpace(scanner.Text()), "=")
		if found && strings.TrimSpace(key) == louderKey {
			loaded.Louder = strings.TrimSpace(value) == "yes"
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("h3xd-volume: settings: %v", err)
	}
	return loaded
}

func (s settings) save() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	value := "no"
	if s.Louder {
		value = "yes"
	}
	content := fmt.Sprintf("# H3XD Volume\n%s=%s\n", louderKey, value)
	return os.WriteFile(path, []byte(content), 0o644)
}

func (s settings) limit() int {
	if s.Louder {
		return louderLimit
	}
	return normalLimit
}
