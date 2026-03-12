package review

import (
	"errors"
	"net/http"
	"strconv"

	reviewdomain "zentora-service/internal/domain/review"
	reviewusecase "zentora-service/internal/service/review"
	"zentora-service/internal/middleware"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	reviews *reviewusecase.Service
}

func NewHandler(reviews *reviewusecase.Service) *Handler {
	return &Handler{reviews: reviews}
}

// POST /api/v1/me/reviews
func (h *Handler) Add(c *gin.Context) {
	userID, ok := middleware.GetIdentityID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}

	var req reviewdomain.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	rv, err := h.reviews.AddReview(c.Request.Context(), userID, &req)
	if err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "review created", gin.H{"review": rv})
}

// GET /api/v1/catalog/products/:id/reviews?limit=20&offset=0
func (h *Handler) ListProductReviews(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || productID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid product id", err)
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	items, total, err := h.reviews.ListProductReviews(c.Request.Context(), productID, limit, offset)
	if err != nil {
		handleErr(c, err)
		return
	}

	response.Success(c, http.StatusOK, "reviews retrieved", gin.H{
		"product_id": productID,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
		"items":      items,
	})
}

func handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, reviewdomain.ErrInvalidInput):
		response.Error(c, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, reviewdomain.ErrForbidden):
		response.Error(c, http.StatusForbidden, "forbidden", err)
	case errors.Is(err, reviewdomain.ErrOrderNotCompleted),
		errors.Is(err, reviewdomain.ErrReviewWindowEnded):
		response.Error(c, http.StatusUnprocessableEntity, err.Error(), err)
	case errors.Is(err, reviewdomain.ErrConflict):
		response.Error(c, http.StatusConflict, "review already exists", err)
	case errors.Is(err, reviewdomain.ErrNotFound):
		response.Error(c, http.StatusNotFound, "not found", err)
	default:
		response.Error(c, http.StatusInternalServerError, "internal server error", err)
	}
}