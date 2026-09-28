package deliveryhandler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"io"
	"net/http"
	"zentora-service/internal/domain/delivery"
	"zentora-service/internal/middleware"
	"zentora-service/internal/pkg/response"
)

type Repository interface {
	Get(context.Context) (*delivery.Policy, error)
	Update(context.Context, delivery.Update, int64) (*delivery.Policy, error)
}
type Handler struct {
	repo   Repository
	logger *zap.Logger
}

func NewHandler(repo Repository, logger *zap.Logger) *Handler {
	return &Handler{repo: repo, logger: logger}
}

func (h *Handler) Get(c *gin.Context) {
	p, err := h.repo.Get(c.Request.Context())
	if err != nil {
		h.logger.Error("delivery settings unavailable", zap.Error(err))
		response.Error(c, http.StatusServiceUnavailable, "Delivery information is temporarily unavailable", nil)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, http.StatusOK, "delivery information", p)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := middleware.GetIdentityID(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	var u delivery.Update
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&u); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid delivery settings", nil)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		response.Error(c, http.StatusBadRequest, "Invalid delivery settings", nil)
		return
	}
	if err := u.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "Use a whole KES fee between 1 and 100000 and a note of at most 1000 characters", nil)
		return
	}
	p, err := h.repo.Update(c.Request.Context(), u, id)
	if errors.Is(err, delivery.ErrConflict) {
		response.Error(c, http.StatusConflict, err.Error(), nil)
		return
	}
	if errors.Is(err, delivery.ErrInvalid) {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if err != nil {
		h.logger.Error("delivery settings update failed", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "Could not save delivery settings", nil)
		return
	}
	response.Success(c, http.StatusOK, "delivery settings saved", p)
}
