// internal/websocket/hub.go
package websocket

import (
	"context"
	"log"
	"sync"

	wstypes "zentora-service/internal/domain/websocket"
	"zentora-service/internal/pkg/jwt"
	"zentora-service/internal/pkg/session"
)

type Hub struct {
	// Registered clients by identity ID
	clients map[int64]map[*Client]bool
	mu      sync.RWMutex

	// Registration/unregistration
	Register   chan *Client
	unregister chan *Client

	// Broadcasting
	broadcast chan *BroadcastMessage

	// Handler registry for modular message handling
	handlerRegistry *HandlerRegistry

	// Auth dependencies
	jwtVerifier    *jwt.Verifier
	sessionManager *session.Manager
}

type BroadcastMessage struct {
	IdentityIDs []int64
	Channel     wstypes.ChannelType
	Message     *wstypes.WSMessage
}

func NewHub(jwtVerifier *jwt.Verifier, sessionManager *session.Manager) *Hub {
	return &Hub{
		clients:         make(map[int64]map[*Client]bool),
		Register:        make(chan *Client),
		unregister:      make(chan *Client),
		broadcast:       make(chan *BroadcastMessage, 256),
		handlerRegistry: NewHandlerRegistry(),
		jwtVerifier:     jwtVerifier,
		sessionManager:  sessionManager,
	}
}

// AuthenticateClient validates the JWT token and creates an authenticated client
func (h *Hub) AuthenticateClient(ctx context.Context, token string) (*ClientAuth, error) {
	// Verify JWT token
	claims, err := h.jwtVerifier.VerifyAccessToken(token)
	if err != nil {
		return nil, err
	}

	// Check if token is blacklisted
	blacklisted, err := h.sessionManager.IsTokenBlacklisted(ctx, claims.ID)
	if err != nil {
		return nil, err
	}
	if blacklisted {
		return nil, ErrTokenBlacklisted
	}

	// Verify session exists
	sessionData, err := h.sessionManager.GetSession(ctx, claims.IdentityID, claims.ID)
	if err != nil {
		return nil, err
	}

	return &ClientAuth{
		IdentityID:  claims.IdentityID,
		SessionID:   claims.ID,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
		Email:       sessionData.Email,
		Device:      claims.Device,
	}, nil
}

// RegisterHandler registers a message handler
func (h *Hub) RegisterHandler(handler MessageHandler) {
	h.handlerRegistry.Register(handler)
}

// HandleClientMessage processes a message from a client using registered handlers
func (h *Hub) HandleClientMessage(ctx context.Context, client *Client, msg *wstypes.WSMessage) error {
	// Check if there's a handler for this event type
	handler, exists := h.handlerRegistry.GetHandler(msg.Type)
	if !exists {
		return nil // Will be handled by client's default handler
	}

	// Delegate to the appropriate handler
	return handler.HandleMessage(ctx, client, msg)
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.shutdown()
			return

		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case msg := <-h.broadcast:
			h.BroadcastMessage(msg)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.identityID] == nil {
		h.clients[client.identityID] = make(map[*Client]bool)
	}
	h.clients[client.identityID][client] = true

	log.Printf("Client connected: identity=%d, session=%s, total=%d",
		client.identityID, client.sessionID, h.totalClients())

	// Send welcome message with user info
	client.SendMessage(wstypes.NewMessage(wstypes.EventTypeConnected, map[string]interface{}{
		"identity_id": client.identityID,
		"session_id":  client.sessionID,
		"roles":       client.roles,
		"permissions": client.permissions,
		"device":      client.device,
	}))
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.identityID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			client.Close()

			if len(clients) == 0 {
				delete(h.clients, client.identityID)
			}

			log.Printf("Client disconnected: identity=%d, session=%s, total=%d",
				client.identityID, client.sessionID, h.totalClients())
		}
	}
}

func (h *Hub) BroadcastMessage(msg *BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if msg.IdentityIDs == nil {
		// Broadcast to all
		for _, clients := range h.clients {
			for client := range clients {
				if client.IsSubscribed(msg.Channel) {
					client.SendMessage(msg.Message)
				}
			}
		}
	} else {
		// Broadcast to specific users
		for _, identityID := range msg.IdentityIDs {
			if clients, ok := h.clients[identityID]; ok {
				for client := range clients {
					if client.IsSubscribed(msg.Channel) {
						client.SendMessage(msg.Message)
					}
				}
			}
		}
	}
}

func (h *Hub) GetConnectedClients(identityID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.clients[identityID]; ok {
		return len(clients)
	}
	return 0
}

func (h *Hub) TotalClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}
	return total
}

// Public methods for broadcasting

func (h *Hub) BroadcastNotification(identityID int64, notification *wstypes.NotificationData) {
	msg := wstypes.NewMessage(wstypes.EventTypeNotification, notification)
	h.broadcast <- &BroadcastMessage{
		IdentityIDs: []int64{identityID},
		Channel:     wstypes.ChannelNotifications,
		Message:     msg,
	}
}

func (h *Hub) BroadcastNotificationCount(identityID int64, count int) {
	msg := wstypes.NewMessage(wstypes.EventTypeNotificationCount, map[string]interface{}{
		"unread_count": count,
	})
	h.broadcast <- &BroadcastMessage{
		IdentityIDs: []int64{identityID},
		Channel:     wstypes.ChannelNotifications,
		Message:     msg,
	}
}

func (h *Hub) BroadcastPermissionChange(identityID int64, change *wstypes.PermissionChangeData) {
	msg := wstypes.NewMessage(wstypes.EventTypePermissionGranted, change)
	h.broadcast <- &BroadcastMessage{
		IdentityIDs: []int64{identityID},
		Channel:     wstypes.ChannelPermissions,
		Message:     msg,
	}
}

func (h *Hub) BroadcastSystemAlert(alert *wstypes.SystemAlertData) {
	msg := wstypes.NewMessage(wstypes.EventTypeSystemAlert, alert)
	h.broadcast <- &BroadcastMessage{
		IdentityIDs: nil,
		Channel:     wstypes.ChannelSystem,
		Message:     msg,
	}
}

func (h *Hub) ForceLogout(identityID int64, sessionID string, reason string) {
	msg := wstypes.NewMessage(wstypes.EventTypeForceLogout, wstypes.SessionEventData{
		SessionID: sessionID,
		Reason:    reason,
		Message:   "You have been logged out",
	})
	h.broadcast <- &BroadcastMessage{
		IdentityIDs: []int64{identityID},
		Channel:     wstypes.ChannelSystem,
		Message:     msg,
	}
}

// IsUserConnected checks if a user has any active connections
func (h *Hub) IsUserConnected(identityID int64) bool {
	return h.GetConnectedClients(identityID) > 0
}

// DisconnectUser forcefully disconnects all sessions for a user
func (h *Hub) DisconnectUser(identityID int64, reason string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[identityID]; ok {
		// Send disconnect message to all clients
		disconnectMsg := wstypes.NewMessage(wstypes.EventTypeDisconnected, map[string]interface{}{
			"reason": reason,
		})

		for client := range clients {
			client.SendMessage(disconnectMsg)
			client.Close()
		}

		delete(h.clients, identityID)
		log.Printf("Disconnected all clients for identity=%d, reason=%s", identityID, reason)
	}
}

func (h *Hub) totalClients() int {
	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}
	return total
}

func (h *Hub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, clients := range h.clients {
		for client := range clients {
			client.Close()
		}
	}
}
