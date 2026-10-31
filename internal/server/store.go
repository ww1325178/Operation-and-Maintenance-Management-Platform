package server

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ops-platform/internal/protocol"
)

// upsertDevice records a full inventory report inside one transaction: it
// upserts the device row and replaces its network interfaces.
func (s *Server) upsertDevice(ctx context.Context, deviceID string, h protocol.Hello) error {
	if deviceID == "" || h.Hostname == "" {
		return errors.New("invalid hello payload")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO devices (device_id, hostname, os_name, os_version, os_build,
		                     agent_version, status, last_seen_at, first_seen_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,'online',now(),now(),now())
		ON CONFLICT (device_id) DO UPDATE SET
			hostname      = EXCLUDED.hostname,
			os_name       = EXCLUDED.os_name,
			os_version    = EXCLUDED.os_version,
			os_build      = EXCLUDED.os_build,
			agent_version = EXCLUDED.agent_version,
			status        = 'online',
			last_seen_at  = now(),
			updated_at    = now()`,
		deviceID, h.Hostname, h.OSName, h.OSVersion, h.OSBuild, h.AgentVersion)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM network_interfaces WHERE device_id=$1`, deviceID); err != nil {
		return err
	}
	for _, ifc := range h.Interfaces {
		ips, _ := json.Marshal(ifc.IPs)
		if _, err := tx.Exec(ctx,
			`INSERT INTO network_interfaces (device_id, name, mac, ips) VALUES ($1,$2,$3,$4)`,
			deviceID, ifc.Name, ifc.MAC, ips); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Server) touchDevice(ctx context.Context, deviceID string) {
	_, _ = s.pool.Exec(ctx,
		`UPDATE devices SET status='online', last_seen_at=now(), updated_at=now() WHERE device_id=$1`,
		deviceID)
}

func (s *Server) markOffline(ctx context.Context, deviceID string) {
	_, _ = s.pool.Exec(ctx,
		`UPDATE devices SET status='offline', updated_at=now() WHERE device_id=$1`, deviceID)
}

// OfflineLoop periodically marks online devices offline when their heartbeat
// has gone silent for longer than the configured timeout.
func (s *Server) OfflineLoop(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_, _ = s.pool.Exec(ctx, `
				UPDATE devices SET status='offline', updated_at=now()
				WHERE status='online' AND last_seen_at < now() - ($1::int * interval '1 second')`,
				s.cfg.HeartbeatTimeoutSec)
		}
	}
}
