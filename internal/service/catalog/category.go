package catalog

import (
	"context"
	"database/sql"
	"fmt"

	"zentora-service/internal/domain/category"
	pgRepo "zentora-service/internal/repository/postgres"
)

// ─── Create ──────────────────────────────────────────────────────────────────

// CreateCategory validates the request, auto-generates the slug from the name,
// guards against circular parent references, and persists the category.
// The slug collision retry loop handles the rare case where the generated slug
// already exists (appends "-2", "-3", … up to 10 attempts).
func (s *CatalogService) CreateCategory(
	ctx context.Context,
	req *category.CreateRequest,
) (*category.Category, error) {
	// 1. Domain validation
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 2. Guard: parent must exist
	if req.ParentID != nil {
		if _, err := s.categoryRepo.GetCategoryByID(ctx, *req.ParentID); err != nil {
			return nil, fmt.Errorf("parent category: %w", err)
		}
	}

	// 3. Build entity (slug auto-generated inside insertCategory, but we
	//    derive the base slug here so we can retry on collision).
	c := &category.Category{
		Name:     req.Name,
		IsActive: true,
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}
	if req.ParentID != nil {
		c.ParentID = sql.NullInt64{Int64: *req.ParentID, Valid: true}
	}

	// 4. Persist — retry on slug collision (up to 10 attempts)
	baseSlug := pgRepo.GenerateSlug(req.Name)
	if err := s.createCategoryWithSlugRetry(ctx, c, baseSlug); err != nil {
		return nil, err
	}

	return c, nil
}

// createCategoryWithSlugRetry tries to insert the category, suffixing the slug
// with "-2", "-3", … on ErrSlugConflict.
func (s *CatalogService) createCategoryWithSlugRetry(
	ctx context.Context,
	c *category.Category,
	baseSlug string,
) error {
	c.Slug = baseSlug
	for attempt := 1; attempt <= 10; attempt++ {
		if attempt > 1 {
			c.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
		}

		err := s.categoryRepo.CreateCategory(ctx, nil, c)
		if err == nil {
			return nil
		}
		if err == category.ErrSlugConflict {
			continue
		}
		return fmt.Errorf("create category: %w", err)
	}
	return fmt.Errorf("create category: could not generate unique slug for %q after 10 attempts", baseSlug)
}

// ─── Read ─────────────────────────────────────────────────────────────────────

// GetCategoryByID returns a single category or category.ErrNotFound.
func (s *CatalogService) GetCategoryByID(
	ctx context.Context,
	id int64,
) (*category.Category, error) {
	c, err := s.categoryRepo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err // already a domain error from the repo
	}
	return c, nil
}

// GetCategoryBySlug returns a single category looked up by its URL slug.
func (s *CatalogService) GetCategoryBySlug(
	ctx context.Context,
	slug string,
) (*category.Category, error) {
	return s.categoryRepo.GetCategoryBySlug(ctx, slug)
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListCategories returns categories matching the supplied filter.
// Pass a zero-value ListFilter{} for "all categories".
func (s *CatalogService) ListCategories(
	ctx context.Context,
	f category.ListFilter,
) ([]category.Category, error) {
	cats, err := s.categoryRepo.ListCategories(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return cats, nil
}

// GetCategoryTree returns a category together with its full ancestor chain.
func (s *CatalogService) GetCategoryTree(
	ctx context.Context,
	id int64,
) (*category.CategoryWithPath, error) {
	c, err := s.categoryRepo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	ancestors, err := s.categoryRepo.GetCategoryAncestors(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get ancestors: %w", err)
	}

	return &category.CategoryWithPath{
		Category:  *c,
		Ancestors: ancestors,
	}, nil
}

// GetCategoryDescendants returns all closure rows under id (excluding self).
func (s *CatalogService) GetCategoryDescendants(
	ctx context.Context,
	id int64,
) ([]category.CategoryClosure, error) {
	return s.categoryRepo.GetCategoryDescendants(ctx, id)
}

// ─── Update ───────────────────────────────────────────────────────────────────

// UpdateCategory applies a partial update to a category.
//   - Name change → slug is regenerated (with collision retry).
//   - ParentID == 0 → clears the parent (makes it a root).
//   - ParentID > 0  → re-parents; circular reference is rejected.
func (s *CatalogService) UpdateCategory(
	ctx context.Context,
	id int64,
	req *category.UpdateRequest,
) (*category.Category, error) {
	// 1. Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 2. Load existing
	c, err := s.categoryRepo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Apply name change + slug regeneration
	nameChanged := false
	if req.Name != nil && *req.Name != c.Name {
		c.Name = *req.Name
		nameChanged = true
	}

	// 4. Apply parent change with circular-reference guard
	if req.ParentID != nil {
		switch {
		case *req.ParentID == 0:
			// Clear parent → make root
			c.ParentID = sql.NullInt64{}

		case *req.ParentID == id:
			return nil, category.ErrCircularParent

		default:
			// Ensure new parent exists
			if _, err := s.categoryRepo.GetCategoryByID(ctx, *req.ParentID); err != nil {
				return nil, fmt.Errorf("new parent category: %w", err)
			}
			// Ensure new parent is not a descendant of id (would create a cycle)
			isDesc, err := s.categoryRepo.IsAncestor(ctx, *req.ParentID, id)
			if err != nil {
				return nil, fmt.Errorf("circular check: %w", err)
			}
			if isDesc {
				return nil, category.ErrCircularParent
			}
			c.ParentID = sql.NullInt64{Int64: *req.ParentID, Valid: true}
		}
	}

	// 5. Apply is_active flag
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}

	// 6. Persist — with slug collision retry when name changed
	if nameChanged {
		if err := s.updateCategoryWithSlugRetry(ctx, c); err != nil {
			return nil, err
		}
	} else {
		if err := s.categoryRepo.UpdateCategory(ctx, c); err != nil {
			return nil, fmt.Errorf("update category: %w", err)
		}
	}

	return c, nil
}

// updateCategoryWithSlugRetry regenerates the slug from the (already updated)
// c.Name and retries on collision.
func (s *CatalogService) updateCategoryWithSlugRetry(
	ctx context.Context,
	c *category.Category,
) error {
	baseSlug := pgRepo.GenerateSlug(c.Name)
	c.Slug = baseSlug

	for attempt := 1; attempt <= 10; attempt++ {
		if attempt > 1 {
			c.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
		}

		err := s.categoryRepo.UpdateCategory(ctx, c)
		if err == nil {
			return nil
		}
		if err == category.ErrSlugConflict {
			continue
		}
		return fmt.Errorf("update category: %w", err)
	}
	return fmt.Errorf("update category: could not generate unique slug for %q after 10 attempts", baseSlug)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// DeleteCategory removes a category by ID.
// The database trigger cascades the closure-table cleanup automatically.
func (s *CatalogService) DeleteCategory(ctx context.Context, id int64) error {
	if err := s.categoryRepo.DeleteCategory(ctx, id); err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

func (s *CatalogService) AddProductCategory(ctx context.Context, productID, categoryID int64) error {
	return s.categoryRepo.AddProductCategory(ctx, productID, categoryID)
}

func (s *CatalogService) RemoveProductCategory(ctx context.Context, productID, categoryID int64) error {
	return s.categoryRepo.RemoveProductCategory(ctx, productID, categoryID)
}

func (s *CatalogService) GetProductCategories(ctx context.Context, productID int64) ([]category.Category, error) {
	return s.categoryRepo.GetProductCategories(ctx, productID)
}
