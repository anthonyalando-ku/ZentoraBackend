package order

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	orderdomain "zentora-service/internal/domain/order"
	"zentora-service/internal/middleware"
	"zentora-service/internal/pkg/response"
	orderusecase "zentora-service/internal/service/order"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	orders *orderusecase.Service
}

func NewHandler(orders *orderusecase.Service) *Handler {
	return &Handler{orders: orders}
}

// POST /api/v1/orders/guest
func (h *Handler) CreateGuestOrder(c *gin.Context) {
	var req orderdomain.CreateGuestOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	o, err := h.orders.CreateGuestOrder(c.Request.Context(), &req)
	if err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "order created", gin.H{"order": o})
}

// POST /api/v1/orders
func (h *Handler) CreateUserOrder(c *gin.Context) {
	userID, ok := middleware.GetIdentityID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req orderdomain.CreateUserOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	o, err := h.orders.CreateUserOrder(c.Request.Context(), userID, &req)
	if err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "order created", gin.H{"order": o})
}

func (h *Handler) GetOrderByID(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "invalid order id", nil)
		return
	}
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid order id", err)
		return
	}
	o, err := h.orders.GetOrderByID(c.Request.Context(), idInt64)
	if err != nil {
		handleErr(c, err)
		return
	}
	response.Success(c, http.StatusOK, "order details", gin.H{"order": o})
}

func (h *Handler) ListOrders(c *gin.Context) {
	var filter orderdomain.ListFilter

	// OrderID
	if v := c.Query("order_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid order_id", err)
			return
		}
		filter.OrderID = &id
	}

	// OrderNumber
	if v := c.Query("order_number"); v != "" {
		filter.OrderNumber = &v
	}

	// UserID
	if v := c.Query("user_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid user_id", err)
			return
		}
		filter.UserID = &id
	}

	// CartID
	if v := c.Query("cart_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid cart_id", err)
			return
		}
		filter.CartID = &id
	}

	// Statuses (comma separated)
	if v := c.Query("statuses"); v != "" {
		statuses := strings.Split(v, ",")
		for _, s := range statuses {
			filter.Statuses = append(filter.Statuses, orderdomain.OrderStatus(strings.TrimSpace(s)))
		}
	}

	// CreatedFrom
	if v := c.Query("created_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid created_from", err)
			return
		}
		filter.CreatedFrom = &t
	}

	// CreatedTo
	if v := c.Query("created_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid created_to", err)
			return
		}
		filter.CreatedTo = &t
	}

	// Limit
	if v := c.DefaultQuery("limit", "20"); v != "" {
		limit, err := strconv.Atoi(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid limit", err)
			return
		}
		filter.Limit = limit
	}

	// Offset
	if v := c.DefaultQuery("offset", "0"); v != "" {
		offset, err := strconv.Atoi(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid offset", err)
			return
		}
		filter.Offset = offset
	}

	// SortBy
	filter.SortBy = c.DefaultQuery("sort_by", "created_at")

	// SortDesc
	if v := c.DefaultQuery("sort_desc", "true"); v != "" {
		desc, err := strconv.ParseBool(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid sort_desc", err)
			return
		}
		filter.SortDesc = desc
	}

	orders, offset, err := h.orders.ListOrders(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list orders", err)
		return
	}

	response.Success(c, http.StatusOK, "orders listed", gin.H{"orders": orders, "offset": offset})
}

// ========== Admin order management ==========

// GET /api/v1/admin/orders/stats
func (h *Handler) AdminOrderStats(c *gin.Context) {
	stats, err := h.orders.GetOrderStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get order stats", err)
		return
	}
	response.Success(c, http.StatusOK, "order stats", stats)
}

// GET /api/v1/admin/orders/by-number?order_number=ZNT-...
func (h *Handler) AdminGetOrderByNumber(c *gin.Context) {
	n := strings.TrimSpace(c.Query("order_number"))
	if n == "" {
		response.Error(c, http.StatusBadRequest, "order_number is required", nil)
		return
	}

	o, err := h.orders.GetOrderByNumber(c.Request.Context(), n)
	if err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusOK, "order details", gin.H{"order": o})
}

// PUT /api/v1/admin/orders/:id/status
func (h *Handler) AdminUpdateOrderStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid order id", err)
		return
	}

	var req orderdomain.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	o, err := h.orders.UpdateOrderStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusOK, "order status updated", gin.H{"order": o})
}

func handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, orderdomain.ErrInvalidInput):
		response.Error(c, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, orderdomain.ErrOutOfStock):
		response.Error(c, http.StatusConflict, "out of stock", err)
	case errors.Is(err, orderdomain.ErrCartNotFound),
		errors.Is(err, orderdomain.ErrAddressNotFound),
		errors.Is(err, orderdomain.ErrVariantNotFound),
		errors.Is(err, orderdomain.ErrNotFound),
		errors.Is(err, orderdomain.ErrProductNotFound):
		response.Error(c, http.StatusNotFound, "not found", err)
	default:
		response.Error(c, http.StatusInternalServerError, "internal server error", err)
	}
}