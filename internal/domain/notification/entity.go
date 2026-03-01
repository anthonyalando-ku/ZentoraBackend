// internal/domain/notification/entity.go
package notification

import (
	"database/sql"
	"time"
)

type NotificationType string

const (
	TypeSystem NotificationType = "system"
	TypeAlert  NotificationType = "alert"
	TypeInfo   NotificationType = "info"
)

type Notification struct {
	ID         int64                  `json:"id" db:"id"`
	IdentityID int64                  `json:"identity_id" db:"identity_id"`
	Title      string                 `json:"title" db:"title"`
	Message    string                 `json:"message" db:"message"`
	Type       NotificationType       `json:"type" db:"type"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	IsRead     bool                   `json:"is_read" db:"is_read"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	ReadAt     sql.NullTime           `json:"read_at,omitempty" db:"read_at"`
	ExpiresAt  sql.NullTime           `json:"expires_at,omitempty" db:"expires_at"`
}

// DTOs

type CreateNotificationRequest struct {
	IdentityID int64                  `json:"identity_id" binding:"required"`
	Title      string                 `json:"title" binding:"required,max=255"`
	Message    string                 `json:"message" binding:"required"`
	Type       NotificationType       `json:"type"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
}

type NotificationListFilters struct {
	IsRead    *bool            `form:"is_read"`
	Type      *NotificationType `form:"type"`
	Page      int              `form:"page" binding:"min=1"`
	PageSize  int              `form:"page_size" binding:"min=1,max=100"`
}

type NotificationSummary struct {
	TotalUnread int `json:"total_unread"`
	TotalRead   int `json:"total_read"`
	Total       int `json:"total"`
}

type NotificationListResponse struct {
	Notifications []Notification      `json:"notifications"`
	Summary       NotificationSummary `json:"summary"`
	Total         int64               `json:"total"`
	Page          int                 `json:"page"`
	PageSize      int                 `json:"page_size"`
	TotalPages    int                 `json:"total_pages"`
}