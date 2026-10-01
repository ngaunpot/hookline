package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleCreateEvent(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid event", `{"type":"user.created","payload":{"id":42}}`, http.StatusAccepted},
		{"malformed JSON", `{"type":`, http.StatusBadRequest},
		{"unknown field", `{"type":"a","payload":{},"extra":1}`, http.StatusBadRequest},
		{"missing type", `{"payload":{"id":42}}`, http.StatusUnprocessableEntity},
		{"missing payload", `{"type":"user.created"}`, http.StatusUnprocessableEntity},
	}

	srv := NewServer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			srv.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
		})
	}
}

func TestHandleCreateEventAssignsID(t *testing.T) {
	srv := NewServer()

	body := strings.NewReader(`{"type":"user.created","payload":{"id":42}}`)
	req := httptest.NewRequest(http.MethodPost, "/events", body)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	var got Event
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" {
		t.Error("expected an event ID, got empty string")
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected a creation timestamp, got zero value")
	}
}
