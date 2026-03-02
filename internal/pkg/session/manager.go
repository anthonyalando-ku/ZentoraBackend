// internal/pkg/session/manager.go
package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	//"strconv"
	"time"

	"zentora-service/internal/repository/postgres"

	"github.com/redis/go-redis/v9"
)

type Manager struct {
	client   *redis.Client
	authRepo *postgres.AuthRepository
}

func NewManager(client *redis.Client, authRepo *postgres.AuthRepository) *Manager {
	return &Manager{
		client:   client,
		authRepo: authRepo,
	}
}

// CreateSession stores a new session in Redis and updates DB
func (m *Manager) CreateSession(ctx context.Context, session *SessionData) error {
	key := m.sessionKey(session.IdentityID, session.JTI)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}

	// Store in Redis
	if err := m.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store session in redis: %w", err)
	}

	// Update last activity in DB
	if session.SessionID > 0 {
		if err := m.authRepo.UpdateSessionActivity(ctx, session.SessionID); err != nil {
			// Log but don't fail - Redis is source of truth
			fmt.Printf("[SESSION] Warning: failed to update DB session activity: %v\n", err)
		}
	}

	return nil
}

// GetSession retrieves a session from Redis with DB fallback
func (m *Manager) GetSession(ctx context.Context, identityID int64, jti string) (*SessionData, error) {
	key := m.sessionKey(identityID, jti)

	// Try Redis first (fast path)
	data, err := m.client.Get(ctx, key).Bytes()
	if err == nil {
		var session SessionData
		if err := json.Unmarshal(data, &session); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session: %w", err)
		}

		// Update last activity
		session.LastActivityAt = time.Now()
		go m.UpdateSessionActivity(context.Background(), identityID, jti)

		return &session, nil
	}

	// Redis miss or error - fallback to database
	if err != redis.Nil {
		fmt.Printf("[REDIS] Warning: Redis error, falling back to DB: %v\n", err)
	}

	// Fetch from database
	dbSession, dbErr := m.authRepo.FindSessionByToken(ctx, jti)
	if dbErr != nil {
		return nil, fmt.Errorf("session not found: %w", dbErr)
	}

	// Verify session belongs to the claimed identity
	if dbSession.IdentityID != identityID {
		return nil, fmt.Errorf("session identity mismatch")
	}

	// Convert DB session to SessionData
	sessionData := &SessionData{
		JTI:            jti,
		IdentityID:     dbSession.IdentityID,
		SessionID:      dbSession.ID,
		Device:         stringFromNull(dbSession.DeviceID),
		DeviceID:       stringFromNull(dbSession.DeviceID),
		DeviceName:     stringFromNull(dbSession.DeviceName),
		IPAddress:      stringFromNull(dbSession.IPAddress),
		UserAgent:      stringFromNull(dbSession.UserAgent),
		Provider:       dbSession.Provider,
		LoginAt:        dbSession.LoginAt,
		LastActivityAt: dbSession.LastActivityAt,
		ExpiresAt:      dbSession.ExpiresAt,
		IsActive:       dbSession.Status == "active",
		Metadata:       dbSession.Metadata,
	}

	// Get user identity and roles/permissions
	identity, err := m.authRepo.FindIdentityByID(ctx, identityID)
	if err == nil && identity.Email.Valid {
		sessionData.Email = identity.Email.String
	}

	// Restore to Redis for next time
	go m.restoreToRedis(context.Background(), sessionData)

	return sessionData, nil
}

// UpdateSessionActivity updates the last activity timestamp
func (m *Manager) UpdateSessionActivity(ctx context.Context, identityID int64, jti string) error {
	key := m.sessionKey(identityID, jti)

	// Get current session
	data, err := m.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil // Session doesn't exist or expired
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return err
	}

	// Update last activity
	session.LastActivityAt = time.Now()

	// Save back to Redis
	updatedData, err := json.Marshal(session)
	if err != nil {
		return err
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl > 0 {
		return m.client.Set(ctx, key, updatedData, ttl).Err()
	}

	return nil
}

// InvalidateSession removes a session from Redis and DB
func (m *Manager) InvalidateSession(ctx context.Context, identityID int64, jti string) error {
	key := m.sessionKey(identityID, jti)

	// Remove from Redis
	if err := m.client.Del(ctx, key).Err(); err != nil {
		fmt.Printf("[SESSION] Warning: failed to delete from Redis: %v\n", err)
	}

	// Get session ID from DB and invalidate
	dbSession, err := m.authRepo.FindSessionByToken(ctx, jti)
	if err == nil {
		if err := m.authRepo.InvalidateSession(ctx, dbSession.ID); err != nil {
			return fmt.Errorf("failed to invalidate DB session: %w", err)
		}
	}

	return nil
}

// InvalidateAllUserSessions removes all sessions for a user
func (m *Manager) InvalidateAllUserSessions(ctx context.Context, identityID int64) error {
	pattern := fmt.Sprintf("session:%d:*", identityID)

	// Delete from Redis
	iter := m.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := m.client.Del(ctx, iter.Val()).Err(); err != nil {
			fmt.Printf("[SESSION] Warning: failed to delete session %s: %v\n", iter.Val(), err)
		}
	}

	// Delete from DB
	if err := m.authRepo.InvalidateAllUserSessions(ctx, identityID); err != nil {
		return fmt.Errorf("failed to invalidate DB sessions: %w", err)
	}

	return iter.Err()
}

// RefreshSession extends the TTL of a session
func (m *Manager) RefreshSession(ctx context.Context, identityID int64, jti string, newExpiry time.Time) error {
	key := m.sessionKey(identityID, jti)
	_ = key

	// Get current session
	session, err := m.GetSession(ctx, identityID, jti)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Update expiry
	session.ExpiresAt = newExpiry
	session.LastActivityAt = time.Now()

	// Save back
	return m.CreateSession(ctx, session)
}

// IsTokenBlacklisted checks if a token is blacklisted
func (m *Manager) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := m.blacklistKey(jti)
	exists, err := m.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}
	return exists > 0, nil
}

// BlacklistToken adds a token to the blacklist
func (m *Manager) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	key := m.blacklistKey(jti)
	return m.client.Set(ctx, key, "1", ttl).Err()
}

// GetUserActiveSessions returns all active sessions for a user
func (m *Manager) GetUserActiveSessions(ctx context.Context, identityID int64) ([]*SessionData, error) {
	pattern := fmt.Sprintf("session:%d:*", identityID)

	var sessions []*SessionData
	iter := m.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		data, err := m.client.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue // Skip invalid sessions
		}

		var session SessionData
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		sessions = append(sessions, &session)
	}

	// If Redis is empty, try DB
	if len(sessions) == 0 {
		// Note: You'd need to add a method to get all sessions by identity
		// For now, we'll just return the Redis results
	}

	return sessions, iter.Err()
}

// Helper functions
func (m *Manager) sessionKey(identityID int64, jti string) string {
	return fmt.Sprintf("session:%d:%s", identityID, jti)
}

func (m *Manager) blacklistKey(jti string) string {
	return fmt.Sprintf("blacklist:%s", jti)
}

func (m *Manager) restoreToRedis(ctx context.Context, session *SessionData) {
	if err := m.CreateSession(ctx, session); err != nil {
		fmt.Printf("[SESSION] Warning: failed to restore session to Redis: %v\n", err)
	}
}

func stringFromNull(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}
