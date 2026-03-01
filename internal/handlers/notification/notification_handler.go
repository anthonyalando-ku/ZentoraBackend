// internal/handlers/notification/notification_handler.go
package notification

import (
	"net/http"
	"strconv"

	"diary-service/internal/domain/notification"
	"diary-service/internal/middleware"
	"diary-service/internal/pkg/response"
	service "diary-service/internal/service/notification"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notificationService *service.NotificationService
}

func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// GetNotifications retrieves paginated notifications for the current user
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	// Parse filters
	var filters notification.NotificationListFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query parameters", err)
		return
	}

	// Get notifications
	result, err := h.notificationService.GetUserNotifications(c.Request.Context(), identityID, &filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get notifications", err)
		return
	}

	response.Success(c, http.StatusOK, "notifications retrieved", result)
}

// GetLatestNotifications retrieves the latest N notifications
func (h *NotificationHandler) GetLatestNotifications(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	// Parse limit from query
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	notifications, err := h.notificationService.GetLatestNotifications(c.Request.Context(), identityID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get notifications", err)
		return
	}

	response.Success(c, http.StatusOK, "notifications retrieved", gin.H{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

// GetNotification retrieves a single notification by ID
func (h *NotificationHandler) GetNotification(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	notifIDStr := c.Param("id")
	notifID, err := strconv.ParseInt(notifIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid notification ID", err)
		return
	}

	notification, err := h.notificationService.GetByID(c.Request.Context(), notifID, identityID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "notification not found", err)
		return
	}

	response.Success(c, http.StatusOK, "notification retrieved", notification)
}

// MarkAsRead marks a notification as read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	notifIDStr := c.Param("id")
	notifID, err := strconv.ParseInt(notifIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid notification ID", err)
		return
	}

	if err := h.notificationService.MarkAsRead(c.Request.Context(), notifID, identityID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to mark as read", err)
		return
	}

	// Get updated unread count
	count, _ := h.notificationService.GetUnreadCount(c.Request.Context(), identityID)

	response.Success(c, http.StatusOK, "notification marked as read", gin.H{
		"unread_count": count,
	})
}

// MarkAllAsRead marks all notifications as read
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	if err := h.notificationService.MarkAllAsRead(c.Request.Context(), identityID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to mark all as read", err)
		return
	}

	response.Success(c, http.StatusOK, "all notifications marked as read", gin.H{
		"unread_count": 0,
	})
}

// GetUnreadCount gets the count of unread notifications
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), identityID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get unread count", err)
		return
	}

	response.Success(c, http.StatusOK, "unread count retrieved", gin.H{
		"unread_count": count,
	})
}

// GetSummary gets notification summary
func (h *NotificationHandler) GetSummary(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	summary, err := h.notificationService.GetSummary(c.Request.Context(), identityID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get summary", err)
		return
	}

	response.Success(c, http.StatusOK, "summary retrieved", summary)
}

// DeleteNotification deletes a notification
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	notifIDStr := c.Param("id")
	notifID, err := strconv.ParseInt(notifIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid notification ID", err)
		return
	}

	if err := h.notificationService.Delete(c.Request.Context(), notifID, identityID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to delete notification", err)
		return
	}

	response.Success(c, http.StatusOK, "notification deleted", nil)
}

// CreateNotification creates a new notification (admin only)
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	var req notification.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.notificationService.CreateAndPush(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create notification", err)
		return
	}

	response.Success(c, http.StatusCreated, "notification created", result)
}

// BroadcastNotification broadcasts a notification to all users (admin only)
func (h *NotificationHandler) BroadcastNotification(c *gin.Context) {
	var req struct {
		Title    string                 `json:"title" binding:"required"`
		Message  string                 `json:"message" binding:"required"`
		Type     notification.NotificationType `json:"type"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.notificationService.BroadcastSystemNotification(
		c.Request.Context(),
		req.Title,
		req.Message,
		req.Metadata,
	); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to broadcast notification", err)
		return
	}

	response.Success(c, http.StatusOK, "notification broadcasted", nil)
}

// SendBulkNotifications sends notifications to multiple users (admin only)
func (h *NotificationHandler) SendBulkNotifications(c *gin.Context) {
	var req struct {
		IdentityIDs []int64                             `json:"identity_ids" binding:"required"`
		Title       string                              `json:"title" binding:"required"`
		Message     string                              `json:"message" binding:"required"`
		Type        notification.NotificationType       `json:"type"`
		Metadata    map[string]interface{}              `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	// Create notification requests for each user
	requests := make([]*notification.CreateNotificationRequest, len(req.IdentityIDs))
	for i, identityID := range req.IdentityIDs {
		requests[i] = &notification.CreateNotificationRequest{
			IdentityID: identityID,
			Title:      req.Title,
			Message:    req.Message,
			Type:       req.Type,
			Metadata:   req.Metadata,
		}
	}

	notifications, err := h.notificationService.CreateBulkAndPush(c.Request.Context(), requests)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to send bulk notifications", err)
		return
	}

	response.Success(c, http.StatusCreated, "bulk notifications sent", gin.H{
		"sent_count": len(notifications),
	})
}