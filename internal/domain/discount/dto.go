package discount

import (
	"strings"
	"time"
	"unicode/utf8"
)

type TargetInput struct {
	TargetType TargetType `json:"target_type"`
	TargetID   int64      `json:"target_id"`
}

type CreateRequest struct {
	Name           string        `json:"name"`
	Code           *string       `json:"code,omitempty"`
	DiscountType   DiscountType  `json:"discount_type"`
	Value          float64       `json:"value"`
	MinOrderAmount *float64      `json:"min_order_amount,omitempty"`
	MaxRedemptions *int64        `json:"max_redemptions,omitempty"`
	StartsAt       *time.Time    `json:"starts_at,omitempty"`
	EndsAt         *time.Time    `json:"ends_at,omitempty"`
	IsActive       *bool         `json:"is_active,omitempty"`
	Targets        []TargetInput `json:"targets,omitempty"`
}

func (r *CreateRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" || utf8.RuneCountInString(r.Name) > 255 {
		return ErrInvalidName
	}
	if r.Code != nil {
		*r.Code = strings.TrimSpace(*r.Code)
		if utf8.RuneCountInString(*r.Code) > 50 {
			return ErrInvalidCode
		}
	}
	if r.DiscountType != TypePercentage && r.DiscountType != TypeFixed {
		return ErrInvalidType
	}
	if r.Value <= 0 {
		return ErrInvalidValue
	}
	if r.DiscountType == TypePercentage && r.Value > 100 {
		return ErrInvalidPercentage
	}
	if r.StartsAt != nil && r.EndsAt != nil && r.EndsAt.Before(*r.StartsAt) {
		return ErrInvalidDateRange
	}
	for _, t := range r.Targets {
		if t.TargetType != TargetProduct && t.TargetType != TargetCategory && t.TargetType != TargetBrand {
			return ErrInvalidTargetType
		}
	}
	return nil
}

type UpdateRequest struct {
	Name           *string      `json:"name,omitempty"`
	Code           *string      `json:"code,omitempty"`
	DiscountType   *DiscountType `json:"discount_type,omitempty"`
	Value          *float64     `json:"value,omitempty"`
	MinOrderAmount *float64     `json:"min_order_amount,omitempty"`
	MaxRedemptions *int64       `json:"max_redemptions,omitempty"`
	StartsAt       *time.Time   `json:"starts_at,omitempty"`
	EndsAt         *time.Time   `json:"ends_at,omitempty"`
	IsActive       *bool        `json:"is_active,omitempty"`
}

func (r *UpdateRequest) Validate() error {
	if r.Name != nil {
		*r.Name = strings.TrimSpace(*r.Name)
		if *r.Name == "" || utf8.RuneCountInString(*r.Name) > 255 {
			return ErrInvalidName
		}
	}
	if r.Code != nil {
		*r.Code = strings.TrimSpace(*r.Code)
		if utf8.RuneCountInString(*r.Code) > 50 {
			return ErrInvalidCode
		}
	}
	if r.DiscountType != nil && *r.DiscountType != TypePercentage && *r.DiscountType != TypeFixed {
		return ErrInvalidType
	}
	if r.Value != nil && *r.Value <= 0 {
		return ErrInvalidValue
	}
	if r.DiscountType != nil && *r.DiscountType == TypePercentage && r.Value != nil && *r.Value > 100 {
		return ErrInvalidPercentage
	}
	if r.StartsAt != nil && r.EndsAt != nil && r.EndsAt.Before(*r.StartsAt) {
		return ErrInvalidDateRange
	}
	return nil
}

type RedeemRequest struct {
	Code    string  `json:"code"`
	OrderID int64   `json:"order_id"`
	UserID  *int64  `json:"user_id,omitempty"`
	Amount  float64 `json:"amount"`
}

type ListFilter struct {
	ActiveOnly bool
	Code       *string
}