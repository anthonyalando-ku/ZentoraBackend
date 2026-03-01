// internal/domain/websocket/types.go
package websocket

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents different real-time event types
type EventType string

const (
	// Connection events
	EventTypePing              EventType = "ping"
	EventTypePong              EventType = "pong"
	EventTypeConnected         EventType = "connected"
	EventTypeDisconnected      EventType = "disconnected"
	EventTypeError             EventType = "error"
	
	// Notification events
	EventTypeNotificationRead    EventType = "notification:read"
	EventTypeNotificationReadAll EventType = "notification:read_all"
	EventTypeNotificationList    EventType = "notification:list"
	
	// Notification events (server -> client)
	EventTypeNotification        EventType = "notification"
	EventTypeNotificationCount   EventType = "notification:count"
	
	// Permission events
	EventTypePermissionGranted EventType = "permission:granted"
	EventTypePermissionRevoked EventType = "permission:revoked"
	EventTypeRoleAssigned      EventType = "role:assigned"
	EventTypeRoleRemoved       EventType = "role:removed"
	
	// Session events
	EventTypeSessionExpired    EventType = "session:expired"
	EventTypeSessionRevoked    EventType = "session:revoked"
	EventTypeForceLogout       EventType = "session:force_logout"
	
	// Audit events
	EventTypeAuditLog          EventType = "audit:log"
	
	// System events
	EventTypeSystemAlert       EventType = "system:alert"
	EventTypeSystemMaintenance EventType = "system:maintenance"
	
	// Subscription events
	EventTypeSubscribe         EventType = "subscribe"
	EventTypeUnsubscribe       EventType = "unsubscribe"
)

// WSMessage is the universal message format
type WSMessage struct {
	Type      EventType              `json:"type"`
	Data      interface{}            `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	ID        string                 `json:"id,omitempty"` // For message tracking/acknowledgment
}

// Subscription channels that clients can subscribe to
type ChannelType string

const (
	ChannelNotifications ChannelType = "notifications"
	ChannelPermissions   ChannelType = "permissions"
	ChannelAudit         ChannelType = "audit"
	ChannelSystem        ChannelType = "system"
)

// SubscribeRequest sent by client to subscribe to specific channels
type SubscribeRequest struct {
	Channels []ChannelType `json:"channels"`
}

// UnsubscribeRequest sent by client to unsubscribe from channels
type UnsubscribeRequest struct {
	Channels []ChannelType `json:"channels"`
}

// ErrorData for error events
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NotificationData for notification events
type NotificationData struct {
	ID         int64                  `json:"id"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Type       string                 `json:"type"`
	IsRead     bool                   `json:"is_read"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// PermissionChangeData for permission events
type PermissionChangeData struct {
	PermissionName string                 `json:"permission_name"`
	Resource       string                 `json:"resource"`
	Action         string                 `json:"action"`
	Granted        bool                   `json:"granted"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// RoleChangeData for role events
type RoleChangeData struct {
	RoleName    string                 `json:"role_name"`
	DisplayName string                 `json:"display_name"`
	Assigned    bool                   `json:"assigned"` // true = assigned, false = removed
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SessionEventData for session events
type SessionEventData struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
}

// SystemAlertData for system-wide alerts
type SystemAlertData struct {
	Severity string `json:"severity"` // info, warning, critical
	Title    string `json:"title"`
	Message  string `json:"message"`
	ActionURL string `json:"action_url,omitempty"`
}

// Helper to create messages
func NewMessage(eventType EventType, data interface{}) *WSMessage {
	return &WSMessage{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
		ID:        generateMessageID(),
	}
}

func (m *WSMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func ParseMessage(data []byte) (*WSMessage, error) {
	var msg WSMessage
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

func generateMessageID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}