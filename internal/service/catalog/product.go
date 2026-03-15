package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"mime/multipart"
	"time"

	"zentora-service/internal/domain/discount"
	"zentora-service/internal/domain/discovery"
	"zentora-service/internal/domain/inventory"
	"zentora-service/internal/domain/product"
	xerrors "zentora-service/internal/pkg/errors"
	pgRepo "zentora-service/internal/repository/postgres"
)

const (
	defaultLocationCode = "MAIN"
	//uploadDir           = "uploads/products"
)

func (s *CatalogService) CreateProduct(
	ctx context.Context,
	req *product.CreateRequest,
	files []*multipart.FileHeader,
	createdBy int64,
) (*product.ProductDetail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, product.ErrImageRequired
	}

	if _, err := s.brandRepo.GetBrandByID(ctx, req.BrandID); err != nil {
		return nil, fmt.Errorf("brand: %w", err)
	}

	for _, catID := range req.CategoryIDs {
		if _, err := s.categoryRepo.GetCategoryByID(ctx, catID); err != nil {
			return nil, fmt.Errorf("category %d: %w", catID, err)
		}
	}

	// imageKit field on the service — nil when not configured (falls back to local)
	savedImages, err := saveImages(files, s.imageKit)
	if err != nil {
		return nil, fmt.Errorf("save images: %w", err)
	}

	images := make([]product.Image, len(savedImages))
	for i, path := range savedImages {
		images[i] = product.Image{
			ImageURL:  path,
			IsPrimary: i == 0,
			SortOrder: i,
		}
	}

	defaultLocID, err := s.resolveDefaultLocation(ctx)
	if err != nil {
		return nil, err
	}

	variants := make([]pgRepo.CreateVariantTxInput, 0, len(req.Variants))
	for _, vi := range req.Variants {
		locID := defaultLocID
		if vi.LocationID != nil {
			locID = *vi.LocationID
		}
		isActive := true
		if vi.IsActive != nil {
			isActive = *vi.IsActive
		}
		variants = append(variants, pgRepo.CreateVariantTxInput{
			SKU:               vi.SKU,
			Price:             vi.Price,
			Weight:            vi.Weight,
			IsActive:          isActive,
			AttributeValueIDs: vi.AttributeValueIDs,
			Quantity:          vi.Quantity,
			LocationID:        locID,
		})
	}

	discountID, err := s.resolveDiscount(ctx, req.Discount)
	if err != nil {
		return nil, err
	}

	p := &product.Product{
		Name:       req.Name,
		Slug:       pgRepo.GenerateSlug(req.Name),
		BasePrice:  req.BasePrice,
		Status:     req.Status,
		IsFeatured: req.IsFeatured,
		IsDigital:  req.IsDigital,
		BrandID:    sql.NullInt64{Int64: req.BrandID, Valid: true},
		CreatedBy:  sql.NullInt64{Int64: createdBy, Valid: true},
	}
	if req.Description != nil {
		p.Description = sql.NullString{String: *req.Description, Valid: true}
	}
	if req.ShortDescription != nil {
		p.ShortDescription = sql.NullString{String: *req.ShortDescription, Valid: true}
	}

	txIn := &pgRepo.CreateProductTxInput{
		Product:           p,
		CategoryIDs:       req.CategoryIDs,
		TagNames:          req.TagNames,
		Images:            images,
		AttributeValueIDs: req.AttributeValueIDs,
		Variants:          variants,
		DiscountID:        discountID,
	}

	if err := s.productRepo.CreateProductTx(ctx, txIn); err != nil {
		if err == product.ErrSlugConflict {
			txIn.Product.Slug = fmt.Sprintf("%s-%d", txIn.Product.Slug, time.Now().UnixMilli())
			if err := s.productRepo.CreateProductTx(ctx, txIn); err != nil {
				return nil, fmt.Errorf("create product (slug retry): %w", err)
			}
		} else {
			return nil, fmt.Errorf("create product: %w", err)
		}
	}

	return s.GetProductDetail(ctx, p.ID, true)
}

func (s *CatalogService) GetProductByID(ctx context.Context, id int64) (*product.Product, error) {
	return s.productRepo.GetProductByID(ctx, id)
}

func (s *CatalogService) GetProductBySlug(ctx context.Context, slug string) (*product.ProductDetail, error) {
	p, err := s.productRepo.GetProductBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, p, true)
}

func (s *CatalogService) GetProductDetail(ctx context.Context, id int64, loadRelated bool) (*product.ProductDetail, error) {
	p, err := s.productRepo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, p, loadRelated)
}

