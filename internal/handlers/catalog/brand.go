package catalog

import (
	"net/http"

	branddomain "zentora-service/internal/domain/brand"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *CatalogHandler) ListBrands(c *gin.Context) {
	filter := branddomain.ListFilter{
		ActiveOnly: c.Query("active_only") == "true",
	}
	brands, err := h.svc.ListBrands(c.Request.Context(), filter)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "brands retrieved", brands)
}

func (h *CatalogHandler) GetBrand(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	b, err := h.svc.GetBrandByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "brand retrieved", b)
}

func (h *CatalogHandler) CreateBrand(c *gin.Context) {
	var req branddomain.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	b, err := h.svc.CreateBrand(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "brand created", b)
}

func (h *CatalogHandler) UpdateBrand(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req branddomain.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	b, err := h.svc.UpdateBrand(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "brand updated", b)
}

func (h *CatalogHandler) DeleteBrand(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteBrand(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "brand deleted", nil)
}
