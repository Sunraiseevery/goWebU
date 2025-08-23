package app

import (
	"context"
	"database/sql"
	"time"
)

// Host represents a host record.
type Host struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
	AuthType  string `json:"auth_type"`
	KeyAlias  string `json:"key_alias"`
	Password  string `json:"password"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type HostsRepo struct{ DB *sql.DB }

func (r *HostsRepo) Upsert(ctx context.Context, h *Host) (int64, error) {
	res, err := r.DB.ExecContext(ctx, `
        INSERT INTO hosts(name,host,port,username,auth_type,key_alias,password,note)
        VALUES(?,?,?,?,?,?,?,?)
        ON CONFLICT(host,port,username) DO UPDATE SET
          name=excluded.name,
          auth_type=excluded.auth_type,
          key_alias=excluded.key_alias,
          password=excluded.password,
          note=excluded.note,
          updated_at=datetime('now')
    `, h.Name, h.Host, h.Port, h.Username, h.AuthType, h.KeyAlias, h.Password, h.Note)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		err = r.DB.QueryRowContext(ctx, `SELECT id FROM hosts WHERE host=? AND port=? AND username=?`, h.Host, h.Port, h.Username).Scan(&id)
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (r *HostsRepo) List(ctx context.Context) ([]Host, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id,name,host,port,username,auth_type,key_alias,password,note,created_at,updated_at FROM hosts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []Host
	for rows.Next() {
		var h Host
		if err := rows.Scan(&h.ID, &h.Name, &h.Host, &h.Port, &h.Username, &h.AuthType, &h.KeyAlias, &h.Password, &h.Note, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, h)
	}
	return res, rows.Err()
}

func (r *HostsRepo) Get(ctx context.Context, id int64) (*Host, error) {
	var h Host
	err := r.DB.QueryRowContext(ctx, `SELECT id,name,host,port,username,auth_type,key_alias,password,note,created_at,updated_at FROM hosts WHERE id=?`, id).
		Scan(&h.ID, &h.Name, &h.Host, &h.Port, &h.Username, &h.AuthType, &h.KeyAlias, &h.Password, &h.Note, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// CommandHistory represents a command record.
type CommandHistory struct {
	ID        int64         `json:"id"`
	SessionID string        `json:"session_id"`
	HostID    sql.NullInt64 `json:"host_id"`
	ForwardID sql.NullInt64 `json:"forward_id"`
	Raw       string        `json:"raw_command"`
	CreatedAt string        `json:"created_at"`
}

type HistoryRepo struct{ DB *sql.DB }

func (r *HistoryRepo) Add(ctx context.Context, h *CommandHistory) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO command_history(session_id,host_id,forward_id,raw_command) VALUES(?,?,?,?)`, h.SessionID, h.HostID, h.ForwardID, h.Raw)
	return err
}

func (r *HistoryRepo) List(ctx context.Context, limit int) ([]CommandHistory, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id,session_id,host_id,forward_id,raw_command,created_at FROM command_history ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []CommandHistory
	for rows.Next() {
		var h CommandHistory
		if err := rows.Scan(&h.ID, &h.SessionID, &h.HostID, &h.ForwardID, &h.Raw, &h.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, h)
	}
	return res, rows.Err()
}

// Session represents a running session.
type Session struct {
	ID        string         `json:"id"`
	HostID    int64          `json:"host_id"`
	StartedAt time.Time      `json:"started_at"`
	StoppedAt sql.NullTime   `json:"stopped_at"`
	Status    string         `json:"status"`
	LastError sql.NullString `json:"last_error"`
}

type SessionsRepo struct{ DB *sql.DB }

func (r *SessionsRepo) Start(ctx context.Context, id string, hostID int64) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO sessions(id,host_id,started_at,status) VALUES(?,?,datetime('now'),'running')`, id, hostID)
	return err
}

func (r *SessionsRepo) Stop(ctx context.Context, id string, status string, errMsg *string) error {
	if errMsg != nil {
		_, err := r.DB.ExecContext(ctx, `UPDATE sessions SET status=?, last_error=?, stopped_at=datetime('now') WHERE id=?`, status, *errMsg, id)
		return err
	}
	_, err := r.DB.ExecContext(ctx, `UPDATE sessions SET status=?, stopped_at=datetime('now') WHERE id=?`, status, id)
	return err
}

// Events repo.
type Event struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type EventsRepo struct{ DB *sql.DB }

func (r *EventsRepo) Add(ctx context.Context, sessionID, level, msg string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO events(session_id,level,message) VALUES(?,?,?)`, sessionID, level, msg)
	return err
}