func (s *CatalogService) buildDetail(ctx context.Context, p *product.Product, loadRelated bool) (*product.ProductDetail, error) {
	detail := &product.ProductDetail{Product: *p}

	images, err := s.productRepo.GetProductImages(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("get images: %w", err)
	}
	detail.Images = images

	if !loadRelated {
		return detail, nil
	}

	if cats, err := s.categoryRepo.GetProductCategories(ctx, p.ID); err == nil {
		for _, c := range cats {
			detail.Categories = append(detail.Categories, product.RelatedRef{ID: c.ID, Name: c.Name})
		}
	}
	if tags, err := s.tagRepo.GetProductTags(ctx, p.ID); err == nil {
		for _, t := range tags {
			detail.Tags = append(detail.Tags, product.RelatedRef{ID: t.ID, Name: t.Name})
		}
	}
	if avs, err := s.attributeRepo.GetProductAttributeValues(ctx, p.ID); err == nil {
		for _, av := range avs {
			detail.AttributeValues = append(detail.AttributeValues, product.RelatedRef{ID: av.ID, Name: av.Value})
		}
	}
	if vs, err := s.variantRepo.ListVariantsByProduct(ctx, p.ID, false); err == nil {
		for _, v := range vs {
			detail.Variants = append(detail.Variants, product.RelatedRef{ID: v.ID, Name: v.SKU})
		}
	}

	return detail, nil
}

func (s *CatalogService) ListProducts(
	ctx context.Context,
	req *product.ListRequest,
	sort string,
) ([]discovery.ProductCard, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	return s.productRepo.ListProductsForCatalog(ctx, req, sort)
}

func (s *CatalogService) UpdateProduct(ctx context.Context, id int64, req *product.UpdateRequest) (*product.Product, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	p, err := s.productRepo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil && *req.Name != p.Name {
		p.Name = *req.Name
		p.Slug = pgRepo.GenerateSlug(*req.Name)
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

func (s *CatalogService) EnsureProductExists(ctx context.Context, productID int64) error {
	if _, err := s.productRepo.GetProductByID(ctx, productID); err != nil {
		return xerrors.ErrNotFound
	}
	return nil
}

// AddProductImage uploads a single file and persists the image record.
func (s *CatalogService) AddProductImage(ctx context.Context, productID int64, file *multipart.FileHeader, isPrimary bool, sortOrder int) (*product.Image, error) {
	paths, err := saveImages([]*multipart.FileHeader{file}, s.imageKit)
	if err != nil {
		return nil, fmt.Errorf("save image: %w", err)
	}
	img := &product.Image{
		ProductID: productID,
		ImageURL:  paths[0],
		IsPrimary: isPrimary,
		SortOrder: sortOrder,
	}
	if err := s.productRepo.AddProductImage(ctx, img); err != nil {
		return nil, fmt.Errorf("add product image: %w", err)
	}
	return img, nil
}

func (s *CatalogService) GetProductImages(ctx context.Context, productID int64) ([]product.Image, error) {
	return s.productRepo.GetProductImages(ctx, productID)
}

func (s *CatalogService) DeleteProductImage(ctx context.Context, imageID int64) error {
	return s.productRepo.DeleteProductImage(ctx, imageID)
}

func (s *CatalogService) SetPrimaryImage(ctx context.Context, productID, imageID int64) error {
	return s.productRepo.SetPrimaryImage(ctx, productID, imageID)
}

func (s *CatalogService) resolveDefaultLocation(ctx context.Context) (int64, error) {
	locs, err := s.inventoryRepo.ListLocations(ctx, inventory.LocationFilter{ActiveOnly: true})
	if err != nil {
		return 0, fmt.Errorf("resolve default location: %w", err)
	}
	for _, l := range locs {
		if l.LocationCode.Valid && l.LocationCode.String == defaultLocationCode {
			return l.ID, nil
		}
	}
	if len(locs) > 0 {
		return locs[0].ID, nil
	}
	return 0, fmt.Errorf("no active inventory location found")
}

func (s *CatalogService) resolveDiscount(ctx context.Context, in *product.DiscountInput) (*int64, error) {
	if in == nil {
		return nil, nil
	}
	if in.DiscountID != nil {
		if _, err := s.discountRepo.GetDiscountByID(ctx, *in.DiscountID); err != nil {
			return nil, fmt.Errorf("discount: %w", err)
		}
		return in.DiscountID, nil
	}

	name := "default-product-discount"
	if in.Name != nil {
		name = *in.Name
	}

	discounts, err := s.discountRepo.ListDiscounts(ctx, discount.ListFilter{ActiveOnly: true})
	if err != nil {
		return nil, fmt.Errorf("list discounts: %w", err)
	}
	for _, d := range discounts {
		if d.Name == name {
			return &d.ID, nil
		}
	}

	createReq := &discount.CreateRequest{
		Name:         name,
		DiscountType: discount.TypeFixed,
		Value:        0.01,
		IsActive:     boolPtr(true),
	}
	if in.Code != nil && *in.Code != "" {
		createReq.Code = in.Code
	}

	created, err := s.CreateDiscount(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("create discount: %w", err)
	}
	return &created.ID, nil
}

func boolPtr(b bool) *bool { return &b }