// internal/handlers/catalog/handler.go
package catalog

import (
	"errors"
	"net/http"
	"strconv"

	"diary-service/internal/domain/catalog"
	"diary-service/internal/middleware"
	xerrors "diary-service/internal/pkg/errors"
	"diary-service/internal/pkg/response"
	catalogSvc "diary-service/internal/service/catalog"

	"github.com/gin-gonic/gin"
)

// CatalogHandler holds HTTP handlers for the catalog domain.
type CatalogHandler struct {
	svc *catalogSvc.CatalogService
}

// NewCatalogHandler creates a new CatalogHandler.
func NewCatalogHandler(svc *catalogSvc.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

// =============================================================================
// Category Handlers
// =============================================================================

// ListCategories   GET /catalog/categories
func (h *CatalogHandler) ListCategories(c *gin.Context) {
	categories, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list categories", err)
		return
	}
	response.Success(c, http.StatusOK, "categories retrieved", categories)
}

// GetCategory   GET /catalog/categories/:id
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

// GetCategoryDescendants   GET /catalog/categories/:id/descendants
func (h *CatalogHandler) GetCategoryDescendants(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	closures, err := h.svc.GetCategoryDescendants(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "descendants retrieved", closures)
}

// CreateCategory   POST /admin/catalog/categories
func (h *CatalogHandler) CreateCategory(c *gin.Context) {
	var req catalog.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	cat, err := h.svc.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create category", err)
		return
	}
	response.Success(c, http.StatusCreated, "category created", cat)
}

// UpdateCategory   PUT /admin/catalog/categories/:id
func (h *CatalogHandler) UpdateCategory(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req catalog.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	cat, err := h.svc.UpdateCategory(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "category updated", cat)
}

// DeleteCategory   DELETE /admin/catalog/categories/:id
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

// =============================================================================
// Brand Handlers
// =============================================================================

// ListBrands   GET /catalog/brands
func (h *CatalogHandler) ListBrands(c *gin.Context) {
	activeOnly := c.DefaultQuery("active_only", "true") == "true"
	brands, err := h.svc.ListBrands(c.Request.Context(), activeOnly)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list brands", err)
		return
	}
	response.Success(c, http.StatusOK, "brands retrieved", brands)
}

// GetBrand   GET /catalog/brands/:id
func (h *CatalogHandler) GetBrand(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	brand, err := h.svc.GetBrandByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "brand retrieved", brand)
}

// CreateBrand   POST /admin/catalog/brands
func (h *CatalogHandler) CreateBrand(c *gin.Context) {
	var req catalog.CreateBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	brand, err := h.svc.CreateBrand(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create brand", err)
		return
	}
	response.Success(c, http.StatusCreated, "brand created", brand)
}

// UpdateBrand   PUT /admin/catalog/brands/:id
func (h *CatalogHandler) UpdateBrand(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req catalog.UpdateBrandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	brand, err := h.svc.UpdateBrand(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "brand updated", brand)
}

// DeleteBrand   DELETE /admin/catalog/brands/:id
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

// =============================================================================
// Product Handlers
// =============================================================================

// ListProducts   GET /catalog/products
func (h *CatalogHandler) ListProducts(c *gin.Context) {
	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 && ps <= 100 {
		pageSize = ps
	}
	products, total, err := h.svc.ListProducts(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list products", err)
		return
	}
	response.Success(c, http.StatusOK, "products retrieved", gin.H{
		"products":   products,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// GetProduct   GET /catalog/products/:id
func (h *CatalogHandler) GetProduct(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	product, err := h.svc.GetProductByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "product retrieved", product)
}

// CreateProduct   POST /admin/catalog/products
func (h *CatalogHandler) CreateProduct(c *gin.Context) {
	createdBy := middleware.MustGetIdentityID(c)

	var req catalog.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	product, err := h.svc.CreateProduct(c.Request.Context(), &req, createdBy)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create product", err)
		return
	}
	response.Success(c, http.StatusCreated, "product created", product)
}

// UpdateProduct   PUT /admin/catalog/products/:id
func (h *CatalogHandler) UpdateProduct(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req catalog.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	product, err := h.svc.UpdateProduct(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "product updated", product)
}

// DeleteProduct   DELETE /admin/catalog/products/:id
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

// ---- Product Tags ----

// GetProductTags   GET /catalog/products/:id/tags
func (h *CatalogHandler) GetProductTags(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	tags, err := h.svc.GetProductTags(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "tags retrieved", tags)
}

// SetProductTags   PUT /admin/catalog/products/:id/tags
func (h *CatalogHandler) SetProductTags(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req catalog.SetProductTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.svc.SetProductTags(c.Request.Context(), id, req.TagNames); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "tags updated", nil)
}

// ---- Product Categories ----

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
	var req catalog.SetProductCategoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
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

// ---- Product Images ----

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

// AddProductImage   POST /admin/catalog/products/:id/images
func (h *CatalogHandler) AddProductImage(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req catalog.AddProductImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	img, err := h.svc.AddProductImage(c.Request.Context(), id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to add image", err)
		return
	}
	response.Success(c, http.StatusCreated, "image added", img)
}

// DeleteProductImage   DELETE /admin/catalog/products/:id/images/:image_id
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

