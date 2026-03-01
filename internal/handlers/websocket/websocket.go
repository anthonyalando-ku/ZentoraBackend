// internal/handlers/websocket_handler.go
package handlers

import (
	"net/http"
	"strings"
	"time"

	"diary-service/internal/pkg/response"
	ws "diary-service/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking for production
		// For now, allow all origins
		return true
	},
}

type WebSocketHandler struct {
	hub    *ws.Hub
	logger *zap.Logger
}

func NewWebSocketHandler(hub *ws.Hub, logger *zap.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
	}
}

// HandleConnection handles WebSocket connection with authentication
func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	// Extract token from query parameter or header
	token := h.extractToken(c)
	if token == "" {
		response.Error(c, http.StatusUnauthorized, "missing authentication token", nil)
		return
	}

	// Authenticate the client
	auth, err := h.hub.AuthenticateClient(c.Request.Context(), token)
	if err != nil {
		h.logger.Error("WebSocket authentication failed",
			zap.Error(err),
			zap.String("ip", c.ClientIP()),
		)
		response.Error(c, http.StatusUnauthorized, "authentication failed", err)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed",
			zap.Error(err),
			zap.String("ip", c.ClientIP()),
		)
		return
	}

	// Create authenticated client
	client := ws.NewClient(h.hub, conn, auth)

	// Register client with hub
	h.hub.Register <- client

	h.logger.Info("WebSocket client connected",
		zap.Int64("identity_id", auth.IdentityID),
		zap.String("session_id", auth.SessionID),
		zap.String("email", auth.Email),
		zap.Strings("roles", auth.Roles),
	)

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()
}

// extractToken extracts token from query param or Authorization header
func (h *WebSocketHandler) extractToken(c *gin.Context) string {
	// Try query parameter first (common for WebSocket)
	token := c.Query("token")
	if token != "" {
		return token
	}

	// Fallback to Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	return ""
}

// GetStats returns WebSocket connection statistics (admin only)
func (h *WebSocketHandler) GetStats(c *gin.Context) {
	// This would be called via REST API with admin auth middleware
	stats := map[string]interface{}{
		"total_connections": h.hub.TotalClients(),
		"timestamp":         time.Now(),
	}

	response.Success(c, http.StatusOK, "WebSocket stats", stats)
}