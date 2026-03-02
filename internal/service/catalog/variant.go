package catalog

import (
	"context"
	"database/sql"
	"fmt"

	"zentora-service/internal/domain/variant"
)

func (s *CatalogService) CreateVariant(ctx context.Context, productID int64, req *variant.CreateRequest) (*variant.Variant, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	v := &variant.Variant{
		ProductID: productID,
		SKU:       req.SKU,
		Price:     req.Price,
		IsActive:  true,
	}
	if req.IsActive != nil {
		v.IsActive = *req.IsActive
	}
	if req.Weight != nil {
		v.Weight = sql.NullFloat64{Float64: *req.Weight, Valid: true}
	}

	if err := s.variantRepo.CreateVariant(ctx, nil, v); err != nil {
		return nil, fmt.Errorf("create variant: %w", err)
	}

	if len(req.AttributeValueIDs) > 0 {
		if err := s.variantRepo.SetVariantAttributeValues(ctx, nil, v.ID, req.AttributeValueIDs); err != nil {
			return nil, fmt.Errorf("set variant attribute values: %w", err)
		}
	}

	return v, nil
}

func (s *CatalogService) GetVariantByID(ctx context.Context, id int64) (*variant.Variant, error) {
	return s.variantRepo.GetVariantByID(ctx, id)
}

func (s *CatalogService) GetVariantBySKU(ctx context.Context, sku string) (*variant.Variant, error) {
	return s.variantRepo.GetVariantBySKU(ctx, sku)
}

func (s *CatalogService) ListVariantsByProduct(ctx context.Context, productID int64, activeOnly bool) ([]variant.Variant, error) {
	variants, err := s.variantRepo.ListVariantsByProduct(ctx, productID, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("list variants: %w", err)
	}
	return variants, nil
}

func (s *CatalogService) UpdateVariant(ctx context.Context, id int64, req *variant.UpdateRequest) (*variant.Variant, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

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

func (s *CatalogService) SetVariantAttributeValues(ctx context.Context, variantID int64, req *variant.SetAttributeValuesRequest) error {
	if _, err := s.variantRepo.GetVariantByID(ctx, variantID); err != nil {
		return err
	}
	return s.variantRepo.SetVariantAttributeValues(ctx, nil, variantID, req.AttributeValueIDs)
}

func (s *CatalogService) GetVariantAttributeValues(ctx context.Context, variantID int64) ([]int64, error) {
	if _, err := s.variantRepo.GetVariantByID(ctx, variantID); err != nil {
		return nil, err
	}
	return s.variantRepo.GetVariantAttributeValues(ctx, variantID)
}
