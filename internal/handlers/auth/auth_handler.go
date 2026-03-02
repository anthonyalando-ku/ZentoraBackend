// internal/handlers/auth/auth_handler.go
package auth

import (
	"net/http"
	//"strings"

	"zentora-service/internal/domain/auth"
	"zentora-service/internal/middleware"
	"zentora-service/internal/pkg/response"
	authUsecase "zentora-service/internal/service/auth"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authService *authUsecase.AuthService
	logger      *zap.Logger
}

func NewAuthHandler(authService *authUsecase.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// ========== Registration ==========

// Register handles user registration (public endpoint)
func (h *AuthHandler) Register(c *gin.Context) {
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	// Set IP and User-Agent
	req.IPAddress = c.ClientIP()
	req.UserAgent = c.GetHeader("User-Agent")

	loginResp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("registration failed",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		response.Error(c, http.StatusBadRequest, "registration failed", err)
		return
	}

	response.Success(c, http.StatusCreated, "registration successful", loginResp)
}

// ========== Login ==========

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	// Set IP and User-Agent
	req.IPAddress = c.ClientIP()
	req.UserAgent = c.GetHeader("User-Agent")

	loginResp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("login failed",
			zap.String("email", req.Email),
			zap.String("ip", req.IPAddress),
			zap.Error(err),
		)
		response.Error(c, http.StatusUnauthorized, "login failed", err)
		return
	}

	h.logger.Info("user logged in",
		zap.Int64("identity_id", loginResp.User.IdentityID),
		zap.String("email", loginResp.User.Email),
	)

	response.Success(c, http.StatusOK, "login successful", loginResp)
}

// ========== Logout ==========

// Logout handles user logout (requires auth)
func (h *AuthHandler) Logout(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)
	jti := middleware.MustGetJTI(c)

	if err := h.authService.Logout(c.Request.Context(), identityID, jti); err != nil {
		h.logger.Error("logout failed",
			zap.Int64("identity_id", identityID),
			zap.Error(err),
		)
		response.Error(c, http.StatusInternalServerError, "logout failed", err)
		return
	}

	response.Success(c, http.StatusOK, "logout successful", nil)
}

// LogoutAll handles logging out all sessions (requires auth)
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	if err := h.authService.LogoutAllSessions(c.Request.Context(), identityID); err != nil {
		response.Error(c, http.StatusInternalServerError, "logout all failed", err)
		return
	}

	response.Success(c, http.StatusOK, "all sessions logged out", nil)
}

// ========== Password Management ==========

// ChangePassword handles password change (requires auth)
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	var req auth.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.authService.ChangePassword(c.Request.Context(), identityID, &req); err != nil {
		response.Error(c, http.StatusBadRequest, "password change failed", err)
		return
	}

	response.Success(c, http.StatusOK, "password changed successfully", nil)
}

// ForgotPassword handles password reset request
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req auth.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		h.logger.Error("forgot password failed",
			zap.String("email", req.Email),
			zap.Error(err),
		)
		// Don't reveal if email exists
	}

	// Always return success to prevent email enumeration
	response.Success(c, http.StatusOK, "if email exists, reset link has been sent", nil)
}

// ResetPassword handles password reset
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req auth.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.Error(c, http.StatusBadRequest, "password reset failed", err)
		return
	}

	response.Success(c, http.StatusOK, "password reset successful", nil)
}

// ========== Profile ==========

// GetMe returns current user profile (requires auth)
func (h *AuthHandler) GetMe(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	profile, err := h.authService.GetProfile(c.Request.Context(), identityID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get profile", err)
		return
	}

	response.Success(c, http.StatusOK, "profile retrieved", profile)
}

// UpdateProfile updates user profile (requires auth)
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	var req auth.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	profile, err := h.authService.UpdateProfile(c.Request.Context(), identityID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update profile", err)
		return
	}

	response.Success(c, http.StatusOK, "profile updated", profile)
}

// ========== Admin-Only Endpoints ==========

// CreateAdmin creates a new admin user (super admin only)
func (h *AuthHandler) CreateAdmin(c *gin.Context) {
	// Verify super admin role (middleware should already check, but double-check)
	if !middleware.IsSuperAdmin(c) {
		response.Error(c, http.StatusForbidden, "only super admins can create admin accounts", nil)
		return
	}

	var req auth.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}

	createdBy := middleware.MustGetIdentityID(c)

	admin, temporaryPassword, err := h.authService.CreateAdmin(c.Request.Context(), &req, createdBy)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create admin", err)
		return
	}

	response.Success(c, http.StatusCreated, "admin created successfully", gin.H{
		"admin":              admin,
		"temporary_password": temporaryPassword,
		"note":               "The temporary password has been sent to the admin's email",
	})
}

// ListAdmins lists all admins (super admin only)
func (h *AuthHandler) ListAdmins(c *gin.Context) {
	admins, err := h.authService.ListAdmins(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list admins", err)
		return
	}

	response.Success(c, http.StatusOK, "admins retrieved", admins)
}

// DeactivateAdmin deactivates an admin account (super admin only)
func (h *AuthHandler) DeactivateAdmin(c *gin.Context) {
	identityID := c.GetInt64("id")
	if identityID == 0 {
		response.Error(c, http.StatusBadRequest, "invalid admin ID", nil)
		return
	}

	if err := h.authService.DeactivateUser(c.Request.Context(), identityID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to deactivate admin", err)
		return
	}

	response.Success(c, http.StatusOK, "admin deactivated", nil)
}

// ========== Email Verification ==========

// VerifyEmail verifies user email
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.Error(c, http.StatusBadRequest, "token is required", nil)
		return
	}

	if err := h.authService.VerifyEmail(c.Request.Context(), token); err != nil {
		response.Error(c, http.StatusBadRequest, "email verification failed", err)
		return
	}

	response.Success(c, http.StatusOK, "email verified successfully", nil)
}

// ResendVerificationEmail resends verification email (requires auth)
func (h *AuthHandler) ResendVerificationEmail(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	if err := h.authService.ResendEmailVerification(c.Request.Context(), identityID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to resend verification email", err)
		return
	}

	response.Success(c, http.StatusOK, "verification email sent", nil)
}

// ========== Session Management ==========

// GetActiveSessions returns all active sessions for current user
func (h *AuthHandler) GetActiveSessions(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)

	sessions, err := h.authService.GetActiveSessions(c.Request.Context(), identityID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get sessions", err)
		return
	}

	response.Success(c, http.StatusOK, "sessions retrieved", sessions)
}

// RevokeSession revokes a specific session
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	identityID := middleware.MustGetIdentityID(c)
	sessionID := c.Param("session_id")

	if err := h.authService.RevokeSession(c.Request.Context(), identityID, sessionID); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to revoke session", err)
		return
	}

	response.Success(c, http.StatusOK, "session revoked", nil)
}
