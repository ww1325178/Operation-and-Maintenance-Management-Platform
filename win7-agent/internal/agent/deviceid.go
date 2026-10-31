package agent

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
)

// LoadOrCreateDeviceID returns a stable per-machine id, generating and
// persisting one on first run. An explicit override is used verbatim.
func LoadOrCreateDeviceID(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "device_id")
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		return string(b), nil
	}
	id, err := newUUID()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(id), 0o644); err != nil {
		return "", err
	}
	return id, nil
}

func dataDir() (string, error) {
	base := os.Getenv("PROGRAMDATA")
	if base == "" {
		base = filepath.Join(os.Getenv("SystemDrive"), "ProgramData")
	}
	if base == "" {
		base = "."
	}
	return filepath.Join(base, "opsagent"), nil
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[:]), nil
}
