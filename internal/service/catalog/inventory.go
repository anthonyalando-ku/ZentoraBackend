package catalog

import (
	"context"
	"database/sql"
	"fmt"

	"zentora-service/internal/domain/inventory"
)

func (s *CatalogService) CreateLocation(ctx context.Context, req *inventory.CreateLocationRequest) (*inventory.Location, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	l := &inventory.Location{
		Name:     req.Name,
		IsActive: true,
	}
	if req.IsActive != nil {
		l.IsActive = *req.IsActive
	}
	if req.LocationCode != nil {
		l.LocationCode = sql.NullString{String: *req.LocationCode, Valid: true}
	}
	if err := s.inventoryRepo.CreateLocation(ctx, l); err != nil {
		return nil, fmt.Errorf("create location: %w", err)
	}
	return l, nil
}

func (s *CatalogService) GetLocationByID(ctx context.Context, id int64) (*inventory.Location, error) {
	return s.inventoryRepo.GetLocationByID(ctx, id)
}

func (s *CatalogService) ListLocations(ctx context.Context, f inventory.LocationFilter) ([]inventory.Location, error) {
	return s.inventoryRepo.ListLocations(ctx, f)
}

func (s *CatalogService) UpdateLocation(ctx context.Context, id int64, req *inventory.UpdateLocationRequest) (*inventory.Location, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	l, err := s.inventoryRepo.GetLocationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		l.Name = *req.Name
	}
	if req.IsActive != nil {
		l.IsActive = *req.IsActive
	}
	if req.LocationCode != nil {
		l.LocationCode = sql.NullString{String: *req.LocationCode, Valid: *req.LocationCode != ""}
	}
	if err := s.inventoryRepo.UpdateLocation(ctx, l); err != nil {
		return nil, fmt.Errorf("update location: %w", err)
	}
	return l, nil
}

func (s *CatalogService) DeleteLocation(ctx context.Context, id int64) error {
	return s.inventoryRepo.DeleteLocation(ctx, id)
}

func (s *CatalogService) UpsertInventoryItem(ctx context.Context, req *inventory.UpsertItemRequest) (*inventory.Item, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.inventoryRepo.GetLocationByID(ctx, req.LocationID); err != nil {
		return nil, fmt.Errorf("location: %w", err)
	}
	item := &inventory.Item{
		VariantID:    req.VariantID,
		LocationID:   req.LocationID,
		AvailableQty: req.AvailableQty,
		ReservedQty:  req.ReservedQty,
		IncomingQty:  req.IncomingQty,
	}
	if err := s.inventoryRepo.UpsertItem(ctx, nil, item); err != nil {
		return nil, fmt.Errorf("upsert inventory item: %w", err)
	}
	return item, nil
}

func (s *CatalogService) GetInventoryItemByID(ctx context.Context, id int64) (*inventory.Item, error) {
	return s.inventoryRepo.GetItemByID(ctx, id)
}

func (s *CatalogService) GetInventoryByVariant(ctx context.Context, variantID int64) ([]inventory.ItemWithLocation, error) {
	return s.inventoryRepo.GetItemsByVariant(ctx, variantID)
}

func (s *CatalogService) GetStockSummary(ctx context.Context, variantID int64) (*inventory.StockSummary, error) {
	return s.inventoryRepo.GetStockSummary(ctx, variantID)
}

func (s *CatalogService) AdjustAvailableStock(ctx context.Context, variantID, locationID int64, req *inventory.AdjustQtyRequest) error {
	if _, err := s.inventoryRepo.GetItemByVariantAndLocation(ctx, variantID, locationID); err != nil {
		return err
	}
	return s.inventoryRepo.AdjustAvailable(ctx, nil, variantID, locationID, req.Delta)
}

func (s *CatalogService) ReserveStock(ctx context.Context, variantID, locationID int64, qty int) error {
	if qty <= 0 {
		return inventory.ErrInvalidQuantity
	}
	return s.inventoryRepo.Reserve(ctx, nil, variantID, locationID, qty)
}

func (s *CatalogService) ReleaseStock(ctx context.Context, variantID, locationID int64, qty int) error {
	if qty <= 0 {
		return inventory.ErrInvalidQuantity
	}
	return s.inventoryRepo.Release(ctx, nil, variantID, locationID, qty)
}

func (s *CatalogService) DeleteInventoryItem(ctx context.Context, variantID, locationID int64) error {
	return s.inventoryRepo.DeleteItem(ctx, variantID, locationID)
}
