package catalog

import (
	"net/http"

	attrdomain "zentora-service/internal/domain/attribute"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *CatalogHandler) ListAttributes(c *gin.Context) {
	if c.Query("with_values") == "true" {
		attrs, err := h.svc.ListAttributesWithValues(c.Request.Context())
		if err != nil {
			handleError(c, err)
			return
		}
		response.Success(c, http.StatusOK, "attributes retrieved", attrs)
		return
	}
	attrs, err := h.svc.ListAttributes(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attributes retrieved", attrs)
}

func (h *CatalogHandler) GetAttribute(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	a, err := h.svc.GetAttributeByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute retrieved", a)
}

func (h *CatalogHandler) CreateAttribute(c *gin.Context) {
	var req attrdomain.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	a, err := h.svc.CreateAttribute(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "attribute created", a)
}

func (h *CatalogHandler) UpdateAttribute(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req attrdomain.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	a, err := h.svc.UpdateAttribute(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute updated", a)
}

func (h *CatalogHandler) DeleteAttribute(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteAttribute(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute deleted", nil)
}

func (h *CatalogHandler) ListAttributeValues(c *gin.Context) {
	attributeID, err := parseID(c, "id")
	if err != nil {
		return
	}
	values, err := h.svc.ListAttributeValues(c.Request.Context(), attributeID)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute values retrieved", values)
}

func (h *CatalogHandler) AddAttributeValue(c *gin.Context) {
	attributeID, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req attrdomain.CreateValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	v, err := h.svc.AddAttributeValue(c.Request.Context(), attributeID, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "attribute value added", v)
}

func (h *CatalogHandler) DeleteAttributeValue(c *gin.Context) {
	id, err := parseID(c, "value_id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteAttributeValue(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute value deleted", nil)
}

func (h *CatalogHandler) GetProductAttributeValues(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	values, err := h.svc.GetProductAttributeValues(c.Request.Context(), productID)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "product attribute values retrieved", values)
}

func (h *CatalogHandler) SetProductAttributeValues(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req attrdomain.SetProductAttributeValuesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if err := h.svc.SetProductAttributeValues(c.Request.Context(), productID, req.AttributeValueIDs); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "product attribute values updated", nil)
}
