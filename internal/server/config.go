package server

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Addr                string
	DBURL               string
	JWTSecret           string
	AgentToken          string
	AdminUser           string
	AdminPassword       string
	HeartbeatTimeoutSec int
}

// Load 读取配置，优先级从低到高：默认值 < server.conf 配置文件 < 环境变量。
func Load() Config {
	cfg := Config{
		Addr:                ":8080",
		DBURL:               "postgres://postgres:postgres@localhost:5432/opsplatform?sslmode=disable",
		JWTSecret:           "dev-insecure-secret-change-me",
		AgentToken:          "",
		AdminUser:           "admin",
		AdminPassword:       "admin123",
		HeartbeatTimeoutSec: 90,
	}

	// 配置文件（server.conf，与可执行文件同目录）
	if p := configFilePath(); p != "" {
		if f, err := os.Open(p); err == nil {
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if i := strings.IndexByte(line, '='); i > 0 {
					applyKey(&cfg, strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]))
				}
			}
			f.Close()
		}
	}

	// 环境变量覆盖（非空才覆盖）
	applyKey(&cfg, "OPS_ADDR", os.Getenv("OPS_ADDR"))
	applyKey(&cfg, "OPS_DB_URL", os.Getenv("OPS_DB_URL"))
	applyKey(&cfg, "OPS_JWT_SECRET", os.Getenv("OPS_JWT_SECRET"))
	applyKey(&cfg, "OPS_AGENT_TOKEN", os.Getenv("OPS_AGENT_TOKEN"))
	applyKey(&cfg, "OPS_ADMIN_USER", os.Getenv("OPS_ADMIN_USER"))
	applyKey(&cfg, "OPS_ADMIN_PASSWORD", os.Getenv("OPS_ADMIN_PASSWORD"))
	applyKey(&cfg, "OPS_HEARTBEAT_TIMEOUT_SEC", os.Getenv("OPS_HEARTBEAT_TIMEOUT_SEC"))

	return cfg
}

func configFilePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "server.conf")
}

// applyKey 把 key=value 应用到配置；value 为空则不覆盖。
func applyKey(cfg *Config, key, val string) {
	if val == "" {
		return
	}
	switch key {
	case "OPS_ADDR", "addr":
		cfg.Addr = val
	case "OPS_DB_URL", "db_url":
		cfg.DBURL = val
	case "OPS_JWT_SECRET", "jwt_secret":
		cfg.JWTSecret = val
	case "OPS_AGENT_TOKEN", "agent_token":
		cfg.AgentToken = val
	case "OPS_ADMIN_USER", "admin_user":
		cfg.AdminUser = val
	case "OPS_ADMIN_PASSWORD", "admin_password":
		cfg.AdminPassword = val
	case "OPS_HEARTBEAT_TIMEOUT_SEC", "heartbeat_timeout_sec":
		if n, err := strconv.Atoi(val); err == nil {
			cfg.HeartbeatTimeoutSec = n
		}
	}
}
