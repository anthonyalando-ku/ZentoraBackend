// internal/websocket/client.go
package websocket

import (
	"context"
	"log"
	"sync"
	"time"

	wstypes "diary-service/internal/domain/websocket"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

// ClientAuth holds authentication information
type ClientAuth struct {
	IdentityID  int64
	SessionID   string
	Roles       []string
	Permissions []string
	Email       string
	Device      string
}

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan []byte
	identityID  int64
	sessionID   string
	roles       []string
	permissions []string
	device      string
	email       string
	
	// Subscriptions - what channels this client is listening to
	subscriptions map[wstypes.ChannelType]bool
	subMutex      sync.RWMutex
	
	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

func NewClient(hub *Hub, conn *websocket.Conn, auth *ClientAuth) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 256),
		identityID:    auth.IdentityID,
		sessionID:     auth.SessionID,
		roles:         auth.Roles,
		permissions:   auth.Permissions,
		device:        auth.Device,
		email:         auth.Email,
		subscriptions: make(map[wstypes.ChannelType]bool),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// HasRole checks if client has a specific role
func (c *Client) HasRole(role string) bool {
	for _, r := range c.roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if client has a specific permission
func (c *Client) HasPermission(permission string) bool {
	for _, p := range c.permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// Subscribe to a channel (with permission check if needed)
func (c *Client) Subscribe(channel wstypes.ChannelType) bool {
	// Add permission checks for sensitive channels
	switch channel {
	case wstypes.ChannelPermissions:
		// Only admins can subscribe to permission changes
		if !c.HasRole("admin") && !c.HasRole("super_admin") {
			return false
		}
	case wstypes.ChannelAudit:
		// Only super admins can subscribe to audit logs
		if !c.HasRole("super_admin") {
			return false
		}
	}

	c.subMutex.Lock()
	defer c.subMutex.Unlock()
	c.subscriptions[channel] = true
	return true
}

// Unsubscribe from a channel
func (c *Client) Unsubscribe(channel wstypes.ChannelType) {
	c.subMutex.Lock()
	defer c.subMutex.Unlock()
	delete(c.subscriptions, channel)
}

// IsSubscribed checks if client is subscribed to a channel
func (c *Client) IsSubscribed(channel wstypes.ChannelType) bool {
	c.subMutex.RLock()
	defer c.subMutex.RUnlock()
	return c.subscriptions[channel]
}

// GetIdentityID returns the client's identity ID
func (c *Client) GetIdentityID() int64 {
	return c.identityID
}

// GetSessionID returns the client's session ID
func (c *Client) GetSessionID() string {
	return c.sessionID
}

// GetRoles returns the client's roles
func (c *Client) GetRoles() []string {
	return c.roles
}

// GetPermissions returns the client's permissions
func (c *Client) GetPermissions() []string {
	return c.permissions
}

// ReadPump handles incoming messages from client
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("websocket error: %v", err)
				}
				return
			}

			c.handleMessage(message)
		}
	}
}

// WritePump handles outgoing messages to client
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from client
func (c *Client) handleMessage(data []byte) {
	msg, err := wstypes.ParseMessage(data)
	if err != nil {
		c.SendError("invalid_message", "Failed to parse message", err.Error())
		return
	}

	// Try to handle with registered handlers first
	if err := c.hub.HandleClientMessage(context.Background(), c, msg); err != nil {
		c.SendError("handler_error", "Failed to process message", err.Error())
		return
	}

	// Built-in message handling
	switch msg.Type {
	case wstypes.EventTypePing:
		c.SendMessage(wstypes.NewMessage(wstypes.EventTypePong, nil))

	case wstypes.EventTypeSubscribe:
		var req wstypes.SubscribeRequest
		if err := mapToStruct(msg.Data, &req); err != nil {
			c.SendError("invalid_subscribe", "Invalid subscribe request", err.Error())
			return
		}
		for _, channel := range req.Channels {
			c.Subscribe(channel)
		}
		c.SendMessage(wstypes.NewMessage(wstypes.EventTypeSubscribe, map[string]interface{}{
			"channels": req.Channels,
			"status":   "subscribed",
		}))

	case wstypes.EventTypeUnsubscribe:
		var req wstypes.UnsubscribeRequest
		if err := mapToStruct(msg.Data, &req); err != nil {
			c.SendError("invalid_unsubscribe", "Invalid unsubscribe request", err.Error())
			return
		}
		for _, channel := range req.Channels {
			c.Unsubscribe(channel)
		}
		c.SendMessage(wstypes.NewMessage(wstypes.EventTypeUnsubscribe, map[string]interface{}{
			"channels": req.Channels,
			"status":   "unsubscribed",
		}))
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(msg *wstypes.WSMessage) {
	data, err := msg.ToJSON()
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return
	}

	select {
	case c.send <- data:
	case <-c.ctx.Done():
	default:
		// Channel full, close connection
		close(c.send)
		c.hub.unregister <- c
	}
}

// SendError sends an error message to the client
func (c *Client) SendError(code, message, details string) {
	c.SendMessage(wstypes.NewMessage(wstypes.EventTypeError, wstypes.ErrorData{
		Code:    code,
		Message: message,
		Details: details,
	}))
}

// Close gracefully closes the client connection
func (c *Client) Close() {
	c.cancel()
	close(c.send)
}