// internal/usecase/notification/notification_service.go
package notification

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"zentora-service/internal/domain/notification"
	"zentora-service/internal/domain/websocket"
	"zentora-service/internal/repository/postgres"
	ws "zentora-service/internal/websocket"
)

// NotificationService handles notification business logic
type NotificationService struct {
	repo *postgres.NotificationRepository
	hub  *ws.Hub
}

func NewNotificationService(repo *postgres.NotificationRepository, hub *ws.Hub) *NotificationService {
	return &NotificationService{
		repo: repo,
		hub:  hub,
	}
}

// CreateAndPush creates a notification and pushes it via WebSocket
func (s *NotificationService) CreateAndPush(ctx context.Context, req *notification.CreateNotificationRequest) (*notification.Notification, error) {
	// Create notification entity
	n := &notification.Notification{
		IdentityID: req.IdentityID,
		Title:      req.Title,
		Message:    req.Message,
		Type:       req.Type,
		Metadata:   req.Metadata,
	}

	if req.ExpiresAt != nil {
		n.ExpiresAt = sql.NullTime{Time: *req.ExpiresAt, Valid: true}
	}

	// Save to database
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Push via WebSocket
	s.pushToWebSocket(n)

	return n, nil
}

// Create creates a notification without pushing to WebSocket
func (s *NotificationService) Create(ctx context.Context, req *notification.CreateNotificationRequest) (*notification.Notification, error) {
	n := &notification.Notification{
		IdentityID: req.IdentityID,
		Title:      req.Title,
		Message:    req.Message,
		Type:       req.Type,
		Metadata:   req.Metadata,
	}

	if req.ExpiresAt != nil {
		n.ExpiresAt = sql.NullTime{Time: *req.ExpiresAt, Valid: true}
	}

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	return n, nil
}

// GetByID retrieves a notification by ID
func (s *NotificationService) GetByID(ctx context.Context, id int64, identityID int64) (*notification.Notification, error) {
	n, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if n.IdentityID != identityID {
		return nil, fmt.Errorf("notification not found")
	}

	return n, nil
}

