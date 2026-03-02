package catalog

import (
	"net/http"
	"strconv"

	categorydomain "zentora-service/internal/domain/category"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *CatalogHandler) ListCategories(c *gin.Context) {
	filter := categorydomain.ListFilter{
		ActiveOnly: c.Query("active_only") == "true",
	}

	if parentIDStr := c.Query("parent_id"); parentIDStr != "" {
		id, err := strconv.ParseInt(parentIDStr, 10, 64)
		if err != nil || id < 0 {
			response.Error(c, http.StatusBadRequest, "invalid parent_id", nil)
			return
		}
		filter.ParentID = &id
	}

	categories, err := h.svc.ListCategories(c.Request.Context(), filter)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "categories retrieved", categories)
}

func (h *CatalogHandler) GetCategory(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	cat, err := h.svc.GetCategoryByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "category retrieved", cat)
}

func (h *CatalogHandler) GetCategoryTree(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	tree, err := h.svc.GetCategoryTree(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "category tree retrieved", tree)
}

func (h *CatalogHandler) GetCategoryDescendants(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	descendants, err := h.svc.GetCategoryDescendants(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "descendants retrieved", descendants)
}

func (h *CatalogHandler) CreateCategory(c *gin.Context) {
	var req categorydomain.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	cat, err := h.svc.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "category created", cat)
}

func (h *CatalogHandler) UpdateCategory(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req categorydomain.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	cat, err := h.svc.UpdateCategory(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "category updated", cat)
}

func (h *CatalogHandler) DeleteCategory(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "category deleted", nil)
}

// GetProductCategories   GET /catalog/products/:id/categories
func (h *CatalogHandler) GetProductCategories(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	cats, err := h.svc.GetProductCategories(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "categories retrieved", cats)
}

// AddProductCategory   POST /admin/catalog/products/:id/categories
func (h *CatalogHandler) AddProductCategory(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req categorydomain.SetProductCategoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	for _, catID := range req.CategoryIDs {
		if err := h.svc.AddProductCategory(c.Request.Context(), productID, catID); err != nil {
			response.Error(c, http.StatusInternalServerError, "failed to link category", err)
			return
		}
	}
	response.Success(c, http.StatusOK, "categories linked", nil)
}

// RemoveProductCategory   DELETE /admin/catalog/products/:id/categories/:cat_id
func (h *CatalogHandler) RemoveProductCategory(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	catID, err := parseID(c, "cat_id")
	if err != nil {
		return
	}
	if err := h.svc.RemoveProductCategory(c.Request.Context(), productID, catID); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "category removed", nil)
}
