package catalog

import (
	"context"
	"fmt"

	"zentora-service/internal/domain/tag"
)

func (s *CatalogService) FindOrCreateTag(ctx context.Context, name string) (*tag.Tag, error) {
	req := &tag.CreateRequest{Name: name}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	t, err := s.tagRepo.FindOrCreateByName(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("find or create tag: %w", err)
	}
	return t, nil
}

func (s *CatalogService) GetTagByID(ctx context.Context, id int64) (*tag.Tag, error) {
	return s.tagRepo.GetTagByID(ctx, id)
}

func (s *CatalogService) ListTags(ctx context.Context) ([]tag.Tag, error) {
	tags, err := s.tagRepo.ListTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return tags, nil
}

func (s *CatalogService) AddTagToProduct(ctx context.Context, productID int64, tagName string) (*tag.Tag, error) {
	req := &tag.CreateRequest{Name: tagName}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	t, err := s.tagRepo.FindOrCreateByName(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("resolve tag: %w", err)
	}

	if err := s.tagRepo.AddTagToProduct(ctx, productID, t.ID); err != nil {
		return nil, fmt.Errorf("add tag to product: %w", err)
	}
	return t, nil
}

func (s *CatalogService) RemoveTagFromProduct(ctx context.Context, productID, tagID int64) error {
	if err := s.tagRepo.RemoveTagFromProduct(ctx, productID, tagID); err != nil {
		return fmt.Errorf("remove tag from product: %w", err)
	}
	return nil
}

func (s *CatalogService) GetProductTags(ctx context.Context, productID int64) ([]tag.Tag, error) {
	tags, err := s.tagRepo.GetProductTags(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("get product tags: %w", err)
	}
	return tags, nil
}

func (s *CatalogService) SetProductTags(ctx context.Context, productID int64, tagNames []string) error {
	for _, name := range tagNames {
		req := &tag.CreateRequest{Name: name}
		if err := req.Validate(); err != nil {
			return fmt.Errorf("invalid tag %q: %w", name, err)
		}
	}
	return s.tagRepo.SetProductTags(ctx, nil, productID, tagNames)
}
