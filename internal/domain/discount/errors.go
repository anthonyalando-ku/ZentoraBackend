package discount

import "errors"

var (
	ErrNotFound            = errors.New("discount not found")
	ErrRedemptionNotFound  = errors.New("discount redemption not found")
	ErrCodeConflict        = errors.New("discount code already exists")
	ErrRedemptionConflict  = errors.New("discount already redeemed for this order")
	ErrInvalidName         = errors.New("discount name is required and must be at most 255 characters")
	ErrInvalidCode         = errors.New("discount code must be at most 50 characters")
	ErrInvalidType         = errors.New("discount type must be 'percentage' or 'fixed'")
	ErrInvalidValue        = errors.New("discount value must be greater than zero")
	ErrInvalidPercentage   = errors.New("percentage discount value must be between 0 and 100")
	ErrInvalidDateRange    = errors.New("ends_at must be after starts_at")
	ErrInvalidTargetType   = errors.New("target type must be 'product', 'category', or 'brand'")
	ErrExpired             = errors.New("discount has expired")
	ErrNotStarted          = errors.New("discount has not started yet")
	ErrInactive            = errors.New("discount is not active")
	ErrMaxRedemptions      = errors.New("discount has reached maximum redemptions")
	ErrMinOrderAmount      = errors.New("order amount does not meet minimum required amount")
)