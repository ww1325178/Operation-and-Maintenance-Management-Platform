package server

import (
	"time"

	"ops-platform/internal/protocol"
)

// Device is a computer as stored and served to the web UI.
type Device struct {
	ID           int64                 `json:"id"`
	DeviceID     string                `json:"device_id"`
	Hostname     string                `json:"hostname"`
	OSName       string                `json:"os_name"`
	OSVersion    string                `json:"os_version"`
	OSBuild      string                `json:"os_build"`
	AgentVersion string                `json:"agent_version"`
	LoggedInUser string                `json:"logged_in_user"`
	Status       string                `json:"status"`
	LastSeenAt   *time.Time            `json:"last_seen_at"`
	FirstSeenAt  time.Time             `json:"first_seen_at"`
	Interfaces   []protocol.Interface  `json:"interfaces"`
}
