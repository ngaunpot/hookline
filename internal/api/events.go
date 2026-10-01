package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Event is a webhook event as the system stores and delivers it.
type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// createEventRequest is what a client is allowed to send. It is kept
// separate from Event so callers can't set the ID or the timestamp.
type createEventRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (r createEventRequest) validate() error {
	if r.Type == "" {
		return errors.New("type is required")
	}
	if len(r.Payload) == 0 {
		return errors.New("payload is required")
	}
	return nil
}

func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	var req createEventRequest

	// MaxBytesReader caps the body at 1 MiB so a huge request can't
	// exhaust memory. DisallowUnknownFields catches client typos early.
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	event := Event{
		ID:        uuid.NewString(),
		Type:      req.Type,
		Payload:   req.Payload,
		CreatedAt: time.Now().UTC(),
	}

	// Sprint 2 persists this; sprint 3 hands it to the worker pool.
	// For now it is accepted and dropped.
	writeJSON(w, http.StatusAccepted, event)
}
