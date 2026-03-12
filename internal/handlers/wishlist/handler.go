package wishlist

import (
	"errors"
	"net/http"
	"strconv"

	wldomain "zentora-service/internal/domain/wishlist"
	wlusecase "zentora-service/internal/service/wishlist"
	"zentora-service/internal/middleware"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	wl *wlusecase.Service
}

func NewHandler(wl *wlusecase.Service) *Handler {
	return &Handler{wl: wl}
}

// GET /api/v1/me/wishlist
func (h *Handler) GetMyWishlist(c *gin.Context) {
	userID, ok := middleware.GetIdentityID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	w, err := h.wl.GetMyWishlist(c.Request.Context(), userID)
	if err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusOK, "wishlist retrieved", gin.H{
		"wishlist": w,
		"count":    len(w.Items),
	})
}

type addReq struct {
	ProductID int64 `json:"product_id"`
	VariantID int64 `json:"variant_id"`
}

// POST /api/v1/me/wishlist/items
func (h *Handler) Add(c *gin.Context) {
	userID, ok := middleware.GetIdentityID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req addReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := h.wl.Add(c.Request.Context(), userID, req.ProductID, req.VariantID); err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "item added to wishlist", nil)
}

// DELETE /api/v1/me/wishlist/items?product_id=1&variant_id=2
func (h *Handler) Remove(c *gin.Context) {
	userID, ok := middleware.GetIdentityID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	productID, err := strconv.ParseInt(c.Query("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid product_id", err)
		return
	}
	variantID, err := strconv.ParseInt(c.Query("variant_id"), 10, 64)
	if err != nil || variantID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid variant_id", err)
		return
	}

	if err := h.wl.Remove(c.Request.Context(), userID, productID, variantID); err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusOK, "item removed from wishlist", nil)
}

// DELETE /api/v1/me/wishlist
func (h *Handler) Clear(c *gin.Context) {
	userID, ok := middleware.GetIdentityID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	if err := h.wl.Clear(c.Request.Context(), userID); err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusOK, "wishlist cleared", nil)
}

func handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, wldomain.ErrInvalidInput):
		response.Error(c, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, wldomain.ErrNotFound):
		response.Error(c, http.StatusNotFound, "not found", err)
	default:
		response.Error(c, http.StatusInternalServerError, "internal server error", err)
	}
}