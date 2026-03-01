// internal/websocket/handlers/notification_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	wstypes "diary-service/internal/domain/websocket"
	"diary-service/internal/repository/postgres"
	ws "diary-service/internal/websocket"
)

type NotificationHandler struct {
	notificationService *postgres.NotificationRepository
}

func NewNotificationHandler(notificationService *postgres.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// SupportedEvents returns events this handler supports
func (h *NotificationHandler) SupportedEvents() []wstypes.EventType {
	return []wstypes.EventType{
		wstypes.EventTypeNotificationRead,
		wstypes.EventTypeNotificationReadAll,
		wstypes.EventTypeNotificationList,
		wstypes.EventTypeNotificationCount,
	}
}

// HandleMessage processes notification-related messages
func (h *NotificationHandler) HandleMessage(ctx context.Context, client *ws.Client, msg *wstypes.WSMessage) error {
	switch msg.Type {
	case wstypes.EventTypeNotificationRead:
		return h.handleMarkAsRead(ctx, client, msg)
	
	case wstypes.EventTypeNotificationReadAll:
		return h.handleMarkAllAsRead(ctx, client, msg)
	
	case wstypes.EventTypeNotificationList:
		return h.handleListNotifications(ctx, client, msg)
	
	case wstypes.EventTypeNotificationCount:
		return h.handleGetCount(ctx, client, msg)
	
	default:
		return fmt.Errorf("unsupported event type: %s", msg.Type)
	}
}

// handleMarkAsRead marks a notification as read
func (h *NotificationHandler) handleMarkAsRead(ctx context.Context, client *ws.Client, msg *wstypes.WSMessage) error {
	var req struct {
		NotificationID int64 `json:"notification_id"`
	}

	if err := mapToStruct(msg.Data, &req); err != nil {
		client.SendError("invalid_request", "Invalid mark as read request", err.Error())
		return err
	}

	// Mark as read in database
	if err := h.notificationService.MarkAsRead(ctx, req.NotificationID, client.GetIdentityID()); err != nil {
		client.SendError("mark_read_failed", "Failed to mark notification as read", err.Error())
		return err
	}

	// Get updated unread count
	count, err := h.notificationService.GetUnreadCount(ctx, client.GetIdentityID())
	if err != nil {
		log.Printf("Failed to get unread count: %v", err)
		count = 0
	}

	// Send success response with updated count
	client.SendMessage(wstypes.NewMessage(wstypes.EventTypeNotificationRead, map[string]interface{}{
		"notification_id": req.NotificationID,
		"success":         true,
		"unread_count":    count,
	}))

	return nil
}

// handleMarkAllAsRead marks all notifications as read
func (h *NotificationHandler) handleMarkAllAsRead(ctx context.Context, client *ws.Client, msg *wstypes.WSMessage) error {
	if err := h.notificationService.MarkAllAsRead(ctx, client.GetIdentityID()); err != nil {
		client.SendError("mark_all_read_failed", "Failed to mark all as read", err.Error())
		return err
	}

	client.SendMessage(wstypes.NewMessage(wstypes.EventTypeNotificationReadAll, map[string]interface{}{
		"success":      true,
		"unread_count": 0,
	}))

	return nil
}

// handleListNotifications returns a list of notifications
func (h *NotificationHandler) handleListNotifications(ctx context.Context, client *ws.Client, msg *wstypes.WSMessage) error {
	var req struct {
		Limit  int    `json:"limit"`
		IsRead *bool  `json:"is_read"`
		Type   *string `json:"type"`
	}

	if err := mapToStruct(msg.Data, &req); err != nil {
		client.SendError("invalid_request", "Invalid list request", err.Error())
		return err
	}

	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}

	notifications, err := h.notificationService.GetLatestNotifications(ctx, client.GetIdentityID(), req.Limit)
	if err != nil {
		client.SendError("list_failed", "Failed to get notifications", err.Error())
		return err
	}

	client.SendMessage(wstypes.NewMessage(wstypes.EventTypeNotificationList, map[string]interface{}{
		"notifications": notifications,
		"count":         len(notifications),
	}))

	return nil
}

// handleGetCount returns unread notification count
func (h *NotificationHandler) handleGetCount(ctx context.Context, client *ws.Client, msg *wstypes.WSMessage) error {
	count, err := h.notificationService.GetUnreadCount(ctx, client.GetIdentityID())
	if err != nil {
		client.SendError("count_failed", "Failed to get unread count", err.Error())
		return err
	}

	client.SendMessage(wstypes.NewMessage(wstypes.EventTypeNotificationCount, map[string]interface{}{
		"unread_count": count,
	}))

	return nil
}

// Helper function to convert interface{} to struct
func mapToStruct(data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, target)
}