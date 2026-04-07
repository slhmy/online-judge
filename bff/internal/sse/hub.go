package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string          // event type (status, run, verdict, etc.)
	Data  json.RawMessage // JSON data payload
}

// Client represents an SSE client connection for a specific submission
type Client struct {
	submissionID string
	send         chan SSEEvent
	ctx          context.Context
	cancel       context.CancelFunc
}

// Hub manages SSE client connections and Redis pub/sub subscriptions
type Hub struct {
	// Clients grouped by submission ID
	clients map[string]map[*Client]bool // submissionID -> clients

	// Redis pub/sub subscriptions per submission
	subscriptions map[string]interface{} // submissionID -> *redis.PubSub (stored as interface{} to avoid type issues)

	mu    sync.RWMutex
	redis *redis.Client

	// Context for hub lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// NewHub creates a new SSE hub
func NewHub(redis *redis.Client) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:       make(map[string]map[*Client]bool),
		subscriptions: make(map[string]interface{}),
		redis:         redis,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// NewClient creates a new SSE client for a submission
func NewClient(submissionID string) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		submissionID: submissionID,
		send:         make(chan SSEEvent, 64),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// AddClient registers a client for a submission and starts Redis subscription if needed
func (h *Hub) AddClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add client to submission's client set
	if h.clients[client.submissionID] == nil {
		h.clients[client.submissionID] = make(map[*Client]bool)
	}
	h.clients[client.submissionID][client] = true

	// Start Redis subscription if this is the first client for this submission
	if len(h.clients[client.submissionID]) == 1 {
		h.startSubscription(client.submissionID)
	}

	log.Printf("SSE client connected for submission %s (total clients: %d)",
		client.submissionID, len(h.clients[client.submissionID]))
}

// RemoveClient unregisters a client and cleans up subscription if no clients remain
func (h *Hub) RemoveClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove client from submission's client set
	if clients, ok := h.clients[client.submissionID]; ok {
		delete(clients, client)
		client.cancel()

		// Clean up subscription if no clients remain
		if len(clients) == 0 {
			delete(h.clients, client.submissionID)
			h.stopSubscription(client.submissionID)
		}
	}

	log.Printf("SSE client disconnected for submission %s", client.submissionID)
}

// startSubscription starts Redis pub/sub subscription for a submission
func (h *Hub) startSubscription(submissionID string) {
	// Subscribe to both result and run channels for this submission
	resultChannel := fmt.Sprintf("judging:result:%s", submissionID)
	runChannel := fmt.Sprintf("judging:run:%s", submissionID)

	pubsub := h.redis.PSubscribe(h.ctx, resultChannel, runChannel)
	h.subscriptions[submissionID] = pubsub

	log.Printf("Started Redis subscription for submission %s", submissionID)

	// Start message handler goroutine
	go h.handleSubscription(submissionID, pubsub)
}

// stopSubscription stops Redis pub/sub subscription for a submission
func (h *Hub) stopSubscription(submissionID string) {
	if pubsub, ok := h.subscriptions[submissionID]; ok {
		if ps, ok := pubsub.(*redis.PubSub); ok {
			_ = ps.Close()
		}
		delete(h.subscriptions, submissionID)
		log.Printf("Stopped Redis subscription for submission %s", submissionID)
	}
}

// handleSubscription processes Redis pub/sub messages for a submission
func (h *Hub) handleSubscription(submissionID string, pubsub *redis.PubSub) {
	ch := pubsub.Channel()

	for {
		select {
		case <-h.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			h.processMessage(submissionID, msg)
		}
	}
}

// processMessage converts Redis message to SSE event and broadcasts to clients
func (h *Hub) processMessage(submissionID string, msg *redis.Message) {
	// Determine event type based on channel
	var eventType string
	if strings.HasPrefix(msg.Channel, "judging:result:") {
		// Parse the payload to determine if it's a final verdict or progress update
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Payload), &payload); err == nil {
			status, _ := payload["status"].(string)
			if status == "completed" {
				eventType = "verdict"
			} else {
				eventType = "status"
			}
		} else {
			eventType = "status"
		}
	} else if strings.HasPrefix(msg.Channel, "judging:run:") {
		eventType = "run"
	} else {
		eventType = "update"
	}

	event := SSEEvent{
		Event: eventType,
		Data:  json.RawMessage(msg.Payload),
	}

	// Broadcast to all clients for this submission
	h.broadcastToSubmission(submissionID, event)
}

// broadcastToSubmission sends an SSE event to all clients for a submission
func (h *Hub) broadcastToSubmission(submissionID string, event SSEEvent) {
	h.mu.RLock()
	clients := h.clients[submissionID]
	h.mu.RUnlock()

	if clients == nil {
		return
	}

	for client := range clients {
		select {
		case client.send <- event:
		default:
			// Client channel full, skip (client might be slow)
			log.Printf("SSE client channel full for submission %s", submissionID)
		}
	}
}

// Stop stops the hub and cleans up all subscriptions
func (h *Hub) Stop() {
	h.cancel()

	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all subscriptions
	for submissionID, pubsub := range h.subscriptions {
		if ps, ok := pubsub.(*redis.PubSub); ok {
			_ = ps.Close()
		}
		delete(h.subscriptions, submissionID)
	}

	// Cancel all client contexts
	for submissionID, clients := range h.clients {
		for client := range clients {
			client.cancel()
		}
		delete(h.clients, submissionID)
	}

	log.Printf("SSE Hub stopped")
}

// ServeHTTP handles the SSE connection for a client
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request, hub *Hub) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Check if response writer supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Register client with hub
	hub.AddClient(c)
	defer hub.RemoveClient(c)

	// Send initial connection message
	_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"submission_id\":\"%s\"}\n\n", c.submissionID)
	flusher.Flush()

	// Keep connection alive and send events
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Client disconnected or hub stopped
			return

		case <-r.Context().Done():
			// HTTP request context done (client disconnected)
			return

		case <-ticker.C:
			// Send keepalive comment to prevent connection timeout
			_, _ = fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case event := <-c.send:
			// Send SSE event
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, event.Data)
			flusher.Flush()
		}
	}
}