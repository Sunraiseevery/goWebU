PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;

CREATE TABLE IF NOT EXISTS hosts (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  name         TEXT,
  host         TEXT NOT NULL,
  port         INTEGER NOT NULL DEFAULT 22,
  username     TEXT NOT NULL,
  auth_type    TEXT NOT NULL CHECK (auth_type IN ('password','key')),
  key_alias    TEXT,
  password     TEXT,
  note         TEXT,
  created_at   TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at   TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(host, port, username)
);

CREATE TABLE IF NOT EXISTS forwards (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  host_id      INTEGER NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
  local_addr   TEXT NOT NULL DEFAULT '127.0.0.1',
  local_port   INTEGER NOT NULL,
  remote_host  TEXT NOT NULL,
  remote_port  INTEGER NOT NULL,
  auto_reconnect INTEGER NOT NULL DEFAULT 0,
  created_at   TEXT NOT NULL DEFAULT (datetime('now')),
  last_used_at TEXT,
  UNIQUE(host_id, local_addr, local_port, remote_host, remote_port)
);

CREATE TABLE IF NOT EXISTS sessions (
  id           TEXT PRIMARY KEY,
  host_id      INTEGER NOT NULL REFERENCES hosts(id),
  forward_id   INTEGER REFERENCES forwards(id),
  started_at   TEXT NOT NULL,
  stopped_at   TEXT,
  status       TEXT NOT NULL,
  last_error   TEXT
);

CREATE TABLE IF NOT EXISTS command_history (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id   TEXT REFERENCES sessions(id) ON DELETE SET NULL,
  host_id      INTEGER REFERENCES hosts(id),
  forward_id   INTEGER REFERENCES forwards(id),
  raw_command  TEXT NOT NULL,
  created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS events (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id   TEXT REFERENCES sessions(id) ON DELETE CASCADE,
  level        TEXT NOT NULL,
  message      TEXT NOT NULL,
  created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_history_created ON command_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_events_session_time ON events(session_id, created_at);