// GetUserNotifications retrieves notifications for a user with filters
func (s *NotificationService) GetUserNotifications(ctx context.Context, identityID int64, filters *notification.NotificationListFilters) (*notification.NotificationListResponse, error) {
	// Set defaults
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}

	notifications, total, err := s.repo.GetUserNotifications(ctx, identityID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	summary, err := s.repo.GetSummary(ctx, identityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	totalPages := int(total) / filters.PageSize
	if int(total)%filters.PageSize > 0 {
		totalPages++
	}

	return &notification.NotificationListResponse{
		Notifications: notifications,
		Summary:       *summary,
		Total:         total,
		Page:          filters.Page,
		PageSize:      filters.PageSize,
		TotalPages:    totalPages,
	}, nil
}

// GetLatestNotifications retrieves the latest N notifications for a user
func (s *NotificationService) GetLatestNotifications(ctx context.Context, identityID int64, limit int) ([]notification.Notification, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	return s.repo.GetLatestNotifications(ctx, identityID, limit)
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, id int64, identityID int64) error {
	if err := s.repo.MarkAsRead(ctx, id, identityID); err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}

	// Push notification count update via WebSocket
	count, err := s.repo.GetUnreadCount(ctx, identityID)
	if err != nil {
		log.Printf("Failed to get unread count: %v", err)
	} else {
		s.hub.BroadcastNotificationCount(identityID, count)
	}

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(ctx context.Context, identityID int64) error {
	if err := s.repo.MarkAllAsRead(ctx, identityID); err != nil {
		return fmt.Errorf("failed to mark all as read: %w", err)
	}

	// Push notification count update via WebSocket
	s.hub.BroadcastNotificationCount(identityID, 0)

	return nil
}

// GetUnreadCount gets the count of unread notifications
func (s *NotificationService) GetUnreadCount(ctx context.Context, identityID int64) (int, error) {
	return s.repo.GetUnreadCount(ctx, identityID)
}

// GetSummary gets notification summary for a user
func (s *NotificationService) GetSummary(ctx context.Context, identityID int64) (*notification.NotificationSummary, error) {
	return s.repo.GetSummary(ctx, identityID)
}

// Delete deletes a notification
func (s *NotificationService) Delete(ctx context.Context, id int64, identityID int64) error {
	if err := s.repo.Delete(ctx, id, identityID); err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	return nil
}

// DeleteExpiredNotifications deletes expired notifications (should be run as a cron job)
func (s *NotificationService) DeleteExpiredNotifications(ctx context.Context) (int64, error) {
	deleted, err := s.repo.DeleteExpiredNotifications(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired notifications: %w", err)
	}

	log.Printf("Deleted %d expired notifications", deleted)
	return deleted, nil
}

// pushToWebSocket pushes notification to WebSocket
func (s *NotificationService) pushToWebSocket(n *notification.Notification) {
	if s.hub == nil {
		return
	}

	// Convert to WebSocket data format
	wsData := &websocket.NotificationData{
		ID:        n.ID,
		Title:     n.Title,
		Message:   n.Message,
		Type:      string(n.Type),
		IsRead:    n.IsRead,
		Metadata:  n.Metadata,
		CreatedAt: n.CreatedAt,
	}

	// Broadcast to user
	s.hub.BroadcastNotification(n.IdentityID, wsData)
}

// Bulk notification methods

// CreateBulkAndPush creates multiple notifications and pushes them via WebSocket
func (s *NotificationService) CreateBulkAndPush(ctx context.Context, requests []*notification.CreateNotificationRequest) ([]*notification.Notification, error) {
	notifications := make([]*notification.Notification, 0, len(requests))

	for _, req := range requests {
		n, err := s.CreateAndPush(ctx, req)
		if err != nil {
			log.Printf("Failed to create notification for identity %d: %v", req.IdentityID, err)
			continue
		}
		notifications = append(notifications, n)
	}

	return notifications, nil
}

// BroadcastSystemNotification sends a system notification to all users
func (s *NotificationService) BroadcastSystemNotification(ctx context.Context, title, message string, metadata map[string]interface{}) error {
	// This would require getting all active users from auth service
	// For now, just broadcast via WebSocket to all connected clients

	s.hub.BroadcastSystemAlert(&websocket.SystemAlertData{
		Severity: "info",
		Title:    title,
		Message:  message,
	})

	return nil
}

// SendAlertNotification sends an alert notification
func (s *NotificationService) SendAlertNotification(ctx context.Context, identityID int64, title, message string, metadata map[string]interface{}) error {
	req := &notification.CreateNotificationRequest{
		IdentityID: identityID,
		Title:      title,
		Message:    message,
		Type:       notification.TypeAlert,
		Metadata:   metadata,
	}

	_, err := s.CreateAndPush(ctx, req)
	return err
}

// SendInfoNotification sends an info notification
func (s *NotificationService) SendInfoNotification(ctx context.Context, identityID int64, title, message string, metadata map[string]interface{}) error {
	req := &notification.CreateNotificationRequest{
		IdentityID: identityID,
		Title:      title,
		Message:    message,
		Type:       notification.TypeInfo,
		Metadata:   metadata,
	}

	_, err := s.CreateAndPush(ctx, req)
	return err
}

// SendTimedNotification sends a notification that expires after duration
func (s *NotificationService) SendTimedNotification(ctx context.Context, identityID int64, title, message string, duration time.Duration, metadata map[string]interface{}) error {
	expiresAt := time.Now().Add(duration)

	req := &notification.CreateNotificationRequest{
		IdentityID: identityID,
		Title:      title,
		Message:    message,
		Type:       notification.TypeInfo,
		Metadata:   metadata,
		ExpiresAt:  &expiresAt,
	}

	_, err := s.CreateAndPush(ctx, req)
	return err
}
