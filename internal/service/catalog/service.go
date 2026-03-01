// internal/service/catalog/service.go
package catalog

import (
	"context"
	"database/sql"
	"fmt"

	"diary-service/internal/domain/catalog"
	"diary-service/internal/repository/postgres"
	xerrors "diary-service/internal/pkg/errors"
)

// CatalogService provides business logic for the product catalog.
type CatalogService struct {
	categoryRepo  *postgres.CategoryRepository
	brandRepo     *postgres.BrandRepository
	tagRepo       *postgres.TagRepository
	productRepo   *postgres.ProductRepository
	attributeRepo *postgres.AttributeRepository
	variantRepo   *postgres.VariantRepository
}

// NewCatalogService creates a new CatalogService.
func NewCatalogService(
	categoryRepo *postgres.CategoryRepository,
	brandRepo *postgres.BrandRepository,
	tagRepo *postgres.TagRepository,
	productRepo *postgres.ProductRepository,
	attributeRepo *postgres.AttributeRepository,
	variantRepo *postgres.VariantRepository,
) *CatalogService {
	return &CatalogService{
		categoryRepo:  categoryRepo,
		brandRepo:     brandRepo,
		tagRepo:       tagRepo,
		productRepo:   productRepo,
		attributeRepo: attributeRepo,
		variantRepo:   variantRepo,
	}
}

// =============================================================================
// Category
// =============================================================================

func (s *CatalogService) CreateCategory(ctx context.Context, req *catalog.CreateCategoryRequest) (*catalog.Category, error) {
	c := &catalog.Category{
		Name:     req.Name,
		Slug:     req.Slug,
		IsActive: true,
	}
	if req.ParentID != nil {
		c.ParentID = sql.NullInt64{Int64: *req.ParentID, Valid: true}
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}
	if err := s.categoryRepo.CreateCategory(ctx, c); err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	return c, nil
}

func (s *CatalogService) GetCategoryByID(ctx context.Context, id int64) (*catalog.Category, error) {
	return s.categoryRepo.GetCategoryByID(ctx, id)
}

func (s *CatalogService) ListCategories(ctx context.Context) ([]catalog.Category, error) {
	return s.categoryRepo.ListCategories(ctx)
}

func (s *CatalogService) UpdateCategory(ctx context.Context, id int64, req *catalog.UpdateCategoryRequest) (*catalog.Category, error) {
	c, err := s.categoryRepo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.Slug != nil {
		c.Slug = *req.Slug
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}
	// ParentID pointer nil means "don't change"; pointer to zero means "clear parent"
	if req.ParentID != nil {
		if *req.ParentID == 0 {
			c.ParentID = sql.NullInt64{}
		} else {
			c.ParentID = sql.NullInt64{Int64: *req.ParentID, Valid: true}
		}
	}
	if err := s.categoryRepo.UpdateCategory(ctx, c); err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}
	return c, nil
}

func (s *CatalogService) DeleteCategory(ctx context.Context, id int64) error {
	return s.categoryRepo.DeleteCategory(ctx, id)
}

func (s *CatalogService) GetCategoryDescendants(ctx context.Context, id int64) ([]catalog.CategoryClosure, error) {
	return s.categoryRepo.GetCategoryDescendants(ctx, id)
}

// =============================================================================
// Brand
// =============================================================================

func (s *CatalogService) CreateBrand(ctx context.Context, req *catalog.CreateBrandRequest) (*catalog.Brand, error) {
	b := &catalog.Brand{
		Name:     req.Name,
		Slug:     req.Slug,
		IsActive: true,
	}
	if req.LogoURL != nil {
		b.LogoURL = sql.NullString{String: *req.LogoURL, Valid: true}
	}
	if err := s.brandRepo.CreateBrand(ctx, b); err != nil {
		return nil, fmt.Errorf("create brand: %w", err)
	}
	return b, nil
}

func (s *CatalogService) GetBrandByID(ctx context.Context, id int64) (*catalog.Brand, error) {
	return s.brandRepo.GetBrandByID(ctx, id)
}

