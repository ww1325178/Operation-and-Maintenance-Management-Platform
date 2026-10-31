-- 0001_init.sql — core schema for the ops platform.

CREATE TABLE IF NOT EXISTS devices (
    id             BIGSERIAL PRIMARY KEY,
    device_id      TEXT NOT NULL UNIQUE,
    hostname       TEXT NOT NULL,
    os_name        TEXT NOT NULL DEFAULT '',
    os_version     TEXT NOT NULL DEFAULT '',
    os_build       TEXT NOT NULL DEFAULT '',
    agent_version  TEXT NOT NULL DEFAULT '',
    logged_in_user TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'offline', -- online | offline
    last_seen_at   TIMESTAMPTZ,
    first_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_devices_hostname ON devices (hostname);
CREATE INDEX IF NOT EXISTS idx_devices_status   ON devices (status);

CREATE TABLE IF NOT EXISTS network_interfaces (
    id         BIGSERIAL PRIMARY KEY,
    device_id  TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    mac        TEXT NOT NULL,
    ips        JSONB NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ni_device ON network_interfaces (device_id);

-- Heartbeat history. Populated only on connect/disconnect events for now;
-- Phase 5 adds time-based partitioning and retention for high-frequency rows.
CREATE TABLE IF NOT EXISTS heartbeats (
    device_id TEXT NOT NULL,
    ts        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_hb_device_ts ON heartbeats (device_id, ts DESC);

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'admin',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_log (
    id        BIGSERIAL PRIMARY KEY,
    user_id   BIGINT,
    device_id TEXT,
    action    TEXT NOT NULL,
    detail    JSONB,
    ts        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log (ts DESC);

CREATE TABLE IF NOT EXISTS remote_sessions (
    id         BIGSERIAL PRIMARY KEY,
    session_id TEXT NOT NULL UNIQUE,
    device_id  TEXT NOT NULL,
    user_id    BIGINT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at   TIMESTAMPTZ
);
