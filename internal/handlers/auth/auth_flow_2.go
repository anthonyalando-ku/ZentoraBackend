// internal/handlers/auth/auth_handler.go
package auth

import (
	"net/http"
	//"strings"

	"diary-service/internal/domain/auth"
	"diary-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)
// SendEmailVerificationOTP sends OTP to email for verification
func (h *AuthHandler) SendEmailVerificationOTP(c *gin.Context) {
	// h.logger.Info("received request to send email verification OTP",
	// 	zap.String("path", c.FullPath()),
	// 	zap.String("method", c.Request.Method),
	// )
	var req auth.EmailVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// h.logger.Error("invalid email verification request", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.authService.SendEmailVerificationOTP(c.Request.Context(), req.Email)
	if err != nil {
		h.logger.Error("failed to send verification OTP",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		response.Error(c, http.StatusBadRequest, "failed to send verification code", err)
		return
	}

	response.Success(c, http.StatusOK, "verification code sent", result)
}

// ResendEmailVerificationOTP resends OTP to email
func (h *AuthHandler) ResendEmailVerificationOTP(c *gin.Context) {
	var req auth.ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.authService.ResendEmailVerificationOTP(c.Request.Context(), req.Email, req.Token)
	if err != nil {
		h.logger.Error("failed to resend verification OTP",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		response.Error(c, http.StatusBadRequest, "failed to resend verification code", err)
		return
	}

	response.Success(c, http.StatusOK, "verification code resent", result)
}

// VerifyEmailOTP verifies the OTP code
func (h *AuthHandler) VerifyEmailOTP(c *gin.Context) {
	var req auth.OTPVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	result, err := h.authService.VerifyEmailOTP(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to verify OTP",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		response.Error(c, http.StatusBadRequest, "failed to verify code", err)
		return
	}

	response.Success(c, http.StatusOK, "email verified successfully", result)
}