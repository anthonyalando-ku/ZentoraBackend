package catalog

import (
	"context"
	"database/sql"
	"fmt"

	"zentora-service/internal/domain/brand"
	pgRepo "zentora-service/internal/repository/postgres"
)

func (s *CatalogService) CreateBrand(ctx context.Context, req *brand.CreateRequest) (*brand.Brand, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	b := &brand.Brand{
		Name:     req.Name,
		IsActive: true,
	}
	if req.IsActive != nil {
		b.IsActive = *req.IsActive
	}
	if req.LogoURL != nil {
		b.LogoURL = sql.NullString{String: *req.LogoURL, Valid: true}
	}

	baseSlug := pgRepo.GenerateSlug(req.Name)
	if err := s.createBrandWithSlugRetry(ctx, b, baseSlug); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *CatalogService) createBrandWithSlugRetry(ctx context.Context, b *brand.Brand, baseSlug string) error {
	b.Slug = baseSlug
	for attempt := 1; attempt <= 10; attempt++ {
		if attempt > 1 {
			b.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
		}
		err := s.brandRepo.CreateBrand(ctx, b)
		if err == nil {
			return nil
		}
		if err == brand.ErrSlugConflict {
			continue
		}
		return fmt.Errorf("create brand: %w", err)
	}
	return fmt.Errorf("create brand: could not generate unique slug for %q after 10 attempts", baseSlug)
}

func (s *CatalogService) GetBrandByID(ctx context.Context, id int64) (*brand.Brand, error) {
	return s.brandRepo.GetBrandByID(ctx, id)
}

func (s *CatalogService) GetBrandBySlug(ctx context.Context, slug string) (*brand.Brand, error) {
	return s.brandRepo.GetBrandBySlug(ctx, slug)
}

func (s *CatalogService) ListBrands(ctx context.Context, f brand.ListFilter) ([]brand.Brand, error) {
	brands, err := s.brandRepo.ListBrands(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("list brands: %w", err)
	}
	return brands, nil
}

func (s *CatalogService) UpdateBrand(ctx context.Context, id int64, req *brand.UpdateRequest) (*brand.Brand, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	b, err := s.brandRepo.GetBrandByID(ctx, id)
	if err != nil {
		return nil, err
	}

	nameChanged := false
	if req.Name != nil && *req.Name != b.Name {
		b.Name = *req.Name
		nameChanged = true
	}
	if req.IsActive != nil {
		b.IsActive = *req.IsActive
	}
	if req.LogoURL != nil {
		b.LogoURL = sql.NullString{String: *req.LogoURL, Valid: *req.LogoURL != ""}
	}

	if nameChanged {
		if err := s.updateBrandWithSlugRetry(ctx, b); err != nil {
			return nil, err
		}
	} else {
		if err := s.brandRepo.UpdateBrand(ctx, b); err != nil {
			return nil, fmt.Errorf("update brand: %w", err)
		}
	}
	return b, nil
}

func (s *CatalogService) updateBrandWithSlugRetry(ctx context.Context, b *brand.Brand) error {
	baseSlug := pgRepo.GenerateSlug(b.Name)
	b.Slug = baseSlug
	for attempt := 1; attempt <= 10; attempt++ {
		if attempt > 1 {
			b.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
		}
		err := s.brandRepo.UpdateBrand(ctx, b)
		if err == nil {
			return nil
		}
		if err == brand.ErrSlugConflict {
			continue
		}
		return fmt.Errorf("update brand: %w", err)
	}
	return fmt.Errorf("update brand: could not generate unique slug for %q after 10 attempts", baseSlug)
}

func (s *CatalogService) DeleteBrand(ctx context.Context, id int64) error {
	return s.brandRepo.DeleteBrand(ctx, id)
}
