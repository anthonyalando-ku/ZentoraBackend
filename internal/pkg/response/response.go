// internal/pkg/response/response.go
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response defines the standard API response format.
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Success sends a successful response with a message and optional data.
func Success(c *gin.Context, status int, message string, data interface{}) {
	if status == 0 {
		status = http.StatusOK
	}

	c.JSON(status, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error sends a standardized error response.
func Error(c *gin.Context, code int, message string, err error, data ...interface{}) {
	// CRITICAL: Abort FIRST before writing response
	c.Abort()
	
	response := Response{
		Success: false,
		Message: message,
	}

	if err != nil {
		response.Error = err.Error()
	}

	if len(data) > 0 {
		response.Data = data[0]
	}

	c.JSON(code, response)
}

// ValidationError sends a 400 Bad Request response for invalid input.
func ValidationError(c *gin.Context, message string, err error) {
	Error(c, http.StatusBadRequest, message, err)
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message, nil)
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message, nil)
}

// NotFound sends a 404 Not Found response.
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message, nil)
}