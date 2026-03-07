package catalog

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"

	productdomain "zentora-service/internal/domain/product"
	"zentora-service/internal/middleware"
	"zentora-service/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *CatalogHandler) CreateProduct(c *gin.Context) {
    // Get JSON string from form field
    jsonData := c.PostForm("data")
    if jsonData == "" {
        response.Error(c, http.StatusBadRequest, "data field is required", nil)
        return
    }

    var req productdomain.CreateRequest
    if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
        response.Error(c, http.StatusBadRequest, "invalid request body", err)
        return
    }

    // Parse files
    files, err := parseImageFiles(c)
    if err != nil {
        response.Error(c, http.StatusBadRequest, err.Error(), nil)
        return
    }

    createdBy := middleware.MustGetIdentityID(c)

    p, err := h.svc.CreateProduct(c.Request.Context(), &req, files, createdBy)
    if err != nil {
		h.logger.Error("failed to create product", zap.Error(err))
        handleError(c, err)
        return
    }

    response.Success(c, http.StatusCreated, "product created", p)
}

func (h *CatalogHandler) GetProduct(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	loadRelated := c.Query("load_related") == "true"
	p, err := h.svc.GetProductDetail(c.Request.Context(), id, loadRelated)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "product retrieved", p)
}

func (h *CatalogHandler) GetProductBySlug(c *gin.Context) {
	slug := c.Param("slug")
	p, err := h.svc.GetProductBySlug(c.Request.Context(), slug)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "product retrieved", p)
}

func (h *CatalogHandler) ListProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	req := &productdomain.ListRequest{
		Page:     page,
		PageSize: pageSize,
	}

	if s := c.Query("status"); s != "" {
		st := productdomain.Status(s)
		req.Filter.Status = &st
	}
	if b := c.Query("brand_id"); b != "" {
		if id, err := strconv.ParseInt(b, 10, 64); err == nil {
			req.Filter.BrandID = &id
		}
	}
	if cat := c.Query("category_id"); cat != "" {
		if id, err := strconv.ParseInt(cat, 10, 64); err == nil {
			req.Filter.CategoryID = &id
		}
	}
	if f := c.Query("is_featured"); f == "true" {
		v := true
		req.Filter.IsFeatured = &v
	}
	if q := c.Query("q"); q != "" {
		req.Filter.Search = &q
	}

	products, total, err := h.svc.ListProducts(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "products retrieved", gin.H{
		"items": products,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func (h *CatalogHandler) UpdateProduct(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req productdomain.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	p, err := h.svc.UpdateProduct(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "product updated", p)
}

func (h *CatalogHandler) DeleteProduct(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteProduct(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "product deleted", nil)
}

func (h *CatalogHandler) AddProductImage(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	file, err := c.FormFile("image")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "image file required", nil)
		return
	}
	isPrimary := c.PostForm("is_primary") == "true"
	sortOrder, _ := strconv.Atoi(c.DefaultPostForm("sort_order", "0"))

	img, err := h.svc.AddProductImage(c.Request.Context(), productID, file, isPrimary, sortOrder)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "image added", img)
}

func (h *CatalogHandler) DeleteProductImage(c *gin.Context) {
	imageID, err := parseID(c, "image_id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteProductImage(c.Request.Context(), imageID); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "image deleted", nil)
}

func (h *CatalogHandler) SetPrimaryImage(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req productdomain.SetPrimaryImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if err := h.svc.SetPrimaryImage(c.Request.Context(), productID, req.ImageID); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "primary image set", nil)
}

// GetProductImages   GET /catalog/products/:id/images
func (h *CatalogHandler) GetProductImages(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	images, err := h.svc.GetProductImages(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "images retrieved", images)
}

func parseImageFiles(c *gin.Context) ([]*multipart.FileHeader, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, fmt.Errorf("multipart form required")
	}
	files := form.File["images"]
	if len(files) == 0 {
		return nil, fmt.Errorf("at least one image is required")
	}
	return files, nil
}
