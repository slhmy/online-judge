package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/online-judge/bff/internal/sse"
)

// SSEHandler handles SSE connections for real-time submission updates
type SSEHandler struct {
	hub *sse.Hub
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(hub *sse.Hub) *SSEHandler {
	return &SSEHandler{
		hub: hub,
	}
}

// Stream handles SSE connection for a specific submission
// Endpoint: GET /api/v1/submissions/:id/stream
func (h *SSEHandler) Stream(w http.ResponseWriter, r *http.Request) {
	submissionID := chi.URLParam(r, "id")
	if submissionID == "" {
		http.Error(w, "submission ID required", http.StatusBadRequest)
		return
	}

	// Create new SSE client for this submission
	client := sse.NewClient(submissionID)

	// Serve SSE stream
	client.ServeHTTP(w, r, h.hub)
}

// RegisterRoutes registers SSE routes
func (h *SSEHandler) RegisterRoutes(r chi.Router) {
	r.Get("/submissions/{id}/stream", h.Stream)
}