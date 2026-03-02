package app

import (
	authHandler "zentora-service/internal/handlers/auth"
	catalogHandler "zentora-service/internal/handlers/catalog"
	notifyHandler "zentora-service/internal/handlers/notification"
	userHandler "zentora-service/internal/handlers/user"
	wsHandler "zentora-service/internal/handlers/websocket"
	"zentora-service/internal/middleware"

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
	// Serve uploaded product images from disk.
	// Example: GET /uploads/products/1234567890_samsung.jpg
	r.Static("/uploads", "./uploads")
	r.Static("/static", "./static")

	api := r.Group("/api/v1")

	r.GET("/ws", h.WSHandler.HandleConnection)

	// ── Auth public ────────────────────────────────────────────────────────────
	authPublic := api.Group("/auth")
	{
		authPublic.POST("/register", h.AuthHandler.Register)
		authPublic.POST("/login", h.AuthHandler.Login)
		authPublic.POST("/forgot-password", h.AuthHandler.ForgotPassword)
		authPublic.POST("/reset-password", h.AuthHandler.ResetPassword)
		authPublic.GET("/verify-email", h.AuthHandler.VerifyEmail)
		authPublic.POST("/verify-email/send-otp", h.AuthHandler.SendEmailVerificationOTP)
		authPublic.POST("/verify-email/verify-otp", h.AuthHandler.VerifyEmailOTP)
		authPublic.POST("/verify-email/resend-otp", h.AuthHandler.ResendEmailVerificationOTP)
	}

	// ── Auth protected ─────────────────────────────────────────────────────────
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

	// ── Notifications (user) ───────────────────────────────────────────────────
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

	// ── Notifications (admin) ──────────────────────────────────────────────────
	adminNotifications := api.Group("/admin/notifications")
	adminNotifications.Use(h.AuthMiddleware.AdminOnly()...)
	{
		adminNotifications.POST("", h.NotifHandler.CreateNotification)
		adminNotifications.POST("/bulk", h.NotifHandler.SendBulkNotifications)
		adminNotifications.POST("/broadcast", h.NotifHandler.BroadcastNotification)
	}

	// ── User / addresses ───────────────────────────────────────────────────────
	userRoutes := api.Group("/me")
	userRoutes.Use(h.AuthMiddleware.Auth())
	{
		userRoutes.GET("/addresses", h.UserHandler.ListAddresses)
		userRoutes.POST("/addresses", h.UserHandler.CreateAddress)
		userRoutes.GET("/addresses/:id", h.UserHandler.GetAddress)
		userRoutes.PUT("/addresses/:id", h.UserHandler.UpdateAddress)
		userRoutes.DELETE("/addresses/:id", h.UserHandler.DeleteAddress)
		userRoutes.PUT("/addresses/:id/default", h.UserHandler.SetDefaultAddress)
	}

	// ── Public catalog ─────────────────────────────────────────────────────────
	catalogPublic := api.Group("/catalog")
	{
		// Categories
		catalogPublic.GET("/categories", h.CatalogHandler.ListCategories)
		catalogPublic.GET("/categories/:id", h.CatalogHandler.GetCategory)
		catalogPublic.GET("/categories/:id/tree", h.CatalogHandler.GetCategoryTree)
		catalogPublic.GET("/categories/:id/descendants", h.CatalogHandler.GetCategoryDescendants)

		// Brands
		catalogPublic.GET("/brands", h.CatalogHandler.ListBrands)
		catalogPublic.GET("/brands/:id", h.CatalogHandler.GetBrand)

		// Tags (read-only public)
		catalogPublic.GET("/tags", h.CatalogHandler.ListTags)
		catalogPublic.GET("/tags/:id", h.CatalogHandler.GetTag)

		// Attributes (read-only public)
		catalogPublic.GET("/attributes", h.CatalogHandler.ListAttributes)
		catalogPublic.GET("/attributes/:id", h.CatalogHandler.GetAttribute)
		catalogPublic.GET("/attributes/:id/values", h.CatalogHandler.ListAttributeValues)

		// Products
		catalogPublic.GET("/products", h.CatalogHandler.ListProducts)
		catalogPublic.GET("/products/slug/:slug", h.CatalogHandler.GetProductBySlug)
		catalogPublic.GET("/products/:id", h.CatalogHandler.GetProduct)
		catalogPublic.GET("/products/:id/images", h.CatalogHandler.GetProductImages)
		catalogPublic.GET("/products/:id/tags", h.CatalogHandler.GetProductTags)
		catalogPublic.GET("/products/:id/categories", h.CatalogHandler.GetProductCategories)
		catalogPublic.GET("/products/:id/attribute-values", h.CatalogHandler.GetProductAttributeValues)
		catalogPublic.GET("/products/:id/variants", h.CatalogHandler.ListVariantsByProduct)
		catalogPublic.GET("/products/:id/variants/:variant_id", h.CatalogHandler.GetVariant)
		catalogPublic.GET("/products/:id/variants/:variant_id/attribute-values", h.CatalogHandler.GetVariantAttributeValues)

		// Inventory (public stock reads)
		catalogPublic.GET("/inventory/variants/:variant_id", h.CatalogHandler.GetInventoryByVariant)
		catalogPublic.GET("/inventory/variants/:variant_id/stock", h.CatalogHandler.GetStockSummary)

		// Inventory locations (public read)
		catalogPublic.GET("/inventory/locations", h.CatalogHandler.ListLocations)
		catalogPublic.GET("/inventory/locations/:id", h.CatalogHandler.GetLocation)
	}

	// ── Super-admin only ───────────────────────────────────────────────────────
	superAdmin := api.Group("/admin")
	superAdmin.Use(h.AuthMiddleware.SuperAdminOnly()...)
	{
		superAdmin.POST("/admins", h.AuthHandler.CreateAdmin)
		superAdmin.GET("/admins", h.AuthHandler.ListAdmins)
		superAdmin.DELETE("/admins/:id", h.AuthHandler.DeactivateAdmin)
		superAdmin.GET("/ws/stats", h.WSHandler.GetStats)
	}

	// ── Admin — reports (permission-gated) ────────────────────────────────────
	reports := api.Group("/reports")
	reports.Use(h.AuthMiddleware.WithPermission("reports.read")...)
	{
		// future report routes
	}

	// ── Admin catalog ──────────────────────────────────────────────────────────
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

		// Attributes
		adminCatalog.POST("/attributes", h.CatalogHandler.CreateAttribute)
		adminCatalog.PUT("/attributes/:id", h.CatalogHandler.UpdateAttribute)
		adminCatalog.DELETE("/attributes/:id", h.CatalogHandler.DeleteAttribute)
		adminCatalog.POST("/attributes/:id/values", h.CatalogHandler.AddAttributeValue)
		adminCatalog.DELETE("/attributes/:id/values/:value_id", h.CatalogHandler.DeleteAttributeValue)

		// Products
		adminCatalog.POST("/products", h.CatalogHandler.CreateProduct)
		adminCatalog.PUT("/products/:id", h.CatalogHandler.UpdateProduct)
		adminCatalog.DELETE("/products/:id", h.CatalogHandler.DeleteProduct)

		// Product images
		adminCatalog.POST("/products/:id/images", h.CatalogHandler.AddProductImage)
		adminCatalog.DELETE("/products/:id/images/:image_id", h.CatalogHandler.DeleteProductImage)
		adminCatalog.PUT("/products/:id/images/:image_id/primary", h.CatalogHandler.SetPrimaryImage)

		// Product categories
		adminCatalog.POST("/products/:id/categories", h.CatalogHandler.AddProductCategory)
		adminCatalog.DELETE("/products/:id/categories/:cat_id", h.CatalogHandler.RemoveProductCategory)

		// Product tags
		adminCatalog.POST("/products/:id/tags", h.CatalogHandler.AddTagToProduct)
		adminCatalog.DELETE("/products/:id/tags/:tag_id", h.CatalogHandler.RemoveTagFromProduct)
		adminCatalog.PUT("/products/:id/tags", h.CatalogHandler.SetProductTags)

		// Product attribute values
		adminCatalog.PUT("/products/:id/attribute-values", h.CatalogHandler.SetProductAttributeValues)

		// Product variants
		adminCatalog.POST("/products/:id/variants", h.CatalogHandler.CreateVariant)
		adminCatalog.PUT("/products/:id/variants/:variant_id", h.CatalogHandler.UpdateVariant)
		adminCatalog.DELETE("/products/:id/variants/:variant_id", h.CatalogHandler.DeleteVariant)
		adminCatalog.PUT("/products/:id/variants/:variant_id/attribute-values", h.CatalogHandler.SetVariantAttributeValues)

		// Inventory locations
		adminCatalog.POST("/inventory/locations", h.CatalogHandler.CreateLocation)
		adminCatalog.PUT("/inventory/locations/:id", h.CatalogHandler.UpdateLocation)
		adminCatalog.DELETE("/inventory/locations/:id", h.CatalogHandler.DeleteLocation)

		// Inventory items
		adminCatalog.PUT("/inventory/items", h.CatalogHandler.UpsertInventoryItem)
		adminCatalog.DELETE("/inventory/variants/:variant_id/locations/:location_id", h.CatalogHandler.DeleteInventoryItem)

		// Stock operations
		adminCatalog.PUT("/inventory/variants/:variant_id/locations/:location_id/adjust", h.CatalogHandler.AdjustAvailableStock)
		adminCatalog.PUT("/inventory/variants/:variant_id/locations/:location_id/reserve", h.CatalogHandler.ReserveStock)
		adminCatalog.PUT("/inventory/variants/:variant_id/locations/:location_id/release", h.CatalogHandler.ReleaseStock)

		// Discounts
		adminCatalog.GET("/discounts", h.CatalogHandler.ListDiscounts)
		adminCatalog.GET("/discounts/:id", h.CatalogHandler.GetDiscount)
		adminCatalog.POST("/discounts", h.CatalogHandler.CreateDiscount)
		adminCatalog.PUT("/discounts/:id", h.CatalogHandler.UpdateDiscount)
		adminCatalog.DELETE("/discounts/:id", h.CatalogHandler.DeleteDiscount)
		adminCatalog.PUT("/discounts/:id/targets", h.CatalogHandler.SetDiscountTargets)
	}

	// ── Health ─────────────────────────────────────────────────────────────────
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}