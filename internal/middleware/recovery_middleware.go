// internal/middleware/recovery_middleware.go
package middleware

import (
	"net/http"

	"diary-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
				)
				response.Error(c, http.StatusInternalServerError, "internal server error", nil)
				c.Abort()
			}
		}()
		c.Next()
	}
}
