// internal/app/router.go
package app

import (
	authHandler "diary-service/internal/handlers/auth"
	catalogHandler "diary-service/internal/handlers/catalog"
	notifyHandler "diary-service/internal/handlers/notification"
	userHandler "diary-service/internal/handlers/user"
	wsHandler "diary-service/internal/handlers/websocket"
	"diary-service/internal/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handlers struct {
	AuthHandler    *authHandler.AuthHandler
	NotifHandler   *notifyHandler.NotificationHandler
	WSHandler      *wsHandler.WebSocketHandler
	CatalogHandler *catalogHandler.CatalogHandler
	UserHandler    *userHandler.UserHandler
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
	userRoutes := api.Group("/me")
	userRoutes.Use(h.AuthMiddleware.Auth())
	{
		// Addresses
		userRoutes.GET("/addresses", h.UserHandler.ListAddresses)
		userRoutes.POST("/addresses", h.UserHandler.CreateAddress)
		userRoutes.GET("/addresses/:id", h.UserHandler.GetAddress)
		userRoutes.PUT("/addresses/:id", h.UserHandler.UpdateAddress)
		userRoutes.DELETE("/addresses/:id", h.UserHandler.DeleteAddress)
		userRoutes.PUT("/addresses/:id/default", h.UserHandler.SetDefaultAddress)
	}

	// ==================== Public Catalog Routes ====================
	catalogPublic := api.Group("/catalog")
	{
		// Categories
		catalogPublic.GET("/categories", h.CatalogHandler.ListCategories)
		catalogPublic.GET("/categories/:id", h.CatalogHandler.GetCategory)
		catalogPublic.GET("/categories/:id/descendants", h.CatalogHandler.GetCategoryDescendants)

		// Brands
		catalogPublic.GET("/brands", h.CatalogHandler.ListBrands)
		catalogPublic.GET("/brands/:id", h.CatalogHandler.GetBrand)

		// Products
		catalogPublic.GET("/products", h.CatalogHandler.ListProducts)
		catalogPublic.GET("/products/:id", h.CatalogHandler.GetProduct)
		catalogPublic.GET("/products/:id/images", h.CatalogHandler.GetProductImages)
		catalogPublic.GET("/products/:id/tags", h.CatalogHandler.GetProductTags)
		catalogPublic.GET("/products/:id/categories", h.CatalogHandler.GetProductCategories)
		catalogPublic.GET("/products/:id/variants", h.CatalogHandler.ListProductVariants)
		catalogPublic.GET("/products/:id/attribute-values", h.CatalogHandler.GetProductAttributeValues)
		catalogPublic.GET("/products/:id/variants/:variant_id/attribute-values", h.CatalogHandler.GetVariantAttributeValues)

		// Attributes
		catalogPublic.GET("/attributes", h.CatalogHandler.ListAttributes)
		catalogPublic.GET("/attributes/:id", h.CatalogHandler.GetAttribute)
		catalogPublic.GET("/attributes/:id/values", h.CatalogHandler.ListAttributeValues)
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

	// ==================== Admin Catalog Management ====================
	adminCatalog := api.Group("/admin/catalog")
	adminCatalog.Use(h.AuthMiddleware.AdminOnly()...)
	{
		// Categories
		adminCatalog.POST("/categories", h.CatalogHandler.CreateCategory)
		adminCatalog.PUT("/categories/:id", h.CatalogHandler.UpdateCategory)
		adminCatalog.DELETE("/categories/:id", h.CatalogHandler.DeleteCategory)

		// Brands
		adminCatalog.POST("/brands", h.CatalogHandler.CreateBrand)
		adminCatalog.PUT("/brands/:id", h.CatalogHandler.UpdateBrand)
		adminCatalog.DELETE("/brands/:id", h.CatalogHandler.DeleteBrand)

		// Products
		adminCatalog.POST("/products", h.CatalogHandler.CreateProduct)
		adminCatalog.PUT("/products/:id", h.CatalogHandler.UpdateProduct)
		adminCatalog.DELETE("/products/:id", h.CatalogHandler.DeleteProduct)

		// Product tags
		adminCatalog.PUT("/products/:id/tags", h.CatalogHandler.SetProductTags)

		// Product categories
		adminCatalog.POST("/products/:id/categories", h.CatalogHandler.AddProductCategory)
		adminCatalog.DELETE("/products/:id/categories/:cat_id", h.CatalogHandler.RemoveProductCategory)

		// Product images
		adminCatalog.POST("/products/:id/images", h.CatalogHandler.AddProductImage)
		adminCatalog.DELETE("/products/:id/images/:image_id", h.CatalogHandler.DeleteProductImage)
		adminCatalog.PUT("/products/:id/images/:image_id/primary", h.CatalogHandler.SetPrimaryImage)

		// Product attribute values
		adminCatalog.PUT("/products/:id/attribute-values", h.CatalogHandler.SetProductAttributeValues)

		// Product variants
		adminCatalog.POST("/products/:id/variants", h.CatalogHandler.CreateVariant)
		adminCatalog.PUT("/products/:id/variants/:variant_id", h.CatalogHandler.UpdateVariant)
		adminCatalog.DELETE("/products/:id/variants/:variant_id", h.CatalogHandler.DeleteVariant)
		adminCatalog.PUT("/products/:id/variants/:variant_id/attribute-values", h.CatalogHandler.SetVariantAttributeValues)

		// Attributes
		adminCatalog.POST("/attributes", h.CatalogHandler.CreateAttribute)
		adminCatalog.PUT("/attributes/:id", h.CatalogHandler.UpdateAttribute)
		adminCatalog.DELETE("/attributes/:id", h.CatalogHandler.DeleteAttribute)
		adminCatalog.POST("/attributes/:id/values", h.CatalogHandler.AddAttributeValue)
		adminCatalog.DELETE("/attributes/:id/values/:val_id", h.CatalogHandler.DeleteAttributeValue)
	}

	// Health check
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}