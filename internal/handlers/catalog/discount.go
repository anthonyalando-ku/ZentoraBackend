package catalog

import (
	"net/http"

	discdomain "zentora-service/internal/domain/discount"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *CatalogHandler) ListDiscounts(c *gin.Context) {
	f := discdomain.ListFilter{ActiveOnly: c.Query("active_only") == "true"}
	if code := c.Query("code"); code != "" {
		f.Code = &code
	}
	discounts, err := h.svc.ListDiscounts(c.Request.Context(), f)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "discounts retrieved", discounts)
}

func (h *CatalogHandler) GetDiscount(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	d, err := h.svc.GetDiscountByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "discount retrieved", d)
}

func (h *CatalogHandler) CreateDiscount(c *gin.Context) {
	var req discdomain.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	d, err := h.svc.CreateDiscount(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "discount created", d)
}

func (h *CatalogHandler) UpdateDiscount(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req discdomain.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	d, err := h.svc.UpdateDiscount(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "discount updated", d)
}

func (h *CatalogHandler) SetDiscountTargets(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req struct {
		Targets []discdomain.TargetInput `json:"targets"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if err := h.svc.SetDiscountTargets(c.Request.Context(), id, req.Targets); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "discount targets updated", nil)
}

func (h *CatalogHandler) DeleteDiscount(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteDiscount(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "discount deleted", nil)
}
