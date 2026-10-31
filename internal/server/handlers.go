package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxKey string

const claimsKey ctxKey = "claims"

type Server struct {
	cfg    Config
	pool   *pgxpool.Pool
	auth   *Auth
	hub    *Hub
	remote *RemoteManager
}

func New(cfg Config, pool *pgxpool.Pool, auth *Auth, hub *Hub, remote *RemoteManager) *Server {
	return &Server{cfg: cfg, pool: pool, auth: auth, hub: hub, remote: remote}
}

func claimsFrom(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"time":      time.Now().Format(time.RFC3339),
		"agents":    s.hub.Count(),
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad request")
		return
	}
	token, err := s.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			writeErr(w, http.StatusUnauthorized, "用户名或密码错误")
			return
		}
		writeErr(w, http.StatusInternalServerError, "login failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

const deviceSelect = `
	SELECT d.id, d.device_id, d.hostname, d.os_name, d.os_version, d.os_build,
	       d.agent_version, d.logged_in_user, d.status, d.last_seen_at, d.first_seen_at,
	       COALESCE(json_agg(json_build_object('name', ni.name, 'mac', ni.mac, 'ips', ni.ips)
	                         ORDER BY ni.id) FILTER (WHERE ni.id IS NOT NULL), '[]') AS interfaces
	FROM devices d
	LEFT JOIN network_interfaces ni ON ni.device_id = d.device_id`

func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), deviceSelect+` GROUP BY d.id ORDER BY d.hostname`)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	out := []Device{}
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, d)
	}
	if rows.Err() != nil {
		writeErr(w, http.StatusInternalServerError, rows.Err().Error())
		return
	}
	if out == nil {
		out = []Device{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rows, err := s.pool.Query(r.Context(), deviceSelect+` WHERE d.device_id = $1 GROUP BY d.id`, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	if !rows.Next() {
		writeErr(w, http.StatusNotFound, "设备不存在")
		return
	}
	d, err := scanDevice(rows)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDevice(r rowScanner) (Device, error) {
	var (
		id          int64
		deviceID    string
		hostname    string
		osName      string
		osVer       string
		osBuild     string
		agentVer    string
		loggedUser  string
		status      string
		lastSeen    *time.Time
		firstSeen   time.Time
		interfaces  []byte
	)
	err := r.Scan(&id, &deviceID, &hostname, &osName, &osVer, &osBuild,
		&agentVer, &loggedUser, &status, &lastSeen, &firstSeen, &interfaces)
	if err != nil {
		return Device{}, err
	}
	d := Device{
		ID:           id,
		DeviceID:     deviceID,
		Hostname:     hostname,
		OSName:       osName,
		OSVersion:    osVer,
		OSBuild:      osBuild,
		AgentVersion: agentVer,
		LoggedInUser: loggedUser,
		Status:       status,
		LastSeenAt:   lastSeen,
		FirstSeenAt:  firstSeen,
	}
	_ = json.Unmarshal(interfaces, &d.Interfaces)
	return d, nil
}
