package product

import (
	"strings"
	"unicode/utf8"
)

type ImageInput struct {
	IsPrimary bool `json:"is_primary"`
	SortOrder int  `json:"sort_order"`
}

type VariantInput struct {
	SKU               string   `json:"sku"`
	Price             float64  `json:"price"`
	Weight            *float64 `json:"weight,omitempty"`
	IsActive          *bool    `json:"is_active,omitempty"`
	AttributeValueIDs []int64  `json:"attribute_value_ids,omitempty"`
	Quantity          int      `json:"quantity"`
	LocationID        *int64   `json:"location_id,omitempty"`
}

type DiscountInput struct {
	DiscountID *int64  `json:"discount_id,omitempty"`
	Name       *string `json:"name,omitempty"`
	Code       *string `json:"code,omitempty"`
}

type CreateRequest struct {
	Name             string         `json:"name"`
	Description      *string        `json:"description,omitempty"`
	ShortDescription *string        `json:"short_description,omitempty"`
	BrandID          int64          `json:"brand_id"`
	BasePrice        float64        `json:"base_price"`
	Status           Status         `json:"status"`
	IsFeatured       bool           `json:"is_featured"`
	IsDigital        bool           `json:"is_digital"`
	CategoryIDs      []int64        `json:"category_ids"`
	TagNames         []string       `json:"tag_names,omitempty"`
	AttributeValueIDs []int64       `json:"attribute_value_ids,omitempty"`
	Variants         []VariantInput `json:"variants"`
	Discount         *DiscountInput `json:"discount,omitempty"`
}

func (r *CreateRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" || utf8.RuneCountInString(r.Name) > 255 {
		return ErrInvalidName
	}
	if r.BrandID <= 0 {
		return ErrBrandRequired
	}
	if r.BasePrice <= 0 {
		return ErrInvalidPrice
	}
	if r.Status == "" {
		r.Status = StatusActive
	}
	if r.Status != StatusActive && r.Status != StatusDraft && r.Status != StatusArchived {
		return ErrInvalidStatus
	}
	if len(r.CategoryIDs) == 0 {
		return ErrCategoryRequired
	}
	if len(r.Variants) == 0 {
		return ErrVariantRequired
	}
	return nil
}

type UpdateRequest struct {
	Name             *string  `json:"name,omitempty"`
	Description      *string  `json:"description,omitempty"`
	ShortDescription *string  `json:"short_description,omitempty"`
	BrandID          *int64   `json:"brand_id,omitempty"`
	BasePrice        *float64 `json:"base_price,omitempty"`
	Status           *Status  `json:"status,omitempty"`
	IsFeatured       *bool    `json:"is_featured,omitempty"`
	IsDigital        *bool    `json:"is_digital,omitempty"`
}

func (r *UpdateRequest) Validate() error {
	if r.Name != nil {
		*r.Name = strings.TrimSpace(*r.Name)
		if *r.Name == "" || utf8.RuneCountInString(*r.Name) > 255 {
			return ErrInvalidName
		}
	}
	if r.BasePrice != nil && *r.BasePrice <= 0 {
		return ErrInvalidPrice
	}
	if r.Status != nil && *r.Status != StatusActive && *r.Status != StatusDraft && *r.Status != StatusArchived {
		return ErrInvalidStatus
	}
	return nil
}

type ListRequest struct {
	Page     int
	PageSize int
	Filter   ListFilter
}

type SetPrimaryImageRequest struct {
	ImageID int64 `json:"image_id"`
}