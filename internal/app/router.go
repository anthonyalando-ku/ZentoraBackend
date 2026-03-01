// internal/app/router.go
package app

import (
	authHandler "diary-service/internal/handlers/auth"
	notifyHandler "diary-service/internal/handlers/notification"
	wsHandler "diary-service/internal/handlers/websocket"
	"diary-service/internal/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handlers struct {
	AuthHandler    *authHandler.AuthHandler
	NotifHandler   *notifyHandler.NotificationHandler
	WSHandler      *wsHandler.WebSocketHandler
	AuthMiddleware *middleware.AuthMiddleware
}

func SetupRouter(r *gin.Engine, logger *zap.Logger, h *Handlers) {
	api := r.Group("/api/v1")

	// ==================== WebSocket ====================
	r.GET("/ws", h.WSHandler.HandleConnection)

	// ==================== Public Auth Routes ====================
	authPublic := api.Group("/auth")
	{
		authPublic.POST("/register", h.AuthHandler.Register)
		authPublic.POST("/login", h.AuthHandler.Login)
		authPublic.POST("/forgot-password", h.AuthHandler.ForgotPassword)
		authPublic.POST("/reset-password", h.AuthHandler.ResetPassword)
		authPublic.GET("/verify-email", h.AuthHandler.VerifyEmail)

		// OTP-based email verification flow
		authPublic.POST("/verify-email/send-otp", h.AuthHandler.SendEmailVerificationOTP)
		authPublic.POST("/verify-email/verify-otp", h.AuthHandler.VerifyEmailOTP)
		authPublic.POST("/verify-email/resend-otp", h.AuthHandler.ResendEmailVerificationOTP)
	
	}

	// ==================== Authenticated Routes ====================
	authProtected := api.Group("/auth")
	authProtected.Use(h.AuthMiddleware.Auth())
	{
		authProtected.POST("/logout", h.AuthHandler.Logout)
		authProtected.POST("/logout-all", h.AuthHandler.LogoutAll)
		authProtected.PUT("/change-password", h.AuthHandler.ChangePassword)
		authProtected.GET("/me", h.AuthHandler.GetMe)
		authProtected.PUT("/profile", h.AuthHandler.UpdateProfile)
		authProtected.POST("/resend-verification", h.AuthHandler.ResendVerificationEmail)
		authProtected.GET("/sessions", h.AuthHandler.GetActiveSessions)
		authProtected.DELETE("/sessions/:session_id", h.AuthHandler.RevokeSession)
	}

	// ==================== Notifications (User) ====================
	notifications := api.Group("/notifications")
	notifications.Use(h.AuthMiddleware.Auth())
	{
		notifications.GET("", h.NotifHandler.GetNotifications)
		notifications.GET("/latest", h.NotifHandler.GetLatestNotifications)
		notifications.GET("/:id", h.NotifHandler.GetNotification)
		notifications.GET("/count/unread", h.NotifHandler.GetUnreadCount)
		notifications.GET("/summary", h.NotifHandler.GetSummary)
		notifications.PUT("/:id/read", h.NotifHandler.MarkAsRead)
		notifications.PUT("/read-all", h.NotifHandler.MarkAllAsRead)
		notifications.DELETE("/:id", h.NotifHandler.DeleteNotification)
	}

	// ==================== Admin Notification Management ====================
	adminNotifications := api.Group("/admin/notifications")
	adminNotifications.Use(h.AuthMiddleware.AdminOnly()...) // Use spread operator for slice of middlewares
	{
		adminNotifications.POST("", h.NotifHandler.CreateNotification)
		adminNotifications.POST("/bulk", h.NotifHandler.SendBulkNotifications)
		adminNotifications.POST("/broadcast", h.NotifHandler.BroadcastNotification)
	}

	// ==================== User Routes ====================
	userRoutes := api.Group("/users")
	userRoutes.Use(h.AuthMiddleware.Auth())
	{
		// Add user-specific routes here
	}

	// ==================== Admin Routes (Super Admin Only) ====================
	admin := api.Group("/admin")
	admin.Use(h.AuthMiddleware.SuperAdminOnly()...) // Use spread operator
	{
		admin.POST("/admins", h.AuthHandler.CreateAdmin)
		admin.GET("/admins", h.AuthHandler.ListAdmins)
		admin.DELETE("/admins/:id", h.AuthHandler.DeactivateAdmin)
		admin.GET("/ws/stats", h.WSHandler.GetStats)
	}

	// ==================== Admin Routes (Any Admin) ====================
	adminGeneral := api.Group("/admin/general")
	adminGeneral.Use(h.AuthMiddleware.AdminOnly()...) // Use spread operator
	{
		// Add general admin routes here
	}

	// ==================== Example: Permission-based routes ====================
	reports := api.Group("/reports")
	reports.Use(h.AuthMiddleware.WithPermission("reports.read")...) // Use spread operator
	{
		// Add report routes here
	}

	// Health check
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}