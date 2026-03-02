package catalog

import (
	"context"
	"fmt"

	"zentora-service/internal/domain/attribute"
	pgRepo "zentora-service/internal/repository/postgres"
)

func (s *CatalogService) CreateAttribute(ctx context.Context, req *attribute.CreateRequest) (*attribute.Attribute, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	a := &attribute.Attribute{
		Name:               req.Name,
		IsVariantDimension: req.IsVariantDimension,
	}
	baseSlug := pgRepo.GenerateSlug(req.Name)
	if err := s.createAttributeWithSlugRetry(ctx, a, baseSlug); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *CatalogService) createAttributeWithSlugRetry(ctx context.Context, a *attribute.Attribute, baseSlug string) error {
	a.Slug = baseSlug
	for attempt := 1; attempt <= 10; attempt++ {
		if attempt > 1 {
			a.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
		}
		err := s.attributeRepo.CreateAttribute(ctx, a)
		if err == nil {
			return nil
		}
		if err == attribute.ErrSlugConflict {
			continue
		}
		return fmt.Errorf("create attribute: %w", err)
	}
	return fmt.Errorf("create attribute: could not generate unique slug for %q after 10 attempts", baseSlug)
}

func (s *CatalogService) GetAttributeByID(ctx context.Context, id int64) (*attribute.Attribute, error) {
	return s.attributeRepo.GetAttributeByID(ctx, id)
}

func (s *CatalogService) ListAttributes(ctx context.Context) ([]attribute.Attribute, error) {
	return s.attributeRepo.ListAttributes(ctx)
}

func (s *CatalogService) ListAttributesWithValues(ctx context.Context) ([]attribute.AttributeWithValues, error) {
	return s.attributeRepo.ListAttributesWithValues(ctx)
}

func (s *CatalogService) UpdateAttribute(ctx context.Context, id int64, req *attribute.UpdateRequest) (*attribute.Attribute, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	a, err := s.attributeRepo.GetAttributeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	nameChanged := false
	if req.Name != nil && *req.Name != a.Name {
		a.Name = *req.Name
		nameChanged = true
	}
	if req.IsVariantDimension != nil {
		a.IsVariantDimension = *req.IsVariantDimension
	}

	if nameChanged {
		if err := s.updateAttributeWithSlugRetry(ctx, a); err != nil {
			return nil, err
		}
	} else {
		if err := s.attributeRepo.UpdateAttribute(ctx, a); err != nil {
			return nil, fmt.Errorf("update attribute: %w", err)
		}
	}
	return a, nil
}

func (s *CatalogService) updateAttributeWithSlugRetry(ctx context.Context, a *attribute.Attribute) error {
	baseSlug := pgRepo.GenerateSlug(a.Name)
	a.Slug = baseSlug
	for attempt := 1; attempt <= 10; attempt++ {
		if attempt > 1 {
			a.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
		}
		err := s.attributeRepo.UpdateAttribute(ctx, a)
		if err == nil {
			return nil
		}
		if err == attribute.ErrSlugConflict {
			continue
		}
		return fmt.Errorf("update attribute: %w", err)
	}
	return fmt.Errorf("update attribute: could not generate unique slug for %q after 10 attempts", baseSlug)
}

func (s *CatalogService) DeleteAttribute(ctx context.Context, id int64) error {
	return s.attributeRepo.DeleteAttribute(ctx, id)
}

func (s *CatalogService) AddAttributeValue(ctx context.Context, attributeID int64, req *attribute.CreateValueRequest) (*attribute.AttributeValue, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.attributeRepo.GetAttributeByID(ctx, attributeID); err != nil {
		return nil, err
	}
	v := &attribute.AttributeValue{
		AttributeID: attributeID,
		Value:       req.Value,
	}
	if err := s.attributeRepo.CreateAttributeValue(ctx, v); err != nil {
		return nil, fmt.Errorf("add attribute value: %w", err)
	}
	return v, nil
}

func (s *CatalogService) GetAttributeValueByID(ctx context.Context, id int64) (*attribute.AttributeValue, error) {
	return s.attributeRepo.GetAttributeValueByID(ctx, id)
}

func (s *CatalogService) ListAttributeValues(ctx context.Context, attributeID int64) ([]attribute.AttributeValue, error) {
	if _, err := s.attributeRepo.GetAttributeByID(ctx, attributeID); err != nil {
		return nil, err
	}
	return s.attributeRepo.ListAttributeValues(ctx, attributeID)
}

func (s *CatalogService) DeleteAttributeValue(ctx context.Context, id int64) error {
	return s.attributeRepo.DeleteAttributeValue(ctx, id)
}

func (s *CatalogService) SetProductAttributeValues(ctx context.Context, productID int64, valueIDs []int64) error {
	return s.attributeRepo.SetProductAttributeValues(ctx, nil, productID, valueIDs)
}

func (s *CatalogService) GetProductAttributeValues(ctx context.Context, productID int64) ([]attribute.AttributeValue, error) {
	return s.attributeRepo.GetProductAttributeValues(ctx, productID)
}
