package websocket

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// Hub maintains the set of active clients and manages subscriptions
type Hub struct {
	// Clients by user ID
	clients map[string]map[*Client]bool // userID -> clients

	// Channel subscriptions: clients can subscribe to specific channels
	// e.g., "submission:{id}", "contest:{id}"
	channelSubs map[string]map[*Client]bool // channel -> clients

	// Reverse lookup: which channels a client is subscribed to
	clientChannels map[*Client]map[string]bool // client -> channels

	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client

	// Channel subscription management
	subscribeChannel   chan *ChannelSubscription
	unsubscribeChannel chan *ChannelSubscription

	mu    sync.RWMutex
	redis *redis.Client

	// Redis pub/sub context and cancel function
	ctx    context.Context
	cancel context.CancelFunc
}

// Message represents a WebSocket message
type Message struct {
	UserID  string          `json:"user_id,omitempty"`
	Channel string          `json:"channel,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// ChannelSubscription represents a channel subscription request
type ChannelSubscription struct {
	Client  *Client
	Channel string
}

// Client represents a WebSocket client connection
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string

	// Channels this client is subscribed to
	channels map[string]bool
	mu       sync.RWMutex
}

// ClientMessage represents a message received from a client
type ClientMessage struct {
	Action  string `json:"action"`  // "subscribe" or "unsubscribe"
	Channel string `json:"channel"` // e.g., "submission:123", "contest:456"
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		channels: make(map[string]bool),
	}
}

// NewHub creates a new WebSocket hub
func NewHub(redis *redis.Client) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		clients:           make(map[string]map[*Client]bool),
		channelSubs:       make(map[string]map[*Client]bool),
		clientChannels:    make(map[*Client]map[string]bool),
		broadcast:         make(chan *Message, 256),
		register:          make(chan *Client),
		unregister:        make(chan *Client),
		subscribeChannel:  make(chan *ChannelSubscription),
		unsubscribeChannel: make(chan *ChannelSubscription),
		redis:             redis,
		ctx:               ctx,
		cancel:            cancel,
	}

	// Start Redis pub/sub listener
	go h.subscribeToRedis()

	return h
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case sub := <-h.subscribeChannel:
			h.handleChannelSubscribe(sub)

		case unsub := <-h.unsubscribeChannel:
			h.handleChannelUnsubscribe(unsub)

		case message := <-h.broadcast:
			h.handleBroadcast(message)
		}
	}
}

// handleRegister handles client registration
func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	if h.clients[client.userID] == nil {
		h.clients[client.userID] = make(map[*Client]bool)
	}
	h.clients[client.userID][client] = true
	h.clientChannels[client] = make(map[string]bool)
	h.mu.Unlock()
	log.Printf("Client connected: user=%s", client.userID)
}

// handleUnregister handles client unregistration
func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()

	// Remove from user clients
	if clients, ok := h.clients[client.userID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)
			if len(clients) == 0 {
				delete(h.clients, client.userID)
			}
		}
	}

	// Remove from all channel subscriptions
	if channels, ok := h.clientChannels[client]; ok {
		for channel := range channels {
			if subs, ok := h.channelSubs[channel]; ok {
				delete(subs, client)
				if len(subs) == 0 {
					delete(h.channelSubs, channel)
				}
			}
		}
		delete(h.clientChannels, client)
	}

	h.mu.Unlock()
	log.Printf("Client disconnected: user=%s", client.userID)
}

// handleChannelSubscribe handles channel subscription
func (h *Hub) handleChannelSubscribe(sub *ChannelSubscription) {
	h.mu.Lock()

	// Add to channel subscribers
	if h.channelSubs[sub.Channel] == nil {
		h.channelSubs[sub.Channel] = make(map[*Client]bool)
	}
	h.channelSubs[sub.Channel][sub.Client] = true

	// Add to client's channels
	if h.clientChannels[sub.Client] == nil {
		h.clientChannels[sub.Client] = make(map[string]bool)
	}
	h.clientChannels[sub.Client][sub.Channel] = true

	// Also track in client's local map
	sub.Client.mu.Lock()
	sub.Client.channels[sub.Channel] = true
	sub.Client.mu.Unlock()

	h.mu.Unlock()
	log.Printf("Client subscribed to channel: user=%s, channel=%s", sub.Client.userID, sub.Channel)
}

// handleChannelUnsubscribe handles channel unsubscription
func (h *Hub) handleChannelUnsubscribe(unsub *ChannelSubscription) {
	h.mu.Lock()

	// Remove from channel subscribers
	if subs, ok := h.channelSubs[unsub.Channel]; ok {
		delete(subs, unsub.Client)
		if len(subs) == 0 {
			delete(h.channelSubs, unsub.Channel)
		}
	}

	// Remove from client's channels
	if channels, ok := h.clientChannels[unsub.Client]; ok {
		delete(channels, unsub.Channel)
	}

	// Also remove from client's local map
	unsub.Client.mu.Lock()
	delete(unsub.Client.channels, unsub.Channel)
	unsub.Client.mu.Unlock()

	h.mu.Unlock()
	log.Printf("Client unsubscribed from channel: user=%s, channel=%s", unsub.Client.userID, unsub.Channel)
}

// handleBroadcast handles broadcasting messages
func (h *Hub) handleBroadcast(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(map[string]interface{}{
		"type":    message.Type,
		"channel": message.Channel,
		"payload": message.Payload,
	})
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	// If message has a specific channel, send to channel subscribers
	if message.Channel != "" {
		if subs, ok := h.channelSubs[message.Channel]; ok {
			for client := range subs {
				select {
				case client.send <- data:
				default:
					// Channel full, client might be slow
				}
			}
		}
		return
	}

	// If message has a specific user ID, send to that user's clients
	if message.UserID != "" {
		if clients, ok := h.clients[message.UserID]; ok {
			for client := range clients {
				select {
				case client.send <- data:
				default:
					// Channel full, close connection
					close(client.send)
					h.mu.Lock()
					delete(h.clients[message.UserID], client)
					h.mu.Unlock()
				}
			}
		}
		return
	}

	// Broadcast to all clients
	for userID, clients := range h.clients {
		for client := range clients {
			select {
			case client.send <- data:
			default:
				close(client.send)
				h.mu.Lock()
				delete(h.clients[userID], client)
				h.mu.Unlock()
			}
		}
	}
}

// subscribeToRedis subscribes to Redis pub/sub channels
func (h *Hub) subscribeToRedis() {
	pubsub := h.redis.PSubscribe(h.ctx, "judging:result:*", "judging:run:*", "contest:scoreboard:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-h.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			h.handleRedisMessage(msg)
		}
	}
}

// handleRedisMessage handles messages from Redis pub/sub
func (h *Hub) handleRedisMessage(msg *redis.Message) {
	var payload json.RawMessage = json.RawMessage(msg.Payload)

	// Determine message type based on channel pattern
	msgType := "update"
	if strings.HasPrefix(msg.Channel, "judging:result:") {
		msgType = "judging_result"
	} else if strings.HasPrefix(msg.Channel, "judging:run:") {
		msgType = "judging_run"
	} else if strings.HasPrefix(msg.Channel, "contest:scoreboard:") {
		msgType = "scoreboard_update"
	}

	h.broadcast <- &Message{
		Channel: msg.Channel,
		Type:    msgType,
		Payload: payload,
	}
}

// RegisterClient registers a new client
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// SubscribeToChannel subscribes a client to a channel
func (h *Hub) SubscribeToChannel(client *Client, channel string) {
	h.subscribeChannel <- &ChannelSubscription{
		Client:  client,
		Channel: channel,
	}
}

// UnsubscribeFromChannel unsubscribes a client from a channel
func (h *Hub) UnsubscribeFromChannel(client *Client, channel string) {
	h.unsubscribeChannel <- &ChannelSubscription{
		Client:  client,
		Channel: channel,
	}
}

// BroadcastToUser sends a message to a specific user
func (h *Hub) BroadcastToUser(userID string, msgType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling payload: %v", err)
		return
	}

	h.broadcast <- &Message{
		UserID:  userID,
		Type:    msgType,
		Payload: data,
	}
}

// BroadcastToChannel sends a message to a specific channel
func (h *Hub) BroadcastToChannel(channel string, msgType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling payload: %v", err)
		return
	}

	h.broadcast <- &Message{
		Channel: channel,
		Type:    msgType,
		Payload: data,
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.hub.UnregisterClient(c)
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Parse the client message
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			log.Printf("Error parsing client message: %v", err)
			continue
		}

		// Handle subscription actions
		switch clientMsg.Action {
		case "subscribe":
			if clientMsg.Channel != "" {
				c.hub.SubscribeToChannel(c, clientMsg.Channel)
				// Send confirmation
				c.send <- []byte(`{"type":"subscribed","channel":"` + clientMsg.Channel + `"}`)
			}
		case "unsubscribe":
			if clientMsg.Channel != "" {
				c.hub.UnsubscribeFromChannel(c, clientMsg.Channel)
				// Send confirmation
				c.send <- []byte(`{"type":"unsubscribed","channel":"` + clientMsg.Channel + `"}`)
			}
		default:
			log.Printf("Unknown action: %s", clientMsg.Action)
		}
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	defer c.conn.Close()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
}

// Stop stops the hub and cleans up resources
func (h *Hub) Stop() {
	h.cancel()
}