// SetPrimaryImage   PUT /admin/catalog/products/:id/images/:image_id/primary
func (h *CatalogHandler) SetPrimaryImage(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	imageID, err := parseID(c, "image_id")
	if err != nil {
		return
	}
	if err := h.svc.SetPrimaryImage(c.Request.Context(), productID, imageID); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "primary image set", nil)
}

// ---- Product Attribute Values ----

// GetProductAttributeValues   GET /catalog/products/:id/attribute-values
func (h *CatalogHandler) GetProductAttributeValues(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	vals, err := h.svc.GetProductAttributeValues(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute values retrieved", vals)
}

// SetProductAttributeValues   PUT /admin/catalog/products/:id/attribute-values
func (h *CatalogHandler) SetProductAttributeValues(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req catalog.SetProductAttributeValuesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.svc.SetProductAttributeValues(c.Request.Context(), id, req.AttributeValueIDs); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute values updated", nil)
}

// =============================================================================
// Attribute Handlers
// =============================================================================

// ListAttributes   GET /catalog/attributes
func (h *CatalogHandler) ListAttributes(c *gin.Context) {
	attrs, err := h.svc.ListAttributes(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list attributes", err)
		return
	}
	response.Success(c, http.StatusOK, "attributes retrieved", attrs)
}

// GetAttribute   GET /catalog/attributes/:id
func (h *CatalogHandler) GetAttribute(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	attr, err := h.svc.GetAttributeByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute retrieved", attr)
}

// CreateAttribute   POST /admin/catalog/attributes
func (h *CatalogHandler) CreateAttribute(c *gin.Context) {
	var req catalog.CreateAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	attr, err := h.svc.CreateAttribute(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create attribute", err)
		return
	}
	response.Success(c, http.StatusCreated, "attribute created", attr)
}

// UpdateAttribute   PUT /admin/catalog/attributes/:id
func (h *CatalogHandler) UpdateAttribute(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req catalog.UpdateAttributeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	attr, err := h.svc.UpdateAttribute(c.Request.Context(), id, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute updated", attr)
}

// DeleteAttribute   DELETE /admin/catalog/attributes/:id
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

// ListAttributeValues   GET /catalog/attributes/:id/values
func (h *CatalogHandler) ListAttributeValues(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	vals, err := h.svc.ListAttributeValues(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute values retrieved", vals)
}

// AddAttributeValue   POST /admin/catalog/attributes/:id/values
func (h *CatalogHandler) AddAttributeValue(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req catalog.CreateAttributeValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	val, err := h.svc.AddAttributeValue(c.Request.Context(), id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to add attribute value", err)
		return
	}
	response.Success(c, http.StatusCreated, "attribute value added", val)
}

// DeleteAttributeValue   DELETE /admin/catalog/attributes/:id/values/:val_id
func (h *CatalogHandler) DeleteAttributeValue(c *gin.Context) {
	valID, err := parseID(c, "val_id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteAttributeValue(c.Request.Context(), valID); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "attribute value deleted", nil)
}

// =============================================================================
// Variant Handlers
// =============================================================================

// ListProductVariants   GET /catalog/products/:id/variants
func (h *CatalogHandler) ListProductVariants(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}
	variants, err := h.svc.ListVariantsByProduct(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variants retrieved", variants)
}

// CreateVariant   POST /admin/catalog/products/:id/variants
func (h *CatalogHandler) CreateVariant(c *gin.Context) {
	productID, err := parseID(c, "id")
	if err != nil {
		return
	}
	var req catalog.CreateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	variant, err := h.svc.CreateVariant(c.Request.Context(), productID, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "variant created", variant)
}

// UpdateVariant   PUT /admin/catalog/products/:id/variants/:variant_id
func (h *CatalogHandler) UpdateVariant(c *gin.Context) {
	variantID, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	var req catalog.UpdateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	variant, err := h.svc.UpdateVariant(c.Request.Context(), variantID, &req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variant updated", variant)
}

// DeleteVariant   DELETE /admin/catalog/products/:id/variants/:variant_id
func (h *CatalogHandler) DeleteVariant(c *gin.Context) {
	variantID, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	if err := h.svc.DeleteVariant(c.Request.Context(), variantID); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variant deleted", nil)
}

// GetVariantAttributeValues   GET /catalog/products/:id/variants/:variant_id/attribute-values
func (h *CatalogHandler) GetVariantAttributeValues(c *gin.Context) {
	variantID, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	vals, err := h.svc.GetVariantAttributeValues(c.Request.Context(), variantID)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variant attribute values retrieved", vals)
}

// SetVariantAttributeValues   PUT /admin/catalog/products/:id/variants/:variant_id/attribute-values
func (h *CatalogHandler) SetVariantAttributeValues(c *gin.Context) {
	variantID, err := parseID(c, "variant_id")
	if err != nil {
		return
	}
	var req catalog.SetVariantAttributeValuesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.svc.SetVariantAttributeValues(c.Request.Context(), variantID, req.AttributeValueIDs); err != nil {
		handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "variant attribute values updated", nil)
}

// =============================================================================
// Helpers
// =============================================================================

func parseID(c *gin.Context, param string) (int64, error) {
	raw := c.Param(param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid "+param, err)
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func handleError(c *gin.Context, err error) {
	if errors.Is(err, xerrors.ErrNotFound) {
		response.Error(c, http.StatusNotFound, "resource not found", err)
		return
	}
	response.Error(c, http.StatusInternalServerError, "internal error", err)
}
