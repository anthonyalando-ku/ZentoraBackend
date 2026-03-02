package catalog

import (
	"net/http"

	tagdomain "zentora-service/internal/domain/tag"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *CatalogHandler) ListTags(c *gin.Context) {
	tags, err := h.svc.ListTags(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "tags retrieved", tags)
}

func (h *CatalogHandler) GetTag(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	t, err := h.svc.GetTagByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "tag retrieved", t)
}

func (h *CatalogHandler) GetProductTags(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	tags, err := h.svc.GetProductTags(c.Request.Context(), productID)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "product tags retrieved", tags)
}

func (h *CatalogHandler) AddTagToProduct(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req tagdomain.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	t, err := h.svc.AddTagToProduct(c.Request.Context(), productID, req.Name)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "tag added", t)
}

func (h *CatalogHandler) RemoveTagFromProduct(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	tagID, err := parseID(c, "tag_id")
	if err != nil {
		return
	}
	if err := h.svc.RemoveTagFromProduct(c.Request.Context(), productID, tagID); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "tag removed", nil)
}

func (h *CatalogHandler) SetProductTags(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req struct {
		TagNames []string `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if err := h.svc.SetProductTags(c.Request.Context(), productID, req.TagNames); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "product tags updated", nil)
}
