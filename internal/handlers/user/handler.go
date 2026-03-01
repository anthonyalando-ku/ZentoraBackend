// internal/handlers/user/handler.go
package user

import (
	"errors"
	"net/http"
	"strconv"

	"diary-service/internal/domain/user"
	"diary-service/internal/middleware"
	xerrors "diary-service/internal/pkg/errors"
	"diary-service/internal/pkg/response"
	userSvc "diary-service/internal/service/user"

	"github.com/gin-gonic/gin"
)

// UserHandler holds HTTP handlers for user-related operations.
type UserHandler struct {
	svc *userSvc.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(svc *userSvc.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// =============================================================================
// Address Handlers
// =============================================================================

// ListAddresses   GET /me/addresses
func (h *UserHandler) ListAddresses(c *gin.Context) {
	userID := middleware.MustGetIdentityID(c)
	addresses, err := h.svc.ListAddresses(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list addresses", err)
		return
	}
	response.Success(c, http.StatusOK, "addresses retrieved", addresses)
}

// GetAddress   GET /me/addresses/:id
func (h *UserHandler) GetAddress(c *gin.Context) {
	userID := middleware.MustGetIdentityID(c)
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	addr, err := h.svc.GetAddressByID(c.Request.Context(), userID, id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "address retrieved", addr)
}

// CreateAddress   POST /me/addresses
func (h *UserHandler) CreateAddress(c *gin.Context) {
	userID := middleware.MustGetIdentityID(c)
	var req user.CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	addr, err := h.svc.CreateAddress(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create address", err)
		return
	}
	response.Success(c, http.StatusCreated, "address created", addr)
}

// UpdateAddress   PUT /me/addresses/:id
func (h *UserHandler) UpdateAddress(c *gin.Context) {
	userID := middleware.MustGetIdentityID(c)
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req user.UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	addr, err := h.svc.UpdateAddress(c.Request.Context(), userID, id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "address updated", addr)
}

// DeleteAddress   DELETE /me/addresses/:id
func (h *UserHandler) DeleteAddress(c *gin.Context) {
	userID := middleware.MustGetIdentityID(c)
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteAddress(c.Request.Context(), userID, id); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "address deleted", nil)
}

// SetDefaultAddress   PUT /me/addresses/:id/default
func (h *UserHandler) SetDefaultAddress(c *gin.Context) {
	userID := middleware.MustGetIdentityID(c)
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.SetDefaultAddress(c.Request.Context(), userID, id); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "default address updated", nil)
}

// =============================================================================
// Helpers
// =============================================================================

func parseID(c *gin.Context, param string) (int64, error) {
	raw := c.Param(param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid "+param, err)
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func handleError(c *gin.Context, err error) {
	if errors.Is(err, xerrors.ErrNotFound) {
		response.Error(c, http.StatusNotFound, "resource not found", err)
		return
	}
	response.Error(c, http.StatusInternalServerError, "internal error", err)
}
