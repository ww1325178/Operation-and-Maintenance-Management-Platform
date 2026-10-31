package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"ops-platform/internal/protocol"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
	// Agents connect with a token; origin checks are not a meaningful boundary
	// here, so accept any origin to keep NAT/proxied setups working.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// outMsg is a queued frame on an agent connection: either a JSON control
// message (text) or a raw VNC stream frame (binary).
type outMsg struct {
	mt   int
	data []byte
}

// AgentConn is one live agent connection.
type AgentConn struct {
	DeviceID  string
	conn      *websocket.Conn
	send      chan outMsg
	done      chan struct{}
	hub       *Hub
	rm        *RemoteManager
	closeOnce sync.Once
}

// Hub tracks live agent connections keyed by device id.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]*AgentConn
}

func NewHub() *Hub {
	return &Hub{conns: map[string]*AgentConn{}}
}

func (h *Hub) add(c *AgentConn) {
	h.mu.Lock()
	if old, ok := h.conns[c.DeviceID]; ok {
		old.close()
	}
	h.conns[c.DeviceID] = c
	h.mu.Unlock()
}

func (h *Hub) remove(deviceID string) {
	h.mu.Lock()
	delete(h.conns, deviceID)
	h.mu.Unlock()
}

// Has reports whether an agent is currently connected.
func (h *Hub) Has(deviceID string) bool {
	h.mu.RLock()
	_, ok := h.conns[deviceID]
	h.mu.RUnlock()
	return ok
}

func (h *Hub) conn(deviceID string) *AgentConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.conns[deviceID]
}

// Send queues an already-marshalled JSON control message to a connected agent.
func (h *Hub) Send(deviceID string, data []byte) bool {
	return h.sendRaw(deviceID, outMsg{mt: websocket.TextMessage, data: data})
}

// SendBinary queues a raw binary stream frame to a connected agent.
func (h *Hub) SendBinary(deviceID string, data []byte) bool {
	return h.sendRaw(deviceID, outMsg{mt: websocket.BinaryMessage, data: data})
}

func (h *Hub) sendRaw(deviceID string, m outMsg) bool {
	c := h.conn(deviceID)
	if c == nil {
		return false
	}
	select {
	case c.send <- m:
		return true
	case <-c.done:
		return false
	}
}

func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}

func (c *AgentConn) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (c *AgentConn) writeLoop() {
	for {
		select {
		case m := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(m.mt, m.data); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

// handleAgentWS is the long-lived agent channel: authenticate, upgrade, then
// process JSON control frames (text) and VNC stream frames (binary).
func (s *Server) handleAgentWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if s.cfg.AgentToken == "" || token != s.cfg.AgentToken {
		writeErr(w, http.StatusUnauthorized, "invalid agent token")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	ac := &AgentConn{
		conn: conn,
		send: make(chan outMsg, 128),
		done: make(chan struct{}),
		hub:  s.hub,
		rm:   s.remote,
	}
	go ac.writeLoop()

	registered := false
	defer func() {
		if registered {
			s.hub.remove(ac.DeviceID)
			s.markOffline(contextBackground(), ac.DeviceID)
			s.remote.AgentGone(ac.DeviceID)
		}
		ac.close()
	}()

	for {
		mt, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if mt == websocket.BinaryMessage {
			if sid, payload, ok := protocol.ParseStreamFrame(data); ok {
				s.remote.OnAgentStream(sid, payload)
			}
			continue
		}

		var msg protocol.Message
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		switch msg.Type {
		case protocol.MsgHello:
			var hello protocol.Hello
			if protocol.UnmarshalData(msg, &hello) != nil {
				continue
			}
			if s.upsertDevice(r.Context(), msg.DeviceID, hello) == nil {
				ac.DeviceID = msg.DeviceID
				s.hub.add(ac)
				registered = true
			}
		case protocol.MsgHeartbeat:
			s.touchDevice(r.Context(), msg.DeviceID)
		case protocol.MsgVNCReady:
			var v protocol.VNCReady
			if protocol.UnmarshalData(msg, &v) == nil {
				s.remote.OnReady(v)
			}
		}
	}
}
