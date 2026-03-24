package auth

import (
	"zentora-service/internal/domain/auth"
	"zentora-service/internal/pkg/errors"

	"net/http"
	"strconv"

	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// AdminListUsers lists users with pagination (admin only)
func (h *AuthHandler) AdminListUsers(c *gin.Context) {
	var q auth.AdminUsersListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query", err)
		return
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Size <= 0 {
		q.Size = 20
	}

	resp, err := h.authService.AdminListUsers(c.Request.Context(), q.Page, q.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list users", err)
		return
	}

	response.Success(c, http.StatusOK, "users retrieved", resp)
}

// AdminGetUser gets a hydrated user card (admin only)
func (h *AuthHandler) AdminGetUser(c *gin.Context) {
	identityID := c.GetInt64("id") // NOTE: if your param middleware differs, replace this parsing
	if identityID == 0 {
		// fallback: parse from path
		var err error
		identityID, err = strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || identityID <= 0 {
			response.Error(c, http.StatusBadRequest, "invalid user id", err)
			return
		}
	}

	user, err := h.authService.AdminGetUser(c.Request.Context(), identityID)
	if err != nil {
		if err == xerrors.ErrNotFound {
			response.Error(c, http.StatusNotFound, "user not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get user", err)
		return
	}

	response.Success(c, http.StatusOK, "user retrieved", user)
}

// AdminSearchUsers searches users and returns hydrated cards (admin only)
func (h *AuthHandler) AdminSearchUsers(c *gin.Context) {
	var q auth.AdminUsersSearchQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid query", err)
		return
	}

	items, err := h.authService.AdminSearchUsers(c.Request.Context(), q.Query, q.Limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to search users", err)
		return
	}

	response.Success(c, http.StatusOK, "search results", gin.H{
		"items": items,
		"count": len(items),
	})
}

// AdminDeleteUser soft-deletes a user (admin only)
func (h *AuthHandler) AdminDeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid user id", err)
		return
	}

	var req auth.AdminDeleteUserRequest
	_ = c.ShouldBindJSON(&req) // optional body

	if err := h.authService.AdminDeleteUser(c.Request.Context(), id, req.Reason); err != nil {
		if err == xerrors.ErrNotFound {
			response.Error(c, http.StatusNotFound, "user not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to delete user", err)
		return
	}

	response.Success(c, http.StatusOK, "user deleted", nil)
}

// AdminUserStats returns user counts for dashboard (admin only)
func (h *AuthHandler) AdminUserStats(c *gin.Context) {
	stats, err := h.authService.AdminUserStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to load user stats", err)
		return
	}
	response.Success(c, http.StatusOK, "user stats retrieved", stats)
}