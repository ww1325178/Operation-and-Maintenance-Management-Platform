package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"ops-platform/internal/protocol"
)

var ErrDeviceOffline = errors.New("设备不在线")

// RemoteSession is one active remote-control session. A single browser
// (noVNC) may be attached at a time; raw VNC bytes flow browser<->agent
// through the agent's existing WebSocket channel.
type RemoteSession struct {
	ID       string
	DeviceID string
	UserID   int64
	Scale    float64

	mu      sync.Mutex
	browser *websocket.Conn
	ended   bool
}

type RemoteManager struct {
	pool *pgxpool.Pool
	hub  *Hub

	mu       sync.Mutex
	sessions map[string]*RemoteSession
}

func NewRemoteManager(pool *pgxpool.Pool, hub *Hub) *RemoteManager {
	return &RemoteManager{pool: pool, hub: hub, sessions: map[string]*RemoteSession{}}
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Start creates a session record. The agent's VNC engine is started lazily
// when the browser actually connects (ServeWS), so RFB handshake bytes are
// never dropped.
func (rm *RemoteManager) Start(ctx context.Context, deviceID string, userID int64, scale float64) (string, error) {
	if !rm.hub.Has(deviceID) {
		return "", ErrDeviceOffline
	}
	id := newID()
	s := &RemoteSession{ID: id, DeviceID: deviceID, UserID: userID, Scale: scale}
	rm.mu.Lock()
	rm.sessions[id] = s
	rm.mu.Unlock()

	_, _ = rm.pool.Exec(ctx,
		`INSERT INTO remote_sessions(session_id, device_id, user_id) VALUES($1,$2,$3)`,
		id, deviceID, userID)
	rm.audit(ctx, userID, deviceID, "remote_start", id)
	return id, nil
}

func (rm *RemoteManager) audit(ctx context.Context, userID int64, deviceID, action, sessionID string) {
	detail, _ := json.Marshal(map[string]string{"session_id": sessionID})
	_, _ = rm.pool.Exec(ctx,
		`INSERT INTO audit_log(user_id, device_id, action, detail) VALUES($1,$2,$3,$4)`,
		userID, deviceID, action, detail)
}

func (rm *RemoteManager) get(id string) *RemoteSession {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.sessions[id]
}

func (rm *RemoteManager) findByDevice(deviceID string) string {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	for id, s := range rm.sessions {
		if s.DeviceID == deviceID && !s.ended {
			return id
		}
	}
	return ""
}

// OnReady is informational; the API no longer blocks on it.
func (rm *RemoteManager) OnReady(v protocol.VNCReady) {
	if !v.OK {
		log.Printf("远程会话 %s 启动失败: %s", v.SessionID, v.Error)
	}
}

// OnAgentStream routes a raw VNC payload arriving from an agent to the
// attached browser for that session.
func (rm *RemoteManager) OnAgentStream(sessionID string, payload []byte) {
	s := rm.get(sessionID)
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.browser != nil {
		_ = s.browser.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_ = s.browser.WriteMessage(websocket.BinaryMessage, payload)
	}
}

// AgentGone tears down every session belonging to a device that went offline.
func (rm *RemoteManager) AgentGone(deviceID string) {
	rm.mu.Lock()
	var ids []string
	for id, s := range rm.sessions {
		if s.DeviceID == deviceID {
			ids = append(ids, id)
		}
	}
	rm.mu.Unlock()
	for _, id := range ids {
		rm.teardown(id)
	}
}

func (rm *RemoteManager) teardown(sessionID string) {
	rm.mu.Lock()
	s := rm.sessions[sessionID]
	delete(rm.sessions, sessionID)
	rm.mu.Unlock()
	if s == nil {
		return
	}
	log.Printf("[remote] teardown session=%s", sessionID)

	s.mu.Lock()
	if s.ended {
		s.mu.Unlock()
		return
	}
	s.ended = true
	b := s.browser
	s.browser = nil
	s.mu.Unlock()

	if m, err := protocol.Marshal(protocol.MsgStopVNC, s.DeviceID, protocol.StopVNC{SessionID: sessionID}); err == nil {
		rm.hub.Send(s.DeviceID, m)
	}
	if b != nil {
		_ = b.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = rm.pool.Exec(ctx,
		`UPDATE remote_sessions SET ended_at=now() WHERE session_id=$1 AND ended_at IS NULL`, sessionID)
	rm.audit(ctx, s.UserID, s.DeviceID, "remote_end", sessionID)
}

// ServeWS relays raw VNC bytes between the browser (noVNC) and the agent.
// Auth is possession of the unguessable session id, single-use and short-lived.
func (rm *RemoteManager) ServeWS(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	s := rm.get(sessionID)
	if s == nil {
		writeErr(w, http.StatusUnauthorized, "会话不存在或已结束")
		return
	}
	if !rm.hub.Has(s.DeviceID) {
		writeErr(w, http.StatusConflict, "设备不在线")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.mu.Lock()
	if s.browser != nil {
		_ = s.browser.Close()
	}
	s.browser = conn
	s.mu.Unlock()
	log.Printf("[remote] browser connected session=%s", sessionID)

	// Bring up the agent's VNC engine now that the browser is attached.
	if m, err := protocol.Marshal(protocol.MsgStartVNC, s.DeviceID, protocol.StartVNC{SessionID: sessionID, Scale: s.Scale}); err == nil {
		rm.hub.Send(s.DeviceID, m)
	}

	defer rm.teardown(sessionID)

	for {
		mt, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if mt == websocket.BinaryMessage || mt == websocket.TextMessage {
			rm.hub.SendBinary(s.DeviceID, protocol.NewStreamFrame(sessionID, data))
		}
	}
}

func (s *Server) handleRemoteStart(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	c := claimsFrom(r.Context())
	if c == nil {
		writeErr(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req struct {
		Scale float64 `json:"scale"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	scale := req.Scale
	if scale <= 0 || scale > 1 {
		scale = 1.0
	}
	sessionID, err := s.remote.Start(r.Context(), deviceID, c.UserID, scale)
	if err != nil {
		if errors.Is(err, ErrDeviceOffline) {
			writeErr(w, http.StatusConflict, "设备不在线")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[remote] start device=%s session=%s", deviceID, sessionID)
	writeJSON(w, http.StatusOK, map[string]string{"session_id": sessionID})
}

func (s *Server) handleRemoteStop(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	sid := s.remote.findByDevice(deviceID)
	if sid == "" {
		writeErr(w, http.StatusNotFound, "没有进行中的远程会话")
		return
	}
	s.remote.teardown(sid)
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}
