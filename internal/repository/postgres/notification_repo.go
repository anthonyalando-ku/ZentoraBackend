// internal/repository/postgres/notification_repository.go
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"zentora-service/internal/domain/notification"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create creates a new notification
func (r *NotificationRepository) Create(ctx context.Context, n *notification.Notification) error {
	query := `
		INSERT INTO notifications (identity_id, title, message, type, metadata, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	var metadataJSON []byte
	var err error
	if n.Metadata != nil {
		metadataJSON, err = json.Marshal(n.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = r.db.QueryRow(
		ctx, query,
		n.IdentityID, n.Title, n.Message, n.Type, metadataJSON, n.ExpiresAt,
	).Scan(&n.ID, &n.CreatedAt)

	return err
}

// FindByID retrieves a notification by ID
func (r *NotificationRepository) FindByID(ctx context.Context, id int64) (*notification.Notification, error) {
	query := `
		SELECT id, identity_id, title, message, type, metadata, is_read, created_at, read_at, expires_at
		FROM notifications
		WHERE id = $1
	`

	var n notification.Notification
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&n.ID, &n.IdentityID, &n.Title, &n.Message, &n.Type,
		&metadataJSON, &n.IsRead, &n.CreatedAt, &n.ReadAt, &n.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("notification not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find notification: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &n.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &n, nil
}

// GetUserNotifications retrieves notifications for a user with filters
func (r *NotificationRepository) GetUserNotifications(ctx context.Context, identityID int64, filters *notification.NotificationListFilters) ([]notification.Notification, int64, error) {
	// Build WHERE clause
	conditions := []string{"identity_id = $1"}
	args := []interface{}{identityID}
	argPos := 2

	// Filter out expired notifications
	conditions = append(conditions, "(expires_at IS NULL OR expires_at > NOW())")

	if filters.IsRead != nil {
		conditions = append(conditions, fmt.Sprintf("is_read = $%d", argPos))
		args = append(args, *filters.IsRead)
		argPos++
	}

	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argPos))
		args = append(args, *filters.Type)
		argPos++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications WHERE %s", whereClause)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Pagination
	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 {
		filters.PageSize = 20
	}

	offset := (filters.Page - 1) * filters.PageSize
	limit := filters.PageSize

	// Query notifications
	query := fmt.Sprintf(`
		SELECT id, identity_id, title, message, type, metadata, is_read, created_at, read_at, expires_at
		FROM notifications
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	notifications := []notification.Notification{}
	for rows.Next() {
		var n notification.Notification
		var metadataJSON []byte

		err := rows.Scan(
			&n.ID, &n.IdentityID, &n.Title, &n.Message, &n.Type,
			&metadataJSON, &n.IsRead, &n.CreatedAt, &n.ReadAt, &n.ExpiresAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &n.Metadata); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		notifications = append(notifications, n)
	}

	return notifications, total, nil
}

// GetLatestNotifications retrieves the latest N notifications for a user
func (r *NotificationRepository) GetLatestNotifications(ctx context.Context, identityID int64, limit int) ([]notification.Notification, error) {
	query := `
		SELECT id, identity_id, title, message, type, metadata, is_read, created_at, read_at, expires_at
		FROM notifications
		WHERE identity_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, identityID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest notifications: %w", err)
	}
	defer rows.Close()

	notifications := []notification.Notification{}
	for rows.Next() {
		var n notification.Notification
		var metadataJSON []byte

		err := rows.Scan(
			&n.ID, &n.IdentityID, &n.Title, &n.Message, &n.Type,
			&metadataJSON, &n.IsRead, &n.CreatedAt, &n.ReadAt, &n.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &n.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		notifications = append(notifications, n)
	}

	return notifications, nil
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id int64, identityID int64) error {
	query := `
		UPDATE notifications
		SET is_read = true, read_at = $1
		WHERE id = $2 AND identity_id = $3 AND is_read = false
	`

	result, err := r.db.Exec(ctx, query, time.Now(), id, identityID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("notification not found or already read")
	}

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, identityID int64) error {
	query := `
		UPDATE notifications
		SET is_read = true, read_at = $1
		WHERE identity_id = $2 AND is_read = false
	`

	_, err := r.db.Exec(ctx, query, time.Now(), identityID)
	return err
}

// GetUnreadCount gets the count of unread notifications
func (r *NotificationRepository) GetUnreadCount(ctx context.Context, identityID int64) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM notifications 
		WHERE identity_id = $1 AND is_read = false AND (expires_at IS NULL OR expires_at > NOW())
	`

	var count int
	err := r.db.QueryRow(ctx, query, identityID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// GetSummary gets notification summary for a user
func (r *NotificationRepository) GetSummary(ctx context.Context, identityID int64) (*notification.NotificationSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN is_read = false THEN 1 END) as unread,
			COUNT(CASE WHEN is_read = true THEN 1 END) as read
		FROM notifications
		WHERE identity_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`

	var summary notification.NotificationSummary
	err := r.db.QueryRow(ctx, query, identityID).Scan(&summary.Total, &summary.TotalUnread, &summary.TotalRead)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification summary: %w", err)
	}

	return &summary, nil
}

// Delete deletes a notification
func (r *NotificationRepository) Delete(ctx context.Context, id int64, identityID int64) error {
	query := `DELETE FROM notifications WHERE id = $1 AND identity_id = $2`

	result, err := r.db.Exec(ctx, query, id, identityID)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

// DeleteExpiredNotifications deletes expired notifications
func (r *NotificationRepository) DeleteExpiredNotifications(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM notifications
		WHERE expires_at IS NOT NULL AND expires_at < NOW()
	`

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired notifications: %w", err)
	}

	return result.RowsAffected(), nil
}
