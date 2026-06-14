package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/gorilla/mux"

	"wakupi/internal/wa"
)

// WAService is the subset of wa.Manager the API needs. Declaring it as an
// interface keeps the api package testable and decoupled.
type WAService interface {
	Sessions() []wa.SessionInfo
	StartLogin(ctx context.Context, sessionName string) (string, error)
	Logout(ctx context.Context, sessionID string) error
	LoadChats(ctx context.Context, accountID string) ([]wa.ChatInfo, error)
	LoadMessages(ctx context.Context, accountID, jid string, limit int, beforeTS int64) ([]wa.MessageInfo, error)
	SendText(ctx context.Context, sessionID, jidStr, text string, quoted *wa.QuotedRef) (string, error)
	SendImage(ctx context.Context, sessionID, jidStr, filePath, caption string, quoted *wa.QuotedRef) (*wa.SendMediaResult, error)
	SendDocument(ctx context.Context, sessionID, jidStr, filePath string, quoted *wa.QuotedRef) (*wa.SendMediaResult, error)
	MarkRead(ctx context.Context, sessionID, jidStr, senderStr string, messageIDs []string) error
	ReactMessage(ctx context.Context, sessionID, jidStr, messageID, sender, emoji string) error
}

// Server is the embedded REST + WebSocket API server.
type Server struct {
	cfg    Config
	wa     WAService
	hub    *Hub
	router *mux.Router
	http   *http.Server
}

// New builds a Server. Call Start to begin listening.
func New(cfg Config, svc WAService, hub *Hub) *Server {
	s := &Server{
		cfg: cfg,
		wa:  svc,
		hub: hub,
	}
	s.router = s.buildRouter()
	s.http = &http.Server{
		Addr:              cfg.Addr,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s
}

// Start begins serving in a background goroutine and returns immediately.
// Errors after startup are delivered on the returned channel.
func (s *Server) Start() <-chan error {
	errc := make(chan error, 1)
	go func() {
		var err error
		if s.cfg.TLSCert != "" && s.cfg.TLSKey != "" {
			err = s.http.ListenAndServeTLS(s.cfg.TLSCert, s.cfg.TLSKey)
		} else {
			err = s.http.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
		close(errc)
	}()
	return errc
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) buildRouter() *mux.Router {
	r := mux.NewRouter()
	v1 := r.PathPrefix("/v1").Subrouter()

	// Health is unauthenticated so liveness probes work without a token.
	v1.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)

	// Everything else requires a bearer token.
	auth := v1.NewRoute().Subrouter()
	auth.Use(s.authMiddleware)

	auth.HandleFunc("/sessions", s.handleListSessions).Methods(http.MethodGet)
	auth.HandleFunc("/sessions", s.handleCreateSession).Methods(http.MethodPost)
	auth.HandleFunc("/sessions/{id}", s.handleDeleteSession).Methods(http.MethodDelete)

	auth.HandleFunc("/chats", s.handleListChats).Methods(http.MethodGet)
	auth.HandleFunc("/chats/{jid}/messages", s.handleListMessages).Methods(http.MethodGet)
	auth.HandleFunc("/chats/{jid}/messages", s.handleSendMessage).Methods(http.MethodPost)
	auth.HandleFunc("/chats/{jid}/read", s.handleMarkRead).Methods(http.MethodPost)
	auth.HandleFunc("/chats/{jid}/react", s.handleReact).Methods(http.MethodPost)

	// WebSocket events. Auth is handled inline (browsers can't set headers on
	// WS, so a ?token= query param is also accepted).
	v1.HandleFunc("/events", s.handleEvents).Methods(http.MethodGet)

	return r
}

// authMiddleware enforces a constant-time bearer token check.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.checkToken(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) checkToken(r *http.Request) bool {
	want := s.cfg.Token
	if want == "" {
		return false
	}
	got := ""
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		got = strings.TrimPrefix(h, "Bearer ")
	} else if q := r.URL.Query().Get("token"); q != "" {
		got = q
	}
	if got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// ----- handlers -----------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]interface{}{
		"ok":      true,
		"version": Version,
		"clients": s.hub.Count(),
	})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	writeOK(w, s.wa.Sessions())
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	id, err := s.wa.StartLogin(r.Context(), body.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login_failed", err.Error())
		return
	}
	writeOK(w, map[string]interface{}{
		"session_id": id,
		"note":       "subscribe to /v1/events for wa:qr and wa:login_success",
	})
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := s.wa.Logout(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, "logout_failed", err.Error())
		return
	}
	writeOK(w, map[string]interface{}{"logged_out": id})
}

