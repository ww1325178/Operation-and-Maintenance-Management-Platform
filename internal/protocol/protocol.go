// Package protocol defines the wire format shared between the ops-platform
// agent and server. Both sides must be kept in sync; when message shapes
// change, bump the protocol and redeploy agents before the server.
package protocol

import (
	"encoding/json"
)

// Message is the envelope for every agent<->server frame.
type Message struct {
	Type     string          `json:"type"`
	DeviceID string          `json:"device_id,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

// Agent -> server message types.
const (
	MsgHello     = "hello"
	MsgHeartbeat = "heartbeat"
	MsgVNCReady  = "vnc_ready"
)

// Server -> agent message types.
const (
	MsgStartVNC = "start_vnc"
	MsgStopVNC  = "stop_vnc"
	MsgPing     = "ping"
)

// Interface is a single network adapter as reported by the agent.
type Interface struct {
	Name string   `json:"name"`
	MAC  string   `json:"mac"`
	IPs  []string `json:"ips"`
}

// Hello is the full inventory the agent sends on connect and after reconnect.
type Hello struct {
	Hostname     string      `json:"hostname"`
	OSName       string      `json:"os_name"`
	OSVersion    string      `json:"os_version"`
	OSBuild      string      `json:"os_build"`
	AgentVersion string      `json:"agent_version"`
	Interfaces   []Interface `json:"interfaces"`
}

// Heartbeat is the periodic keep-alive payload.
type Heartbeat struct {
	Ts int64 `json:"ts"` // unix milliseconds
}

// StartVNC instructs an agent to start a VNC session for the given id.
// Scale is the transmitted-resolution factor (1.0 / 0.5 / 0.25).
type StartVNC struct {
	SessionID string  `json:"session_id"`
	Scale     float64 `json:"scale"`
}

// StopVNC instructs an agent to tear down a session's VNC server.
type StopVNC struct {
	SessionID string `json:"session_id"`
}

// VNCReady is the agent's acknowledgement that a VNC session is up (or failed).
type VNCReady struct {
	SessionID string `json:"session_id"`
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
}

// Stream framing for tunneling raw VNC bytes over the agent channel.
// Binary WebSocket messages begin with the magic byte, followed by a 32-byte
// session id, then the raw payload. Text messages are JSON control frames.
const (
	StreamMagic      = 0x01
	StreamHeaderLen  = 1 + 32
)

// NewStreamFrame wraps payload into a stream frame for the given session.
func NewStreamFrame(sessionID string, payload []byte) []byte {
	frame := make([]byte, StreamHeaderLen+len(payload))
	frame[0] = StreamMagic
	copy(frame[1:33], sessionID)
	copy(frame[33:], payload)
	return frame
}

// ParseStreamFrame extracts the session id and payload from a stream frame.
// ok is false when the frame is malformed.
func ParseStreamFrame(b []byte) (sessionID string, payload []byte, ok bool) {
	if len(b) < StreamHeaderLen || b[0] != StreamMagic {
		return "", nil, false
	}
	return string(b[1:33]), b[33:], true
}

// Marshal builds a Message carrying payload.
func Marshal(t string, deviceID string, payload any) ([]byte, error) {
	var data json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		data = b
	}
	return json.Marshal(Message{Type: t, DeviceID: deviceID, Data: data})
}

// Unmarshal parses a raw frame into a Message.
func Unmarshal(b []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(b, &m)
	return m, err
}

// UnmarshalData decodes m.Data into v.
func UnmarshalData(m Message, v any) error {
	if len(m.Data) == 0 {
		return nil
	}
	return json.Unmarshal(m.Data, v)
}