func (s *CatalogService) ListBrands(ctx context.Context, activeOnly bool) ([]catalog.Brand, error) {
	return s.brandRepo.ListBrands(ctx, activeOnly)
}

func (s *CatalogService) UpdateBrand(ctx context.Context, id int64, req *catalog.UpdateBrandRequest) (*catalog.Brand, error) {
	b, err := s.brandRepo.GetBrandByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		b.Name = *req.Name
	}
	if req.Slug != nil {
		b.Slug = *req.Slug
	}
	if req.IsActive != nil {
		b.IsActive = *req.IsActive
	}
	if req.LogoURL != nil {
		b.LogoURL = sql.NullString{String: *req.LogoURL, Valid: *req.LogoURL != ""}
	}
	if err := s.brandRepo.UpdateBrand(ctx, b); err != nil {
		return nil, fmt.Errorf("update brand: %w", err)
	}
	return b, nil
}

func (s *CatalogService) DeleteBrand(ctx context.Context, id int64) error {
	return s.brandRepo.DeleteBrand(ctx, id)
}

// =============================================================================
// Product
// =============================================================================

func (s *CatalogService) CreateProduct(ctx context.Context, req *catalog.CreateProductRequest, createdBy int64) (*catalog.Product, error) {
	status := "active"
	if req.Status != "" {
		status = req.Status
	}

	p := &catalog.Product{
		Name:       req.Name,
		Slug:       req.Slug,
		BasePrice:  req.BasePrice,
		Status:     status,
		IsFeatured: req.IsFeatured,
		IsDigital:  req.IsDigital,
		CreatedBy:  sql.NullInt64{Int64: createdBy, Valid: true},
	}
	if req.Description != nil {
		p.Description = sql.NullString{String: *req.Description, Valid: true}
	}
	if req.ShortDescription != nil {
		p.ShortDescription = sql.NullString{String: *req.ShortDescription, Valid: true}
	}
	if req.BrandID != nil {
		p.BrandID = sql.NullInt64{Int64: *req.BrandID, Valid: true}
	}

	if err := s.productRepo.CreateProduct(ctx, p); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	// Set categories
	for _, catID := range req.CategoryIDs {
		if err := s.productRepo.AddProductCategory(ctx, p.ID, catID); err != nil {
			return nil, fmt.Errorf("link category %d: %w", catID, err)
		}
	}

	// Set tags (transactional find-or-create inside repo)
	if len(req.TagNames) > 0 {
		if err := s.productRepo.SetProductTags(ctx, p.ID, req.TagNames); err != nil {
			return nil, fmt.Errorf("set tags: %w", err)
		}
	}

	return p, nil
}

func (s *CatalogService) GetProductByID(ctx context.Context, id int64) (*catalog.Product, error) {
	return s.productRepo.GetProductByID(ctx, id)
}

func (s *CatalogService) ListProducts(ctx context.Context, page, pageSize int) ([]catalog.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.productRepo.ListProducts(ctx, pageSize, (page-1)*pageSize)
}

func (s *CatalogService) UpdateProduct(ctx context.Context, id int64, req *catalog.UpdateProductRequest) (*catalog.Product, error) {
	p, err := s.productRepo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Slug != nil {
		p.Slug = *req.Slug
	}
	if req.Description != nil {
		p.Description = sql.NullString{String: *req.Description, Valid: true}
	}
	if req.ShortDescription != nil {
		p.ShortDescription = sql.NullString{String: *req.ShortDescription, Valid: true}
	}
	if req.BrandID != nil {
		p.BrandID = sql.NullInt64{Int64: *req.BrandID, Valid: *req.BrandID != 0}
	}
	if req.BasePrice != nil {
		p.BasePrice = *req.BasePrice
	}
	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.IsFeatured != nil {
		p.IsFeatured = *req.IsFeatured
	}
	if req.IsDigital != nil {
		p.IsDigital = *req.IsDigital
	}
	if err := s.productRepo.UpdateProduct(ctx, p); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	return p, nil
}

func (s *CatalogService) DeleteProduct(ctx context.Context, id int64) error {
	return s.productRepo.DeleteProduct(ctx, id)
}

