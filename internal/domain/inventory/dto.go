package inventory

import (
	"strings"
	"unicode/utf8"
)

type CreateLocationRequest struct {
	Name         string  `json:"name"`
	LocationCode *string `json:"location_code,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

func (r *CreateLocationRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" || utf8.RuneCountInString(r.Name) > 150 {
		return ErrInvalidName
	}
	if r.LocationCode != nil {
		*r.LocationCode = strings.TrimSpace(*r.LocationCode)
		if utf8.RuneCountInString(*r.LocationCode) > 50 {
			return ErrInvalidLocationCode
		}
	}
	return nil
}

type UpdateLocationRequest struct {
	Name         *string `json:"name,omitempty"`
	LocationCode *string `json:"location_code,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

func (r *UpdateLocationRequest) Validate() error {
	if r.Name != nil {
		*r.Name = strings.TrimSpace(*r.Name)
		if *r.Name == "" || utf8.RuneCountInString(*r.Name) > 150 {
			return ErrInvalidName
		}
	}
	if r.LocationCode != nil {
		*r.LocationCode = strings.TrimSpace(*r.LocationCode)
		if utf8.RuneCountInString(*r.LocationCode) > 50 {
			return ErrInvalidLocationCode
		}
	}
	return nil
}

type UpsertItemRequest struct {
	VariantID    int64 `json:"variant_id"`
	LocationID   int64 `json:"location_id"`
	AvailableQty int   `json:"available_qty"`
	ReservedQty  int   `json:"reserved_qty"`
	IncomingQty  int   `json:"incoming_qty"`
}

func (r *UpsertItemRequest) Validate() error {
	if r.AvailableQty < 0 || r.ReservedQty < 0 || r.IncomingQty < 0 {
		return ErrInvalidQuantity
	}
	return nil
}

type AdjustQtyRequest struct {
	Delta int `json:"delta"`
}

type LocationFilter struct {
	ActiveOnly bool
}