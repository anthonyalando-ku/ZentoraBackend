package catalog

import (
	"net/http"

	variantdomain "zentora-service/internal/domain/variant"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *CatalogHandler) ListVariantsByProduct(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	activeOnly := c.Query("active_only") == "true"
	variants, err := h.svc.ListVariantsByProduct(c.Request.Context(), productID, activeOnly)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variants retrieved", variants)
}

func (h *CatalogHandler) GetVariant(c *gin.Context) {
	id, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	v, err := h.svc.GetVariantByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variant retrieved", v)
}

func (h *CatalogHandler) CreateVariant(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req variantdomain.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	v, err := h.svc.CreateVariant(c.Request.Context(), productID, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "variant created", v)
}

func (h *CatalogHandler) UpdateVariant(c *gin.Context) {
	id, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	var req variantdomain.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	v, err := h.svc.UpdateVariant(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variant updated", v)
}

func (h *CatalogHandler) DeleteVariant(c *gin.Context) {
	id, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteVariant(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variant deleted", nil)
}

func (h *CatalogHandler) GetVariantAttributeValues(c *gin.Context) {
	id, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	values, err := h.svc.GetVariantAttributeValues(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variant attribute values retrieved", values)
}

func (h *CatalogHandler) SetVariantAttributeValues(c *gin.Context) {
	id, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	var req variantdomain.SetAttributeValuesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if err := h.svc.SetVariantAttributeValues(c.Request.Context(), id, &req); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variant attribute values updated", nil)
}
