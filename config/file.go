package config

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ConfigFilePath returns the path to the user's config file, respecting XDG_CONFIG_HOME.
func ConfigFilePath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "config.toml"
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "monkey", "config.toml")
}

// parseConfigFile reads a simple TOML-subset config from r.
// Supports string values (key = "value") and integer values (key = 123).
// Lines starting with # and blank lines are ignored.
// Returns a map of key → string-value.
func parseConfigFile(r io.Reader) (map[string]string, error) {
	kv := make(map[string]string)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		// Unquote string values.
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		if key != "" {
			kv[key] = val
		}
	}
	return kv, scanner.Err()
}

// LoadConfigFile reads the config file at path and returns its key-value pairs.
// If the file does not exist, an empty map is returned without error.
func LoadConfigFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	defer f.Close()
	return parseConfigFile(f)
}
