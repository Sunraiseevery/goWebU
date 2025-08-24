package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	Hosts    *HostsRepo
	History  *HistoryRepo
	Sessions *SessionsRepo
	Events   *EventsRepo
	TunMgr   *TunnelManager
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	msg := err.Error()
	if errors.Is(err, sql.ErrNoRows) {
		status = http.StatusNotFound
		msg = "no matching record found in the database (sql: no rows in result set)"
	}
	w.WriteHeader(status)
	writeJSON(w, map[string]any{"error": msg})
}

func (s *Server) hostsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		list, err := s.Hosts.List(ctx)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	case http.MethodPost:
		var h Host
		if err := json.NewDecoder(r.Body).Decode(&h); err != nil {
			writeErr(w, err)
			return
		}
		id, err := s.Hosts.Upsert(ctx, &h)
		if err != nil {
			writeErr(w, err)
			return
		}
		h.ID = id
		writeJSON(w, h)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// StartReq defines request for starting a tunnel.
type StartReq struct {
	HostID int64  `json:"host_id"`
	LPort  int    `json:"lport"`
	RHost  string `json:"rhost"`
	RPort  int    `json:"rport"`
}

func (s *Server) buildAuth(authType, password, keyAlias string) (ssh.AuthMethod, error) {
	switch authType {
	case "password":
		return ssh.Password(password), nil
	case "key":
		// simplified: load private key from file path in keyAlias
		key, err := ssh.ParsePrivateKey([]byte(keyAlias)) // expecting key content
		if err != nil {
			return nil, err
		}
		return ssh.PublicKeys(key), nil
	default:
		return nil, fmt.Errorf("unknown auth type")
	}
}

func BuildSSHCommandLine(lport int, rhost string, rport int, user, host string) string {
	return fmt.Sprintf("ssh -L %d:%s:%d %s@%s", lport, rhost, rport, user, host)
}

func (s *Server) startHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req StartReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	h, err := s.Hosts.Get(ctx, req.HostID)
	if err != nil {
		writeErr(w, err)
		return
	}
	auth, err := s.buildAuth(h.AuthType, h.Password, h.KeyAlias)
	if err != nil {
		writeErr(w, err)
		return
	}
	sessID := uuid.NewString()
	if err := s.Sessions.Start(ctx, sessID, h.ID); err != nil {
		writeErr(w, err)
		return
	}
	err = s.TunMgr.Connect(sessID, h.Host, h.Port, h.Username, auth, func(level, msg string) {
		_ = s.Events.Add(ctx, sessID, level, msg)
	})
	if err != nil {
		msg := err.Error()
		_ = s.Sessions.Stop(ctx, sessID, "error", &msg)
		writeErr(w, err)
		return
	}
	err = s.TunMgr.Forward(sessID, req.LPort, req.RHost, req.RPort)
	if err != nil {
		s.TunMgr.Stop(sessID)
		msg := err.Error()
		_ = s.Sessions.Stop(ctx, sessID, "error", &msg)
		writeErr(w, err)
		return
	}
	raw := BuildSSHCommandLine(req.LPort, req.RHost, req.RPort, h.Username, h.Host)
	_ = s.History.Add(ctx, &CommandHistory{SessionID: sessID, HostID: sql.NullInt64{Int64: h.ID, Valid: true}, Raw: raw})
	writeJSON(w, map[string]any{"session_id": sessID})
}

func (s *Server) stopHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	s.TunMgr.Stop(req.SessionID)
	_ = s.Sessions.Stop(ctx, req.SessionID, "closed", nil)
	writeJSON(w, map[string]any{"stopped": true})
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	alive := s.TunMgr.Alive(req.SessionID)
	if !alive {
		_ = s.Sessions.Stop(ctx, req.SessionID, "closed", nil)
	}
	writeJSON(w, map[string]any{"alive": alive})
}

func (s *Server) historyHandler(w http.ResponseWriter, r *http.Request) {
	list, err := s.History.List(r.Context(), 50)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, list)
}

func NewServer(db *sql.DB) *Server {
	return &Server{
		Hosts:    &HostsRepo{DB: db},
		History:  &HistoryRepo{DB: db},
		Sessions: &SessionsRepo{DB: db},
		Events:   &EventsRepo{DB: db},
		TunMgr:   NewTunnelManager(),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/hosts", s.hostsHandler)
	mux.HandleFunc("/start", s.startHandler)
	mux.HandleFunc("/stop", s.stopHandler)
	mux.HandleFunc("/status", s.statusHandler)
	mux.HandleFunc("/history", s.historyHandler)
	mux.Handle("/", http.FileServer(http.Dir("ui")))
	return mux
}