func (s *Server) handleListChats(w http.ResponseWriter, r *http.Request) {
	account := r.URL.Query().Get("session")
	if account == "" {
		account = s.firstSession()
	}
	chats, err := s.wa.LoadChats(r.Context(), account)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load_chats_failed", err.Error())
		return
	}
	writeOK(w, chats)
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	jid := mux.Vars(r)["jid"]
	account := r.URL.Query().Get("session")
	if account == "" {
		account = s.firstSession()
	}
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	var before int64
	if b := r.URL.Query().Get("before"); b != "" {
		if n, err := strconv.ParseInt(b, 10, 64); err == nil {
			before = n
		}
	}
	msgs, err := s.wa.LoadMessages(r.Context(), account, jid, limit, before)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load_messages_failed", err.Error())
		return
	}
	writeOK(w, msgs)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	jid := mux.Vars(r)["jid"]
	var body struct {
		Session   string `json:"session"`
		Text      string `json:"text"`
		MediaPath string `json:"media_path"`
		Caption   string `json:"caption"`
		ReplyTo   string `json:"reply_to"`
		ReplyText string `json:"reply_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	account := body.Session
	if account == "" {
		account = s.firstSession()
	}
	var quoted *wa.QuotedRef
	if body.ReplyTo != "" {
		quoted = &wa.QuotedRef{ID: body.ReplyTo, Text: body.ReplyText}
	}

	// Media path takes precedence; route by mime sniff on extension.
	if body.MediaPath != "" {
		if isImagePath(body.MediaPath) {
			res, err := s.wa.SendImage(r.Context(), account, jid, body.MediaPath, body.Caption, quoted)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "send_image_failed", err.Error())
				return
			}
			writeOK(w, res)
			return
		}
		res, err := s.wa.SendDocument(r.Context(), account, jid, body.MediaPath, quoted)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "send_doc_failed", err.Error())
			return
		}
		writeOK(w, res)
		return
	}

	if body.Text == "" {
		writeError(w, http.StatusBadRequest, "empty_message", "text or media_path required")
		return
	}
	id, err := s.wa.SendText(r.Context(), account, jid, body.Text, quoted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "send_failed", err.Error())
		return
	}
	writeOK(w, map[string]interface{}{"message_id": id})
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	jid := mux.Vars(r)["jid"]
	var body struct {
		Session    string   `json:"session"`
		Sender     string   `json:"sender"`
		MessageIDs []string `json:"message_ids"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	account := body.Session
	if account == "" {
		account = s.firstSession()
	}
	if err := s.wa.MarkRead(r.Context(), account, jid, body.Sender, body.MessageIDs); err != nil {
		writeError(w, http.StatusInternalServerError, "mark_read_failed", err.Error())
		return
	}
	writeOK(w, map[string]interface{}{"read": true})
}

func (s *Server) handleReact(w http.ResponseWriter, r *http.Request) {
	jid := mux.Vars(r)["jid"]
	var body struct {
		Session string `json:"session"`
		MsgID   string `json:"msg_id"`
		Sender  string `json:"sender"`
		Emoji   string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	account := body.Session
	if account == "" {
		account = s.firstSession()
	}
	if err := s.wa.ReactMessage(r.Context(), account, jid, body.MsgID, body.Sender, body.Emoji); err != nil {
		writeError(w, http.StatusInternalServerError, "react_failed", err.Error())
		return
	}
	writeOK(w, map[string]interface{}{"reacted": true})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if !s.checkToken(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid bearer token")
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Wakupi is loopback-only by default; allow any origin so the Wails
		// webview and local tools can connect.
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	s.hub.Serve(r.Context(), conn)
}

// ----- helpers ------------------------------------------------------------

func (s *Server) firstSession() string {
	sessions := s.wa.Sessions()
	if len(sessions) > 0 {
		return sessions[0].ID
	}
	return ""
}

func isImagePath(p string) bool {
	switch strings.ToLower(p[strings.LastIndex(p, ".")+1:]) {
	case "jpg", "jpeg", "png", "webp", "gif":
		return true
	}
	return false
}

func writeOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "data": data})
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": false,
		"error": map[string]string{
			"code":    code,
			"message": msg,
		},
	})
}
