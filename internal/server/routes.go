package server

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
)

// Routes wires the HTTP mux together. /api/* is JSON, /ws/agent is the agent
// channel, and everything else serves the built frontend (SPA fallback).
func Routes(s *Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.Handle("GET /api/devices", s.auth.Middleware(http.HandlerFunc(s.listDevices)))
	mux.Handle("GET /api/devices/{id}", s.auth.Middleware(http.HandlerFunc(s.getDevice)))
	mux.Handle("POST /api/devices/{id}/remote", s.auth.Middleware(http.HandlerFunc(s.handleRemoteStart)))
	mux.Handle("DELETE /api/devices/{id}/remote", s.auth.Middleware(http.HandlerFunc(s.handleRemoteStop)))
	mux.HandleFunc("GET /ws/agent", s.handleAgentWS)
	mux.HandleFunc("GET /ws/remote", s.remote.ServeWS)
	mux.Handle("/", s.staticHandler())
	return mux
}

func (s *Server) staticHandler() http.Handler {
	dist := webDistPath()
	fileServer := http.FileServer(http.Dir(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(dist); err != nil {
			writeJSON(w, http.StatusOK, map[string]string{
				"status": "ok",
				"note":   "前端未构建，请运行 web/ 下的 npm run build",
			})
			return
		}
		if r.URL.Path != "/" {
			p := filepath.Join(dist, filepath.Clean("/"+r.URL.Path))
			if _, err := os.Stat(p); err != nil {
				http.ServeFile(w, r, filepath.Join(dist, "index.html"))
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}

// contextBackground is a small indirection so the defer in handleAgentWS can
// update status after the request context has been cancelled.
func contextBackground() context.Context {
	return context.Background()
}

// webDistPath 返回前端静态目录：优先用可执行文件同级目录（部署场景），
// 回退到当前工作目录（开发场景）。
func webDistPath() string {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "web", "dist")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "web/dist"
}