func (s *CatalogService) SetProductTags(ctx context.Context, productID int64, tagNames []string) error {
	return s.productRepo.SetProductTags(ctx, productID, tagNames)
}

func (s *CatalogService) GetProductTags(ctx context.Context, productID int64) ([]catalog.Tag, error) {
	return s.productRepo.GetProductTags(ctx, productID)
}

func (s *CatalogService) AddProductCategory(ctx context.Context, productID, categoryID int64) error {
	return s.productRepo.AddProductCategory(ctx, productID, categoryID)
}

func (s *CatalogService) RemoveProductCategory(ctx context.Context, productID, categoryID int64) error {
	return s.productRepo.RemoveProductCategory(ctx, productID, categoryID)
}

func (s *CatalogService) GetProductCategories(ctx context.Context, productID int64) ([]catalog.Category, error) {
	return s.productRepo.GetProductCategories(ctx, productID)
}

// ---- Product Images ----

func (s *CatalogService) AddProductImage(ctx context.Context, productID int64, req *catalog.AddProductImageRequest) (*catalog.ProductImage, error) {
	img := &catalog.ProductImage{
		ProductID: productID,
		ImageURL:  req.ImageURL,
		IsPrimary: req.IsPrimary,
		SortOrder: req.SortOrder,
	}
	if err := s.productRepo.AddProductImage(ctx, img); err != nil {
		return nil, fmt.Errorf("add image: %w", err)
	}
	return img, nil
}

func (s *CatalogService) GetProductImages(ctx context.Context, productID int64) ([]catalog.ProductImage, error) {
	return s.productRepo.GetProductImages(ctx, productID)
}

func (s *CatalogService) DeleteProductImage(ctx context.Context, imageID int64) error {
	return s.productRepo.DeleteProductImage(ctx, imageID)
}

func (s *CatalogService) SetPrimaryImage(ctx context.Context, productID, imageID int64) error {
	return s.productRepo.SetPrimaryImage(ctx, productID, imageID)
}

// ---- Product Attribute Values ----

func (s *CatalogService) SetProductAttributeValues(ctx context.Context, productID int64, ids []int64) error {
	return s.productRepo.SetProductAttributeValues(ctx, productID, ids)
}

func (s *CatalogService) GetProductAttributeValues(ctx context.Context, productID int64) ([]catalog.AttributeValue, error) {
	return s.productRepo.GetProductAttributeValues(ctx, productID)
}

// =============================================================================
// Attribute
// =============================================================================

func (s *CatalogService) CreateAttribute(ctx context.Context, req *catalog.CreateAttributeRequest) (*catalog.Attribute, error) {
	a := &catalog.Attribute{
		Name:               req.Name,
		Slug:               req.Slug,
		IsVariantDimension: req.IsVariantDimension,
	}
	if err := s.attributeRepo.CreateAttribute(ctx, a); err != nil {
		return nil, fmt.Errorf("create attribute: %w", err)
	}
	return a, nil
}

func (s *CatalogService) GetAttributeByID(ctx context.Context, id int64) (*catalog.Attribute, error) {
	return s.attributeRepo.GetAttributeByID(ctx, id)
}

func (s *CatalogService) ListAttributes(ctx context.Context) ([]catalog.Attribute, error) {
	return s.attributeRepo.ListAttributes(ctx)
}

func (s *CatalogService) UpdateAttribute(ctx context.Context, id int64, req *catalog.UpdateAttributeRequest) (*catalog.Attribute, error) {
	a, err := s.attributeRepo.GetAttributeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		a.Name = *req.Name
	}
	if req.Slug != nil {
		a.Slug = *req.Slug
	}
	if req.IsVariantDimension != nil {
		a.IsVariantDimension = *req.IsVariantDimension
	}
	if err := s.attributeRepo.UpdateAttribute(ctx, a); err != nil {
		return nil, fmt.Errorf("update attribute: %w", err)
	}
	return a, nil
}

func (s *CatalogService) DeleteAttribute(ctx context.Context, id int64) error {
	return s.attributeRepo.DeleteAttribute(ctx, id)
}

