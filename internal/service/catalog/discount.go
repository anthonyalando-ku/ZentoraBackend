package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"zentora-service/internal/domain/discount"
)

func (s *CatalogService) CreateDiscount(ctx context.Context, req *discount.CreateRequest) (*discount.DiscountWithTargets, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	d := buildDiscountEntity(req)
	targets := buildTargets(req.Targets)

	if err := s.discountRepo.CreateDiscount(ctx, nil, d, targets); err != nil {
		return nil, fmt.Errorf("create discount: %w", err)
	}
	return &discount.DiscountWithTargets{Discount: *d, Targets: targets}, nil
}

func (s *CatalogService) GetDiscountByID(ctx context.Context, id int64) (*discount.DiscountWithTargets, error) {
	return s.discountRepo.GetDiscountWithTargets(ctx, id)
}

func (s *CatalogService) GetDiscountByCode(ctx context.Context, code string) (*discount.DiscountWithTargets, error) {
	d, err := s.discountRepo.GetDiscountByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	targets, err := s.discountRepo.GetTargets(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	return &discount.DiscountWithTargets{Discount: *d, Targets: targets}, nil
}

func (s *CatalogService) ListDiscounts(ctx context.Context, f discount.ListFilter) ([]discount.Discount, error) {
	return s.discountRepo.ListDiscounts(ctx, f)
}

func (s *CatalogService) UpdateDiscount(ctx context.Context, id int64, req *discount.UpdateRequest) (*discount.Discount, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	d, err := s.discountRepo.GetDiscountByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		d.Name = *req.Name
	}
	if req.Code != nil {
		d.Code = sql.NullString{String: *req.Code, Valid: *req.Code != ""}
	}
	if req.DiscountType != nil {
		d.DiscountType = *req.DiscountType
	}
	if req.Value != nil {
		d.Value = *req.Value
	}
	if req.MinOrderAmount != nil {
		d.MinOrderAmount = sql.NullFloat64{Float64: *req.MinOrderAmount, Valid: true}
	}
	if req.MaxRedemptions != nil {
		d.MaxRedemptions = sql.NullInt64{Int64: *req.MaxRedemptions, Valid: true}
	}
	if req.StartsAt != nil {
		d.StartsAt = sql.NullTime{Time: *req.StartsAt, Valid: true}
	}
	if req.EndsAt != nil {
		d.EndsAt = sql.NullTime{Time: *req.EndsAt, Valid: true}
	}
	if req.IsActive != nil {
		d.IsActive = *req.IsActive
	}

	if err := s.discountRepo.UpdateDiscount(ctx, d); err != nil {
		return nil, fmt.Errorf("update discount: %w", err)
	}
	return d, nil
}

func (s *CatalogService) SetDiscountTargets(ctx context.Context, discountID int64, targets []discount.TargetInput) error {
	if _, err := s.discountRepo.GetDiscountByID(ctx, discountID); err != nil {
		return err
	}
	for _, t := range targets {
		if t.TargetType != discount.TargetProduct &&
			t.TargetType != discount.TargetCategory &&
			t.TargetType != discount.TargetBrand {
			return discount.ErrInvalidTargetType
		}
	}
	return s.discountRepo.SetTargets(ctx, nil, discountID, buildTargets(targets))
}

func (s *CatalogService) DeleteDiscount(ctx context.Context, id int64) error {
	return s.discountRepo.DeleteDiscount(ctx, id)
}

// ValidateAndRedeem validates a discount code against an order and records
// the redemption inside the provided transaction (or its own if nil).
func (s *CatalogService) ValidateAndRedeem(ctx context.Context, tx interface {
	Begin(context.Context) (interface{}, error)
}, req *discount.RedeemRequest) (*discount.Discount, float64, error) {
	d, err := s.discountRepo.GetDiscountByCode(ctx, req.Code)
	if err != nil {
		return nil, 0, err
	}

	if err := validateDiscount(d, req.Amount); err != nil {
		return nil, 0, err
	}

	if d.MaxRedemptions.Valid {
		count, err := s.discountRepo.CountRedemptions(ctx, d.ID)
		if err != nil {
			return nil, 0, err
		}
		if int64(count) >= d.MaxRedemptions.Int64 {
			return nil, 0, discount.ErrMaxRedemptions
		}
	}

	red := &discount.DiscountRedemption{
		DiscountID: d.ID,
		OrderID:    req.OrderID,
	}
	if req.UserID != nil {
		red.UserID = sql.NullInt64{Int64: *req.UserID, Valid: true}
	}

	if err := s.discountRepo.RecordRedemption(ctx, nil, red); err != nil {
		return nil, 0, err
	}

	amount := calculateDiscount(d, req.Amount)
	return d, amount, nil
}

func validateDiscount(d *discount.Discount, orderAmount float64) error {
	if !d.IsActive {
		return discount.ErrInactive
	}
	now := time.Now()
	if d.StartsAt.Valid && now.Before(d.StartsAt.Time) {
		return discount.ErrNotStarted
	}
	if d.EndsAt.Valid && now.After(d.EndsAt.Time) {
		return discount.ErrExpired
	}
	if d.MinOrderAmount.Valid && orderAmount < d.MinOrderAmount.Float64 {
		return discount.ErrMinOrderAmount
	}
	return nil
}

func calculateDiscount(d *discount.Discount, orderAmount float64) float64 {
	switch d.DiscountType {
	case discount.TypePercentage:
		return orderAmount * d.Value / 100
	case discount.TypeFixed:
		if d.Value > orderAmount {
			return orderAmount
		}
		return d.Value
	}
	return 0
}

func buildDiscountEntity(req *discount.CreateRequest) *discount.Discount {
	d := &discount.Discount{
		Name:         req.Name,
		DiscountType: req.DiscountType,
		Value:        req.Value,
		IsActive:     true,
	}
	if req.IsActive != nil {
		d.IsActive = *req.IsActive
	}
	if req.Code != nil {
		d.Code = sql.NullString{String: *req.Code, Valid: true}
	}
	if req.MinOrderAmount != nil {
		d.MinOrderAmount = sql.NullFloat64{Float64: *req.MinOrderAmount, Valid: true}
	}
	if req.MaxRedemptions != nil {
		d.MaxRedemptions = sql.NullInt64{Int64: *req.MaxRedemptions, Valid: true}
	}
	if req.StartsAt != nil {
		d.StartsAt = sql.NullTime{Time: *req.StartsAt, Valid: true}
	}
	if req.EndsAt != nil {
		d.EndsAt = sql.NullTime{Time: *req.EndsAt, Valid: true}
	}
	return d
}

func buildTargets(inputs []discount.TargetInput) []discount.DiscountTarget {
	out := make([]discount.DiscountTarget, 0, len(inputs))
	for _, t := range inputs {
		out = append(out, discount.DiscountTarget{
			TargetType: t.TargetType,
			TargetID:   t.TargetID,
		})
	}
	return out
}
