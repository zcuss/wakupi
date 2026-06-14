package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"wakupi/internal/gateway"
)

// registerGatewayRoutes mounts /v1/gateway/* endpoints. They manage the
// webhook subscriptions persisted by the gateway.Dispatcher. Each request
// goes through the same auth middleware as the rest of /v1/*.
func (s *Server) registerGatewayRoutes(r *mux.Router) {
	if s.gw == nil {
		return // gateway disabled — don't mount the routes at all
	}
	g := r.PathPrefix("/gateway").Subrouter()
	g.HandleFunc("/webhooks", s.handleListWebhooks).Methods(http.MethodGet)
	g.HandleFunc("/webhooks", s.handleCreateWebhook).Methods(http.MethodPost)
	g.HandleFunc("/webhooks/{id}", s.handleGetWebhook).Methods(http.MethodGet)
	g.HandleFunc("/webhooks/{id}", s.handleUpdateWebhook).Methods(http.MethodPut, http.MethodPatch)
	g.HandleFunc("/webhooks/{id}", s.handleDeleteWebhook).Methods(http.MethodDelete)
	g.HandleFunc("/webhooks/{id}/status", s.handleWebhookStatus).Methods(http.MethodGet)
	g.HandleFunc("/webhooks/{id}/test", s.handleTestWebhook).Methods(http.MethodPost)
	g.HandleFunc("/events", s.handleListEvents).Methods(http.MethodGet)
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	writeOK(w, s.gw.List())
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var hook gateway.Hook
	if err := json.NewDecoder(r.Body).Decode(&hook); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	created, err := s.gw.Add(hook)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "data": created})
}

func (s *Server) handleGetWebhook(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	hook, ok := s.gw.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "hook not found")
		return
	}
	writeOK(w, hook)
}

func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var patch gateway.Hook
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	updated, err := s.gw.Update(id, patch)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	writeOK(w, updated)
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := s.gw.Remove(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	writeOK(w, map[string]string{"id": id})
}

func (s *Server) handleWebhookStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if _, ok := s.gw.Get(id); !ok {
		writeError(w, http.StatusNotFound, "not_found", "hook not found")
		return
	}
	writeOK(w, s.gw.Status(id))
}

// handleTestWebhook fires a synthetic test event at the hook so users can
// verify URL/headers/retry without waiting for a real WhatsApp message.
func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	hook, ok := s.gw.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "hook not found")
		return
	}
	go s.gw.Dispatch(hook.Event, map[string]any{
		"_test":   true,
		"message": "this is a test delivery from Wakupi",
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true,
		"data": map[string]string{
			"status": "dispatched",
			"event":  hook.Event,
		},
	})
}

// handleListEvents returns the static list of event names Wakupi emits.
// Useful for the UI's "event picker" dropdown.
func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	events := []string{}
	for _, e := range strings.Split(knownEvents, ",") {
		events = append(events, strings.TrimSpace(e))
	}
	writeOK(w, events)
}

const knownEvents = "wa:message, wa:chat, wa:receipt, wa:reaction, wa:status, wa:deleted, wa:avatar, wa:presence, wa:chat_presence, wa:connected, wa:disconnected, wa:logged_out, wa:pair_success, wa:sync_complete, wa:qr, wa:qr_event, wa:qr_timeout, wa:login_success, wa:error"