func (s *CatalogService) AddAttributeValue(ctx context.Context, attributeID int64, req *catalog.CreateAttributeValueRequest) (*catalog.AttributeValue, error) {
	// Ensure attribute exists
	if _, err := s.attributeRepo.GetAttributeByID(ctx, attributeID); err != nil {
		return nil, err
	}
	v := &catalog.AttributeValue{
		AttributeID: attributeID,
		Value:       req.Value,
	}
	if err := s.attributeRepo.CreateAttributeValue(ctx, v); err != nil {
		return nil, fmt.Errorf("create attribute value: %w", err)
	}
	return v, nil
}

func (s *CatalogService) ListAttributeValues(ctx context.Context, attributeID int64) ([]catalog.AttributeValue, error) {
	return s.attributeRepo.ListAttributeValues(ctx, attributeID)
}

func (s *CatalogService) DeleteAttributeValue(ctx context.Context, id int64) error {
	return s.attributeRepo.DeleteAttributeValue(ctx, id)
}

// =============================================================================
// Variant
// =============================================================================

func (s *CatalogService) CreateVariant(ctx context.Context, productID int64, req *catalog.CreateVariantRequest) (*catalog.ProductVariant, error) {
	// Ensure product exists
	if _, err := s.productRepo.GetProductByID(ctx, productID); err != nil {
		return nil, err
	}
	v := &catalog.ProductVariant{
		ProductID: productID,
		SKU:       req.SKU,
		Price:     req.Price,
		IsActive:  req.IsActive,
	}
	if req.Weight != nil {
		v.Weight = sql.NullFloat64{Float64: *req.Weight, Valid: true}
	}
	if err := s.variantRepo.CreateVariant(ctx, v); err != nil {
		return nil, fmt.Errorf("create variant: %w", err)
	}
	// Link attribute values
	if len(req.AttributeValueIDs) > 0 {
		if err := s.variantRepo.SetVariantAttributeValues(ctx, v.ID, req.AttributeValueIDs); err != nil {
			return nil, fmt.Errorf("set variant attribute values: %w", err)
		}
	}
	return v, nil
}

func (s *CatalogService) GetVariantByID(ctx context.Context, id int64) (*catalog.ProductVariant, error) {
	return s.variantRepo.GetVariantByID(ctx, id)
}

func (s *CatalogService) ListVariantsByProduct(ctx context.Context, productID int64) ([]catalog.ProductVariant, error) {
	return s.variantRepo.ListVariantsByProduct(ctx, productID)
}

func (s *CatalogService) UpdateVariant(ctx context.Context, id int64, req *catalog.UpdateVariantRequest) (*catalog.ProductVariant, error) {
	v, err := s.variantRepo.GetVariantByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.SKU != nil {
		v.SKU = *req.SKU
	}
	if req.Price != nil {
		v.Price = *req.Price
	}
	if req.IsActive != nil {
		v.IsActive = *req.IsActive
	}
	if req.Weight != nil {
		v.Weight = sql.NullFloat64{Float64: *req.Weight, Valid: true}
	}
	if err := s.variantRepo.UpdateVariant(ctx, v); err != nil {
		return nil, fmt.Errorf("update variant: %w", err)
	}
	return v, nil
}

func (s *CatalogService) DeleteVariant(ctx context.Context, id int64) error {
	return s.variantRepo.DeleteVariant(ctx, id)
}

func (s *CatalogService) SetVariantAttributeValues(ctx context.Context, variantID int64, ids []int64) error {
	return s.variantRepo.SetVariantAttributeValues(ctx, variantID, ids)
}

func (s *CatalogService) GetVariantAttributeValues(ctx context.Context, variantID int64) ([]catalog.AttributeValue, error) {
	return s.variantRepo.GetVariantAttributeValues(ctx, variantID)
}

// =============================================================================
// Helpers
// =============================================================================

// EnsureProductExists returns ErrNotFound when product does not exist.
func (s *CatalogService) EnsureProductExists(ctx context.Context, productID int64) error {
	_, err := s.productRepo.GetProductByID(ctx, productID)
	if err != nil {
		return xerrors.ErrNotFound
	}
	return nil
}
