package cart

import (
	"errors"
	"net/http"
	"strconv"

	cartdomain "zentora-service/internal/domain/cart"
	cartusecase "zentora-service/internal/service/cart"
	"zentora-service/internal/middleware"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	cart *cartusecase.Service
}

func NewHandler(cart *cartusecase.Service) *Handler {
	return &Handler{cart: cart}
}

// GET /api/v1/me/cart
func (h *Handler) GetMyCart(c *gin.Context) {
	userID, ok := middleware.GetIdentityID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	cart, err := h.cart.GetActiveCart(c.Request.Context(), userID)
	if err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusOK, "cart retrieved", gin.H{
		"cart": cart,
	})
}

type upsertItemRequest struct {
	ProductID    int64  `json:"product_id"`
	VariantID    int64  `json:"variant_id"`
	Quantity     int    `json:"quantity"`
	PriceAtAdded string `json:"price_at_added"`
}

// POST /api/v1/me/cart/items
func (h *Handler) AddOrUpdateItem(c *gin.Context) {
	userID, ok := middleware.GetIdentityID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req upsertItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	updated, err := h.cart.AddOrUpdateItem(c.Request.Context(), userID, cartdomain.UpsertCartItemInput{
		ProductID:    req.ProductID,
		VariantID:    req.VariantID,
		Quantity:     req.Quantity,
		PriceAtAdded: req.PriceAtAdded,
	})
	if err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusOK, "cart updated", gin.H{
		"cart": updated,
	})
}

// DELETE /api/v1/me/cart/items/:id
func (h *Handler) RemoveItem(c *gin.Context) {
	userID, ok := middleware.GetIdentityID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || itemID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid item id", err)
		return
	}

	updated, err := h.cart.RemoveItem(c.Request.Context(), userID, itemID)
	if err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusOK, "item removed", gin.H{
		"cart": updated,
	})
}

func handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, cartdomain.ErrInvalidInput),
		errors.Is(err, cartdomain.ErrVariantRequired):
		response.Error(c, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, cartdomain.ErrCartNotFound),
		errors.Is(err, cartdomain.ErrCartItemNotFound):
		response.Error(c, http.StatusNotFound, "not found", err)
	default:
		response.Error(c, http.StatusInternalServerError, "internal server error", err)
	}
}